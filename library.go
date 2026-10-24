package main

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Photo is one image file found under the source folder.
type Photo struct {
	Rel     string // slash-separated path relative to the source folder
	ModTime time.Time
	Size    int64
	Taken   time.Time // EXIF capture time; zero if unknown
}

// When returns the best known date for a photo: capture time, else file time.
func (p Photo) When() time.Time {
	if !p.Taken.IsZero() {
		return p.Taken
	}
	return p.ModTime
}

var photoExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	".heic": true, ".heif": true, ".tif": true, ".tiff": true, ".bmp": true,
	// Camera RAW formats (previewed through macOS `sips`).
	".dng": true, ".cr2": true, ".cr3": true, ".nef": true, ".arw": true,
	".raf": true, ".orf": true, ".rw2": true, ".srw": true, ".pef": true,
}

// Folders that external drives and OSes litter around; never photos.
var skipDirs = map[string]bool{
	"$RECYCLE.BIN":              true,
	"System Volume Information": true,
	"lost+found":                true,
}

func isPhoto(name string) bool {
	return photoExts[strings.ToLower(filepath.Ext(name))]
}

// Scan walks root and returns every photo, skipping hidden files (including the
// "._" AppleDouble files macOS writes on exFAT drives) and the exclude folder.
// Unreadable sub-folders are skipped rather than aborting the whole scan.
func Scan(root, exclude string, progress func(n int)) ([]Photo, error) {
	var photos []Photo
	exclude = filepath.Clean(exclude)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (strings.HasPrefix(name, ".") || skipDirs[name] || filepath.Clean(path) == exclude) {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") || !d.Type().IsRegular() || !isPhoto(name) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		photos = append(photos, Photo{Rel: filepath.ToSlash(rel), ModTime: info.ModTime(), Size: info.Size()})
		if progress != nil && len(photos)%250 == 0 {
			progress(len(photos))
		}
		return nil
	})
	return photos, err
}

// SortPhotos orders photos by "name" (folder, then natural filename order so
// IMG_2 comes before IMG_10) or by "date" (capture time, else file time).
func SortPhotos(photos []Photo, by string) {
	sort.SliceStable(photos, func(i, j int) bool {
		if wi, wj := photos[i].When(), photos[j].When(); by == "date" && !wi.Equal(wj) {
			return wi.Before(wj)
		}
		return naturalLess(photos[i].Rel, photos[j].Rel)
	})
}

// naturalLess compares strings case-insensitively, treating digit runs as numbers.
func naturalLess(a, b string) bool {
	ra, rb := []rune(strings.ToLower(a)), []rune(strings.ToLower(b))
	i, j := 0, 0
	for i < len(ra) && j < len(rb) {
		if unicode.IsDigit(ra[i]) && unicode.IsDigit(rb[j]) {
			si, sj := i, j
			for i < len(ra) && unicode.IsDigit(ra[i]) {
				i++
			}
			for j < len(rb) && unicode.IsDigit(rb[j]) {
				j++
			}
			na := strings.TrimLeft(string(ra[si:i]), "0")
			nb := strings.TrimLeft(string(rb[sj:j]), "0")
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			if na != nb {
				return na < nb
			}
			continue
		}
		if ra[i] != rb[j] {
			// Sort the path separator first so a folder's files group together.
			if ra[i] == '/' || rb[j] == '/' {
				return ra[i] == '/'
			}
			return ra[i] < rb[j]
		}
		i++
		j++
	}
	if len(ra)-i != len(rb)-j {
		return len(ra)-i < len(rb)-j
	}
	return a < b
}
