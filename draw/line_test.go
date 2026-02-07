package draw

import "testing"

func TestAddcoordSmallDelta(t *testing.T) {
	// Small deltas (fits in 7 signed bits: -64 to 63) should be 1 byte
	buf := make([]byte, 3)

	// Delta = 0
	n := addcoord(buf, 0, 0)
	if n != 1 || buf[0] != 0 {
		t.Errorf("addcoord(0,0) = %d bytes [%x], want 1 byte [00]", n, buf[:n])
	}

	// Delta = 1
	n = addcoord(buf, 0, 1)
	if n != 1 || buf[0] != 1 {
		t.Errorf("addcoord(0,1) = %d bytes [%x], want 1 byte [01]", n, buf[:n])
	}

	// Delta = -1
	n = addcoord(buf, 1, 0)
	if n != 1 || buf[0] != 0x7F { // -1 & 0x7F = 0x7F
		t.Errorf("addcoord(1,0) = %d bytes [%x], want 1 byte [7f]", n, buf[:n])
	}

	// Delta = 63 (max 7-bit signed)
	n = addcoord(buf, 0, 63)
	if n != 1 || buf[0] != 63 {
		t.Errorf("addcoord(0,63) = %d bytes [%x], want 1 byte [3f]", n, buf[:n])
	}

	// Delta = -64 (min 7-bit signed)
	n = addcoord(buf, 64, 0)
	if n != 1 || buf[0] != 0x40 { // -64 & 0x7F = 0x40
		t.Errorf("addcoord(64,0) = %d bytes [%x], want 1 byte [40]", n, buf[:n])
	}
}

func TestAddcoordLargeDelta(t *testing.T) {
	// Large deltas (outside -64 to 63) should be 3 bytes
	buf := make([]byte, 3)

	// Delta = 64 (just outside 7-bit range)
	n := addcoord(buf, 0, 64)
	if n != 3 {
		t.Errorf("addcoord(0,64) = %d bytes, want 3", n)
	}
	// Verify: 0x80 | (64 & 0x7F), 64>>7, 64>>15
	if buf[0] != (0x80 | 64) { // 0xC0
		t.Errorf("addcoord(0,64)[0] = 0x%x, want 0xc0", buf[0])
	}

	// Delta = 1000
	n = addcoord(buf, 0, 1000)
	if n != 3 {
		t.Errorf("addcoord(0,1000) = %d bytes, want 3", n)
	}
	// Decode: val = (buf[0]&0x7F) | (buf[1]<<7) | (buf[2]<<15)
	val := int(buf[0]&0x7F) | int(buf[1])<<7 | int(buf[2])<<15
	if val != 1000 {
		t.Errorf("addcoord(0,1000) decoded to %d, want 1000", val)
	}
}

func TestNormsq(t *testing.T) {
	tests := []struct {
		p    Point
		want int
	}{
		{Pt(0, 0), 0},
		{Pt(1, 0), 1},
		{Pt(0, 1), 1},
		{Pt(3, 4), 25},
		{Pt(-3, 4), 25},
	}
	for _, tc := range tests {
		got := normsq(tc.p)
		if got != tc.want {
			t.Errorf("normsq(%v) = %d, want %d", tc.p, got, tc.want)
		}
	}
}
