// Acme is a text editor modelled on Plan 9's acme(1), built entirely
// with the ui framework. It demonstrates that the framework can express
// Acme's core interaction model: editable tags, multi-line bodies,
// columns with resizable windows, B2 execute, B3 look, file I/O.
//
// Tags are editable text — any word you type can be B2-executed.
// The app registers builtins (New, Del, Get, Put, Newcol, Delcol, Exit);
// anything else is looked up as an external command in $PATH.
// Type "date" in a tag and B2 it — it runs.
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
	"sort"
	"strconv"
	"strings"

	"github.com/elizafairlady/go-libui/ui"
	"github.com/elizafairlady/go-libui/ui/proto"
	"github.com/elizafairlady/go-libui/ui/view"
)

type acmeApp struct{}

// --- Executor interface ---

// Builtins returns the built-in commands. The framework calls these
// on B2 before trying external commands.
func (a *acmeApp) Builtins() map[string]view.Builtin {
	return map[string]view.Builtin{
		// Row-level
		"Newcol": a.cmdNewcol,
		"Exit":   a.cmdExit,
		"Putall": a.cmdPutall,

		// Column-level
		"New":    a.cmdNew,
		"Delcol": a.cmdDelcol,

		// Window-level
		"Del": a.cmdDel,
		"Get": a.cmdGet,
		"Put": a.cmdPut,
	}
}

// BinDirs returns additional paths to search for external commands.
func (a *acmeApp) BinDirs() []string {
	return nil // could add $home/bin/acme here
}

// --- Builtin implementations ---

func (a *acmeApp) cmdNewcol(ctx *view.ExecContext) error {
	next := nextID(ctx.State, "next-col")
	ctx.State.Set("cols/"+next+"/name", "")
	return nil
}

func (a *acmeApp) cmdExit(ctx *view.ExecContext) error {
	ctx.State.Set("_quit", "1")
	return nil
}

func (a *acmeApp) cmdPutall(ctx *view.ExecContext) error {
	s := ctx.State
	for _, cid := range s.List("cols") {
		for _, wid := range s.List("cols/" + cid + "/wins") {
			name := s.Get("cols/" + cid + "/wins/" + wid + "/name")
			if name != "" && name != "scratch" {
				bodyID := "wb-" + cid + "-" + wid
				if s.Get("_body/"+bodyID+"/dirty") == "1" {
					a.putFileByIDs(s, cid, wid)
				}
			}
		}
	}
	return nil
}

func (a *acmeApp) cmdNew(ctx *view.ExecContext) error {
	colID := parseColID(ctx.ID)
	if colID == "" {
		// If executed from row tag, use first column
		cols := sortedIntList(ctx.State.List("cols"))
		if len(cols) == 0 {
			return fmt.Errorf("no columns")
		}
		colID = cols[0]
	}
	a.newWindow(ctx.State, colID, "")
	return nil
}

func (a *acmeApp) cmdDelcol(ctx *view.ExecContext) error {
	colID := parseColID(ctx.ID)
	if colID == "" {
		return fmt.Errorf("no column context")
	}
	s := ctx.State
	for _, wid := range s.List("cols/" + colID + "/wins") {
		s.Del("cols/" + colID + "/wins/" + wid + "/name")
		s.Del("cols/" + colID + "/wins/" + wid + "/dirty")
	}
	s.Del("cols/" + colID + "/name")
	return nil
}

func (a *acmeApp) cmdDel(ctx *view.ExecContext) error {
	colID, winID := parseWinID(ctx.ID)
	if colID == "" || winID == "" {
		return fmt.Errorf("no window context")
	}
	ctx.State.Del("cols/" + colID + "/wins/" + winID + "/name")
	ctx.State.Del("cols/" + colID + "/wins/" + winID + "/dirty")
	return nil
}

