package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Config is one photo-selection project. It's saved in the user's config dir
// (so the app reopens the last project) and in the output folder.
type Config struct {
	Source     string `json:"source"`
	Output     string `json:"output"`
	Target     int    `json:"target"`
	SelectMode string `json:"selectMode"` // copy | move
	RejectMode string `json:"rejectMode"` // none | copy | move
	Sort       string `json:"sort"`       // always "date" (kept so old config files still load)
}

const (
	selectedFolder = "Selected"
	rejectedFolder = "Rejected"
)

func (c *Config) Validate() error {
	c.Source, c.Output = cleanPath(c.Source), cleanPath(c.Output)
	if c.Source == "" {
		return errors.New("choose the folder that contains your photos")
	}
	if info, err := os.Stat(c.Source); err != nil || !info.IsDir() {
		return fmt.Errorf("photo folder not found: %s (is the drive plugged in?)", c.Source)
	}
	if c.Output == "" {
		return errors.New("choose where the selected photos should go")
	}
	c.Output = projectRoot(c.Output, c.Source)
	if c.Output == c.Source {
		return errors.New("the output folder must be different from the photo folder")
	}
	if isWithin(c.Source, c.Output) {
		return errors.New("the photo folder can't be inside the output folder")
	}
	if c.Target <= 0 {
		c.Target = 500
	}
	if c.SelectMode != "move" {
		c.SelectMode = "copy"
	}
	if c.RejectMode != "copy" && c.RejectMode != "move" {
		c.RejectMode = "none"
	}
	c.Sort = "date"
	return nil
}

type App struct {
	mu       sync.Mutex
	cfgPath  string
	cfg      *Config
	photos   []Photo
	index    map[string]int
	sess     *Session
	scanning bool
	phase    string // what the scan is doing, for the progress message
	scanned  int
	hidden   int // duplicates / lower-quality versions left out
	err      string
	gen      int
	prev     *Previewer
}

func NewApp(cfgPath string, prev *Previewer) *App {
	return &App{cfgPath: cfgPath, prev: prev}
}

func defaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "photo-select", "config.json")
}

func loadConfig(p string) (*Config, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var c Config
	return &c, json.Unmarshal(b, &c)
}

func saveJSON(p string, v any) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Resume reopens the last project, if any. A missing drive is reported in the
// UI rather than treated as fatal.
func (a *App) Resume() {
	c, err := loadConfig(a.cfgPath)
	if err != nil {
		return
	}
	if err := a.Start(*c); err != nil {
		a.mu.Lock()
		a.cfg = c
		a.err = err.Error()
		a.mu.Unlock()
	}
}

// Start opens a project: validates it, opens the decision log and scans the
// photo folder in the background.
func (a *App) Start(c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(c.Output, 0o755); err != nil {
		return fmt.Errorf("can't create output folder: %v", err)
	}
	sess, err := OpenSession(c.Output)
	if err != nil {
		return fmt.Errorf("can't open saved progress: %v", err)
	}
	_ = saveJSON(a.cfgPath, c)
	_ = saveJSON(filepath.Join(stateDir(c.Output), "project.json"), c)

	a.mu.Lock()
	if a.sess != nil {
		a.sess.Close()
	}
	a.cfg, a.sess = &c, sess
	a.photos, a.index = nil, nil
	a.scanning, a.phase, a.scanned, a.hidden, a.err = true, "Finding photos", 0, 0, ""
	a.gen++
	gen := a.gen
	a.mu.Unlock()

	go a.scan(gen, c)
	return nil
}

