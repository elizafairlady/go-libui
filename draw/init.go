package draw

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Init opens a connection to the display and returns a Display.
// The fontname argument specifies the default font; if empty, the
// system default is used. The label argument is used as the window title.
// The errfn argument is called on errors; if nil, errors are fatal.
func Init(errfn func(string), fontname, label string) (*Display, error) {
	return geninitdraw("/dev", errfn, fontname, label, "", false)
}

// InitDraw is like Init but with a window directory.
func InitDraw(errfn func(string), fontname, label, windir string) (*Display, error) {
	return geninitdraw("/dev", errfn, fontname, label, windir, false)
}

func geninitdraw(devdir string, errfn func(string), fontname, label, windir string, scalable bool) (*Display, error) {
	if windir == "" {
		windir = devdir
	}
	d := &Display{
		Error:   errfn,
		bufsize: drawBufSize,
		bufp:    0, // buffer starts empty
		devdir:  devdir,
		windir:  windir,
	}
	d.buf = make([]byte, d.bufsize+5) // +5 for flush message

	// Open /dev/draw/new to get a connection
	ctlpath := devdir + "/draw/new"
	var err error
	d.ctlfd, err = os.OpenFile(ctlpath, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("initdraw: open %s: %v", ctlpath, err)
	}

	// Read the initial reply to get directory number and display info
	// Format: "%11d %11d %11s %11d %11d %11d %11d %11d %11d %11d %11d %11d "
	buf := make([]byte, 12*12)
	n, err := d.ctlfd.Read(buf)
	if err != nil {
		d.ctlfd.Close()
		return nil, fmt.Errorf("initdraw: read ctl: %v", err)
	}
	if n < 12*12 {
		d.ctlfd.Close()
		return nil, fmt.Errorf("initdraw: short read from ctl: %d bytes", n)
	}

	// Parse the control reply
	// Format: dirno[12] ?[12] chan[12] repl[12] r.min.x[12] r.min.y[12] r.max.x[12] r.max.y[12] clipr.min.[12]x clipr.min.y[12] clipr.max.x[12] clipr.max.y[12]
	fields := parseCtlLine(string(buf[:n]))
	if len(fields) < 12 {
		d.ctlfd.Close()
		return nil, fmt.Errorf("initdraw: malformed ctl reply")
	}

	d.dirno, _ = strconv.Atoi(fields[0])
	// NOTE: fields[1] is not used (some implementations have DPI or other info)
	// The display image always has id 0
	// fields[2] is the channel format string
	chanstr := fields[2]
	// fields[3] is the repl flag (but for display image we ignore it)
	// fields[4..7] are the display rectangle
	minx, _ := strconv.Atoi(fields[4])
	miny, _ := strconv.Atoi(fields[5])
	maxx, _ := strconv.Atoi(fields[6])
	maxy, _ := strconv.Atoi(fields[7])
	// fields[8..11] are the clip rectangle
	clipminx, _ := strconv.Atoi(fields[8])
	clipminy, _ := strconv.Atoi(fields[9])
	clipmaxx, _ := strconv.Atoi(fields[10])
	clipmaxy, _ := strconv.Atoi(fields[11])

	// Open data file
	datapath := fmt.Sprintf("%s/draw/%d/data", devdir, d.dirno)
	d.datafd, err = os.OpenFile(datapath, os.O_RDWR, 0)
	if err != nil {
		d.ctlfd.Close()
		return nil, fmt.Errorf("initdraw: open %s: %v", datapath, err)
	}

	// Open refresh file (optional, for resize events)
	refpath := fmt.Sprintf("%s/draw/%d/refresh", devdir, d.dirno)
	d.reffd, _ = os.Open(refpath) // ignore error, not all systems have it

	// Create the display image
	// The display image always has id 0
	pix := strtochan(chanstr)
	d.Image = &Image{
		Display: d,
		id:      0,
		Pix:     pix,
		Depth:   chantodepth(pix),
		R:       Rect(minx, miny, maxx, maxy),
		Clipr:   Rect(clipminx, clipminy, clipmaxx, clipmaxy),
		Repl:    false,
	}

	// Allocate standard colors
	d.White, err = d.AllocImage(Rect(0, 0, 1, 1), GREY1, true, DWhite)
	if err != nil {
		d.Close()
		return nil, fmt.Errorf("initdraw: alloc white: %v", err)
	}
	d.Black, err = d.AllocImage(Rect(0, 0, 1, 1), GREY1, true, DBlack)
	if err != nil {
		d.Close()
		return nil, fmt.Errorf("initdraw: alloc black: %v", err)
	}
	d.Opaque = d.White
	d.Transparent = d.Black

	// Set window label if provided
	if label != "" {
		d.SetLabel(label)
	}

	// Open default font
	if fontname == "" {
		fontname = getenv("font")
		if fontname == "" {
			fontname = "/lib/font/bit/vga/unicode.font"
		}
	}
	d.DefaultFont, err = d.OpenFont(fontname)
	if err != nil {
		// Try to get the built-in default font
		d.DefaultSubfont = d.getdefont()
		if d.DefaultSubfont != nil {
			d.DefaultFont = &Font{
				Display: d,
				Name:    "*default*",
				Height:  d.DefaultSubfont.Height,
				Ascent:  d.DefaultSubfont.Ascent,
			}
		}
	}

	return d, nil
}

