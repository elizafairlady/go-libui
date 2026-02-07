// body.go implements multi-line editable text areas backed by frame.Frame.
//
// A body is the Acme-style editable text area — a viewport into a
// potentially large rune buffer. The frame only displays what fits;
// scrolling adjusts the origin (org). Text is owned by a window.Buffer,
// not by the renderer — the renderer is just a view into the buffer.
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
	"github.com/elizafairlady/go-libui/ui/text"
	"github.com/elizafairlady/go-libui/ui/theme"
)

// BodyState holds the renderer state for a body text area.
// Text is owned by Buf (a *text.Buffer). The buffer may be
// standalone (renderer-owned) or provided externally via
// the BufferProvider interface.
type BodyState struct {
	Frame *frame.Frame
	Buf   *text.Buffer   // the authoritative text
	Org   int            // first rune visible in frame
	Rect  draw.Rectangle // current layout rect
	Init  bool           // has been initialized
	seq   int            // last seq we synced from buffer
}

// ensureBody ensures a BodyState exists for the given node ID.
// If a Buffer is provided (from BodyBufferProvider), it's used;
// otherwise a standalone Buffer is created.
func (r *Renderer) ensureBody(id string, buf *text.Buffer) *BodyState {
	if r.Bodies == nil {
		r.Bodies = make(map[string]*BodyState)
	}
	bs, ok := r.Bodies[id]
	if ok {
		// If buffer changed (e.g. provider returned a different one), update it
		if buf != nil && bs.Buf != buf {
			bs.Buf = buf
			bs.seq = -1 // force re-sync
		}
		return bs
	}
	if buf == nil {
		buf = &text.Buffer{}
	}
	bs = &BodyState{
		Frame: &frame.Frame{},
		Buf:   buf,
		seq:   -1,
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
	runes := bs.Buf.Runes()
	if bs.Org > len(runes) {
		bs.Org = len(runes)
	}
	visible := runes[bs.Org:]
	if len(visible) > 0 {
		bs.Frame.Insert(visible, 0)
	}
	bs.seq = bs.Buf.Seq()
}

// paintBody renders a body node using the frame package.
func (r *Renderer) paintBody(n *layout.RNode) {
	// Ask the BufferProvider for an external buffer, if available
	var buf *text.Buffer
	if r.BufferProvider != nil {
		buf = r.BufferProvider.BodyBuffer(n.ID, n.Props)
	}

	bs := r.ensureBody(n.ID, buf)

	// On first init, seed buffer from tree props if buffer is empty
	if !bs.Init && bs.Buf.Nc() == 0 {
		if text := n.Props["text"]; text != "" {
			bs.Buf.SetAll(text)
		}
	}

	// Check if buffer was externally modified (e.g. by ctl write, Get, etc.)
	if bs.seq != bs.Buf.Seq() && bs.Init {
		p0, p1 := bs.Frame.P0, bs.Frame.P1
		bs.Frame.Clear(false)
		r.initBody(bs, n)
		// Try to restore selection
		if bs.Frame.Nchars > 0 {
			if p0 > bs.Frame.Nchars {
				p0 = bs.Frame.Nchars
			}
			if p1 > bs.Frame.Nchars {
				p1 = bs.Frame.Nchars
			}
			bs.Frame.P0, bs.Frame.P1 = p0, p1
		}
		return
	}

	// First time or rect changed: full reinit
	if !bs.Init || bs.Rect != n.Rect {
		p0, p1 := bs.Frame.P0, bs.Frame.P1
		bs.Frame.Clear(false)
		r.initBody(bs, n)
		// Restore selection (in frame coordinates)
		if bs.Frame.Nchars > 0 {
			if p0 > bs.Frame.Nchars {
				p0 = bs.Frame.Nchars
			}
			if p1 > bs.Frame.Nchars {
				p1 = bs.Frame.Nchars
			}
			bs.Frame.P0, bs.Frame.P1 = p0, p1
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
	runes := bs.Buf.Runes()

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
	if newOrg > len(runes) {
		newOrg = len(runes)
	}

	// Snap to line boundaries if scrolling forward
	if newOrg > 0 && newOrg < len(runes) {
		for newOrg > 0 && runes[newOrg-1] != '\n' {
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
		word := wordAt(bs.Buf.Runes(), bufPos)
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
		word := wordAt(bs.Buf.Runes(), bufPos)
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

// BodyType handles typing into a body. Edits go directly into the Buffer.
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
					bs.Buf.Delete(bs.Org, bs.Org+1)
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
		if bufQ1 > bs.Buf.Nc() {
			bufQ1 = bs.Buf.Nc()
		}
		bs.Buf.Delete(bufQ0, bufQ1)
		// Delete in frame
		bs.Frame.Delete(q0, q1)
		bs.seq = bs.Buf.Seq()

	case key == '\n' || key == '\t' || (key >= 32 && key < draw.KF): // printable + newline + tab
		q0, q1 := bs.Frame.P0, bs.Frame.P1
		// Delete selection first if any
		if q0 != q1 {
			bufQ0 := int(q0) + bs.Org
			bufQ1 := int(q1) + bs.Org
			bs.Buf.Delete(bufQ0, bufQ1)
			bs.Frame.Delete(q0, q1)
		}
		pos := q0
		bufPos := int(pos) + bs.Org
		// Insert into buffer
		ch := []rune{key}
		bs.Buf.Insert(bufPos, ch)
		// Insert into frame
		bs.Frame.Insert(ch, pos)
		bs.seq = bs.Buf.Seq()

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
		r.BodyScrollTo(id, bs.Buf.Nc())
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

// BodyText returns the complete text in a body's buffer.
func (r *Renderer) BodyText(id string) string {
	if bs, ok := r.Bodies[id]; ok {
		return bs.Buf.ReadAll()
	}
	return ""
}

// SetBodyText replaces the complete text in a body's buffer.
func (r *Renderer) SetBodyText(id string, text string) {
	bs := r.ensureBody(id, nil)
	bs.Buf.SetAll(text)
	bs.Buf.Clean()
	bs.Org = 0
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
	runes := bs.Buf.Runes()
	if pos < 0 {
		pos = 0
	}
	if pos > len(runes) {
		pos = len(runes)
	}
	// Snap to line start
	for pos > 0 && runes[pos-1] != '\n' {
		pos--
	}
	bs.Org = pos
	if bs.Init {
		r.bodyRebuild(bs)
	}
}

// BodyDirty returns whether a body's buffer has been modified.
func (r *Renderer) BodyDirty(id string) bool {
	if bs, ok := r.Bodies[id]; ok {
		return bs.Buf.Dirty()
	}
	return false
}

// BodyClean marks a body's buffer as clean (after saving, etc).
func (r *Renderer) BodyClean(id string) {
	if bs, ok := r.Bodies[id]; ok {
		bs.Buf.Clean()
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
	nc := bs.Buf.Nc()
	if q1 > nc {
		q1 = nc
	}
	if q0 >= q1 {
		return ""
	}
	return bs.Buf.ReadRange(q0, q1)
}

// Unused import suppressor
var _ = theme.ParseColor
var _ = unicode.IsSpace
