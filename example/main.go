// Example program demonstrating the draw library.
// Run this on Plan 9 / 9front or via drawterm.
package main

import (
	"log"

	"github.com/elizafairlady/go-libui/draw"
)

func main() {
	// Initialize display
	d, err := draw.Init(nil, "", "draw example")
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	// Get screen
	scr := d.Image

	// Clear to white
	scr.Draw(scr.R, d.White, draw.ZP)

	// Draw some shapes
	// Red rectangle
	red, _ := d.AllocImage(draw.Rect(0, 0, 1, 1), draw.RGB24, true, draw.DRed)
	scr.Draw(draw.Rect(50, 50, 150, 100), red, draw.ZP)

	// Blue ellipse
	blue, _ := d.AllocImage(draw.Rect(0, 0, 1, 1), draw.RGB24, true, draw.DBlue)
	scr.FillEllipse(draw.Pt(300, 150), 50, 30, blue, draw.ZP)

	// Green line
	green, _ := d.AllocImage(draw.Rect(0, 0, 1, 1), draw.RGB24, true, draw.DGreen)
	scr.Line(draw.Pt(50, 200), draw.Pt(350, 200), draw.Endsquare, draw.Endsquare, 2, green, draw.ZP)

	// Draw text
	if d.DefaultFont != nil {
		scr.String(draw.Pt(50, 250), d.Black, draw.ZP, d.DefaultFont, "Hello, Plan 9!")
	}

	// Border
	scr.Border(draw.Rect(20, 20, 380, 300), 3, d.Black, draw.ZP)

	// Flush to display
	d.Flush()

	// Initialize mouse and keyboard
	mc, err := d.InitMouse()
	if err != nil {
		log.Fatal(err)
	}
	defer mc.Close()

	kc, err := d.InitKeyboard()
	if err != nil {
		log.Fatal(err)
	}
	defer kc.Close()

	// Event loop
	for {
		select {
		case m := <-mc.C:
			// Mouse event
			if m.Buttons&1 != 0 {
				// Left click - draw a point
				scr.FillEllipse(m.Point, 5, 5, d.Black, draw.ZP)
				d.Flush()
			}
			if m.Buttons&4 != 0 {
				// Right click - show menu
				menu := &draw.Menu{
					Item: []string{"Clear", "Quit"},
				}
				sel := mc.Menuhit(4, scr, menu)
				if sel == 0 {
					scr.Draw(scr.R, d.White, draw.ZP)
					d.Flush()
				} else if sel == 1 {
					return
				}
			}
		case r := <-kc.C:
			// Keyboard event
			if r == 'q' || r == draw.KeyEscape {
				return
			}
		case <-mc.Resize:
			// Resize event
			d.GetWindow(draw.Refnone)
			scr = d.Image
			scr.Draw(scr.R, d.White, draw.ZP)
			d.Flush()
		}
	}
}
