package proto

import (
	"strings"
	"testing"
)

func TestEscapeValueSimple(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"hello", "hello"},
		{"foo_bar", "foo_bar"},
		{"123", "123"},
	}
	for _, tt := range tests {
		got := EscapeValue(tt.in)
		if got != tt.want {
			t.Errorf("EscapeValue(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestEscapeValueQuoted(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", `""`},
		{"hello world", `"hello world"`},
		{"line\none", `"line\none"`},
		{"tab\there", `"tab\there"`},
		{`back\slash`, `"back\\slash"`},
		{`say "hi"`, `"say \"hi\""`},
		{"has=eq", `"has=eq"`},
	}
	for _, tt := range tests {
		got := EscapeValue(tt.in)
		if got != tt.want {
			t.Errorf("EscapeValue(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestUnescapeValue(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"hello", "hello"},
		{`"hello world"`, "hello world"},
		{`"line\none"`, "line\none"},
		{`"tab\there"`, "tab\there"},
		{`"back\\slash"`, "back\\slash"},
		{`"say \"hi\""`, `say "hi"`},
		{`""`, ""},
	}
	for _, tt := range tests {
		got := UnescapeValue(tt.in)
		if got != tt.want {
			t.Errorf("UnescapeValue(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestEscapeUnescapeRoundtrip(t *testing.T) {
	values := []string{
		"", "hello", "hello world", "line\none",
		"tab\there", `back\slash`, `say "hi"`,
		"日本語", "mixed 日本 text", "a=b",
	}
	for _, v := range values {
		escaped := EscapeValue(v)
		got := UnescapeValue(escaped)
		if got != v {
			t.Errorf("roundtrip(%q): escaped=%q, unescaped=%q", v, escaped, got)
		}
	}
}

func TestParseKV(t *testing.T) {
	tests := []struct {
		in     string
		wantK  string
		wantV  string
		wantOK bool
	}{
		{"key=value", "key", "value", true},
		{`text="hello world"`, "text", "hello world", true},
		{`fg=red`, "fg", "red", true},
		{"noequal", "", "", false},
	}
	for _, tt := range tests {
		k, v, ok := ParseKV(tt.in)
		if ok != tt.wantOK || k != tt.wantK || v != tt.wantV {
			t.Errorf("ParseKV(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.in, k, v, ok, tt.wantK, tt.wantV, tt.wantOK)
		}
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"hello world", []string{"hello", "world"}},
		{`prop n1 text="hello world" fg=red`, []string{"prop", "n1", `text="hello world"`, "fg=red"}},
		{`click id=btn1 button=1`, []string{"click", "id=btn1", "button=1"}},
		{"  spaces  around  ", []string{"spaces", "around"}},
	}
	for _, tt := range tests {
		got := Tokenize(tt.in)
		if len(got) != len(tt.want) {
			t.Errorf("Tokenize(%q) = %v (len %d), want %v (len %d)",
				tt.in, got, len(got), tt.want, len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("Tokenize(%q)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
			}
		}
	}
}

func TestSerializeParseTree(t *testing.T) {
	tree := &Tree{
		Rev:  42,
		Root: "root",
		Nodes: map[string]*Node{
			"root": {
				ID: "root", Type: "vbox",
				Props:    map[string]string{"pad": "8", "gap": "4"},
				Children: []string{"title", "list"},
			},
			"title": {
				ID: "title", Type: "text",
				Props: map[string]string{"text": "Hello World"},
			},
			"list": {
				ID: "list", Type: "scroll",
				Props: map[string]string{"scroll": "auto"},
			},
		},
		Order: []string{"root", "title", "list"},
	}

	text := SerializeTree(tree)

	// Verify it contains expected lines
	if !strings.Contains(text, "rev 42") {
		t.Error("missing rev")
	}
	if !strings.Contains(text, "root root") {
		t.Error("missing root")
	}
	if !strings.Contains(text, "node root vbox") {
		t.Error("missing node root")
	}
	if !strings.Contains(text, "child root title") {
		t.Error("missing child root title")
	}

	// Parse it back
	parsed, err := ParseTree(text)
	if err != nil {
		t.Fatalf("ParseTree: %v", err)
	}
	if parsed.Rev != 42 {
		t.Errorf("parsed rev = %d, want 42", parsed.Rev)
	}
	if parsed.Root != "root" {
		t.Errorf("parsed root = %q, want root", parsed.Root)
	}
	if len(parsed.Nodes) != 3 {
		t.Errorf("parsed %d nodes, want 3", len(parsed.Nodes))
	}
	rootNode := parsed.Nodes["root"]
	if rootNode == nil {
		t.Fatal("missing root node")
	}
	if rootNode.Type != "vbox" {
		t.Errorf("root type = %q, want vbox", rootNode.Type)
	}
	if rootNode.Props["pad"] != "8" {
		t.Errorf("root pad = %q, want 8", rootNode.Props["pad"])
	}
	if len(rootNode.Children) != 2 {
		t.Errorf("root children = %d, want 2", len(rootNode.Children))
	}
	titleNode := parsed.Nodes["title"]
	if titleNode == nil {
		t.Fatal("missing title node")
	}
	if titleNode.Props["text"] != "Hello World" {
		t.Errorf("title text = %q, want 'Hello World'", titleNode.Props["text"])
	}
}

func TestSerializeParseAction(t *testing.T) {
	action := &Action{
		Kind: "click",
		KVs: map[string]string{
			"id":     "btn1",
			"button": "1",
			"x":      "100",
			"y":      "200",
		},
	}

	text := SerializeAction(action)

	// Parse it back
	parsed, err := ParseAction(text)
	if err != nil {
		t.Fatalf("ParseAction: %v", err)
	}
	if parsed.Kind != "click" {
		t.Errorf("kind = %q, want click", parsed.Kind)
	}
	if parsed.KVs["id"] != "btn1" {
		t.Errorf("id = %q, want btn1", parsed.KVs["id"])
	}
	if parsed.KVs["button"] != "1" {
		t.Errorf("button = %q, want 1", parsed.KVs["button"])
	}
}

func TestParseActionWithQuotedValue(t *testing.T) {
	line := `input id=tb1 text="hello world" cursor=5`
	a, err := ParseAction(line)
	if err != nil {
		t.Fatalf("ParseAction: %v", err)
	}
	if a.Kind != "input" {
		t.Errorf("kind = %q, want input", a.Kind)
	}
	if a.KVs["text"] != "hello world" {
		t.Errorf("text = %q, want 'hello world'", a.KVs["text"])
	}
	if a.KVs["cursor"] != "5" {
		t.Errorf("cursor = %q, want 5", a.KVs["cursor"])
	}
}
