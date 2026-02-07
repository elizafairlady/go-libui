package draw

// Ellipse draws an ellipse centered at c with semi-axes a and b.
// The thickness is 1+2*thick.
func (dst *Image) Ellipse(c Point, a, b, thick int, src *Image, sp Point) {
	dst.EllipseOp(c, a, b, thick, src, sp, SoverD)
}

// EllipseOp is Ellipse with a compositing operator.
func (dst *Image) EllipseOp(c Point, a, b, thick int, src *Image, sp Point, op Op) {
	dst.doellipse('e', c, a, b, thick, src, sp, 0, 64, op)
}

// FillEllipse fills an ellipse centered at c with semi-axes a and b.
func (dst *Image) FillEllipse(c Point, a, b int, src *Image, sp Point) {
	dst.FillEllipseOp(c, a, b, src, sp, SoverD)
}

// FillEllipseOp is FillEllipse with a compositing operator.
func (dst *Image) FillEllipseOp(c Point, a, b int, src *Image, sp Point, op Op) {
	dst.doellipse('E', c, a, b, 0, src, sp, 0, 64, op)
}

// Arc draws an arc of an ellipse centered at c with semi-axes a and b.
// The arc extends from angle alpha to alpha+phi, measured in degrees*64.
func (dst *Image) Arc(c Point, a, b, thick int, src *Image, sp Point, alpha, phi int) {
	dst.ArcOp(c, a, b, thick, src, sp, alpha, phi, SoverD)
}

// ArcOp is Arc with a compositing operator.
func (dst *Image) ArcOp(c Point, a, b, thick int, src *Image, sp Point, alpha, phi int, op Op) {
	dst.doellipse('e', c, a, b, thick, src, sp, alpha, phi, op)
}

// FillArc fills an arc (pie slice) of an ellipse.
func (dst *Image) FillArc(c Point, a, b int, src *Image, sp Point, alpha, phi int) {
	dst.FillArcOp(c, a, b, src, sp, alpha, phi, SoverD)
}

// FillArcOp is FillArc with a compositing operator.
func (dst *Image) FillArcOp(c Point, a, b int, src *Image, sp Point, alpha, phi int, op Op) {
	dst.doellipse('E', c, a, b, 0, src, sp, alpha, phi, op)
}

func (dst *Image) doellipse(cmd byte, c Point, a, b, thick int, src *Image, sp Point, alpha, phi int, op Op) {
	if dst == nil || dst.Display == nil {
		return
	}
	d := dst.Display

	srcid := 0
	if src != nil {
		srcid = src.id
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Build the ellipse message
	// Format: 'e'/'E' dstid[4] c[2*4] a[4] b[4] thick[4] srcid[4] sp[2*4] alpha[4] phi[4]
	var msg [1 + 4 + 2*4 + 4 + 4 + 4 + 4 + 2*4 + 4 + 4 + 1]byte
	msg[0] = cmd
	bplong(msg[1:], uint32(dst.id))
	bplong(msg[5:], uint32(c.X))
	bplong(msg[9:], uint32(c.Y))
	bplong(msg[13:], uint32(a))
	bplong(msg[17:], uint32(b))
	bplong(msg[21:], uint32(thick))
	bplong(msg[25:], uint32(srcid))
	bplong(msg[29:], uint32(sp.X))
	bplong(msg[33:], uint32(sp.Y))
	bplong(msg[37:], uint32(alpha))
	bplong(msg[41:], uint32(phi))

	n := 45
	if op != SoverD {
		msg[n] = byte(op)
		n++
	}

	if err := d.flushBuffer(n); err != nil {
		return
	}
	copy(d.buf[d.bufsize:], msg[:n])
	d.bufsize += n
}
