// Todo is THE example app for the ui framework.
//
// It demonstrates dynamic lists, hierarchical state, checkbox
// bindings, text input, and Acme-style tag commands — everything
// a real application needs.
//
// Usage: todo
//
// B2 on "Add" to add the typed item. B2 on "Clear" to remove
// completed items. B2 on "Del" to quit. Enter in the input box
// also adds an item.
package main

import (
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/elizafairlady/go-libui/ui"
	"github.com/elizafairlady/go-libui/ui/proto"
	"github.com/elizafairlady/go-libui/ui/view"
)

type todoApp struct{}

func (a *todoApp) View(s view.State) *view.Node {
	// Gather and sort item IDs numerically
	ids := s.List("items")
	sort.Slice(ids, func(i, j int) bool {
		ni, _ := strconv.Atoi(ids[i])
		nj, _ := strconv.Atoi(ids[j])
		return ni < nj
	})

	// Count stats
	total := len(ids)
	done := 0
	for _, id := range ids {
		if s.Get("items/"+id+"/done") == "1" {
			done++
		}
	}

	// Build item rows
	var items []*view.Node
	for _, id := range ids {
		text := s.Get("items/" + id + "/text")
		checked := s.Get("items/"+id+"/done") == "1"
		items = append(items,
			view.Checkbox("item-"+id, text, checked).
				Prop("bind", "items/"+id+"/done"),
		)
	}

	// Empty state message
	if len(items) == 0 {
		items = append(items,
			view.TextNode("empty", "No items yet. Type above and B2 Add.").
				Prop("fg", "acmedim").PropInt("pad", 4),
		)
	}

	// Status line
	status := strconv.Itoa(done) + "/" + strconv.Itoa(total) + " done"
	if total == 0 {
		status = "no items"
	}

	// Assemble body children
	var body []*view.Node

	// Text input for new items
	body = append(body,
		view.HBox("input-row",
			view.TextBox("input").
				Prop("bind", "input").
				Prop("placeholder", "new todo...").
				Prop("flex", "1"),
		).PropInt("pad", 2),
	)

	// Thin separator between input and list
	body = append(body,
		view.Rect("sep2").Prop("bg", "acmeborder").
			PropInt("minh", 1).PropInt("maxh", 1),
	)

	// Todo items
	body = append(body, items...)

	// Flexible space pushes footer down
	body = append(body, view.Spacer("sp"))

	// Status footer
	body = append(body,
		view.TextNode("status", status).
			Prop("fg", "acmedim").PropInt("pad", 4),
	)

	// Help hint
	body = append(body,
		view.TextNode("help", "B1 select · B2 execute · B3 look · Tab ↹ navigate").
			Prop("fg", "acmedim").PropInt("pad", 4),
	)

	return view.VBox("root",
		// Acme-style tag bar
		view.Tag("tag", "Todo Add Clear Del").PropInt("pad", 2),

		// Thin separator
		view.Rect("sep1").Prop("bg", "acmeborder").
			PropInt("minh", 1).PropInt("maxh", 1),

		// Body
		view.VBox("body", body...).
			Prop("flex", "1").PropInt("pad", 6).PropInt("gap", 4),
	).PropInt("pad", 0).PropInt("gap", 0)
}

func (a *todoApp) Handle(s view.State, act *proto.Action) {
	switch act.Kind {
	case "execute":
		// B2 on tag word
		switch act.KVs["text"] {
		case "Add":
			a.addItem(s)
		case "Clear":
			a.clearDone(s)
		case "Del":
			s.Set("_quit", "1")
		}
	case "key":
		// Enter in the input textbox also adds an item
		if act.KVs["key"] == "Enter" && act.KVs["id"] == "input" {
			a.addItem(s)
		}
	}
}

// addItem takes the current input text, creates a new todo item,
// and clears the input.
func (a *todoApp) addItem(s view.State) {
	text := strings.TrimSpace(s.Get("input"))
	if text == "" {
		return
	}

	// Auto-increment ID
	next, _ := strconv.Atoi(s.Get("next"))
	next++
	id := strconv.Itoa(next)

	s.Set("items/"+id+"/text", text)
	s.Set("items/"+id+"/done", "0")
	s.Set("next", id)
	s.Set("input", "")
}

// clearDone removes all completed items from state.
func (a *todoApp) clearDone(s view.State) {
	for _, id := range s.List("items") {
		if s.Get("items/"+id+"/done") == "1" {
			s.Del("items/" + id + "/text")
			s.Del("items/" + id + "/done")
		}
	}
}

func main() {
	if err := ui.Run("Todo", &todoApp{}); err != nil {
		log.Fatal(err)
	}
}
