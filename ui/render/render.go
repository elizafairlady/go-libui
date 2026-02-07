// Package render implements the draw backend for the UI framework.
// It takes a layout tree and paints it to a /dev/draw image, handles
// mouse/keyboard input, performs hit-testing, and emits actions.
package render

import (
	"strconv"
	"strings"

	"github.com/elizafairlady/go-libui/draw"
	"github.com/elizafairlady/go-libui/ui/layout"
	"github.com/elizafairlady/go-libui/ui/proto"
	"github.com/elizafairlady/go-libui/ui/theme"
)

// Renderer paints a layout tree to a draw.Image.
type Renderer struct {
	Display *draw.Display
	Screen  *draw.Image
	Font    *draw.Font
	Theme   *theme.Theme

	// State
	Focus     string         // focused node ID
	Hover     string         // hovered node ID
	ScrollOff map[string]int // scroll offsets by node ID

	// Tag frames (Acme-style editable tag bars)
	Tags map[string]*TagState

	// Cached color images
	colors map[uint32]*draw.Image
}

// New creates a renderer for the given display.
func New(d *draw.Display, t *theme.Theme) *Renderer {
	r := &Renderer{
		Display:   d,
		Screen:    d.ScreenImage,
		Font:      d.DefaultFont,
		Theme:     t,
		ScrollOff: make(map[string]int),
		colors:    make(map[uint32]*draw.Image),
	}
	t.Alloc(d)
	return r
}

// LayoutConfig returns a layout.Config using the renderer's font metrics.
func (r *Renderer) LayoutConfig() *layout.Config {
	return &layout.Config{
		Measure: func(text, font string, size int) (int, int) {
			f := r.Font
			return f.StringWidth(text), f.Height
		},
		DefaultPad: r.Theme.Pad,
		DefaultGap: r.Theme.Gap,
		FontHeight: r.Font.Height,
	}
}

// Paint draws the entire layout tree to screen.
func (r *Renderer) Paint(root *layout.RNode) {
	if root == nil {
		return
	}
	// Clear background
	r.Screen.Draw(root.Rect, r.Theme.BgImage, draw.ZP)
	r.paintNode(root)
	r.Display.Flush()
}

func (r *Renderer) paintNode(n *layout.RNode) {
	rect := n.Rect
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}

	switch n.Type {
	case "rect":
		r.paintRect(n)
	case "text":
		r.paintText(n)
	case "button":
		r.paintButton(n)
	case "checkbox":
		r.paintCheckbox(n)
	case "textbox":
		r.paintTextbox(n)
	case "tag":
		r.paintTag(n)
	case "vbox", "hbox", "stack", "row", "scroll", "spacer":
		r.paintContainer(n)
	default:
		r.paintContainer(n)
	}
}

func (r *Renderer) paintContainer(n *layout.RNode) {
	// Draw background if specified
	if bg := n.Props["bg"]; bg != "" {
		col := r.colorImage(theme.ParseColor(bg))
		if col != nil {
			r.Screen.Draw(n.Rect, col, draw.ZP)
		}
	}
	// Draw border if specified
	if border := n.Props["border"]; border != "" {
		bw := propInt(n.Props, "borderw", r.Theme.BorderW)
		col := r.colorImage(theme.ParseColor(border))
		if col != nil {
			r.Screen.Border(n.Rect, bw, col, draw.ZP)
		}
	}
	// Draw focus ring
	if n.ID == r.Focus && n.Props["focusable"] == "1" {
		r.Screen.Border(n.Rect, r.Theme.FocusRingW, r.Theme.FocusRingImage, draw.ZP)
	}
	// Paint children
	for _, c := range n.Children {
		r.paintNode(c)
	}
}

func (r *Renderer) paintRect(n *layout.RNode) {
	bg := n.Props["bg"]
	if bg == "" {
		bg = "black"
	}
	col := r.colorImage(theme.ParseColor(bg))
	if col != nil {
		r.Screen.Draw(n.Rect, col, draw.ZP)
	}
}

func (r *Renderer) paintText(n *layout.RNode) {
	text := n.Props["text"]
	if text == "" {
		return
	}
	fg := r.Theme.FgImage
	if fgc := n.Props["fg"]; fgc != "" {
		if c := r.colorImage(theme.ParseColor(fgc)); c != nil {
			fg = c
		}
	}
	bg := r.Theme.BgImage
	if bgc := n.Props["bg"]; bgc != "" {
		if c := r.colorImage(theme.ParseColor(bgc)); c != nil {
			bg = c
			r.Screen.Draw(n.Rect, bg, draw.ZP)
		}
	}
	pad := propInt(n.Props, "pad", r.Theme.Pad)
	pt := draw.Pt(n.Rect.Min.X+pad, n.Rect.Min.Y+pad)
	r.Screen.StringBg(pt, fg, draw.ZP, r.Font, text, bg, draw.ZP)
}

