package main

// Todo represents a single todo item.
type Todo struct {
	ID   int
	Text string
	Done bool
}

// Model represents the application state.
type Model struct {
	Todos  []Todo
	Input  string
	NextID int
	Cols   int // derived from resize
}
