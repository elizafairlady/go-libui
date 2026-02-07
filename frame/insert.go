package frame

import (
	"unicode/utf8"

	"github.com/elizafairlady/go-libui/draw"
)

const tmpSize = 256

// bxscan scans runes into boxes for a temporary frame, computing
// their layout starting at *ppt. Returns the end point and the
// temporary frame holding the scanned boxes.
func (f *Frame) bxscan(runes []rune, ppt *draw.Point) (draw.Point, *Frame) {
	tmp := &Frame{
		R:      f.R,
		B:      f.B,
		Font:   f.Font,
		Maxtab: f.Maxtab,
		Cols:   f.Cols,
	}
	delta := 25
	nl := 0
	ri := 0 // index into runes
	for ri < len(runes) && nl <= f.Maxlines {
		if tmp.nbox == tmp.nalloc {
			tmp.growbox(delta)
			if delta < 10000 {
				delta *= 2
			}
		}
		nb := tmp.nbox
		tmp.nbox++
		b := &tmp.box[nb]
		c := runes[ri]
		if c == '\t' || c == '\n' {
			b.bc = c
			b.wid = 5000
			if c == '\n' {
				b.minwid = 0
			} else {
				b.minwid = f.Font.StringWidth(" ")
			}
			b.nrune = -1
			if c == '\n' {
				nl++
			}
			tmp.Nchars++
			ri++
		} else {
			// Accumulate text runes into a box
			var buf []byte
			nr := 0
			w := 0
			for ri < len(runes) {
				c = runes[ri]
				if c == '\t' || c == '\n' {
					break
				}
				var rbuf [utf8.UTFMax]byte
				n := utf8.EncodeRune(rbuf[:], c)
				if len(buf)+n >= tmpSize {
					break
				}
				w += f.Font.RuneWidth(c)
				buf = append(buf, rbuf[:n]...)
				ri++
				nr++
			}
			p := make([]byte, len(buf))
			copy(p, buf)
			b = &tmp.box[nb] // re-take pointer after possible growbox
			b.ptr = p
			b.wid = w
			b.nrune = nr
			tmp.Nchars += uint32(nr)
		}
	}
	f.cklinewrap0(ppt, &tmp.box[0])
	return tmp.fdraw(*ppt), tmp
}

// chopframe truncates the frame at point pt, at box bn, character
// position p, removing everything that falls below the frame.
func (f *Frame) chopframe(pt draw.Point, p uint32, bn int) {
	for ; bn < f.nbox; bn++ {
		b := &f.box[bn]
		f.cklinewrap(&pt, b)
		if pt.Y >= f.R.Max.Y {
			break
		}
		p += uint32(b.nRune())
		f.advance(&pt, b)
	}
	f.Nchars = p
	f.Nlines = f.Maxlines
	if bn < f.nbox {
		f.delbox(bn, f.nbox-1)
	}
}

