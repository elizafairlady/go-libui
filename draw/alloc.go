package draw

import (
	"fmt"
)

// AllocImage allocates a new image with the given rectangle, pixel format,
// replication flag, and fill color.
func (d *Display) AllocImage(r Rectangle, pix Pix, repl bool, val uint32) (*Image, error) {
	return d.allocImage(nil, r, pix, repl, val, 0, 0)
}

// allocImage is the internal implementation of AllocImage.
func (d *Display) allocImage(ai *Image, r Rectangle, pix Pix, repl bool, val uint32, screenid int, refresh int) (*Image, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if Badrect(r) {
		return nil, fmt.Errorf("allocimage: bad rectangle")
	}
	if pix == 0 {
		return nil, fmt.Errorf("allocimage: bad channel descriptor")
	}

	depth := chantodepth(pix)
	if depth == 0 {
		return nil, fmt.Errorf("allocimage: bad channel descriptor")
	}

	// Allocate image id
	d.imageid++
	id := d.imageid

	// Build the 'b' (allocate) message
	// Format: 'b' id[4] screenid[4] refresh[1] chan[4] repl[1] r[4*4] clipr[4*4] color[4]
	a, err := d.bufimage(1 + 4 + 4 + 1 + 4 + 1 + 4*4 + 4*4 + 4)
	if err != nil {
		return nil, err
	}

	a[0] = 'b'
	bplong(a[1:], uint32(id))
	bplong(a[5:], uint32(screenid))
	a[9] = byte(refresh)
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

	// Clip rectangle - same as r for non-replicating, huge for replicating
	var clipr Rectangle
	if repl {
		// huge but not infinite, so various offsets leave it huge, not overflow
		clipr = Rect(-0x3FFFFFFF, -0x3FFFFFFF, 0x3FFFFFFF, 0x3FFFFFFF)
	} else {
		clipr = r
	}
	bplong(a[31:], uint32(clipr.Min.X))
	bplong(a[35:], uint32(clipr.Min.Y))
	bplong(a[39:], uint32(clipr.Max.X))
	bplong(a[43:], uint32(clipr.Max.Y))
	bplong(a[47:], val)

	// Create the Image struct
	var img *Image
	if ai != nil {
		img = ai
	} else {
		img = &Image{}
	}
	img.Display = d
	img.id = id
	img.Pix = pix
	img.Depth = depth
	img.Repl = repl
	img.R = r
	img.Clipr = clipr
	img.Screen = nil
	img.next = nil

	return img, nil
}

// freeimage1 is the internal free that doesn't free the Go struct.
func (i *Image) freeimage1() error {
	if i == nil || i.Display == nil {
		return nil
	}
	d := i.Display

	a, err := d.bufimage(1 + 4)
	if err != nil {
		return err
	}
	a[0] = 'f'
	bplong(a[1:], uint32(i.id))

	// Remove from screen windows list if needed
	if i.Screen != nil {
		w := &d.Windows
		for *w != nil {
			if *w == i {
				*w = i.next
				break
			}
			w = &(*w).next
		}
	}
	return nil
}

// Free releases the resources associated with an image.
func (i *Image) Free() error {
	if i == nil || i.Display == nil {
		return nil
	}
	d := i.Display

	d.mu.Lock()
	defer d.mu.Unlock()

	err := i.freeimage1()
	i.Display = nil
	return err
}

// AllocScreen allocates a new screen backed by image fill.
func (d *Display) AllocScreen(image, fill *Image, public bool) (*Screen, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.imageid++
	id := d.imageid

	a, err := d.bufimage(1 + 4 + 4 + 4 + 1)
	if err != nil {
		return nil, err
	}
	a[0] = 'A'
	bplong(a[1:], uint32(id))
	bplong(a[5:], uint32(image.id))
	bplong(a[9:], uint32(fill.id))
	if public {
		a[13] = 1
	} else {
		a[13] = 0
	}

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

	a, err := d.bufimage(1 + 4)
	if err != nil {
		return err
	}
	a[0] = 'F'
	bplong(a[1:], uint32(s.id))

	s.Display = nil
	return nil
}

