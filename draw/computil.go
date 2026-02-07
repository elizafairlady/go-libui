package draw

import (
	"fmt"
)

// Load loads pixel data into an image.
func (i *Image) Load(r Rectangle, data []byte) error {
	if i == nil || i.Display == nil {
		return fmt.Errorf("nil image or display")
	}
	d := i.Display

	// Clip to image bounds
	r, ok := r.Clip(i.R)
	if !ok || r.Empty() {
		return nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Calculate bytes per line
	depth := i.Depth
	bpl := bytesPerLine(r, depth)
	if bpl <= 0 {
		return nil
	}

	// Calculate total data needed
	total := bpl * r.Dy()
	if len(data) < total {
		return fmt.Errorf("data too short: need %d, got %d", total, len(data))
	}

	// Send data in chunks that fit in the buffer
	// 'y' command: load image
	// Format: 'y' id[4] r[4*4] data[...]
	maxChunk := len(d.buf) - 1 - 4 - 16 - 10 // leave room for header and flush

	y := r.Min.Y
	offset := 0
	for y < r.Max.Y {
		// Calculate how many lines fit in this chunk
		lines := (maxChunk / bpl)
		if lines < 1 {
			lines = 1
		}
		if y+lines > r.Max.Y {
			lines = r.Max.Y - y
		}

		chunk := bpl * lines
		rr := Rect(r.Min.X, y, r.Max.X, y+lines)

		// Build 'y' message
		var a [1 + 4 + 4*4]byte
		a[0] = 'y'
		bplong(a[1:], uint32(i.id))
		bplong(a[5:], uint32(rr.Min.X))
		bplong(a[9:], uint32(rr.Min.Y))
		bplong(a[13:], uint32(rr.Max.X))
		bplong(a[17:], uint32(rr.Max.Y))

		if err := d.flushBuffer(len(a) + chunk); err != nil {
			return err
		}
		copy(d.buf[d.bufsize:], a[:])
		d.bufsize += len(a)
		copy(d.buf[d.bufsize:], data[offset:offset+chunk])
		d.bufsize += chunk

		y += lines
		offset += chunk
	}

	return nil
}

// Unload reads pixel data from an image.
func (i *Image) Unload(r Rectangle, data []byte) (int, error) {
	if i == nil || i.Display == nil {
		return 0, fmt.Errorf("nil image or display")
	}
	d := i.Display

	// Clip to image bounds
	r, ok := r.Clip(i.R)
	if !ok || r.Empty() {
		return 0, nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Calculate bytes needed
	depth := i.Depth
	bpl := bytesPerLine(r, depth)
	total := bpl * r.Dy()
	if len(data) < total {
		return 0, fmt.Errorf("data buffer too small: need %d, got %d", total, len(data))
	}

	// Build 'r' (read) message
	// Format: 'r' id[4] r[4*4]
	var a [1 + 4 + 4*4]byte
	a[0] = 'r'
	bplong(a[1:], uint32(i.id))
	bplong(a[5:], uint32(r.Min.X))
	bplong(a[9:], uint32(r.Min.Y))
	bplong(a[13:], uint32(r.Max.X))
	bplong(a[17:], uint32(r.Max.Y))

	if err := d.flushBuffer(len(a)); err != nil {
		return 0, err
	}
	copy(d.buf[d.bufsize:], a[:])
	d.bufsize += len(a)

	// Flush to send the request
	if err := d.flush(false); err != nil {
		return 0, err
	}

	// Read response
	n, err := d.datafd.Read(data[:total])
	if err != nil {
		return 0, err
	}

	return n, nil
}

// Cload loads compressed image data.
func (i *Image) Cload(r Rectangle, data []byte) error {
	if i == nil || i.Display == nil {
		return fmt.Errorf("nil image or display")
	}
	d := i.Display

	d.mu.Lock()
	defer d.mu.Unlock()

	// Build 'Y' (compressed load) message
	// Format: 'Y' id[4] r[4*4] data[...]
	var a [1 + 4 + 4*4]byte
	a[0] = 'Y'
	bplong(a[1:], uint32(i.id))
	bplong(a[5:], uint32(r.Min.X))
	bplong(a[9:], uint32(r.Min.Y))
	bplong(a[13:], uint32(r.Max.X))
	bplong(a[17:], uint32(r.Max.Y))

	if err := d.flushBuffer(len(a) + len(data)); err != nil {
		return err
	}
	copy(d.buf[d.bufsize:], a[:])
	d.bufsize += len(a)
	copy(d.buf[d.bufsize:], data)
	d.bufsize += len(data)

	return nil
}
