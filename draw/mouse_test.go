package draw

import (
	"fmt"
	"testing"
)

// TestAtoiField tests the Plan 9 mouse message field parser.
func TestAtoiField(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"         100", 100},
		{"           0", 0},
		{"        -123", -123},
		{"  1234567890", 1234567890},
		{"         256", 256},
		{"           1", 1},
	}
	for _, tt := range tests {
		got := atoiField([]byte(tt.input))
		if got != tt.want {
			t.Errorf("atoiField(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// TestMouseMessageParsing tests parsing a complete Plan 9 mouse message.
func TestMouseMessageParsing(t *testing.T) {
	// Build a fake mouse message: 'm' + x[12] + y[12] + buttons[12] + msec[12]
	msg := fmt.Sprintf("m%12d%12d%12d%12d", 100, 200, 1, 12345678)
	buf := []byte(msg)

	if buf[0] != 'm' {
		t.Fatal("expected 'm' prefix")
	}
	if len(buf) != 1+4*12 {
		t.Fatalf("message length = %d, want %d", len(buf), 1+4*12)
	}

	x := atoiField(buf[1 : 1+12])
	y := atoiField(buf[1+12 : 1+2*12])
	buttons := atoiField(buf[1+2*12 : 1+3*12])
	msec := uint32(atoiField(buf[1+3*12 : 1+4*12]))

	if x != 100 {
		t.Errorf("x = %d, want 100", x)
	}
	if y != 200 {
		t.Errorf("y = %d, want 200", y)
	}
	if buttons != 1 {
		t.Errorf("buttons = %d, want 1", buttons)
	}
	if msec != 12345678 {
		t.Errorf("msec = %d, want 12345678", msec)
	}
}

// TestMouseResizeMessage tests parsing a resize message.
func TestMouseResizeMessage(t *testing.T) {
	msg := fmt.Sprintf("r%12d%12d%12d%12d", 50, 75, 0, 99999)
	buf := []byte(msg)

	if buf[0] != 'r' {
		t.Fatal("expected 'r' prefix")
	}

	x := atoiField(buf[1 : 1+12])
	y := atoiField(buf[1+12 : 1+2*12])
	buttons := atoiField(buf[1+2*12 : 1+3*12])
	msec := uint32(atoiField(buf[1+3*12 : 1+4*12]))

	if x != 50 {
		t.Errorf("x = %d, want 50", x)
	}
	if y != 75 {
		t.Errorf("y = %d, want 75", y)
	}
	if buttons != 0 {
		t.Errorf("buttons = %d, want 0", buttons)
	}
	if msec != 99999 {
		t.Errorf("msec = %d, want 99999", msec)
	}
}

// TestMouseButtonBits verifies the button bit convention: LMR=124.
func TestMouseButtonBits(t *testing.T) {
	// In Plan 9, buttons are: left=1, middle=2, right=4
	left := 1
	middle := 2
	right := 4

	if left&1 == 0 {
		t.Error("left button bit 1 not set")
	}
	if middle&2 == 0 {
		t.Error("middle button bit 2 not set")
	}
	if right&4 == 0 {
		t.Error("right button bit 4 not set")
	}

	// Combined buttons
	all := left | middle | right
	if all != 7 {
		t.Errorf("all buttons = %d, want 7", all)
	}
}
