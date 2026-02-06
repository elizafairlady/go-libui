// Package ui provides a minimal Go library for Plan 9 / 9front
// with a disciplined UI control loop, reducer-based state updates,
// single redraw path, and view-local state.
package ui

// Event represents a raw input event from the system.
type Event struct {
	Kind string // "mouse", "key", "resize"
	Data any
}

// Reducer processes an event and returns a new model.
// Must be a pure function - never mutate view state.
type Reducer func(model any, ev Event) any

// Drawer renders the model to the screen.
// Must be a pure function - never mutate model.
type Drawer func(model any, ctx *DrawContext)

// App defines the application structure.
type App struct {
	Model  any
	Reduce Reducer
	Draw   Drawer
}
