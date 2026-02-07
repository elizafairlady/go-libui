package frame

import (
	"unicode/utf8"

	"github.com/elizafairlady/go-libui/draw"
)

// canfit returns how many runes of box b fit starting at point pt
// within the frame's rectangle. For break chars, returns 1 if the
// minimum width fits, 0 otherwise.
func (f *Frame) canfit(pt draw.Point, b *frbox) int {
	left := f.R.Max.X - pt.X
	if b.nrune < 0 {
		if b.minwid <= left {
			return 1
		}
		return 0
	}
	if left >= b.wid {
		return b.nrune
	}
	nr := 0
	p := b.ptr
	for len(p) > 0 {
		_, size := utf8.DecodeRune(p)
		w := f.Font.StringNWidth(string(p), 1)
		left -= w
		if left < 0 {
			break
		}
		p = p[size:]
		nr++
	}
	return nr
}

// cklinewrap checks whether box b fits at point p within the frame.
// If it doesn't, p is moved to the start of the next line.
// Uses the box's full width (or minwid for break chars).
func (f *Frame) cklinewrap(p *draw.Point, b *frbox) {
	w := b.wid
	if b.nrune < 0 {
		w = b.minwid
	}
	if w > f.R.Max.X-p.X {
		p.X = f.R.Min.X
		p.Y += f.Font.Height
	}
}

// cklinewrap0 is like cklinewrap but uses canfit to check
// whether any part of the box fits (for text boxes that might
// be partially fittable).
func (f *Frame) cklinewrap0(p *draw.Point, b *frbox) {
	if f.canfit(*p, b) == 0 {
		p.X = f.R.Min.X
		p.Y += f.Font.Height
	}
}

// advance moves point p past box b. For newlines, moves to the
// start of the next line. For everything else, advances by wid.
func (f *Frame) advance(p *draw.Point, b *frbox) {
	if b.nrune < 0 && b.bc == '\n' {
		p.X = f.R.Min.X
		p.Y += f.Font.Height
	} else {
		p.X += b.wid
	}
}

// newwid computes the display width of box b at point pt,
// updating b.wid. For tab chars, the width depends on position.
func (f *Frame) newwid(pt draw.Point, b *frbox) int {
	b.wid = f.newwid0(pt, b)
	return b.wid
}

// newwid0 computes display width of b at position pt without
// modifying b.wid.
func (f *Frame) newwid0(pt draw.Point, b *frbox) int {
	c := f.R.Max.X
	x := pt.X
	if b.nrune >= 0 || b.bc != '\t' {
		return b.wid
	}
	if x+b.minwid > c {
		x = f.R.Min.X
	}
	x += f.Maxtab
	x -= (x - f.R.Min.X) % f.Maxtab
	if x-pt.X < b.minwid || x > c {
		x = pt.X + b.minwid
	}
	return x - pt.X
}

// clean looks for mergeable adjacent text boxes and merges them
// where possible (when they fit on the same line).
func (f *Frame) clean(pt draw.Point, n0, n1 int) {
	c := f.R.Max.X
	for nb := n0; nb < n1-1; nb++ {
		b := &f.box[nb]
		f.cklinewrap(&pt, b)
		for b.nrune >= 0 && nb < n1-1 && f.box[nb+1].nrune >= 0 && pt.X+b.wid+f.box[nb+1].wid < c {
			f.mergebox(nb)
			n1--
			b = &f.box[nb]
		}
		f.advance(&pt, &f.box[nb])
	}
	for nb := n1 - 1; nb < f.nbox; nb++ {
		// The C code continues checking from n1-1 to end of boxes
		// but only advances without merging.
		if nb < 0 {
			continue
		}
		b := &f.box[nb]
		f.cklinewrap(&pt, b)
		f.advance(&pt, &f.box[nb])
	}
	f.Lastlinefull = 0
	if pt.Y >= f.R.Max.Y {
		f.Lastlinefull = 1
	}
}
