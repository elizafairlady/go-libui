package draw

import "testing"

// TestKeyboardConstants verifies all key constants match 9front keyboard.h exactly.
func TestKeyboardConstants(t *testing.T) {
	// Base ranges
	if KF != 0xF000 {
		t.Errorf("KF = %#x, want 0xF000", KF)
	}
	if Spec != 0xF800 {
		t.Errorf("Spec = %#x, want 0xF800", Spec)
	}
	if PF != 0xF820 {
		t.Errorf("PF = %#x, want 0xF820", PF)
	}

	tests := []struct {
		name string
		got  int
		want int
	}{
		// Navigation keys
		{"Kview", Kview, 0xF800},
		{"KF1", KF1, 0xF001},
		{"KF2", KF2, 0xF002},
		{"KF12", KF12, 0xF00C},
		{"Khome", Khome, 0xF00D},
		{"Kup", Kup, 0xF00E},
		{"Kdown", Kdown, 0xF800}, // Kdown == Kview
		{"Kpgup", Kpgup, 0xF00F},
		{"Kprint", Kprint, 0xF010},
		{"Kleft", Kleft, 0xF011},
		{"Kright", Kright, 0xF012},
		{"Kpgdown", Kpgdown, 0xF013},
		{"Kins", Kins, 0xF014},
		{"Kalt", Kalt, 0xF015},
		{"Kshift", Kshift, 0xF016},
		{"Kctl", Kctl, 0xF017},
		{"Kend", Kend, 0xF018},
		{"Kscroll", Kscroll, 0xF019},
		{"Kscrolloneup", Kscrolloneup, 0xF020},
		{"Kscrollonedown", Kscrollonedown, 0xF021},

		// Multimedia keys
		{"Ksbwd", Ksbwd, 0xF022},
		{"Ksfwd", Ksfwd, 0xF023},
		{"Kpause", Kpause, 0xF024},
		{"Kvoldn", Kvoldn, 0xF025},
		{"Kvolup", Kvolup, 0xF026},
		{"Kmute", Kmute, 0xF027},
		{"Kbrtdn", Kbrtdn, 0xF028},
		{"Kbrtup", Kbrtup, 0xF029},

		// Control characters
		{"Ksoh", Ksoh, 0x01},
		{"Kstx", Kstx, 0x02},
		{"Ketx", Ketx, 0x03},
		{"Keof", Keof, 0x04},
		{"Kenq", Kenq, 0x05},
		{"Kack", Kack, 0x06},
		{"Kbs", Kbs, 0x08},
		{"Knack", Knack, 0x15},
		{"Ketb", Ketb, 0x17},
		{"Kdel", Kdel, 0x7f},
		{"Kesc", Kesc, 0x1b},

		// Special keys
		{"Kbreak", Kbreak, 0xF861},
		{"Kcaps", Kcaps, 0xF864},
		{"Knum", Knum, 0xF865},
		{"Kmiddle", Kmiddle, 0xF866},
		{"Kaltgr", Kaltgr, 0xF867},
		{"Kmod4", Kmod4, 0xF868},
		{"Kmouse", Kmouse, 0xF900},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

// TestKdownEqualsKview verifies Kdown == Kview as in 9front.
func TestKdownEqualsKview(t *testing.T) {
	if Kdown != Kview {
		t.Errorf("Kdown (%#x) != Kview (%#x)", Kdown, Kview)
	}
}