func (a *App) scan(gen int, c Config) {
	progress := func(n int) {
		a.mu.Lock()
		if a.gen == gen {
			a.scanned = n
		}
		a.mu.Unlock()
	}
	photos, err := Scan(c.Source, c.Output, progress)
	if err != nil {
		a.mu.Lock()
		if a.gen == gen {
			a.scanning = false
			a.err = fmt.Sprintf("couldn't read photo folder: %v", err)
		}
		a.mu.Unlock()
		return
	}
	a.mu.Lock()
	if a.gen != gen {
		a.mu.Unlock()
		return
	}
	a.phase, a.scanned = fmt.Sprintf("Reading dates of %d photos", len(photos)), 0
	sess := a.sess
	a.mu.Unlock()
	cache := Enrich(c.Source, c.Output, photos, progress)
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.gen != gen {
		return
	}
	photos, a.hidden = Dedupe(c.Source, photos, cache, func(rel string) bool {
		_, _, ok := sess.Decision(rel)
		return ok
	})
	_ = cache.save(c.Output) // keeps the duplicate checks for next time
	a.scanning = false
	// Photos moved out of the source folder are still part of the project.
	seen := make(map[string]bool, len(photos))
	for _, p := range photos {
		seen[p.Rel] = true
	}
	for _, e := range a.sess.stack {
		if e.Moved && !seen[e.Path] {
			if cur, _, ok := a.sess.Decision(e.Path); ok && cur.Moved {
				seen[e.Path] = true
				var p = Photo{Rel: e.Path}
				if info, err := os.Stat(filepath.Join(c.Output, filepath.FromSlash(cur.Dest))); err == nil {
					p.ModTime, p.Size = info.ModTime(), info.Size()
				}
				photos = append(photos, p)
			}
		}
	}
	SortPhotos(photos)
	a.photos = photos
	a.index = make(map[string]int, len(photos))
	for i, p := range photos {
		a.index[p.Rel] = i
	}
	a.scanned = len(photos)
	a.prefetchLocked()
}

// locateLocked returns where a photo's file currently lives on disk.
func (a *App) locateLocked(rel string) string {
	if e, _, ok := a.sess.Decision(rel); ok && e.Moved {
		return filepath.Join(a.cfg.Output, filepath.FromSlash(e.Dest))
	}
	return filepath.Join(a.cfg.Source, filepath.FromSlash(rel))
}

// queueLocked returns up to n photo indexes still to be decided: undecided
// photos in order, then "maybe" photos, oldest maybe first.
func (a *App) queueLocked(n int) []int {
	var out []int
	for i, p := range a.photos {
		if _, _, ok := a.sess.Decision(p.Rel); !ok {
			out = append(out, i)
			if len(out) == n {
				return out
			}
		}
	}
	type mb struct{ idx, seq int }
	var maybes []mb
	for i, p := range a.photos {
		if e, seq, ok := a.sess.Decision(p.Rel); ok && e.Op == OpMaybe {
			maybes = append(maybes, mb{i, seq})
		}
	}
	sort.Slice(maybes, func(i, j int) bool { return maybes[i].seq < maybes[j].seq })
	for _, m := range maybes {
		if len(out) == n {
			break
		}
		out = append(out, m.idx)
	}
	return out
}

func (a *App) prefetchLocked() {
	var srcs []string
	for _, i := range a.queueLocked(5) {
		srcs = append(srcs, a.locateLocked(a.photos[i].Rel))
	}
	a.prev.Prefetch(srcs)
}

// PhotoSrc returns the file to serve for a photo path, or "" if unknown.
func (a *App) PhotoSrc(rel string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.index == nil {
		return ""
	}
	if _, ok := a.index[rel]; !ok {
		return ""
	}
	return a.locateLocked(rel)
}

var ErrStale = errors.New("that photo was already decided (maybe on another device)")

// Decide records a swipe on the current photo and copies/moves the file.
func (a *App) Decide(rel, op string) error {
	if op != OpSelect && op != OpReject && op != OpMaybe {
		return fmt.Errorf("unknown decision %q", op)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.index == nil {
		return errors.New("photos are still loading")
	}
	q := a.queueLocked(1)
	if len(q) == 0 || a.photos[q[0]].Rel != rel {
		return ErrStale
	}
	mode, folder := "none", ""
	switch op {
	case OpSelect:
		mode, folder = a.cfg.SelectMode, selectedFolder
	case OpReject:
		mode, folder = a.cfg.RejectMode, rejectedFolder
	}
	e := Entry{Op: op, Path: rel}
	src := a.locateLocked(rel)
	if mode == "copy" || mode == "move" {
		// Keep the photo's sub-folder, e.g. Selected/2023/Goa/IMG_1.jpg.
		dst := uniqueDest(filepath.Join(a.cfg.Output, folder, filepath.FromSlash(path.Dir(rel))), path.Base(rel))
		var err error
		if mode == "move" {
			a.prev.Forget(src)
			err = moveFile(src, dst)
		} else {
			err = copyFile(src, dst)
		}
		if err != nil {
			return fmt.Errorf("couldn't %s %s: %v", mode, path.Base(rel), err)
		}
		r, _ := filepath.Rel(a.cfg.Output, dst)
		e.Dest, e.Moved = filepath.ToSlash(r), mode == "move"
	}
	if err := a.sess.Push(e); err != nil {
		return fmt.Errorf("couldn't save progress: %v", err)
	}
	if op != OpMaybe {
		a.prev.Forget(src)
	}
	a.prefetchLocked()
	return nil
}

// Undo reverses the most recent decision, including its file copy/move.
func (a *App) Undo() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.sess == nil {
		return nil
	}
	e, ok := a.sess.Last()
	if !ok {
		return errors.New("nothing to undo")
	}
	if e.Dest != "" {
		dst := filepath.Join(a.cfg.Output, filepath.FromSlash(e.Dest))
		if e.Moved {
			if err := moveFile(dst, filepath.Join(a.cfg.Source, filepath.FromSlash(e.Path))); err != nil {
				return fmt.Errorf("couldn't move the photo back: %v", err)
			}
		} else if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("couldn't remove the copy: %v", err)
		}
	}
	if err := a.sess.Undo(); err != nil {
		return fmt.Errorf("couldn't save progress: %v", err)
	}
	a.prefetchLocked()
	return nil
}