func (a *acmeApp) cmdGet(ctx *view.ExecContext) error {
	colID, winID := parseWinID(ctx.ID)
	if colID == "" || winID == "" {
		return fmt.Errorf("no window context")
	}
	s := ctx.State
	name := s.Get("cols/" + colID + "/wins/" + winID + "/name")
	if name == "" || name == "scratch" {
		return fmt.Errorf("no file name")
	}
	data, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	bodyID := "wb-" + colID + "-" + winID
	s.Set("_body/"+bodyID, string(data))
	s.Set("_body/"+bodyID+"/clean", "1")
	s.Set("cols/"+colID+"/wins/"+winID+"/dirty", "0")
	return nil
}

func (a *acmeApp) cmdPut(ctx *view.ExecContext) error {
	colID, winID := parseWinID(ctx.ID)
	if colID == "" || winID == "" {
		return fmt.Errorf("no window context")
	}
	return a.putFileByIDs(ctx.State, colID, winID)
}

func (a *acmeApp) putFileByIDs(s view.State, colID, winID string) error {
	name := s.Get("cols/" + colID + "/wins/" + winID + "/name")
	if name == "" || name == "scratch" {
		return fmt.Errorf("no file name")
	}
	bodyID := "wb-" + colID + "-" + winID
	text := s.Get("_body/" + bodyID)
	err := os.WriteFile(name, []byte(text), 0644)
	if err != nil {
		return err
	}
	s.Set("_body/"+bodyID+"/clean", "1")
	s.Set("cols/"+colID+"/wins/"+winID+"/dirty", "0")
	return nil
}

// --- View ---

func (a *acmeApp) View(s view.State) *view.Node {
	a.viewInit(s)

	colIDs := sortedIntList(s.List("cols"))

	var colNodes []*view.Node
	for _, cid := range colIDs {
		colNodes = append(colNodes, a.viewColumn(s, cid))
	}

	if len(colNodes) == 0 {
		colNodes = append(colNodes,
			view.VBox("empty",
				view.TextNode("empty-text", "B2 on Newcol to create a column").
					Prop("fg", "acmedim").PropInt("pad", 20),
			).Prop("flex", "1"),
		)
	}

	return view.VBox("root",
		view.Tag("row-tag", "Newcol Kill Putall Dump Load Exit").PropInt("pad", 2),
		view.Rect("row-sep").Prop("bg", "acmeborder").PropInt("minh", 1).PropInt("maxh", 1),
		view.SplitBox("row-cols", colNodes...).
			Prop("direction", "horizontal").
			Prop("flex", "1"),
	).PropInt("pad", 0).PropInt("gap", 0)
}

