package uifs

import (
	"strings"
	"testing"

	"github.com/elizafairlady/go-libui/ui/proto"
	"github.com/elizafairlady/go-libui/ui/view"
)

// testApp is a simple counter app for testing.
type testApp struct{}

func (a *testApp) View(s view.State) *view.Node {
	count := s.Get("count")
	if count == "" {
		count = "0"
	}
	return view.VBox("root",
		view.TextNode("label", "Count: "+count),
		view.Button("inc", "Increment").Prop("on", "inc"),
		view.TextBox("input").Prop("bind", "query"),
		view.Checkbox("done", "Done", s.Get("done") == "1").Prop("bind", "done"),
	)
}

func (a *testApp) Handle(s view.State, act *proto.Action) {
	switch act.Kind {
	case "click":
		if act.KVs["action"] == "inc" {
			n := 0
			if v := s.Get("count"); v != "" {
				for _, c := range v {
					n = n*10 + int(c-'0')
				}
			}
			n++
			s.Set("count", string(rune('0'+n%10)))
		}
	}
}

func TestUIFSBasic(t *testing.T) {
	u := New(&testApp{})

	// Get initial tree
	tree := u.Tree()
	if tree == nil {
		t.Fatal("tree is nil")
	}
	if tree.Root != "root" {
		t.Errorf("root = %q", tree.Root)
	}
	if len(tree.Nodes) != 5 {
		t.Errorf("nodes = %d, want 5", len(tree.Nodes))
	}

	// Tree text should be readable
	text := u.TreeText()
	if !strings.Contains(text, "Count: 0") {
		t.Errorf("tree text missing 'Count: 0', got:\n%s", text)
	}
}

func TestUIFSAction(t *testing.T) {
	u := New(&testApp{})
	u.ActionLog = []string{} // enable logging

	// Process increment action
	err := u.ProcessAction(`click id=inc button=1 action=inc x=0 y=0`)
	if err != nil {
		t.Fatal(err)
	}

	// Check count was incremented
	text := u.TreeText()
	if !strings.Contains(text, "Count: 1") {
		t.Errorf("after inc, tree text missing 'Count: 1', got:\n%s", text)
	}

	// Check action log
	if len(u.ActionLog) != 1 {
		t.Errorf("action log = %d entries", len(u.ActionLog))
	}
}

func TestUIFSBinding(t *testing.T) {
	u := New(&testApp{})

	// Simulate input action on textbox with bind=query
	u.HandleAction(&proto.Action{
		Kind: "input",
		KVs:  map[string]string{"id": "input", "text": "hello"},
	})

	// Check state was updated via binding
	if v := u.GetState("query"); v != "hello" {
		t.Errorf("query = %q, want hello", v)
	}

	// Toggle checkbox
	u.HandleAction(&proto.Action{
		Kind: "toggle",
		KVs:  map[string]string{"id": "done", "value": "1"},
	})
	if v := u.GetState("done"); v != "1" {
		t.Errorf("done = %q, want 1", v)
	}
}

func TestUIFSState(t *testing.T) {
	u := New(&testApp{})

	u.SetState("foo", "bar")
	if v := u.GetState("foo"); v != "bar" {
		t.Errorf("foo = %q", v)
	}

	// SetState should invalidate tree
	rev1 := u.Rev()
	u.SetState("x", "y")
	tree := u.Tree()
	if tree.Rev <= rev1 {
		t.Errorf("rev did not increase: %d <= %d", tree.Rev, rev1)
	}
}

func TestUIFSFocus(t *testing.T) {
	u := New(&testApp{})

	u.SetFocus("input")
	if u.Focus != "input" {
		t.Errorf("focus = %q", u.Focus)
	}
}
