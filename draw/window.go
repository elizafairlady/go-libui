package draw

import (
	"fmt"
	"os"
	"strings"
)

// PublicScreen acquires an existing public screen by id and channel format.
// Port of 9front publicscreen().
func (d *Display) PublicScreen(id int, pix Pix) (*Screen, error) {
	s := &Screen{}

	d.mu.Lock()
	defer d.mu.Unlock()

	a, err := d.bufimage(1 + 4 + 4)
	if err != nil {
		return nil, err
	}
	a[0] = 'S'
	bplong(a[1:], uint32(id))
	bplong(a[5:], uint32(pix))

	if err := d.doflush(); err != nil {
		return nil, err
	}

	s.Display = d
	s.id = id
	s.Image = nil
	s.Fill = nil
	return s, nil
}

// TopWindow raises a window to the top of the stacking order.
// Port of 9front topwindow().
func (w *Image) TopWindow() {
	if w == nil || w.Screen == nil {
		return
	}
	TopNWindows([]*Image{w})
}

// BottomWindow lowers a window to the bottom of the stacking order.
// Port of 9front bottomwindow().
func (w *Image) BottomWindow() {
	if w == nil || w.Screen == nil {
		return
	}
	BottomNWindows([]*Image{w})
}

// TopNWindows raises n windows to the top.
// Port of 9front topnwindows().
func TopNWindows(w []*Image) {
	topbottom(w, true)
}

// BottomNWindows lowers n windows to the bottom.
// Port of 9front bottomnwindows().
func BottomNWindows(w []*Image) {
	topbottom(w, false)
}

