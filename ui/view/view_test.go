package view

import (
	"testing"

	"github.com/elizafairlady/go-libui/ui/proto"
)

func TestNodeBuilders(t *testing.T) {
	root := VBox("root",
		TextNode("title", "Hello"),
		HBox("bar",
			Button("ok", "OK").Prop("on", "submit"),
			Button("cancel", "Cancel"),
		),
		TextBox("input").Prop("bind", "state/query").Prop("placeholder", "Search..."),
		Checkbox("done", "Done", true),
	)

	if root.Type != "vbox" {
		t.Errorf("root type = %q", root.Type)
	}
	if len(root.Children) != 4 {
		t.Fatalf("root children = %d, want 4", len(root.Children))
	}
	if root.Children[0].Props["text"] != "Hello" {
		t.Errorf("title text = %q", root.Children[0].Props["text"])
	}
	ok := root.Children[1].Children[0]
	if ok.Props["on"] != "submit" {
		t.Errorf("ok button on = %q", ok.Props["on"])
	}
	tb := root.Children[2]
	if tb.Props["bind"] != "state/query" {
		t.Errorf("textbox bind = %q", tb.Props["bind"])
	}
	cb := root.Children[3]
	if cb.Props["checked"] != "1" {
		t.Errorf("checkbox checked = %q", cb.Props["checked"])
	}
}

func TestSerialize(t *testing.T) {
	root := VBox("root",
		TextNode("t1", "Hello"),
		Button("b1", "OK"),
	)
	tree := Serialize(root, 7)

	if tree.Rev != 7 {
		t.Errorf("rev = %d", tree.Rev)
	}
	if tree.Root != "root" {
		t.Errorf("root = %q", tree.Root)
	}
	if len(tree.Nodes) != 3 {
		t.Errorf("nodes = %d", len(tree.Nodes))
	}
	if len(tree.Order) != 3 {
		t.Errorf("order = %d", len(tree.Order))
	}
	// Check children
	rn := tree.Nodes["root"]
	if len(rn.Children) != 2 || rn.Children[0] != "t1" || rn.Children[1] != "b1" {
		t.Errorf("root children = %v", rn.Children)
	}

	// Roundtrip through proto serialization
	text := proto.SerializeTree(tree)
	parsed, err := proto.ParseTree(text)
	if err != nil {
		t.Fatalf("ParseTree: %v", err)
	}
	if parsed.Rev != 7 || parsed.Root != "root" {
		t.Errorf("roundtrip: rev=%d root=%q", parsed.Rev, parsed.Root)
	}
}

func TestMemState(t *testing.T) {
	s := NewMemState()

	// Get/Set
	s.Set("a", "1")
	if s.Get("a") != "1" {
		t.Errorf("Get(a) = %q", s.Get("a"))
	}
	if s.Get("b") != "" {
		t.Errorf("Get(b) = %q", s.Get("b"))
	}

	// Del
	s.Del("a")
	if s.Get("a") != "" {
		t.Errorf("after Del, Get(a) = %q", s.Get("a"))
	}

	// List
	s.Set("items/0/name", "apple")
	s.Set("items/0/done", "0")
	s.Set("items/1/name", "banana")
	children := s.List("items")
	if len(children) != 2 {
		t.Fatalf("List(items) = %v", children)
	}
	// Order not guaranteed, check both exist
	found := map[string]bool{}
	for _, c := range children {
		found[c] = true
	}
	if !found["0"] || !found["1"] {
		t.Errorf("List(items) = %v, want [0 1]", children)
	}

	// GetInt, GetBool
	s.Set("count", "42")
	if s.GetInt("count", 0) != 42 {
		t.Errorf("GetInt(count) = %d", s.GetInt("count", 0))
	}
	if s.GetInt("missing", 5) != 5 {
		t.Errorf("GetInt(missing) = %d", s.GetInt("missing", 5))
	}
	s.Set("flag", "1")
	if !s.GetBool("flag") {
		t.Error("GetBool(flag) = false")
	}
	s.Set("flag", "0")
	if s.GetBool("flag") {
		t.Error("GetBool(flag) = true")
	}
}

func TestNodePropInt(t *testing.T) {
	n := N("x", "rect").PropInt("pad", 8).PropInt("gap", 4)
	if n.Props["pad"] != "8" || n.Props["gap"] != "4" {
		t.Errorf("props = %v", n.Props)
	}
}
