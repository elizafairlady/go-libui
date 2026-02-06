package ui

import (
	"9fans.net/go/draw"
)

// DrawContext wraps /dev/draw primitives minimally.
type DrawContext struct {
	Display *draw.Display
	Screen  *draw.Image
	Font    *draw.Font
	bg      *draw.Image
	fg      *draw.Image
	offsetX int
	offsetY int
}

// NewDrawContext initializes the drawing context.
func NewDrawContext() (*DrawContext, error) {
	d, err := draw.Init(nil, "", "ui", "")
	if err != nil {
		return nil, err
	}

	font, err := d.OpenFont("/lib/font/bit/pelm/euro.9.font")
	if err != nil {
		return nil, err
	}

	ctx := &DrawContext{
		Display: d,
		Screen:  d.ScreenImage,
		Font:    font,
	}

	ctx.bg, err = d.AllocImage(draw.Rect(0, 0, 1, 1), d.ScreenImage.Pix, true, draw.White)
	if err != nil {
		return nil, err
	}

	ctx.fg, err = d.AllocImage(draw.Rect(0, 0, 1, 1), d.ScreenImage.Pix, true, draw.Black)
	if err != nil {
		return nil, err
	}

	return ctx, nil
}

// Clear fills the screen with the background color.
func (c *DrawContext) Clear() {
	c.Screen.Draw(c.Screen.R, c.bg, nil, draw.ZP)
	c.offsetX = 0
	c.offsetY = 0
}

// Text draws a string at the given position.
func (c *DrawContext) Text(x, y int, s string) {
	pt := draw.Pt(x+c.offsetX, y+c.offsetY)
	c.Screen.String(pt, c.fg, draw.ZP, c.Font, s)
}

// Translate shifts subsequent drawing operations.
func (c *DrawContext) Translate(dx, dy int) {
	c.offsetX += dx
	c.offsetY += dy
}

// Flush flushes the draw buffer to the screen.
func (c *DrawContext) Flush() {
	c.Display.Flush()
}

// Bounds returns the current screen dimensions.
func (c *DrawContext) Bounds() (width, height int) {
	r := c.Screen.R
	return r.Dx(), r.Dy()
}

// FontHeight returns the height of the default font.
func (c *DrawContext) FontHeight() int {
	return c.Font.Height
}

// Rect draws a rectangle outline.
func (c *DrawContext) Rect(x, y, w, h int) {
	r := draw.Rect(x+c.offsetX, y+c.offsetY, x+c.offsetX+w, y+c.offsetY+h)
	c.Screen.Border(r, 1, c.fg, draw.ZP)
}

// FillRect draws a filled rectangle.
func (c *DrawContext) FillRect(x, y, w, h int, fill bool) {
	r := draw.Rect(x+c.offsetX, y+c.offsetY, x+c.offsetX+w, y+c.offsetY+h)
	if fill {
		c.Screen.Draw(r, c.fg, nil, draw.ZP)
	} else {
		c.Screen.Draw(r, c.bg, nil, draw.ZP)
	}
}

// StringWidth returns the pixel width of a string.
func (c *DrawContext) StringWidth(s string) int {
	return c.Font.StringWidth(s)
}

// Reattach reattaches the display after a resize.
func (c *DrawContext) Reattach() error {
	if err := c.Display.Attach(draw.RefNone); err != nil {
		return err
	}
	c.Screen = c.Display.ScreenImage
	return nil
}
