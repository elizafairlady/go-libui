package main

import (
	ui "go-libui/libui"
)

const (
	hitPadding    = 10
	hitLineHeight = 20 // approximate, should match draw
)

// HitTodo determines if a mouse click hit a todo checkbox.
// Returns the todo index and whether a hit occurred.
// Manual, explicit hit-testing - no widget types, no abstraction.
func HitTodo(ctx *ui.DrawContext, todos []Todo, mouseX, mouseY int) (id int, ok bool) {
	lineHeight := ctx.FontHeight() + 4
	y := hitPadding

	// Skip input line
	y += lineHeight + hitPadding

	// Skip separator
	y += lineHeight

	// Check each todo
	for _, todo := range todos {
		// Check if click is in checkbox area
		if mouseX >= hitPadding && mouseX <= hitPadding+checkboxW {
			if mouseY >= y && mouseY < y+lineHeight {
				return todo.ID, true
			}
		}
		y += lineHeight
	}

	return 0, false
}