// topbottom implements the window stacking order change.
// Port of 9front topbottom().
func topbottom(w []*Image, top bool) {
	n := len(w)
	if n <= 0 {
		return
	}
	d := w[0].Display
	for i := 1; i < n; i++ {
		if w[i].Display != d {
			return
		}
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	a, err := d.bufimage(1 + 1 + 2 + 4*n)
	if err != nil {
		return
	}
	a[0] = 't'
	if top {
		a[1] = 1
	} else {
		a[1] = 0
	}
	bpshort(a[2:], uint16(n))
	for i := 0; i < n; i++ {
		bplong(a[4+4*i:], uint32(w[i].id))
	}
}

// OriginWindow changes the origin of a window.
// Port of 9front originwindow().
func (w *Image) OriginWindow(log, scr Point) error {
	if w == nil || w.Display == nil {
		return nil
	}
	d := w.Display
	d.mu.Lock()
	defer d.mu.Unlock()

	a, err := d.bufimage(1 + 4 + 2*4 + 2*4)
	if err != nil {
		return err
	}
	a[0] = 'o'
	bplong(a[1:], uint32(w.id))
	bplong(a[5:], uint32(log.X))
	bplong(a[9:], uint32(log.Y))
	bplong(a[13:], uint32(scr.X))
	bplong(a[17:], uint32(scr.Y))

	delta := log.Sub(w.R.Min)
	w.R = w.R.Add(delta)
	w.Clipr = w.Clipr.Add(delta)
	return nil
}

// GetWindow reads the window image from the display via winname + namedimage.
// It is typically called during init and after a resize event.
// The ref argument specifies the refresh mode: Refbackup, Refnone, or Refmesg.
//
// Port of 9front getwindow().
func (d *Display) GetWindow(ref int) error {
	winname := fmt.Sprintf("%s/winname", d.windir)
	return d.gengetwindow(winname, ref)
}

// gengetwindow is the faithful port of 9front gengetwindow().
//
// The flow is:
//  1. Read $windir/winname to get the window image name
//  2. Call namedimage() to acquire that image from devdraw
//  3. If no winname (not running under rio), fall back to display image
//  4. Call allocscreen() on the image
//  5. Call _allocwindow() with Borderwidth inset
//  6. Set d.ScreenImage to the window
func (d *Display) gengetwindow(winnamepath string, ref int) error {
	var image *Image
	noborder := false

	fd, err := os.Open(winnamepath)
	if err != nil {
		// Can't open winname — not running under rio.
		// Fall back to the raw display image.
		noborder = true
		image = d.Image
	} else {
		buf := make([]byte, 64)
		n, err := fd.Read(buf)
		fd.Close()
		if err != nil || n <= 0 {
			noborder = true
			image = d.Image
		} else {
			name := strings.TrimRight(string(buf[:n]), "\n\r \t")
			if name == "" || name == "noborder" {
				noborder = true
				image = d.Image
			} else {
				// Look up the named window image from devdraw.
				// There's a race where winname can change after we read it,
				// so retry if namedimage fails with a different name.
				for try := 0; try < 3; try++ {
					image, err = d.Namedimage(name)
					if err == nil {
						break
					}
					// Re-read winname and retry
					fd2, err2 := os.Open(winnamepath)
					if err2 != nil {
						break
					}
					n, err2 = fd2.Read(buf)
					fd2.Close()
					if err2 != nil || n <= 0 {
						break
					}
					newname := strings.TrimRight(string(buf[:n]), "\n\r \t")
					if newname == name {
						break // same name, real failure
					}
					name = newname
				}
				if image == nil {
					// namedimage failed; fall back
					d.ScreenImage = nil
					return fmt.Errorf("getwindow: namedimage %q: %v", name, err)
				}
			}
		}
	}

	// Free old screen/window if reattaching
	if d.screen != nil {
		if d.ScreenImage != nil {
			d.ScreenImage.freeimage1()
		}
		if d.screen.Image != nil && d.screen.Image != d.Image {
			d.screen.Image.Free()
		}
		d.screen.Free()
		d.screen = nil
	}

	if image == nil {
		d.ScreenImage = nil
		return fmt.Errorf("getwindow: no image")
	}

	// Set d.ScreenImage to the backing image first
	d.ScreenImage = image

	// Allocate a Screen on this image
	d.mu.Lock()
	scr, err := d.allocScreenLocked(image, d.White, false)
	d.mu.Unlock()
	if err != nil {
		d.ScreenImage = nil
		if image != d.Image {
			image.Free()
		}
		return fmt.Errorf("getwindow: allocscreen: %v", err)
	}
	d.screen = scr

	// Inset by Borderwidth unless "noborder"
	r := image.R
	if !noborder {
		r = r.Inset(Borderwidth)
	}

	// Allocate the window (layer) on the screen
	d.mu.Lock()
	win, err := d.allocWindow(nil, scr, r, ref, DWhite)
	d.mu.Unlock()
	if err != nil {
		scr.Free()
		d.screen = nil
		d.ScreenImage = nil
		if image != d.Image {
			image.Free()
		}
		return fmt.Errorf("getwindow: allocwindow: %v", err)
	}

	// The window IS the screen image that programs draw to
	d.ScreenImage = win
	return nil
}

// allocScreenLocked is allocscreen with the lock already held.
func (d *Display) allocScreenLocked(image, fill *Image, public bool) (*Screen, error) {
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

// Namedimage looks up an image by name from devdraw.
// Port of 9front namedimage().
//
// This sends the 'n' command to devdraw to bind a local image id
// to a named (shared) image, then reads back the image properties.
func (d *Display) Namedimage(name string) (*Image, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	n := len(name)
	if n >= 256 {
		return nil, fmt.Errorf("name too long")
	}

	// Flush pending data so we don't get error allocating the image
	if err := d.doflush(); err != nil {
		return nil, err
	}

	d.imageid++
	id := d.imageid

	a, err := d.bufimage(1 + 4 + 1 + n)
	if err != nil {
		return nil, err
	}
	a[0] = 'n'
	bplong(a[1:], uint32(id))
	a[5] = byte(n)
	copy(a[6:], name)

	if err := d.doflush(); err != nil {
		// Free the id on error
		d.freeRemoteId(id)
		return nil, err
	}

	// Read back image properties from ctl
	_, err = d.ctlfd.Seek(0, 0)
	if err != nil {
		d.freeRemoteId(id)
		return nil, err
	}

	buf := make([]byte, 12*12+1)
	m, err := d.ctlfd.Read(buf)
	if err != nil {
		d.freeRemoteId(id)
		return nil, err
	}
	if m < 12*12 {
		d.freeRemoteId(id)
		return nil, fmt.Errorf("short read from ctl")
	}

	fields := parseCtlLine(string(buf[:m]))
	if len(fields) < 12 {
		d.freeRemoteId(id)
		return nil, fmt.Errorf("namedimage: malformed ctl reply")
	}

	chanstr := fields[2]
	pix := strtochan(chanstr)
	if pix == 0 {
		d.freeRemoteId(id)
		return nil, fmt.Errorf("bad channel from devdraw: %s", chanstr)
	}

	img := &Image{
		Display: d,
		id:      id,
		Pix:     pix,
		Depth:   chantodepth(pix),
		Repl:    atoi(fields[3]) != 0,
	}
	img.R.Min.X = atoi(fields[4])
	img.R.Min.Y = atoi(fields[5])
	img.R.Max.X = atoi(fields[6])
	img.R.Max.Y = atoi(fields[7])
	img.Clipr.Min.X = atoi(fields[8])
	img.Clipr.Min.Y = atoi(fields[9])
	img.Clipr.Max.X = atoi(fields[10])
	img.Clipr.Max.Y = atoi(fields[11])

	return img, nil
}

// freeRemoteId sends an 'f' command to free an image id on the server.
func (d *Display) freeRemoteId(id int) {
	a, err := d.bufimage(1 + 4)
	if err != nil {
		return
	}
	a[0] = 'f'
	bplong(a[1:], uint32(id))
	d.doflush()
}
