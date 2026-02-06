package ui

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"strconv"
	"strings"
)

// DrawContext wraps /dev/draw primitives minimally.
type DrawContext struct {
	wsys     *os.File
	data     *os.File
	winid    int
	Screen   image.Rectangle
	fontH    int
	charW    int
	offsetX  int
	offsetY  int
	white    int // image id for white color
	black    int // image id for black color
	nextID   int // next available image id
	screenID int // our window's screen image
}

// NewDrawContext initializes the drawing context.
// For rio windows, we need to create a window through /dev/wsys.
func NewDrawContext() (*DrawContext, error) {
	ctx := &DrawContext{
		fontH:  13, // default font height
		charW:  7,  // default char width (monospace approximation)
		nextID: 1,  // start allocating from id 1
	}

	var err error

	// First, try to find our window ID from environment or /dev/winid
	winid, err := getWinID()
	if err != nil {
		// No existing window, try creating one through wsys
		return nil, fmt.Errorf("no window context: %w (must run in rio window)", err)
	}
	ctx.winid = winid

	// Read window dimensions from /dev/wsys/winid/wctl
	wctlPath := fmt.Sprintf("/dev/wsys/%d/wctl", winid)
	wctl, err := os.Open(wctlPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", wctlPath, err)
	}

	buf := make([]byte, 256)
	n, err := wctl.Read(buf)
	wctl.Close()
	if err != nil {
		return nil, fmt.Errorf("read wctl: %w", err)
	}

	// Parse: "winid minx miny maxx maxy ..."
	fields := strings.Fields(string(buf[:n]))
	if len(fields) >= 5 {
		ctx.Screen.Min.X, _ = strconv.Atoi(fields[1])
		ctx.Screen.Min.Y, _ = strconv.Atoi(fields[2])
		ctx.Screen.Max.X, _ = strconv.Atoi(fields[3])
		ctx.Screen.Max.Y, _ = strconv.Atoi(fields[4])
	}

	// Open /dev/draw/new to get a draw connection
	newf, err := os.OpenFile("/dev/draw/new", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open /dev/draw/new: %w", err)
	}

	// Read connection info
	drawBuf := make([]byte, 12*12)
	n, err = newf.Read(drawBuf)
	if err != nil {
		newf.Close()
		return nil, fmt.Errorf("read /dev/draw/new: %w", err)
	}
	newf.Close()

	if n < 12*12 {
		return nil, fmt.Errorf("short read from /dev/draw/new: got %d bytes", n)
	}

	connID := atoi(string(drawBuf[0:11]))
	ctx.screenID = atoi(string(drawBuf[12:23]))

	// Open data file for drawing commands
	dataPath := fmt.Sprintf("/dev/draw/%d/data", connID)
	ctx.data, err = os.OpenFile(dataPath, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", dataPath, err)
	}

	// Allocate our window image using 'A' (attach window)
	// Actually, we need to create an image for our window area
	ctx.nextID = ctx.screenID + 1

	// Allocate white and black color images
	ctx.white = ctx.nextID
	ctx.nextID++
	if err := ctx.allocColor(ctx.white, 0xFFFFFFFF); err != nil { // white
		ctx.Close()
		return nil, fmt.Errorf("alloc white: %w", err)
	}

	ctx.black = ctx.nextID
	ctx.nextID++
	if err := ctx.allocColor(ctx.black, 0x000000FF); err != nil { // black
		ctx.Close()
		return nil, fmt.Errorf("alloc black: %w", err)
	}

	return ctx, nil
}

func getWinID() (int, error) {
	// Try reading from /dev/winid
	f, err := os.Open("/dev/winid")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		id, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
		if err != nil {
			return 0, err
		}
		return id, nil
	}
	return 0, fmt.Errorf("empty winid")
}

// allocColor allocates a 1x1 replicated image filled with the given color.
// color format: RRGGBBAA
func (c *DrawContext) allocColor(id int, color uint32) error {
	// 'b' message: allocate image
	// b id[4] screenid[4] refresh[1] chan[4] repl[1] r[4*4] clipr[4*4] color[4]
	buf := make([]byte, 1+4+4+1+4+1+16+16+4)
	buf[0] = 'b'
	putlong(buf[1:], uint32(id))
	putlong(buf[5:], 0)           // screenid = 0 (no backing screen)
	buf[9] = 0                    // refresh = Refnone
	putlong(buf[10:], 0x38281808) // RGBA32: r8g8b8a8
	buf[14] = 1                   // repl = 1 (replicate)
	// r = (0,0)-(1,1)
	putlong(buf[15:], 0) // r.min.x
	putlong(buf[19:], 0) // r.min.y
	putlong(buf[23:], 1) // r.max.x
	putlong(buf[27:], 1) // r.max.y
	// clipr = large rectangle to allow replication
	putlong(buf[31:], 0x80000000) // clipr.min.x = -2^31
	putlong(buf[35:], 0x80000000) // clipr.min.y
	putlong(buf[39:], 0x7FFFFFFF) // clipr.max.x = 2^31-1
	putlong(buf[43:], 0x7FFFFFFF) // clipr.max.y
	putlong(buf[47:], color)      // fill color

	_, err := c.data.Write(buf)
	return err
}

