// tag.go implements Acme-style tag bars backed by the frame package.
//
// A tag is an editable text area using frame.Frame. Middle-click (B2)
// on a word in the tag executes it. Right-click (B3) looks it up.
// The tag text buffer is maintained by the renderer.
package render

import (
	"unicode"

	"github.com/elizafairlady/go-libui/draw"
	"github.com/elizafairlady/go-libui/frame"
	"github.com/elizafairlady/go-libui/ui/layout"
	"github.com/elizafairlady/go-libui/ui/proto"
	"github.com/elizafairlady/go-libui/ui/theme"
)

// TagState holds the state for a tag bar frame.
type TagState struct {
	Frame *frame.Frame
	Text  []rune // full text buffer
	Rect  draw.Rectangle
	Init  bool
}

// ensureTag ensures a TagState exists for the given node ID.
func (r *Renderer) ensureTag(id string) *TagState {
	if r.Tags == nil {
		r.Tags = make(map[string]*TagState)
	}
	if ts, ok := r.Tags[id]; ok {
		return ts
	}
	ts := &TagState{
		Frame: &frame.Frame{},
	}
	r.Tags[id] = ts
	return ts
}

// initTag initializes or reinitializes the frame for a tag node.
func (r *Renderer) initTag(ts *TagState, n *layout.RNode) {
	rect := n.Rect
	// Allocate colors for the tag
	cols := r.tagColors()

	ts.Frame.Init(rect, r.Font, r.Screen, cols)
	ts.Rect = rect
	ts.Init = true

	// Insert current text
	if len(ts.Text) > 0 {
		ts.Frame.Insert(ts.Text, 0)
	}
}

// tagColors returns the Acme-style tag colors.
func (r *Renderer) tagColors() [frame.NCol]*draw.Image {
	tagBg := r.colorImage(draw.DAcmeCyan)
	high := r.colorImage(draw.DAcmeHigh)
	bord := r.colorImage(draw.DAcmeBorder)
	text := r.colorImage(draw.DAcmeText)
	htext := r.colorImage(draw.DAcmeText)

	if tagBg == nil {
		tagBg = r.Theme.BgImage
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
		frame.ColBack:  tagBg,
		frame.ColHigh:  high,
		frame.ColBord:  bord,
		frame.ColText:  text,
		frame.ColHText: htext,
	}
}

// paintTag renders a tag node using the frame package.
func (r *Renderer) paintTag(n *layout.RNode) {
	ts := r.ensureTag(n.ID)

	// Only set initial text from tree props on first init.
	// After that, renderer owns the tag text (user-editable).
	if !ts.Init {
		text := n.Props["text"]
		ts.Text = []rune(text)
	}

	// First time or rect changed: full init
	if !ts.Init || ts.Rect != n.Rect {
		p0, p1 := ts.Frame.P0, ts.Frame.P1
		ts.Frame.Clear(false)
		r.initTag(ts, n)
		// Restore selection
		if ts.Frame.Nchars > 0 {
			if p0 > ts.Frame.Nchars {
				p0 = ts.Frame.Nchars
			}
			if p1 > ts.Frame.Nchars {
				p1 = ts.Frame.Nchars
			}
			ts.Frame.P0, ts.Frame.P1 = p0, p1
		}
	} else {
		// Frame is already correct — just redraw its content
		// (The full-screen paint cleared our pixels, so we need to repaint)
		tagBg := r.colorImage(draw.DAcmeCyan)
		if tagBg != nil {
			r.Screen.Draw(n.Rect, tagBg, draw.ZP)
		}
		ts.Frame.Redraw()
		// Redraw selection highlight if any
		if ts.Frame.P0 != ts.Frame.P1 {
			pt0 := ts.Frame.PtOfChar(ts.Frame.P0)
			ts.Frame.DrawSel(pt0, ts.Frame.P0, ts.Frame.P1, true)
		}
	}

	// Draw bottom border
	bord := r.colorImage(draw.DAcmeBorder)
	if bord != nil {
		bot := draw.Rect(n.Rect.Min.X, n.Rect.Max.Y-1, n.Rect.Max.X, n.Rect.Max.Y)
		r.Screen.Draw(bot, bord, draw.ZP)
	}
}

// TagClick handles a mouse click on a tag.
// button is 1, 2, or 3. Returns an action, or nil.
func (r *Renderer) TagClick(id string, mc *draw.Mousectl, button int) *proto.Action {
	ts, ok := r.Tags[id]
	if !ok || !ts.Init {
		return nil
	}

	switch button {
	case 1:
		// B1: selection — use frame.Select
		ts.Frame.Select(mc)
		return nil

	case 2:
		// B2: execute — find word at click position
		pos := ts.Frame.CharOfPt(mc.Mouse.Point)
		word := wordAt(ts.Text, int(pos))
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
		pos := ts.Frame.CharOfPt(mc.Mouse.Point)
		word := wordAt(ts.Text, int(pos))
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

// TagType handles typing into a tag.
func (r *Renderer) TagType(id string, key rune) {
	ts, ok := r.Tags[id]
	if !ok || !ts.Init {
		return
	}

	switch {
	case key == draw.Kbs: // backspace
		if ts.Frame.P0 > 0 {
			if ts.Frame.P0 == ts.Frame.P1 {
				ts.Frame.P0--
			}
			ts.Frame.Delete(ts.Frame.P0, ts.Frame.P1)
			// Update text buffer
			ts.Text = append(ts.Text[:ts.Frame.P0], ts.Text[ts.Frame.P1:]...)
		}
	case key >= 32 && key < draw.KF: // printable
		r := []rune{key}
		// Delete selection first if any
		if ts.Frame.P0 != ts.Frame.P1 {
			ts.Text = append(ts.Text[:ts.Frame.P0], ts.Text[ts.Frame.P1:]...)
			ts.Frame.Delete(ts.Frame.P0, ts.Frame.P1)
		}
		pos := ts.Frame.P0
		// Insert into text buffer
		newText := make([]rune, 0, len(ts.Text)+1)
		newText = append(newText, ts.Text[:pos]...)
		newText = append(newText, r...)
		newText = append(newText, ts.Text[pos:]...)
		ts.Text = newText
		// Insert into frame
		ts.Frame.Insert(r, pos)
	}
}

// TagText returns the current text in a tag.
func (r *Renderer) TagText(id string) string {
	if ts, ok := r.Tags[id]; ok {
		return string(ts.Text)
	}
	return ""
}

// wordAt extracts the word at position pos in the rune slice.
// Words are delimited by whitespace.
func wordAt(text []rune, pos int) string {
	if pos < 0 || pos >= len(text) {
		if pos == len(text) && pos > 0 {
			pos = pos - 1 // click at end of text, select last word
		} else {
			return ""
		}
	}
	// Skip if on whitespace, try to go right
	if unicode.IsSpace(text[pos]) {
		return ""
	}
	// Find word boundaries
	start := pos
	for start > 0 && !unicode.IsSpace(text[start-1]) {
		start--
	}
	end := pos
	for end < len(text) && !unicode.IsSpace(text[end]) {
		end++
	}
	return string(text[start:end])
}

func runesEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ParseColor re-export for tag colors
var _ = theme.ParseColor
