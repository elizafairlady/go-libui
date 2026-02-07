package draw

import (
	"fmt"
)

// AllocImage allocates a new image with the given rectangle, pixel format,
// replication flag, and fill color.
func (d *Display) AllocImage(r Rectangle, pix Pix, repl bool, val uint32) (*Image, error) {
	return d.allocImage(nil, r, pix, repl, val, 0)
}

// allocImage is the internal implementation of AllocImage.
func (d *Display) allocImage(ai *Image, r Rectangle, pix Pix, repl bool, val uint32, screenid int) (*Image, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	depth := chantodepth(pix)
	if depth == 0 {
		return nil, fmt.Errorf("allocimage: bad channel descriptor")
	}

	// Allocate image id
	id := d.imageid
	d.imageid++

	// Build the 'b' (allocate) message
	// Format: 'b' id[4] screenid[4] refresh[1] chan[4] repl[1] r[4*4] clipr[4*4] color[4]
	var a [1 + 4 + 4 + 1 + 4 + 1 + 4*4 + 4*4 + 4]byte
	a[0] = 'b'
	bplong(a[1:], uint32(id))
	bplong(a[5:], uint32(screenid))
	a[9] = 0 // refresh
	bplong(a[10:], uint32(pix))
	if repl {
		a[14] = 1
	} else {
		a[14] = 0
	}
	bplong(a[15:], uint32(r.Min.X))
	bplong(a[19:], uint32(r.Min.Y))
	bplong(a[23:], uint32(r.Max.X))
	bplong(a[27:], uint32(r.Max.Y))
	// Clip rectangle - same as r for non-replicating, large for replicating
	var clipr Rectangle
	if repl {
		clipr = Rect(-0x3FFFFFFF, -0x3FFFFFFF, 0x3FFFFFFF, 0x3FFFFFFF)
	} else {
		clipr = r
	}
	bplong(a[31:], uint32(clipr.Min.X))
	bplong(a[35:], uint32(clipr.Min.Y))
	bplong(a[39:], uint32(clipr.Max.X))
	bplong(a[43:], uint32(clipr.Max.Y))
	bplong(a[47:], val)

	if err := d.flushBuffer(len(a)); err != nil {
		return nil, err
	}
	copy(d.buf[d.bufsize:], a[:])
	d.bufsize += len(a)

	img := &Image{
		Display: d,
		id:      id,
		Pix:     pix,
		Depth:   depth,
		Repl:    repl,
		R:       r,
		Clipr:   clipr,
	}

	return img, nil
}

// Free releases the resources associated with an image.
func (i *Image) Free() error {
	if i == nil || i.Display == nil {
		return nil
	}
	d := i.Display

	d.mu.Lock()
	defer d.mu.Unlock()

	// Build the 'f' (free) message
	// Format: 'f' id[4]
	var a [1 + 4]byte
	a[0] = 'f'
	bplong(a[1:], uint32(i.id))

	if err := d.flushBuffer(len(a)); err != nil {
		return err
	}
	copy(d.buf[d.bufsize:], a[:])
	d.bufsize += len(a)

	i.Display = nil
	return nil
}

// AllocScreen allocates a new screen backed by image fill.
func (d *Display) AllocScreen(image, fill *Image, public bool) (*Screen, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	id := d.imageid
	d.imageid++

	// Build the 'A' (alloc screen) message
	// Format: 'A' id[4] imageid[4] fillid[4] public[1]
	var a [1 + 4 + 4 + 4 + 1]byte
	a[0] = 'A'
	bplong(a[1:], uint32(id))
	bplong(a[5:], uint32(image.id))
	bplong(a[9:], uint32(fill.id))
	if public {
		a[13] = 1
	}

	if err := d.flushBuffer(len(a)); err != nil {
		return nil, err
	}
	copy(d.buf[d.bufsize:], a[:])
	d.bufsize += len(a)

	return &Screen{
		Display: d,
		id:      id,
		Image:   image,
		Fill:    fill,
	}, nil
}

// Free releases the resources associated with a screen.
func (s *Screen) Free() error {
	if s == nil || s.Display == nil {
		return nil
	}
	d := s.Display

	d.mu.Lock()
	defer d.mu.Unlock()

	// Build the 'F' (free screen) message
	// Format: 'F' id[4]
	var a [1 + 4]byte
	a[0] = 'F'
	bplong(a[1:], uint32(s.id))

	if err := d.flushBuffer(len(a)); err != nil {
		return err
	}
	copy(d.buf[d.bufsize:], a[:])
	d.bufsize += len(a)

	s.Display = nil
	return nil
}

