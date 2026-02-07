// Package draw implements the Plan 9 draw library.
// See draw(2) and graphics(2) from the Plan 9 manual.
package draw

import (
	"os"
	"sync"
)

// Point is a location in the integer grid.
type Point struct {
	X, Y int
}

// ZP is the zero point.
var ZP Point

// Pt returns the point (x, y).
func Pt(x, y int) Point {
	return Point{x, y}
}

// Add returns p translated by q.
func (p Point) Add(q Point) Point {
	return Point{p.X + q.X, p.Y + q.Y}
}

// Sub returns p translated by -q.
func (p Point) Sub(q Point) Point {
	return Point{p.X - q.X, p.Y - q.Y}
}

// Mul returns p scaled by k.
func (p Point) Mul(k int) Point {
	return Point{p.X * k, p.Y * k}
}

// Div returns p divided by k.
func (p Point) Div(k int) Point {
	return Point{p.X / k, p.Y / k}
}

// Eq reports whether p and q are equal.
func (p Point) Eq(q Point) bool {
	return p.X == q.X && p.Y == q.Y
}

// In reports whether p is in r.
func (p Point) In(r Rectangle) bool {
	return r.Min.X <= p.X && p.X < r.Max.X &&
		r.Min.Y <= p.Y && p.Y < r.Max.Y
}

// Rectangle is a rectangle in the integer grid.
type Rectangle struct {
	Min, Max Point
}

// ZR is the zero rectangle.
var ZR Rectangle

// Rect returns the rectangle with corners (x0, y0) and (x1, y1).
// The corners don't need to be in any particular order.
func Rect(x0, y0, x1, y1 int) Rectangle {
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	return Rectangle{Point{x0, y0}, Point{x1, y1}}
}

// Rpt returns the rectangle with corners min and max.
func Rpt(min, max Point) Rectangle {
	return Rectangle{min, max}
}

// Canon returns a canonical form of r: Min is to the left of Max.
func (r Rectangle) Canon() Rectangle {
	return Rect(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y)
}

// Dx returns the width of r.
func (r Rectangle) Dx() int {
	return r.Max.X - r.Min.X
}

// Dy returns the height of r.
func (r Rectangle) Dy() int {
	return r.Max.Y - r.Min.Y
}

// Inset returns r shrunk by n pixels.
func (r Rectangle) Inset(n int) Rectangle {
	if r.Dx() < 2*n {
		r.Min.X = (r.Min.X + r.Max.X) / 2
		r.Max.X = r.Min.X
	} else {
		r.Min.X += n
		r.Max.X -= n
	}
	if r.Dy() < 2*n {
		r.Min.Y = (r.Min.Y + r.Max.Y) / 2
		r.Max.Y = r.Min.Y
	} else {
		r.Min.Y += n
		r.Max.Y -= n
	}
	return r
}

// Add returns r translated by p.
func (r Rectangle) Add(p Point) Rectangle {
	return Rectangle{r.Min.Add(p), r.Max.Add(p)}
}

// Sub returns r translated by -p.
func (r Rectangle) Sub(p Point) Rectangle {
	return Rectangle{r.Min.Sub(p), r.Max.Sub(p)}
}

// Empty reports whether r contains no points.
func (r Rectangle) Empty() bool {
	return r.Min.X >= r.Max.X || r.Min.Y >= r.Max.Y
}

// Eq reports whether r and s are equal.
func (r Rectangle) Eq(s Rectangle) bool {
	return r.Min.Eq(s.Min) && r.Max.Eq(s.Max)
}

// Overlaps reports whether r and s share any point.
func (r Rectangle) Overlaps(s Rectangle) bool {
	return r.Min.X < s.Max.X && s.Min.X < r.Max.X &&
		r.Min.Y < s.Max.Y && s.Min.Y < r.Max.Y
}

// In reports whether r is entirely inside s.
func (r Rectangle) In(s Rectangle) bool {
	if r.Empty() {
		return true
	}
	return s.Min.X <= r.Min.X && r.Max.X <= s.Max.X &&
		s.Min.Y <= r.Min.Y && r.Max.Y <= s.Max.Y
}

// Clip clips r to be inside s, returning clipped rectangle and
// whether any pixels remain.
func (r Rectangle) Clip(s Rectangle) (Rectangle, bool) {
	if r.Min.X < s.Min.X {
		r.Min.X = s.Min.X
	}
	if r.Min.Y < s.Min.Y {
		r.Min.Y = s.Min.Y
	}
	if r.Max.X > s.Max.X {
		r.Max.X = s.Max.X
	}
	if r.Max.Y > s.Max.Y {
		r.Max.Y = s.Max.Y
	}
	return r, !r.Empty()
}

// Combine returns the smallest rectangle containing both r and s.
func (r Rectangle) Combine(s Rectangle) Rectangle {
	if r.Empty() {
		return s
	}
	if s.Empty() {
		return r
	}
	if r.Min.X > s.Min.X {
		r.Min.X = s.Min.X
	}
	if r.Min.Y > s.Min.Y {
		r.Min.Y = s.Min.Y
	}
	if r.Max.X < s.Max.X {
		r.Max.X = s.Max.X
	}
	if r.Max.Y < s.Max.Y {
		r.Max.Y = s.Max.Y
	}
	return r
}

// Pix is a channel descriptor: what order and depth of pixel components.
type Pix uint32

// Channel descriptor bits.
const (
	CRed    = 0
	CGreen  = 1
	CBlue   = 2
	CGrey   = 3
	CAlpha  = 4
	CMap    = 5
	CIgnore = 6
	NChan   = 7
)

