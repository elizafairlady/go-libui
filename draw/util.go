package draw

import (
	"fmt"
	"io"
)

// atoi parses a decimal integer from a string, ignoring whitespace.
func atoi(s string) int {
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

// trimSpace trims leading and trailing spaces and tabs.
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

// unitsPerLine returns the number of units of bitsperunit bits needed
// to cover pixels from r.Min.X to r.Max.X at depth d.
func unitsPerLine(r Rectangle, d int, bitsperunit int) int {
	if d <= 0 || d > 32 {
		return 0
	}
	return (r.Max.X*d - (r.Min.X * d & -bitsperunit) + bitsperunit - 1) / bitsperunit
}

// wordsPerLine returns 32-bit words per scan line.
func wordsPerLine(r Rectangle, d int) int {
	return unitsPerLine(r, d, 32)
}

// bytesPerLine returns bytes per scan line.
func bytesPerLine(r Rectangle, d int) int {
	return unitsPerLine(r, d, 8)
}

// ReadImageReader reads an image from an io.Reader (not just *os.File).
func (d *Display) ReadImageReader(r io.Reader) (*Image, error) {
	// Read image header: 5 × 12 bytes
	header := make([]byte, 5*12)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("readimage: header read error: %v", err)
	}

	chanstr := string(header[0:11])
	pix := strtochan(chanstr)
	if pix == 0 {
		return nil, fmt.Errorf("readimage: bad channel string: %s", chanstr)
	}

	minx := atoi(string(header[12:23]))
	miny := atoi(string(header[24:35]))
	maxx := atoi(string(header[36:47]))
	maxy := atoi(string(header[48:59]))

	rect := Rect(minx, miny, maxx, maxy)

	img, err := d.AllocImage(rect, pix, false, DTransparent)
	if err != nil {
		return nil, err
	}

	depth := chantodepth(pix)
	bpl := bytesPerLine(rect, depth)
	data := make([]byte, bpl*rect.Dy())
	if _, err := io.ReadFull(r, data); err != nil {
		img.Free()
		return nil, fmt.Errorf("readimage: data read error: %v", err)
	}

	if _, err := img.Load(rect, data); err != nil {
		img.Free()
		return nil, err
	}

	return img, nil
}