func (a *acmeApp) viewColumn(s view.State, cid string) *view.Node {
	winIDs := sortedIntList(s.List("cols/" + cid + "/wins"))

	var winNodes []*view.Node
	for _, wid := range winIDs {
		winNodes = append(winNodes, a.viewWindow(s, cid, wid))
	}

	var children []*view.Node
	children = append(children,
		view.Tag("ct-"+cid, "New Cut Paste Snarf Zerox Delcol").
			PropInt("pad", 2).Prop("bg", "acmetag"),
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

func (a *acmeApp) viewWindow(s view.State, cid, wid string) *view.Node {
	name := s.Get("cols/" + cid + "/wins/" + wid + "/name")
	dirty := s.Get("cols/"+cid+"/wins/"+wid+"/dirty") == "1"

	displayName := name
	if dirty {
		displayName += "+"
	}
	tagText := displayName + " Del Snarf Get Put Look |"

	return view.VBox("win-"+cid+"-"+wid,
		view.Tag("wt-"+cid+"-"+wid, tagText).PropInt("pad", 2),
		view.Rect("ws-"+cid+"-"+wid).Prop("bg", "acmeborder").
			PropInt("minh", 1).PropInt("maxh", 1),
		view.Body("wb-"+cid+"-"+wid).Prop("flex", "1"),
	).PropInt("pad", 0).PropInt("gap", 0).Prop("flex", "1")
}

// --- Handle ---
// Handle still receives non-execute actions (look, cmdoutput, cmderror, etc.)
// Execute actions only arrive here if not handled by a builtin or external cmd.

func (a *acmeApp) Handle(s view.State, act *proto.Action) {
	switch act.Kind {
	case "look":
		a.handleLook(s, act)
	case "cmdoutput":
		// Output from an external command — append to body
		// For now, just ignore (TODO: append to body or +Errors)
	case "cmderror":
		// Error from an external command
		// TODO: show in +Errors window
	case "execute":
		// Fallthrough from framework — command wasn't a builtin or external.
		// Could still be something the app wants to handle.
	}
}

func (a *acmeApp) handleLook(s view.State, act *proto.Action) {
	text := act.KVs["text"]
	if text == "" {
		return
	}
	info, err := os.Stat(text)
	if err == nil && !info.IsDir() {
		colIDs := sortedIntList(s.List("cols"))
		if len(colIDs) == 0 {
			next := nextID(s, "next-col")
			s.Set("cols/"+next+"/name", "")
			colIDs = sortedIntList(s.List("cols"))
		}
		if len(colIDs) > 0 {
			a.newWindow(s, colIDs[0], text)
		}
	}
}

func (a *acmeApp) newWindow(s view.State, colID string, filename string) {
	next := nextID(s, "next-win")
	prefix := "cols/" + colID + "/wins/" + next

	name := filename
	if name == "" {
		name = "scratch"
	}
	s.Set(prefix+"/name", name)
	s.Set(prefix+"/dirty", "0")

	if filename != "" {
		data, err := os.ReadFile(filename)
		if err == nil {
			bodyID := "wb-" + colID + "-" + next
			s.Set("_body/"+bodyID, string(data))
			s.Set("_body/"+bodyID+"/clean", "1")
		}
	}
}

// --- ID parsing ---

func parseColID(id string) string {
	if strings.HasPrefix(id, "ct-") {
		return id[3:]
	}
	if strings.HasPrefix(id, "wt-") || strings.HasPrefix(id, "wb-") {
		parts := strings.SplitN(id[3:], "-", 2)
		if len(parts) >= 1 {
			return parts[0]
		}
	}
	return ""
}

func parseWinID(id string) (colID, winID string) {
	if strings.HasPrefix(id, "wt-") || strings.HasPrefix(id, "wb-") {
		parts := strings.SplitN(id[3:], "-", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}
	return "", ""
}

// --- Helpers ---

func nextID(s view.State, key string) string {
	n, _ := strconv.Atoi(s.Get(key))
	n++
	id := strconv.Itoa(n)
	s.Set(key, id)
	return id
}

func sortedIntList(ids []string) []string {
	sort.Slice(ids, func(i, j int) bool {
		ni, _ := strconv.Atoi(ids[i])
		nj, _ := strconv.Atoi(ids[j])
		return ni < nj
	})
	return ids
}

// --- Initialization ---

var initialFiles []string
var initialized bool

func (a *acmeApp) viewInit(s view.State) {
	if initialized {
		return
	}
	initialized = true

	s.Set("next-col", "1")
	s.Set("cols/1/name", "")

	if len(initialFiles) > 0 {
		winID := 0
		for _, f := range initialFiles {
			winID++
			wid := strconv.Itoa(winID)
			s.Set("cols/1/wins/"+wid+"/name", f)
			s.Set("cols/1/wins/"+wid+"/dirty", "0")
			data, err := os.ReadFile(f)
			if err == nil {
				bodyID := "wb-1-" + wid
				s.Set("_body/"+bodyID, string(data))
			}
		}
		s.Set("next-win", strconv.Itoa(winID))
	} else {
		s.Set("next-win", "1")
		s.Set("cols/1/wins/1/name", "scratch")
		s.Set("cols/1/wins/1/dirty", "0")
	}
}

// --- Main ---

func main() {
	if len(os.Args) > 1 {
		initialFiles = os.Args[1:]
	}
	if err := ui.Run("Acme", &acmeApp{}); err != nil {
		log.Fatal(err)
	}
}
