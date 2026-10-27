package main

import (
	"embed"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

//go:embed web
var webFS embed.FS

func init() {
	for ext, typ := range map[string]string{
		".heic": "image/heic", ".heif": "image/heif", ".webp": "image/webp",
		".dng": "image/x-adobe-dng", ".webmanifest": "application/manifest+json",
	} {
		_ = mime.AddExtensionType(ext, typ)
	}
}

// isLocal reports whether the request comes from the Mac running the app.
// Only the Mac may change project settings; phones on the Wi-Fi can only swipe.
func isLocal(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

type Server struct {
	app  *App
	port int
}

type stateResponse struct {
	State
	IsLocal bool     `json:"isLocal"`
	URLs    []string `json:"urls"`
}

func (s *Server) stateResp(r *http.Request) stateResponse {
	return stateResponse{State: s.app.State(), IsLocal: isLocal(r), URLs: phoneURLs(s.port)}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(webFS, "web")
	files := http.FileServer(http.FS(static))

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		st := s.app.State()
		if !st.Configured && isLocal(r) {
			http.Redirect(w, r, "/setup", http.StatusFound)
			return
		}
		servePage(w, "index.html")
	})
	mux.HandleFunc("GET /setup", func(w http.ResponseWriter, r *http.Request) {
		servePage(w, "setup.html")
	})
	mux.Handle("GET /static/", http.StripPrefix("/static", files))

	mux.HandleFunc("GET /api/state", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.stateResp(r))
	})

	mux.HandleFunc("POST /api/decide", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Path, Op string }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request")
			return
		}
		if err := s.app.Decide(req.Path, req.Op); err != nil {
			code := http.StatusInternalServerError
			if errors.Is(err, ErrStale) {
				code = http.StatusConflict
			}
			writeJSON(w, code, struct {
				stateResponse
				Err string `json:"error"`
			}{s.stateResp(r), err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, s.stateResp(r))
	})

	mux.HandleFunc("POST /api/undo", func(w http.ResponseWriter, r *http.Request) {
		if err := s.app.Undo(); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, s.stateResp(r))
	})

	mux.HandleFunc("GET /photo", func(w http.ResponseWriter, r *http.Request) {
		src := s.app.PhotoSrc(r.URL.Query().Get("p"))
		if src == "" {
			http.NotFound(w, r)
			return
		}
		f, err := os.Open(s.app.prev.Get(src))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "private, max-age=3600")
		http.ServeContent(w, r, f.Name(), info.ModTime(), f)
	})

	mux.HandleFunc("GET /api/export.csv", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="photo-select-decisions.csv"`)
		_ = csv.NewWriter(w).WriteAll(s.app.ExportRows())
	})

	// Setup endpoints: Mac only.
	mux.HandleFunc("POST /api/setup", func(w http.ResponseWriter, r *http.Request) {
		if !isLocal(r) {
			writeErr(w, http.StatusForbidden, "change settings on the computer running Photo Select")
			return
		}
		var c Config
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request")
			return
		}
		if err := s.app.Start(c); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, s.stateResp(r))
	})

	// Quit: the Mac app has no Terminal window to close, so the setup page offers a button.
	mux.HandleFunc("POST /api/quit", func(w http.ResponseWriter, r *http.Request) {
		if !isLocal(r) {
			writeErr(w, http.StatusForbidden, "quit Photo Select on the computer running it")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		go func() { time.Sleep(200 * time.Millisecond); os.Exit(0) }()
	})

	mux.HandleFunc("POST /api/pick-folder", func(w http.ResponseWriter, r *http.Request) {
		if !isLocal(r) || runtime.GOOS != "darwin" {
			writeErr(w, http.StatusForbidden, "folder picker is only available on the Mac")
			return
		}
		out, err := exec.CommandContext(r.Context(), "osascript", "-e",
			`POSIX path of (choose folder with prompt "Choose a folder for Photo Select")`).Output()
		if err != nil { // user pressed Cancel
			writeJSON(w, http.StatusOK, map[string]string{"path": ""})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"path": filepath.Clean(strings.TrimSpace(string(out)))})
	})

	return mux
}

func servePage(w http.ResponseWriter, name string) {
	b, err := webFS.ReadFile("web/" + name)
	if err != nil {
		http.Error(w, "missing page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(b)
}

// phoneURLs lists the addresses a phone on the same Wi-Fi can open.
func phoneURLs(port int) []string {
	var urls []string
	p := ":" + strconv.Itoa(port)
	if host, err := os.Hostname(); err == nil && host != "" && runtime.GOOS == "darwin" {
		host = strings.TrimSuffix(host, ".local")
		urls = append(urls, "http://"+host+".local"+p)
	}
	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok {
				if ip := ipn.IP.To4(); ip != nil && ip.IsPrivate() {
					urls = append([]string{"http://" + ip.String() + p}, urls...)
				}
			}
		}
	}
	return urls
}
