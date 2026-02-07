package frame

import (
	"github.com/elizafairlady/go-libui/draw"
)

// Init initializes a frame to display text in rectangle r on image b,
// using font ft, with the given color images. cols must have NCol entries:
// [ColBack, ColHigh, ColBord, ColText, ColHText].
// If cols is nil, the colors must be set separately before drawing.
func (f *Frame) Init(r draw.Rectangle, ft *draw.Font, b *draw.Image, cols [NCol]*draw.Image) {
	f.Font = ft
	f.Display = b.Display
	f.Maxtab = 8 * ft.StringWidth("0")
	f.nbox = 0
	f.nalloc = 0
	f.Nchars = 0
	f.Nlines = 0
	f.P0 = 0
	f.P1 = 0
	f.box = nil
	f.Lastlinefull = 0
	f.Cols = cols
	f.SetRects(r, b)
	if f.tick == nil && f.Cols[ColBack] != nil {
		f.InitTick()
	}
}

// InitTick creates the tick (cursor) images.
func (f *Frame) InitTick() {
	b := f.Display.ScreenImage
	if b == nil {
		b = f.Display.Image
	}
	ft := f.Font
	if f.tick != nil {
		f.tick.Free()
	}
	var err error
	f.tick, err = f.Display.AllocImage(
		draw.Rect(0, 0, FRTICKW, ft.Height),
		b.Pix, false, draw.DWhite,
	)
	if err != nil {
		f.tick = nil
		return
	}
	if f.tickback != nil {
		f.tickback.Free()
	}
	f.tickback, err = f.Display.AllocImage(
		f.tick.R,
		b.Pix, false, draw.DWhite,
	)
	if err != nil {
		f.tick.Free()
		f.tick = nil
		f.tickback = nil
		return
	}
	// Fill tick with black (background of cursor)
	f.tick.Draw(f.tick.R, f.Display.Black, draw.ZP)
	// Vertical line in center
	f.tick.Draw(
		draw.Rect(FRTICKW/2, 0, FRTICKW/2+1, ft.Height),
		f.Display.White, draw.ZP,
	)
	// Box on top end
	f.tick.Draw(
		draw.Rect(0, 0, FRTICKW, FRTICKW),
		f.Display.White, draw.ZP,
	)
	// Box on bottom end
	f.tick.Draw(
		draw.Rect(0, ft.Height-FRTICKW, FRTICKW, ft.Height),
		f.Display.White, draw.ZP,
	)
}

// SetRects sets the frame's rectangle and backing image.
// The text rectangle is adjusted to be an exact multiple of
// the font height.
func (f *Frame) SetRects(r draw.Rectangle, b *draw.Image) {
	f.B = b
	f.Entire = r
	f.R = r
	f.R.Max.Y -= (r.Max.Y - r.Min.Y) % f.Font.Height
	f.Maxlines = (r.Max.Y - r.Min.Y) / f.Font.Height
}

// Clear frees the internal box structures. If freeall is true,
// also frees the tick images.
func (f *Frame) Clear(freeall bool) {
	if f.nbox > 0 {
		f.delbox(0, f.nbox-1)
	}
	f.box = nil
	f.nbox = 0
	f.nalloc = 0
	if freeall {
		if f.tick != nil {
			f.tick.Free()
			f.tick = nil
		}
		if f.tickback != nil {
			f.tickback.Free()
			f.tickback = nil
		}
	}
	f.Ticked = 0
}
