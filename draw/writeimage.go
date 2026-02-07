package draw

import (
	"compress/zlib"
	"fmt"
	"io"
	"os"
)

// Compression constants from draw.h / computil.c.
const (
	NMATCH  = 3 // minimum match length
	NRUN    = 12 + NMATCH
	NDUMP   = 128 // maximum dump length
	NMEM    = 1024
	NCBLOCK = 6000 // max size of compressed block
)

// CompBlockSize returns the maximum compressed block size for an image.
// Port of 9front _compblocksize().
func CompBlockSize(r Rectangle, depth int) int {
	bpl := bytesPerLine(r, depth)
	return bpl * r.Dy()
}

// WriteImage writes an uncompressed image to a writer.
// Port of 9front writeimage() (uncompressed path).
func (i *Image) WriteImage(f *os.File) error {
	return i.WriteImageWriter(f)
}

// WriteImageWriter writes an uncompressed image to an io.Writer.
func (i *Image) WriteImageWriter(w io.Writer) error {
	if i == nil {
		return fmt.Errorf("writeimage: nil image")
	}

	// Write header: chan[12] r.min.x[12] r.min.y[12] r.max.x[12] r.max.y[12]
	chanstr := chantostr(i.Pix)
	header := fmt.Sprintf("%11s %11d %11d %11d %11d ",
		chanstr, i.R.Min.X, i.R.Min.Y, i.R.Max.X, i.R.Max.Y)
	if _, err := w.Write([]byte(header)); err != nil {
		return err
	}

	// We can only unload from display-backed images
	if i.Display != nil {
		depth := i.Depth
		bpl := bytesPerLine(i.R, depth)
		data := make([]byte, bpl*i.R.Dy())
		_, err := i.Unload(i.R, data)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}

	return nil
}

// WriteImageFile writes an image to a file by name.
func (i *Image) WriteImageFile(name string) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return i.WriteImage(f)
}

// CwriteImage writes a compressed image.
// The 9front format uses "compressed\n" marker followed by the header,
// then compressed blocks with per-block headers.
// We use Go's zlib for the compression.
func (i *Image) CwriteImage(f *os.File) error {
	return i.CwriteImageWriter(f)
}

// CwriteImageWriter writes a compressed image to an io.Writer.
func (i *Image) CwriteImageWriter(w io.Writer) error {
	if i == nil {
		return fmt.Errorf("cwriteimage: nil image")
	}

	// Write compressed marker
	if _, err := w.Write([]byte("compressed\n")); err != nil {
		return err
	}

	// Write header
	chanstr := chantostr(i.Pix)
	header := fmt.Sprintf("%11s %11d %11d %11d %11d ",
		chanstr, i.R.Min.X, i.R.Min.Y, i.R.Max.X, i.R.Max.Y)
	if _, err := w.Write([]byte(header)); err != nil {
		return err
	}

	if i.Display == nil {
		return nil
	}

	// Unload and compress image data
	depth := i.Depth
	bpl := bytesPerLine(i.R, depth)
	data := make([]byte, bpl*i.R.Dy())
	if _, err := i.Unload(i.R, data); err != nil {
		return err
	}

	zw := zlib.NewWriter(w)
	if _, err := zw.Write(data); err != nil {
		zw.Close()
		return err
	}
	return zw.Close()
}

// WriteImageHeader writes just the image header to a writer.
// This is useful for building subfont files etc.
func WriteImageHeader(w io.Writer, pix Pix, r Rectangle) error {
	chanstr := chantostr(pix)
	header := fmt.Sprintf("%11s %11d %11d %11d %11d ",
		chanstr, r.Min.X, r.Min.Y, r.Max.X, r.Max.Y)
	_, err := w.Write([]byte(header))
	return err
}
