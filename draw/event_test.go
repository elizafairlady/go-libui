package draw

import (
	"testing"
	"unsafe"
)

// TestEventConstants verifies event constants match 9front event.h.
func TestEventConstants(t *testing.T) {
	if Emouse != 1 {
		t.Errorf("Emouse = %d, want 1", Emouse)
	}
	if Ekeyboard != 2 {
		t.Errorf("Ekeyboard = %d, want 2", Ekeyboard)
	}
	if MAXSLAVE != 32 {
		t.Errorf("MAXSLAVE = %d, want 32", MAXSLAVE)
	}
	if EMAXMSG != 128+8192 {
		t.Errorf("EMAXMSG = %d, want %d", EMAXMSG, 128+8192)
	}
}

// TestEventStruct verifies the Event struct has the expected fields.
func TestEventStruct(t *testing.T) {
	var ev Event
	// Verify we can set all fields
	ev.Kbdc = 'a'
	ev.Mouse = Mouse{Point: Pt(10, 20), Buttons: 1, Msec: 100}
	ev.N = 5
	ev.Data[0] = 0x42

	if ev.Kbdc != 'a' {
		t.Errorf("Kbdc = %d, want 'a'", ev.Kbdc)
	}
	if ev.Mouse.X != 10 || ev.Mouse.Y != 20 {
		t.Errorf("Mouse.Point = (%d,%d), want (10,20)", ev.Mouse.X, ev.Mouse.Y)
	}
	if ev.Mouse.Buttons != 1 {
		t.Errorf("Mouse.Buttons = %d, want 1", ev.Mouse.Buttons)
	}
	if ev.N != 5 {
		t.Errorf("N = %d, want 5", ev.N)
	}
	if ev.Data[0] != 0x42 {
		t.Errorf("Data[0] = %#x, want 0x42", ev.Data[0])
	}

	// Data buffer is EMAXMSG bytes
	if len(ev.Data) != EMAXMSG {
		t.Errorf("len(Data) = %d, want %d", len(ev.Data), EMAXMSG)
	}
}

// TestEventKeyMask verifies we can combine event masks.
func TestEventKeyMask(t *testing.T) {
	both := Emouse | Ekeyboard
	if both != 3 {
		t.Errorf("Emouse|Ekeyboard = %d, want 3", both)
	}
	if both&Emouse == 0 {
		t.Error("Emouse bit not set in combined mask")
	}
	if both&Ekeyboard == 0 {
		t.Error("Ekeyboard bit not set in combined mask")
	}
}

// TestEventDataSize verifies the EMAXMSG buffer can hold a 9p header + data.
func TestEventDataSize(t *testing.T) {
	_ = unsafe.Sizeof(Event{}) // just verify it compiles/is usable
	if EMAXMSG < 128 {
		t.Error("EMAXMSG too small for 9p header")
	}
}
