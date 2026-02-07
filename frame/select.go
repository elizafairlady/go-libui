package frame

import (
	"github.com/elizafairlady/go-libui/draw"
)

// region returns the comparison of a and b:
// -1 if a < b, 0 if a == b, 1 if a > b.
func region(a, b uint32) int {
	if a < b {
		return -1
	}
	if a == b {
		return 0
	}
	return 1
}

// Select tracks mouse selection in the frame. It should be called
// when button 1 is pressed, with mc providing mouse events.
// The frame's P0 and P1 are updated to reflect the selection.
func (f *Frame) Select(mc *draw.Mousectl) {
	mp := mc.Mouse.Point
	b := mc.Mouse.Buttons

	f.Modified = 0
	f.DrawSel(f.PtOfChar(f.P0), f.P0, f.P1, false)
	p0 := f.CharOfPt(mp)
	p1 := p0
	f.P0 = p0
	f.P1 = p1
	pt0 := f.PtOfChar(p0)
	pt1 := f.PtOfChar(p1)
	f.DrawSel(pt0, p0, p1, true)
	reg := 0

	for {
		scrled := false
		if f.Scroll != nil {
			if mp.Y < f.R.Min.Y {
				f.Scroll(f, -(f.R.Min.Y-mp.Y)/f.Font.Height-1)
				p0 = f.P1
				p1 = f.P0
				scrled = true
			} else if mp.Y > f.R.Max.Y {
				f.Scroll(f, (mp.Y-f.R.Max.Y)/f.Font.Height+1)
				p0 = f.P0
				p1 = f.P1
				scrled = true
			}
			if scrled {
				if reg != region(p1, p0) {
					p0, p1 = p1, p0 // undo the swap that will happen below
				}
				pt0 = f.PtOfChar(p0)
				pt1 = f.PtOfChar(p1)
				reg = region(p1, p0)
			}
		}

		q := f.CharOfPt(mp)
		if p1 != q {
			if reg != region(q, p0) {
				// Crossed starting point; reset
				if reg > 0 {
					f.DrawSel(pt0, p0, p1, false)
				} else if reg < 0 {
					f.DrawSel(pt1, p1, p0, false)
				}
				p1 = p0
				pt1 = pt0
				reg = region(q, p0)
				if reg == 0 {
					f.DrawSel(pt0, p0, p1, true)
				}
			}
			qt := f.PtOfChar(q)
			if reg > 0 {
				if q > p1 {
					f.DrawSel(pt1, p1, q, true)
				} else if q < p1 {
					f.DrawSel(qt, q, p1, false)
				}
			} else if reg < 0 {
				if q > p1 {
					f.DrawSel(pt1, p1, q, false)
				} else {
					f.DrawSel(qt, q, p1, true)
				}
			}
			p1 = q
			pt1 = qt
		}

		f.Modified = 0
		if p0 < p1 {
			f.P0 = p0
			f.P1 = p1
		} else {
			f.P0 = p1
			f.P1 = p0
		}

		if scrled {
			f.Scroll(f, 0)
		}

		if !scrled {
			mc.ReadMouse()
		}
		mp = mc.Mouse.Point
		if mc.Mouse.Buttons != b {
			break
		}
	}
}
