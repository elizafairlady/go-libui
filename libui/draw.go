package ui

import (
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"strings"
)

// DrawContext wraps /dev/draw primitives minimally.
type DrawContext struct {
	ctl      *os.File
	data     *os.File
	connID   int
	screenID int // the display image id (id 0)
	winID    int // our window image id (from namedimage)
	Screen   image.Rectangle
	fontH    int
	charW    int
	offsetX  int
	offsetY  int
	white    int // image id for white color
	black    int // image id for black color
	nextID   int // next available image id
}

// NewDrawContext initializes the drawing context.
func NewDrawContext() (*DrawContext, error) {
	ctx := &DrawContext{
		fontH: 13, // default font height
		charW: 7,  // default char width (monospace approximation)
	}

	var err error

	// Open /dev/draw/new to create a NEW draw connection for this process
	// This file stays open as our ctl file
	ctx.ctl, err = os.OpenFile("/dev/draw/new", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open /dev/draw/new: %w", err)
	}

	// Read the connection info: 12 strings, each 11 characters + blank
	// Format: n, imageid, chan, repl, minx, miny, maxx, maxy, clipminx, clipminy, clipmaxx, clipmaxy
	buf := make([]byte, 12*12)
	n, err := ctx.ctl.Read(buf)
	if err != nil {
		ctx.ctl.Close()
		return nil, fmt.Errorf("read /dev/draw/new: %w", err)
	}

	if n < 12*12 {
		ctx.ctl.Close()
		return nil, fmt.Errorf("short read from /dev/draw/new: got %d bytes", n)
	}

	// Parse connection ID (first field)
	ctx.connID = atoi(string(buf[0:11]))
	// Parse display image id (second field) - always 0
	ctx.screenID = atoi(string(buf[12:23]))

	// Parse screen rectangle (fields 5-8: minx, miny, maxx, maxy)
	ctx.Screen.Min.X = atoi(string(buf[4*12 : 5*12]))
	ctx.Screen.Min.Y = atoi(string(buf[5*12 : 6*12]))
	ctx.Screen.Max.X = atoi(string(buf[6*12 : 7*12]))
	ctx.Screen.Max.Y = atoi(string(buf[7*12 : 8*12]))

	// Open /dev/draw/n/data for drawing commands
	dataPath := fmt.Sprintf("/dev/draw/%d/data", ctx.connID)
	ctx.data, err = os.OpenFile(dataPath, os.O_RDWR, 0)
	if err != nil {
		ctx.ctl.Close()
		return nil, fmt.Errorf("open %s: %w", dataPath, err)
	}

	// Start allocating IDs after the screen ID
	ctx.nextID = ctx.screenID + 1

	// Read window name from /dev/winname and get the window image
	winname, err := readWinname()
	if err == nil && winname != "" {
		// Use 'n' message to look up the named window image
		ctx.winID = ctx.nextID
		ctx.nextID++
		if err := ctx.namedImage(ctx.winID, winname); err != nil {
			// If namedimage fails, fall back to screen
			ctx.winID = ctx.screenID
		} else {
			// Read window dimensions from the allocated image
			ctx.readImageInfo(ctx.winID)
		}
	} else {
		// No window name, use display image directly
		ctx.winID = ctx.screenID
	}

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

func readWinname() (string, error) {
	f, err := os.Open("/dev/winname")
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, 256)
	n, err := f.Read(buf)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(buf[:n])), nil
}

// namedImage sends the 'n' message to look up a named image
// n id[4] j[1] name[j]
func (c *DrawContext) namedImage(id int, name string) error {
	nameBytes := []byte(name)
	if len(nameBytes) > 255 {
		nameBytes = nameBytes[:255]
	}

	buf := make([]byte, 1+4+1+len(nameBytes))
	buf[0] = 'n'
	putlong(buf[1:], uint32(id))
	buf[5] = byte(len(nameBytes))
	copy(buf[6:], nameBytes)

	_, err := c.data.Write(buf)
	if err != nil {
		return err
	}

	// Read response - need to read back the image info
	// The response comes as data that we need to interpret
	return nil
}

// readImageInfo reads image info by querying via 'I' message or re-reading ctl
func (c *DrawContext) readImageInfo(id int) {
	// Re-read ctl to get updated info for this image
	buf := make([]byte, 12*12)
	c.ctl.Seek(0, 0)
	n, err := c.ctl.Read(buf)
	if err != nil || n < 12*8 {
		return
	}
	c.Screen.Min.X = atoi(string(buf[4*12 : 5*12]))
	c.Screen.Min.Y = atoi(string(buf[5*12 : 6*12]))
	c.Screen.Max.X = atoi(string(buf[6*12 : 7*12]))
	c.Screen.Max.Y = atoi(string(buf[7*12 : 8*12]))
}

// allocColor allocates a 1x1 replicated image filled with the given color.
// color format: RRGGBBAA
// 'b' message: b id[4] screenid[4] refresh[1] chan[4] repl[1] r[4*4] clipr[4*4] color[4]
func (c *DrawContext) allocColor(id int, color uint32) error {
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
// 'd' message: d dstid[4] srcid[4] maskid[4] dstr[4*4] srcp[2*4] maskp[2*4]
func (c *DrawContext) Clear() {
	c.offsetX = 0
	c.offsetY = 0

	r := c.Screen
	buf := make([]byte, 1+4+4+4+16+8+8)
	buf[0] = 'd'
	putlong(buf[1:], uint32(c.winID))  // dst = our window
	putlong(buf[5:], uint32(c.white))  // src = white
	putlong(buf[9:], uint32(c.white))  // mask = white (opaque)
	putlong(buf[13:], uint32(r.Min.X)) // dstr
	putlong(buf[17:], uint32(r.Min.Y))
	putlong(buf[21:], uint32(r.Max.X))
	putlong(buf[25:], uint32(r.Max.Y))
	putlong(buf[29:], 0) // srcp
	putlong(buf[33:], 0)
	putlong(buf[37:], 0) // maskp
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
		putlong(buf[1:], uint32(c.winID))
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
// 'v' message flushes
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
	// Re-read ctl to get new dimensions
	buf := make([]byte, 12*12)
	_, err := c.ctl.Seek(0, 0)
	if err != nil {
		return err
	}
	n, err := c.ctl.Read(buf)
	if err != nil {
		return err
	}
	if n >= 12*8 {
		c.Screen.Min.X = atoi(string(buf[4*12 : 5*12]))
		c.Screen.Min.Y = atoi(string(buf[5*12 : 6*12]))
		c.Screen.Max.X = atoi(string(buf[6*12 : 7*12]))
		c.Screen.Max.Y = atoi(string(buf[7*12 : 8*12]))
	}
	return nil
}

// Close closes the draw context.
func (c *DrawContext) Close() {
	if c.data != nil {
		c.data.Close()
	}
	if c.ctl != nil {
		c.ctl.Close()
	}
}
