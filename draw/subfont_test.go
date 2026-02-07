package draw

import (
	"bytes"
	"fmt"
	"testing"
)

// TestAllocSubfont tests basic subfont allocation.
func TestAllocSubfont(t *testing.T) {
	info := []Fontchar{
		{X: 0, Top: 1, Bottom: 12, Left: 0, Width: 8},
		{X: 8, Top: 1, Bottom: 12, Left: 0, Width: 8},
		{X: 16, Top: 1, Bottom: 12, Left: 0, Width: 0}, // n+1 sentinel
	}
	sf := AllocSubfont("test", 2, 16, 12, info, nil)
	if sf == nil {
		t.Fatal("AllocSubfont returned nil")
	}
	if sf.N != 2 {
		t.Errorf("N = %d, want 2", sf.N)
	}
	if sf.Height != 16 {
		t.Errorf("Height = %d, want 16", sf.Height)
	}
	if sf.Ascent != 12 {
		t.Errorf("Ascent = %d, want 12", sf.Ascent)
	}
	if sf.ref != 1 {
		t.Errorf("ref = %d, want 1", sf.ref)
	}
	if sf.Name != "test" {
		t.Errorf("Name = %q, want %q", sf.Name, "test")
	}
}

// TestAllocSubfontZeroHeight tests that height=0 returns nil.
func TestAllocSubfontZeroHeight(t *testing.T) {
	sf := AllocSubfont("test", 2, 0, 0, nil, nil)
	if sf != nil {
		t.Error("AllocSubfont with height=0 should return nil")
	}
}

// TestSubfontFreeRefcount tests reference counting on Free.
func TestSubfontFreeRefcount(t *testing.T) {
	info := make([]Fontchar, 3)
	sf := AllocSubfont("", 2, 16, 12, info, nil)
	sf.ref = 3
	sf.Free()
	if sf.ref != 2 {
		t.Errorf("ref after first Free = %d, want 2", sf.ref)
	}
	sf.Free()
	if sf.ref != 1 {
		t.Errorf("ref after second Free = %d, want 1", sf.ref)
	}
	sf.Free() // ref goes to 0, actually frees
	if sf.Info != nil {
		t.Error("Info should be nil after final Free")
	}
}

// TestSubfontFreeNil tests that Free(nil) is safe.
func TestSubfontFreeNil(t *testing.T) {
	var sf *Subfont
	sf.Free() // should not panic
}

// TestSubfontCache tests the global subfont cache.
func TestSubfontCache(t *testing.T) {
	info := make([]Fontchar, 3)
	sf := AllocSubfont("cacheme", 2, 16, 12, info, nil)

	// Should find it
	got := LookupSubfont(nil, "cacheme")
	if got != sf {
		t.Error("LookupSubfont failed to find installed subfont")
	}
	if got.ref != 2 {
		t.Errorf("ref after lookup = %d, want 2", got.ref)
	}

	// Different name should not find
	got = LookupSubfont(nil, "other")
	if got != nil {
		t.Error("LookupSubfont found wrong name")
	}

	// Uninstall
	UninstallSubfont(sf)
	got = LookupSubfont(nil, "cacheme")
	if got != nil {
		t.Error("LookupSubfont should return nil after uninstall")
	}
}

// TestUnpackInfo tests unpacking fontchar data.
func TestUnpackInfo(t *testing.T) {
	// Pack 2 chars + sentinel = 3 entries * 6 bytes
	p := []byte{
		// Entry 0: X=0, Top=1, Bottom=12, Left=0, Width=8
		0, 0, 1, 12, 0, 8,
		// Entry 1: X=256 (0x100), Top=2, Bottom=14, Left=-1, Width=7
		0, 1, 2, 14, 0xFF, 7,
		// Entry 2 (sentinel): X=512 (0x200), Top=0, Bottom=0, Left=0, Width=0
		0, 2, 0, 0, 0, 0,
	}

	fc := unpackInfo(p, 2)
	if len(fc) != 3 {
		t.Fatalf("len(fc) = %d, want 3", len(fc))
	}

	if fc[0].X != 0 || fc[0].Top != 1 || fc[0].Bottom != 12 || fc[0].Width != 8 {
		t.Errorf("fc[0] = %+v", fc[0])
	}
	if fc[1].X != 256 || fc[1].Top != 2 || fc[1].Bottom != 14 || fc[1].Left != -1 || fc[1].Width != 7 {
		t.Errorf("fc[1] = %+v", fc[1])
	}
	if fc[2].X != 512 {
		t.Errorf("fc[2].X = %d, want 512", fc[2].X)
	}
}

