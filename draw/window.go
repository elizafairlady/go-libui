package draw

import (
	"fmt"
	"os"
	"strconv"
)

// AllocScreen allocates a new screen on image with fill color.
// Port of 9front allocscreen().
func AllocScreen(image, fill *Image, public bool) (*Screen, error) {
	d := image.Display
	if d != fill.Display {
		return nil, fmt.Errorf("allocscreen: image and fill on different displays")
	}

	s := &Screen{
		Display: d,
		Image:   image,
		Fill:    fill,
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.imageid++
	id := d.imageid
	s.id = id

	pub := byte(0)
	if public {
		pub = 1
	}

	a, err := d.bufimage(1 + 4 + 4 + 4 + 1)
	if err != nil {
		return nil, err
	}
	a[0] = 'A'
	bplong(a[1:], uint32(id))
	bplong(a[5:], uint32(image.id))
	bplong(a[9:], uint32(fill.id))
	a[13] = pub

	if err := d.doflush(); err != nil {
		return nil, err
	}

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

// GetWindow reads the window image from the display.
// It is typically called after a resize event.
// The ref argument specifies the refresh mode: Refbackup, Refnone, or Refmesg.
func (d *Display) GetWindow(ref int) error {
	return d.gengetwindow("/dev/wctl", ref)
}

func (d *Display) gengetwindow(wctlname string, ref int) error {
	// Read window info from wctl
	wctl, err := os.Open(wctlname)
	if err != nil {
		return err
	}
	defer wctl.Close()

	buf := make([]byte, 256)
	n, err := wctl.Read(buf)
	if err != nil {
		return err
	}

	// Parse window control line
	fields := parseCtlLine(string(buf[:n]))
	if len(fields) < 5 {
		return fmt.Errorf("bad wctl format")
	}

	minx, _ := strconv.Atoi(fields[1])
	miny, _ := strconv.Atoi(fields[2])
	maxx, _ := strconv.Atoi(fields[3])
	maxy, _ := strconv.Atoi(fields[4])

	r := Rect(minx, miny, maxx, maxy)

	// If we already have a window, free it
	if d.Windows != nil && d.Windows != d.Image {
		d.Windows.Free()
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Re-read ctl to refresh display image dimensions
	_, err = d.ctlfd.Seek(0, 0)
	if err != nil {
		return err
	}
	ctlbuf := make([]byte, 12*12)
	m, err := d.ctlfd.Read(ctlbuf)
	if err != nil {
		return err
	}
	if m < 12*12 {
		return fmt.Errorf("short ctl read")
	}
	ctlfields := parseCtlLine(string(ctlbuf[:m]))
	if len(ctlfields) < 12 {
		return fmt.Errorf("malformed ctl")
	}

	// Update display image rectangle
	dminx, _ := strconv.Atoi(ctlfields[3])
	dminy, _ := strconv.Atoi(ctlfields[4])
	dmaxx, _ := strconv.Atoi(ctlfields[5])
	dmaxy, _ := strconv.Atoi(ctlfields[6])
	clipminx, _ := strconv.Atoi(ctlfields[7])
	clipminy, _ := strconv.Atoi(ctlfields[8])
	clipmaxx, _ := strconv.Atoi(ctlfields[9])
	clipmaxy, _ := strconv.Atoi(ctlfields[10])

	d.Image.R = Rect(dminx, dminy, dmaxx, dmaxy)
	d.Image.Clipr = Rect(clipminx, clipminy, clipmaxx, clipmaxy)

	// Set Windows to point to the display image with the window rectangle
	if d.Windows == nil {
		d.Windows = d.Image
	}
	d.Windows.R = r
	d.Windows.Clipr = r

	return nil
}

// Namedimage looks up an image by name.
func (d *Display) Namedimage(name string) (*Image, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	n := len(name)
	if n >= 256 {
		return nil, fmt.Errorf("name too long")
	}

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
		return nil, err
	}

	// Re-read ctl to get image info
	_, err = d.ctlfd.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 12*12+1)
	m, err := d.ctlfd.Read(buf)
	if err != nil {
		return nil, err
	}
	if m < 12*12 {
		return nil, fmt.Errorf("short read from ctl")
	}

	chanstr := trimSpace(string(buf[2*12 : 3*12]))
	pix := strtochan(chanstr)
	if pix == 0 {
		return nil, fmt.Errorf("bad channel from devdraw: %s", chanstr)
	}

	img := &Image{
		Display: d,
		id:      id,
		Pix:     pix,
		Depth:   chantodepth(pix),
	}

	fields := parseCtlLine(string(buf[:m]))
	if len(fields) >= 12 {
		img.Repl = fields[3] == "1"
		img.R.Min.X, _ = strconv.Atoi(fields[4])
		img.R.Min.Y, _ = strconv.Atoi(fields[5])
		img.R.Max.X, _ = strconv.Atoi(fields[6])
		img.R.Max.Y, _ = strconv.Atoi(fields[7])
		img.Clipr.Min.X, _ = strconv.Atoi(fields[8])
		img.Clipr.Min.Y, _ = strconv.Atoi(fields[9])
		img.Clipr.Max.X, _ = strconv.Atoi(fields[10])
		img.Clipr.Max.Y, _ = strconv.Atoi(fields[11])
	}

	return img, nil
}
