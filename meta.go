package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Photo metadata read from the file itself: when it was taken (EXIF) and a
// fingerprint for spotting byte-identical duplicates. Reading it means opening
// every file, which is slow on USB drives, so results are cached per project.

const headBytes = 128 << 10

type metaEntry struct {
	Size  int64     `json:"size"`
	MTime int64     `json:"mtime"`
	Taken time.Time `json:"taken,omitempty"`
	Head  string    `json:"head"`           // sha1 of the first headBytes
	Tail  string    `json:"tail,omitempty"` // sha1 of the last headBytes, only computed when heads collide
}

type metaCache struct {
	mu sync.Mutex
	m  map[string]*metaEntry
}

func metaPath(output string) string { return filepath.Join(stateDir(output), "meta.json") }

func loadMeta(output string) *metaCache {
	c := &metaCache{m: map[string]*metaEntry{}}
	if b, err := os.ReadFile(metaPath(output)); err == nil {
		_ = json.Unmarshal(b, &c.m)
	}
	return c
}

func (c *metaCache) save(output string) error { return saveJSON(metaPath(output), c.m) }

func (c *metaCache) get(p Photo) *metaEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.m[p.Rel]; e != nil && e.Size == p.Size && e.MTime == p.ModTime.Unix() {
		return e
	}
	return nil
}

func (c *metaCache) put(rel string, e *metaEntry) {
	c.mu.Lock()
	c.m[rel] = e
	c.mu.Unlock()
}

// readHead returns the capture time and head fingerprint of one file.
func readHead(file string) (time.Time, string, error) {
	f, err := os.Open(file)
	if err != nil {
		return time.Time{}, "", err
	}
	defer f.Close()
	buf := make([]byte, headBytes)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return time.Time{}, "", err
	}
	buf = buf[:n]
	sum := sha1.Sum(buf)
	return exifTaken(buf), hex.EncodeToString(sum[:]), nil
}

func readTail(file string, size int64) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	off := size - headBytes
	if off < 0 {
		off = 0
	}
	h := sha1.New()
	if _, err := io.Copy(h, io.NewSectionReader(f, off, size-off)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Enrich fills in Taken for every photo (reading files not yet cached) and
// returns the cache so duplicate detection can reuse the fingerprints.
func Enrich(root, output string, photos []Photo, progress func(n int)) *metaCache {
	cache := loadMeta(output)
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		done int
	)
	jobs := make(chan int)
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				p := &photos[i]
				e := cache.get(*p)
				if e == nil {
					taken, head, err := readHead(filepath.Join(root, filepath.FromSlash(p.Rel)))
					if err == nil {
						e = &metaEntry{Size: p.Size, MTime: p.ModTime.Unix(), Taken: taken, Head: head}
						cache.put(p.Rel, e)
					}
				}
				if e != nil {
					p.Taken = e.Taken
				}
				mu.Lock()
				done++
				if progress != nil && done%100 == 0 {
					progress(done)
				}
				mu.Unlock()
			}
		}()
	}
	for i := range photos {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	_ = cache.save(output)
	return cache
}

// formatRank orders file types by quality when the same shot exists in
// several formats (e.g. a camera's RAW + JPEG pair).
func formatRank(name string) int {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".dng", ".cr2", ".cr3", ".nef", ".arw", ".raf", ".orf", ".rw2", ".srw", ".pef":
		return 3
	case ".tif", ".tiff", ".heic", ".heif", ".png":
		return 2
	}
	return 1
}

