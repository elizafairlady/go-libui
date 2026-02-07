package draw

// Cmap2rgb converts a CMAP8 colormap index to packed RGB (r<<16 | g<<8 | b).
// This is a direct port of 9front's cmap2rgb().
func Cmap2rgb(c int) int {
	r := c >> 6
	v := (c >> 4) & 3
	j := (c - v + r) & 15
	g := j >> 2
	b := j & 3

	den := r
	if g > den {
		den = g
	}
	if b > den {
		den = b
	}

	if den == 0 {
		v *= 17
		return (v << 16) | (v << 8) | v
	}
	num := 17 * (4*den + v)
	return ((r * num / den) << 16) | ((g * num / den) << 8) | (b * num / den)
}

// Cmap2rgba converts a CMAP8 colormap index to packed RGBA.
// This is a direct port of 9front's cmap2rgba().
func Cmap2rgba(c int) int {
	return (Cmap2rgb(c) << 8) | 0xFF
}

// Rgb2cmap finds the closest CMAP8 colormap index for an RGB triple.
// This is a direct port of 9front's rgb2cmap(), which uses brute force
// nearest-neighbor search in RGB space.
func Rgb2cmap(cr, cg, cb int) int {
	best := 0
	bestsq := 0x7FFFFFFF

	for i := 0; i < 256; i++ {
		rgb := Cmap2rgb(i)
		r := (rgb >> 16) & 0xFF
		g := (rgb >> 8) & 0xFF
		b := rgb & 0xFF

		sq := (r-cr)*(r-cr) + (g-cg)*(g-cg) + (b-cb)*(b-cb)
		if sq < bestsq {
			bestsq = sq
			best = i
		}
	}
	return best
}
