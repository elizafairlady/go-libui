// body.go implements multi-line editable text areas backed by frame.Frame.
//
// A body is the Acme-style editable text area — a viewport into a
// potentially large rune buffer. The frame only displays what fits;
// scrolling adjusts the origin (org). The renderer owns the text buffer
// and frame state, which persists across view tree rebuilds.
//
// Mouse interaction:
//   - B1: select (frame.Select)
//   - B2: execute (word at point → "execute" action)
//   - B3: look (word at point → "look" action)
//
// Keyboard: printable runes insert at selection, backspace deletes.
// The scroll callback enables selecting past the visible region.
package render

import (
	"unicode"

	"github.com/elizafairlady/go-libui/draw"
	"github.com/elizafairlady/go-libui/frame"
	"github.com/elizafairlady/go-libui/ui/layout"
	"github.com/elizafairlady/go-libui/ui/proto"
	"github.com/elizafairlady/go-libui/ui/theme"
)

// BodyState holds the state for a body text area.
type BodyState struct {
	Frame *frame.Frame
	Text  []rune         // complete text buffer
	Org   int            // first rune visible in frame
	Rect  draw.Rectangle // current layout rect
	Init  bool           // has been initialized
	// Dirty tracks whether the text has been modified since last clean.
	Dirty bool
}

// ensureBody ensures a BodyState exists for the given node ID.
func (r *Renderer) ensureBody(id string) *BodyState {
	if r.Bodies == nil {
		r.Bodies = make(map[string]*BodyState)
	}
	if bs, ok := r.Bodies[id]; ok {
		return bs
	}
	bs := &BodyState{
		Frame: &frame.Frame{},
	}
	r.Bodies[id] = bs
	return bs
}

// initBody initializes or reinitializes the frame for a body node.
func (r *Renderer) initBody(bs *BodyState, n *layout.RNode) {
	rect := n.Rect
	cols := r.bodyColors(n)

	bs.Frame.Init(rect, r.Font, r.Screen, cols)
	bs.Frame.Scroll = func(f *frame.Frame, dl int) {
		r.bodyScroll(bs, dl)
	}
	bs.Rect = rect
	bs.Init = true

	// Insert visible portion of text starting from org
	r.bodyFill(bs)
}

// bodyColors returns the Acme-style body colors, checking node props for overrides.
func (r *Renderer) bodyColors(n *layout.RNode) [frame.NCol]*draw.Image {
	bgColor := uint32(draw.DAcmeYellow)
	if bg := n.Props["bg"]; bg != "" {
		if c := theme.ParseColor(bg); c != 0 {
			bgColor = c
		}
	}
	bodyBg := r.colorImage(bgColor)
	high := r.colorImage(draw.DAcmeHigh)
	bord := r.colorImage(draw.DAcmeBorder)
	text := r.colorImage(draw.DAcmeText)
	htext := r.colorImage(draw.DAcmeText)

	if bodyBg == nil {
		bodyBg = r.Theme.BgImage
	}
	if high == nil {
		high = r.Theme.HighImage
	}
	if bord == nil {
		bord = r.Theme.BorderImage
	}
	if text == nil {
		text = r.Theme.FgImage
	}
	if htext == nil {
		htext = r.Theme.FgImage
	}

	return [frame.NCol]*draw.Image{
		frame.ColBack:  bodyBg,
		frame.ColHigh:  high,
		frame.ColBord:  bord,
		frame.ColText:  text,
		frame.ColHText: htext,
	}
}

// bodyFill inserts text from org into the frame up to what fits.
func (r *Renderer) bodyFill(bs *BodyState) {
	if bs.Org > len(bs.Text) {
		bs.Org = len(bs.Text)
	}
	end := len(bs.Text)
	runes := bs.Text[bs.Org:]
	if len(runes) > 0 {
		bs.Frame.Insert(runes, 0)
	}
	_ = end
}

