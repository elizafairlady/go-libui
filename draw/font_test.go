package draw

import "testing"

// TestFontCacheConstants verifies cache constants match draw.h.
func TestFontCacheConstants(t *testing.T) {
	if NFCACHE != 64 {
		t.Errorf("NFCACHE = %d, want 64", NFCACHE)
	}
	if NFLOOK != 5 {
		t.Errorf("NFLOOK = %d, want 5", NFLOOK)
	}
	if NFSUBF != 2 {
		t.Errorf("NFSUBF = %d, want 2", NFSUBF)
	}
	if MAXFCACHE != 1029 {
		t.Errorf("MAXFCACHE = %d, want 1029", MAXFCACHE)
	}
	if MAXSUBF != 50 {
		t.Errorf("MAXSUBF = %d, want 50", MAXSUBF)
	}
	if DSUBF != 4 {
		t.Errorf("DSUBF = %d, want 4", DSUBF)
	}
	if SUBFAGE != 10000 {
		t.Errorf("SUBFAGE = %d, want 10000", SUBFAGE)
	}
}

// TestBuildFontSimple tests parsing a simple font file.
func TestBuildFontSimple(t *testing.T) {
	fontdata := "16 12\n0x0000 0x007F /lib/font/bit/lucm/latin1.9\n"
	d := &Display{}
	f, err := d.BuildFont([]byte(fontdata), "/lib/font/bit/lucm/euro.9.font")
	if err != nil {
		t.Fatal(err)
	}
	if f.Height != 16 {
		t.Errorf("Height = %d, want 16", f.Height)
	}
	if f.Ascent != 12 {
		t.Errorf("Ascent = %d, want 12", f.Ascent)
	}
	if f.nsub != 1 {
		t.Errorf("nsub = %d, want 1", f.nsub)
	}
	if f.sub[0].Min != 0 {
		t.Errorf("sub[0].Min = %d, want 0", f.sub[0].Min)
	}
	if f.sub[0].Max != 0x7F {
		t.Errorf("sub[0].Max = %#x, want 0x7F", f.sub[0].Max)
	}
	if f.sub[0].Offset != 0 {
		t.Errorf("sub[0].Offset = %d, want 0", f.sub[0].Offset)
	}
	if f.sub[0].Name != "/lib/font/bit/lucm/latin1.9" {
		t.Errorf("sub[0].Name = %q, want /lib/font/bit/lucm/latin1.9", f.sub[0].Name)
	}
	// Check cache was initialized
	if f.ncache != NFCACHE+NFLOOK {
		t.Errorf("ncache = %d, want %d", f.ncache, NFCACHE+NFLOOK)
	}
	if f.nsubf != NFSUBF {
		t.Errorf("nsubf = %d, want %d", f.nsubf, NFSUBF)
	}
	if f.age != 1 {
		t.Errorf("age = %d, want 1", f.age)
	}
}

// TestBuildFontWithOffset tests parsing a font file with an offset field.
func TestBuildFontWithOffset(t *testing.T) {
	fontdata := "16 12\n0x0000 0x00FF 32 /lib/font/bit/lucm/latin1.9\n"
	d := &Display{}
	f, err := d.BuildFont([]byte(fontdata), "test.font")
	if err != nil {
		t.Fatal(err)
	}
	if f.sub[0].Offset != 32 {
		t.Errorf("sub[0].Offset = %d, want 32", f.sub[0].Offset)
	}
	if f.sub[0].Name != "/lib/font/bit/lucm/latin1.9" {
		t.Errorf("sub[0].Name = %q", f.sub[0].Name)
	}
}

// TestBuildFontMultipleRanges tests parsing multiple subfont ranges.
func TestBuildFontMultipleRanges(t *testing.T) {
	fontdata := "20 14\n0x0000 0x007F /lib/font/bit/lucm/latin1.20\n0x0080 0x00FF /lib/font/bit/lucm/latineur.20\n0x0100 0x017E /lib/font/bit/lucm/latin-ext.20\n"
	d := &Display{}
	f, err := d.BuildFont([]byte(fontdata), "test.font")
	if err != nil {
		t.Fatal(err)
	}
	if f.nsub != 3 {
		t.Fatalf("nsub = %d, want 3", f.nsub)
	}
	if f.sub[1].Min != 0x80 || f.sub[1].Max != 0xFF {
		t.Errorf("sub[1] range = [%#x, %#x], want [0x80, 0xFF]", f.sub[1].Min, f.sub[1].Max)
	}
	if f.sub[2].Min != 0x100 || f.sub[2].Max != 0x17E {
		t.Errorf("sub[2] range = [%#x, %#x], want [0x100, 0x17E]", f.sub[2].Min, f.sub[2].Max)
	}
}

// TestBuildFontBadHeader tests error on bad header.
func TestBuildFontBadHeader(t *testing.T) {
	_, err := (&Display{}).BuildFont([]byte(""), "test.font")
	if err == nil {
		t.Error("expected error for empty font data")
	}
	_, err = (&Display{}).BuildFont([]byte("0 0\n"), "test.font")
	if err == nil {
		t.Error("expected error for zero height/ascent")
	}
}

// TestParseInt tests the C-style integer parser.
func TestParseInt(t *testing.T) {
	tests := []struct {
		input string
		val   int
		rest  string
		ok    bool
	}{
		{"123 rest", 123, " rest", true},
		{"0x1F rest", 0x1F, " rest", true},
		{"0777 rest", 0777, " rest", true},
		{"0 rest", 0, " rest", true},
		{"abc", 0, "abc", false},
		{"", 0, "", false},
	}
	for _, tt := range tests {
		v, r, ok := parseInt(tt.input)
		if ok != tt.ok || v != tt.val || r != tt.rest {
			t.Errorf("parseInt(%q) = (%d, %q, %v), want (%d, %q, %v)",
				tt.input, v, r, ok, tt.val, tt.rest, tt.ok)
		}
	}
}

// TestAgefont tests the age renormalization.
func TestAgefont(t *testing.T) {
	f := &Font{
		age:    65535,
		ncache: 4,
		nsubf:  2,
		cache:  make([]Cacheinfo, 4),
		subf:   make([]Cachesubf, 2),
	}
	f.cache[0].age = 1000
	f.cache[1].age = 0
	f.Agefont()
	// After age wraps at 65536, it should renormalize
	if f.age != (65536>>2)+1 {
		t.Errorf("age after renormalize = %d, want %d", f.age, (65536>>2)+1)
	}
	if f.cache[0].age != (1000>>2)+1 {
		t.Errorf("cache[0].age = %d, want %d", f.cache[0].age, (1000>>2)+1)
	}
	if f.cache[1].age != 0 {
		t.Errorf("cache[1].age = %d, want 0", f.cache[1].age)
	}
}

// TestSubfontName tests subfont name resolution.
func TestSubfontName(t *testing.T) {
	// Absolute path stays as is
	if got := SubfontName("/lib/font/x", "/lib/font/f.font", 8); got != "/lib/font/x" {
		t.Errorf("got %q", got)
	}
	// Relative path gets directory from font name
	if got := SubfontName("latin1.16", "/lib/font/bit/lucm/euro.font", 8); got != "/lib/font/bit/lucm/latin1.16" {
		t.Errorf("got %q", got)
	}
	// No directory in font name
	if got := SubfontName("latin1.16", "euro.font", 8); got != "latin1.16" {
		t.Errorf("got %q", got)
	}
}
