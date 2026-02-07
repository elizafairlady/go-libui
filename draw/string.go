package draw

const maxCacheChars = 100

func dstClipr(dst *Image) Rectangle {
	if dst == nil {
		return ZR
	}
	return dst.Clipr
}

// String draws the string s in the given font at point p.
// Returns the point at the end of the string.
func (dst *Image) String(p Point, src *Image, sp Point, f *Font, s string) Point {
	return dst.stringImpl(p, src, sp, f, s, nil, 1<<24, dstClipr(dst), nil, ZP, SoverD)
}

// StringOp is String with a compositing operator.
func (dst *Image) StringOp(p Point, src *Image, sp Point, f *Font, s string, op Op) Point {
	return dst.stringImpl(p, src, sp, f, s, nil, 1<<24, dstClipr(dst), nil, ZP, op)
}

// Stringn draws at most n characters of s.
func (dst *Image) Stringn(p Point, src *Image, sp Point, f *Font, s string, n int) Point {
	return dst.stringImpl(p, src, sp, f, s, nil, n, dstClipr(dst), nil, ZP, SoverD)
}

// StringnOp is Stringn with a compositing operator.
func (dst *Image) StringnOp(p Point, src *Image, sp Point, f *Font, s string, n int, op Op) Point {
	return dst.stringImpl(p, src, sp, f, s, nil, n, dstClipr(dst), nil, ZP, op)
}

// StringBg draws s with a background color.
func (dst *Image) StringBg(p Point, src *Image, sp Point, f *Font, s string, bg *Image, bgp Point) Point {
	return dst.stringImpl(p, src, sp, f, s, nil, 1<<24, dstClipr(dst), bg, bgp, SoverD)
}

// StringBgOp is StringBg with a compositing operator.
func (dst *Image) StringBgOp(p Point, src *Image, sp Point, f *Font, s string, bg *Image, bgp Point, op Op) Point {
	return dst.stringImpl(p, src, sp, f, s, nil, 1<<24, dstClipr(dst), bg, bgp, op)
}

// StringnBg draws at most n characters with a background.
func (dst *Image) StringnBg(p Point, src *Image, sp Point, f *Font, s string, n int, bg *Image, bgp Point) Point {
	return dst.stringImpl(p, src, sp, f, s, nil, n, dstClipr(dst), bg, bgp, SoverD)
}

// StringnBgOp is StringnBg with a compositing operator.
func (dst *Image) StringnBgOp(p Point, src *Image, sp Point, f *Font, s string, n int, bg *Image, bgp Point, op Op) Point {
	return dst.stringImpl(p, src, sp, f, s, nil, n, dstClipr(dst), bg, bgp, op)
}

// RuneString draws runes.
func (dst *Image) RuneString(p Point, src *Image, sp Point, f *Font, r []rune) Point {
	return dst.stringImpl(p, src, sp, f, "", r, 1<<24, dstClipr(dst), nil, ZP, SoverD)
}

// RuneStringOp is RuneString with a compositing operator.
func (dst *Image) RuneStringOp(p Point, src *Image, sp Point, f *Font, r []rune, op Op) Point {
	return dst.stringImpl(p, src, sp, f, "", r, 1<<24, dstClipr(dst), nil, ZP, op)
}

// RuneStringn draws at most n runes.
func (dst *Image) RuneStringn(p Point, src *Image, sp Point, f *Font, r []rune, n int) Point {
	return dst.stringImpl(p, src, sp, f, "", r, n, dstClipr(dst), nil, ZP, SoverD)
}

// RuneStringnOp is RuneStringn with a compositing operator.
func (dst *Image) RuneStringnOp(p Point, src *Image, sp Point, f *Font, r []rune, n int, op Op) Point {
	return dst.stringImpl(p, src, sp, f, "", r, n, dstClipr(dst), nil, ZP, op)
}

// RuneStringBg draws runes with a background.
func (dst *Image) RuneStringBg(p Point, src *Image, sp Point, f *Font, r []rune, bg *Image, bgp Point) Point {
	return dst.stringImpl(p, src, sp, f, "", r, 1<<24, dstClipr(dst), bg, bgp, SoverD)
}

