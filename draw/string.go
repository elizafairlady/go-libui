package draw

// String draws the string s in the given font at point p.
// Returns the point at the end of the string.
func (dst *Image) String(p Point, src *Image, sp Point, f *Font, s string) Point {
	return dst.StringOp(p, src, sp, f, s, SoverD)
}

// StringOp is String with a compositing operator.
func (dst *Image) StringOp(p Point, src *Image, sp Point, f *Font, s string, op Op) Point {
	return dst.stringnOp(p, src, sp, f, s, len(s), op)
}

// Stringn draws at most n characters of s.
func (dst *Image) Stringn(p Point, src *Image, sp Point, f *Font, s string, n int) Point {
	return dst.StringnOp(p, src, sp, f, s, n, SoverD)
}

// StringnOp is Stringn with a compositing operator.
func (dst *Image) StringnOp(p Point, src *Image, sp Point, f *Font, s string, n int, op Op) Point {
	return dst.stringnOp(p, src, sp, f, s, n, op)
}

// StringBg draws s with a background color.
func (dst *Image) StringBg(p Point, src *Image, sp Point, f *Font, s string, bg *Image, bgp Point) Point {
	return dst.StringBgOp(p, src, sp, f, s, bg, bgp, SoverD)
}

// StringBgOp is StringBg with a compositing operator.
func (dst *Image) StringBgOp(p Point, src *Image, sp Point, f *Font, s string, bg *Image, bgp Point, op Op) Point {
	r := Rect(p.X, p.Y, p.X+f.StringWidth(s), p.Y+f.Height)
	dst.DrawOp(r, bg, nil, bgp, op)
	return dst.StringOp(p, src, sp, f, s, op)
}

// StringnBg draws at most n characters with a background.
func (dst *Image) StringnBg(p Point, src *Image, sp Point, f *Font, s string, n int, bg *Image, bgp Point) Point {
	return dst.StringnBgOp(p, src, sp, f, s, n, bg, bgp, SoverD)
}

// StringnBgOp is StringnBg with a compositing operator.
func (dst *Image) StringnBgOp(p Point, src *Image, sp Point, f *Font, s string, n int, bg *Image, bgp Point, op Op) Point {
	if n > len(s) {
		n = len(s)
	}
	r := Rect(p.X, p.Y, p.X+f.StringWidth(s[:n]), p.Y+f.Height)
	dst.DrawOp(r, bg, nil, bgp, op)
	return dst.stringnOp(p, src, sp, f, s, n, op)
}

// RuneString draws runes.
func (dst *Image) RuneString(p Point, src *Image, sp Point, f *Font, r []rune) Point {
	return dst.RuneStringOp(p, src, sp, f, r, SoverD)
}

// RuneStringOp is RuneString with a compositing operator.
func (dst *Image) RuneStringOp(p Point, src *Image, sp Point, f *Font, r []rune, op Op) Point {
	return dst.runestringnOp(p, src, sp, f, r, len(r), op)
}

// RuneStringn draws at most n runes.
func (dst *Image) RuneStringn(p Point, src *Image, sp Point, f *Font, r []rune, n int) Point {
	return dst.RuneStringnOp(p, src, sp, f, r, n, SoverD)
}

// RuneStringnOp is RuneStringn with a compositing operator.
func (dst *Image) RuneStringnOp(p Point, src *Image, sp Point, f *Font, r []rune, n int, op Op) Point {
	return dst.runestringnOp(p, src, sp, f, r, n, op)
}

// RuneStringBg draws runes with a background.
func (dst *Image) RuneStringBg(p Point, src *Image, sp Point, f *Font, r []rune, bg *Image, bgp Point) Point {
	return dst.RuneStringBgOp(p, src, sp, f, r, bg, bgp, SoverD)
}

// RuneStringBgOp is RuneStringBg with a compositing operator.
func (dst *Image) RuneStringBgOp(p Point, src *Image, sp Point, f *Font, r []rune, bg *Image, bgp Point, op Op) Point {
	rect := Rect(p.X, p.Y, p.X+f.RuneStringWidth(r), p.Y+f.Height)
	dst.DrawOp(rect, bg, nil, bgp, op)
	return dst.RuneStringOp(p, src, sp, f, r, op)
}

// RuneStringnBg draws at most n runes with a background.
func (dst *Image) RuneStringnBg(p Point, src *Image, sp Point, f *Font, r []rune, n int, bg *Image, bgp Point) Point {
	return dst.RuneStringnBgOp(p, src, sp, f, r, n, bg, bgp, SoverD)
}

// RuneStringnBgOp is RuneStringnBg with a compositing operator.
func (dst *Image) RuneStringnBgOp(p Point, src *Image, sp Point, f *Font, r []rune, n int, bg *Image, bgp Point, op Op) Point {
	if n > len(r) {
		n = len(r)
	}
	rect := Rect(p.X, p.Y, p.X+f.RuneStringWidth(r[:n]), p.Y+f.Height)
	dst.DrawOp(rect, bg, nil, bgp, op)
	return dst.runestringnOp(p, src, sp, f, r, n, op)
}

