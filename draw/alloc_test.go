package draw

import "testing"

func TestBadrect(t *testing.T) {
	tests := []struct {
		name string
		r    Rectangle
		bad  bool
	}{
		{"zero rect", Rect(0, 0, 0, 0), true},
		{"negative width", Rpt(Pt(10, 0), Pt(5, 10)), true},  // use Rpt to avoid canonicalization
		{"negative height", Rpt(Pt(0, 10), Pt(10, 5)), true}, // use Rpt to avoid canonicalization
		{"normal rect", Rect(0, 0, 100, 100), false},
		{"1x1 rect", Rect(0, 0, 1, 1), false},
		{"huge rect", Rect(0, 0, 0x10000, 0x10000), true},        // 0x10000^2 = 0x100000000 > 0x10000000
		{"large but ok rect", Rect(0, 0, 0x1000, 0x1000), false}, // 0x1000^2 = 0x1000000 < 0x10000000
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Badrect(tc.r)
			if result != tc.bad {
				t.Errorf("Badrect(%v) = %v, want %v", tc.r, result, tc.bad)
			}
		})
	}
}

func TestBplong(t *testing.T) {
	b := make([]byte, 4)
	bplong(b, 0x12345678)
	if b[0] != 0x78 || b[1] != 0x56 || b[2] != 0x34 || b[3] != 0x12 {
		t.Errorf("bplong(0x12345678) = %v, want [0x78 0x56 0x34 0x12]", b)
	}
}

func TestGlong(t *testing.T) {
	b := []byte{0x78, 0x56, 0x34, 0x12}
	result := glong(b)
	if result != 0x12345678 {
		t.Errorf("glong(%v) = 0x%x, want 0x12345678", b, result)
	}
}

func TestBpshort(t *testing.T) {
	b := make([]byte, 2)
	bpshort(b, 0x1234)
	if b[0] != 0x34 || b[1] != 0x12 {
		t.Errorf("bpshort(0x1234) = %v, want [0x34 0x12]", b)
	}
}

func TestGshort(t *testing.T) {
	b := []byte{0x34, 0x12}
	result := gshort(b)
	if result != 0x1234 {
		t.Errorf("gshort(%v) = 0x%x, want 0x1234", b, result)
	}
}

func TestBplongGlongRoundtrip(t *testing.T) {
	values := []uint32{0, 1, 0x7FFFFFFF, 0x80000000, 0xFFFFFFFF, 0x12345678}
	for _, v := range values {
		b := make([]byte, 4)
		bplong(b, v)
		result := glong(b)
		if result != v {
			t.Errorf("roundtrip(%x): got %x", v, result)
		}
	}
}
