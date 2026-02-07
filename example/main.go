// Comprehensive example demonstrating the go-libui draw library.
// Run this on Plan 9 / 9front or via drawterm.
//
// This example is based on the patterns from:
//
//	https://wiki.9front.org/programming-gui
//	https://wiki.9front.org/libdraw-tips
//
// It exercises: Init, color allocation, Draw, Line, Ellipse, Arc,
// Poly, Bezier, Border, String, Menu, mouse/keyboard input, and resize.
package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/elizafairlady/go-libui/draw"
)

// Globals matching 9front convention.
var (
	display *draw.Display
	screen  *draw.Image
	font    *draw.Font

	// Colors: 1×1 replicated images (the 9front idiom).
	red, green, blue, yellow, cyan, magenta *draw.Image
	darkgreen, paleblue, greyblue           *draw.Image
	back                                    *draw.Image

	mc *draw.Mousectl
	kc *draw.Keyboardctl

	drawCount int // counts draw-dot interactions
)

// alloccolor allocates a 1×1 replicated image filled with the given color.
// This is the standard Plan 9 way to make a "color brush".
//
// From https://wiki.9front.org/libdraw-tips:
//
//	"Colors in libdraw are represented as 1×1 replicated images."
func alloccolor(d *draw.Display, col uint32) *draw.Image {
	i, err := d.AllocImage(draw.Rect(0, 0, 1, 1), draw.RGB24, true, col)
	if err != nil {
		log.Fatalf("alloccolor: %v", err)
	}
	return i
}

// redraw repaints the entire window. Call after resize or clear.
func redraw() {
	r := screen.R

	// 1. Clear background to pale yellow (like acme)
	screen.Draw(r, back, draw.ZP)

	// 2. Window border
	screen.Border(r, 3, display.Black, draw.ZP)

	// Inner area
	inner := r.Inset(10)

	// Title
	title := "go-libui: Plan 9 Draw Library Demo"
	if font != nil {
		tw := font.StringWidth(title)
		tp := draw.Pt(inner.Min.X+(inner.Dx()-tw)/2, inner.Min.Y+5)
		screen.String(tp, display.Black, draw.ZP, font, title)
	}

	// Draw several demo sections
	y := inner.Min.Y + 30
	x := inner.Min.X + 10

	// --- Section: Rectangles ---
	drawRectangles(x, y, inner.Dx()/2-20)
	y += 90

	// --- Section: Lines ---
	drawLines(x, y, inner.Dx()-20)
	y += 80

	// --- Section: Ellipses & Arcs ---
	drawEllipses(x, y, inner.Dx()-20)
	y += 100

	// --- Section: Polygons & Beziers ---
	drawPolygons(x, y, inner.Dx()-20)
	y += 100

	// --- Section: Text rendering ---
	drawText(x, y)
	y += 60

	// --- Section: Color palette ---
	drawColorPalette(x, y, inner.Dx()-20)
	y += 50

	// Status bar
	drawStatus(inner, y)

	display.Flush()
}

// drawRectangles demonstrates filled and outlined rectangles.
func drawRectangles(x, y, w int) {
	if font != nil {
		screen.String(draw.Pt(x, y), display.Black, draw.ZP, font, "Rectangles:")
		y += font.Height + 2
	}

	// Filled rectangles
	rw := 60
	rh := 50
	screen.Draw(draw.Rect(x, y, x+rw, y+rh), red, draw.ZP)
	screen.Draw(draw.Rect(x+rw+10, y, x+2*rw+10, y+rh), green, draw.ZP)
	screen.Draw(draw.Rect(x+2*(rw+10), y, x+3*rw+20, y+rh), blue, draw.ZP)

	// Outlined rectangle (draw border)
	screen.Border(draw.Rect(x+3*(rw+10), y, x+4*rw+30, y+rh), 2, display.Black, draw.ZP)

	// Overlapping translucent rectangles
	screen.DrawOp(draw.Rect(x+4*(rw+10), y+5, x+4*(rw+10)+40, y+rh-5),
		cyan, nil, draw.ZP, draw.SoverD)
	screen.DrawOp(draw.Rect(x+4*(rw+10)+20, y+5, x+4*(rw+10)+60, y+rh-5),
		magenta, nil, draw.ZP, draw.SoverD)
}

