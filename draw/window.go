package draw

import (
	"fmt"
	"os"
	"strconv"
)

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
	// Format: id[11] minx[11] miny[11] maxx[11] maxy[11] ...
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

	// Look up the named window image
	d.mu.Lock()
	defer d.mu.Unlock()

	// Use 'n' command to look up "noborder" or window name
	// Format: 'n' id[4] nname[1] name[nname]
	// Actually, for getwindow we just update the existing Window image
	// by re-reading the ctl file

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

	// Flush and read response from ctl
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
	buf[12*12] = 0

	// Parse the info
	chanstr := string(buf[2*12 : 3*12])
	chanstr = chanstr[:len(chanstr)-1] // trim trailing space

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

	// Parse repl and rectangle from ctl output
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

// PublicScreen looks up a public screen by id.
func (d *Display) PublicScreen(screenid int) (*Screen, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Build the 'S' message to look up public screen
	// Actually 'S' allocates a subfont; we need different approach
	// Public screens are accessed via the image name mechanism

	return nil, fmt.Errorf("not implemented")
}
