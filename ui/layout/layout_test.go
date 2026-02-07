package layout

import (
	"testing"

	"github.com/elizafairlady/go-libui/draw"
	"github.com/elizafairlady/go-libui/ui/proto"
)

func testConfig() *Config {
	return &Config{
		DefaultPad: 4,
		DefaultGap: 2,
		FontHeight: 14,
	}
}

func TestBuildAndLayout(t *testing.T) {
	tree := &proto.Tree{
		Rev:  1,
		Root: "root",
		Nodes: map[string]*proto.Node{
			"root": {ID: "root", Type: "vbox", Props: map[string]string{"pad": "0", "gap": "0"}, Children: []string{"a", "b"}},
			"a":    {ID: "a", Type: "text", Props: map[string]string{"text": "Hello", "pad": "0"}},
			"b":    {ID: "b", Type: "text", Props: map[string]string{"text": "World", "pad": "0"}},
		},
		Order: []string{"root", "a", "b"},
	}

	conf := testConfig()
	root := Build(tree, conf)
	if root == nil {
		t.Fatal("Build returned nil")
	}
	if root.ID != "root" || root.Type != "vbox" {
		t.Errorf("root = %s %s", root.ID, root.Type)
	}
	if len(root.Children) != 2 {
		t.Fatalf("children = %d", len(root.Children))
	}

	bounds := draw.Rect(0, 0, 200, 100)
	Layout(root, bounds, conf)

	if root.Rect != bounds {
		t.Errorf("root rect = %v, want %v", root.Rect, bounds)
	}
	// a and b should stack vertically
	a := root.Children[0]
	b := root.Children[1]
	if a.Rect.Min.Y != 0 {
		t.Errorf("a.Min.Y = %d", a.Rect.Min.Y)
	}
	if b.Rect.Min.Y != a.Rect.Max.Y {
		t.Errorf("b.Min.Y = %d, want %d", b.Rect.Min.Y, a.Rect.Max.Y)
	}
}

func TestFlexLayout(t *testing.T) {
	tree := &proto.Tree{
		Rev:  1,
		Root: "root",
		Nodes: map[string]*proto.Node{
			"root": {ID: "root", Type: "vbox", Props: map[string]string{"pad": "0", "gap": "0"}, Children: []string{"top", "mid", "bot"}},
			"top":  {ID: "top", Type: "text", Props: map[string]string{"text": "T", "pad": "0"}},
			"mid":  {ID: "mid", Type: "spacer", Props: map[string]string{"flex": "1"}},
			"bot":  {ID: "bot", Type: "text", Props: map[string]string{"text": "B", "pad": "0"}},
		},
		Order: []string{"root", "top", "mid", "bot"},
	}

	conf := testConfig()
	root := Build(tree, conf)
	Layout(root, draw.Rect(0, 0, 200, 200), conf)

	top := root.Children[0]
	mid := root.Children[1]
	bot := root.Children[2]

	// Mid spacer should take remaining space
	if mid.Rect.Dy() <= 0 {
		t.Errorf("mid spacer height = %d", mid.Rect.Dy())
	}
	total := top.Rect.Dy() + mid.Rect.Dy() + bot.Rect.Dy()
	if total != 200 {
		t.Errorf("total height = %d, want 200", total)
	}
}

func TestHBox(t *testing.T) {
	tree := &proto.Tree{
		Rev:  1,
		Root: "root",
		Nodes: map[string]*proto.Node{
			"root": {ID: "root", Type: "hbox", Props: map[string]string{"pad": "0", "gap": "0"}, Children: []string{"a", "b"}},
			"a":    {ID: "a", Type: "rect", Props: map[string]string{"minw": "50", "minh": "20"}},
			"b":    {ID: "b", Type: "rect", Props: map[string]string{"minw": "30", "minh": "20"}},
		},
		Order: []string{"root", "a", "b"},
	}

	conf := testConfig()
	root := Build(tree, conf)
	Layout(root, draw.Rect(0, 0, 200, 50), conf)

	a := root.Children[0]
	b := root.Children[1]
	if a.Rect.Dx() != 50 {
		t.Errorf("a width = %d, want 50", a.Rect.Dx())
	}
	if b.Rect.Min.X != 50 {
		t.Errorf("b.Min.X = %d, want 50", b.Rect.Min.X)
	}
}

func TestHitTest(t *testing.T) {
	tree := &proto.Tree{
		Rev:  1,
		Root: "root",
		Nodes: map[string]*proto.Node{
			"root": {ID: "root", Type: "vbox", Props: map[string]string{"pad": "0", "gap": "0"}, Children: []string{"btn"}},
			"btn":  {ID: "btn", Type: "button", Props: map[string]string{"text": "OK", "pad": "0"}},
		},
		Order: []string{"root", "btn"},
	}

	conf := testConfig()
	root := Build(tree, conf)
	Layout(root, draw.Rect(0, 0, 200, 100), conf)

	hit := HitTest(root, draw.Pt(10, 5))
	if hit == nil || hit.ID != "btn" {
		id := ""
		if hit != nil {
			id = hit.ID
		}
		t.Errorf("hit = %q, want btn", id)
	}

	hit = HitTest(root, draw.Pt(10, 90))
	if hit != nil {
		t.Errorf("hit at 90 = %q, want nil", hit.ID)
	}
}

func TestFlatten(t *testing.T) {
	tree := &proto.Tree{
		Rev:  1,
		Root: "r",
		Nodes: map[string]*proto.Node{
			"r": {ID: "r", Type: "vbox", Props: map[string]string{}, Children: []string{"a", "b"}},
			"a": {ID: "a", Type: "text", Props: map[string]string{"text": "A"}},
			"b": {ID: "b", Type: "text", Props: map[string]string{"text": "B"}},
		},
		Order: []string{"r", "a", "b"},
	}

	conf := testConfig()
	root := Build(tree, conf)
	flat := Flatten(root)
	if len(flat) != 3 {
		t.Errorf("flatten = %d nodes, want 3", len(flat))
	}
}
