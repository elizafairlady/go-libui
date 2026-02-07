package draw

import "testing"

func TestStrtochan(t *testing.T) {
	tests := []struct {
		s    string
		want Pix
	}{
		{"k1", GREY1},
		{"k2", GREY2},
		{"k4", GREY4},
		{"k8", GREY8},
		{"m8", CMAP8},
		{"r8g8b8", RGB24},
		{"r8g8b8a8", RGBA32},
		{"a8r8g8b8", ARGB32},
		{"a8b8g8r8", ABGR32},
		{"x8r8g8b8", XRGB32},
		{"x8b8g8r8", XBGR32},
		{"b8g8r8", BGR24},
		{"r5g6b5", RGB16},
		{"x1r5g5b5", RGB15},
		{"", 0},
		{"z8", 0}, // invalid channel type
	}

	for _, tc := range tests {
		t.Run(tc.s, func(t *testing.T) {
			got := strtochan(tc.s)
			if got != tc.want {
				t.Errorf("strtochan(%q) = 0x%08x, want 0x%08x", tc.s, got, tc.want)
			}
		})
	}
}

func TestChantostr(t *testing.T) {
	tests := []struct {
		pix  Pix
		want string
	}{
		{GREY1, "k1"},
		{GREY2, "k2"},
		{GREY4, "k4"},
		{GREY8, "k8"},
		{CMAP8, "m8"},
		{RGB24, "r8g8b8"},
		{RGBA32, "r8g8b8a8"},
		{ARGB32, "a8r8g8b8"},
		{ABGR32, "a8b8g8r8"},
		{XRGB32, "x8r8g8b8"},
		{BGR24, "b8g8r8"},
		{0, ""},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := chantostr(tc.pix)
			if got != tc.want {
				t.Errorf("chantostr(0x%08x) = %q, want %q", tc.pix, got, tc.want)
			}
		})
	}
}

func TestChantodepth(t *testing.T) {
	tests := []struct {
		pix   Pix
		depth int
	}{
		{GREY1, 1},
		{GREY2, 2},
		{GREY4, 4},
		{GREY8, 8},
		{CMAP8, 8},
		{RGB24, 24},
		{RGBA32, 32},
		{ARGB32, 32},
		{RGB16, 16},
		{RGB15, 16},
		{0, 0},
	}

	for _, tc := range tests {
		got := chantodepth(tc.pix)
		if got != tc.depth {
			t.Errorf("chantodepth(0x%08x) = %d, want %d", tc.pix, got, tc.depth)
		}
	}
}

func TestStrtochanChantostrRoundtrip(t *testing.T) {
	formats := []string{"k1", "k2", "k4", "k8", "m8", "r8g8b8", "r8g8b8a8", "a8r8g8b8", "b8g8r8", "r5g6b5", "x1r5g5b5"}
	for _, s := range formats {
		pix := strtochan(s)
		if pix == 0 {
			t.Errorf("strtochan(%q) = 0", s)
			continue
		}
		got := chantostr(pix)
		if got != s {
			t.Errorf("roundtrip(%q): strtochan=0x%08x chantostr=%q", s, pix, got)
		}
	}
}

func TestBytesPerLine(t *testing.T) {
	tests := []struct {
		name  string
		r     Rectangle
		depth int
		want  int
	}{
		{"1bpp 8wide", Rect(0, 0, 8, 1), 1, 1},
		{"1bpp 9wide", Rect(0, 0, 9, 1), 1, 2},
		{"8bpp 10wide", Rect(0, 0, 10, 1), 8, 10},
		{"24bpp 1wide", Rect(0, 0, 1, 1), 24, 3},
		{"32bpp 1wide", Rect(0, 0, 1, 1), 32, 4},
		{"1bpp offset", Rect(3, 0, 11, 1), 1, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := bytesPerLine(tc.r, tc.depth)
			if got != tc.want {
				t.Errorf("bytesPerLine(%v, %d) = %d, want %d", tc.r, tc.depth, got, tc.want)
			}
		})
	}
}
