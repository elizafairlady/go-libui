package draw

// Integer cos and sin tables using 10-bit fixed point.
// Angles are in units of 1/64 of a degree (0-23040 = 0-360 degrees).

const (
	fixshift = 10
	fixscale = 1 << fixshift
)

// Icossin returns (cos(a), sin(a)) in fixed-point format.
// Angle a is in 64ths of a degree.
func Icossin(a int) (cos, sin int) {
	// Normalize angle to 0-23039
	if a < 0 {
		a = -a
		sin = -1
	} else {
		sin = 1
	}
	a %= 23040
	if a < 0 {
		a += 23040
	}

	// Map to quadrant
	var neg bool
	if a >= 11520 { // 180 degrees
		a -= 11520
		neg = !neg
	}
	if a >= 5760 { // 90 degrees
		a = 11520 - a
	}

	// Look up in table
	idx := a / 64
	if idx >= len(costab) {
		idx = len(costab) - 1
	}
	cos = costab[idx]
	sin = sintab[idx] * sin
	if neg {
		cos = -cos
	}
	return
}

// Icossin2 is like Icossin but uses a different angle base.
func Icossin2(x, y int) (cos, sin int) {
	if x == 0 && y == 0 {
		return fixscale, 0
	}
	// Compute angle from vector
	// This is a simplified version
	mag := isqrt(x*x + y*y)
	if mag == 0 {
		return fixscale, 0
	}
	cos = x * fixscale / mag
	sin = y * fixscale / mag
	return
}

// isqrt computes integer square root.
func isqrt(n int) int {
	if n < 0 {
		return 0
	}
	if n == 0 {
		return 0
	}
	x := n
	y := (x + 1) / 2
	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return x
}

// Cos and sin tables for 0-90 degrees in increments of 1 degree.
// Values are scaled by fixscale (1024).
var costab = [91]int{
	1024, 1024, 1023, 1022, 1021, 1019, 1016, 1013, 1009, 1004,
	999, 993, 987, 980, 972, 964, 955, 946, 936, 925,
	914, 903, 891, 878, 865, 851, 837, 822, 807, 791,
	775, 758, 741, 724, 706, 688, 669, 650, 630, 610,
	590, 569, 548, 526, 505, 483, 460, 438, 415, 392,
	369, 345, 321, 297, 273, 249, 224, 200, 175, 150,
	125, 100, 75, 50, 25, 0, -25, -50, -75, -100,
	-125, -150, -175, -200, -224, -249, -273, -297, -321, -345,
	-369, -392, -415, -438, -460, -483, -505, -526, -548, -569,
	-590,
}

var sintab = [91]int{
	0, 18, 36, 54, 71, 89, 107, 125, 143, 160,
	178, 195, 213, 230, 248, 265, 282, 299, 316, 333,
	350, 367, 383, 400, 416, 432, 448, 464, 480, 496,
	511, 526, 541, 556, 571, 586, 600, 614, 628, 642,
	656, 669, 682, 695, 708, 720, 732, 744, 756, 768,
	779, 790, 801, 811, 821, 831, 841, 850, 859, 868,
	877, 885, 893, 901, 908, 915, 922, 928, 935, 941,
	946, 952, 957, 961, 966, 970, 974, 978, 981, 984,
	987, 990, 992, 994, 996, 997, 998, 999, 1000, 1000,
	1000,
}
