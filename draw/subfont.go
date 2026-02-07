package draw

import (
	"encoding/binary"
	"fmt"
	"os"
)

// OpenSubfont opens a subfont file.
func (d *Display) OpenSubfont(name string) (*Subfont, error) {
	sf := d.openSubfont(name)
	if sf == nil {
		return nil, fmt.Errorf("cannot open subfont: %s", name)
	}
	return sf, nil
}

func (d *Display) openSubfont(name string) *Subfont {
	f, err := os.Open(name)
	if err != nil {
		return nil
	}
	defer f.Close()

	sf, err := d.readSubfont(f, name)
	if err != nil {
		return nil
	}
	return sf
}

// readSubfont reads a subfont from an image file.
func (d *Display) readSubfont(f *os.File, name string) (*Subfont, error) {
	// Read the image first
	img, err := d.ReadImage(f)
	if err != nil {
		return nil, err
	}

	// Read the subfont header from the end of the file
	// Subfont data comes after image data
	// Format: n[2] height[1] ascent[1] info[n+1][6]

	// Actually, subfonts have their glyph info embedded
	// The format is: the image followed by character descriptions

	// Read character info
	// Seek to find the character info after the image
	// For now, try to read from a .subfont companion file
	// or embedded in the image file

	// Simplified: assume fixed-width font for now
	n := 256 // assume 256 characters
	height := img.R.Dy()
	ascent := height * 3 / 4

	info := make([]Fontchar, n+1)
	charWidth := img.R.Dx() / n
	if charWidth < 1 {
		charWidth = 1
	}

	for i := 0; i <= n; i++ {
		info[i] = Fontchar{
			X:      i * charWidth,
			Top:    0,
			Bottom: byte(height),
			Left:   0,
			Width:  byte(charWidth),
		}
	}

	return &Subfont{
		Name:   name,
		N:      n,
		Height: height,
		Ascent: ascent,
		Info:   info,
		Bits:   img,
		ref:    1,
	}, nil
}

// AllocSubfont creates a new subfont from an image and character info.
func (d *Display) AllocSubfont(name string, height, ascent, n int, info []Fontchar, bits *Image) *Subfont {
	if len(info) < n+1 {
		return nil
	}
	return &Subfont{
		Name:   name,
		N:      n,
		Height: height,
		Ascent: ascent,
		Info:   info,
		Bits:   bits,
		ref:    1,
	}
}

// Free releases the subfont resources.
func (sf *Subfont) Free() {
	if sf == nil {
		return
	}
	sf.ref--
	if sf.ref > 0 {
		return
	}
	if sf.Bits != nil {
		sf.Bits.Free()
		sf.Bits = nil
	}
}

// InstallSubfont installs a subfont in a font's cache.
func (f *Font) InstallSubfont(name string, sf *Subfont) {
	if f == nil || sf == nil {
		return
	}
	// Find or create the cachefont entry
	for i := range f.sub {
		if f.sub[i] != nil && f.sub[i].Name == name {
			// Install into subf cache
			f.subf = append(f.subf, Cachesubf{cf: f.sub[i], f: sf})
			sf.ref++
			return
		}
	}
}

// LookupSubfont finds a cached subfont by name.
func (f *Font) LookupSubfont(name string) *Subfont {
	if f == nil {
		return nil
	}
	for i := range f.subf {
		if f.subf[i].f != nil && f.subf[i].f.Name == name {
			return f.subf[i].f
		}
	}
	return nil
}

// ReadSubfontFile reads subfont info from a file.
// The format is: n[2] height[1] ascent[1] info[n+1][6]
func ReadSubfontFile(f *os.File) (height, ascent, n int, info []Fontchar, err error) {
	// Read header: 2 bytes for n, 1 for height, 1 for ascent
	header := make([]byte, 4)
	_, err = f.Read(header)
	if err != nil {
		return
	}

	n = int(binary.LittleEndian.Uint16(header[0:2]))
	height = int(header[2])
	ascent = int(header[3])

	// Read fontchar info: (n+1) entries of 6 bytes each
	info = make([]Fontchar, n+1)
	buf := make([]byte, 6)
	for i := 0; i <= n; i++ {
		_, err = f.Read(buf)
		if err != nil {
			return
		}
		info[i] = Fontchar{
			X:      int(binary.LittleEndian.Uint16(buf[0:2])),
			Top:    buf[2],
			Bottom: buf[3],
			Left:   int8(buf[4]),
			Width:  buf[5],
		}
	}

	return height, ascent, n, info, nil
}

// WriteSubfontFile writes subfont info to a file.
func WriteSubfontFile(f *os.File, height, ascent, n int, info []Fontchar) error {
	// Write header
	header := make([]byte, 4)
	binary.LittleEndian.PutUint16(header[0:2], uint16(n))
	header[2] = byte(height)
	header[3] = byte(ascent)
	_, err := f.Write(header)
	if err != nil {
		return err
	}

	// Write fontchar info
	buf := make([]byte, 6)
	for i := 0; i <= n && i < len(info); i++ {
		binary.LittleEndian.PutUint16(buf[0:2], uint16(info[i].X))
		buf[2] = info[i].Top
		buf[3] = info[i].Bottom
		buf[4] = byte(info[i].Left)
		buf[5] = info[i].Width
		_, err = f.Write(buf)
		if err != nil {
			return err
		}
	}

	return nil
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

// ReadImage reads an image from a file.
func (d *Display) ReadImage(f *os.File) (*Image, error) {
	// Read image header
	// Format: chan[12] r.min.x[12] r.min.y[12] r.max.x[12] r.max.y[12]
	header := make([]byte, 5*12)
	n, err := f.Read(header)
	if err != nil {
		return nil, err
	}
	if n < 5*12 {
		return nil, fmt.Errorf("short image header")
	}

	chanstr := string(header[0:11])
	pix := strtochan(chanstr)
	if pix == 0 {
		return nil, fmt.Errorf("bad channel string: %s", chanstr)
	}

	minx := atoi(string(header[12:23]))
	miny := atoi(string(header[24:35]))
	maxx := atoi(string(header[36:47]))
	maxy := atoi(string(header[48:59]))

	r := Rect(minx, miny, maxx, maxy)

	// Allocate the image
	img, err := d.AllocImage(r, pix, false, DTransparent)
	if err != nil {
		return nil, err
	}

	// Read and load image data
	depth := chantodepth(pix)
	bpl := bytesPerLine(r, depth)
	data := make([]byte, bpl*r.Dy())
	_, err = f.Read(data)
	if err != nil {
		img.Free()
		return nil, err
	}

	err = img.Load(r, data)
	if err != nil {
		img.Free()
		return nil, err
	}

	return img, nil
}

func atoi(s string) int {
	// Parse number, ignoring leading/trailing whitespace
	s = trimSpace(s)
	n := 0
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	if neg {
		return -n
	}
	return n
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func bytesPerLine(r Rectangle, depth int) int {
	if depth <= 0 {
		return 0
	}
	w := r.Dx()
	if w <= 0 {
		return 0
	}
	bits := w * depth
	return (bits + 7) / 8
}
