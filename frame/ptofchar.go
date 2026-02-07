package frame

import (
	"unicode/utf8"

	"github.com/elizafairlady/go-libui/draw"
)

// ptofcharptb returns the Point of character position p, starting
// from point pt at box index bn.
func (f *Frame) ptofcharptb(p uint32, pt draw.Point, bn int) draw.Point {
	for ; bn < f.nbox; bn++ {
		b := &f.box[bn]
		f.cklinewrap(&pt, b)
		l := uint32(b.nRune())
		if p < l {
			if b.nrune > 0 {
				s := b.ptr
				for p > 0 {
					_, size := utf8.DecodeRune(s)
					w := f.Font.StringNWidth(string(s), 1)
					pt.X += w
					s = s[size:]
					p--
				}
			}
			break
		}
		p -= l
		f.advance(&pt, b)
	}
	return pt
}

// PtOfChar returns the Point at which character position p is drawn.
func (f *Frame) PtOfChar(p uint32) draw.Point {
	return f.ptofcharptb(p, f.R.Min, 0)
}

// ptofcharnb returns the Point for character p, but only considers
// boxes up to (not including) nb. Does not do a final advance
// to the next line.
func (f *Frame) ptofcharnb(p uint32, nb int) draw.Point {
	nbox := f.nbox
	f.nbox = nb
	pt := f.ptofcharptb(p, f.R.Min, 0)
	f.nbox = nbox
	return pt
}

// grid snaps a point to the character grid.
func (f *Frame) grid(p draw.Point) draw.Point {
	p.Y -= f.R.Min.Y
	p.Y -= p.Y % f.Font.Height
	p.Y += f.R.Min.Y
	if p.X > f.R.Max.X {
		p.X = f.R.Max.X
	}
	return p
}

// CharOfPt returns the character position closest to point pt.
func (f *Frame) CharOfPt(pt draw.Point) uint32 {
	pt = f.grid(pt)
	qt := f.R.Min
	var p uint32
	bn := 0
	// Advance past lines above pt.Y
	for bn < f.nbox && qt.Y < pt.Y {
		b := &f.box[bn]
		f.cklinewrap(&qt, b)
		if qt.Y >= pt.Y {
			break
		}
		f.advance(&qt, b)
		p += uint32(b.nRune())
		bn++
	}
	// Now on the right line; advance past boxes to pt.X
	for bn < f.nbox && qt.X <= pt.X {
		b := &f.box[bn]
		f.cklinewrap(&qt, b)
		if qt.Y > pt.Y {
			break
		}
		if qt.X+b.wid > pt.X {
			if b.nrune < 0 {
				f.advance(&qt, b)
			} else {
				s := b.ptr
				for len(s) > 0 {
					_, size := utf8.DecodeRune(s)
					w := f.Font.StringNWidth(string(s), 1)
					qt.X += w
					s = s[size:]
					if qt.X > pt.X {
						break
					}
					p++
				}
			}
			break
		}
		p += uint32(b.nRune())
		f.advance(&qt, b)
		bn++
	}
	return p
}
