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

	srcid := 0
	if src != nil {
		srcid = src.id
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Build the 'l' (line) or 'L' (line with op) message
	// Format: 'l' dstid[4] p0[2*4] p1[2*4] end0[4] end1[4] radius[4] srcid[4] sp[2*4]
	var a [1 + 4 + 2*4 + 2*4 + 4 + 4 + 4 + 4 + 2*4 + 1]byte
	if op != SoverD {
		a[0] = 'L'
	} else {
		a[0] = 'l'
	}
	bplong(a[1:], uint32(dst.id))
	bplong(a[5:], uint32(p0.X))
	bplong(a[9:], uint32(p0.Y))
	bplong(a[13:], uint32(p1.X))
	bplong(a[17:], uint32(p1.Y))
	bplong(a[21:], uint32(end0))
	bplong(a[25:], uint32(end1))
	bplong(a[29:], uint32(radius))
	bplong(a[33:], uint32(srcid))
	bplong(a[37:], uint32(sp.X))
	bplong(a[41:], uint32(sp.Y))

	n := 45
	if op != SoverD {
		a[n] = byte(op)
		n++
	}

	if err := d.flushBuffer(n); err != nil {
		return
	}
	copy(d.buf[d.bufsize:], a[:n])
	d.bufsize += n
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

	srcid := 0
	if src != nil {
		srcid = src.id
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Build the 'p' (poly) or 'P' (poly with op) message
	// Format: 'p' dstid[4] n[2] end0[4] end1[4] radius[4] srcid[4] sp[2*4] p[n][2*4]
	headerSize := 1 + 4 + 2 + 4 + 4 + 4 + 4 + 2*4
	if op != SoverD {
		headerSize++ // op byte
	}
	totalSize := headerSize + len(p)*8

	if err := d.flushBuffer(totalSize); err != nil {
		return
	}

	buf := d.buf[d.bufsize:]
	if op != SoverD {
		buf[0] = 'P'
	} else {
		buf[0] = 'p'
	}
	bplong(buf[1:], uint32(dst.id))
	bpshort(buf[5:], uint16(len(p)))
	bplong(buf[7:], uint32(end0))
	bplong(buf[11:], uint32(end1))
	bplong(buf[15:], uint32(radius))
	bplong(buf[19:], uint32(srcid))
	bplong(buf[23:], uint32(sp.X))
	bplong(buf[27:], uint32(sp.Y))

	off := 31
	if op != SoverD {
		buf[off] = byte(op)
		off++
	}

	for _, pt := range p {
		bplong(buf[off:], uint32(pt.X))
		bplong(buf[off+4:], uint32(pt.Y))
		off += 8
	}

	d.bufsize += totalSize
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

	srcid := 0
	if src != nil {
		srcid = src.id
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Build the 'P' (fill poly) message with wind instead of end styles
	// Actually use 'y' for filled polygon
	headerSize := 1 + 4 + 2 + 4 + 4 + 2*4
	if op != SoverD {
		headerSize++
	}
	totalSize := headerSize + len(p)*8

	if err := d.flushBuffer(totalSize); err != nil {
		return
	}

	buf := d.buf[d.bufsize:]
	buf[0] = 'y'
	bplong(buf[1:], uint32(dst.id))
	bpshort(buf[5:], uint16(len(p)))
	bplong(buf[7:], uint32(wind))
	bplong(buf[11:], uint32(srcid))
	bplong(buf[15:], uint32(sp.X))
	bplong(buf[19:], uint32(sp.Y))

	off := 23
	if op != SoverD {
		buf[off] = byte(op)
		off++
	}

	for _, pt := range p {
		bplong(buf[off:], uint32(pt.X))
		bplong(buf[off+4:], uint32(pt.Y))
		off += 8
	}

	d.bufsize += totalSize
}

// Bezier draws a cubic Bezier curve.
func (dst *Image) Bezier(a, b, c, d Point, end0, end1, radius int, src *Image, sp Point) {
	dst.BezierOp(a, b, c, d, end0, end1, radius, src, sp, SoverD)
}

// BezierOp is Bezier with a compositing operator.
func (dst *Image) BezierOp(a, b, c, dd Point, end0, end1, radius int, src *Image, sp Point, op Op) {
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

	// Build the 'z' (bezier) message
	// Format: 'z' dstid[4] a[2*4] b[2*4] c[2*4] d[2*4] end0[4] end1[4] radius[4] srcid[4] sp[2*4]
	var msg [1 + 4 + 4*2*4 + 4 + 4 + 4 + 4 + 2*4 + 1]byte
	if op != SoverD {
		msg[0] = 'Z'
	} else {
		msg[0] = 'z'
	}
	bplong(msg[1:], uint32(dst.id))
	bplong(msg[5:], uint32(a.X))
	bplong(msg[9:], uint32(a.Y))
	bplong(msg[13:], uint32(b.X))
	bplong(msg[17:], uint32(b.Y))
	bplong(msg[21:], uint32(c.X))
	bplong(msg[25:], uint32(c.Y))
	bplong(msg[29:], uint32(dd.X))
	bplong(msg[33:], uint32(dd.Y))
	bplong(msg[37:], uint32(end0))
	bplong(msg[41:], uint32(end1))
	bplong(msg[45:], uint32(radius))
	bplong(msg[49:], uint32(srcid))
	bplong(msg[53:], uint32(sp.X))
	bplong(msg[57:], uint32(sp.Y))

	n := 61
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

// BezSpline draws a bezier spline through the given points.
func (dst *Image) BezSpline(p []Point, end0, end1, radius int, src *Image, sp Point) {
	dst.BezSplineOp(p, end0, end1, radius, src, sp, SoverD)
}

// BezSplineOp is BezSpline with a compositing operator.
func (dst *Image) BezSplineOp(p []Point, end0, end1, radius int, src *Image, sp Point, op Op) {
	if dst == nil || dst.Display == nil || len(p) < 3 {
		return
	}

	// Convert bezier spline control points to cubic bezier segments
	// and draw each segment
	// B-spline to Bezier conversion for cubic splines
	for i := 0; i+3 <= len(p); i += 3 {
		dst.BezierOp(p[i], p[i+1], p[i+2], p[(i+3)%len(p)], end0, end1, radius, src, sp, op)
	}
}

// FillBezier fills a region bounded by a bezier curve.
func (dst *Image) FillBezier(a, b, c, dd Point, wind int, src *Image, sp Point) {
	dst.FillBezierOp(a, b, c, dd, wind, src, sp, SoverD)
}

// FillBezierOp is FillBezier with a compositing operator.
func (dst *Image) FillBezierOp(a, b, c, dd Point, wind int, src *Image, sp Point, op Op) {
	// Approximate with filled polygon
	pts := bezierToPoints(a, b, c, dd, 16)
	dst.FillPolyOp(pts, wind, src, sp, op)
}

// FillBezSpline fills a region bounded by a bezier spline.
func (dst *Image) FillBezSpline(p []Point, wind int, src *Image, sp Point) {
	dst.FillBezSplineOp(p, wind, src, sp, SoverD)
}

// FillBezSplineOp is FillBezSpline with a compositing operator.
func (dst *Image) FillBezSplineOp(p []Point, wind int, src *Image, sp Point, op Op) {
	if len(p) < 4 {
		return
	}
	// Approximate spline with polygon
	var pts []Point
	for i := 0; i+3 <= len(p); i += 3 {
		seg := bezierToPoints(p[i], p[i+1], p[i+2], p[(i+3)%len(p)], 16)
		pts = append(pts, seg...)
	}
	dst.FillPolyOp(pts, wind, src, sp, op)
}

// bezierToPoints converts a cubic bezier to a series of points.
func bezierToPoints(a, b, c, d Point, n int) []Point {
	pts := make([]Point, n+1)
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		t2 := t * t
		t3 := t2 * t
		mt := 1 - t
		mt2 := mt * mt
		mt3 := mt2 * mt

		x := mt3*float64(a.X) + 3*mt2*t*float64(b.X) + 3*mt*t2*float64(c.X) + t3*float64(d.X)
		y := mt3*float64(a.Y) + 3*mt2*t*float64(b.Y) + 3*mt*t2*float64(c.Y) + t3*float64(d.Y)
		pts[i] = Pt(int(x+0.5), int(y+0.5))
	}
	return pts
}