// RuneStringBgOp is RuneStringBg with a compositing operator.
func (dst *Image) RuneStringBgOp(p Point, src *Image, sp Point, f *Font, r []rune, bg *Image, bgp Point, op Op) Point {
	return dst.stringImpl(p, src, sp, f, "", r, 1<<24, dstClipr(dst), bg, bgp, op)
}

// RuneStringnBg draws at most n runes with a background.
func (dst *Image) RuneStringnBg(p Point, src *Image, sp Point, f *Font, r []rune, n int, bg *Image, bgp Point) Point {
	return dst.stringImpl(p, src, sp, f, "", r, n, dstClipr(dst), bg, bgp, SoverD)
}

// RuneStringnBgOp is RuneStringnBg with a compositing operator.
func (dst *Image) RuneStringnBgOp(p Point, src *Image, sp Point, f *Font, r []rune, n int, bg *Image, bgp Point, op Op) Point {
	return dst.stringImpl(p, src, sp, f, "", r, n, dstClipr(dst), bg, bgp, op)
}

// stringImpl is the unified _string() implementation.
// Port of 9front _string().
func (dst *Image) stringImpl(pt Point, src *Image, sp Point, f *Font, s string, runes []rune, maxn int, clipr Rectangle, bg *Image, bgp Point, op Op) Point {
	if dst == nil || dst.Display == nil || f == nil {
		return pt
	}

	d := dst.Display

	var sptr *string
	var rptr *[]rune

	if len(s) > 0 {
		sptr = &s
	}
	if len(runes) > 0 {
		rptr = &runes
	}

	var subfontname *string
	try := 0
	cbuf := make([]uint16, maxCacheChars)

	for (sptr != nil && len(*sptr) > 0) || (rptr != nil && len(*rptr) > 0) {
		if maxn <= 0 {
			break
		}
		max := maxCacheChars
		if maxn < max {
			max = maxn
		}

		if subfontname != nil {
			// Need to load a subfont
			sf := d.openSubfont(*subfontname)
			if sf == nil {
				if d.DefaultFont == nil || d.DefaultFont == f {
					break
				}
				f = d.DefaultFont
			}
			subfontname = nil
		}

		// cachechars runs WITHOUT the display lock (matches C).
		// loadchar/fontresize inside it take the lock only for bufimage calls.
		n, wid, sfname := f.cachechars(sptr, rptr, cbuf, max)
		subfontname = sfname

		if n <= 0 {
			if n == 0 {
				try++
				if try > 10 {
					break
				}
				continue
			}
			// n < 0: skip one character
			if rptr != nil && len(*rptr) > 0 {
				r := *rptr
				*rptr = r[1:]
			} else if sptr != nil && len(*sptr) > 0 {
				rs := []rune(*sptr)
				if len(rs) > 0 {
					rest := string(rs[1:])
					*sptr = rest
				}
			}
			maxn--
			continue
		}
		try = 0

		// Build 's' or 'x' protocol message — lock only for bufimage
		m := 47 + 2*n
		if bg != nil {
			m += 4 + 2*4
		}

		d.mu.Lock()
		b, err := bufimageop(d, m, op)
		if err != nil {
			d.mu.Unlock()
			break
		}

		if bg != nil {
			b[0] = 'x'
		} else {
			b[0] = 's'
		}
		bplong(b[1:], uint32(dst.id))
		bplong(b[5:], uint32(src.id))
		bplong(b[9:], uint32(f.cacheimage.id))
		bplong(b[13:], uint32(pt.X))
		bplong(b[17:], uint32(pt.Y+f.Ascent))
		bplong(b[21:], uint32(clipr.Min.X))
		bplong(b[25:], uint32(clipr.Min.Y))
		bplong(b[29:], uint32(clipr.Max.X))
		bplong(b[33:], uint32(clipr.Max.Y))
		bplong(b[37:], uint32(sp.X))
		bplong(b[41:], uint32(sp.Y))
		bpshort(b[45:], uint16(n))
		off := 47
		if bg != nil {
			bplong(b[off:], uint32(bg.id))
			bplong(b[off+4:], uint32(bgp.X))
			bplong(b[off+8:], uint32(bgp.Y))
			off += 12
		}
		for i := 0; i < n; i++ {
			bpshort(b[off+2*i:], cbuf[i])
		}
		d.mu.Unlock()

		pt.X += wid
		bgp.X += wid
		f.Agefont()
		maxn -= n
	}

	return pt
}