// paintBody renders a body node using the frame package.
func (r *Renderer) paintBody(n *layout.RNode) {
	bs := r.ensureBody(n.ID)

	// Only set initial text from tree props on first init
	if !bs.Init {
		text := n.Props["text"]
		if text != "" {
			bs.Text = []rune(text)
		}
	}

	// First time or rect changed: full reinit
	if !bs.Init || bs.Rect != n.Rect {
		p0, p1 := bs.Frame.P0, bs.Frame.P1
		bs.Frame.Clear(false)
		r.initBody(bs, n)
		// Restore selection (in frame coordinates)
		fp0 := uint32(0)
		fp1 := uint32(0)
		if bs.Frame.Nchars > 0 {
			// Convert buffer selection to frame-relative
			if int(p0)+bs.Org <= len(bs.Text) {
				fp0 = p0
			}
			if int(p1)+bs.Org <= len(bs.Text) {
				fp1 = p1
			}
			if fp0 > bs.Frame.Nchars {
				fp0 = bs.Frame.Nchars
			}
			if fp1 > bs.Frame.Nchars {
				fp1 = bs.Frame.Nchars
			}
			bs.Frame.P0, bs.Frame.P1 = fp0, fp1
		}
	} else {
		// Frame is correct — just redraw (full-screen paint cleared our pixels)
		bgColor := uint32(draw.DAcmeYellow)
		if bg := n.Props["bg"]; bg != "" {
			if c := theme.ParseColor(bg); c != 0 {
				bgColor = c
			}
		}
		bodyBg := r.colorImage(bgColor)
		if bodyBg != nil {
			r.Screen.Draw(n.Rect, bodyBg, draw.ZP)
		}
		bs.Frame.Redraw()
		if bs.Frame.P0 != bs.Frame.P1 {
			pt0 := bs.Frame.PtOfChar(bs.Frame.P0)
			bs.Frame.DrawSel(pt0, bs.Frame.P0, bs.Frame.P1, true)
		}
	}

	// Draw left border (subtle)
	bord := r.colorImage(draw.DAcmeBorder)
	if bord != nil {
		left := draw.Rect(n.Rect.Min.X, n.Rect.Min.Y, n.Rect.Min.X+1, n.Rect.Max.Y)
		r.Screen.Draw(left, bord, draw.ZP)
	}
}

// bodyScroll scrolls the body by dl lines. Positive = scroll down.
func (r *Renderer) bodyScroll(bs *BodyState, dl int) {
	if dl == 0 {
		return
	}
	// Estimate characters per line
	charsPerLine := 0
	if bs.Frame.Nchars > 0 && bs.Frame.Nlines > 0 {
		charsPerLine = int(bs.Frame.Nchars) / bs.Frame.Nlines
	}
	if charsPerLine < 1 {
		charsPerLine = 80
	}

	newOrg := bs.Org + dl*charsPerLine
	if newOrg < 0 {
		newOrg = 0
	}
	if newOrg > len(bs.Text) {
		newOrg = len(bs.Text)
	}

	// Snap to line boundaries if scrolling forward
	if newOrg > 0 && newOrg < len(bs.Text) {
		// Find previous newline
		for newOrg > 0 && bs.Text[newOrg-1] != '\n' {
			newOrg--
		}
	}

	bs.Org = newOrg

	// Rebuild frame content
	bs.Frame.Clear(false)
	bs.Frame.Init(bs.Rect, bs.Frame.Font, bs.Frame.B, bs.Frame.Cols)
	bs.Frame.Scroll = func(f *frame.Frame, dl int) {
		r.bodyScroll(bs, dl)
	}
	r.bodyFill(bs)
}

// BodyClick handles a mouse click on a body.
// Returns an action, or nil.
func (r *Renderer) BodyClick(id string, mc *draw.Mousectl, button int) *proto.Action {
	bs, ok := r.Bodies[id]
	if !ok || !bs.Init {
		return nil
	}

	switch button {
	case 1:
		// B1: selection
		bs.Frame.Select(mc)
		return nil

	case 2:
		// B2: execute — find word at click position
		pos := bs.Frame.CharOfPt(mc.Mouse.Point)
		bufPos := int(pos) + bs.Org
		word := wordAt(bs.Text, bufPos)
		if word == "" {
			return nil
		}
		return &proto.Action{
			Kind: "execute",
			KVs: map[string]string{
				"id":   id,
				"text": word,
			},
		}

	case 3:
		// B3: look — find word at click position
		pos := bs.Frame.CharOfPt(mc.Mouse.Point)
		bufPos := int(pos) + bs.Org
		word := wordAt(bs.Text, bufPos)
		if word == "" {
			return nil
		}
		return &proto.Action{
			Kind: "look",
			KVs: map[string]string{
				"id":   id,
				"text": word,
			},
		}
	}
	return nil
}

