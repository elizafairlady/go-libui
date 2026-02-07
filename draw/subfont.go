package draw

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// Subfont cache: simple last-used cache matching 9front subfontcache.c.
var (
	subfontMu       sync.Mutex
	lastSubfontName string
	lastSubfont     *Subfont
)

// LookupSubfont looks up a subfont in the global cache.
// Port of 9front lookupsubfont().
func LookupSubfont(d *Display, name string) *Subfont {
	subfontMu.Lock()
	defer subfontMu.Unlock()

	if d != nil && name == "*default*" {
		return d.DefaultSubfont
	}
	if lastSubfontName == name && lastSubfont != nil {
		if d == nil || lastSubfont.Bits == nil || lastSubfont.Bits.Display == d {
			lastSubfont.ref++
			return lastSubfont
		}
	}
	return nil
}

// InstallSubfont installs a subfont in the global cache.
// Port of 9front installsubfont().
func InstallSubfont(name string, sf *Subfont) {
	subfontMu.Lock()
	defer subfontMu.Unlock()
	lastSubfontName = name
	lastSubfont = sf
}

// UninstallSubfont removes a subfont from the global cache.
// Port of 9front uninstallsubfont().
func UninstallSubfont(sf *Subfont) {
	subfontMu.Lock()
	defer subfontMu.Unlock()
	if sf == lastSubfont {
		lastSubfontName = ""
		lastSubfont = nil
	}
}

// AllocSubfont creates a new subfont.
// Port of 9front allocsubfont().
func AllocSubfont(name string, n, height, ascent int, info []Fontchar, i *Image) *Subfont {
	if height == 0 {
		return nil
	}
	sf := &Subfont{
		N:      n,
		Height: height,
		Ascent: ascent,
		Info:   info,
		Bits:   i,
		ref:    1,
	}
	if name != "" {
		sf.Name = name
		InstallSubfont(name, sf)
	}
	return sf
}

// Free releases the subfont resources.
// Port of 9front freesubfont().
func (sf *Subfont) Free() {
	if sf == nil {
		return
	}
	sf.ref--
	if sf.ref > 0 {
		return
	}
	UninstallSubfont(sf)
	if sf.Bits != nil {
		sf.Bits.Free()
		sf.Bits = nil
	}
	sf.Info = nil
}

// ReadSubfonti reads a subfont from a reader, optionally with an already-read image.
// Port of 9front readsubfonti().
// The format after the image data is:
//
//	n[12] height[12] ascent[12]   (3 × 12-char decimal fields)
//	info[(n+1)*6]                 (6-byte fontchar entries)
func ReadSubfonti(d *Display, name string, r io.Reader, ai *Image) (*Subfont, error) {
	var i *Image
	var err error

	if ai != nil {
		i = ai
	} else {
		i, err = d.ReadImageReader(r)
		if err != nil {
			return nil, fmt.Errorf("readsubfont: image read error: %v", err)
		}
	}

	// Read 3*12 byte header
	hdr := make([]byte, 3*12)
	if _, err := io.ReadFull(r, hdr); err != nil {
		if ai == nil {
			i.Free()
		}
		return nil, fmt.Errorf("readsubfont: header read error: %v", err)
	}

	n := atoi12(hdr[0:12])
	height := atoi12(hdr[12:24])
	ascent := atoi12(hdr[24:36])

	if n <= 0 || n > 0x7FFF {
		if ai == nil {
			i.Free()
		}
		return nil, fmt.Errorf("readsubfont: bad fontchar count %d", n)
	}

	// Read (n+1) * 6 bytes of fontchar data
	p := make([]byte, 6*(n+1))
	if _, err := io.ReadFull(r, p); err != nil {
		if ai == nil {
			i.Free()
		}
		return nil, fmt.Errorf("readsubfont: fontchar read error: %v", err)
	}

	fc := unpackInfo(p, n)
	sf := AllocSubfont(name, n, height, ascent, fc, i)
	if sf == nil {
		if ai == nil {
			i.Free()
		}
		return nil, fmt.Errorf("readsubfont: allocsubfont failed")
	}
	return sf, nil
}

// ReadSubfont reads a subfont from a reader.
// Port of 9front readsubfont().
func ReadSubfont(d *Display, name string, r io.Reader) (*Subfont, error) {
	return ReadSubfonti(d, name, r, nil)
}

// unpackInfo unpacks n+1 fontchar entries from 6-byte packed form.
// Port of 9front _unpackinfo().
func unpackInfo(p []byte, n int) []Fontchar {
	fc := make([]Fontchar, n+1)
	for i := 0; i <= n; i++ {
		off := i * 6
		fc[i] = Fontchar{
			X:      int(p[off]) | int(p[off+1])<<8,
			Top:    p[off+2],
			Bottom: p[off+3],
			Left:   int8(p[off+4]),
			Width:  p[off+5],
		}
	}
	return fc
}

// packInfo packs fontchar entries into 6-byte packed form.
func packInfo(fc []Fontchar, n int) []byte {
	p := make([]byte, 6*(n+1))
	for i := 0; i <= n && i < len(fc); i++ {
		off := i * 6
		p[off] = byte(fc[i].X)
		p[off+1] = byte(fc[i].X >> 8)
		p[off+2] = fc[i].Top
		p[off+3] = fc[i].Bottom
		p[off+4] = byte(fc[i].Left)
		p[off+5] = fc[i].Width
	}
	return p
}

// WriteSubfont writes subfont info to a writer.
// Port of 9front writesubfont().
func WriteSubfont(w io.Writer, sf *Subfont) error {
	// Write header: n[12] height[12] ascent[12]
	hdr := fmt.Sprintf("%12d%12d%12d", sf.N, sf.Height, sf.Ascent)
	if _, err := w.Write([]byte(hdr)); err != nil {
		return err
	}
	// Write fontchar data
	p := packInfo(sf.Info, sf.N)
	_, err := w.Write(p)
	return err
}

// OpenSubfont opens a subfont file.
func (d *Display) OpenSubfont(name string) (*Subfont, error) {
	// Check cache first
	sf := LookupSubfont(d, name)
	if sf != nil {
		return sf, nil
	}

	f, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("cannot open subfont: %s: %v", name, err)
	}
	defer f.Close()

	return ReadSubfont(d, name, f)
}

// openSubfont is the internal version that returns nil on error.
func (d *Display) openSubfont(name string) *Subfont {
	sf, _ := d.OpenSubfont(name)
	return sf
}

// CharWidth returns the width of character i in the subfont.
func (sf *Subfont) CharWidth(i int) int {
	if sf == nil || i < 0 || i >= sf.N {
		return 0
	}
	return int(sf.Info[i].Width)
}

// CharInfo returns the Fontchar info for character i.
func (sf *Subfont) CharInfo(i int) *Fontchar {
	if sf == nil || i < 0 || i >= sf.N {
		return nil
	}
	return &sf.Info[i]
}

// atoi12 parses a 12-char decimal string field (Plan 9 image format).
func atoi12(b []byte) int {
	return atoi(string(b))
}
