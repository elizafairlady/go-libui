// Acme is a text editor modelled on Plan 9's acme(1), built with the
// ui framework. Windows are acme-style: each has a tag (editable) and
// a body (editable text area). Body and tag text live in Buffer files,
// matching the real acme filesystem model (see /sys/src/cmd/acme/dat.h).
//
// Usage: acme [file ...]
//
// B1: select text
// B2: execute command word
// B3: look — open file or search for text
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/elizafairlady/go-libui/ui"
	"github.com/elizafairlady/go-libui/ui/cmd/acme/window"
	"github.com/elizafairlady/go-libui/ui/proto"
	"github.com/elizafairlady/go-libui/ui/text"
	"github.com/elizafairlady/go-libui/ui/view"
)

// acmeApp owns a window.Row — the authoritative data store for all
// columns, windows, body text, and tag text. This matches how real
// acme works: the Row is the root, it contains Columns, each Column
// contains Windows, each Window has Tag and Body buffers exposed as
// files in the per-window directory.
type acmeApp struct {
	row         *window.Row
	initialized bool
}

// --- Executor interface ---

func (a *acmeApp) Builtins() map[string]view.Builtin {
	return map[string]view.Builtin{
		"Newcol": a.cmdNewcol,
		"Exit":   a.cmdExit,
		"Putall": a.cmdPutall,
		"New":    a.cmdNew,
		"Delcol": a.cmdDelcol,
		"Del":    a.cmdDel,
		"Get":    a.cmdGet,
		"Put":    a.cmdPut,
		"Cut":    a.cmdCut,
		"Snarf":  a.cmdSnarf,
		"Paste":  a.cmdPaste,
	}
}

func (a *acmeApp) BinDirs() []string { return nil }

// --- Builtins ---

func (a *acmeApp) cmdNewcol(ctx *view.ExecContext) error {
	a.row.NewColumn()
	ctx.State.Set("_rev", nextRev(ctx.State)) // trigger re-render
	return nil
}

func (a *acmeApp) cmdExit(ctx *view.ExecContext) error {
	ctx.State.Set("_quit", "1")
	return nil
}

func (a *acmeApp) cmdPutall(ctx *view.ExecContext) error {
	for _, c := range a.row.Cols {
		for _, w := range c.Windows {
			if w.Name != "" && !w.IsScratch && w.Body.Dirty() {
				a.putWindow(w)
			}
		}
	}
	ctx.State.Set("_rev", nextRev(ctx.State))
	return nil
}

func (a *acmeApp) cmdNew(ctx *view.ExecContext) error {
	col := a.colFromID(ctx.ID)
	if col == nil {
		if len(a.row.Cols) == 0 {
			return fmt.Errorf("no columns")
		}
		col = a.row.Cols[0]
	}
	w := a.row.NewWindow(col)
	w.Name = "scratch"
	w.IsScratch = true
	w.Tag.SetAll(w.Name + " Del Snarf Get Put Look |")
	ctx.State.Set("_rev", nextRev(ctx.State))
	return nil
}

func (a *acmeApp) cmdDelcol(ctx *view.ExecContext) error {
	col := a.colFromID(ctx.ID)
	if col == nil {
		return fmt.Errorf("no column context")
	}
	a.row.CloseColumn(col)
	ctx.State.Set("_rev", nextRev(ctx.State))
	return nil
}

func (a *acmeApp) cmdDel(ctx *view.ExecContext) error {
	w := a.winFromID(ctx.ID)
	if w == nil {
		return fmt.Errorf("no window context")
	}
	a.row.CloseWindow(w)
	ctx.State.Set("_rev", nextRev(ctx.State))
	return nil
}

func (a *acmeApp) cmdGet(ctx *view.ExecContext) error {
	w := a.winFromID(ctx.ID)
	if w == nil {
		return fmt.Errorf("no window context")
	}
	if w.Name == "" || w.IsScratch {
		return fmt.Errorf("no file name")
	}
	data, err := os.ReadFile(w.Name)
	if err != nil {
		return err
	}
	w.Body.SetAll(string(data))
	w.Body.Clean()
	ctx.State.Set("_rev", nextRev(ctx.State))
	return nil
}

func (a *acmeApp) cmdPut(ctx *view.ExecContext) error {
	w := a.winFromID(ctx.ID)
	if w == nil {
		return fmt.Errorf("no window context")
	}
	if err := a.putWindow(w); err != nil {
		return err
	}
	ctx.State.Set("_rev", nextRev(ctx.State))
	return nil
}

