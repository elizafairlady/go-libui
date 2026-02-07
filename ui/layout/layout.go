// Package layout implements the box layout engine for the UI framework.
//
// The engine performs two passes:
//  1. Measure: computes intrinsic/minimum sizes bottom-up.
//  2. Layout: assigns rectangles top-down with flex distribution.
//
// Supported container types: vbox, hbox, stack, scroll.
// Leaf types: text, button, checkbox, textbox, rect, spacer, row.
package layout

import (
	"strconv"

	"github.com/elizafairlady/go-libui/draw"
	"github.com/elizafairlady/go-libui/ui/proto"
)

// RNode is a resolved node used for layout and rendering.
type RNode struct {
	ID       string
	Type     string
	Props    map[string]string
	Parent   *RNode
	Children []*RNode
	// Layout results
	Rect draw.Rectangle // assigned rectangle
	MinW int            // minimum width
	MinH int            // minimum height
	Flex int            // flex weight (0=fixed, >0=flex)
}

// FontMeasure is called to measure text dimensions.
type FontMeasure func(text string, font string, size int) (w, h int)

// Config holds layout configuration.
type Config struct {
	Measure    FontMeasure
	DefaultPad int
	DefaultGap int
	FontHeight int // default font height for sizing
}

// Build creates an RNode tree from a proto.Tree.
func Build(t *proto.Tree, conf *Config) *RNode {
	if t.Root == "" || t.Nodes[t.Root] == nil {
		return nil
	}
	cache := make(map[string]*RNode)
	root := buildNode(t, t.Root, nil, cache)
	Measure(root, conf)
	return root
}

func buildNode(t *proto.Tree, id string, parent *RNode, cache map[string]*RNode) *RNode {
	if rn, ok := cache[id]; ok {
		return rn
	}
	pn := t.Nodes[id]
	if pn == nil {
		return nil
	}
	rn := &RNode{
		ID:     pn.ID,
		Type:   pn.Type,
		Props:  pn.Props,
		Parent: parent,
		Flex:   propInt(pn.Props, "flex", 0),
	}
	cache[id] = rn
	for _, childID := range pn.Children {
		child := buildNode(t, childID, rn, cache)
		if child != nil {
			rn.Children = append(rn.Children, child)
		}
	}
	return rn
}

// --- Measure pass ---

// Measure computes minimum sizes bottom-up.
func Measure(n *RNode, conf *Config) {
	if n == nil {
		return
	}
	for _, child := range n.Children {
		Measure(child, conf)
	}

	pad := propIntDef(n.Props, "pad", conf.DefaultPad)
	gap := propIntDef(n.Props, "gap", conf.DefaultGap)
	minw := propInt(n.Props, "minw", 0)
	minh := propInt(n.Props, "minh", 0)

	switch n.Type {
	case "vbox", "scroll":
		w := 0
		h := pad * 2
		for i, c := range n.Children {
			if c.MinW > w {
				w = c.MinW
			}
			h += c.MinH
			if i > 0 {
				h += gap
			}
		}
		w += pad * 2
		n.MinW = max(w, minw)
		n.MinH = max(h, minh)

	case "hbox", "row":
		w := pad * 2
		h := 0
		for i, c := range n.Children {
			w += c.MinW
			if c.MinH > h {
				h = c.MinH
			}
			if i > 0 {
				w += gap
			}
		}
		h += pad * 2
		n.MinW = max(w, minw)
		n.MinH = max(h, minh)

	case "stack":
		w := 0
		h := 0
		for _, c := range n.Children {
			if c.MinW > w {
				w = c.MinW
			}
			if c.MinH > h {
				h = c.MinH
			}
		}
		w += pad * 2
		h += pad * 2
		n.MinW = max(w, minw)
		n.MinH = max(h, minh)

	case "text":
		text := n.Props["text"]
		w, h := measureText(conf, n.Props, text)
		w += pad * 2
		h += pad * 2
		n.MinW = max(w, minw)
		n.MinH = max(h, minh)

	case "button":
		text := n.Props["text"]
		w, h := measureText(conf, n.Props, text)
		w += pad*2 + 4 // extra for button decoration
		h += pad*2 + 2
		n.MinW = max(w, minw)
		n.MinH = max(h, minh)

	case "checkbox":
		text := n.Props["text"]
		w, h := measureText(conf, n.Props, text)
		w += pad*2 + conf.FontHeight + 4 // box + gap + text
		h += pad * 2
		if h < conf.FontHeight+pad*2 {
			h = conf.FontHeight + pad*2
		}
		n.MinW = max(w, minw)
		n.MinH = max(h, minh)

	case "textbox":
		h := conf.FontHeight + pad*2 + 2 // border
		w := 80                          // default min width
		n.MinW = max(w, minw)
		n.MinH = max(h, minh)

	case "tag":
		// Tag is a text frame — needs at least one line height
		text := n.Props["text"]
		w, _ := measureText(conf, n.Props, text)
		w += pad * 2
		h := conf.FontHeight + pad*2 // one line minimum
		n.MinW = max(w, minw)
		n.MinH = max(h, minh)

	case "rect":
		n.MinW = max(minw, 1)
		n.MinH = max(minh, 1)

	case "spacer":
		n.MinW = minw
		n.MinH = minh
		if n.Flex == 0 {
			n.Flex = 1
		}

	default:
		// Unknown type: treat like vbox
		w := 0
		h := pad * 2
		for i, c := range n.Children {
			if c.MinW > w {
				w = c.MinW
			}
			h += c.MinH
			if i > 0 {
				h += gap
			}
		}
		w += pad * 2
		n.MinW = max(w, minw)
		n.MinH = max(h, minh)
	}
}

