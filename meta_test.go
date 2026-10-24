package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fakeJPEG builds a minimal JPEG whose EXIF holds DateTimeOriginal.
func fakeJPEG(taken string, body string) []byte {
	le := binary.LittleEndian
	dt := append([]byte(taken), 0)
	// TIFF header, IFD0 with one entry (ExifIFD pointer), Exif IFD with one entry.
	tiff := []byte("II*\x00")
	tiff = le.AppendUint32(tiff, 8)
	tiff = le.AppendUint16(tiff, 1) // IFD0 at 8
	tiff = le.AppendUint16(tiff, 0x8769)
	tiff = le.AppendUint16(tiff, 4)
	tiff = le.AppendUint32(tiff, 1)
	tiff = le.AppendUint32(tiff, 26) // Exif IFD offset
	tiff = le.AppendUint32(tiff, 0)  // next IFD
	tiff = le.AppendUint16(tiff, 1)  // Exif IFD at 26
	tiff = le.AppendUint16(tiff, 0x9003)
	tiff = le.AppendUint16(tiff, 2)
	tiff = le.AppendUint32(tiff, uint32(len(dt)))
	tiff = le.AppendUint32(tiff, 44) // string offset
	tiff = le.AppendUint32(tiff, 0)
	tiff = append(tiff, dt...)
	seg := append([]byte("Exif\x00\x00"), tiff...)
	out := []byte{0xFF, 0xD8, 0xFF, 0xE1}
	out = binary.BigEndian.AppendUint16(out, uint16(len(seg)+2))
	out = append(out, seg...)
	out = append(out, 0xFF, 0xDA)
	return append(out, body...)
}

func TestExifTaken(t *testing.T) {
	got := exifTaken(fakeJPEG("2019:12:31 23:59:58", "x"))
	want := time.Date(2019, 12, 31, 23, 59, 58, 0, time.Local)
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if !exifTaken([]byte("not a jpeg")).IsZero() || !exifTaken([]byte{0xFF, 0xD8, 0xFF}).IsZero() {
		t.Fatal("garbage should give zero time")
	}
}

func TestDateOrderDedupeAndSubfolders(t *testing.T) {
	root := t.TempDir()
	src, out := filepath.Join(root, "src"), filepath.Join(root, "out")
	put := func(rel string, b []byte) {
		write(t, filepath.Join(src, rel), string(b))
		// File times are useless on copied drives; make them disagree with EXIF.
		_ = os.Chtimes(filepath.Join(src, rel), time.Now(), time.Now())
	}
	late := fakeJPEG("2020:05:01 10:00:00", "late")
	put("A/DSC1.JPG", late)
	put("A/DSC1_1.JPG", late) // byte-identical copy: hidden
	put("A/DSC1_2.JPG", fakeJPEG("2020:05:01 10:00:00", "other camera"))
	put("B/IMG_9.JPG", fakeJPEG("2018:01:01 08:00:00", "early"))
	put("B/IMG_9.CR2", []byte("raw version")) // RAW beats its JPEG
	a := startApp(t, Config{Source: src, Output: out, Sort: "date"})

	s := a.State()
	if s.Total != 3 || s.Hidden != 2 {
		t.Fatalf("total %d hidden %d", s.Total, s.Hidden)
	}
	var order []string
	for _, p := range a.photos {
		order = append(order, p.Rel)
	}
	// IMG_9.CR2 has no readable date, so it uses its file time (now) and sorts last.
	want := []string{"A/DSC1.JPG", "A/DSC1_2.JPG", "B/IMG_9.CR2"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order %v want %v", order, want)
		}
	}
	if err := a.Decide(current(t, a), OpSelect); err != nil {
		t.Fatal(err)
	}
	if !exists(filepath.Join(out, "Selected", "A", "DSC1.JPG")) {
		t.Fatal("select should keep the sub-folder")
	}
	// Second start uses the cache and gives the same result.
	a.sess.Close()
	b := startApp(t, Config{Source: src, Output: out, Sort: "date"})
	if s := b.State(); s.Total != 3 || s.Hidden != 2 || s.Selected != 1 {
		t.Fatalf("after resume: %+v", s)
	}
}
