// Package ui provides the top-level API for running a UI application.
//
// Example usage:
//
//	err := ui.Run("My App", &myApp{})
//	if err != nil {
//		log.Fatal(err)
//	}
package ui

import (
	"fmt"
	"os"

	"github.com/elizafairlady/go-libui/draw"
	"github.com/elizafairlady/go-libui/ui/fsys"
	"github.com/elizafairlady/go-libui/ui/layout"
	"github.com/elizafairlady/go-libui/ui/proto"
	"github.com/elizafairlady/go-libui/ui/render"
	"github.com/elizafairlady/go-libui/ui/theme"
	"github.com/elizafairlady/go-libui/ui/uifs"
	"github.com/elizafairlady/go-libui/ui/view"
)

// Run creates a window, initializes the display, and runs the
// event loop for the given app. This is the main entry point.
func Run(title string, app view.App) error {
	d, err := draw.Init(nil, "", title)
	if err != nil {
		return fmt.Errorf("ui: init display: %w", err)
	}
	defer d.Close()

	mc, err := draw.InitMouse("", d.ScreenImage)
	if err != nil {
		return fmt.Errorf("ui: init mouse: %w", err)
	}
	defer mc.Close()

	kc, err := draw.InitKeyboard("")
	if err != nil {
		return fmt.Errorf("ui: init keyboard: %w", err)
	}
	defer kc.Close()

	th := theme.Default()
	r := render.New(d, th)
	u := uifs.New(app)

	// If the app provides body buffers, wire them to the renderer
	if bp, ok := app.(render.BodyBufferProvider); ok {
		r.BufferProvider = bp
	}

	// Start the 9P state server — post to /srv so clients can mount it
	prov := &stateProvider{u: u, r: r}
	srv := fsys.NewStateServer(prov)
	srvName := uiSrvName(title)
	if err := srv.Post(srvName); err != nil {
		// Non-fatal: log and continue without 9P
		fmt.Fprintf(os.Stderr, "ui: 9P server: %v\n", err)
	}

	// Create executor for B2 command handling
	ex := newExecutor(app, u, r)

	// Build initial tree and layout
	conf := r.LayoutConfig()

	buildAndLayout := func() (*proto.Tree, *layout.RNode) {
		tree := u.Tree()
		if tree == nil {
			return nil, nil
		}
		// Apply renderer-persisted split weights to tree nodes
		applySplitWeights(tree, r)
		root := layout.Build(tree, conf)
		if root == nil {
			return tree, nil
		}
		layout.Layout(root, d.ScreenImage.R, conf)
		return tree, root
	}

	// checkQuit returns true if the app requested exit.
	checkQuit := func() bool {
		return u.GetState("_quit") == "1"
	}

	repaint := func() {
		tree, root := buildAndLayout()
		if tree == nil || root == nil {
			return
		}
		r.Focus = u.Focus
		r.Paint(root)
	}

	repaint()
	if checkQuit() {
		return nil
	}

	// Event loop
	for {
		select {
		case m, ok := <-mc.C:
			if !ok {
				return nil
			}
			if m.Buttons == 0 {
				// Update hover
				tree, root := buildAndLayout()
				if tree != nil && root != nil {
					hit := layout.HitTest(root, m.Point)
					if hit != nil {
						r.Hover = hit.ID
					} else {
						r.Hover = ""
					}
				}
				repaint()
				continue
			}

			// Click
			tree, root := buildAndLayout()
			if tree == nil || root == nil {
				continue
			}

			// Determine button
			button := 1
			if m.Buttons&2 != 0 {
				button = 2
			} else if m.Buttons&4 != 0 {
				button = 3
			}

			// Check for splitbox handle drag first (B1 only)
			if button == 1 {
				if splitID, handleIdx, ok := r.SplitHitHandle(root, m.Point); ok {
					mc.Mouse = m
					r.SplitDrag(splitID, handleIdx, mc, root, conf, func() {
						// Re-apply weights and repaint during drag
						tree := u.Tree()
						if tree == nil {
							return
						}
						applySplitWeights(tree, r)
						newRoot := layout.Build(tree, conf)
						if newRoot == nil {
							return
						}
						layout.Layout(newRoot, d.ScreenImage.R, conf)
						r.Focus = u.Focus
						r.Paint(newRoot)
					})
					repaint()
					continue
				}
			}

			hit := layout.HitTest(root, m.Point)
			if hit != nil {
				// Update focus
				if u.Focus != hit.ID {
					u.SetFocus(hit.ID)
					r.Focus = hit.ID
				}

				switch hit.Type {
				case "tag":
					// Tag nodes get special handling
					mc.Mouse = m
					act := r.TagClick(hit.ID, mc, button)
					if act != nil {
						// B2 execute: try executor first
						if act.Kind == "execute" && ex.execute(act) {
							u.Invalidate() // builtin modified state; dirty the tree
						} else {
							u.HandleAction(act)
						}
					}

				case "body":
					// Body nodes get frame-based handling
					mc.Mouse = m
					act := r.BodyClick(hit.ID, mc, button)
					if act != nil {
						// B2 execute: try executor first
						if act.Kind == "execute" && ex.execute(act) {
							u.Invalidate() // builtin modified state; dirty the tree
						} else {
							u.HandleAction(act)
						}
					}

				default:
					act := render.MouseAction(hit, button, m.Point)
					if act != nil {
						u.HandleAction(act)
					}
				}
			}
			repaint()
			if checkQuit() {
				return nil
			}

		case key, ok := <-kc.C:
			if !ok {
				return nil
			}
			if key == 0 {
				continue
			}
			// Handle keyboard
			name := render.KeyName(key)

			// Tab navigation
			if key == '\t' {
				_, root := buildAndLayout()
				if root != nil {
					next := render.NextFocusable(root, u.Focus)
					u.SetFocus(next)
					r.Focus = next
				}
				repaint()
				continue
			}

			// Enter on button
			if (key == '\n' || key == '\r') && u.Focus != "" {
				tree := u.Tree()
				if tree != nil {
					node := tree.Nodes[u.Focus]
					if node != nil && node.Type == "button" {
						act := &proto.Action{
							Kind: "click",
							KVs: map[string]string{
								"id":     u.Focus,
								"button": "1",
							},
						}
						if on := node.Props["on"]; on != "" {
							act.KVs["action"] = on
						}
						u.HandleAction(act)
						repaint()
						continue
					}
				}
			}

			// Space on checkbox
			if key == ' ' && u.Focus != "" {
				tree := u.Tree()
				if tree != nil {
					node := tree.Nodes[u.Focus]
					if node != nil && node.Type == "checkbox" {
						v := "1"
						if node.Props["checked"] == "1" {
							v = "0"
						}
						act := &proto.Action{
							Kind: "toggle",
							KVs: map[string]string{
								"id":    u.Focus,
								"value": v,
							},
						}
						u.HandleAction(act)
						repaint()
						continue
					}
				}
			}

			// Tag typing — editable tag bar
			if u.Focus != "" {
				tree := u.Tree()
				if tree != nil {
					node := tree.Nodes[u.Focus]
					if node != nil && node.Type == "tag" {
						r.TagType(u.Focus, key)
						d.Flush()
						continue
					}
				}
			}

			// Body typing — multi-line editable text area
			if u.Focus != "" {
				tree := u.Tree()
				if tree != nil {
					node := tree.Nodes[u.Focus]
					if node != nil && node.Type == "body" {
						r.BodyType(u.Focus, key)
						d.Flush()
						continue
					}
				}
			}

			// Text input for textbox
			if u.Focus != "" {
				tree := u.Tree()
				if tree != nil {
					node := tree.Nodes[u.Focus]
					if node != nil && node.Type == "textbox" {
						text := u.GetState(node.Props["bind"])
						runes := []rune(text)

						switch {
						case key == draw.Kbs || key == draw.Kdel: // Backspace/Del
							if len(runes) > 0 {
								runes = runes[:len(runes)-1]
							}
						case key >= 32 && key < draw.KF: // Printable
							runes = append(runes, key)
						default:
							// Send generic key action
							act := render.KeyAction(u.Focus, key, name)
							u.HandleAction(act)
							repaint()
							continue
						}

						text = string(runes)
						cursor := len(runes)
						act := render.InputAction(u.Focus, text, cursor)
						u.HandleAction(act)
						repaint()
						continue
					}
				}
			}

			// Generic key action
			if u.Focus != "" {
				act := render.KeyAction(u.Focus, key, name)
				u.HandleAction(act)
			}

			// Quit on DEL (Ctrl+Q equivalent in Plan 9)
			if key == draw.Kdel {
				return nil
			}

			repaint()
			if checkQuit() {
				return nil
			}

		case <-mc.Resize:
			d.GetWindow(draw.Refnone)
			r.Screen = d.ScreenImage
			repaint()
		}
	}
}

// applySplitWeights updates splitbox nodes in the tree with
// renderer-persisted weights from drag operations.
func applySplitWeights(tree *proto.Tree, r *render.Renderer) {
	if r.SplitWeights == nil {
		return
	}
	for id, weights := range r.SplitWeights {
		if node, ok := tree.Nodes[id]; ok && node.Type == "splitbox" {
			node.Props["weights"] = weights
		}
	}
}
