package main

import (
	"log"

	ui "go-libui/libui"
)

// drawCtx is stored for hit testing in translateEvent
var drawCtx *ui.DrawContext

func main() {
	app := ui.App{
		Model:  Model{NextID: 1},
		Reduce: translateAndReduce,
		Draw:   drawWithCtx,
	}

	if err := ui.Run(app); err != nil {
		log.Fatal(err)
	}
}

// drawWithCtx wraps Draw to capture the context for hit testing
func drawWithCtx(model any, ctx *ui.DrawContext) {
	drawCtx = ctx
	Draw(model, ctx)
}

// translateAndReduce translates raw ui.Event into semantic events,
// then calls the reducer. This glue lives outside libui.
func translateAndReduce(model any, ev ui.Event) any {
	m := model.(Model)

	switch ev.Kind {
	case "key":
		k := ev.Data.(ui.Key)
		r := k.Rune

		switch r {
		case '\n', '\r': // Enter
			return Reduce(m, ui.Event{Kind: "app", Data: AddTodo{}})

		case '\b', 0x7f: // Backspace or DEL
			return Reduce(m, ui.Event{Kind: "app", Data: Backspace{}})

		case 0x03: // Ctrl+C - exit
			// Could handle exit here if needed
			return m

		default:
			// Printable character
			if r >= 32 && r < 127 {
				return Reduce(m, ui.Event{Kind: "app", Data: EditInput{Rune: r}})
			}
		}

	case "mouse":
		mouse := ev.Data.(ui.Mouse)
		// Check for button 1 click (left button)
		if mouse.Buttons&1 != 0 && drawCtx != nil {
			if id, ok := HitTodo(drawCtx, m.Todos, mouse.X, mouse.Y); ok {
				return Reduce(m, ui.Event{Kind: "app", Data: ToggleTodo{ID: id}})
			}
		}

	case "resize":
		r := ev.Data.(ui.Resize)
		return Reduce(m, ui.Event{Kind: "app", Data: ResizeCols{Cols: r.Width}})
	}

	return m
}
