// Package uifs implements the UI filesystem server core.
//
// It hosts an App, manages state, generates tree snapshots on demand,
// processes actions, and resolves data bindings. The UIFS can be
// used directly in-process or exported as a 9P server.
//
// The UIFS is the authoritative model/controller boundary:
//   - State is stored in hierarchical path-keyed entries
//   - The tree is computed from state via App.View()
//   - Actions are processed via App.Handle()
//   - Bindings are resolved by matching node props to state paths
package uifs

import (
	"sync"

	"github.com/elizafairlady/go-libui/ui/proto"
	"github.com/elizafairlady/go-libui/ui/view"
)

// UIFS is the core UI filesystem server.
type UIFS struct {
	mu   sync.Mutex
	app  view.App
	st   *view.MemState
	rev  uint64
	tree *proto.Tree // cached tree snapshot

	// Focus is stored here for transparency
	Focus string

	// Notify is called when the tree has been invalidated.
	// The renderer should repaint.
	Notify func()

	// ActionLog records processed actions (for debugging).
	// Set to non-nil to enable logging.
	ActionLog []string
}

// New creates a new UIFS with the given app and initial state.
func New(app view.App) *UIFS {
	return &UIFS{
		app: app,
		st:  view.NewMemState(),
	}
}

// State returns the state store.
func (u *UIFS) State() *view.MemState {
	return u.st
}

// Rev returns the current revision number.
func (u *UIFS) Rev() uint64 {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.rev
}

// Tree returns the current tree snapshot, recomputing if necessary.
func (u *UIFS) Tree() *proto.Tree {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.tree == nil {
		u.recompute()
	}
	return u.tree
}

// TreeText returns the serialized tree text (for cat /mnt/ui/app/tree).
func (u *UIFS) TreeText() string {
	t := u.Tree()
	if t == nil {
		return "rev 0\nroot \n"
	}
	return proto.SerializeTree(t)
}

// Invalidate marks the tree as needing recomputation.
func (u *UIFS) Invalidate() {
	u.mu.Lock()
	u.tree = nil
	u.mu.Unlock()
	if u.Notify != nil {
		u.Notify()
	}
}

// ProcessAction parses and processes an action line
// (as would be written to /mnt/ui/app/actions).
func (u *UIFS) ProcessAction(line string) error {
	a, err := proto.ParseAction(line)
	if err != nil {
		return err
	}
	u.HandleAction(a)
	return nil
}

// HandleAction processes a semantic action.
func (u *UIFS) HandleAction(a *proto.Action) {
	u.mu.Lock()
	defer u.mu.Unlock()

	// Log if enabled
	if u.ActionLog != nil {
		u.ActionLog = append(u.ActionLog, proto.SerializeAction(a))
	}

	// Handle focus changes
	if a.Kind == "focus" {
		u.Focus = a.KVs["id"]
	}

	// Resolve bindings before passing to app
	u.resolveBindings(a)

	// Pass to app handler
	u.app.Handle(u.st, a)

	// Invalidate tree
	u.tree = nil
	u.rev++

	// Notify outside lock
	notify := u.Notify
	u.mu.Unlock()
	if notify != nil {
		notify()
	}
	u.mu.Lock()
}

// resolveBindings resolves data bindings based on the current tree
// and the action. For example, an "input" action on a textbox with
// bind=state/query will update state/query with the new text.
func (u *UIFS) resolveBindings(a *proto.Action) {
	if u.tree == nil {
		u.recompute()
	}
	id := a.KVs["id"]
	if id == "" || u.tree == nil {
		return
	}
	node := u.tree.Nodes[id]
	if node == nil {
		return
	}

	switch a.Kind {
	case "input":
		// Resolve textbox bind
		if bindPath := node.Props["bind"]; bindPath != "" {
			if text, ok := a.KVs["text"]; ok {
				u.st.Set(bindPath, text)
			}
		}
	case "toggle":
		// Resolve checkbox binding
		bindPath := node.Props["bindchecked"]
		if bindPath == "" {
			bindPath = node.Props["bind"]
		}
		if bindPath != "" {
			if val, ok := a.KVs["value"]; ok {
				u.st.Set(bindPath, val)
			}
		}
	}
}

// recompute generates a new tree from the app. Must be called with mu held.
func (u *UIFS) recompute() {
	root := u.app.View(u.st)
	if root == nil {
		u.tree = nil
		return
	}
	u.rev++
	u.tree = view.Serialize(root, u.rev)
	u.populateBindings()
}

// populateBindings walks the tree and fills in bound values from state.
// For textbox nodes with bind=X, sets text=state.Get(X).
// For checkbox nodes with bind=X or bindchecked=X, sets checked from state.
func (u *UIFS) populateBindings() {
	if u.tree == nil {
		return
	}
	for _, node := range u.tree.Nodes {
		bindPath := node.Props["bind"]
		if bindPath == "" {
			continue
		}
		switch node.Type {
		case "textbox":
			// Auto-populate text from bound state
			if _, explicit := node.Props["text"]; !explicit || node.Props["text"] == "" {
				node.Props["text"] = u.st.Get(bindPath)
			}
		case "checkbox":
			checkPath := node.Props["bindchecked"]
			if checkPath == "" {
				checkPath = bindPath
			}
			if _, explicit := node.Props["checked"]; !explicit {
				node.Props["checked"] = u.st.Get(checkPath)
			}
		}
	}
}

// SetState sets a state value and invalidates the tree.
func (u *UIFS) SetState(path, value string) {
	u.st.Set(path, value)
	u.Invalidate()
}

// GetState gets a state value.
func (u *UIFS) GetState(path string) string {
	return u.st.Get(path)
}

// SetFocus sets the focus and invalidates.
func (u *UIFS) SetFocus(id string) {
	u.mu.Lock()
	u.Focus = id
	u.tree = nil
	u.mu.Unlock()
	if u.Notify != nil {
		u.Notify()
	}
}