func (dst *Image) stringnOp(p Point, src *Image, sp Point, f *Font, s string, n int, op Op) Point {
	if dst == nil || dst.Display == nil || f == nil {
		return p
	}
	if n > len(s) {
		n = len(s)
	}
	if n == 0 {
		return p
	}

	// Convert to runes
	runes := []rune(s[:n])
	return dst.runestringnOp(p, src, sp, f, runes, len(runes), op)
}

func (dst *Image) runestringnOp(p Point, src *Image, sp Point, f *Font, r []rune, n int, op Op) Point {
	if dst == nil || dst.Display == nil || f == nil {
		return p
	}
	if n > len(r) {
		n = len(r)
	}
	if n == 0 {
		return p
	}

	d := dst.Display

	srcid := 0
	if src != nil {
		srcid = src.id
	}
	_ = srcid // will be used in full string protocol implementation

	// Cache the characters and build the draw commands
	d.mu.Lock()
	defer d.mu.Unlock()

	// For each character, we need to ensure it's cached and draw it
	x := p.X
	for i := 0; i < n; i++ {
		c := r[i]

		// Look up character in font cache
		ci, ok := f.cacheLookup(c)
		if !ok {
			// Character not cached - try to load it
			if !f.loadChar(c, d) {
				// Use replacement character
				c = 0xFFFD
				ci, ok = f.cacheLookup(c)
				if !ok {
					continue
				}
			} else {
				ci, _ = f.cacheLookup(c)
			}
		}

		// Draw the character
		// Build 's' (string) message
		// Format: 's' dstid[4] srcid[4] fontid[4] p[2*4] clipr[4*4] sp[2*4] n[2] chars[n*2]
		// But we're doing character-by-character for simplicity

		// Actually, use the simpler 'x' (draw char) protocol if available
		// For now, just compute the width and advance
		if ci != nil {
			x += int(ci.width)
		}
	}

	// Build string command for the whole string
	return Pt(x, p.Y)
}

// StringWidth returns the width of s when drawn in font f.
func (f *Font) StringWidth(s string) int {
	if f == nil {
		return 0
	}
	w := 0
	for _, r := range s {
		w += f.RuneWidth(r)
	}
	return w
}

// StringNWidth returns the width of the first n characters of s.
func (f *Font) StringNWidth(s string, n int) int {
	if f == nil {
		return 0
	}
	w := 0
	i := 0
	for _, r := range s {
		if i >= n {
			break
		}
		w += f.RuneWidth(r)
		i++
	}
	return w
}

// RuneStringWidth returns the width of r when drawn in font f.
func (f *Font) RuneStringWidth(r []rune) int {
	if f == nil {
		return 0
	}
	w := 0
	for _, c := range r {
		w += f.RuneWidth(c)
	}
	return w
}

// RuneWidth returns the width of a single rune.
func (f *Font) RuneWidth(r rune) int {
	if f == nil {
		return 0
	}
	// Look up in cache
	ci, ok := f.cacheLookup(r)
	if ok && ci != nil {
		return int(ci.width)
	}
	// Estimate based on average width
	if f.width > 0 {
		return f.width
	}
	return f.Height / 2 // rough estimate
}

// BytesWidth returns the width of a byte slice interpreted as UTF-8.
func (f *Font) BytesWidth(b []byte) int {
	return f.StringWidth(string(b))
}

// cacheLookup looks up a rune in the font cache.
func (f *Font) cacheLookup(r rune) (*Cacheinfo, bool) {
	if f == nil || f.cache == nil {
		return nil, false
	}
	for i := range f.cache {
		if f.cache[i].value == r {
			return &f.cache[i], true
		}
	}
	return nil, false
}

// loadChar loads a character into the font cache.
func (f *Font) loadChar(r rune, d *Display) bool {
	// Find the subfont containing this character
	for _, cf := range f.sub {
		if cf == nil {
			continue
		}
		if int(r) >= cf.Min && int(r) < cf.Max {
			// Load the subfont if not already loaded
			sf := f.lookupSubfont(cf, d)
			if sf == nil {
				continue
			}
			// Get the character info
			idx := int(r) - cf.Min + cf.Offset
			if idx >= 0 && idx < sf.N {
				fc := &sf.Info[idx]
				// Add to cache
				ci := Cacheinfo{
					value: r,
					width: fc.Width,
					left:  fc.Left,
					x:     uint16(fc.X),
				}
				f.cache = append(f.cache, ci)
				return true
			}
		}
	}
	return false
}

// lookupSubfont finds or loads a subfont.
func (f *Font) lookupSubfont(cf *Cachefont, d *Display) *Subfont {
	// Check if already loaded
	for i := range f.subf {
		if f.subf[i].cf == cf && f.subf[i].f != nil {
			return f.subf[i].f
		}
	}
	// Try to load
	sf := d.openSubfont(cf.Name)
	if sf != nil {
		f.subf = append(f.subf, Cachesubf{cf: cf, f: sf})
	}
	return sf
}