func (a *acmeApp) cmdCut(ctx *view.ExecContext) error {
	w := a.winFromID(ctx.ID)
	if w == nil {
		return fmt.Errorf("no window context")
	}
	a.row.Cut(w)
	ctx.State.Set("_rev", nextRev(ctx.State))
	return nil
}

func (a *acmeApp) cmdSnarf(ctx *view.ExecContext) error {
	w := a.winFromID(ctx.ID)
	if w == nil {
		return fmt.Errorf("no window context")
	}
	a.row.Snarf(w)
	return nil
}

func (a *acmeApp) cmdPaste(ctx *view.ExecContext) error {
	w := a.winFromID(ctx.ID)
	if w == nil {
		return fmt.Errorf("no window context")
	}
	a.row.Paste(w)
	ctx.State.Set("_rev", nextRev(ctx.State))
	return nil
}

func (a *acmeApp) putWindow(w *window.Window) error {
	if w.Name == "" || w.IsScratch {
		return fmt.Errorf("no file name")
	}
	text := w.Body.ReadAll()
	if err := os.WriteFile(w.Name, []byte(text), 0644); err != nil {
		return err
	}
	w.Body.Clean()
	return nil
}

// --- View ---
// View reads entirely from the Row to build the UI tree.
// State is only used for triggering re-renders (via _rev).

func (a *acmeApp) View(s view.State) *view.Node {
	if !a.initialized {
		a.init(s)
	}

	var colNodes []*view.Node
	for _, col := range a.row.Cols {
		colNodes = append(colNodes, a.viewColumn(col))
	}

	if len(colNodes) == 0 {
		colNodes = append(colNodes,
			view.VBox("empty",
				view.TextNode("empty-text", "B2 on Newcol to create a column").
					Prop("fg", "acmedim").PropInt("pad", 20),
			).Prop("flex", "1"),
		)
	}

	// Read _rev to subscribe to changes (pure side-effect for reactivity)
	_ = s.Get("_rev")

	return view.VBox("root",
		view.Tag("row-tag", "Newcol Kill Putall Dump Load Exit").PropInt("pad", 0),
		view.Rect("row-sep").Prop("bg", "acmeborder").PropInt("minh", 1).PropInt("maxh", 1),
		view.SplitBox("row-cols", colNodes...).
			Prop("direction", "horizontal").
			Prop("flex", "1"),
	).PropInt("pad", 0).PropInt("gap", 0)
}

func (a *acmeApp) viewColumn(col *window.Column) *view.Node {
	cid := strconv.Itoa(col.ID)

	var winNodes []*view.Node
	for _, w := range col.Windows {
		winNodes = append(winNodes, a.viewWindow(cid, w))
	}

	var children []*view.Node
	children = append(children,
		view.Tag("ct-"+cid, "New Cut Paste Snarf Zerox Delcol").
			PropInt("pad", 0).Prop("bg", "acmetag"),
		view.Rect("cs-"+cid).Prop("bg", "acmeborder").
			PropInt("minh", 1).PropInt("maxh", 1),
	)

	if len(winNodes) > 0 {
		children = append(children,
			view.SplitBox("cw-"+cid, winNodes...).
				Prop("direction", "vertical").
				Prop("flex", "1"),
		)
	} else {
		children = append(children,
			view.VBox("ce-"+cid,
				view.TextNode("ce-text-"+cid, "B2 New").
					Prop("fg", "acmedim").PropInt("pad", 10),
			).Prop("flex", "1").Prop("bg", "acmeyellow"),
		)
	}

	return view.VBox("col-"+cid, children...).
		PropInt("pad", 0).PropInt("gap", 0).Prop("flex", "1")
}

func (a *acmeApp) viewWindow(cid string, w *window.Window) *view.Node {
	wid := strconv.Itoa(w.ID)

	displayName := w.Name
	if w.Body.Dirty() {
		displayName += "+"
	}
	tagText := displayName + " Del Snarf Get Put Look |"

	return view.VBox("win-"+cid+"-"+wid,
		view.Tag("wt-"+cid+"-"+wid, tagText).PropInt("pad", 0),
		view.Rect("ws-"+cid+"-"+wid).Prop("bg", "acmeborder").
			PropInt("minh", 1).PropInt("maxh", 1),
		view.Body("wb-"+cid+"-"+wid).
			Prop("flex", "1").
			PropInt("winid", w.ID),
	).PropInt("pad", 0).PropInt("gap", 0).Prop("flex", "1")
}

