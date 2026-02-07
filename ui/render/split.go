// split.go implements rendering and drag interaction for splitbox containers.
//
// A splitbox divides space between children with thin drag handles
// between them. Dragging a handle redistributes space by updating the
// weights stored in the renderer's SplitWeights map.
package render

import (
	"strconv"

	"github.com/elizafairlady/go-libui/draw"
	"github.com/elizafairlady/go-libui/ui/layout"
	"github.com/elizafairlady/go-libui/ui/proto"
)

// paintSplitBox renders a splitbox: its children plus the drag handles.
func (r *Renderer) paintSplitBox(n *layout.RNode) {
	// Paint children first
	for _, c := range n.Children {
		r.paintNode(c)
	}

	// Paint drag handles between children
	handles := layout.SplitHandleRects(n)
	bord := r.colorImage(draw.DAcmeBorder)
	if bord == nil {
		return
	}
	for _, hr := range handles {
		r.Screen.Draw(hr, bord, draw.ZP)
	}
}

// SplitHitHandle checks if a point hits a splitbox drag handle.
// Returns the splitbox node ID, handle index, and true if hit.
func (r *Renderer) SplitHitHandle(root *layout.RNode, pt draw.Point) (string, int, bool) {
	nodes := layout.Flatten(root)
	for _, n := range nodes {
		if n.Type != "splitbox" || !pt.In(n.Rect) {
			continue
		}
		handles := layout.SplitHandleRects(n)
		for i, hr := range handles {
			if pt.In(hr) {
				return n.ID, i, true
			}
		}
	}
	return "", 0, false
}

// SplitDrag handles dragging a splitbox handle.
// It reads mouse events until the button is released, updating
// the weights in real time.
func (r *Renderer) SplitDrag(splitID string, handleIdx int, mc *draw.Mousectl, root *layout.RNode, conf *layout.Config, repaint func()) {
	// Find the splitbox node
	node := findNode(root, splitID)
	if node == nil || handleIdx < 0 || handleIdx >= len(node.Children)-1 {
		return
	}

	vertical := node.Props["direction"] != "horizontal"

	// Get current weights
	weights := getWeights(r, splitID, len(node.Children))

	for {
		mc.ReadMouse()
		if mc.Mouse.Buttons == 0 {
			break
		}

		// Calculate new split position based on mouse
		var mousePos int
		var startPos int
		var totalSize int
		if vertical {
			mousePos = mc.Mouse.Y
			startPos = node.Children[0].Rect.Min.Y
			totalSize = node.Children[len(node.Children)-1].Rect.Max.Y - startPos
		} else {
			mousePos = mc.Mouse.X
			startPos = node.Children[0].Rect.Min.X
			totalSize = node.Children[len(node.Children)-1].Rect.Max.X - startPos
		}

		handleSpace := layout.SplitHandleSize * (len(node.Children) - 1)
		distributable := totalSize - handleSpace
		if distributable <= 0 {
			continue
		}

		// Position relative to start
		relPos := mousePos - startPos
		if relPos < 0 {
			relPos = 0
		}
		if relPos > totalSize {
			relPos = totalSize
		}

		// Calculate sizes for the two children around the handle
		// Sum of weights before and after the handle
		totalWeight := 0
		for _, w := range weights {
			totalWeight += w
		}

		// The handle splits children [handleIdx] and [handleIdx+1].
		// Calculate position as fraction of total.
		beforeSize := 0
		for i := 0; i < handleIdx; i++ {
			beforeSize += distributable * weights[i] / totalWeight
			beforeSize += layout.SplitHandleSize
		}

		// Size of child handleIdx based on mouse position
		newChildSize := relPos - beforeSize
		if newChildSize < layout.SplitHandleSize {
			newChildSize = layout.SplitHandleSize
		}

		// Size of child handleIdx+1
		afterStart := relPos + layout.SplitHandleSize
		remainingSize := 0
		for i := handleIdx + 2; i < len(node.Children); i++ {
			remainingSize += distributable * weights[i] / totalWeight
			if i < len(node.Children)-1 {
				remainingSize += layout.SplitHandleSize
			}
		}
		nextChildSize := totalSize - afterStart - remainingSize
		if nextChildSize < layout.SplitHandleSize {
			nextChildSize = layout.SplitHandleSize
		}

		// Convert pixel sizes to weights
		// Use pixel sizes directly as weights for simplicity
		weights[handleIdx] = newChildSize
		weights[handleIdx+1] = nextChildSize

		// Store weights
		setWeights(r, splitID, weights)

		// Relayout and repaint
		if repaint != nil {
			repaint()
		}
	}
}

// GetSplitWeights returns the current weights string for a splitbox,
// usable as a node prop override.
func (r *Renderer) GetSplitWeights(id string) string {
	if r.SplitWeights == nil {
		return ""
	}
	return r.SplitWeights[id]
}

func getWeights(r *Renderer, id string, n int) []int {
	weights := make([]int, n)
	for i := range weights {
		weights[i] = 1
	}
	if r.SplitWeights == nil {
		return weights
	}
	s := r.SplitWeights[id]
	if s == "" {
		return weights
	}
	parts := splitCommaStr(s)
	for i := 0; i < n && i < len(parts); i++ {
		v, err := strconv.Atoi(parts[i])
		if err == nil && v > 0 {
			weights[i] = v
		}
	}
	return weights
}

func setWeights(r *Renderer, id string, weights []int) {
	if r.SplitWeights == nil {
		r.SplitWeights = make(map[string]string)
	}
	s := ""
	for i, w := range weights {
		if i > 0 {
			s += ","
		}
		s += strconv.Itoa(w)
	}
	r.SplitWeights[id] = s
}

func splitCommaStr(s string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	return parts
}

func findNode(root *layout.RNode, id string) *layout.RNode {
	if root == nil {
		return nil
	}
	if root.ID == id {
		return root
	}
	for _, c := range root.Children {
		if found := findNode(c, id); found != nil {
			return found
		}
	}
	return nil
}

// Unused import suppressor
var _ = proto.SerializeAction
