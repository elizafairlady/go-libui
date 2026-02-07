package draw

// Border draws a border of thickness n inside rectangle r.
func (dst *Image) Border(r Rectangle, n int, color *Image, sp Point) {
	dst.BorderOp(r, n, color, sp, SoverD)
}

// BorderOp is Border with a compositing operator.
func (dst *Image) BorderOp(r Rectangle, n int, color *Image, sp Point, op Op) {
	if n < 0 {
		// Negative n means border outside r
		r = r.Inset(n)
		n = -n
	}
	// Draw four rectangles for the border
	// Top
	dst.DrawOp(Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+n), color, nil, sp, op)
	// Bottom
	dst.DrawOp(Rect(r.Min.X, r.Max.Y-n, r.Max.X, r.Max.Y), color, nil, sp, op)
	// Left
	dst.DrawOp(Rect(r.Min.X, r.Min.Y+n, r.Min.X+n, r.Max.Y-n), color, nil, sp, op)
	// Right
	dst.DrawOp(Rect(r.Max.X-n, r.Min.Y+n, r.Max.X, r.Max.Y-n), color, nil, sp, op)
}