// --- Handle ---

func (a *acmeApp) Handle(s view.State, act *proto.Action) {
	switch act.Kind {
	case "look":
		a.handleLook(s, act)
	case "bodychange":
		// Body text changed by user typing — sync from renderer to buffer
		a.handleBodyChange(s, act)
	}
}

func (a *acmeApp) handleLook(s view.State, act *proto.Action) {
	text := act.KVs["text"]
	if text == "" {
		return
	}
	info, err := os.Stat(text)
	if err == nil && !info.IsDir() {
		if len(a.row.Cols) == 0 {
			a.row.NewColumn()
		}
		col := a.row.Cols[0]
		w := a.row.NewWindow(col)
		w.Name = text
		data, err := os.ReadFile(text)
		if err == nil {
			w.Body.SetAll(string(data))
			w.Body.Clean()
		}
		w.Tag.SetAll(text + " Del Snarf Get Put Look |")
		s.Set("_rev", nextRev(s))
	}
}

func (a *acmeApp) handleBodyChange(s view.State, act *proto.Action) {
	widStr := act.KVs["winid"]
	if widStr == "" {
		return
	}
	wid, err := strconv.Atoi(widStr)
	if err != nil {
		return
	}
	w := a.row.LookID(wid)
	if w == nil {
		return
	}
	// The new text comes from the renderer
	if text, ok := act.KVs["text"]; ok {
		w.Body.SetAll(text)
	}
}

// --- ID parsing ---

func (a *acmeApp) colFromID(id string) *window.Column {
	// Extract column ID from node IDs like "ct-0", "wt-0-1", "wb-0-1"
	var cidStr string
	if len(id) > 3 {
		switch {
		case id[:3] == "ct-":
			cidStr = id[3:]
		case id[:3] == "wt-" || id[:3] == "wb-":
			for i := 3; i < len(id); i++ {
				if id[i] == '-' {
					cidStr = id[3:i]
					break
				}
			}
		}
	}
	if cidStr == "" {
		return nil
	}
	cid, err := strconv.Atoi(cidStr)
	if err != nil {
		return nil
	}
	for _, c := range a.row.Cols {
		if c.ID == cid {
			return c
		}
	}
	return nil
}

func (a *acmeApp) winFromID(id string) *window.Window {
	// Extract winid from node IDs like "wt-0-1", "wb-0-1"
	if len(id) < 5 {
		return nil
	}
	prefix := id[:3]
	if prefix != "wt-" && prefix != "wb-" {
		return nil
	}
	rest := id[3:]
	dashIdx := -1
	for i := 0; i < len(rest); i++ {
		if rest[i] == '-' {
			dashIdx = i
			break
		}
	}
	if dashIdx < 0 {
		return nil
	}
	widStr := rest[dashIdx+1:]
	wid, err := strconv.Atoi(widStr)
	if err != nil {
		return nil
	}
	return a.row.LookID(wid)
}

// --- Helpers ---

func nextRev(s view.State) string {
	n, _ := strconv.Atoi(s.Get("_rev"))
	return strconv.Itoa(n + 1)
}

// --- Initialization ---

func (a *acmeApp) init(s view.State) {
	a.initialized = true
	col := a.row.NewColumn()

	if len(os.Args) > 1 {
		for _, f := range os.Args[1:] {
			w := a.row.NewWindow(col)
			w.Name = f
			data, err := os.ReadFile(f)
			if err == nil {
				w.Body.SetAll(string(data))
				w.Body.Clean()
			}
			w.Tag.SetAll(f + " Del Snarf Get Put Look |")
		}
	} else {
		w := a.row.NewWindow(col)
		w.Name = "scratch"
		w.IsScratch = true
		w.Tag.SetAll("scratch Del Snarf Get Put Look |")
	}

	s.Set("_rev", "1")
}

// --- BodyBufferProvider interface ---
// Implements render.BodyBufferProvider so that body nodes with a "winid"
// prop share the Window's body buffer with the renderer.

func (a *acmeApp) BodyBuffer(nodeID string, props map[string]string) *text.Buffer {
	if widStr := props["winid"]; widStr != "" {
		wid, err := strconv.Atoi(widStr)
		if err == nil {
			if w := a.row.LookID(wid); w != nil {
				return &w.Body
			}
		}
	}
	return nil
}

// --- Main ---

func main() {
	app := &acmeApp{
		row: window.NewRow(),
	}
	if err := ui.Run("Acme", app); err != nil {
		log.Fatal(err)
	}
}