// drawLines demonstrates various line types and styles.
func drawLines(x, y, w int) {
	if font != nil {
		screen.String(draw.Pt(x, y), display.Black, draw.ZP, font, "Lines:")
		y += font.Height + 2
	}

	// Basic lines with different thicknesses
	for i := 0; i < 5; i++ {
		lx := x + i*60
		screen.Line(draw.Pt(lx, y), draw.Pt(lx+50, y+40),
			draw.Endsquare, draw.Endsquare, i+1, display.Black, draw.ZP)
	}

	// Line with different end styles
	ly := y + 50
	// Square ends
	screen.Line(draw.Pt(x+320, y), draw.Pt(x+320, ly),
		draw.Endsquare, draw.Endsquare, 3, red, draw.ZP)
	// Disc ends
	screen.Line(draw.Pt(x+340, y), draw.Pt(x+340, ly),
		draw.Enddisc, draw.Enddisc, 3, green, draw.ZP)
	// Arrow ends
	screen.Line(draw.Pt(x+360, y), draw.Pt(x+420, y+25),
		draw.Endsquare, draw.Endarrow, 1, blue, draw.ZP)
}

// drawEllipses demonstrates circles, ellipses, arcs.
func drawEllipses(x, y, w int) {
	if font != nil {
		screen.String(draw.Pt(x, y), display.Black, draw.ZP, font, "Ellipses & Arcs:")
		y += font.Height + 2
	}

	cy := y + 40

	// Filled circle
	screen.FillEllipse(draw.Pt(x+40, cy), 30, 30, red, draw.ZP)

	// Outlined circle
	screen.Ellipse(draw.Pt(x+120, cy), 30, 30, 2, display.Black, draw.ZP)

	// Filled ellipse
	screen.FillEllipse(draw.Pt(x+210, cy), 50, 25, blue, draw.ZP)

	// Outlined ellipse
	screen.Ellipse(draw.Pt(x+320, cy), 50, 25, 2, darkgreen, draw.ZP)

	// Arc (quarter circle)
	screen.Arc(draw.Pt(x+420, cy), 30, 30, 2, display.Black, draw.ZP, 0, 90)

	// Filled arc (pie slice)
	screen.FillArc(draw.Pt(x+500, cy), 30, 30, green, draw.ZP, 45, 270)
}

// drawPolygons demonstrates polygon and bezier drawing.
func drawPolygons(x, y, w int) {
	if font != nil {
		screen.String(draw.Pt(x, y), display.Black, draw.ZP, font, "Polygons & Beziers:")
		y += font.Height + 2
	}

	// Triangle (filled polygon)
	tri := []draw.Point{
		draw.Pt(x+30, y+60),
		draw.Pt(x+60, y+10),
		draw.Pt(x+90, y+60),
	}
	screen.FillPoly(tri, 0, green, draw.ZP)
	// Poly doesn't auto-close; append first point to close outline
	triClosed := append(tri, tri[0])
	screen.Poly(triClosed, draw.Endsquare, draw.Endsquare, 1, display.Black, draw.ZP)

	// Pentagon (outlined polygon)
	pent := make([]draw.Point, 5)
	cx, cy := x+170, y+35
	for i := 0; i < 5; i++ {
		angle := float64(i)*2*math.Pi/5 - math.Pi/2
		pent[i] = draw.Pt(cx+int(25*math.Cos(angle)), cy+int(25*math.Sin(angle)))
	}
	pentClosed := append(pent, pent[0])
	screen.Poly(pentClosed, draw.Endsquare, draw.Endsquare, 2, blue, draw.ZP)

	// Star (filled)
	star := make([]draw.Point, 10)
	cx2, cy2 := x+270, y+35
	for i := 0; i < 10; i++ {
		angle := float64(i)*math.Pi/5 - math.Pi/2
		r := 25.0
		if i%2 == 1 {
			r = 12.0
		}
		star[i] = draw.Pt(cx2+int(r*math.Cos(angle)), cy2+int(r*math.Sin(angle)))
	}
	screen.FillPoly(star, 0, yellow, draw.ZP)
	starClosed := append(star, star[0])
	screen.Poly(starClosed, draw.Endsquare, draw.Endsquare, 1, display.Black, draw.ZP)

	// Bezier curve
	screen.Bezier(
		draw.Pt(x+330, y+60),
		draw.Pt(x+350, y),
		draw.Pt(x+400, y),
		draw.Pt(x+420, y+60),
		draw.Endsquare, draw.Endsquare, 2, red, draw.ZP)

	// Bezier spline through control points
	spline := []draw.Point{
		draw.Pt(x+440, y+60),
		draw.Pt(x+460, y+10),
		draw.Pt(x+490, y+50),
		draw.Pt(x+520, y+10),
		draw.Pt(x+540, y+60),
	}
	screen.BezSpline(spline, draw.Endsquare, draw.Endsquare, 2, darkgreen, draw.ZP)
}

