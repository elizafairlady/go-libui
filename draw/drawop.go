package draw

// Draw copies src to dst at point p within rectangle r.
// This is equivalent to gendraw with nil mask.
func (dst *Image) Draw(r Rectangle, src *Image, sp Point) {
	dst.DrawOp(r, src, nil, sp, SoverD)
}

// DrawOp is Draw with a compositing operator.
func (dst *Image) DrawOp(r Rectangle, src, mask *Image, sp Point, op Op) {
	dst.gendrawop(r, src, sp, mask, sp, op)
}

// GenDraw is the general drawing operation.
// It composites src onto dst through mask.
func (dst *Image) GenDraw(r Rectangle, src *Image, sp Point, mask *Image, mp Point) {
	dst.gendrawop(r, src, sp, mask, mp, SoverD)
}

// GenDrawOp is GenDraw with a compositing operator.
func (dst *Image) GenDrawOp(r Rectangle, src *Image, sp Point, mask *Image, mp Point, op Op) {
	dst.gendrawop(r, src, sp, mask, mp, op)
}

func (dst *Image) gendrawop(r Rectangle, src *Image, sp Point, mask *Image, mp Point, op Op) {
	if dst == nil || dst.Display == nil {
		return
	}
	d := dst.Display

	var srcid, maskid int
	if src != nil {
		srcid = src.id
	} else {
		srcid = dst.id
	}
	if mask != nil {
		maskid = mask.id
	} else {
		// Use display opaque as mask (all 1s)
		if d.Opaque != nil {
			maskid = d.Opaque.id
		} else {
			maskid = srcid
		}
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Build the 'd' (draw) message
	// Format: 'd' dstid[4] srcid[4] maskid[4] r[4*4] sp[2*4] mp[2*4]
	// With op: 'D' is used for draw with op
	var a [1 + 4 + 4 + 4 + 4*4 + 2*4 + 2*4]byte
	if op != SoverD {
		a[0] = 'D'
	} else {
		a[0] = 'd'
	}
	bplong(a[1:], uint32(dst.id))
	bplong(a[5:], uint32(srcid))
	bplong(a[9:], uint32(maskid))
	bplong(a[13:], uint32(r.Min.X))
	bplong(a[17:], uint32(r.Min.Y))
	bplong(a[21:], uint32(r.Max.X))
	bplong(a[25:], uint32(r.Max.Y))
	bplong(a[29:], uint32(sp.X))
	bplong(a[33:], uint32(sp.Y))
	bplong(a[37:], uint32(mp.X))
	bplong(a[41:], uint32(mp.Y))

	n := 45
	if op != SoverD {
		// 'D' message includes op byte
		a[n] = byte(op)
		n++
	}

	if err := d.flushBuffer(n); err != nil {
		return
	}
	copy(d.buf[d.bufsize:], a[:n])
	d.bufsize += n
}

// ReplClipr sets the clipping rectangle and replication flag of an image.
func (i *Image) ReplClipr(repl bool, clipr Rectangle) {
	if i == nil || i.Display == nil {
		return
	}
	d := i.Display

	d.mu.Lock()
	defer d.mu.Unlock()

	// Build the 'c' (clipr) message
	// Format: 'c' id[4] repl[1] clipr[4*4]
	var a [1 + 4 + 1 + 4*4]byte
	a[0] = 'c'
	bplong(a[1:], uint32(i.id))
	if repl {
		a[5] = 1
	}
	bplong(a[6:], uint32(clipr.Min.X))
	bplong(a[10:], uint32(clipr.Min.Y))
	bplong(a[14:], uint32(clipr.Max.X))
	bplong(a[18:], uint32(clipr.Max.Y))

	if err := d.flushBuffer(len(a)); err != nil {
		return
	}
	copy(d.buf[d.bufsize:], a[:])
	d.bufsize += len(a)

	i.Repl = repl
	i.Clipr = clipr
}
