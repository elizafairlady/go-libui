package draw

import (
	"strings"
)

// Channel descriptor encoding matches 9front's libdraw exactly.
// Each channel occupies one byte: (type<<4)|nbits
// Channels are packed from MSB to LSB in order of the string representation.
//
// TYPE(v) = (v>>4)&0xF
// NBITS(v) = v&0xF
// __DC(t,n) = (t<<4)|n

var channames = "rgbkamx"

// strtochan converts a channel format string to a Pix value.
// Format strings are like "r8g8b8" or "m8" or "k8".
func strtochan(s string) Pix {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	var c Pix
	d := 0

	for len(s) > 0 && s[0] != ' ' && s[0] != '\t' && s[0] != '\n' {
		idx := strings.IndexByte(channames, s[0])
		if idx < 0 {
			return 0
		}
		t := idx
		s = s[1:]
		if len(s) == 0 || s[0] < '0' || s[0] > '9' {
			return 0
		}
		n := int(s[0] - '0')
		s = s[1:]
		d += n
		c = (c << 8) | Pix((t<<4)|n)
	}
	if d == 0 || (d > 8 && d%8 != 0) || (d < 8 && 8%d != 0) {
		return 0
	}
	return c
}

// chantostr converts a Pix value to a channel format string.
func chantostr(cc Pix) string {
	if chantodepth(cc) == 0 {
		return ""
	}

	// Reverse the channel descriptor
	var rc Pix
	for c := cc; c != 0; c >>= 8 {
		rc <<= 8
		rc |= c & 0xFF
	}

	var buf []byte
	for c := rc; c != 0; c >>= 8 {
		t := (c >> 4) & 0xF
		n := c & 0xF
		if int(t) >= len(channames) {
			return ""
		}
		buf = append(buf, channames[t])
		buf = append(buf, '0'+byte(n))
	}
	return string(buf)
}

// chantodepth returns the total bits per pixel for a channel format.
func chantodepth(c Pix) int {
	n := 0
	for c != 0 {
		t := (c >> 4) & 0xF
		nb := c & 0xF
		if t >= NChan || nb > 8 || nb <= 0 {
			return 0
		}
		n += int(nb)
		c >>= 8
	}
	if n == 0 || (n > 8 && n%8 != 0) || (n < 8 && 8%n != 0) {
		return 0
	}
	return n
}
