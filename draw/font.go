package draw

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Font cache constants from draw.h.
const (
	LOG2NFCACHE = 6
	NFCACHE     = 1 << LOG2NFCACHE // 64, #chars cached
	NFLOOK      = 5                // #chars to scan in cache
	NFSUBF      = 2                // #subfonts to cache
	MAXFCACHE   = 1024 + NFLOOK    // upper limit
	MAXSUBF     = 50               // generous upper limit
	DSUBF       = 4                // delta for subfont growth
	SUBFAGE     = 10000            // expiry age for subfonts
	CACHEAGE    = 10000            // expiry age for cache entries
)

const PJW = 0 // use NUL==pjw for invisible characters

// OpenFont opens a font file and returns a Font.
func (d *Display) OpenFont(name string) (*Font, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return d.BuildFont(data, name)
}

// BuildFont parses a font description from buf and creates a Font.
// This is a direct port of 9front's buildfont().
func (d *Display) BuildFont(buf []byte, name string) (*Font, error) {
	s := string(buf)

	fnt := &Font{
		Display: d,
		Name:    name,
		ncache:  NFCACHE + NFLOOK,
		nsubf:   NFSUBF,
		age:     1,
	}
	fnt.cache = make([]Cacheinfo, fnt.ncache)
	fnt.subf = make([]Cachesubf, fnt.nsubf)

	// Parse height and ascent
	s = skipWhitespace(s)
	h, rest, ok := parseInt(s)
	if !ok {
		return nil, fmt.Errorf("bad font format: expected height")
	}
	fnt.Height = h
	s = skipWhitespace(rest)

	a, rest, ok := parseInt(s)
	if !ok {
		return nil, fmt.Errorf("bad font format: expected ascent")
	}
	fnt.Ascent = a
	s = skipWhitespace(rest)

	if fnt.Height <= 0 || fnt.Ascent <= 0 {
		return nil, fmt.Errorf("bad height or ascent in font file")
	}

	// Parse subfont ranges
	for len(s) > 0 {
		// Must be looking at a number
		if s[0] < '0' || s[0] > '9' {
			return nil, fmt.Errorf("bad font format: number expected")
		}

		min, rest, ok := parseInt(s)
		if !ok {
			return nil, fmt.Errorf("bad font format: min")
		}
		s = skipWhitespace(rest)

		if len(s) == 0 || s[0] < '0' || s[0] > '9' {
			return nil, fmt.Errorf("bad font format: max expected")
		}
		max, rest, ok := parseInt(s)
		if !ok {
			return nil, fmt.Errorf("bad font format: max")
		}
		s = skipWhitespace(rest)

		if len(s) == 0 || min > 0x10FFFF || max > 0x10FFFF || min > max {
			return nil, fmt.Errorf("illegal subfont range")
		}

		// Try to parse offset: a number followed by whitespace
		offset := 0
		t := s
		if n, rest2, ok := parseInt(t); ok && len(rest2) > 0 && (rest2[0] == ' ' || rest2[0] == '\t' || rest2[0] == '\n') {
			offset = n
			s = skipWhitespace(rest2)
		}

		// Parse filename (non-whitespace token)
		end := 0
		for end < len(s) && s[end] != ' ' && s[end] != '\n' && s[end] != '\t' && s[end] != 0 {
			end++
		}
		if end == 0 {
			return nil, fmt.Errorf("bad font format: missing filename")
		}
		filename := s[:end]
		if end < len(s) {
			s = skipWhitespace(s[end:])
		} else {
			s = ""
		}

		cf := &Cachefont{
			Min:    min,
			Max:    max,
			Offset: offset,
			Name:   filename,
		}
		fnt.sub = append(fnt.sub, cf)
	}

	fnt.nsub = len(fnt.sub)
	return fnt, nil
}