// AllocWindow allocates a window on a screen.
func (s *Screen) AllocWindow(r Rectangle, ref int, val uint32) (*Image, error) {
	d := s.Display

	d.mu.Lock()
	defer d.mu.Unlock()

	id := d.imageid
	d.imageid++

	pix := s.Image.Pix
	depth := s.Image.Depth

	// Build the 'b' message with screen id
	var a [1 + 4 + 4 + 1 + 4 + 1 + 4*4 + 4*4 + 4]byte
	a[0] = 'b'
	bplong(a[1:], uint32(id))
	bplong(a[5:], uint32(s.id))
	a[9] = byte(ref)
	bplong(a[10:], uint32(pix))
	a[14] = 0 // not replicating
	bplong(a[15:], uint32(r.Min.X))
	bplong(a[19:], uint32(r.Min.Y))
	bplong(a[23:], uint32(r.Max.X))
	bplong(a[27:], uint32(r.Max.Y))
	bplong(a[31:], uint32(r.Min.X))
	bplong(a[35:], uint32(r.Min.Y))
	bplong(a[39:], uint32(r.Max.X))
	bplong(a[43:], uint32(r.Max.Y))
	bplong(a[47:], val)

	if err := d.flushBuffer(len(a)); err != nil {
		return nil, err
	}
	copy(d.buf[d.bufsize:], a[:])
	d.bufsize += len(a)

	return &Image{
		Display: d,
		Screen:  s,
		id:      id,
		Pix:     pix,
		Depth:   depth,
		Repl:    false,
		R:       r,
		Clipr:   r,
	}, nil
}

// NameImage associates a name with an image so other programs can access it.
func (i *Image) Name(name string) error {
	if i == nil || i.Display == nil {
		return nil
	}
	d := i.Display

	d.mu.Lock()
	defer d.mu.Unlock()

	if len(name) >= 256 {
		return fmt.Errorf("name too long")
	}

	// Build the 'N' message
	// Format: 'N' id[4] in[1] nname[1] name[nname]
	var a [1 + 4 + 1 + 1]byte
	a[0] = 'N'
	bplong(a[1:], uint32(i.id))
	a[5] = 1 // in = 1 means we're adding a name
	a[6] = byte(len(name))

	if err := d.flushBuffer(len(a) + len(name)); err != nil {
		return err
	}
	copy(d.buf[d.bufsize:], a[:])
	d.bufsize += len(a)
	copy(d.buf[d.bufsize:], name)
	d.bufsize += len(name)

	return nil
}

// Top moves a window to the top of the screen stack.
func (i *Image) Top() {
	if i == nil || i.Display == nil || i.Screen == nil {
		return
	}
	d := i.Display

	d.mu.Lock()
	defer d.mu.Unlock()

	// Build the 't' message
	// Format: 't' top[1] nid[2] id[4]
	var a [1 + 1 + 2 + 4]byte
	a[0] = 't'
	a[1] = 1 // top
	bpshort(a[2:], 1)
	bplong(a[4:], uint32(i.id))

	if err := d.flushBuffer(len(a)); err != nil {
		return
	}
	copy(d.buf[d.bufsize:], a[:])
	d.bufsize += len(a)
}

// Bottom moves a window to the bottom of the screen stack.
func (i *Image) Bottom() {
	if i == nil || i.Display == nil || i.Screen == nil {
		return
	}
	d := i.Display

	d.mu.Lock()
	defer d.mu.Unlock()

	// Build the 't' message
	var a [1 + 1 + 2 + 4]byte
	a[0] = 't'
	a[1] = 0 // bottom
	bpshort(a[2:], 1)
	bplong(a[4:], uint32(i.id))

	if err := d.flushBuffer(len(a)); err != nil {
		return
	}
	copy(d.buf[d.bufsize:], a[:])
	d.bufsize += len(a)
}

// SetOrigin sets the origin of the image.
func (i *Image) SetOrigin(log, scr Point) error {
	if i == nil || i.Display == nil {
		return nil
	}
	d := i.Display

	d.mu.Lock()
	defer d.mu.Unlock()

	// Build the 'o' message
	// Format: 'o' id[4] rmin.x[4] rmin.y[4] screenrmin.x[4] screenrmin.y[4]
	var a [1 + 4 + 4 + 4 + 4 + 4]byte
	a[0] = 'o'
	bplong(a[1:], uint32(i.id))
	bplong(a[5:], uint32(log.X))
	bplong(a[9:], uint32(log.Y))
	bplong(a[13:], uint32(scr.X))
	bplong(a[17:], uint32(scr.Y))

	if err := d.flushBuffer(len(a)); err != nil {
		return err
	}
	copy(d.buf[d.bufsize:], a[:])
	d.bufsize += len(a)

	delta := log.Sub(i.R.Min)
	i.R = i.R.Add(delta)
	i.Clipr = i.Clipr.Add(delta)
	return nil
}

// bplong puts a 32-bit little-endian value into a byte slice.
func bplong(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

// bpshort puts a 16-bit little-endian value into a byte slice.
func bpshort(b []byte, v uint16) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
}

// glong gets a 32-bit little-endian value from a byte slice.
func glong(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// gshort gets a 16-bit little-endian value from a byte slice.
func gshort(b []byte) uint16 {
	return uint16(b[0]) | uint16(b[1])<<8
}