func measureText(conf *Config, props map[string]string, text string) (int, int) {
	if conf.Measure != nil {
		font := props["font"]
		size := propInt(props, "size", 0)
		w, h := conf.Measure(text, font, size)
		return w, h
	}
	// Fallback: estimate
	w := len(text) * (conf.FontHeight * 6 / 10) // rough monospace estimate
	h := conf.FontHeight
	if w < 1 {
		w = 1
	}
	return w, h
}

// --- Layout pass ---

// Layout assigns rectangles to the tree, starting from the given bounds.
func Layout(n *RNode, bounds draw.Rectangle, conf *Config) {
	if n == nil {
		return
	}
	n.Rect = bounds

	pad := propIntDef(n.Props, "pad", conf.DefaultPad)
	gap := propIntDef(n.Props, "gap", conf.DefaultGap)
	inner := draw.Rect(
		bounds.Min.X+pad, bounds.Min.Y+pad,
		bounds.Max.X-pad, bounds.Max.Y-pad,
	)

	switch n.Type {
	case "vbox", "scroll":
		layoutBox(n.Children, inner, gap, true, conf)

	case "hbox", "row":
		layoutBox(n.Children, inner, gap, false, conf)

	case "stack":
		for _, c := range n.Children {
			Layout(c, inner, conf)
		}

	default:
		// Leaf or unknown: children get inner rect
		for _, c := range n.Children {
			Layout(c, inner, conf)
		}
	}
}

// layoutBox distributes space among children along an axis.
// If vertical=true, distributes along Y; otherwise along X.
func layoutBox(children []*RNode, bounds draw.Rectangle, gap int, vertical bool, conf *Config) {
	if len(children) == 0 {
		return
	}

	totalAvail := bounds.Dy()
	if !vertical {
		totalAvail = bounds.Dx()
	}

	// Calculate fixed size and total flex weight
	fixedSize := gap * (len(children) - 1)
	totalFlex := 0
	for _, c := range children {
		if c.Flex > 0 {
			totalFlex += c.Flex
		} else {
			if vertical {
				fixedSize += c.MinH
			} else {
				fixedSize += c.MinW
			}
		}
	}

	flexSpace := totalAvail - fixedSize
	if flexSpace < 0 {
		flexSpace = 0
	}

	pos := bounds.Min.Y
	if !vertical {
		pos = bounds.Min.X
	}

	for _, c := range children {
		var size int
		if c.Flex > 0 && totalFlex > 0 {
			size = flexSpace * c.Flex / totalFlex
		} else {
			if vertical {
				size = c.MinH
			} else {
				size = c.MinW
			}
		}

		var r draw.Rectangle
		if vertical {
			r = draw.Rect(bounds.Min.X, pos, bounds.Max.X, pos+size)
		} else {
			r = draw.Rect(pos, bounds.Min.Y, pos+size, bounds.Max.Y)
		}

		// Enforce max constraints
		if maxw := propInt(c.Props, "maxw", 0); maxw > 0 && r.Dx() > maxw {
			r.Max.X = r.Min.X + maxw
		}
		if maxh := propInt(c.Props, "maxh", 0); maxh > 0 && r.Dy() > maxh {
			r.Max.Y = r.Min.Y + maxh
		}

		Layout(c, r, conf)
		pos += size + gap
	}
}

// --- Helpers ---

func propInt(props map[string]string, key string, def int) int {
	v, ok := props[key]
	if !ok {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func propIntDef(props map[string]string, key string, def int) int {
	return propInt(props, key, def)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Flatten returns all nodes in the tree in depth-first order.
func Flatten(n *RNode) []*RNode {
	if n == nil {
		return nil
	}
	result := []*RNode{n}
	for _, c := range n.Children {
		result = append(result, Flatten(c)...)
	}
	return result
}

// HitTest finds the deepest node at point pt that has focusable=1
// or is interactive (button, checkbox, textbox, row).
func HitTest(n *RNode, pt draw.Point) *RNode {
	if n == nil || !pt.In(n.Rect) {
		return nil
	}
	// Check children in reverse order (last = topmost)
	for i := len(n.Children) - 1; i >= 0; i-- {
		if hit := HitTest(n.Children[i], pt); hit != nil {
			return hit
		}
	}
	// Self: is this interactive?
	if isInteractive(n) {
		return n
	}
	return nil
}

func isInteractive(n *RNode) bool {
	switch n.Type {
	case "button", "checkbox", "textbox", "tag", "row":
		return true
	}
	return n.Props["focusable"] == "1"
}
