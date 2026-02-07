package draw

import (
	"strings"
)

// strtochan converts a channel format string to a Pix value.
// Format strings are like "r8g8b8" or "m8" or "k8".
func strtochan(s string) Pix {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	var pix Pix
	shift := uint(0)

	for len(s) > 0 {
		var t int
		switch s[0] {
		case 'r':
			t = CRed
		case 'g':
			t = CGreen
		case 'b':
			t = CBlue
		case 'k':
			t = CGrey
		case 'a':
			t = CAlpha
		case 'm':
			t = CMap
		case 'x':
			t = CIgnore
		default:
			return 0
		}
		s = s[1:]

		// Parse the depth
		if len(s) == 0 || s[0] < '0' || s[0] > '9' {
			return 0
		}
		d := 0
		for len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
			d = d*10 + int(s[0]-'0')
			s = s[1:]
		}
		if d < 1 || d > 8 {
			return 0
		}

		pix |= Pix((t+1)<<shift | d<<(shift+4))
		shift += 8
		if shift >= 32 {
			break
		}
	}

	return pix
}

// chantostr converts a Pix value to a channel format string.
func chantostr(pix Pix) string {
	if pix == 0 {
		return ""
	}

	names := "rgbkamx"
	var buf strings.Builder

	for shift := uint(0); shift < 32; shift += 8 {
		t := int((pix >> shift) & 0xF)
		d := int((pix >> (shift + 4)) & 0xF)
		if t == 0 || d == 0 {
			break
		}
		t-- // convert back from 1-based
		if t >= len(names) {
			return ""
		}
		buf.WriteByte(names[t])
		if d > 9 {
			buf.WriteByte('1')
			buf.WriteByte('0' + byte(d-10))
		} else {
			buf.WriteByte('0' + byte(d))
		}
	}

	return buf.String()
}

// chantodepth returns the total bits per pixel for a channel format.
func chantodepth(pix Pix) int {
	if pix == 0 {
		return 0
	}

	depth := 0
	for shift := uint(0); shift < 32; shift += 8 {
		d := int((pix >> (shift + 4)) & 0xF)
		if d == 0 {
			break
		}
		depth += d
	}

	return depth
}

// chandepth returns the depth of a specific channel type in a pixel format.
func chandepth(pix Pix, ch int) int {
	for shift := uint(0); shift < 32; shift += 8 {
		t := int((pix >> shift) & 0xF)
		d := int((pix >> (shift + 4)) & 0xF)
		if t == 0 || d == 0 {
			break
		}
		if t-1 == ch {
			return d
		}
	}
	return 0
}

// unit returns the byte width containing all channels for a pixel format.
func unit(pix Pix) int {
	depth := chantodepth(pix)
	return (depth + 7) / 8
}
