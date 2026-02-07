package draw

import (
	"compress/zlib"
	"fmt"
	"io"
	"os"
)

// WriteImage writes an image to a file.
func (i *Image) WriteImage(f *os.File) error {
	return i.writeImage(f, false)
}

// WriteimageFull writes an image to a file.
func (i *Image) WriteImageFile(name string) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return i.WriteImage(f)
}

// writeImage writes an image, optionally compressed.
func (i *Image) writeImage(w io.Writer, compressed bool) error {
	if i == nil || i.Display == nil {
		return fmt.Errorf("nil image")
	}

	// Write header
	// Format: chan[12] r.min.x[12] r.min.y[12] r.max.x[12] r.max.y[12]
	chanstr := chantostr(i.Pix)
	header := fmt.Sprintf("%11s %11d %11d %11d %11d ",
		chanstr, i.R.Min.X, i.R.Min.Y, i.R.Max.X, i.R.Max.Y)
	_, err := w.Write([]byte(header))
	if err != nil {
		return err
	}

	// Unload image data
	depth := i.Depth
	bpl := bytesPerLine(i.R, depth)
	data := make([]byte, bpl*i.R.Dy())
	_, err = i.Unload(i.R, data)
	if err != nil {
		return err
	}

	// Write data
	_, err = w.Write(data)
	return err
}

// CwriteImage writes a compressed image.
func (i *Image) CwriteImage(f *os.File) error {
	if i == nil || i.Display == nil {
		return fmt.Errorf("nil image")
	}

	// Write compressed marker
	_, err := f.Write([]byte("compressed\n"))
	if err != nil {
		return err
	}

	// Write header
	chanstr := chantostr(i.Pix)
	header := fmt.Sprintf("%11s %11d %11d %11d %11d ",
		chanstr, i.R.Min.X, i.R.Min.Y, i.R.Max.X, i.R.Max.Y)
	_, err = f.Write([]byte(header))
	if err != nil {
		return err
	}

	// Unload and compress image data
	depth := i.Depth
	bpl := bytesPerLine(i.R, depth)
	data := make([]byte, bpl*i.R.Dy())
	_, err = i.Unload(i.R, data)
	if err != nil {
		return err
	}

	zw := zlib.NewWriter(f)
	_, err = zw.Write(data)
	if err != nil {
		zw.Close()
		return err
	}
	return zw.Close()
}
