// Counter is a minimal example app using the ui framework.
//
// It demonstrates Acme-style interaction: the tag bar at the top
// is a real editable text frame. Middle-click (B2) on a command
// word to execute it (Inc, Dec, Reset, Del).
//
// Usage: counter
// B2 on "Del" in the tag to quit.
package main

import (
	"log"
	"strconv"

	"github.com/elizafairlady/go-libui/ui"
	"github.com/elizafairlady/go-libui/ui/proto"
	"github.com/elizafairlady/go-libui/ui/view"
)

type counterApp struct{}

func (a *counterApp) View(s view.State) *view.Node {
	count, _ := strconv.Atoi(s.Get("count"))
	name := s.Get("name")

	return view.VBox("root",
		// Acme-style tag bar — editable, B2 on words executes them
		view.Tag("tag", "Counter Inc Dec Reset Del").PropInt("pad", 2),

		// Thin separator
		view.Rect("sep1").Prop("bg", "acmeborder").PropInt("minh", 1).PropInt("maxh", 1),

		// Body content
		view.VBox("body",
			// Count display
			view.TextNode("count-display", "Count: "+strconv.Itoa(count)).
				PropInt("pad", 6),

			// Name input row
			view.HBox("input-row",
				view.TextNode("label", "Name").PropInt("pad", 6),
				view.TextBox("name").Prop("bind", "name").
					Prop("placeholder", "type a name...").
					Prop("flex", "1"),
			).PropInt("gap", 6).PropInt("pad", 2),

			// Greeting (only shown when name is set)
			greetingNode(name),

			// Checkbox
			view.Checkbox("agree", "I agree to the terms", s.Get("agree") == "1").
				Prop("bind", "agree"),

			// Flexible space pushes footer down
			view.Spacer("body-sp"),

			// Footer hint
			view.TextNode("help", "B1 select · B2 execute · B3 look · Tab ↹ navigate").
				Prop("fg", "acmedim").PropInt("pad", 4),
		).Prop("flex", "1").PropInt("pad", 6).PropInt("gap", 6),
	).PropInt("pad", 0).PropInt("gap", 0)
}

func greetingNode(name string) *view.Node {
	if name == "" {
		return view.TextNode("greeting", "").PropInt("minh", 0).PropInt("maxh", 0)
	}
	return view.TextNode("greeting", "Hello, "+name+"!").
		Prop("bg", "acmehigh").PropInt("pad", 6)
}

func (a *counterApp) Handle(s view.State, act *proto.Action) {
	switch act.Kind {
	case "execute":
		// B2 on tag word
		switch act.KVs["text"] {
		case "Inc":
			n, _ := strconv.Atoi(s.Get("count"))
			s.Set("count", strconv.Itoa(n+1))
		case "Dec":
			n, _ := strconv.Atoi(s.Get("count"))
			s.Set("count", strconv.Itoa(n-1))
		case "Reset":
			s.Set("count", "0")
			s.Set("name", "")
			s.Set("agree", "0")
		case "Del":
			// Signal quit — for now just set a state flag
			// (In real usage, the framework would handle "Del" specially)
			s.Set("_quit", "1")
		}
	}
}

func main() {
	if err := ui.Run("Counter", &counterApp{}); err != nil {
		log.Fatal(err)
	}
}