func atoi(s string) int {
	n := 0
	neg := false
	i := 0
	for i < len(s) && s[i] == ' ' {
		i++
	}
	if i < len(s) && s[i] == '-' {
		neg = true
		i++
	}
	for ; i < len(s) && s[i] >= '0' && s[i] <= '9'; i++ {
		n = n*10 + int(s[i]-'0')
	}
	if neg {
		n = -n
	}
	return n
}

func putlong(b []byte, v uint32) {
	binary.LittleEndian.PutUint32(b, v)
}

// Clear fills the window with the background color.
func (c *DrawContext) Clear() {
	c.offsetX = 0
	c.offsetY = 0

	// 'd' message: draw
	// d dstid[4] srcid[4] maskid[4] dstr[4*4] srcp[2*4] maskp[2*4]
	r := c.Screen
	buf := make([]byte, 1+4+4+4+16+8+8)
	buf[0] = 'd'
	putlong(buf[1:], uint32(c.screenID)) // dst = our screen
	putlong(buf[5:], uint32(c.white))    // src = white
	putlong(buf[9:], uint32(c.white))    // mask = white (opaque)
	// dstr = screen rectangle
	putlong(buf[13:], uint32(r.Min.X))
	putlong(buf[17:], uint32(r.Min.Y))
	putlong(buf[21:], uint32(r.Max.X))
	putlong(buf[25:], uint32(r.Max.Y))
	// srcp = (0,0)
	putlong(buf[29:], 0)
	putlong(buf[33:], 0)
	// maskp = (0,0)
	putlong(buf[37:], 0)
	putlong(buf[41:], 0)

	c.data.Write(buf)
}

// Text draws a string at the given position.
func (c *DrawContext) Text(x, y int, s string) {
	if len(s) == 0 {
		return
	}

	x += c.offsetX + c.Screen.Min.X
	y += c.offsetY + c.Screen.Min.Y

	// Skip if off screen
	if y > c.Screen.Max.Y || y+c.fontH < c.Screen.Min.Y {
		return
	}

	charX := x
	for _, ch := range s {
		if ch == ' ' {
			charX += c.charW
			continue
		}

		// Draw a small filled rectangle for each char
		buf := make([]byte, 1+4+4+4+16+8+8)
		buf[0] = 'd'
		putlong(buf[1:], uint32(c.screenID))
		putlong(buf[5:], uint32(c.black))
		putlong(buf[9:], uint32(c.black))
		putlong(buf[13:], uint32(charX+1))
		putlong(buf[17:], uint32(y+2))
		putlong(buf[21:], uint32(charX+c.charW-1))
		putlong(buf[25:], uint32(y+c.fontH-2))
		putlong(buf[29:], 0)
		putlong(buf[33:], 0)
		putlong(buf[37:], 0)
		putlong(buf[41:], 0)

		c.data.Write(buf)
		charX += c.charW
	}
}

// Translate shifts subsequent drawing operations.
func (c *DrawContext) Translate(dx, dy int) {
	c.offsetX += dx
	c.offsetY += dy
}

// Flush flushes the draw buffer to the screen.
func (c *DrawContext) Flush() {
	c.data.Write([]byte{'v'})
}

// Bounds returns the current window dimensions.
func (c *DrawContext) Bounds() (width, height int) {
	return c.Screen.Dx(), c.Screen.Dy()
}

// FontHeight returns the height of the default font.
func (c *DrawContext) FontHeight() int {
	return c.fontH
}

// StringWidth returns the pixel width of a string.
func (c *DrawContext) StringWidth(s string) int {
	return len(s) * c.charW
}

// Reattach reattaches the display after a resize.
func (c *DrawContext) Reattach() error {
	wctlPath := fmt.Sprintf("/dev/wsys/%d/wctl", c.winid)
	wctl, err := os.Open(wctlPath)
	if err != nil {
		return err
	}
	defer wctl.Close()

	buf := make([]byte, 256)
	n, err := wctl.Read(buf)
	if err != nil {
		return err
	}

	fields := strings.Fields(string(buf[:n]))
	if len(fields) >= 5 {
		c.Screen.Min.X, _ = strconv.Atoi(fields[1])
		c.Screen.Min.Y, _ = strconv.Atoi(fields[2])
		c.Screen.Max.X, _ = strconv.Atoi(fields[3])
		c.Screen.Max.Y, _ = strconv.Atoi(fields[4])
	}
	return nil
}

// Close closes the draw context.
func (c *DrawContext) Close() {
	if c.data != nil {
		c.data.Close()
	}
	if c.wsys != nil {
		c.wsys.Close()
	}
}
