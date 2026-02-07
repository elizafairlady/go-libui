package draw

import "testing"

func TestIcossin(t *testing.T) {
	tests := []struct {
		deg     int
		wantCos int
		wantSin int
	}{
		{0, 1024, 0},
		{90, 0, 1024},
		{180, -1024, 0},
		{270, 0, -1024},
		{360, 1024, 0},
		{45, 724, 724}, // cos(45) = sin(45) ≈ 0.7071 * 1024 = 724
		{30, 887, 512}, // cos(30) ≈ 0.866 * 1024 = 887, sin(30) = 0.5 * 1024 = 512
		{60, 512, 887}, // cos(60) = sin(30), sin(60) = cos(30)
		{-90, 0, -1024},
		{-180, -1024, 0},
	}

	for _, tc := range tests {
		cos, sin := Icossin(tc.deg)
		if cos != tc.wantCos || sin != tc.wantSin {
			t.Errorf("Icossin(%d) = (%d, %d), want (%d, %d)", tc.deg, cos, sin, tc.wantCos, tc.wantSin)
		}
	}
}

func TestIcossinSymmetry(t *testing.T) {
	// sin(x) = sin(180-x)
	for deg := 0; deg <= 90; deg++ {
		_, sin1 := Icossin(deg)
		_, sin2 := Icossin(180 - deg)
		if sin1 != sin2 {
			t.Errorf("sin(%d)=%d != sin(%d)=%d", deg, sin1, 180-deg, sin2)
		}
	}

	// cos(x) = -cos(180-x)
	for deg := 0; deg <= 90; deg++ {
		cos1, _ := Icossin(deg)
		cos2, _ := Icossin(180 - deg)
		if cos1 != -cos2 {
			t.Errorf("cos(%d)=%d != -cos(%d)=%d", deg, cos1, 180-deg, cos2)
		}
	}
}

func TestIsqrt(t *testing.T) {
	tests := []struct {
		n    int
		want int
	}{
		{0, 0},
		{1, 1},
		{4, 2},
		{9, 3},
		{16, 4},
		{25, 5},
		{100, 10},
		{2, 1},
		{3, 1},
		{5, 2},
		{8, 2},
	}
	for _, tc := range tests {
		got := isqrt(tc.n)
		if got != tc.want {
			t.Errorf("isqrt(%d) = %d, want %d", tc.n, got, tc.want)
		}
	}
}