// BodyType handles typing into a body.
func (r *Renderer) BodyType(id string, key rune) {
	bs, ok := r.Bodies[id]
	if !ok || !bs.Init {
		return
	}

	switch {
	case key == draw.Kbs: // backspace
		q0, q1 := bs.Frame.P0, bs.Frame.P1
		if q0 == q1 {
			if q0 == 0 {
				if bs.Org > 0 {
					// Delete into the off-screen buffer
					bs.Org--
					bs.Text = append(bs.Text[:bs.Org], bs.Text[bs.Org+1:]...)
					bs.Dirty = true
					// Rebuild frame
					r.bodyRebuild(bs)
				}
				return
			}
			q0--
		}
		// Delete in buffer
		bufQ0 := int(q0) + bs.Org
		bufQ1 := int(q1) + bs.Org
		if bufQ0 < 0 {
			bufQ0 = 0
		}
		if bufQ1 > len(bs.Text) {
			bufQ1 = len(bs.Text)
		}
		newText := make([]rune, 0, len(bs.Text))
		newText = append(newText, bs.Text[:bufQ0]...)
		newText = append(newText, bs.Text[bufQ1:]...)
		bs.Text = newText
		bs.Dirty = true
		// Delete in frame
		bs.Frame.Delete(q0, q1)

	case key == '\n' || key == '\t' || (key >= 32 && key < draw.KF): // printable + newline + tab
		q0, q1 := bs.Frame.P0, bs.Frame.P1
		// Delete selection first if any
		if q0 != q1 {
			bufQ0 := int(q0) + bs.Org
			bufQ1 := int(q1) + bs.Org
			newText := make([]rune, 0, len(bs.Text))
			newText = append(newText, bs.Text[:bufQ0]...)
			newText = append(newText, bs.Text[bufQ1:]...)
			bs.Text = newText
			bs.Frame.Delete(q0, q1)
		}
		pos := q0
		bufPos := int(pos) + bs.Org
		// Insert into text buffer
		ch := []rune{key}
		newText := make([]rune, 0, len(bs.Text)+1)
		newText = append(newText, bs.Text[:bufPos]...)
		newText = append(newText, ch...)
		newText = append(newText, bs.Text[bufPos:]...)
		bs.Text = newText
		bs.Dirty = true
		// Insert into frame
		bs.Frame.Insert(ch, pos)

	case key == draw.Kup: // scroll up
		r.bodyScroll(bs, -bs.Frame.Maxlines/2)
	case key == draw.Kdown: // scroll down
		r.bodyScroll(bs, bs.Frame.Maxlines/2)
	case key == draw.Kpgup:
		r.bodyScroll(bs, -bs.Frame.Maxlines)
	case key == draw.Kpgdown:
		r.bodyScroll(bs, bs.Frame.Maxlines)
	case key == draw.Khome:
		r.BodyScrollTo(id, 0)
	case key == draw.Kend:
		r.BodyScrollTo(id, len(bs.Text))
	}
}

// bodyRebuild resets the frame and refills from the current org.
func (r *Renderer) bodyRebuild(bs *BodyState) {
	bs.Frame.Clear(false)
	bs.Frame.Init(bs.Rect, bs.Frame.Font, bs.Frame.B, bs.Frame.Cols)
	bs.Frame.Scroll = func(f *frame.Frame, dl int) {
		r.bodyScroll(bs, dl)
	}
	r.bodyFill(bs)
}

// BodyText returns the complete text in a body.
func (r *Renderer) BodyText(id string) string {
	if bs, ok := r.Bodies[id]; ok {
		return string(bs.Text)
	}
	return ""
}

// SetBodyText replaces the complete text in a body.
func (r *Renderer) SetBodyText(id string, text string) {
	bs := r.ensureBody(id)
	bs.Text = []rune(text)
	bs.Org = 0
	bs.Dirty = false
	if bs.Init {
		r.bodyRebuild(bs)
	}
}

// BodyScrollTo scrolls the body to make the given buffer position visible.
func (r *Renderer) BodyScrollTo(id string, pos int) {
	bs, ok := r.Bodies[id]
	if !ok {
		return
	}
	if pos < 0 {
		pos = 0
	}
	if pos > len(bs.Text) {
		pos = len(bs.Text)
	}
	// Snap to line start
	for pos > 0 && bs.Text[pos-1] != '\n' {
		pos--
	}
	bs.Org = pos
	if bs.Init {
		r.bodyRebuild(bs)
	}
}

// BodyDirty returns whether a body has been modified.
func (r *Renderer) BodyDirty(id string) bool {
	if bs, ok := r.Bodies[id]; ok {
		return bs.Dirty
	}
	return false
}

// BodyClean marks a body as clean (after saving, etc).
func (r *Renderer) BodyClean(id string) {
	if bs, ok := r.Bodies[id]; ok {
		bs.Dirty = false
	}
}

// BodySelection returns the selected text in a body (buffer coordinates).
func (r *Renderer) BodySelection(id string) string {
	bs, ok := r.Bodies[id]
	if !ok || !bs.Init {
		return ""
	}
	q0 := int(bs.Frame.P0) + bs.Org
	q1 := int(bs.Frame.P1) + bs.Org
	if q0 < 0 {
		q0 = 0
	}
	if q1 > len(bs.Text) {
		q1 = len(bs.Text)
	}
	if q0 >= q1 {
		return ""
	}
	return string(bs.Text[q0:q1])
}

// Unused import suppressor
var _ = theme.ParseColor
var _ = unicode.IsSpace