type PhotoInfo struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Folder   string `json:"folder"`
	Date     string `json:"date"`
	URL      string `json:"url"`
	Maybe    bool   `json:"maybe"`
	Position int    `json:"position"`
}

type State struct {
	Configured bool        `json:"configured"`
	Scanning   bool        `json:"scanning"`
	Phase      string      `json:"phase,omitempty"`
	Scanned    int         `json:"scanned"`
	Hidden     int         `json:"hidden"`
	Error      string      `json:"error,omitempty"`
	Total      int         `json:"total"`
	Selected   int         `json:"selected"`
	Rejected   int         `json:"rejected"`
	Maybe      int         `json:"maybe"`
	Undecided  int         `json:"undecided"`
	Target     int         `json:"target"`
	Current    *PhotoInfo  `json:"current"`
	Upcoming   []PhotoInfo `json:"upcoming"`
	Done       bool        `json:"done"`
	CanUndo    bool        `json:"canUndo"`
	Last       *Entry      `json:"last,omitempty"`
	Config     *Config     `json:"config,omitempty"`
}

func (a *App) State() State {
	a.mu.Lock()
	defer a.mu.Unlock()
	s := State{Configured: a.cfg != nil, Scanning: a.scanning, Phase: a.phase, Scanned: a.scanned, Hidden: a.hidden, Error: a.err, Config: a.cfg}
	if a.cfg != nil {
		s.Target = a.cfg.Target
	}
	if a.index == nil {
		return s
	}
	s.Total = len(a.photos)
	for _, p := range a.photos {
		e, _, ok := a.sess.Decision(p.Rel)
		switch {
		case !ok:
			s.Undecided++
		case e.Op == OpSelect:
			s.Selected++
		case e.Op == OpReject:
			s.Rejected++
		case e.Op == OpMaybe:
			s.Maybe++
		}
	}
	q := a.queueLocked(4)
	for k, i := range q {
		p := a.photos[i]
		e, _, ok := a.sess.Decision(p.Rel)
		info := PhotoInfo{
			Path:     p.Rel,
			Name:     path.Base(p.Rel),
			Folder:   path.Dir(p.Rel),
			URL:      "/photo?p=" + url.QueryEscape(p.Rel),
			Maybe:    ok && e.Op == OpMaybe,
			Position: i + 1,
		}
		if info.Folder == "." {
			info.Folder = ""
		}
		if w := p.When(); !w.IsZero() {
			info.Date = w.Format("2 Jan 2006, 15:04")
		} else {
			info.Date = "date unknown"
		}
		if k == 0 {
			s.Current = &info
		} else {
			s.Upcoming = append(s.Upcoming, info)
		}
	}
	s.Done = s.Current == nil
	if last, ok := a.sess.Last(); ok {
		s.CanUndo, s.Last = true, &last
	}
	return s
}

// ExportRows lists every decided photo for the CSV export.
func (a *App) ExportRows() [][]string {
	a.mu.Lock()
	defer a.mu.Unlock()
	rows := [][]string{{"photo", "decision", "saved_as", "decided_at"}}
	if a.index == nil {
		return rows
	}
	for _, p := range a.photos {
		if e, _, ok := a.sess.Decision(p.Rel); ok {
			rows = append(rows, []string{p.Rel, e.Op, e.Dest, e.T.Format(time.RFC3339)})
		}
	}
	return rows
}