// Standard pixel formats.
const (
	GREY1  Pix = 0x00000013
	GREY2  Pix = 0x00000023
	GREY4  Pix = 0x00000043
	GREY8  Pix = 0x00000083
	CMAP8  Pix = 0x00000585
	RGB15  Pix = 0x05050155
	RGB16  Pix = 0x06050165
	RGB24  Pix = 0x08080888
	RGBA32 Pix = 0x08080888 | (CAlpha+1)<<24 | 8<<28
	ARGB32 Pix = (CAlpha+1)<<0 | 8<<4 | 0x00888808
	ABGR32 Pix = (CAlpha+1)<<0 | 8<<4 | (CBlue+1)<<8 | 8<<12 | (CGreen+1)<<16 | 8<<20 | (CRed+1)<<24 | 8<<28
	XRGB32 Pix = (CIgnore+1)<<0 | 8<<4 | 0x00888808
	XBGR32 Pix = (CIgnore+1)<<0 | 8<<4 | (CBlue+1)<<8 | 8<<12 | (CGreen+1)<<16 | 8<<20 | (CRed+1)<<24 | 8<<28
	BGR24  Pix = (CBlue+1)<<0 | 8<<4 | (CGreen+1)<<8 | 8<<12 | (CRed+1)<<16 | 8<<20
)

// Display represents a connection to a Plan 9 draw device.
type Display struct {
	mu sync.Mutex

	// File descriptors
	ctlfd  *os.File
	datafd *os.File
	reffd  *os.File

	// Display info
	dirno       int     // directory number in /dev/draw
	Image       *Image  // the display memory
	Screen      *Screen // the window's screen (for window images)
	Windows     *Image  // the window (for rio)
	White       *Image  // white
	Black       *Image  // black
	Opaque      *Image  // white with alpha = 0xFF
	Transparent *Image  // black with alpha = 0x00

	// Buffer for protocol messages
	buf     []byte
	bufsize int

	// Default font
	DefaultFont    *Font
	DefaultSubfont *Subfont

	// Image id counter
	imageid int

	// Error handler
	Error func(string)

	// Screen DPI
	DPI int
}

// Screen represents a Plan 9 screen (for layers).
type Screen struct {
	Display *Display
	id      int
	Image   *Image // backing image
	Fill    *Image // fill color
}

// Image represents a Plan 9 image.
type Image struct {
	Display *Display
	Screen  *Screen // nil if not a window
	id      int
	Pix     Pix       // pixel format
	Depth   int       // bits per pixel
	Repl    bool      // whether image replicates
	R       Rectangle // bounds
	Clipr   Rectangle // clipping region
	next    *Image    // for screen windows
	// For fonts
	width int // for subfont glyphs: bytes per scan line
}

// Refresh constants for getwindow.
const (
	Refbackup = 0
	Refnone   = 1
	Refmesg   = 2
)

// End styles for line.
const (
	Endsquare = 0
	Enddisc   = 1
	Endarrow  = 2
	Endmask   = 0x1F
)

// Compositing operators (Porter-Duff).
type Op int

const (
	SoverD Op = iota // default
	DoverS
	SatopD
	DatopS
	SxorD
	DxorS
	Clear
	S
	D
	SoutD
	DoutS
	SinD
	DinS
	Ncomp
)

// drawBufSize is the size of the protocol message buffer.
const drawBufSize = 8000

// Font represents a font.
type Font struct {
	Display    *Display
	Name       string
	Height     int // line height
	Ascent     int // height above baseline
	width      int // of widest char (for snarf/paste optimization)
	age        uint32
	maxdepth   int
	ncache     int
	nsubf      int
	cache      []Cacheinfo
	subf       []Cachesubf
	sub        []*Cachefont
	cacheimage *Image
}

// Subfont is a collection of character glyphs forming part of a font.
type Subfont struct {
	Name   string
	N      int        // number of chars
	Height int        // line height
	Ascent int        // height above baseline
	Info   []Fontchar // character descriptions
	Bits   *Image     // image holding glyphs
	ref    int
}

// Fontchar describes a single character in a subfont.
type Fontchar struct {
	X      int  // left edge of glyph in Bits
	Top    byte // pixels above baseline
	Bottom byte // pixels below baseline
	Left   int8 // pixels to left of origin
	Width  byte // pixels to advance
}

// Cachefont describes a range of characters in a font.
type Cachefont struct {
	Min         int    // low rune
	Max         int    // high rune + 1
	Offset      int    // index offset to add
	Name        string // file name
	Subfontname string
}

// Cacheinfo describes a cached glyph.
type Cacheinfo struct {
	x     uint16
	width byte
	left  int8
	value rune
	age   uint32
}

// Cachesubf describes a cached subfont.
type Cachesubf struct {
	age uint32
	cf  *Cachefont
	f   *Subfont
}

// Mouse represents mouse state.
type Mouse struct {
	Point          // position
	Buttons int    // button bits
	Msec    uint32 // timestamp
}

// Mousectl provides access to mouse events.
type Mousectl struct {
	Mouse   // current state
	C       chan Mouse
	Resize  chan bool
	Display *Display
	file    *os.File
}

// Keyboardctl provides access to keyboard events.
type Keyboardctl struct {
	C    chan rune
	file *os.File
}

// Menu for menuhit.
type Menu struct {
	Item    []string
	Gen     func(int) string
	Lasthit int
}