func (r *Renderer) paintButton(n *layout.RNode) {
	text := n.Props["text"]
	bg := r.Theme.ButtonBgImage
	fg := r.Theme.ButtonFgImage
	bord := r.Theme.BorderImage

	// Hover effect — subtle highlight
	if n.ID == r.Hover {
		bg = r.Theme.HighImage
	}

	// Fill background
	r.Screen.Draw(n.Rect, bg, draw.ZP)

	// 1px border
	r.Screen.Border(n.Rect, r.Theme.BorderW, bord, draw.ZP)

	// Focus: draw a subtle left accent line instead of full border
	if n.ID == r.Focus {
		accent := draw.Rect(n.Rect.Min.X, n.Rect.Min.Y, n.Rect.Min.X+2, n.Rect.Max.Y)
		r.Screen.Draw(accent, r.Theme.FocusRingImage, draw.ZP)
	}

	// Center text vertically
	tw := r.Font.StringWidth(text)
	tx := n.Rect.Min.X + (n.Rect.Dx()-tw)/2
	ty := n.Rect.Min.Y + (n.Rect.Dy()-r.Font.Height)/2
	if tx < n.Rect.Min.X+2 {
		tx = n.Rect.Min.X + 2
	}
	r.Screen.StringBg(draw.Pt(tx, ty), fg, draw.ZP, r.Font, text, bg, draw.ZP)
}

func (r *Renderer) paintCheckbox(n *layout.RNode) {
	pad := propInt(n.Props, "pad", r.Theme.Pad)
	boxSize := r.Font.Height - 2
	bx := n.Rect.Min.X + pad
	by := n.Rect.Min.Y + pad + 1
	boxR := draw.Rect(bx, by, bx+boxSize, by+boxSize)

	r.Screen.Draw(boxR, r.Theme.InputBgImage, draw.ZP)
	r.Screen.Border(boxR, 1, r.Theme.BorderImage, draw.ZP)

	if n.Props["checked"] == "1" {
		// Draw X mark
		inner := boxR.Inset(2)
		r.Screen.Line(inner.Min, draw.Pt(inner.Max.X-1, inner.Max.Y-1), 0, 0, 1, r.Theme.FgImage, draw.ZP)
		r.Screen.Line(draw.Pt(inner.Max.X-1, inner.Min.Y), draw.Pt(inner.Min.X, inner.Max.Y-1), 0, 0, 1, r.Theme.FgImage, draw.ZP)
	}

	text := n.Props["text"]
	if text != "" {
		pt := draw.Pt(bx+boxSize+4, n.Rect.Min.Y+pad)
		r.Screen.StringBg(pt, r.Theme.FgImage, draw.ZP, r.Font, text, r.Theme.BgImage, draw.ZP)
	}

	if n.ID == r.Focus {
		r.Screen.Border(n.Rect, r.Theme.FocusRingW, r.Theme.FocusRingImage, draw.ZP)
	}
}

func (r *Renderer) paintTextbox(n *layout.RNode) {
	pad := propInt(n.Props, "pad", r.Theme.Pad)
	r.Screen.Draw(n.Rect, r.Theme.InputBgImage, draw.ZP)
	r.Screen.Border(n.Rect, 1, r.Theme.BorderImage, draw.ZP)

	text := n.Props["text"]
	placeholder := n.Props["placeholder"]
	fg := r.Theme.InputFgImage
	display := text
	if display == "" && placeholder != "" {
		display = placeholder
		// Dimmed placeholder using a lighter color
		dimCol := r.colorImage(draw.DAcmeDim)
		if dimCol != nil {
			fg = dimCol
		} else {
			fg = r.Theme.BorderImage
		}
	}

	// Vertically center text
	ty := n.Rect.Min.Y + (n.Rect.Dy()-r.Font.Height)/2
	pt := draw.Pt(n.Rect.Min.X+pad+1, ty)
	if display != "" {
		r.Screen.StringBg(pt, fg, draw.ZP, r.Font, display, r.Theme.InputBgImage, draw.ZP)
	}

	// Draw cursor if focused
	if n.ID == r.Focus {
		cx := pt.X
		if text != "" {
			cursorPos := propInt(n.Props, "cursor", len([]rune(text)))
			ctext := string([]rune(text)[:cursorPos])
			cx += r.Font.StringWidth(ctext)
		}
		// Thin 1px cursor
		r.Screen.Draw(draw.Rect(cx, pt.Y, cx+1, pt.Y+r.Font.Height), r.Theme.FgImage, draw.ZP)
	}

	// Focus: bottom accent line
	if n.ID == r.Focus {
		bot := draw.Rect(n.Rect.Min.X, n.Rect.Max.Y-2, n.Rect.Max.X, n.Rect.Max.Y)
		r.Screen.Draw(bot, r.Theme.FocusRingImage, draw.ZP)
	}
}

