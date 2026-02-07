// Package proto implements the text serialization formats for the
// UI framework's tree and action protocols.
//
// Tree format (line-oriented, deterministic, diff-friendly):
//
//	rev <uint64>
//	root <nodeid>
//	node <id> <type>
//	prop <id> <k>=<v> <k>=<v> ...
//	child <parent> <child>
//
// Action format (one per line):
//
//	<kind> <k>=<v> <k>=<v> ...
//
// String escaping: values containing spaces, tabs, newlines, or
// backslashes are quoted with double quotes. Inside quotes,
// \n, \t, \\, and \" are recognized escapes.
package proto

import (
	"fmt"
	"strconv"
	"strings"
)

// Node represents a node in the UI tree.
type Node struct {
	ID       string
	Type     string
	Props    map[string]string
	Children []string // child IDs in order
}

// Tree is a complete UI tree snapshot.
type Tree struct {
	Rev   uint64
	Root  string
	Nodes map[string]*Node // keyed by ID
	Order []string         // node IDs in declaration order
}

// Action is a semantic UI action.
type Action struct {
	Kind string
	KVs  map[string]string
}

// --- Escaping ---

// needsQuote reports whether the string needs quoting.
func needsQuote(s string) bool {
	if len(s) == 0 {
		return true
	}
	for _, c := range s {
		if c == ' ' || c == '\t' || c == '\n' || c == '\\' || c == '"' || c == '=' {
			return true
		}
	}
	return false
}

// EscapeValue encodes a string for the protocol, quoting if necessary.
func EscapeValue(s string) string {
	if !needsQuote(s) {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, c := range s {
		switch c {
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		default:
			b.WriteRune(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// UnescapeValue decodes a possibly-quoted protocol string.
func UnescapeValue(s string) string {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return s
	}
	s = s[1 : len(s)-1]
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			default:
				b.WriteByte(s[i])
				b.WriteByte(s[i+1])
			}
			i++
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// --- KV parsing ---

// FormatKV formats a key=value pair with proper escaping.
func FormatKV(k, v string) string {
	return k + "=" + EscapeValue(v)
}

// ParseKV parses a key=value token. Returns key, value, ok.
func ParseKV(token string) (string, string, bool) {
	eq := strings.IndexByte(token, '=')
	if eq < 0 {
		return "", "", false
	}
	k := token[:eq]
	v := UnescapeValue(token[eq+1:])
	return k, v, true
}

// --- Tokenization ---

// Tokenize splits a line into space-separated tokens, respecting
// quoted strings. Returns the tokens.
func Tokenize(line string) []string {
	var tokens []string
	i := 0
	for i < len(line) {
		// Skip whitespace
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		if i >= len(line) {
			break
		}
		if line[i] == '"' {
			// Quoted string: find matching close quote
			j := i + 1
			for j < len(line) {
				if line[j] == '\\' && j+1 < len(line) {
					j += 2
					continue
				}
				if line[j] == '"' {
					j++
					break
				}
				j++
			}
			tokens = append(tokens, line[i:j])
			i = j
		} else {
			// Unquoted token, but handle k="quoted" by looking ahead
			j := i
			for j < len(line) && line[j] != ' ' && line[j] != '\t' {
				if line[j] == '"' {
					// Inside a k=v where v is quoted
					j++
					for j < len(line) {
						if line[j] == '\\' && j+1 < len(line) {
							j += 2
							continue
						}
						if line[j] == '"' {
							j++
							break
						}
						j++
					}
					continue
				}
				j++
			}
			tokens = append(tokens, line[i:j])
			i = j
		}
	}
	return tokens
}

// --- Tree serialization ---

// SerializeTree encodes a tree to the text protocol format.
func SerializeTree(t *Tree) string {
	var b strings.Builder
	fmt.Fprintf(&b, "rev %d\n", t.Rev)
	fmt.Fprintf(&b, "root %s\n", t.Root)
	for _, id := range t.Order {
		n := t.Nodes[id]
		if n == nil {
			continue
		}
		fmt.Fprintf(&b, "node %s %s\n", n.ID, n.Type)
		if len(n.Props) > 0 {
			b.WriteString("prop ")
			b.WriteString(n.ID)
			// Sort keys for determinism
			keys := sortedKeys(n.Props)
			for _, k := range keys {
				b.WriteByte(' ')
				b.WriteString(FormatKV(k, n.Props[k]))
			}
			b.WriteByte('\n')
		}
		for _, child := range n.Children {
			fmt.Fprintf(&b, "child %s %s\n", n.ID, child)
		}
	}
	return b.String()
}

// sortedKeys returns map keys in sorted order.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple insertion sort (maps are typically small)
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

// ParseTree decodes a tree from the text protocol format.
func ParseTree(text string) (*Tree, error) {
	t := &Tree{
		Nodes: make(map[string]*Node),
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		tokens := Tokenize(line)
		if len(tokens) == 0 {
			continue
		}
		switch tokens[0] {
		case "rev":
			if len(tokens) < 2 {
				return nil, fmt.Errorf("proto: rev missing value")
			}
			v, err := strconv.ParseUint(tokens[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("proto: bad rev: %v", err)
			}
			t.Rev = v
		case "root":
			if len(tokens) < 2 {
				return nil, fmt.Errorf("proto: root missing value")
			}
			t.Root = tokens[1]
		case "node":
			if len(tokens) < 3 {
				return nil, fmt.Errorf("proto: node missing id or type")
			}
			id := tokens[1]
			typ := tokens[2]
			n := t.Nodes[id]
			if n == nil {
				n = &Node{ID: id, Type: typ, Props: make(map[string]string)}
				t.Nodes[id] = n
				t.Order = append(t.Order, id)
			} else {
				n.Type = typ
			}
		case "prop":
			if len(tokens) < 2 {
				return nil, fmt.Errorf("proto: prop missing id")
			}
			id := tokens[1]
			n := t.Nodes[id]
			if n == nil {
				n = &Node{ID: id, Props: make(map[string]string)}
				t.Nodes[id] = n
				t.Order = append(t.Order, id)
			}
			for _, kv := range tokens[2:] {
				k, v, ok := ParseKV(kv)
				if ok {
					n.Props[k] = v
				}
			}
		case "child":
			if len(tokens) < 3 {
				return nil, fmt.Errorf("proto: child missing parent or child")
			}
			parent := tokens[1]
			child := tokens[2]
			n := t.Nodes[parent]
			if n == nil {
				n = &Node{ID: parent, Props: make(map[string]string)}
				t.Nodes[parent] = n
				t.Order = append(t.Order, parent)
			}
			n.Children = append(n.Children, child)
		default:
			// Unknown directive: skip for forward compatibility
		}
	}
	return t, nil
}

// --- Action serialization ---

// SerializeAction encodes an action to the text protocol format.
func SerializeAction(a *Action) string {
	var b strings.Builder
	b.WriteString(a.Kind)
	keys := sortedKeys(a.KVs)
	for _, k := range keys {
		b.WriteByte(' ')
		b.WriteString(FormatKV(k, a.KVs[k]))
	}
	return b.String()
}

// ParseAction decodes an action from the text protocol format.
func ParseAction(line string) (*Action, error) {
	tokens := Tokenize(strings.TrimSpace(line))
	if len(tokens) == 0 {
		return nil, fmt.Errorf("proto: empty action")
	}
	a := &Action{
		Kind: tokens[0],
		KVs:  make(map[string]string),
	}
	for _, kv := range tokens[1:] {
		k, v, ok := ParseKV(kv)
		if ok {
			a.KVs[k] = v
		}
	}
	return a, nil
}
