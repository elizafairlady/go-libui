package draw

import "testing"

func TestParseCtlLine(t *testing.T) {
	// Simulated ctl output: 12 fields of 11 chars each + space
	s := "          3           0     r8g8b8           0           0           0        1024         768           0           0        1024         768 "
	fields := parseCtlLine(s)
	if len(fields) != 12 {
		t.Fatalf("parseCtlLine: got %d fields, want 12", len(fields))
	}
	if fields[0] != "3" {
		t.Errorf("field[0] = %q, want %q", fields[0], "3")
	}
	if fields[1] != "0" {
		t.Errorf("field[1] = %q, want %q", fields[1], "0")
	}
	if fields[2] != "r8g8b8" {
		t.Errorf("field[2] = %q, want %q", fields[2], "r8g8b8")
	}
	if fields[4] != "0" {
		t.Errorf("field[4] (min.x) = %q, want %q", fields[4], "0")
	}
	if fields[6] != "1024" {
		t.Errorf("field[6] (max.x) = %q, want %q", fields[6], "1024")
	}
	if fields[7] != "768" {
		t.Errorf("field[7] (max.y) = %q, want %q", fields[7], "768")
	}
}

func TestBufimage(t *testing.T) {
	// Create a minimal display to test bufimage
	d := &Display{
		bufsize: 100,
		bufp:    0,
	}
	d.buf = make([]byte, d.bufsize+5)

	// Allocate some space
	b, err := d.bufimage(10)
	if err != nil {
		t.Fatalf("bufimage(10): %v", err)
	}
	if len(b) != 10 {
		t.Errorf("bufimage(10) returned %d bytes, want 10", len(b))
	}
	if d.bufp != 10 {
		t.Errorf("after bufimage(10), bufp = %d, want 10", d.bufp)
	}

	// Allocate more
	b, err = d.bufimage(20)
	if err != nil {
		t.Fatalf("bufimage(20): %v", err)
	}
	if len(b) != 20 {
		t.Errorf("bufimage(20) returned %d bytes, want 20", len(b))
	}
	if d.bufp != 30 {
		t.Errorf("after bufimage(20), bufp = %d, want 30", d.bufp)
	}

	// Too large
	_, err = d.bufimage(200)
	if err == nil {
		t.Error("bufimage(200) should fail on bufsize=100")
	}
}

func TestBufimageop(t *testing.T) {
	d := &Display{
		bufsize: 100,
		bufp:    0,
	}
	d.buf = make([]byte, d.bufsize+5)

	// SoverD should not add prefix
	b, err := d.bufimageop(10, SoverD)
	if err != nil {
		t.Fatalf("bufimageop(10, SoverD): %v", err)
	}
	if len(b) != 10 {
		t.Errorf("bufimageop SoverD: got %d bytes, want 10", len(b))
	}
	if d.bufp != 10 {
		t.Errorf("bufimageop SoverD: bufp = %d, want 10", d.bufp)
	}

	// Non-SoverD should add 'O' + op prefix
	b, err = d.bufimageop(10, DoverS)
	if err != nil {
		t.Fatalf("bufimageop(10, DoverS): %v", err)
	}
	if len(b) != 10 {
		t.Errorf("bufimageop DoverS: returned %d bytes, want 10", len(b))
	}
	// Total should be 10 + 12 (10 prev + 2 prefix + 10 data)
	if d.bufp != 22 {
		t.Errorf("bufimageop DoverS: bufp = %d, want 22", d.bufp)
	}
	// Check prefix bytes
	if d.buf[10] != 'O' {
		t.Errorf("bufimageop DoverS: prefix[0] = %c, want 'O'", d.buf[10])
	}
	if d.buf[11] != byte(DoverS) {
		t.Errorf("bufimageop DoverS: prefix[1] = %d, want %d", d.buf[11], DoverS)
	}
}
