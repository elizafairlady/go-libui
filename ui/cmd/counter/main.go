// Counter is a minimal example app using the ui framework.
//
// It displays a counter with increment/decrement buttons,
// a text input, and a checkbox — demonstrating the full
// framework with an Acme-inspired visual style.
//
// Usage: counter
// Quit with DEL key.
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
		// Tag bar — Acme-style pale cyan header with controls
		view.HBox("tag",
			view.TextNode("tag-title", "Counter").PropInt("pad", 6),
			view.Spacer("tag-sp"),
			view.Button("dec", " − ").Prop("on", "dec"),
			view.TextNode("count-display", " "+strconv.Itoa(count)+" ").
				PropInt("pad", 6).PropInt("minw", 40),
			view.Button("inc", " + ").Prop("on", "inc"),
		).Prop("bg", "acmetag").PropInt("pad", 2).PropInt("gap", 2),

		// Thin separator
		view.Rect("sep1").Prop("bg", "acmeborder").PropInt("minh", 1).PropInt("maxh", 1),

		// Body content
		view.VBox("body",
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
			view.TextNode("help", "Tab ↹ navigate · Enter ↵ click · DEL quit").
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
	case "click":
		switch act.KVs["action"] {
		case "inc":
			n, _ := strconv.Atoi(s.Get("count"))
			s.Set("count", strconv.Itoa(n+1))
		case "dec":
			n, _ := strconv.Atoi(s.Get("count"))
			s.Set("count", strconv.Itoa(n-1))
		}
	}
}

func main() {
	if err := ui.Run("Counter", &counterApp{}); err != nil {
		log.Fatal(err)
	}
}
