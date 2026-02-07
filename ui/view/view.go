// Package view provides the Go API for building declarative UI
// view trees and the App/State interfaces for the UI framework.
//
// An application implements the App interface, providing View to
// build a node tree from state, and Handle to process actions.
// The framework calls these as needed.
package view

import (
	"strconv"
	"sync"

	"github.com/elizafairlady/go-libui/ui/proto"
	"github.com/elizafairlady/go-libui/ui/window"
)

// Node is a UI view tree node with an ID, type, props, and children.
type Node struct {
	ID       string
	Type     string
	Props    map[string]string
	Children []*Node
}

// State is a hierarchical key-value store for application state.
type State interface {
	Get(path string) string
	Set(path, value string)
	Del(path string)
	List(dir string) []string
}

// Action is a semantic UI action sent from the renderer.
type Action = proto.Action

// App is the application interface. The framework calls View to
// get the current UI tree, and Handle to process user actions.
type App interface {
	View(s State) *Node
	Handle(s State, a *Action)
}

// ExecContext provides the context for executing a command via B2.
// This is passed to builtins and made available to external commands
// via environment variables.
type ExecContext struct {
	// ID is the tag/body node ID where the command was invoked.
	ID string
	// Cmd is the command word that was B2-clicked.
	Cmd string
	// Selection is the current text selection in the focused body, if any.
	Selection string
	// State gives access to the UI state (including _body/ and _tag/ proxies).
	State State
}

// Builtin is a function that implements a built-in command.
// It receives the execution context and returns an optional error message.
type Builtin func(ctx *ExecContext) error

// Executor is an optional interface that apps can implement to
// provide built-in commands and control external command execution.
// If an app implements Executor, the framework uses it for B2.
// If not, the framework falls back to the generic "execute" action.
type Executor interface {
	// Builtins returns the app's registered built-in commands.
	// The framework checks these before trying external commands.
	Builtins() map[string]Builtin

	// BinDirs returns additional directories to search for external
	// commands (like Plan 9's $home/bin). Can return nil.
	BinDirs() []string
}

// RowProvider is an optional interface that apps can implement to
// provide a window.Row. When present, body nodes with a "winid"
// prop get their text buffer from the Row's Window, making the body
// a proper file-backed view rather than renderer-owned state.
type RowProvider interface {
	WindowRow() *window.Row
}

// --- Node builder helpers ---

// N creates a new node with the given id and type.
func N(id, typ string) *Node {
	return &Node{
		ID:    id,
		Type:  typ,
		Props: make(map[string]string),
	}
}

// Prop sets a property on the node and returns it for chaining.
func (n *Node) Prop(k, v string) *Node {
	n.Props[k] = v
	return n
}

// PropInt sets an integer property.
func (n *Node) PropInt(k string, v int) *Node {
	n.Props[k] = strconv.Itoa(v)
	return n
}

// Text sets the "text" property (convenience for text/button/etc).
func (n *Node) Text(s string) *Node {
	return n.Prop("text", s)
}

// Child appends child nodes and returns the parent for chaining.
func (n *Node) Child(children ...*Node) *Node {
	n.Children = append(n.Children, children...)
	return n
}

// --- Node types (convenience constructors) ---

// VBox creates a vertical box layout node.
func VBox(id string, children ...*Node) *Node {
	return N(id, "vbox").Child(children...)
}

// HBox creates a horizontal box layout node.
func HBox(id string, children ...*Node) *Node {
	return N(id, "hbox").Child(children...)
}

// Stack creates a stack layout node (children overlap).
func Stack(id string, children ...*Node) *Node {
	return N(id, "stack").Child(children...)
}

// Spacer creates a flexible spacer.
func Spacer(id string) *Node {
	return N(id, "spacer").Prop("flex", "1")
}

// Scroll creates a scrollable container.
func Scroll(id string, children ...*Node) *Node {
	return N(id, "scroll").Child(children...)
}

