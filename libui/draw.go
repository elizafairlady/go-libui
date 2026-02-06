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
	refresh *os.File
	id      int
	Screen  image.Rectangle
	fontH   int
	charW   int
	offsetX int
	offsetY int
}

// NewDrawContext initializes the drawing context.
func NewDrawContext() (*DrawContext, error) {
	ctx := &DrawContext{
		fontH: 13, // default font height
		charW: 7,  // default char width (monospace approximation)
	}

	var err error

	// In 9front, /dev/draw/new when opened and read gives connection info
	// and that fd becomes the ctl file
	ctx.ctl, err = os.OpenFile("/dev/draw/new", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open /dev/draw/new: %w", err)
	}

	// Read the connection info
	buf := make([]byte, 12*12)
	n, err := ctx.ctl.Read(buf)
	if err != nil {
		ctx.ctl.Close()
		return nil, fmt.Errorf("read /dev/draw/new: %w", err)
	}

	// Parse connection ID (first field)
	ctx.id = atoi(string(buf[:11]))

	// Parse screen rectangle from the initial data
	// Format: id chan minx miny maxx maxy ...
	if n >= 12*6 {
		ctx.Screen.Min.X = atoi(string(buf[2*12 : 3*12]))
		ctx.Screen.Min.Y = atoi(string(buf[3*12 : 4*12]))
		ctx.Screen.Max.X = atoi(string(buf[4*12 : 5*12]))
		ctx.Screen.Max.Y = atoi(string(buf[5*12 : 6*12]))
	}

	// Open data file for drawing commands
	dataPath := fmt.Sprintf("/dev/draw/%d/data", ctx.id)
	ctx.data, err = os.OpenFile(dataPath, os.O_RDWR, 0)
	if err != nil {
		// Try alternate path structure
		ctx.data, err = os.OpenFile("/dev/draw/data", os.O_RDWR, 0)
		if err != nil {
			ctx.ctl.Close()
			return nil, fmt.Errorf("open data: %w", err)
		}
	}

	// Open refresh file to detect resize events
	refreshPath := fmt.Sprintf("/dev/draw/%d/refresh", ctx.id)
	ctx.refresh, _ = os.Open(refreshPath) // optional, ignore error

	return ctx, nil
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

// Clear fills the screen with the background color.
func (c *DrawContext) Clear() {
	c.offsetX = 0
	c.offsetY = 0

	// Use draw protocol 'd' command to draw a filled rectangle
	// 'd' dstid srcid maskid dstr srcpt maskpt
	// But simpler: just use 'r' for allocating a solid color image first

	// For now, let's write a simpler rectangle fill
	// Protocol: 'r' id R color
	r := c.Screen
	buf := make([]byte, 1+4+4*4+4)
	buf[0] = 'r'
	putint(buf[1:5], 0) // screen image id = 0
	putint(buf[5:9], r.Min.X)
	putint(buf[9:13], r.Min.Y)
	putint(buf[13:17], r.Max.X)
	putint(buf[17:21], r.Max.Y)
	putint(buf[21:25], 0xFFFFFFFF) // white

	c.data.Write(buf)
}

func putint(b []byte, v int) {
	binary.LittleEndian.PutUint32(b, uint32(v))
}

// Text draws a string at the given position.
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

	// Draw each character as a small filled rect for now
	// This is a fallback until we get proper font rendering
	// TODO: implement proper 'x' draw string command

	// For debugging, let's draw character positions as dots
	charX := x
	for range s {
		// Draw a small rectangle for each char position
		buf := make([]byte, 1+4+4*4+4)
		buf[0] = 'r'
		putint(buf[1:5], 0)
		putint(buf[5:9], charX)
		putint(buf[9:13], y)
		putint(buf[13:17], charX+c.charW-1)
		putint(buf[17:21], y+c.fontH-1)
		putint(buf[21:25], 0x000000FF) // black

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
	if n >= 12*6 {
		c.Screen.Min.X = atoi(string(buf[2*12 : 3*12]))
		c.Screen.Min.Y = atoi(string(buf[3*12 : 4*12]))
		c.Screen.Max.X = atoi(string(buf[4*12 : 5*12]))
		c.Screen.Max.Y = atoi(string(buf[5*12 : 6*12]))
	}
	return nil
}

// Close closes the draw context.
func (c *DrawContext) Close() {
	if c.refresh != nil {
		c.refresh.Close()
	}
	if c.data != nil {
		c.data.Close()
	}
	if c.ctl != nil {
		c.ctl.Close()
	}
}
