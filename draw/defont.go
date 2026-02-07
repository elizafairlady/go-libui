package draw

// getdefont returns the built-in default font.
func (d *Display) getdefont() *Subfont {
	// Create a minimal built-in font for when no font files are available
	// This is a simple fixed-width bitmap font

	// Create a small 8x16 character set for ASCII
	charWidth := 8
	charHeight := 16
	nchars := 128

	// Allocate an image for the font glyphs
	width := charWidth * nchars
	img, err := d.AllocImage(Rect(0, 0, width, charHeight), GREY1, false, DWhite)
	if err != nil {
		return nil
	}

	// Build fontchar info
	info := make([]Fontchar, nchars+1)
	for i := 0; i <= nchars; i++ {
		info[i] = Fontchar{
			X:      i * charWidth,
			Top:    0,
			Bottom: byte(charHeight),
			Left:   0,
			Width:  byte(charWidth),
		}
	}

	return &Subfont{
		Name:   "*default*",
		N:      nchars,
		Height: charHeight,
		Ascent: charHeight - 4,
		Info:   info,
		Bits:   img,
		ref:    1,
	}
}

// Default font data - a minimal bitmap font
// This would normally contain actual glyph bitmaps
var defaultFontData = []byte{
	// Font header: height, ascent
	16, 12,
	// Character data would go here
	// For now, using a placeholder
}
