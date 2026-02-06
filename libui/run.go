package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Run starts the main event loop for the application.
// This is the heart of libui - a single blocking loop.
func Run(app App) error {
	// Initialize draw environment
	ctx, err := NewDrawContext()
	if err != nil {
		return fmt.Errorf("init draw: %w", err)
	}

	// Open input devices
	mouse, err := os.Open("/dev/mouse")
	if err != nil {
		return fmt.Errorf("open mouse: %w", err)
	}
	defer mouse.Close()

	kbd, err := os.Open("/dev/cons")
	if err != nil {
		return fmt.Errorf("open cons: %w", err)
	}
	defer kbd.Close()

	// Set console to raw mode
	consctl, err := os.OpenFile("/dev/consctl", os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("open consctl: %w", err)
	}
	defer consctl.Close()
	consctl.Write([]byte("rawon"))

	// Initialize state
	model := app.Model
	view := ViewState{}

	// Get initial size
	w, h := ctx.Bounds()
	view.Width = w
	view.Height = h

	// Initial draw
	ctx.Clear()
	app.Draw(model, ctx)
	ctx.Flush()

	// Create event channel
	events := make(chan Event, 10)

	// Mouse reader goroutine
	go func() {
		buf := make([]byte, 49)
		for {
			n, err := mouse.Read(buf)
			if err != nil {
				return
			}
			if n >= 49 && buf[0] == 'm' {
				// Parse mouse: m x y buttons time
				parts := strings.Fields(string(buf[1:n]))
				if len(parts) >= 3 {
					x, _ := strconv.Atoi(parts[0])
					y, _ := strconv.Atoi(parts[1])
					buttons, _ := strconv.Atoi(parts[2])
					m := Mouse{X: x, Y: y, Buttons: buttons}
					// Check for scroll wheel (buttons 8 and 16)
					if buttons&8 != 0 {
						m.ScrollY = -1
					} else if buttons&16 != 0 {
						m.ScrollY = 1
					}
					events <- Event{Kind: "mouse", Data: m}
				}
			} else if n > 0 && buf[0] == 'r' {
				// Resize event
				if err := ctx.Reattach(); err == nil {
					w, h := ctx.Bounds()
					events <- Event{Kind: "resize", Data: Resize{Width: w, Height: h}}
				}
			}
		}
	}()

	// Keyboard reader goroutine
	go func() {
		reader := bufio.NewReader(kbd)
		for {
			r, _, err := reader.ReadRune()
			if err != nil {
				return
			}
			events <- Event{Kind: "key", Data: Key{Rune: r}}
		}
	}()

	// Main event loop - single blocking loop
	for {
		ev := <-events

		// Handle view-local state updates
		switch ev.Kind {
		case "resize":
			r := ev.Data.(Resize)
			view.Width = r.Width
			view.Height = r.Height
		case "mouse":
			m := ev.Data.(Mouse)
			if m.ScrollY != 0 {
				view.ScrollY += m.ScrollY * 20
				if view.ScrollY < 0 {
					view.ScrollY = 0
				}
			}
		}

		// Run reducer
		model = app.Reduce(model, ev)

		// Redraw
		ctx.Clear()
		ctx.Translate(0, -view.ScrollY)
		app.Draw(model, ctx)
		ctx.Flush()
	}
}