// drawText demonstrates text rendering.
func drawText(x, y int) {
	if font == nil {
		return
	}
	screen.String(draw.Pt(x, y), display.Black, draw.ZP, font, "Text rendering:")
	y += font.Height + 2

	// Basic string
	screen.String(draw.Pt(x, y), display.Black, draw.ZP, font,
		"The quick brown fox jumps over the lazy dog.")
	y += font.Height

	// Colored text
	screen.String(draw.Pt(x, y), red, draw.ZP, font, "Red text  ")
	rx := x + font.StringWidth("Red text  ")
	screen.String(draw.Pt(rx, y), green, draw.ZP, font, "Green text  ")
	rx += font.StringWidth("Green text  ")
	screen.String(draw.Pt(rx, y), blue, draw.ZP, font, "Blue text")
	y += font.Height

	// Text with background
	screen.StringBg(draw.Pt(x, y), display.White, draw.ZP, font,
		"White on dark", darkgreen, draw.ZP)
}

// drawColorPalette shows the Plan 9 standard color palette.
func drawColorPalette(x, y, w int) {
	if font != nil {
		screen.String(draw.Pt(x, y), display.Black, draw.ZP, font, "Color palette (Cmap2rgb):")
		y += font.Height + 2
	}

	// Draw 16×16 grid of color map entries
	sz := 12
	for row := 0; row < 16; row++ {
		for col := 0; col < 16; col++ {
			idx := row*16 + col
			rgba := draw.Cmap2rgba(idx)
			c, err := display.AllocImage(draw.Rect(0, 0, 1, 1), draw.RGB24, true, uint32(rgba))
			if err != nil {
				continue
			}
			r := draw.Rect(x+col*(sz+1), y+row*(sz+1),
				x+col*(sz+1)+sz, y+row*(sz+1)+sz)
			screen.Draw(r, c, draw.ZP)
			c.Free()
		}
	}
}

// drawStatus draws the status bar at the bottom.
func drawStatus(inner draw.Rectangle, y int) {
	if font == nil {
		return
	}
	statusY := inner.Max.Y - font.Height - 5
	if y > statusY {
		statusY = y + 5
	}
	// Separator line
	screen.Line(draw.Pt(inner.Min.X+5, statusY-3),
		draw.Pt(inner.Max.X-5, statusY-3),
		draw.Endsquare, draw.Endsquare, 0, greyblue, draw.ZP)

	status := fmt.Sprintf("Dots: %d | Left-click: draw dot | Middle: menu | Right: color menu | 'q'/Esc: quit", drawCount)
	screen.String(draw.Pt(inner.Min.X+5, statusY), greyblue, draw.ZP, font, status)
}

// handleDot draws a dot at the mouse position (left click interaction).
func handleDot(p draw.Point, color *draw.Image) {
	screen.FillEllipse(p, 5, 5, color, draw.ZP)
	drawCount++
	// Refresh status
	if font != nil {
		inner := screen.R.Inset(10)
		statusY := inner.Max.Y - font.Height - 5
		// Clear status area
		screen.Draw(draw.Rect(inner.Min.X+5, statusY, inner.Max.X-5, inner.Max.Y-5), back, draw.ZP)
		drawStatus(inner, 0)
	}
	display.Flush()
}

