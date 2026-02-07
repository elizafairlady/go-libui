// Acme is a text editor modelled on Plan 9's acme(1), built entirely
// with the ui framework. It demonstrates that the framework can express
// Acme's core interaction model: editable tags, multi-line bodies,
// columns with resizable windows, B2 execute, B3 look, file I/O.
//
// This is not a complete Acme — it is a proof that the declarative ui
// framework can host Acme-style interaction. Shell execution, plumbing,
// address parsing, undo, and snarf are left for later.
//
// Usage: acme [file ...]
//
// B1: select text
// B2: execute command (tag words: New, Del, Get, Put, Newcol, Delcol, Exit)
// B3: look — open file or search for text
package main

import (
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

// --- View ---

func (a *acmeApp) View(s view.State) *view.Node {
	// Initialize on first call
	a.viewInit(s)

	// Gather columns
	colIDs := sortedIntList(s.List("cols"))

	var colNodes []*view.Node
	for _, cid := range colIDs {
		colNodes = append(colNodes, a.viewColumn(s, cid))
	}

	// If no columns, show empty state
	if len(colNodes) == 0 {
		colNodes = append(colNodes,
			view.VBox("empty",
				view.TextNode("empty-text", "B2 on Newcol to create a column").
					Prop("fg", "acmedim").PropInt("pad", 20),
			).Prop("flex", "1"),
		)
	}

	return view.VBox("root",
		// Row tag
		view.Tag("row-tag", "Newcol Kill Putall Dump Load Exit").PropInt("pad", 2),
		view.Rect("row-sep").Prop("bg", "acmeborder").PropInt("minh", 1).PropInt("maxh", 1),

		// Columns in horizontal splitbox
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

	// Column tag + separator + windows
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
		// Empty column body
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

func (a *acmeApp) Handle(s view.State, act *proto.Action) {
	switch act.Kind {
	case "execute":
		a.handleExecute(s, act)
	case "look":
		a.handleLook(s, act)
	}
}

func (a *acmeApp) handleExecute(s view.State, act *proto.Action) {
	text := act.KVs["text"]
	id := act.KVs["id"]

	switch text {
	// Row-level commands
	case "Newcol":
		a.newColumn(s)
	case "Exit":
		s.Set("_quit", "1")
	case "Putall":
		a.putAll(s)

	// Column-level commands
	case "New":
		colID := parseColID(id)
		if colID != "" {
			a.newWindow(s, colID, "")
		}
	case "Delcol":
		colID := parseColID(id)
		if colID != "" {
			a.delColumn(s, colID)
		}

	// Window-level commands
	case "Del":
		colID, winID := parseWinID(id)
		if colID != "" && winID != "" {
			a.delWindow(s, colID, winID)
		}
	case "Get":
		colID, winID := parseWinID(id)
		if colID != "" && winID != "" {
			a.getFile(s, colID, winID)
		}
	case "Put":
		colID, winID := parseWinID(id)
		if colID != "" && winID != "" {
			a.putFile(s, colID, winID)
		}

	default:
		// Try to execute as a shell command or find in a body
		// For now, unknown commands are ignored
	}
}

func (a *acmeApp) handleLook(s view.State, act *proto.Action) {
	text := act.KVs["text"]
	if text == "" {
		return
	}

	// Try to open as a file
	info, err := os.Stat(text)
	if err == nil && !info.IsDir() {
		// Open the file in a new window in the first column
		colIDs := sortedIntList(s.List("cols"))
		if len(colIDs) == 0 {
			// Create a column first
			a.newColumn(s)
			colIDs = sortedIntList(s.List("cols"))
		}
		if len(colIDs) > 0 {
			a.newWindow(s, colIDs[0], text)
		}
	}
}

// --- Structural operations ---

func (a *acmeApp) newColumn(s view.State) {
	next := nextID(s, "next-col")
	s.Set("cols/"+next+"/name", "")
}

func (a *acmeApp) delColumn(s view.State, colID string) {
	// Delete all windows in the column
	for _, wid := range s.List("cols/" + colID + "/wins") {
		s.Del("cols/" + colID + "/wins/" + wid + "/name")
		s.Del("cols/" + colID + "/wins/" + wid + "/dirty")
	}
	s.Del("cols/" + colID + "/name")
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

	// If a filename was given, load the file into the body
	if filename != "" {
		data, err := os.ReadFile(filename)
		if err == nil {
			bodyID := "wb-" + colID + "-" + next
			s.Set("_body/"+bodyID, string(data))
			s.Set("_body/"+bodyID+"/clean", "1")
		}
	}
}

func (a *acmeApp) delWindow(s view.State, colID, winID string) {
	s.Del("cols/" + colID + "/wins/" + winID + "/name")
	s.Del("cols/" + colID + "/wins/" + winID + "/dirty")
}

// --- File I/O ---

func (a *acmeApp) getFile(s view.State, colID, winID string) {
	name := s.Get("cols/" + colID + "/wins/" + winID + "/name")
	if name == "" || name == "scratch" {
		return
	}
	data, err := os.ReadFile(name)
	if err != nil {
		return // TODO: show error in +Errors
	}
	bodyID := "wb-" + colID + "-" + winID
	s.Set("_body/"+bodyID, string(data))
	s.Set("_body/"+bodyID+"/clean", "1")
	s.Set("cols/"+colID+"/wins/"+winID+"/dirty", "0")
}

func (a *acmeApp) putFile(s view.State, colID, winID string) {
	name := s.Get("cols/" + colID + "/wins/" + winID + "/name")
	if name == "" || name == "scratch" {
		return
	}
	bodyID := "wb-" + colID + "-" + winID
	text := s.Get("_body/" + bodyID)
	err := os.WriteFile(name, []byte(text), 0644)
	if err != nil {
		return // TODO: show error
	}
	s.Set("_body/"+bodyID+"/clean", "1")
	s.Set("cols/"+colID+"/wins/"+winID+"/dirty", "0")
}

func (a *acmeApp) putAll(s view.State) {
	for _, cid := range s.List("cols") {
		for _, wid := range s.List("cols/" + cid + "/wins") {
			name := s.Get("cols/" + cid + "/wins/" + wid + "/name")
			if name != "" && name != "scratch" {
				bodyID := "wb-" + cid + "-" + wid
				dirty := s.Get("_body/" + bodyID + "/dirty")
				if dirty == "1" {
					a.putFile(s, cid, wid)
				}
			}
		}
	}
}

// --- ID parsing ---

// parseColID extracts column ID from a tag or body node ID.
// Tag IDs: "ct-1", "row-tag"
// Window tag IDs: "wt-1-2"
// Body IDs: "wb-1-2"
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
	// If the execute came from the row tag, pick the first column
	// (New from row tag creates in first column)
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

// --- Main ---

func main() {
	app := &acmeApp{}

	// If files are given on command line, set up initial state to open them
	// We need to set initial state before Run, but Run creates the UIFS.
	// Solution: use an init hook via the View function.
	// On first View call, if no columns exist and args were given, create them.
	if len(os.Args) > 1 {
		initialFiles = os.Args[1:]
	}

	if err := ui.Run("Acme", app); err != nil {
		log.Fatal(err)
	}
}

var initialFiles []string
var initialized bool

// viewInit sets up initial columns, windows, and file contents
// on the first View call. The _body proxy is wired by this point
// (run.go wires callbacks before the first Tree()/View() call).
func (a *acmeApp) viewInit(s view.State) {
	if initialized {
		return
	}
	initialized = true

	// Create initial column
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
		// Create a scratch window
		s.Set("next-win", "1")
		s.Set("cols/1/wins/1/name", "scratch")
		s.Set("cols/1/wins/1/dirty", "0")
	}
}
