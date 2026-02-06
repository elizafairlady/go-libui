package main

import (
	ui "go-libui/libui"
)

// Reduce handles all state transitions.
// This is a pure function - no drawing, no hit-testing, no scroll logic.
func Reduce(model any, ev ui.Event) any {
	m := model.(Model)

	// Handle semantic events from Data field
	switch e := ev.Data.(type) {
	case AddTodo:
		if m.Input != "" {
			m.Todos = append(m.Todos, Todo{
				ID:   m.NextID,
				Text: m.Input,
				Done: false,
			})
			m.NextID++
			m.Input = ""
		}

	case ToggleTodo:
		for i := range m.Todos {
			if m.Todos[i].ID == e.ID {
				m.Todos[i].Done = !m.Todos[i].Done
				break
			}
		}

	case EditInput:
		m.Input = m.Input + string(e.Rune)

	case Backspace:
		if len(m.Input) > 0 {
			// Remove last rune
			runes := []rune(m.Input)
			m.Input = string(runes[:len(runes)-1])
		}

	case ResizeCols:
		m.Cols = e.Cols
	}

	return m
}