// Insert inserts runes into the frame at character position p0.
// The caller must keep a separate copy of the text; the frame
// does not store the full document.
func (f *Frame) Insert(runes []rune, p0 uint32) {
	if p0 > f.Nchars || len(runes) == 0 || f.B == nil {
		return
	}
	n0 := f.findbox(0, 0, p0)
	cn0 := p0
	nn0 := n0
	pt0 := f.ptofcharnb(p0, n0)
	ppt0 := pt0
	opt0 := pt0

	pt1, tmp := f.bxscan(runes, &ppt0)
	ppt1 := pt1

	if n0 < f.nbox {
		f.cklinewrap(&pt0, &f.box[n0])
		f.cklinewrap0(&ppt1, &f.box[n0])
	}
	f.Modified = 1

	// ppt0 and ppt1 are start and end of insertion as they will appear
	// when insertion is complete. pt0 is current location of insertion
	// position (p0); pt1 is terminal point of insertion.
	if f.P0 == f.P1 {
		f.Tick(f.PtOfChar(f.P0), false)
	}

	// Find point where old and new x's line up.
	// pt0 is where the next box is now; pt1 is where it will be after insertion.
	type ptpair struct {
		pt0, pt1 draw.Point
	}
	var pts []ptpair
	npts := 0

	bn := n0
	for pt1.X != pt0.X && pt1.Y != f.R.Max.Y && bn < f.nbox {
		b := &f.box[bn]
		f.cklinewrap(&pt0, b)
		f.cklinewrap0(&pt1, b)
		if b.nrune > 0 {
			n := f.canfit(pt1, b)
			if n == 0 {
				panic("frame: canfit==0 in Insert")
			}
			if n != b.nrune {
				f.splitbox(bn, n)
				b = &f.box[bn]
			}
		}
		pts = append(pts, ptpair{pt0, pt1})
		npts++
		if pt1.Y == f.R.Max.Y {
			break
		}
		f.advance(&pt0, b)
		pt1.X += f.newwid(pt1, b)
		cn0 += uint32(b.nRune())
		bn++
	}

	if pt1.Y > f.R.Max.Y {
		panic("frame: Insert pt1 too far")
	}
	if pt1.Y == f.R.Max.Y && bn < f.nbox {
		f.Nchars -= uint32(f.strlen(bn))
		f.delbox(bn, f.nbox-1)
	}
	if bn == f.nbox {
		f.Nlines = (pt1.Y-f.R.Min.Y)/f.Font.Height + boolToInt(pt1.X > f.R.Min.X)
	} else if pt1.Y != pt0.Y {
		y := f.R.Max.Y
		q0 := pt0.Y + f.Font.Height
		q1 := pt1.Y + f.Font.Height
		f.Nlines += (q1 - q0) / f.Font.Height
		if f.Nlines > f.Maxlines {
			f.chopframe(ppt1, p0, nn0)
		}
		if pt1.Y < y {
			r := f.R
			r.Min.Y = q1
			r.Max.Y = y
			if q1 < y {
				f.B.Draw(r, f.B, draw.Pt(f.R.Min.X, q0))
			}
			r.Min = pt1
			r.Max.X = pt1.X + (f.R.Max.X - pt0.X)
			r.Max.Y = q1
			f.B.Draw(r, f.B, pt0)
		}
	}

	// Move old stuff down to make room. The loop moves stuff between
	// insertion and the point where x's lined up.
	y := 0
	if pt1.Y == f.R.Max.Y {
		y = pt1.Y
	}
	for i := npts - 1; i >= 0; i-- {
		bIdx := bn - (npts - 1 - i) - 1
		if bIdx < 0 || bIdx >= f.nbox {
			continue
		}
		b := &f.box[bIdx]
		pt := pts[i].pt1
		if b.nrune > 0 {
			r := draw.Rectangle{Min: pt}
			r.Max.X = r.Min.X + b.wid
			r.Max.Y = r.Min.Y + f.Font.Height
			f.B.Draw(r, f.B, pts[i].pt0)
			// Clear bit hanging off right
			if i == 0 && pt.Y > pt0.Y {
				r.Min = opt0
				r.Max = opt0
				r.Max.X = f.R.Max.X
				r.Max.Y += f.Font.Height
				var back *draw.Image
				if f.P0 <= cn0 && cn0 < f.P1 {
					back = f.Cols[ColHigh]
				} else {
					back = f.Cols[ColBack]
				}
				f.B.Draw(r, back, r.Min)
			} else if pt.Y < y {
				r.Min = pt
				r.Max = pt
				r.Min.X += b.wid
				r.Max.X = f.R.Max.X
				r.Max.Y += f.Font.Height
				var back *draw.Image
				if f.P0 <= cn0 && cn0 < f.P1 {
					back = f.Cols[ColHigh]
				} else {
					back = f.Cols[ColBack]
				}
				f.B.Draw(r, back, r.Min)
			}
			y = pt.Y
			cn0 -= uint32(b.nrune)
		} else {
			r := draw.Rectangle{Min: pt}
			r.Max.X = r.Min.X + b.wid
			r.Max.Y = r.Min.Y + f.Font.Height
			if r.Max.X >= f.R.Max.X {
				r.Max.X = f.R.Max.X
			}
			cn0--
			var back *draw.Image
			if f.P0 <= cn0 && cn0 < f.P1 {
				back = f.Cols[ColHigh]
			} else {
				back = f.Cols[ColBack]
			}
			f.B.Draw(r, back, r.Min)
			y = 0
			if pt.X == f.R.Min.X {
				y = pt.Y
			}
		}
	}

	// Paint insertion area and draw new text
	var text, back *draw.Image
	if f.P0 < p0 && p0 <= f.P1 {
		text = f.Cols[ColHText]
		back = f.Cols[ColHigh]
	} else {
		text = f.Cols[ColText]
		back = f.Cols[ColBack]
	}
	f.SelectPaint(ppt0, ppt1, back)
	tmp.drawtext(ppt0, text, back)

	// Copy new boxes into frame
	f.addbox(nn0, tmp.nbox)
	for i := 0; i < tmp.nbox; i++ {
		f.box[nn0+i] = tmp.box[i]
	}
	if nn0 > 0 && f.box[nn0-1].nrune >= 0 && ppt0.X-f.box[nn0-1].wid >= f.R.Min.X {
		nn0--
		ppt0.X -= f.box[nn0].wid
	}
	n0End := nn0 + tmp.nbox
	cleanEnd := n0End
	if cleanEnd < f.nbox-1 {
		cleanEnd++
	}
	f.clean(ppt0, nn0, cleanEnd)
	f.Nchars += tmp.Nchars
	if f.P0 >= p0 {
		f.P0 += tmp.Nchars
	}
	if f.P0 > f.Nchars {
		f.P0 = f.Nchars
	}
	if f.P1 >= p0 {
		f.P1 += tmp.Nchars
	}
	if f.P1 > f.Nchars {
		f.P1 = f.Nchars
	}
	if f.P0 == f.P1 {
		f.Tick(f.PtOfChar(f.P0), true)
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