func main() {
	var err error

	// Initialize display — the core init from programming-gui wiki.
	//
	// From https://wiki.9front.org/programming-gui:
	//   initdraw(nil, nil, "mywindow");
	display, err = draw.Init(nil, "", "go-libui demo")
	if err != nil {
		log.Fatal(err)
	}
	defer display.Close()

	screen = display.ScreenImage
	if screen == nil {
		screen = display.Image
	}
	font = display.DefaultFont

	// Allocate colors as 1×1 replicated images.
	// From https://wiki.9front.org/libdraw-tips:
	//   "To draw a colored rectangle, first make a 1×1 image of that color
	//    with the repl flag set, then use draw()."
	red = alloccolor(display, draw.DRed)
	green = alloccolor(display, draw.DGreen)
	blue = alloccolor(display, draw.DBlue)
	yellow = alloccolor(display, draw.DYellow)
	cyan = alloccolor(display, draw.DCyan)
	magenta = alloccolor(display, draw.DMagenta)
	darkgreen = alloccolor(display, draw.DDarkgreen)
	paleblue = alloccolor(display, draw.DPaleblue)
	greyblue = alloccolor(display, draw.DGreyblue)
	back = alloccolor(display, draw.DPaleyellow)

	defer func() {
		red.Free()
		green.Free()
		blue.Free()
		yellow.Free()
		cyan.Free()
		magenta.Free()
		darkgreen.Free()
		paleblue.Free()
		greyblue.Free()
		back.Free()
	}()

	// Initial draw
	redraw()

	// Initialize mouse — the input setup from programming-gui wiki.
	//
	// From https://wiki.9front.org/programming-gui:
	//   einit(Emouse|Ekeyboard);
	mc, err = draw.InitMouse("", screen)
	if err != nil {
		log.Fatal(err)
	}
	defer mc.Close()

	kc, err = draw.InitKeyboard("")
	if err != nil {
		log.Fatal(err)
	}
	defer kc.Close()

	// Current brush color for dot drawing
	dotColor := display.Black

	// Main menu
	mainMenu := &draw.Menu{
		Item: []string{"Clear", "Redraw", "---", "Quit"},
	}

	// Color menu
	colorMenu := &draw.Menu{
		Item:    []string{"Black", "Red", "Green", "Blue", "Cyan", "Magenta"},
		Lasthit: 0,
	}

	// Timer for animation demos (optional)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Event loop — the standard select-on-channels pattern.
	//
	// From https://wiki.9front.org/programming-gui:
	//   for(;;) switch(event(&e)) {
	//   case Emouse: ... break;
	//   case Ekeyboard: ... break;
	//   }
	for {
		select {
		case m := <-mc.C:
			if m.Buttons&1 != 0 {
				// Left click — draw a dot
				handleDot(m.Point, dotColor)
			}
			if m.Buttons&2 != 0 {
				// Middle click — main menu
				// The menuhit pattern from programming-gui wiki:
				//   n = emenuhit(2, &m, &menu);
				sel := mc.Menuhit(2, screen, mainMenu)
				switch sel {
				case 0: // Clear
					drawCount = 0
					redraw()
				case 1: // Redraw
					redraw()
				case 3: // Quit
					return
				}
			}
			if m.Buttons&4 != 0 {
				// Right click — color menu
				sel := mc.Menuhit(4, screen, colorMenu)
				switch sel {
				case 0:
					dotColor = display.Black
				case 1:
					dotColor = red
				case 2:
					dotColor = green
				case 3:
					dotColor = blue
				case 4:
					dotColor = cyan
				case 5:
					dotColor = magenta
				}
				colorMenu.Lasthit = sel
			}

		case r := <-kc.C:
			switch r {
			case 'q', draw.Kdel:
				// Quit
				return
			case 'c':
				// Clear
				drawCount = 0
				redraw()
			case 'r':
				// Redraw
				redraw()
			default:
				// Draw the typed character at center of window
				if font != nil {
					cx := screen.R.Min.X + screen.R.Dx()/2
					cy := screen.R.Min.Y + screen.R.Dy()/2
					s := string(r)
					w := font.StringWidth(s)
					screen.StringBg(
						draw.Pt(cx-w/2, cy-font.Height/2),
						display.Black, draw.ZP, font, s,
						back, draw.ZP)
					display.Flush()
				}
			}

		case <-mc.Resize:
			// Resize handling — the getwindow pattern from programming-gui wiki.
			//
			// From https://wiki.9front.org/programming-gui:
			//   if(getwindow(display, Refnone) < 0)
			//     sysfatal("resize failed: %r");
			if err := display.GetWindow(draw.Refnone); err != nil {
				log.Printf("resize: %v", err)
				continue
			}
			screen = display.ScreenImage
			if screen == nil {
				screen = display.Image
			}
			redraw()

		case <-ticker.C:
			// Periodic: could animate, update clock, etc.
			// For now just a no-op heartbeat.
		}
	}
}