// allocWindow allocates a window on a screen (internal, holds lock).
func (d *Display) allocWindow(win *Image, s *Screen, r Rectangle, ref int, val uint32) (*Image, error) {
	d.imageid++
	id := d.imageid

	a, err := d.bufimage(1 + 4 + 4 + 1 + 4 + 1 + 4*4 + 4*4 + 4)
	if err != nil {
		return nil, err
	}

	a[0] = 'b'
	bplong(a[1:], uint32(id))
	bplong(a[5:], uint32(s.id))
	a[9] = byte(ref)
	bplong(a[10:], uint32(s.Image.Pix))
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

	var i *Image
	if win != nil {
		i = win
	} else {
		i = &Image{}
	}
	i.Display = d
	i.Screen = s
	i.id = id
	i.Pix = s.Image.Pix
	i.Depth = s.Image.Depth
	i.Repl = false
	i.R = r
	i.Clipr = r

	// Add to windows list
	i.next = d.Windows
	d.Windows = i

	return i, nil
}

// AllocWindow allocates a window on a screen.
func (s *Screen) AllocWindow(r Rectangle, ref int, val uint32) (*Image, error) {
	d := s.Display
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.allocWindow(nil, s, r, ref, val)
}

// NameImage associates a name with an image so other programs can access it.
func (i *Image) Name(name string, in bool) error {
	if i == nil || i.Display == nil {
		return nil
	}
	d := i.Display

	d.mu.Lock()
	defer d.mu.Unlock()

	n := len(name)
	if n >= 256 {
		return fmt.Errorf("name too long")
	}

	a, err := d.bufimage(1 + 4 + 1 + 1 + n)
	if err != nil {
		return err
	}
	a[0] = 'N'
	bplong(a[1:], uint32(i.id))
	if in {
		a[5] = 1
	} else {
		a[5] = 0
	}
	a[6] = byte(n)
	copy(a[7:], name)

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

	a, err := d.bufimage(1 + 1 + 2 + 4)
	if err != nil {
		return
	}
	a[0] = 't'
	a[1] = 1 // top
	bpshort(a[2:], 1)
	bplong(a[4:], uint32(i.id))
}

// Bottom moves a window to the bottom of the screen stack.
func (i *Image) Bottom() {
	if i == nil || i.Display == nil || i.Screen == nil {
		return
	}
	d := i.Display

	d.mu.Lock()
	defer d.mu.Unlock()

	a, err := d.bufimage(1 + 1 + 2 + 4)
	if err != nil {
		return
	}
	a[0] = 't'
	a[1] = 0 // bottom
	bpshort(a[2:], 1)
	bplong(a[4:], uint32(i.id))
}

// SetOrigin sets the origin of the image.
func (i *Image) SetOrigin(log, scr Point) error {
	if i == nil || i.Display == nil {
		return nil
	}
	d := i.Display

	d.mu.Lock()
	defer d.mu.Unlock()

	a, err := d.bufimage(1 + 4 + 4 + 4 + 4 + 4)
	if err != nil {
		return err
	}
	a[0] = 'o'
	bplong(a[1:], uint32(i.id))
	bplong(a[5:], uint32(log.X))
	bplong(a[9:], uint32(log.Y))
	bplong(a[13:], uint32(scr.X))
	bplong(a[17:], uint32(scr.Y))

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

// AllocImageMix allocates a 1x1 replicated image blending two colors.
// Used for creating halftone patterns.
func (d *Display) AllocImageMix(color1, color3 uint32) (*Image, error) {
	// For high bit depth, use alpha blending with ~25% mask
	t, err := d.AllocImage(Rect(0, 0, 1, 1), d.ScreenImage.Pix, true, color1)
	if err != nil {
		return nil, err
	}

	b, err := d.AllocImage(Rect(0, 0, 1, 1), d.ScreenImage.Pix, true, color3)
	if err != nil {
		t.Free()
		return nil, err
	}

	// Create mask for ~25% blend (0x3F = 63 out of 255 ≈ 25%)
	qmask, err := d.AllocImage(Rect(0, 0, 1, 1), GREY8, true, 0x3F3F3FFF)
	if err != nil {
		t.Free()
		b.Free()
		return nil, err
	}
	defer qmask.Free()

	// Blend color1 onto color3 using the mask
	b.GenDraw(b.R, t, ZP, qmask, ZP)
	t.Free()
	return b, nil
}