// colorImage returns a cached 1x1 replicated image for the given color.
func (r *Renderer) colorImage(col uint32) *draw.Image {
	if col == 0 {
		return nil
	}
	if img, ok := r.colors[col]; ok {
		return img
	}
	img, err := r.Display.AllocImage(draw.Rect(0, 0, 1, 1), draw.RGB24, true, col)
	if err != nil {
		return nil
	}
	r.colors[col] = img
	return img
}

// --- Action generation ---

// MouseAction generates a semantic action from a mouse event and hit-test result.
func MouseAction(hit *layout.RNode, button int, pt draw.Point) *proto.Action {
	if hit == nil {
		return nil
	}
	kind := "click"
	a := &proto.Action{
		Kind: kind,
		KVs: map[string]string{
			"id":     hit.ID,
			"button": strconv.Itoa(button),
			"x":      strconv.Itoa(pt.X),
			"y":      strconv.Itoa(pt.Y),
		},
	}
	// For specific widget types, generate semantic actions
	switch hit.Type {
	case "checkbox":
		a.Kind = "toggle"
		v := "1"
		if hit.Props["checked"] == "1" {
			v = "0"
		}
		a.KVs["value"] = v
	case "button":
		if on := hit.Props["on"]; on != "" {
			a.KVs["action"] = on
		}
	}
	return a
}

// KeyAction generates a semantic action from a keyboard event.
func KeyAction(focusID string, key rune, name string) *proto.Action {
	a := &proto.Action{
		Kind: "key",
		KVs: map[string]string{
			"id": focusID,
		},
	}
	if name != "" {
		a.KVs["key"] = name
	}
	if key != 0 {
		a.KVs["rune"] = string(key)
	}
	return a
}

// ScrollAction generates a scroll action.
func ScrollAction(nodeID string, dy int) *proto.Action {
	return &proto.Action{
		Kind: "scroll",
		KVs: map[string]string{
			"id": nodeID,
			"dy": strconv.Itoa(dy),
		},
	}
}

// FocusAction generates a focus change action.
func FocusAction(nodeID string) *proto.Action {
	return &proto.Action{
		Kind: "focus",
		KVs: map[string]string{
			"id": nodeID,
		},
	}
}

// InputAction generates a text input action for a textbox.
func InputAction(nodeID, text string, cursor int) *proto.Action {
	return &proto.Action{
		Kind: "input",
		KVs: map[string]string{
			"id":     nodeID,
			"text":   text,
			"cursor": strconv.Itoa(cursor),
		},
	}
}

// --- Focus navigation ---

// NextFocusable finds the next focusable node after the current focus.
func NextFocusable(root *layout.RNode, currentFocus string) string {
	nodes := layout.Flatten(root)
	var focusable []string
	for _, n := range nodes {
		if isFocusable(n) {
			focusable = append(focusable, n.ID)
		}
	}
	if len(focusable) == 0 {
		return ""
	}
	if currentFocus == "" {
		return focusable[0]
	}
	for i, id := range focusable {
		if id == currentFocus {
			return focusable[(i+1)%len(focusable)]
		}
	}
	return focusable[0]
}

// PrevFocusable finds the previous focusable node before the current focus.
func PrevFocusable(root *layout.RNode, currentFocus string) string {
	nodes := layout.Flatten(root)
	var focusable []string
	for _, n := range nodes {
		if isFocusable(n) {
			focusable = append(focusable, n.ID)
		}
	}
	if len(focusable) == 0 {
		return ""
	}
	if currentFocus == "" {
		return focusable[len(focusable)-1]
	}
	for i, id := range focusable {
		if id == currentFocus {
			return focusable[(i-1+len(focusable))%len(focusable)]
		}
	}
	return focusable[len(focusable)-1]
}

func isFocusable(n *layout.RNode) bool {
	if n.Props["enabled"] == "0" {
		return false
	}
	if n.Props["focusable"] == "1" {
		return true
	}
	switch n.Type {
	case "button", "checkbox", "textbox", "tag":
		return true
	}
	return false
}

func propInt(props map[string]string, key string, def int) int {
	v, ok := props[key]
	if !ok || v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// KeyName maps keyboard runes/codes to human-readable names.
func KeyName(key rune) string {
	switch key {
	case '\n', '\r':
		return "Enter"
	case '\t':
		return "Tab"
	case 27:
		return "Esc"
	case 127:
		return "Del"
	case 8:
		return "Backspace"
	}
	// Check for well-known key codes from draw/keyboard.go
	names := map[rune]string{
		0xF800: "Home",
		0xF801: "Up",
		0xF802: "PgUp",
		0xF803: "Print",
		0xF804: "Left",
		0xF805: "Right",
		0xF807: "Down",
		0xF808: "End",
		0xF809: "PgDn",
		0xF80A: "Ins",
	}
	if name, ok := names[key]; ok {
		return name
	}
	if key >= 1 && key <= 26 {
		return "Ctrl+" + string(rune('A'+key-1))
	}
	return ""
}

// Unused import suppressor
var _ = strings.Contains
