package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// newProject makes a source folder with a few photos plus junk to ignore.
func newProject(t *testing.T) (src, out string) {
	t.Helper()
	root := t.TempDir()
	src = filepath.Join(root, "Photos")
	out = filepath.Join(root, "Photos - Selected")
	write(t, filepath.Join(src, "2023", "IMG_10.jpg"), "a10")
	write(t, filepath.Join(src, "2023", "IMG_2.JPG"), "a2")
	write(t, filepath.Join(src, "2024", "IMG_2.JPG"), "b2") // same name, other folder
	write(t, filepath.Join(src, "2024", "clip.heic"), "heic")
	write(t, filepath.Join(src, "2024", "._IMG_2.jpg"), "appledouble")
	write(t, filepath.Join(src, "2024", "notes.txt"), "not a photo")
	write(t, filepath.Join(src, ".Trashes", "x.jpg"), "trash")
	return src, out
}

func startApp(t *testing.T, c Config) *App {
	t.Helper()
	a := NewApp(filepath.Join(t.TempDir(), "config.json"), &Previewer{})
	if err := a.Start(c); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for a.State().Scanning {
		if time.Now().After(deadline) {
			t.Fatal("scan did not finish")
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Cleanup(func() { a.sess.Close() })
	return a
}

func current(t *testing.T, a *App) string {
	t.Helper()
	s := a.State()
	if s.Current == nil {
		t.Fatal("no current photo")
	}
	return s.Current.Path
}

func TestScanAndOrder(t *testing.T) {
	src, out := newProject(t)
	a := startApp(t, Config{Source: src, Output: out})
	var got []string
	for _, p := range a.photos {
		got = append(got, p.Rel)
	}
	want := []string{"2023/IMG_2.JPG", "2023/IMG_10.jpg", "2024/clip.heic", "2024/IMG_2.JPG"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestScanExcludesOutputInsideSource(t *testing.T) {
	src, _ := newProject(t)
	out := filepath.Join(src, "Picked")
	write(t, filepath.Join(out, "Selected", "old.jpg"), "x")
	a := startApp(t, Config{Source: src, Output: out})
	if a.State().Total != 4 {
		t.Fatalf("total = %d, want 4", a.State().Total)
	}
}

func TestDecideCopyRejectUndoAndResume(t *testing.T) {
	src, out := newProject(t)
	cfg := Config{Source: src, Output: out, Target: 2}
	a := startApp(t, cfg)

	if err := a.Decide(current(t, a), OpSelect); err != nil { // 2023/IMG_2.JPG
		t.Fatal(err)
	}
	if err := a.Decide(current(t, a), OpReject); err != nil { // 2023/IMG_10.jpg
		t.Fatal(err)
	}
	if err := a.Decide(current(t, a), OpMaybe); err != nil { // 2024/clip.heic
		t.Fatal(err)
	}
	if err := a.Decide(current(t, a), OpSelect); err != nil { // 2024/IMG_2.JPG: same name, own sub-folder
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(out, "Selected", "2023", "IMG_2.JPG")); string(b) != "a2" {
		t.Fatalf("first copy wrong: %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(out, "Selected", "2024", "IMG_2.JPG")); string(b) != "b2" {
		t.Fatalf("second copy wrong: %q", b)
	}
	if !exists(filepath.Join(src, "2023", "IMG_2.JPG")) {
		t.Fatal("copy mode must keep the original")
	}
	if exists(filepath.Join(out, "Rejected")) {
		t.Fatal("reject mode none must not create files")
	}

	s := a.State()
	if s.Selected != 2 || s.Rejected != 1 || s.Maybe != 1 || s.Undecided != 0 {
		t.Fatalf("counts: %+v", s)
	}
	if s.Current == nil || s.Current.Path != "2024/clip.heic" || !s.Current.Maybe {
		t.Fatalf("maybe photo should come back: %+v", s.Current)
	}

	if err := a.Undo(); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(out, "Selected", "2024", "IMG_2.JPG")) {
		t.Fatal("undo should delete the copy")
	}
	if got := current(t, a); got != "2024/IMG_2.JPG" {
		t.Fatalf("after undo current = %s", got)
	}

	// Deciding something that isn't the current photo is rejected.
	if err := a.Decide("2023/IMG_10.jpg", OpSelect); err != ErrStale {
		t.Fatalf("want ErrStale, got %v", err)
	}

	// Restart: progress comes back from the log.
	a.sess.Close()
	b := startApp(t, cfg)
	s = b.State()
	if s.Selected != 1 || s.Rejected != 1 || s.Maybe != 1 || s.Current.Path != "2024/IMG_2.JPG" {
		t.Fatalf("after resume: %+v current=%+v", s, s.Current)
	}
}

func TestMoveModeAndResume(t *testing.T) {
	src, out := newProject(t)
	cfg := Config{Source: src, Output: out, SelectMode: "move", RejectMode: "move"}
	a := startApp(t, cfg)

	first := current(t, a)
	if err := a.Decide(first, OpSelect); err != nil {
		t.Fatal(err)
	}
	if err := a.Decide(current(t, a), OpReject); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(src, "2023", "IMG_2.JPG")) || !exists(filepath.Join(out, "Selected", "2023", "IMG_2.JPG")) {
		t.Fatal("select should move the file")
	}
	if !exists(filepath.Join(out, "Rejected", "2023", "IMG_10.jpg")) {
		t.Fatal("reject should move the file")
	}
	if got := a.PhotoSrc(first); got != filepath.Join(out, "Selected", "2023", "IMG_2.JPG") {
		t.Fatalf("moved photo located at %s", got)
	}

	// Moved photos still count after a restart even though the source no longer has them.
	a.sess.Close()
	b := startApp(t, cfg)
	if s := b.State(); s.Total != 4 || s.Selected != 1 || s.Rejected != 1 {
		t.Fatalf("after resume: %+v", s)
	}
	// Undo moves the file back.
	if err := b.Undo(); err != nil {
		t.Fatal(err)
	}
	if !exists(filepath.Join(src, "2023", "IMG_10.jpg")) || exists(filepath.Join(out, "Rejected", "2023", "IMG_10.jpg")) {
		t.Fatal("undo should move the file back")
	}
}

func TestSessionIgnoresTornLine(t *testing.T) {
	out := t.TempDir()
	s, err := OpenSession(out)
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Push(Entry{Op: OpSelect, Path: "a.jpg"})
	_ = s.Push(Entry{Op: OpReject, Path: "b.jpg"})
	s.Close()
	f, _ := os.OpenFile(filepath.Join(stateDir(out), "decisions.jsonl"), os.O_APPEND|os.O_WRONLY, 0)
	f.WriteString(`{"t":"2024-01-01T00:00:00Z","op":"sel`)
	f.Close()

	s, err = OpenSession(out)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, _, ok := s.Decision("b.jpg"); !ok || len(s.stack) != 2 {
		t.Fatalf("stack = %+v", s.stack)
	}
}

func TestValidate(t *testing.T) {
	src, _ := newProject(t)
	for _, c := range []Config{
		{Source: "", Output: "/tmp/x"},
		{Source: filepath.Join(src, "missing"), Output: "/tmp/x"},
		{Source: src, Output: src},
		{Source: src, Output: filepath.Dir(src)},
	} {
		if err := c.Validate(); err == nil {
			t.Errorf("expected error for %+v", c)
		}
	}
	c := Config{Source: src, Output: src + " - Selected", SelectMode: "bogus"}
	if err := c.Validate(); err != nil || c.SelectMode != "copy" || c.RejectMode != "none" || c.Target != 500 {
		t.Fatalf("defaults: %+v %v", c, err)
	}
}

func TestCleanPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	cases := map[string]string{
		`  "/Volumes/My Drive/Photos/" `:     "/Volumes/My Drive/Photos",
		`/Volumes/My\ Drive/Photos`:          "/Volumes/My Drive/Photos",
		"file:///Volumes/My%20Drive/Photos/": "/Volumes/My Drive/Photos",
		"~/Pictures":                         filepath.Join(home, "Pictures"),
	}
	for in, want := range cases {
		if got := cleanPath(in); got != want {
			t.Errorf("cleanPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNaturalLess(t *testing.T) {
	in := []string{"b/IMG_10.jpg", "a/x.jpg", "b/IMG_9.jpg", "b/IMG_009b.jpg", "a b/y.jpg"}
	photos := make([]Photo, len(in))
	for i, p := range in {
		photos[i] = Photo{Rel: p}
	}
	SortPhotos(photos, "name")
	var got []string
	for _, p := range photos {
		got = append(got, p.Rel)
	}
	if strings.Join(got, ",") != "a/x.jpg,a b/y.jpg,b/IMG_9.jpg,b/IMG_009b.jpg,b/IMG_10.jpg" {
		t.Fatalf("got %v", got)
	}
}
