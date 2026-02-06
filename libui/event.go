package ui

// Mouse represents a decoded mouse event.
type Mouse struct {
	X       int
	Y       int
	Buttons int
	ScrollY int // +1 / -1 for wheel
}

// Resize represents a window resize event.
type Resize struct {
	Width  int
	Height int
}

// Key represents a decoded keyboard event.
type Key struct {
	Rune rune
}
