package ui

// ViewState holds view-local state that is NOT part of the model.
// This is owned by Run, not by the application.
type ViewState struct {
	ScrollY int
	Width   int
	Height  int
}
