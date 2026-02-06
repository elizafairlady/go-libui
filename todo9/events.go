package main

// AddTodo signals that the current input should be added as a todo.
type AddTodo struct{}

// ToggleTodo signals that a todo should be toggled.
type ToggleTodo struct {
	ID int
}

// EditInput signals that a character was typed.
type EditInput struct {
	Rune rune
}

// Backspace signals that a character should be deleted.
type Backspace struct{}

// ResizeCols signals that the window width changed.
type ResizeCols struct {
	Cols int
}