// Close closes the display connection and frees all resources.
func (d *Display) Close() error {
	if d.reffd != nil {
		d.reffd.Close()
	}
	if d.datafd != nil {
		d.datafd.Close()
	}
	if d.ctlfd != nil {
		d.ctlfd.Close()
	}
	return nil
}

// SetLabel sets the window title.
func (d *Display) SetLabel(label string) error {
	// Write label file in windir
	path := d.windir + "/label"
	fd, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		// Try creating it
		fd, err = os.Create(path)
		if err != nil {
			return err
		}
	}
	defer fd.Close()
	_, err = fd.WriteString(label)
	return err
}

// Flush flushes any buffered draw commands to the display.
func (d *Display) Flush() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.flush(true)
}

func (d *Display) doflush() error {
	if d.bufp <= 0 {
		return nil
	}
	n, err := d.datafd.Write(d.buf[:d.bufp])
	if err != nil || n != d.bufp {
		d.bufp = 0 // reset anyway to try to recover
		return err
	}
	d.bufp = 0
	return nil
}

func (d *Display) flush(visible bool) error {
	if visible {
		// Add 'v' command for visible flush
		d.buf[d.bufp] = 'v'
		d.bufp++
	}
	return d.doflush()
}

// bufimage reserves n bytes in the draw buffer.
// Returns a slice to write the command into.
func (d *Display) bufimage(n int) ([]byte, error) {
	if n < 0 || n > d.bufsize {
		return nil, fmt.Errorf("bad count in bufimage: %d", n)
	}
	if d.bufp+n > d.bufsize {
		if err := d.doflush(); err != nil {
			return nil, err
		}
	}
	p := d.buf[d.bufp : d.bufp+n]
	d.bufp += n
	return p, nil
}

// Attach re-attaches to the display after a resize.
func (d *Display) Attach(ref int) error {
	// Re-read ctl to get new display dimensions
	_, err := d.ctlfd.Seek(0, 0)
	if err != nil {
		return err
	}
	buf := make([]byte, 12*12)
	n, err := d.ctlfd.Read(buf)
	if err != nil {
		return err
	}
	if n < 12*12 {
		return errors.New("short read from ctl")
	}
	fields := parseCtlLine(string(buf[:n]))
	if len(fields) < 12 {
		return errors.New("malformed ctl reply")
	}

	minx, _ := strconv.Atoi(fields[3])
	miny, _ := strconv.Atoi(fields[4])
	maxx, _ := strconv.Atoi(fields[5])
	maxy, _ := strconv.Atoi(fields[6])
	clipminx, _ := strconv.Atoi(fields[7])
	clipminy, _ := strconv.Atoi(fields[8])
	clipmaxx, _ := strconv.Atoi(fields[9])
	clipmaxy, _ := strconv.Atoi(fields[10])

	d.Image.R = Rect(minx, miny, maxx, maxy)
	d.Image.Clipr = Rect(clipminx, clipminy, clipmaxx, clipmaxy)
	return nil
}

// ScaleSize scales a size according to display DPI.
func (d *Display) ScaleSize(n int) int {
	if d.DPI <= 0 {
		return n
	}
	return (n*d.DPI + 100/2) / 100
}

// parseCtlLine parses a space-separated control line.
func parseCtlLine(s string) []string {
	var fields []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		fields = append(fields, scanner.Text())
	}
	return fields
}

func getenv(name string) string {
	// On Plan 9, environment variables are in /env
	data, err := os.ReadFile("/env/" + name)
	if err != nil {
		// Fall back to OS environment
		return os.Getenv(name)
	}
	// Remove trailing null if present
	if len(data) > 0 && data[len(data)-1] == 0 {
		data = data[:len(data)-1]
	}
	return string(data)
}
