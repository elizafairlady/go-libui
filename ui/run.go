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

	"github.com/elizafairlady/go-libui/draw"
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

	// Build initial tree and layout
	conf := r.LayoutConfig()
	repaint := func() {
		tree := u.Tree()
		if tree == nil {
			return
		}
		root := layout.Build(tree, conf)
		if root == nil {
			return
		}
		layout.Layout(root, d.ScreenImage.R, conf)
		r.Focus = u.Focus
		r.Paint(root)
	}

	repaint()

	// Event loop
	for {
		select {
		case m, ok := <-mc.C:
			if !ok {
				return nil
			}
			if m.Buttons == 0 {
				// Update hover
				tree := u.Tree()
				if tree != nil {
					root := layout.Build(tree, conf)
					if root != nil {
						layout.Layout(root, d.ScreenImage.R, conf)
						hit := layout.HitTest(root, m.Point)
						if hit != nil {
							r.Hover = hit.ID
						} else {
							r.Hover = ""
						}
					}
				}
				repaint()
				continue
			}

			// Click
			tree := u.Tree()
			if tree == nil {
				continue
			}
			root := layout.Build(tree, conf)
			if root == nil {
				continue
			}
			layout.Layout(root, d.ScreenImage.R, conf)
			hit := layout.HitTest(root, m.Point)
			if hit != nil {
				// Update focus
				if u.Focus != hit.ID {
					u.SetFocus(hit.ID)
					r.Focus = hit.ID
				}
				// Determine button
				button := 1
				if m.Buttons&2 != 0 {
					button = 2
				} else if m.Buttons&4 != 0 {
					button = 3
				}

				// Tag nodes get special handling
				if hit.Type == "tag" {
					mc.Mouse = m
					act := r.TagClick(hit.ID, mc, button)
					if act != nil {
						u.HandleAction(act)
					}
				} else {
					act := render.MouseAction(hit, button, m.Point)
					if act != nil {
						u.HandleAction(act)
					}
				}
			}
			repaint()

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
				tree := u.Tree()
				if tree != nil {
					root := layout.Build(tree, conf)
					if root != nil {
						layout.Layout(root, d.ScreenImage.R, conf)
						next := render.NextFocusable(root, u.Focus)
						u.SetFocus(next)
						r.Focus = next
					}
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
			// Tags are renderer-owned; typing doesn't go through UIFS.
			// Only flush the display after frame.Insert/Delete draws.
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

		case <-mc.Resize:
			d.GetWindow(draw.Refnone)
			r.Screen = d.ScreenImage
			repaint()
		}
	}
}
