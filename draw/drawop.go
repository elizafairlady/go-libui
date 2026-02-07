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

	if src == nil {
		src = d.Black
	}
	if mask == nil {
		mask = d.Opaque
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	a, err := d.bufimageop(1+4+4+4+4*4+2*4+2*4, op)
	if err != nil {
		return
	}

	a[0] = 'd'
	bplong(a[1:], uint32(dst.id))
	bplong(a[5:], uint32(src.id))
	bplong(a[9:], uint32(mask.id))
	bplong(a[13:], uint32(r.Min.X))
	bplong(a[17:], uint32(r.Min.Y))
	bplong(a[21:], uint32(r.Max.X))
	bplong(a[25:], uint32(r.Max.Y))
	bplong(a[29:], uint32(sp.X))
	bplong(a[33:], uint32(sp.Y))
	bplong(a[37:], uint32(mp.X))
	bplong(a[41:], uint32(mp.Y))
}

// bufimageop is like bufimage but prepends an 'O' op command if op != SoverD.
// Returns a slice starting after the op prefix (if any).
func (d *Display) bufimageop(n int, op Op) ([]byte, error) {
	if op != SoverD {
		a, err := d.bufimage(1 + 1 + n)
		if err != nil {
			return nil, err
		}
		a[0] = 'O'
		a[1] = byte(op)
		return a[2:], nil
	}
	return d.bufimage(n)
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
	a, err := d.bufimage(1 + 4 + 1 + 4*4)
	if err != nil {
		return
	}

	a[0] = 'c'
	bplong(a[1:], uint32(i.id))
	if repl {
		a[5] = 1
	} else {
		a[5] = 0
	}
	bplong(a[6:], uint32(clipr.Min.X))
	bplong(a[10:], uint32(clipr.Min.Y))
	bplong(a[14:], uint32(clipr.Max.X))
	bplong(a[18:], uint32(clipr.Max.Y))

	i.Repl = repl
	i.Clipr = clipr
}

// Drawreplxy maps x into the range [min, max) using replication.
func Drawreplxy(min, max, x int) int {
	sx := (x - min) % (max - min)
	if sx < 0 {
		sx += max - min
	}
	return sx + min
}

// Drawrepl maps a point into a rectangle using replication.
func Drawrepl(r Rectangle, p Point) Point {
	p.X = Drawreplxy(r.Min.X, r.Max.X, p.X)
	p.Y = Drawreplxy(r.Min.Y, r.Max.Y, p.Y)
	return p
}
