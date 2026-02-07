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
