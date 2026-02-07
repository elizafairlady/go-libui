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

	if len(name) >= 256 {
		return nil, fmt.Errorf("name too long")
	}

	// Build the 'n' message
	// Format: 'n' dstid[4] nname[1] name[nname]
	// Returns image info if found

	id := d.imageid
	d.imageid++

	var a [1 + 4 + 1]byte
	a[0] = 'n'
	bplong(a[1:], uint32(id))
	a[5] = byte(len(name))

	if err := d.flushBuffer(len(a) + len(name)); err != nil {
		return nil, err
	}
	copy(d.buf[d.bufsize:], a[:])
	d.bufsize += len(a)
	copy(d.buf[d.bufsize:], name)
	d.bufsize += len(name)

	// Flush and read response
	if err := d.flush(false); err != nil {
		return nil, err
	}

	// Read response from data - contains image info
	// Format: chan[4] repl[1] r[4*4] clipr[4*4]
	buf := make([]byte, 4+1+16+16)
	n, err := d.datafd.Read(buf)
	if err != nil {
		return nil, err
	}
	if n < len(buf) {
		return nil, fmt.Errorf("short read from data")
	}

	pix := Pix(glong(buf[0:]))
	repl := buf[4] != 0
	minx := int(int32(glong(buf[5:])))
	miny := int(int32(glong(buf[9:])))
	maxx := int(int32(glong(buf[13:])))
	maxy := int(int32(glong(buf[17:])))
	cminx := int(int32(glong(buf[21:])))
	cminy := int(int32(glong(buf[25:])))
	cmaxx := int(int32(glong(buf[29:])))
	cmaxy := int(int32(glong(buf[33:])))

	return &Image{
		Display: d,
		id:      id,
		Pix:     pix,
		Depth:   chantodepth(pix),
		Repl:    repl,
		R:       Rect(minx, miny, maxx, maxy),
		Clipr:   Rect(cminx, cminy, cmaxx, cmaxy),
	}, nil
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