// bufimageop is the Go port of _bufimageop.
// If op != SoverD, prepends 'O' + op byte.
func bufimageop(d *Display, n int, op Op) ([]byte, error) {
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

// StringWidth returns the width of s when drawn in font f.
func (f *Font) StringWidth(s string) int {
	if f == nil || len(s) == 0 {
		return 0
	}
	return f.stringWidthImpl(&s, nil, 1<<24)
}

// StringNWidth returns the width of the first n characters of s.
func (f *Font) StringNWidth(s string, n int) int {
	if f == nil || len(s) == 0 || n <= 0 {
		return 0
	}
	return f.stringWidthImpl(&s, nil, n)
}

// RuneStringWidth returns the width of r when drawn in font f.
func (f *Font) RuneStringWidth(r []rune) int {
	if f == nil || len(r) == 0 {
		return 0
	}
	return f.stringWidthImpl(nil, &r, 1<<24)
}

// RuneWidth returns the width of a single rune.
func (f *Font) RuneWidth(r rune) int {
	if f == nil {
		return 0
	}
	rs := []rune{r}
	return f.stringWidthImpl(nil, &rs, 1)
}

// BytesWidth returns the width of a byte slice interpreted as UTF-8.
func (f *Font) BytesWidth(b []byte) int {
	s := string(b)
	return f.StringWidth(s)
}

// stringWidthImpl is the unified width calculation.
// Port of 9front _stringnwidth.
func (f *Font) stringWidthImpl(s *string, r *[]rune, max int) int {
	// If cache is not properly initialized, fall back to f.width estimate
	if f.ncache < NFLOOK+1 || len(f.cache) < f.ncache {
		n := 0
		if s != nil {
			n = len([]rune(*s))
		}
		if r != nil {
			n = len(*r)
		}
		if n > max {
			n = max
		}
		charW := f.width
		if charW <= 0 {
			charW = f.Height / 2
		}
		return n * charW
	}

	wid := 0
	cbuf := make([]uint16, maxCacheChars)

	// Make copies to avoid modifying originals
	var scopy string
	var rcopy []rune
	var sptr *string
	var rptr *[]rune
	if s != nil {
		scopy = *s
		sptr = &scopy
	}
	if r != nil {
		rcopy = make([]rune, len(*r))
		copy(rcopy, *r)
		rptr = &rcopy
	}

	for (sptr != nil && len(*sptr) > 0) || (rptr != nil && len(*rptr) > 0) {
		if max <= 0 {
			break
		}
		m := maxCacheChars
		if max < m {
			m = max
		}

		n, w, _ := f.cachechars(sptr, rptr, cbuf, m)

		if n <= 0 {
			// Skip one character
			if rptr != nil && len(*rptr) > 0 {
				rr := *rptr
				*rptr = rr[1:]
			} else if sptr != nil && len(*sptr) > 0 {
				rs := []rune(*sptr)
				if len(rs) > 0 {
					rest := string(rs[1:])
					*sptr = rest
				}
			}
			max--
			continue
		}
		max -= n
		wid += w
	}
	return wid
}

// cacheLookup looks up a rune in the font cache (simple linear scan).
// Used only for simple width estimation when cache system is not active.
func (f *Font) cacheLookup(r rune) (*Cacheinfo, bool) {
	if f == nil || f.cache == nil {
		return nil, false
	}
	for i := range f.cache {
		if f.cache[i].value == r && f.cache[i].age != 0 {
			return &f.cache[i], true
		}
	}
	return nil, false
}
