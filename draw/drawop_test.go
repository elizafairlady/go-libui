package draw

import "testing"

func TestDrawreplxy(t *testing.T) {
	tests := []struct {
		min, max, x int
		want        int
	}{
		{0, 10, 0, 0},
		{0, 10, 5, 5},
		{0, 10, 10, 0},
		{0, 10, 15, 5},
		{0, 10, -1, 9},
		{0, 10, -10, 0},
		{5, 15, 5, 5},
		{5, 15, 20, 10},
		{5, 15, 3, 13},
	}
	for _, tc := range tests {
		got := Drawreplxy(tc.min, tc.max, tc.x)
		if got != tc.want {
			t.Errorf("Drawreplxy(%d, %d, %d) = %d, want %d", tc.min, tc.max, tc.x, got, tc.want)
		}
	}
}

func TestDrawrepl(t *testing.T) {
	r := Rect(0, 0, 10, 10)
	p := Drawrepl(r, Pt(15, 25))
	if p.X != 5 || p.Y != 5 {
		t.Errorf("Drawrepl(%v, (15,25)) = %v, want (5,5)", r, p)
	}

	p = Drawrepl(r, Pt(-3, -7))
	if p.X != 7 || p.Y != 3 {
		t.Errorf("Drawrepl(%v, (-3,-7)) = %v, want (7,3)", r, p)
	}
}
