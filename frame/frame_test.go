package frame

import (
	"testing"
)

// Tests for frame internals that don't require a display connection.
// The full frame requires /dev/draw, but we can test box management,
// utility functions, and parsing logic without one.

func TestFrboxNRune(t *testing.T) {
	// Text box
	b := frbox{nrune: 5}
	if b.nRune() != 5 {
		t.Errorf("text box nRune = %d, want 5", b.nRune())
	}

	// Break box (tab)
	b = frbox{nrune: -1, bc: '\t'}
	if b.nRune() != 1 {
		t.Errorf("break box nRune = %d, want 1", b.nRune())
	}

	// Break box (newline)
	b = frbox{nrune: -1, bc: '\n'}
	if b.nRune() != 1 {
		t.Errorf("break box nRune = %d, want 1", b.nRune())
	}
}

func TestFrboxNByte(t *testing.T) {
	b := frbox{nrune: 3, ptr: []byte("abc")}
	if b.nbyte() != 3 {
		t.Errorf("nbyte = %d, want 3", b.nbyte())
	}
	// Multi-byte runes
	b = frbox{nrune: 2, ptr: []byte("日本")}
	if b.nbyte() != 6 {
		t.Errorf("nbyte for CJK = %d, want 6", b.nbyte())
	}
}

func TestRuneIndex(t *testing.T) {
	tests := []struct {
		text string
		n    int
		want int
	}{
		{"hello", 0, 0},
		{"hello", 3, 3},
		{"hello", 5, 5},
		{"日本語", 0, 0},
		{"日本語", 1, 3},
		{"日本語", 2, 6},
		{"日本語", 3, 9},
		{"aé日", 0, 0},
		{"aé日", 1, 1},
		{"aé日", 2, 3},
		{"aé日", 3, 6},
	}
	for _, tt := range tests {
		got := runeIndex([]byte(tt.text), tt.n)
		if got != tt.want {
			t.Errorf("runeIndex(%q, %d) = %d, want %d", tt.text, tt.n, got, tt.want)
		}
	}
}

func TestGrowbox(t *testing.T) {
	f := &Frame{}
	f.growbox(10)
	if f.nalloc != 10 {
		t.Errorf("nalloc = %d, want 10", f.nalloc)
	}
	if len(f.box) != 10 {
		t.Errorf("len(box) = %d, want 10", len(f.box))
	}
	// Grow again
	f.growbox(5)
	if f.nalloc != 15 {
		t.Errorf("nalloc = %d, want 15", f.nalloc)
	}
}

func TestAddboxClosebox(t *testing.T) {
	f := &Frame{}
	f.growbox(10)
	f.nbox = 3
	f.box[0] = frbox{nrune: -1, bc: 'a'}
	f.box[1] = frbox{nrune: -1, bc: 'b'}
	f.box[2] = frbox{nrune: -1, bc: 'c'}

	// Add 2 boxes after position 1
	f.addbox(1, 2)
	if f.nbox != 5 {
		t.Errorf("nbox = %d, want 5", f.nbox)
	}
	// box[0] = a, box[1..2] = empty, box[3] = b, box[4] = c
	if f.box[0].bc != 'a' {
		t.Errorf("box[0].bc = %c, want a", f.box[0].bc)
	}
	if f.box[3].bc != 'b' {
		t.Errorf("box[3].bc = %c, want b", f.box[3].bc)
	}
	if f.box[4].bc != 'c' {
		t.Errorf("box[4].bc = %c, want c", f.box[4].bc)
	}

	// Close boxes 1..2
	f.closebox(1, 2)
	if f.nbox != 3 {
		t.Errorf("nbox after close = %d, want 3", f.nbox)
	}
	if f.box[0].bc != 'a' || f.box[1].bc != 'b' || f.box[2].bc != 'c' {
		t.Errorf("boxes after close: %c %c %c, want a b c",
			f.box[0].bc, f.box[1].bc, f.box[2].bc)
	}
}

func TestStrlen(t *testing.T) {
	f := &Frame{}
	f.growbox(5)
	f.nbox = 3
	f.box[0] = frbox{nrune: 3, ptr: []byte("abc")}
	f.box[1] = frbox{nrune: -1, bc: '\n'}
	f.box[2] = frbox{nrune: 5, ptr: []byte("hello")}

	if n := f.strlen(0); n != 9 {
		t.Errorf("strlen(0) = %d, want 9", n)
	}
	if n := f.strlen(1); n != 6 {
		t.Errorf("strlen(1) = %d, want 6", n)
	}
	if n := f.strlen(2); n != 5 {
		t.Errorf("strlen(2) = %d, want 5", n)
	}
}

func TestNBytes(t *testing.T) {
	tests := []struct {
		s    string
		nr   int
		want int
	}{
		{"hello", 3, 3},
		{"hello", 5, 5},
		{"日本語", 1, 3},
		{"日本語", 2, 6},
		{"aé日", 2, 3},
	}
	for _, tt := range tests {
		got := nbytes(tt.s, tt.nr)
		if got != tt.want {
			t.Errorf("nbytes(%q, %d) = %d, want %d", tt.s, tt.nr, got, tt.want)
		}
	}
}

func TestRegion(t *testing.T) {
	if region(1, 2) != -1 {
		t.Errorf("region(1,2) = %d, want -1", region(1, 2))
	}
	if region(2, 2) != 0 {
		t.Errorf("region(2,2) = %d, want 0", region(2, 2))
	}
	if region(3, 2) != 1 {
		t.Errorf("region(3,2) = %d, want 1", region(3, 2))
	}
}

func TestBoolToInt(t *testing.T) {
	if boolToInt(true) != 1 {
		t.Error("boolToInt(true) != 1")
	}
	if boolToInt(false) != 0 {
		t.Error("boolToInt(false) != 0")
	}
}

func TestConstants(t *testing.T) {
	if NCol != 5 {
		t.Errorf("NCol = %d, want 5", NCol)
	}
	if FRTICKW != 3 {
		t.Errorf("FRTICKW = %d, want 3", FRTICKW)
	}
	if ColBack != 0 || ColHigh != 1 || ColBord != 2 || ColText != 3 || ColHText != 4 {
		t.Error("color constants are wrong")
	}
}
