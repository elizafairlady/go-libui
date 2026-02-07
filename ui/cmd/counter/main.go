// Counter is a minimal example app using the ui framework.
//
// It displays a counter with increment/decrement buttons,
// a text input, and a checkbox — demonstrating the full
// framework: view trees, state, actions, bindings.
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

	return view.VBox("root",
		view.TextNode("title", "Counter Demo").Prop("bg", "paleblue").PropInt("pad", 8),
		view.HBox("counter",
			view.Button("dec", "-").Prop("on", "dec").PropInt("minw", 40),
			view.TextNode("count", strconv.Itoa(count)).PropInt("pad", 8).PropInt("minw", 60),
			view.Button("inc", "+").Prop("on", "inc").PropInt("minw", 40),
		).PropInt("gap", 4).PropInt("pad", 4),
		view.HBox("input-row",
			view.TextNode("label", "Name:").PropInt("pad", 4),
			view.TextBox("name").Prop("bind", "name").Prop("placeholder", "type here...").Prop("flex", "1"),
		).PropInt("gap", 4).PropInt("pad", 4),
		view.TextNode("greeting", greeting(s.Get("name"))).PropInt("pad", 4),
		view.Checkbox("agree", "I agree", s.Get("agree") == "1").Prop("bind", "agree"),
		view.Spacer("sp"),
		view.TextNode("help", "Tab to navigate, Enter to click, DEL to quit").
			Prop("fg", "greyblue").PropInt("pad", 4),
	).PropInt("pad", 4).PropInt("gap", 2)
}

func greeting(name string) string {
	if name == "" {
		return ""
	}
	return "Hello, " + name + "!"
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