// cachechars looks up/loads characters and returns cache indices.
// Port of 9front cachechars().
// Returns number of translated cache indices, 0 = must retry, -1 = error.
func (f *Font) cachechars(s *string, r *[]rune, cp []uint16, max int) (int, int, *string) {
	// Guard: if cache is not properly initialized, return 0
	if f.ncache < NFLOOK+1 || len(f.cache) < f.ncache {
		return 0, 0, nil
	}

	wid := 0
	var subfontname *string

	si := 0 // position in string
	ri := 0 // position in rune slice
	var sp string
	var rp []rune
	useStr := false

	if s != nil {
		sp = *s
		useStr = true
	}
	if r != nil {
		rp = *r
	}

	i := 0
	for i < max {
		var ch rune
		var sw int
		if useStr {
			if si >= len(sp) {
				break
			}
			// Decode one rune
			ch = rune(sp[si])
			sw = 1
			if ch >= 0x80 {
				// Multi-byte UTF-8: convert via rune
				rs := []rune(sp[si:])
				if len(rs) == 0 {
					break
				}
				ch = rs[0]
				sw = len(string(ch))
			}
		} else {
			if ri >= len(rp) {
				break
			}
			ch = rp[ri]
			sw = 0
		}

		// Hash lookup in cache
		h := (17 * int(uint(ch))) & (f.ncache - NFLOOK - 1)
		var c *Cacheinfo
		var bestAge uint32 = ^uint32(0)
		bestIdx := h
		found := false

		for j := h; j < h+NFLOOK; j++ {
			if f.cache[j].value == ch && f.cache[j].age != 0 {
				c = &f.cache[j]
				h = j
				found = true
				break
			}
			if f.cache[j].age < bestAge {
				bestAge = f.cache[j].age
				bestIdx = j
			}
		}

		if !found {
			// Not found; use oldest entry
			c = &f.cache[bestIdx]
			h = bestIdx

			if bestAge != 0 && (f.age-bestAge) < 500 {
				// Kicking out too recent; try to resize
				nc := 2*(f.ncache-NFLOOK) + NFLOOK
				if nc <= MAXFCACHE {
					if i == 0 {
						f.fontresize(f.width, nc, f.maxdepth)
					}
					break // flush first; retry will resize
				}
			}

			if i > 0 && c.age == f.age {
				break // flush pending string output
			}

			j, sfname := f.loadchar(ch, c, h, i)
			if j <= 0 {
				if j < 0 || i > 0 {
					if sfname != nil {
						subfontname = sfname
					}
					break
				}
				// Skip this character
				if useStr {
					si += sw
				} else {
					ri++
				}
				continue // return -1 = stop retrying
			}
			if sfname != nil {
				subfontname = sfname
			}
		}

		wid += int(c.width)
		c.age = f.age
		cp[i] = uint16(h)
		i++

		if useStr {
			si += sw
		} else {
			ri++
		}
	}

	if s != nil {
		rest := sp[si:]
		*s = rest
	}
	if r != nil {
		*r = rp[ri:]
	}

	return i, wid, subfontname
}

// cf2subfont loads a subfont for a Cachefont entry.
// Port of 9front cf2subfont().
func cf2subfont(cf *Cachefont, f *Font) *Subfont {
	name := cf.Subfontname
	if name == "" {
		depth := 8
		if f.Display != nil && f.Display.ScreenImage != nil {
			depth = f.Display.ScreenImage.Depth
		}
		name = SubfontName(cf.Name, f.Name, depth)
		if name == "" {
			return nil
		}
		cf.Subfontname = name
	}
	sf := LookupSubfont(f.Display, name)
	if sf != nil {
		return sf
	}
	// Try to open from file
	if f.Display != nil {
		sf = f.Display.openSubfont(name)
	}
	return sf
}

