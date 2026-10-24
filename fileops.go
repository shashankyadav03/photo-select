package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// uniqueDest returns a path inside dir named like base that doesn't exist yet:
// IMG_0001.JPG, then IMG_0001 (2).JPG, IMG_0001 (3).JPG, ...
// Cameras reuse filenames, so collisions are common when flattening folders.
func uniqueDest(dir, base string) string {
	p := filepath.Join(dir, base)
	if _, err := os.Lstat(p); os.IsNotExist(err) {
		return p
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for n := 2; ; n++ {
		p = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", stem, n, ext))
		if _, err := os.Lstat(p); os.IsNotExist(err) {
			return p
		}
	}
}

// copyFile copies src to dst (which must not exist) via a temp file + rename,
// so a half-written copy never appears under the final name. Keeps the mtime.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".photo-select-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op once renamed
	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	_ = os.Chtimes(tmp.Name(), info.ModTime(), info.ModTime())
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("%s already exists", dst)
	}
	return os.Rename(tmp.Name(), dst)
}

// moveFile moves src to dst (which must not exist), falling back to
// copy + delete when they are on different drives.
func moveFile(src, dst string) error {
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("%s already exists", dst)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

// cleanPath tidies a folder path typed or pasted by a user: strips quotes,
// shell escapes from Terminal drag-and-drop, file:// URLs, and expands "~".
func cleanPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, `"'`)
	if strings.HasPrefix(p, "file://") {
		if u, err := url.Parse(p); err == nil {
			p = u.Path
		}
	}
	if !strings.Contains(p, `:\`) { // leave Windows paths alone
		p = strings.ReplaceAll(p, `\ `, " ")
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, p[1:])
		}
	}
	if p == "" {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	return filepath.Clean(p)
}

// isWithin reports whether path is dir or inside it.
func isWithin(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
