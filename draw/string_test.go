package draw

import "testing"

// TestStringWidthNil tests nil font safety.
func TestStringWidthNil(t *testing.T) {
	var f *Font
	if got := f.StringWidth("hello"); got != 0 {
		t.Errorf("nil font StringWidth = %d, want 0", got)
	}
	if got := f.StringNWidth("hello", 3); got != 0 {
		t.Errorf("nil font StringNWidth = %d, want 0", got)
	}
	if got := f.RuneStringWidth([]rune("hello")); got != 0 {
		t.Errorf("nil font RuneStringWidth = %d, want 0", got)
	}
	if got := f.RuneWidth('A'); got != 0 {
		t.Errorf("nil font RuneWidth = %d, want 0", got)
	}
}

// TestStringWidthEstimate tests that a font with width set returns width * nchars.
func TestStringWidthEstimate(t *testing.T) {
	f := &Font{
		Height: 16,
		width:  8,
		cache:  make([]Cacheinfo, 0),
	}
	// With empty cache, RuneWidth falls back to f.width
	if got := f.RuneWidth('A'); got != 8 {
		t.Errorf("RuneWidth('A') = %d, want 8", got)
	}
	if got := f.StringWidth("ABC"); got != 24 {
		t.Errorf("StringWidth(\"ABC\") = %d, want 24", got)
	}
}

// TestStringNWidthLimit tests the character limit.
func TestStringNWidthLimit(t *testing.T) {
	f := &Font{
		Height: 16,
		width:  10,
		cache:  make([]Cacheinfo, 0),
	}
	// 2 of 5 chars
	if got := f.StringNWidth("Hello", 2); got != 20 {
		t.Errorf("StringNWidth(\"Hello\", 2) = %d, want 20", got)
	}
	// More than available
	if got := f.StringNWidth("Hi", 10); got != 20 {
		t.Errorf("StringNWidth(\"Hi\", 10) = %d, want 20", got)
	}
}

// TestCacheLookup tests the font cache lookup.
func TestCacheLookup(t *testing.T) {
	f := &Font{
		cache: []Cacheinfo{
			{value: 'A', width: 8, left: 0, age: 1},
			{value: 'B', width: 9, left: 1, age: 1},
		},
	}
	ci, ok := f.cacheLookup('A')
	if !ok || ci == nil {
		t.Fatal("cache lookup 'A' failed")
	}
	if ci.width != 8 {
		t.Errorf("'A' width = %d, want 8", ci.width)
	}

	ci, ok = f.cacheLookup('B')
	if !ok || ci == nil {
		t.Fatal("cache lookup 'B' failed")
	}
	if ci.width != 9 {
		t.Errorf("'B' width = %d, want 9", ci.width)
	}

	ci, ok = f.cacheLookup('Z')
	if ok {
		t.Error("cache lookup 'Z' should fail")
	}
}

// TestCacheLookupNil tests nil font cache.
func TestCacheLookupNil(t *testing.T) {
	var f *Font
	ci, ok := f.cacheLookup('A')
	if ok || ci != nil {
		t.Error("nil font cacheLookup should return nil, false")
	}
}

// TestRuneStringWidth tests rune string width with cached chars.
func TestRuneStringWidth(t *testing.T) {
	// With uninitialized cache (ncache=0), falls back to f.width * n
	f := &Font{
		Height: 16,
		width:  6,
	}
	runes := []rune{'H', 'i'}
	if got := f.RuneStringWidth(runes); got != 12 {
		t.Errorf("RuneStringWidth = %d, want 12", got)
	}
}

// TestBytesWidth tests byte slice width.
func TestBytesWidth(t *testing.T) {
	f := &Font{
		Height: 16,
		width:  7,
		cache:  make([]Cacheinfo, 0),
	}
	if got := f.BytesWidth([]byte("abc")); got != 21 {
		t.Errorf("BytesWidth = %d, want 21", got)
	}
}

// TestStringNilDst tests string drawing with nil destination.
func TestStringNilDst(t *testing.T) {
	var dst *Image
	p := Pt(10, 20)
	f := &Font{Height: 16}
	got := dst.String(p, nil, ZP, f, "hello")
	if !got.Eq(p) {
		t.Errorf("nil dst String = %v, want %v", got, p)
	}
}