// loadchar loads a glyph from subfont into cache and sends 'l' to devdraw.
// Port of 9front loadchar(). Returns 1 on success, 0 on failure, -1 on retry needed.
func (f *Font) loadchar(r rune, c *Cacheinfo, h int, noflush int) (int, *string) {
	pic := r

Again:
	var cf *Cachefont
	for i := 0; i < f.nsub; i++ {
		cf = f.sub[i]
		if cf.Min <= int(pic) && int(pic) <= cf.Max {
			goto Found
		}
	}
	if pic != PJW {
		pic = PJW
		goto Again
	}
	return 0, nil

Found:
	// Choose exact or oldest subf slot
	oi := 0
	var subf *Cachesubf
	for i := 0; i < f.nsubf; i++ {
		if cf == f.subf[i].cf {
			subf = &f.subf[i]
			goto Found2
		}
		if f.subf[i].age < f.subf[oi].age {
			oi = i
		}
	}
	subf = &f.subf[oi]

	if subf.f != nil {
		if f.age-subf.age > SUBFAGE || f.nsubf > MAXSUBF {
			// Ancient data; toss
			if f.Display == nil || subf.f != f.Display.DefaultSubfont {
				subf.f.Free()
			}
			subf.cf = nil
			subf.f = nil
			subf.age = 0
		} else {
			// Too recent; grow instead
			newsubf := make([]Cachesubf, f.nsubf+DSUBF)
			copy(newsubf, f.subf)
			f.subf = newsubf
			subf = &f.subf[f.nsubf]
			f.nsubf += DSUBF
		}
	}
	subf.age = 0
	subf.cf = nil
	subf.f = cf2subfont(cf, f)
	if subf.f == nil {
		if cf.Subfontname == "" {
			if pic != PJW {
				pic = PJW
				goto Again
			}
			return 0, nil
		}
		sfn := cf.Subfontname
		return -1, &sfn
	}
	subf.cf = cf

	// Adjust ascent if subfont has larger ascent than font
	if subf.f.Ascent > f.Ascent && f.Display != nil {
		d := subf.f.Ascent - f.Ascent
		b := subf.f.Bits
		if b != nil {
			b.Draw(b.R, b, b.R.Min.Add(Pt(0, d)))
			b.Draw(Rect(b.R.Min.X, b.R.Max.Y-d, b.R.Max.X, b.R.Max.Y),
				f.Display.Black, b.R.Min)
		}
		for i := 0; i < subf.f.N; i++ {
			t := int(subf.f.Info[i].Top) - d
			if t < 0 {
				t = 0
			}
			subf.f.Info[i].Top = byte(t)
			t = int(subf.f.Info[i].Bottom) - d
			if t < 0 {
				t = 0
			}
			subf.f.Info[i].Bottom = byte(t)
		}
		subf.f.Ascent = f.Ascent
	}

Found2:
	subf.age = f.age

	// Possible overflow here, but works out okay
	idx := int(pic) + cf.Offset - cf.Min
	if idx >= subf.f.N {
		if pic != PJW {
			pic = PJW
			goto Again
		}
		return 0, nil
	}
	fi := &subf.f.Info[idx]
	if fi.Width == 0 {
		if pic != PJW {
			pic = PJW
			goto Again
		}
		return 0, nil
	}

	wid := int(subf.f.Info[idx+1].X) - int(fi.X)
	if f.width < wid || f.width == 0 || f.maxdepth < subf.f.Bits.Depth ||
		(f.Display != nil && f.cacheimage == nil) {
		// Need to resize cache
		if noflush > 0 {
			return -1, nil
		}
		if f.width < wid {
			f.width = wid
		}
		if f.maxdepth < subf.f.Bits.Depth {
			f.maxdepth = subf.f.Bits.Depth
		}
		if !f.fontresize(f.width, f.ncache, f.maxdepth) {
			return -1, nil
		}
	}

	c.value = r
	c.width = fi.Width
	c.x = uint16(h * f.width)
	c.left = fi.Left

	if f.Display == nil {
		return 1, nil
	}

	d := f.Display
	// Send 'l' command to load glyph into cache image.
	// Lock only around bufimage, matching C's _lockdisplay/_unlockdisplay pattern.
	d.mu.Lock()
	b, err := d.bufimage(37)
	if err != nil {
		d.mu.Unlock()
		return 0, nil
	}

	top := int(fi.Top) + (f.Ascent - subf.f.Ascent)
	bottom := int(fi.Bottom) + (f.Ascent - subf.f.Ascent)

	b[0] = 'l'
	bplong(b[1:], uint32(f.cacheimage.id))
	bplong(b[5:], uint32(subf.f.Bits.id))
	bpshort(b[9:], uint16(h))
	bplong(b[11:], uint32(c.x))
	bplong(b[15:], uint32(top))
	bplong(b[19:], uint32(int(c.x)+wid))
	bplong(b[23:], uint32(bottom))
	bplong(b[27:], uint32(fi.X))
	bplong(b[31:], uint32(fi.Top))
	b[35] = byte(fi.Left)
	b[36] = fi.Width
	d.mu.Unlock()

	return 1, nil
}

