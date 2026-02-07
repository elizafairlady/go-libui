package draw

// Line draws a line from p0 to p1 with thickness 1+2*radius.
// End0 and end1 specify the end styles (Endsquare, Enddisc, or Endarrow).
func (dst *Image) Line(p0, p1 Point, end0, end1, radius int, src *Image, sp Point) {
	dst.LineOp(p0, p1, end0, end1, radius, src, sp, SoverD)
}

// LineOp is Line with a compositing operator.
func (dst *Image) LineOp(p0, p1 Point, end0, end1, radius int, src *Image, sp Point, op Op) {
	if dst == nil || dst.Display == nil {
		return
	}
	d := dst.Display

	if src == nil {
		src = d.Black
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Format: 'L' dstid[4] p0[2*4] p1[2*4] end0[4] end1[4] radius[4] srcid[4] sp[2*4]
	a, err := d.bufimageop(1+4+2*4+2*4+4+4+4+4+2*4, op)
	if err != nil {
		return
	}

	a[0] = 'L'
	bplong(a[1:], uint32(dst.id))
	bplong(a[5:], uint32(p0.X))
	bplong(a[9:], uint32(p0.Y))
	bplong(a[13:], uint32(p1.X))
	bplong(a[17:], uint32(p1.Y))
	bplong(a[21:], uint32(end0))
	bplong(a[25:], uint32(end1))
	bplong(a[29:], uint32(radius))
	bplong(a[33:], uint32(src.id))
	bplong(a[37:], uint32(sp.X))
	bplong(a[41:], uint32(sp.Y))
}

// addcoord appends a compressed coordinate delta to buf.
// Returns the number of bytes written.
func addcoord(buf []byte, oldx, newx int) int {
	dx := newx - oldx
	// does dx fit in 7 signed bits?
	if uint(dx-(-0x40)) <= 0x7F {
		buf[0] = byte(dx) & 0x7F
		return 1
	}
	buf[0] = 0x80 | byte(newx&0x7F)
	buf[1] = byte(newx >> 7)
	buf[2] = byte(newx >> 15)
	return 3
}

// Poly draws a polygon connecting the points.
func (dst *Image) Poly(p []Point, end0, end1, radius int, src *Image, sp Point) {
	dst.PolyOp(p, end0, end1, radius, src, sp, SoverD)
}

// PolyOp is Poly with a compositing operator.
func (dst *Image) PolyOp(p []Point, end0, end1, radius int, src *Image, sp Point, op Op) {
	if dst == nil || dst.Display == nil || len(p) == 0 {
		return
	}
	d := dst.Display

	if src == nil {
		src = d.Black
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.dopoly('p', dst, p, end0, end1, radius, src, sp, op)
}

// FillPoly fills a polygon.
func (dst *Image) FillPoly(p []Point, wind int, src *Image, sp Point) {
	dst.FillPolyOp(p, wind, src, sp, SoverD)
}

// FillPolyOp is FillPoly with a compositing operator.
func (dst *Image) FillPolyOp(p []Point, wind int, src *Image, sp Point, op Op) {
	if dst == nil || dst.Display == nil || len(p) == 0 {
		return
	}
	d := dst.Display

	if src == nil {
		src = d.Black
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.dopoly('P', dst, p, wind, 0, 0, src, sp, op)
}

// dopoly issues a poly draw command with compressed point encoding.
func (d *Display) dopoly(cmd byte, dst *Image, pp []Point, end0, end1, radius int, src *Image, sp Point, op Op) {
	if len(pp) == 0 {
		return
	}

	// Encode points with addcoord compression
	// Each point can take at most 6 bytes (3 for x + 3 for y)
	t := make([]byte, len(pp)*6)
	u := 0
	ox, oy := 0, 0
	for _, p := range pp {
		n := addcoord(t[u:], ox, p.X)
		u += n
		ox = p.X
		n = addcoord(t[u:], oy, p.Y)
		u += n
		oy = p.Y
	}

	a, err := d.bufimageop(1+4+2+4+4+4+4+2*4+u, op)
	if err != nil {
		return
	}

	a[0] = cmd
	bplong(a[1:], uint32(dst.id))
	bpshort(a[5:], uint16(len(pp)-1))
	bplong(a[7:], uint32(end0))
	bplong(a[11:], uint32(end1))
	bplong(a[15:], uint32(radius))
	bplong(a[19:], uint32(src.id))
	bplong(a[23:], uint32(sp.X))
	bplong(a[27:], uint32(sp.Y))
	copy(a[31:], t[:u])
}

// plist is used internally for converting bezier curves to point lists.
type plist struct {
	p []Point
}

func (l *plist) append(p Point) {
	l.p = append(l.p, p)
}

func normsq(p Point) int {
	return p.X*p.X + p.Y*p.Y
}

func psdist(p, a, b Point) int {
	p = p.Sub(a)
	b = b.Sub(a)
	num := p.X*b.X + p.Y*b.Y
	if num <= 0 {
		return normsq(p)
	}
	den := normsq(b)
	if num >= den {
		return normsq(b.Sub(p))
	}
	return normsq(b.Mul(num).Div(den).Sub(p))
}

func bpts1(l *plist, p0, p1, p2, p3 Point, scale int) {
	tp0 := p0.Div(scale)
	tp1 := p1.Div(scale)
	tp2 := p2.Div(scale)
	tp3 := p3.Div(scale)
	if psdist(tp1, tp0, tp3) <= 1 && psdist(tp2, tp0, tp3) <= 1 {
		l.append(tp0)
		l.append(tp1)
		l.append(tp2)
	} else {
		if scale > (1 << 12) {
			p0 = tp0
			p1 = tp1
			p2 = tp2
			p3 = tp3
			scale = 1
		}
		p01 := p0.Add(p1)
		p12 := p1.Add(p2)
		p23 := p2.Add(p3)
		p012 := p01.Add(p12)
		p123 := p12.Add(p23)
		p0123 := p012.Add(p123)
		bpts1(l, p0.Mul(8), p01.Mul(4), p012.Mul(2), p0123, scale*8)
		bpts1(l, p0123, p123.Mul(2), p23.Mul(4), p3.Mul(8), scale*8)
	}
}

func bpts(l *plist, p0, p1, p2, p3 Point) {
	bpts1(l, p0, p1, p2, p3, 1)
}

func bezierpts(l *plist, p0, p1, p2, p3 Point) {
	bpts(l, p0, p1, p2, p3)
	l.append(p3)
}

func bezsplinepts(l *plist, pt []Point) {
	npt := len(pt)
	if npt < 3 {
		return
	}
	ep := npt - 3
	periodic := pt[0].Eq(pt[ep+2])

	if periodic {
		a := pt[ep+1].Add(pt[0]).Div(2)
		b := pt[ep+1].Add(pt[0].Mul(5)).Div(6)
		c := pt[0].Mul(5).Add(pt[1]).Div(6)
		dd := pt[0].Add(pt[1]).Div(2)
		bpts(l, a, b, c, dd)
	}

	var lastd Point
	for i := 0; i <= ep; i++ {
		var a, b, c, dd Point
		if i == 0 && !periodic {
			a = pt[0]
			b = pt[0].Add(pt[1].Mul(2)).Div(3)
		} else {
			a = pt[i].Add(pt[i+1]).Div(2)
			b = pt[i].Add(pt[i+1].Mul(5)).Div(6)
		}
		if i == ep && !periodic {
			c = pt[i+1].Mul(2).Add(pt[i+2]).Div(3)
			dd = pt[i+2]
		} else {
			c = pt[i+1].Mul(5).Add(pt[i+2]).Div(6)
			dd = pt[i+1].Add(pt[i+2]).Div(2)
		}
		bpts(l, a, b, c, dd)
		lastd = dd
	}
	l.append(lastd)
}

// Bezier draws a cubic Bezier curve.
func (dst *Image) Bezier(a, b, c, dd Point, end0, end1, radius int, src *Image, sp Point) {
	dst.BezierOp(a, b, c, dd, end0, end1, radius, src, sp, SoverD)
}

// BezierOp is Bezier with a compositing operator.
func (dst *Image) BezierOp(a, b, c, dd Point, end0, end1, radius int, src *Image, sp Point, op Op) {
	if dst == nil || dst.Display == nil {
		return
	}
	var l plist
	bezierpts(&l, a, b, c, dd)
	if len(l.p) == 0 {
		return
	}
	dst.PolyOp(l.p, end0, end1, radius, src, sp.Add(a.Sub(l.p[0])), op)
}

// FillBezier fills a region bounded by a bezier curve.
func (dst *Image) FillBezier(a, b, c, dd Point, wind int, src *Image, sp Point) {
	dst.FillBezierOp(a, b, c, dd, wind, src, sp, SoverD)
}

// FillBezierOp is FillBezier with a compositing operator.
func (dst *Image) FillBezierOp(a, b, c, dd Point, wind int, src *Image, sp Point, op Op) {
	if dst == nil || dst.Display == nil {
		return
	}
	var l plist
	bezierpts(&l, a, b, c, dd)
	if len(l.p) == 0 {
		return
	}
	dst.FillPolyOp(l.p, wind, src, sp.Add(a.Sub(l.p[0])), op)
}

// BezSpline draws a bezier spline through the given points.
func (dst *Image) BezSpline(p []Point, end0, end1, radius int, src *Image, sp Point) {
	dst.BezSplineOp(p, end0, end1, radius, src, sp, SoverD)
}

// BezSplineOp is BezSpline with a compositing operator.
func (dst *Image) BezSplineOp(pts []Point, end0, end1, radius int, src *Image, sp Point, op Op) {
	if dst == nil || dst.Display == nil || len(pts) < 3 {
		return
	}
	var l plist
	bezsplinepts(&l, pts)
	if len(l.p) == 0 {
		return
	}
	dst.PolyOp(l.p, end0, end1, radius, src, sp.Add(pts[0].Sub(l.p[0])), op)
}

// FillBezSpline fills a region bounded by a bezier spline.
func (dst *Image) FillBezSpline(p []Point, wind int, src *Image, sp Point) {
	dst.FillBezSplineOp(p, wind, src, sp, SoverD)
}

// FillBezSplineOp is FillBezSpline with a compositing operator.
func (dst *Image) FillBezSplineOp(pts []Point, wind int, src *Image, sp Point, op Op) {
	if dst == nil || dst.Display == nil || len(pts) < 3 {
		return
	}
	var l plist
	bezsplinepts(&l, pts)
	if len(l.p) == 0 {
		return
	}
	dst.FillPolyOp(l.p, wind, src, sp.Add(pts[0].Sub(l.p[0])), op)
}
