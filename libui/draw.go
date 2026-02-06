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
	winctl  *os.File
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

	// Open /dev/draw/new to get a connection
	newf, err := os.Open("/dev/draw/new")
	if err != nil {
		return nil, fmt.Errorf("open /dev/draw/new: %w", err)
	}

	// Read the connection info
	buf := make([]byte, 12*12)
	n, err := newf.Read(buf)
	if err != nil {
		newf.Close()
		return nil, fmt.Errorf("read /dev/draw/new: %w", err)
	}
	newf.Close()

	// Parse connection ID (first field)
	ctx.id = atoi(string(buf[:11]))

	// Open control and data files
	prefix := fmt.Sprintf("/dev/draw/%d", ctx.id)

	ctx.ctl, err = os.OpenFile(prefix+"/ctl", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open ctl: %w", err)
	}

	ctx.data, err = os.OpenFile(prefix+"/data", os.O_RDWR, 0)
	if err != nil {
		ctx.ctl.Close()
		return nil, fmt.Errorf("open data: %w", err)
	}

	// Parse screen rectangle from the initial data
	// Format: id chan minx miny maxx maxy ...
	if n >= 12*5 {
		ctx.Screen.Min.X = atoi(string(buf[2*12 : 3*12]))
		ctx.Screen.Min.Y = atoi(string(buf[3*12 : 4*12]))
		ctx.Screen.Max.X = atoi(string(buf[4*12 : 5*12]))
		ctx.Screen.Max.Y = atoi(string(buf[5*12 : 6*12]))
	}

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

	// Draw command: 'r' screenid rect
	// Fill screen with white
	r := c.Screen
	buf := make([]byte, 1+4+4*4+4)
	buf[0] = 'r'                               // draw rect
	binary.LittleEndian.PutUint32(buf[1:5], 0) // screen id
	binary.LittleEndian.PutUint32(buf[5:9], uint32(r.Min.X))
	binary.LittleEndian.PutUint32(buf[9:13], uint32(r.Min.Y))
	binary.LittleEndian.PutUint32(buf[13:17], uint32(r.Max.X))
	binary.LittleEndian.PutUint32(buf[17:21], uint32(r.Max.Y))
	binary.LittleEndian.PutUint32(buf[21:25], 0xFFFFFFFF) // white

	c.data.Write(buf)
}

// Text draws a string at the given position.
func (c *DrawContext) Text(x, y int, s string) {
	// For Plan 9, we use the 's' draw command
	// But the draw protocol is complex - for now use simpler approach
	// Write to /dev/draw using the string command

	x += c.offsetX
	y += c.offsetY

	// Skip if off screen
	if y > c.Screen.Max.Y || y+c.fontH < c.Screen.Min.Y {
		return
	}

	// Simple string draw command
	buf := make([]byte, 1+4+4+4+4+4+2+2+len(s))
	buf[0] = 's'                                        // string command
	binary.LittleEndian.PutUint32(buf[1:5], 0)          // dst id
	binary.LittleEndian.PutUint32(buf[5:9], uint32(x))  // point x
	binary.LittleEndian.PutUint32(buf[9:13], uint32(y)) // point y
	binary.LittleEndian.PutUint32(buf[13:17], 1)        // src id (black)
	binary.LittleEndian.PutUint32(buf[17:21], 0)        // font id
	binary.LittleEndian.PutUint16(buf[21:23], uint16(len(s)))
	binary.LittleEndian.PutUint16(buf[23:25], 0) // index
	copy(buf[25:], s)

	c.data.Write(buf)
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
	if n >= 12*5 {
		c.Screen.Min.X = atoi(string(buf[2*12 : 3*12]))
		c.Screen.Min.Y = atoi(string(buf[3*12 : 4*12]))
		c.Screen.Max.X = atoi(string(buf[4*12 : 5*12]))
		c.Screen.Max.Y = atoi(string(buf[5*12 : 6*12]))
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
