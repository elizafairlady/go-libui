package frame

import (
	"github.com/elizafairlady/go-libui/draw"
)

// Delete removes characters p0 through p1-1 from the frame.
// Returns the number of lines freed (may be negative if lines
// were added due to reflow, though typically >= 0).
func (f *Frame) Delete(p0, p1 uint32) int {
	if p0 >= f.Nchars || p0 == p1 || f.B == nil {
		return 0
	}
	if p1 > f.Nchars {
		p1 = f.Nchars
	}

	n0 := f.findbox(0, 0, p0)
	if n0 == f.nbox {
		panic("frame: off end in Delete")
	}
	n1 := f.findbox(n0, p0, p1)
	pt0 := f.ptofcharnb(p0, n0)
	pt1 := f.PtOfChar(p1)

	if f.P0 == f.P1 {
		f.Tick(f.PtOfChar(f.P0), false)
	}

	nn0 := n0
	ppt0 := pt0
	f.freebox(n0, n1-1)
	f.Modified = 1

	// Invariants:
	//   pt0 points to beginning, pt1 points to end
	//   n0 is box containing beginning of stuff being deleted
	//   n1, b are box containing beginning of stuff to keep after deletion
	//   cn1 is char position of n1
	cn1 := p1
	for pt1.X != pt0.X && n1 < f.nbox {
		b := &f.box[n1]
		f.cklinewrap0(&pt0, b)
		f.cklinewrap(&pt1, b)
		r := draw.Rectangle{Min: pt0}
		r.Max = pt0
		r.Max.Y += f.Font.Height
		if b.nrune > 0 {
			n := f.canfit(pt0, b)
			if n == 0 {
				panic("frame: canfit==0 in Delete")
			}
			if n != b.nrune {
				f.splitbox(n1, n)
				b = &f.box[n1]
			}
			r.Max.X += b.wid
			f.B.Draw(r, f.B, pt1)
			cn1 += uint32(b.nrune)
		} else {
			r.Max.X += f.newwid0(pt0, b)
			if r.Max.X > f.R.Max.X {
				r.Max.X = f.R.Max.X
			}
			col := f.Cols[ColBack]
			if f.P0 <= cn1 && cn1 < f.P1 {
				col = f.Cols[ColHigh]
			}
			f.B.Draw(r, col, pt0)
			cn1++
		}
		f.advance(&pt1, b)
		pt0.X += f.newwid(pt0, b)
		f.box[n0] = f.box[n1]
		n0++
		n1++
	}

	if n1 == f.nbox && pt0.X != pt1.X {
		// Deleting last thing in window; clean up
		f.SelectPaint(pt0, pt1, f.Cols[ColBack])
	}

	if pt1.Y != pt0.Y {
		pt2 := f.ptofcharptb(32767, pt1, n1)
		if pt2.Y > f.R.Max.Y {
			panic("frame: ptofchar in Delete")
		}
		if n1 < f.nbox {
			q0 := pt0.Y + f.Font.Height
			q1 := pt1.Y + f.Font.Height
			q2 := pt2.Y + f.Font.Height
			if q2 > f.R.Max.Y {
				q2 = f.R.Max.Y
			}
			f.B.Draw(
				draw.Rect(pt0.X, pt0.Y, pt0.X+(f.R.Max.X-pt1.X), q0),
				f.B, pt1,
			)
			f.B.Draw(
				draw.Rect(f.R.Min.X, q0, f.R.Max.X, q0+(q2-q1)),
				f.B, draw.Pt(f.R.Min.X, q1),
			)
			f.SelectPaint(
				draw.Pt(pt2.X, pt2.Y-(pt1.Y-pt0.Y)),
				pt2,
				f.Cols[ColBack],
			)
		} else {
			f.SelectPaint(pt0, pt2, f.Cols[ColBack])
		}
	}

	f.closebox(n0, n1-1)
	if nn0 > 0 && f.box[nn0-1].nrune >= 0 && ppt0.X-f.box[nn0-1].wid >= f.R.Min.X {
		nn0--
		ppt0.X -= f.box[nn0].wid
	}
	cleanEnd := n0
	if cleanEnd < f.nbox-1 {
		cleanEnd++
	}
	f.clean(ppt0, nn0, cleanEnd)

	if f.P1 > p1 {
		f.P1 -= p1 - p0
	} else if f.P1 > p0 {
		f.P1 = p0
	}
	if f.P0 > p1 {
		f.P0 -= p1 - p0
	} else if f.P0 > p0 {
		f.P0 = p0
	}
	f.Nchars -= p1 - p0
	if f.P0 == f.P1 {
		f.Tick(f.PtOfChar(f.P0), true)
	}
	pt0 = f.PtOfChar(f.Nchars)
	n := f.Nlines
	f.Nlines = (pt0.Y-f.R.Min.Y)/f.Font.Height + boolToInt(pt0.X > f.R.Min.X)
	return n - f.Nlines
}
