package draw

import "testing"

// TestCmap2rgb tests known colormap values against 9front's algorithm.
func TestCmap2rgb(t *testing.T) {
	tests := []struct {
		index int
		rgb   int // expected packed RGB
	}{
		// Index 0: black (den=0, v=0)
		{0, 0x000000},
		// Index 255: white (r=g=b=3, v=3)
		{255, 0xFFFFFF},
		// Index 17: grey 17 (den=0, v=1, c-v=16→j=0)
		{17, 0x111111},
		// Index 34: grey 34 (den=0, v=2)
		{34, 0x222222},
		// Index 51: grey 51 (den=0, v=3)
		{51, 0x333333},
	}
	for _, tt := range tests {
		got := Cmap2rgb(tt.index)
		if got != tt.rgb {
			t.Errorf("Cmap2rgb(%d) = %#06x, want %#06x", tt.index, got, tt.rgb)
		}
	}
}

// TestCmap2rgba tests the RGBA conversion wraps RGB correctly.
func TestCmap2rgba(t *testing.T) {
	for i := 0; i < 256; i++ {
		rgb := Cmap2rgb(i)
		rgba := Cmap2rgba(i)
		want := (rgb << 8) | 0xFF
		if rgba != want {
			t.Errorf("Cmap2rgba(%d) = %#08x, want %#08x", i, rgba, want)
		}
	}
}

// TestRgb2cmapRoundtrip verifies that rgb2cmap(cmap2rgb(c)) == c for all c.
// This is the key invariant from the C source comment.
func TestRgb2cmapRoundtrip(t *testing.T) {
	for c := 0; c < 256; c++ {
		rgb := Cmap2rgb(c)
		r := (rgb >> 16) & 0xFF
		g := (rgb >> 8) & 0xFF
		b := rgb & 0xFF
		got := Rgb2cmap(r, g, b)
		if got != c {
			t.Errorf("Rgb2cmap(Cmap2rgb(%d)) = %d, want %d (rgb=%#06x)", c, got, c, rgb)
		}
	}
}

// TestRgb2cmapKnownColors tests a few well-known colors.
func TestRgb2cmapKnownColors(t *testing.T) {
	// Black should map to index 0
	if got := Rgb2cmap(0, 0, 0); got != 0 {
		t.Errorf("Rgb2cmap(0,0,0) = %d, want 0", got)
	}
	// White should map to index 255
	if got := Rgb2cmap(255, 255, 255); got != 255 {
		t.Errorf("Rgb2cmap(255,255,255) = %d, want 255", got)
	}
}

// TestCmap2rgbGreyRamp tests the den==0 grey values.
// Grey indices in the first quadrant (r=0) are 0, 17, 34, 51.
func TestCmap2rgbGreyRamp(t *testing.T) {
	greys := []struct {
		idx  int
		grey int // expected grey value v*17
	}{
		{0, 0},
		{17, 17},
		{34, 34},
		{51, 51},
	}
	for _, tt := range greys {
		rgb := Cmap2rgb(tt.idx)
		r := (rgb >> 16) & 0xFF
		g := (rgb >> 8) & 0xFF
		b := rgb & 0xFF
		if r != tt.grey || g != tt.grey || b != tt.grey {
			t.Errorf("Cmap2rgb(%d) = (%d,%d,%d), want grey(%d)", tt.idx, r, g, b, tt.grey)
		}
	}
}