// fontresize allocates/resizes the font cache image and sends 'i' to devdraw.
// Port of 9front fontresize(). Returns true if cache pointer unchanged.
func (f *Font) fontresize(wid, ncache, depth int) bool {
	if depth <= 0 {
		depth = 1
	}
	if wid <= 0 {
		wid = 1
	}

	d := f.Display
	if d == nil {
		goto Nodisplay
	}

	{
		// AllocImage takes its own lock
		newimg, err := d.AllocImage(Rect(0, 0, ncache*wid, f.Height),
			MakePix(CGrey, depth), false, 0)
		if err != nil {
			return false
		}

		// Send 'i' command: initialize font cache.
		// Lock only around bufimage, matching C's _lockdisplay/_unlockdisplay.
		d.mu.Lock()
		b, err := d.bufimage(1 + 4 + 4 + 1)
		if err != nil {
			d.mu.Unlock()
			newimg.Free()
			return false
		}
		b[0] = 'i'
		bplong(b[1:], uint32(newimg.id))
		bplong(b[5:], uint32(ncache))
		b[9] = byte(f.Ascent)
		d.mu.Unlock()

		// Free takes its own lock
		if f.cacheimage != nil {
			f.cacheimage.Free()
		}
		f.cacheimage = newimg
	}

Nodisplay:
	f.width = wid
	f.maxdepth = depth
	ret := true
	if f.ncache != ncache {
		f.cache = make([]Cacheinfo, ncache)
		f.ncache = ncache
		ret = false
	} else {
		// Zero out existing cache
		for i := range f.cache {
			f.cache[i] = Cacheinfo{}
		}
	}
	return ret
}

// MakePix creates a Pix descriptor for a single channel.
func MakePix(typ int, nbits int) Pix {
	return Pix(typ<<4 | nbits)
}

// Agefont increments the font age and renormalizes if needed.
// This is a direct port of 9front's agefont().
func (f *Font) Agefont() {
	f.age++
	if f.age == 65536 {
		// Renormalize ages
		for i := range f.cache {
			if f.cache[i].age != 0 {
				f.cache[i].age >>= 2
				f.cache[i].age++
			}
		}
		for i := range f.subf {
			if f.subf[i].age != 0 {
				if f.subf[i].age < SUBFAGE && f.subf[i].cf != nil && f.subf[i].cf.Name != "" {
					// clean up
					if f.Display == nil || f.subf[i].f != f.Display.DefaultSubfont {
						f.subf[i].f.Free()
					}
					f.subf[i].cf = nil
					f.subf[i].f = nil
					f.subf[i].age = 0
				} else {
					f.subf[i].age >>= 2
					f.subf[i].age++
				}
			}
		}
		f.age = (65536 >> 2) + 1
	}
}

// Free releases the resources associated with a font.
// This is a port of 9front's freefont().
func (f *Font) Free() {
	if f == nil {
		return
	}
	for i := range f.sub {
		f.sub[i] = nil
	}
	for i := range f.subf {
		s := f.subf[i].f
		if s != nil {
			if f.Display == nil || s != f.Display.DefaultSubfont {
				s.Free()
			}
		}
	}
	if f.cacheimage != nil {
		f.cacheimage.Free()
	}
	f.cache = nil
	f.subf = nil
	f.sub = nil
}

// skipWhitespace skips leading spaces, tabs, and newlines.
func skipWhitespace(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\n' || s[i] == '\t') {
		i++
	}
	return s[i:]
}

// parseInt parses a C-style integer (decimal, 0x hex, 0 octal).
func parseInt(s string) (int, string, bool) {
	if len(s) == 0 {
		return 0, s, false
	}
	// Find the end of the number token
	end := 0
	if end < len(s) && (s[end] == '+' || s[end] == '-') {
		end++
	}
	if end < len(s) && s[end] == '0' && end+1 < len(s) && (s[end+1] == 'x' || s[end+1] == 'X') {
		end += 2
		for end < len(s) && isHexDigit(s[end]) {
			end++
		}
	} else if end < len(s) && s[end] == '0' {
		for end < len(s) && s[end] >= '0' && s[end] <= '7' {
			end++
		}
	} else {
		for end < len(s) && s[end] >= '0' && s[end] <= '9' {
			end++
		}
	}
	if end == 0 {
		return 0, s, false
	}
	n, err := strconv.ParseInt(s[:end], 0, 64)
	if err != nil {
		return 0, s, false
	}
	return int(n), s[end:], true
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// SubfontName returns the subfont file name for a given font name and depth.
func SubfontName(cfname, fname string, maxdepth int) string {
	// Port of 9front subfontname()
	if strings.HasPrefix(cfname, "/") {
		return cfname
	}
	dir := ""
	if idx := strings.LastIndex(fname, "/"); idx >= 0 {
		dir = fname[:idx+1]
	}
	return dir + cfname
}
