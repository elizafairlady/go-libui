package draw

import (
	"fmt"
)

// Load loads pixel data into an image.
func (i *Image) Load(r Rectangle, data []byte) (int, error) {
	if i == nil || i.Display == nil {
		return -1, fmt.Errorf("load: nil image or display")
	}
	d := i.Display

	if !r.In(i.R) {
		return -1, fmt.Errorf("loadimage: bad rectangle")
	}

	bpl := bytesPerLine(r, i.Depth)
	n := bpl * r.Dy()
	if n > len(data) {
		return -1, fmt.Errorf("loadimage: insufficient data")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	chunk := d.bufsize - 64
	ndata := 0
	for r.Max.Y > r.Min.Y {
		dy := r.Dy()
		dx := r.Dx()
		if dy*bpl > chunk {
			dy = chunk / bpl
		}
		if dy <= 0 {
			dy = 1
			dx = ((chunk * dx) / bpl) & ^7
			nn := bytesPerLine(Rect(r.Min.X, r.Min.Y, r.Min.X+dx, r.Min.Y+dy), i.Depth)
			d.mu.Unlock()
			_, err := i.Load(Rect(r.Min.X+dx, r.Min.Y, r.Max.X, r.Min.Y+dy), data[nn:nn+(bpl-nn)])
			d.mu.Lock()
			if err != nil {
				return -1, err
			}
			n = nn
		} else {
			n = dy * bpl
		}

		a, err := d.bufimage(21 + n)
		if err != nil {
			return -1, fmt.Errorf("loadimage: %v", err)
		}
		a[0] = 'y'
		bplong(a[1:], uint32(i.id))
		bplong(a[5:], uint32(r.Min.X))
		bplong(a[9:], uint32(r.Min.Y))
		bplong(a[13:], uint32(r.Min.X+dx))
		bplong(a[17:], uint32(r.Min.Y+dy))
		copy(a[21:], data[:n])

		ndata += dy * bpl
		data = data[dy*bpl:]
		r.Min.Y += dy
	}

	return ndata, nil
}

// Unload reads pixel data from an image.
func (i *Image) Unload(r Rectangle, data []byte) (int, error) {
	if i == nil || i.Display == nil {
		return 0, fmt.Errorf("unloadimage: nil image or display")
	}
	d := i.Display

	if !r.In(i.R) {
		return 0, fmt.Errorf("unloadimage: bad rectangle")
	}

	bpl := bytesPerLine(r, i.Depth)
	total := bpl * r.Dy()
	if len(data) < total {
		return 0, fmt.Errorf("unloadimage: buffer too small: need %d, got %d", total, len(data))
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	chunk := d.bufsize - 64
	ndata := 0
	for r.Max.Y > r.Min.Y {
		dy := r.Dy()
		if dy*bpl > chunk {
			dy = chunk / bpl
			if dy <= 0 {
				return 0, fmt.Errorf("unloadimage: image too wide")
			}
		}
		n := dy * bpl

		a, err := d.bufimage(1 + 4 + 4*4)
		if err != nil {
			return ndata, err
		}
		a[0] = 'r'
		bplong(a[1:], uint32(i.id))
		bplong(a[5:], uint32(r.Min.X))
		bplong(a[9:], uint32(r.Min.Y))
		bplong(a[13:], uint32(r.Max.X))
		bplong(a[17:], uint32(r.Min.Y+dy))

		if err := d.doflush(); err != nil {
			return ndata, err
		}

		nn, err := d.datafd.Read(data[ndata : ndata+n])
		if err != nil || nn != n {
			if err != nil {
				return ndata, err
			}
			return ndata, fmt.Errorf("unloadimage: short read")
		}

		ndata += n
		r.Min.Y += dy
	}

	return ndata, nil
}

// Cload loads compressed image data.
func (i *Image) Cload(r Rectangle, data []byte) (int, error) {
	if i == nil || i.Display == nil {
		return 0, fmt.Errorf("cloadimage: nil image or display")
	}
	d := i.Display

	d.mu.Lock()
	defer d.mu.Unlock()

	chunk := d.bufsize - 64
	ndata := 0
	for len(data) > 0 && r.Max.Y > r.Min.Y {
		n := len(data)
		if n > chunk {
			n = chunk
		}

		a, err := d.bufimage(21 + n)
		if err != nil {
			return ndata, err
		}
		a[0] = 'Y'
		bplong(a[1:], uint32(i.id))
		bplong(a[5:], uint32(r.Min.X))
		bplong(a[9:], uint32(r.Min.Y))
		bplong(a[13:], uint32(r.Max.X))
		bplong(a[17:], uint32(r.Max.Y))
		copy(a[21:], data[:n])

		ndata += n
		data = data[n:]
	}

	return ndata, nil
}
