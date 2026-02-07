package draw

// Borderwidth is the standard border width.
const Borderwidth = 4

// Border draws a border of thickness n inside rectangle r.
func (dst *Image) Border(r Rectangle, n int, color *Image, sp Point) {
	dst.BorderOp(r, n, color, sp, SoverD)
}

// BorderOp is Border with a compositing operator.
func (dst *Image) BorderOp(r Rectangle, n int, color *Image, sp Point, op Op) {
	if n < 0 {
		r = r.Inset(n)
		sp = sp.Add(Pt(n, n))
		n = -n
	}
	// Top
	dst.DrawOp(Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+n),
		color, nil, sp, op)
	// Bottom
	dst.DrawOp(Rect(r.Min.X, r.Max.Y-n, r.Max.X, r.Max.Y),
		color, nil, Pt(sp.X, sp.Y+r.Dy()-n), op)
	// Left
	dst.DrawOp(Rect(r.Min.X, r.Min.Y+n, r.Min.X+n, r.Max.Y-n),
		color, nil, Pt(sp.X, sp.Y+n), op)
	// Right
	dst.DrawOp(Rect(r.Max.X-n, r.Min.Y+n, r.Max.X, r.Max.Y-n),
		color, nil, Pt(sp.X+r.Dx()-n, sp.Y+n), op)
}
