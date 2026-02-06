package main

import (
	ui "go-libui/libui"
)

const (
	padding   = 10
	checkboxW = 20
)

// Draw renders the model to the screen.
// This is a pure function - never mutates model.
func Draw(model any, ctx *ui.DrawContext) {
	m := model.(Model)
	lineHeight := ctx.FontHeight() + 4
	y := padding

	// Draw input line at top
	inputText := "> " + m.Input + "_"
	ctx.Text(padding, y, inputText)
	y += lineHeight + padding

	// Draw separator
	ctx.Text(padding, y, "---")
	y += lineHeight

	// Draw todo list
	for _, todo := range m.Todos {
		// Draw checkbox
		checkbox := "[ ]"
		if todo.Done {
			checkbox = "[x]"
		}
		ctx.Text(padding, y, checkbox)

		// Draw todo text
		ctx.Text(padding+checkboxW+padding, y, todo.Text)

		y += lineHeight
	}

	// Draw empty state hint
	if len(m.Todos) == 0 {
		ctx.Text(padding, y, "(type and press Enter to add a todo)")
	}
}
