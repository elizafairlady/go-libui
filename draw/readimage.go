package draw

import (
	"compress/zlib"
	"fmt"
	"io"
	"os"
)

// ReadImageFile reads an image from a file by name.
func (d *Display) ReadImageFile(name string) (*Image, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return d.ReadImage(f)
}

// Creadimage reads a compressed image.
func (d *Display) Creadimage(f io.Reader) (*Image, error) {
	// Read header
	header := make([]byte, 5*12)
	n, err := io.ReadFull(f, header)
	if err != nil {
		return nil, err
	}
	if n < 5*12 {
		return nil, fmt.Errorf("short header")
	}

	// Check for compressed marker
	if string(header[0:11]) != "compressed\n" {
		return nil, fmt.Errorf("not a compressed image")
	}

	// Read the actual image header
	_, err = io.ReadFull(f, header)
	if err != nil {
		return nil, err
	}

	chanstr := trimSpace(string(header[0:11]))
	pix := strtochan(chanstr)
	if pix == 0 {
		return nil, fmt.Errorf("bad channel: %s", chanstr)
	}

	minx := atoi(string(header[12:23]))
	miny := atoi(string(header[24:35]))
	maxx := atoi(string(header[36:47]))
	maxy := atoi(string(header[48:59]))
	r := Rect(minx, miny, maxx, maxy)

	// Allocate image
	img, err := d.AllocImage(r, pix, false, DTransparent)
	if err != nil {
		return nil, err
	}

	// Decompress and load
	zr, err := zlib.NewReader(f)
	if err != nil {
		img.Free()
		return nil, err
	}
	defer zr.Close()

	depth := chantodepth(pix)
	bpl := bytesPerLine(r, depth)
	data := make([]byte, bpl*r.Dy())
	_, err = io.ReadFull(zr, data)
	if err != nil {
		img.Free()
		return nil, err
	}

	_, err = img.Load(r, data)
	if err != nil {
		img.Free()
		return nil, err
	}

	return img, nil
}

// ReadNImage reads n bytes of image data from a reader.
func ReadNImage(r io.Reader, n int) ([]byte, error) {
	data := make([]byte, n)
	_, err := io.ReadFull(r, data)
	return data, err
}
