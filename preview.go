package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const previewSize = 2048 // longest edge in pixels; sharp on a phone and on a TV

// Previewer makes phone-sized JPEG previews with macOS's built-in `sips`
// tool, which also decodes HEIC and camera RAW. Elsewhere (or if sips fails)
// the original file is served as-is.
type Previewer struct {
	dir      string
	enabled  bool
	sem      chan struct{}
	mu       sync.Mutex
	inflight map[string]chan struct{}
}

func NewPreviewer() *Previewer {
	p := &Previewer{sem: make(chan struct{}, 2), inflight: map[string]chan struct{}{}}
	if runtime.GOOS != "darwin" {
		return p
	}
	if _, err := exec.LookPath("sips"); err != nil {
		return p
	}
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	p.dir = filepath.Join(base, "photo-select", "previews")
	// Previews are cheap to remake; start fresh so the cache never grows unbounded.
	os.RemoveAll(p.dir)
	if os.MkdirAll(p.dir, 0o755) == nil {
		p.enabled = true
	}
	return p
}

// needsPreview: small JPEG/PNG/GIF/WebP files are served directly.
func needsPreview(src string, size int64) bool {
	switch strings.ToLower(filepath.Ext(src)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return size > 1500*1024
	}
	return true
}

func (p *Previewer) key(src string) (string, bool) {
	info, err := os.Stat(src)
	if err != nil || !p.enabled || !needsPreview(src, info.Size()) {
		return "", false
	}
	h := sha1.Sum([]byte(fmt.Sprintf("%s|%d|%d", src, info.Size(), info.ModTime().UnixNano())))
	return filepath.Join(p.dir, hex.EncodeToString(h[:10])+".jpg"), true
}

// Get returns the file to serve for src: a cached preview, or src itself.
func (p *Previewer) Get(src string) string {
	out, ok := p.key(src)
	if !ok {
		return src
	}
	for {
		if _, err := os.Stat(out); err == nil {
			return out
		}
		p.mu.Lock()
		if ch, busy := p.inflight[out]; busy {
			p.mu.Unlock()
			<-ch
			if _, err := os.Stat(out); err == nil {
				return out
			}
			return src
		}
		ch := make(chan struct{})
		p.inflight[out] = ch
		p.mu.Unlock()

		err := p.generate(src, out)

		p.mu.Lock()
		delete(p.inflight, out)
		close(ch)
		p.mu.Unlock()
		if err != nil {
			return src
		}
	}
}

func (p *Previewer) generate(src, out string) error {
	p.sem <- struct{}{}
	defer func() { <-p.sem }()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	tmp := out + ".tmp.jpg"
	cmd := exec.CommandContext(ctx, "sips", "-s", "format", "jpeg", "-s", "formatOptions", "82",
		"-Z", fmt.Sprint(previewSize), src, "--out", tmp)
	if b, err := cmd.CombinedOutput(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("sips: %v: %s", err, b)
	}
	return os.Rename(tmp, out)
}

// Prefetch builds previews for upcoming photos in the background.
func (p *Previewer) Prefetch(srcs []string) {
	if !p.enabled {
		return
	}
	for _, s := range srcs {
		go p.Get(s)
	}
}

// Forget deletes the cached preview of a photo that has been decided.
func (p *Previewer) Forget(src string) {
	if out, ok := p.key(src); ok {
		os.Remove(out)
	}
}
