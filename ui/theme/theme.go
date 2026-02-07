// Package theme defines the visual style model for the UI framework.
// It specifies colors, fonts, spacing, and border treatment.
// The renderer resolves final styles by overlaying node props on top
// of theme defaults.
package theme

import (
	"github.com/elizafairlady/go-libui/draw"
)

// Theme holds the visual style defaults for rendering.
type Theme struct {
	// Colors (as 0xRRGGBBFF uint32 values)
	Background uint32
	Foreground uint32
	Highlight  uint32
	HighText   uint32
	Border     uint32
	ButtonBg   uint32
	ButtonFg   uint32
	InputBg    uint32
	InputFg    uint32
	FocusRing  uint32

	// Font names (Plan 9 font paths)
	FontName     string
	BoldFontName string

	// Metrics (in pixels)
	Pad        int // default padding
	Gap        int // default gap between children
	BorderW    int // border width
	Radius     int // corner radius (0 = square)
	FocusRingW int // focus ring width
	ScrollW    int // scrollbar width

	// Cached allocated images (filled by Alloc)
	BgImage        *draw.Image
	FgImage        *draw.Image
	HighImage      *draw.Image
	HighTextImage  *draw.Image
	BorderImage    *draw.Image
	ButtonBgImage  *draw.Image
	ButtonFgImage  *draw.Image
	InputBgImage   *draw.Image
	InputFgImage   *draw.Image
	FocusRingImage *draw.Image
}

// Default returns the default Acme-inspired theme.
// Warm cream background, subtle grey borders, calm blue accents.
func Default() *Theme {
	return &Theme{
		Background: draw.DAcmeYellow,
		Foreground: draw.DAcmeText,
		Highlight:  draw.DAcmeHigh,
		HighText:   draw.DAcmeText,
		Border:     draw.DAcmeBorder,
		ButtonBg:   draw.DAcmeButton,
		ButtonFg:   draw.DAcmeText,
		InputBg:    draw.DAcmeInput,
		InputFg:    draw.DAcmeText,
		FocusRing:  draw.DAcmeFocus,

		FontName:     "",
		BoldFontName: "",

		Pad:        6,
		Gap:        4,
		BorderW:    1,
		Radius:     0,
		FocusRingW: 1,
		ScrollW:    10,
	}
}

// Alloc allocates display images for all theme colors.
// Call this after display init. On error, falls back to nil images.
func (t *Theme) Alloc(d *draw.Display) {
	t.BgImage = allocColor(d, t.Background)
	t.FgImage = allocColor(d, t.Foreground)
	t.HighImage = allocColor(d, t.Highlight)
	t.HighTextImage = allocColor(d, t.HighText)
	t.BorderImage = allocColor(d, t.Border)
	t.ButtonBgImage = allocColor(d, t.ButtonBg)
	t.ButtonFgImage = allocColor(d, t.ButtonFg)
	t.InputBgImage = allocColor(d, t.InputBg)
	t.InputFgImage = allocColor(d, t.InputFg)
	t.FocusRingImage = allocColor(d, t.FocusRing)
}

// Free releases allocated color images.
func (t *Theme) Free() {
	imgs := []*draw.Image{
		t.BgImage, t.FgImage, t.HighImage, t.HighTextImage,
		t.BorderImage, t.ButtonBgImage, t.ButtonFgImage,
		t.InputBgImage, t.InputFgImage, t.FocusRingImage,
	}
	for _, img := range imgs {
		if img != nil {
			img.Free()
		}
	}
}

// allocColor allocates a 1x1 replicated image of the given color.
func allocColor(d *draw.Display, col uint32) *draw.Image {
	img, err := d.AllocImage(draw.Rect(0, 0, 1, 1), draw.RGB24, true, col)
	if err != nil {
		return nil
	}
	return img
}

// ParseColor parses a color string. Supports:
//   - Named colors: "black", "white", "red", etc.
//   - Hex: "0xFF0000FF"
//
// Returns 0 on failure.
func ParseColor(s string) uint32 {
	switch s {
	case "black":
		return draw.DBlack
	case "white":
		return draw.DWhite
	case "red":
		return draw.DRed
	case "green":
		return draw.DGreen
	case "blue":
		return draw.DBlue
	case "cyan":
		return draw.DCyan
	case "magenta":
		return draw.DMagenta
	case "yellow":
		return draw.DYellow
	case "paleyellow":
		return draw.DPaleyellow
	case "darkyellow":
		return draw.DDarkyellow
	case "darkgreen":
		return draw.DDarkgreen
	case "palegreen":
		return draw.DPalegreen
	case "paleblue":
		return draw.DPaleblue
	case "greyblue":
		return draw.DGreyblue
	case "acmeyellow":
		return draw.DAcmeYellow
	case "acmecyan", "acmetag":
		return draw.DAcmeCyan
	case "acmeborder":
		return draw.DAcmeBorder
	case "acmetext":
		return draw.DAcmeText
	case "acmedim":
		return draw.DAcmeDim
	case "acmefocus":
		return draw.DAcmeFocus
	case "acmebutton":
		return draw.DAcmeButton
	case "acmeinput":
		return draw.DAcmeInput
	case "acmehigh":
		return draw.DAcmeHigh
	}
	// Try hex
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		var v uint32
		for _, c := range s[2:] {
			v <<= 4
			switch {
			case c >= '0' && c <= '9':
				v |= uint32(c - '0')
			case c >= 'a' && c <= 'f':
				v |= uint32(c-'a') + 10
			case c >= 'A' && c <= 'F':
				v |= uint32(c-'A') + 10
			default:
				return 0
			}
		}
		return v
	}
	return 0
}