// TestPackUnpackRoundtrip tests pack/unpack roundtrip.
func TestPackUnpackRoundtrip(t *testing.T) {
	fc := []Fontchar{
		{X: 0, Top: 1, Bottom: 16, Left: 0, Width: 10},
		{X: 10, Top: 0, Bottom: 15, Left: -2, Width: 8},
		{X: 18, Top: 0, Bottom: 0, Left: 0, Width: 0}, // sentinel
	}
	p := packInfo(fc, 2)
	got := unpackInfo(p, 2)

	for i := range fc {
		if got[i].X != fc[i].X || got[i].Top != fc[i].Top ||
			got[i].Bottom != fc[i].Bottom || got[i].Left != fc[i].Left ||
			got[i].Width != fc[i].Width {
			t.Errorf("entry %d: got %+v, want %+v", i, got[i], fc[i])
		}
	}
}

// TestWriteSubfont tests writing subfont data.
func TestWriteSubfont(t *testing.T) {
	info := []Fontchar{
		{X: 0, Top: 1, Bottom: 12, Left: 0, Width: 8},
		{X: 8, Top: 0, Bottom: 0, Left: 0, Width: 0},
	}
	sf := &Subfont{N: 1, Height: 16, Ascent: 12, Info: info}

	var buf bytes.Buffer
	err := WriteSubfont(&buf, sf)
	if err != nil {
		t.Fatal(err)
	}

	// Header should be 3*12 = 36 bytes, data should be 2*6 = 12 bytes
	if buf.Len() != 36+12 {
		t.Errorf("written %d bytes, want %d", buf.Len(), 36+12)
	}

	// Verify header
	hdr := buf.Bytes()[:36]
	n := atoi12(hdr[0:12])
	height := atoi12(hdr[12:24])
	ascent := atoi12(hdr[24:36])
	if n != 1 || height != 16 || ascent != 12 {
		t.Errorf("header: n=%d height=%d ascent=%d, want 1 16 12", n, height, ascent)
	}
}

// TestCharWidth tests character width lookup.
func TestCharWidth(t *testing.T) {
	info := []Fontchar{
		{Width: 8},
		{Width: 6},
		{Width: 0},
	}
	sf := &Subfont{N: 2, Info: info}

	if got := sf.CharWidth(0); got != 8 {
		t.Errorf("CharWidth(0) = %d, want 8", got)
	}
	if got := sf.CharWidth(1); got != 6 {
		t.Errorf("CharWidth(1) = %d, want 6", got)
	}
	if got := sf.CharWidth(-1); got != 0 {
		t.Errorf("CharWidth(-1) = %d, want 0", got)
	}
	if got := sf.CharWidth(5); got != 0 {
		t.Errorf("CharWidth(5) = %d, want 0", got)
	}
}

// TestCharInfo tests character info lookup.
func TestCharInfo(t *testing.T) {
	info := []Fontchar{
		{X: 0, Top: 1, Bottom: 12, Left: 0, Width: 8},
		{X: 8, Top: 2, Bottom: 14, Left: -1, Width: 7},
		{X: 15, Top: 0, Bottom: 0, Left: 0, Width: 0},
	}
	sf := &Subfont{N: 2, Info: info}

	ci := sf.CharInfo(1)
	if ci == nil {
		t.Fatal("CharInfo(1) returned nil")
	}
	if ci.X != 8 || ci.Width != 7 {
		t.Errorf("CharInfo(1) = {X:%d Width:%d}, want {X:8 Width:7}", ci.X, ci.Width)
	}
	if sf.CharInfo(-1) != nil {
		t.Error("CharInfo(-1) should be nil")
	}
}

// TestAtoi12 tests 12-char decimal field parsing.
func TestAtoi12(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{fmt.Sprintf("%12d", 256), 256},
		{fmt.Sprintf("%12d", 0), 0},
		{fmt.Sprintf("%12d", 16), 16},
		{fmt.Sprintf("%12d", 12), 12},
	}
	for _, tt := range tests {
		got := atoi12([]byte(tt.input))
		if got != tt.want {
			t.Errorf("atoi12(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
