package draw

// Ellipse draws an ellipse centered at c with semi-axes a and b.
// The thickness is 1+2*thick.
func (dst *Image) Ellipse(c Point, a, b, thick int, src *Image, sp Point) {
	dst.EllipseOp(c, a, b, thick, src, sp, SoverD)
}

// EllipseOp is Ellipse with a compositing operator.
func (dst *Image) EllipseOp(c Point, a, b, thick int, src *Image, sp Point, op Op) {
	dst.doellipse('e', c, a, b, thick, src, sp, 0, 0, op)
}

// FillEllipse fills an ellipse centered at c with semi-axes a and b.
func (dst *Image) FillEllipse(c Point, a, b int, src *Image, sp Point) {
	dst.FillEllipseOp(c, a, b, src, sp, SoverD)
}

// FillEllipseOp is FillEllipse with a compositing operator.
func (dst *Image) FillEllipseOp(c Point, a, b int, src *Image, sp Point, op Op) {
	dst.doellipse('E', c, a, b, 0, src, sp, 0, 0, op)
}

// Arc draws an arc of an ellipse centered at c with semi-axes a and b.
// The arc extends from angle alpha to alpha+phi, measured in degrees.
func (dst *Image) Arc(c Point, a, b, thick int, src *Image, sp Point, alpha, phi int) {
	dst.ArcOp(c, a, b, thick, src, sp, alpha, phi, SoverD)
}

// ArcOp is Arc with a compositing operator.
func (dst *Image) ArcOp(c Point, a, b, thick int, src *Image, sp Point, alpha, phi int, op Op) {
	alpha |= 1 << 31
	dst.doellipse('e', c, a, b, thick, src, sp, alpha, phi, op)
}

// FillArc fills an arc (pie slice) of an ellipse.
func (dst *Image) FillArc(c Point, a, b int, src *Image, sp Point, alpha, phi int) {
	dst.FillArcOp(c, a, b, src, sp, alpha, phi, SoverD)
}

// FillArcOp is FillArc with a compositing operator.
func (dst *Image) FillArcOp(c Point, a, b int, src *Image, sp Point, alpha, phi int, op Op) {
	alpha |= 1 << 31
	dst.doellipse('E', c, a, b, 0, src, sp, alpha, phi, op)
}

func (dst *Image) doellipse(cmd byte, c Point, xr, yr, thick int, src *Image, sp Point, alpha, phi int, op Op) {
	if dst == nil || dst.Display == nil {
		return
	}
	d := dst.Display

	if src == nil {
		src = d.Black
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Uses _bufimageop: 'O' op prefix for non-SoverD
	a, err := d.bufimageop(1+4+4+2*4+4+4+4+2*4+2*4, op)
	if err != nil {
		return
	}

	a[0] = cmd
	bplong(a[1:], uint32(dst.id))
	bplong(a[5:], uint32(src.id))
	bplong(a[9:], uint32(c.X))
	bplong(a[13:], uint32(c.Y))
	bplong(a[17:], uint32(xr))
	bplong(a[21:], uint32(yr))
	bplong(a[25:], uint32(thick))
	bplong(a[29:], uint32(sp.X))
	bplong(a[33:], uint32(sp.Y))
	bplong(a[37:], uint32(alpha))
	bplong(a[41:], uint32(phi))
}