// Dedupe drops byte-identical copies and, for shots saved in several formats
// (IMG_1.CR2 + IMG_1.JPG in the same folder), keeps only the best one.
// keep reports photos that must survive (already decided ones). It returns the
// remaining photos and how many were hidden.
func Dedupe(root string, photos []Photo, cache *metaCache, keep func(rel string) bool) ([]Photo, int) {
	better := func(a, b Photo) bool { // is a preferable to b?
		if ka, kb := keep(a.Rel), keep(b.Rel); ka != kb {
			return ka
		}
		if ra, rb := formatRank(a.Rel), formatRank(b.Rel); ra != rb {
			return ra > rb
		}
		if a.Size != b.Size {
			return a.Size > b.Size
		}
		return naturalLess(a.Rel, b.Rel)
	}
	pick := func(groups map[string][]int, alive []bool) {
		for _, g := range groups {
			if len(g) < 2 {
				continue
			}
			best := g[0]
			for _, i := range g[1:] {
				if better(photos[i], photos[best]) {
					best = i
				}
			}
			for _, i := range g {
				if i != best && !keep(photos[i].Rel) {
					alive[i] = false
				}
			}
		}
	}
	alive := make([]bool, len(photos))
	for i := range alive {
		alive[i] = true
	}

	// 1. Identical files: same size and head, confirmed by the tail.
	heads := map[string][]int{}
	for i, p := range photos {
		if e := cache.get(p); e != nil && e.Head != "" {
			k := fmt.Sprintf("%d/%s/", p.Size, e.Head)
			heads[k] = append(heads[k], i)
		}
	}
	full := map[string][]int{}
	for k, g := range heads {
		if len(g) < 2 {
			continue
		}
		for _, i := range g {
			e := cache.get(photos[i])
			if e.Tail == "" {
				t, err := readTail(filepath.Join(root, filepath.FromSlash(photos[i].Rel)), photos[i].Size)
				if err != nil {
					continue
				}
				e.Tail = t
			}
			full[k+e.Tail] = append(full[k+e.Tail], i)
		}
	}
	pick(full, alive)

	// 2. Same shot in several formats.
	stems := map[string][]int{}
	for i, p := range photos {
		if alive[i] {
			k := strings.ToLower(strings.TrimSuffix(p.Rel, path.Ext(p.Rel)))
			stems[k] = append(stems[k], i)
		}
	}
	pick(stems, alive)

	out := photos[:0:0]
	for i, p := range photos {
		if alive[i] {
			out = append(out, p)
		}
	}
	return out, len(photos) - len(out)
}

// exifTaken extracts DateTimeOriginal (or DateTime) from the start of a JPEG
// or TIFF-based RAW file. Zero if absent.
func exifTaken(b []byte) time.Time {
	var tiff []byte
	switch {
	case len(b) > 4 && b[0] == 0xFF && b[1] == 0xD8:
		for i := 2; i+4 <= len(b); {
			if b[i] != 0xFF {
				return time.Time{}
			}
			marker := b[i+1]
			if marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD7) || marker == 0x01 {
				i += 2
				continue
			}
			if marker == 0xDA || marker == 0xD9 {
				return time.Time{}
			}
			n := int(binary.BigEndian.Uint16(b[i+2:]))
			seg := b[i+4 : min(len(b), i+2+n)]
			if marker == 0xE1 && bytes.HasPrefix(seg, []byte("Exif\x00\x00")) {
				tiff = seg[6:]
				break
			}
			i += 2 + n
		}
	case bytes.HasPrefix(b, []byte("II*\x00")) || bytes.HasPrefix(b, []byte("MM\x00*")):
		tiff = b
	}
	if len(tiff) < 8 {
		return time.Time{}
	}
	var bo binary.ByteOrder = binary.LittleEndian
	if tiff[0] == 'M' {
		bo = binary.BigEndian
	}
	// ifd returns tag -> raw 12-byte entry for the IFD at off.
	ifd := func(off uint32) map[uint16][]byte {
		m := map[uint16][]byte{}
		if int(off)+2 > len(tiff) {
			return m
		}
		cnt := int(bo.Uint16(tiff[off:]))
		for k := 0; k < cnt; k++ {
			p := int(off) + 2 + k*12
			if p+12 > len(tiff) {
				break
			}
			m[bo.Uint16(tiff[p:])] = tiff[p : p+12]
		}
		return m
	}
	str := func(ent []byte) string {
		if ent == nil {
			return ""
		}
		n := bo.Uint32(ent[4:])
		var s []byte
		if n <= 4 {
			s = ent[8 : 8+n]
		} else if off := bo.Uint32(ent[8:]); int(off)+int(n) <= len(tiff) {
			s = tiff[off : off+n]
		}
		return strings.TrimRight(string(s), "\x00 ")
	}
	parse := func(s string) time.Time {
		t, err := time.ParseInLocation("2006:01:02 15:04:05", s, time.Local)
		if err != nil || t.Year() < 1971 {
			return time.Time{}
		}
		return t
	}
	ifd0 := ifd(bo.Uint32(tiff[4:]))
	if ent := ifd0[0x8769]; ent != nil {
		if t := parse(str(ifd(bo.Uint32(ent[8:]))[0x9003])); !t.IsZero() {
			return t
		}
	}
	return parse(str(ifd0[0x0132]))
}
