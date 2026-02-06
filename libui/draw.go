package ui

import (
	"encoding/binary"
	"fmt"
	"image"
	"os"
)

// DrawContext wraps /dev/draw primitives minimally.
type DrawContext struct {
	ctl     *os.File
	data    *os.File
	id      int
	imgID   int // the display image id (from ctl)
	Screen  image.Rectangle
	fontH   int
	charW   int
	offsetX int
	offsetY int
	white   int // image id for white color
	black   int // image id for black color
	nextID  int // next available image id
}

// NewDrawContext initializes the drawing context.
func NewDrawContext() (*DrawContext, error) {
	ctx := &DrawContext{
		fontH:  13, // default font height
		charW:  7,  // default char width (monospace approximation)
		nextID: 1,  // start allocating from id 1
	}

	var err error

	// In 9front, /dev/draw/new when opened and read gives connection info
	// and that fd becomes the ctl file
	ctx.ctl, err = os.OpenFile("/dev/draw/new", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open /dev/draw/new: %w", err)
	}

	// Read the connection info
	// Format: 12 strings, each 11 characters wide followed by a blank
	// n, image_id, chan, repl, minx, miny, maxx, maxy, clipminx, clipminy, clipmaxx, clipmaxy
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
	ctx.id = atoi(string(buf[0:11]))
	// Parse image id (second field) - this is the display image
	ctx.imgID = atoi(string(buf[12:23]))

	// Parse screen rectangle
	ctx.Screen.Min.X = atoi(string(buf[4*12 : 5*12]))
	ctx.Screen.Min.Y = atoi(string(buf[5*12 : 6*12]))
	ctx.Screen.Max.X = atoi(string(buf[6*12 : 7*12]))
	ctx.Screen.Max.Y = atoi(string(buf[7*12 : 8*12]))

	// Open data file for drawing commands
	dataPath := fmt.Sprintf("/dev/draw/%d/data", ctx.id)
	ctx.data, err = os.OpenFile(dataPath, os.O_RDWR, 0)
	if err != nil {
		ctx.ctl.Close()
		return nil, fmt.Errorf("open %s: %w", dataPath, err)
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

// allocColor allocates a 1x1 replicated image filled with the given color.
// color format: RRGGBBAA
func (c *DrawContext) allocColor(id int, color uint32) error {
	// 'b' message: allocate image
	// b id[4] screenid[4] refresh[1] chan[4] repl[1] r[4*4] clipr[4*4] color[4]
	// For a simple solid color, we use a degenerate rectangle and repl=1
	buf := make([]byte, 1+4+4+1+4+1+16+16+4)
	buf[0] = 'b'
	putlong(buf[1:], uint32(id))
	putlong(buf[5:], 0) // screenid = 0 (no screen, just image)
	buf[9] = 0          // refresh = Refnone
	// chan = RGBA32 = 0x0 0x08 0x08 0x08 -> packed as single uint32
	// Actually for a solid color image, use RGBA32: r8g8b8a8 = 0x20202028
	// But simpler: use m8 (8-bit greyscale mapped) which is 0x18
	// Actually, let's use RGBA32: CRed<<4|8, CGreen<<4|8, CBlue<<4|8, CAlpha<<4|8
	// CRed=0, CGreen=1, CBlue=2, CAlpha=3, CGrey=4, CMap=5
	// r8g8b8a8 = (0<<4|8) | (1<<4|8)<<8 | (2<<4|8)<<16 | (3<<4|8)<<24
	// = 0x08 | 0x1800 | 0x280000 | 0x38000000 = 0x38281808
	putlong(buf[10:], 0x38281808) // RGBA32
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

func putshort(b []byte, v uint16) {
	binary.LittleEndian.PutUint16(b, v)
}

// Clear fills the screen with the background color.
func (c *DrawContext) Clear() {
	c.offsetX = 0
	c.offsetY = 0

	// 'd' message: draw
	// d dstid[4] srcid[4] maskid[4] dstr[4*4] srcp[2*4] maskp[2*4]
	r := c.Screen
	buf := make([]byte, 1+4+4+4+16+8+8)
	buf[0] = 'd'
	putlong(buf[1:], uint32(c.imgID)) // dst = screen
	putlong(buf[5:], uint32(c.white)) // src = white
	putlong(buf[9:], uint32(c.white)) // mask = white (opaque)
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
// Since fonts are complex, we draw simple rectangles as placeholders.
func (c *DrawContext) Text(x, y int, s string) {
	if len(s) == 0 {
		return
	}

	x += c.offsetX
	y += c.offsetY

	// Skip if off screen
	if y > c.Screen.Max.Y || y+c.fontH < c.Screen.Min.Y {
		return
	}

	// Draw each character as a small rectangle
	// This is a placeholder until proper font support
	charX := x
	for _, ch := range s {
		if ch == ' ' {
			charX += c.charW
			continue
		}

		// Draw a small filled rectangle for each char
		buf := make([]byte, 1+4+4+4+16+8+8)
		buf[0] = 'd'
		putlong(buf[1:], uint32(c.imgID))  // dst = screen
		putlong(buf[5:], uint32(c.black))  // src = black
		putlong(buf[9:], uint32(c.black))  // mask = black (opaque for now)
		putlong(buf[13:], uint32(charX+1)) // dstr.min.x
		putlong(buf[17:], uint32(y+2))     // dstr.min.y
		putlong(buf[21:], uint32(charX+c.charW-1))
		putlong(buf[25:], uint32(y+c.fontH-2))
		putlong(buf[29:], 0) // srcp
		putlong(buf[33:], 0)
		putlong(buf[37:], 0) // maskp
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
	// 'v' command flushes
	c.data.Write([]byte{'v'})
}

// Bounds returns the current screen dimensions.
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
	// Re-read screen dimensions from ctl
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