// TextNode creates a text display node.
func TextNode(id, text string) *Node {
	return N(id, "text").Text(text)
}

// Button creates a button node.
func Button(id, text string) *Node {
	return N(id, "button").Text(text).Prop("focusable", "1")
}

// Checkbox creates a checkbox node.
func Checkbox(id, text string, checked bool) *Node {
	v := "0"
	if checked {
		v = "1"
	}
	return N(id, "checkbox").Text(text).Prop("checked", v).Prop("focusable", "1")
}

// TextBox creates a text input node.
func TextBox(id string) *Node {
	return N(id, "textbox").Prop("focusable", "1")
}

// Rect creates a colored rectangle node.
func Rect(id string) *Node {
	return N(id, "rect")
}

// Row creates a semantic row container (for lists).
func Row(id string, children ...*Node) *Node {
	return N(id, "row").Child(children...)
}

// Tag creates an Acme-style editable tag bar.
// The text contains command words that can be B2-executed.
func Tag(id, text string) *Node {
	return N(id, "tag").Text(text).Prop("focusable", "1")
}

// Body creates a multi-line editable text area backed by frame.Frame.
// Like tag, the renderer owns the text buffer. The view tree provides
// initial text; after first init the frame state persists across rebuilds.
// Props: text (initial), bg, fg, scrollbar (0|1).
func Body(id string) *Node {
	return N(id, "body").Prop("focusable", "1")
}

// SplitBox creates a container with draggable resize handles between children.
// Props: direction (vertical|horizontal), weights (comma-separated ints).
// Children get space proportional to their weight. Drag handles between
// children allow the user to redistribute space.
func SplitBox(id string, children ...*Node) *Node {
	return N(id, "splitbox").Child(children...)
}

// --- Serialization ---

// Serialize converts the node tree to a proto.Tree for the protocol.
func Serialize(root *Node, rev uint64) *proto.Tree {
	t := &proto.Tree{
		Rev:   rev,
		Root:  root.ID,
		Nodes: make(map[string]*proto.Node),
	}
	var walk func(n *Node)
	walk = func(n *Node) {
		pn := &proto.Node{
			ID:    n.ID,
			Type:  n.Type,
			Props: make(map[string]string),
		}
		for k, v := range n.Props {
			pn.Props[k] = v
		}
		for _, child := range n.Children {
			pn.Children = append(pn.Children, child.ID)
		}
		t.Nodes[n.ID] = pn
		t.Order = append(t.Order, n.ID)
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(root)
	return t
}

// --- In-memory State implementation ---

// MemState is a simple in-memory hierarchical state store.
type MemState struct {
	mu   sync.RWMutex
	data map[string]string
}

// NewMemState creates a new empty in-memory state.
func NewMemState() *MemState {
	return &MemState{data: make(map[string]string)}
}

// Get returns the value at path, or "" if not set.
func (s *MemState) Get(path string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[path]
}

// Set sets the value at path.
func (s *MemState) Set(path, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[path] = value
}

// Del deletes the value at path.
func (s *MemState) Del(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, path)
}

// List returns the direct children under dir.
// Keys are stored as "dir/child"; this returns the "child" parts.
func (s *MemState) List(dir string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	prefix := dir + "/"
	seen := make(map[string]bool)
	var result []string
	for k := range s.data {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			rest := k[len(prefix):]
			// Take only the first segment
			slash := -1
			for i := 0; i < len(rest); i++ {
				if rest[i] == '/' {
					slash = i
					break
				}
			}
			name := rest
			if slash >= 0 {
				name = rest[:slash]
			}
			if !seen[name] {
				seen[name] = true
				result = append(result, name)
			}
		}
	}
	return result
}

// GetInt returns the integer value at path, or def if not set or invalid.
func (s *MemState) GetInt(path string, def int) int {
	v := s.Get(path)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// GetBool returns the boolean value at path (truthy: "1", "true").
func (s *MemState) GetBool(path string) bool {
	v := s.Get(path)
	return v == "1" || v == "true"
}
