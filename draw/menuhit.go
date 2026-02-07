package draw

// Menu layout constants from menuhit.c.
const (
	MenuMargin      = 4  // outside to text
	MenuBorder      = 2  // outside to selection boxes
	MenuBlackborder = 2  // width of outlining border
	MenuVspacing    = 2  // extra spacing between lines of text
	MenuMaxunscroll = 25 // maximum #entries before scrolling turns on
	MenuNscroll     = 20 // number entries in scrolling part
	MenuScrollwid   = 14 // width of scroll bar
	MenuGap         = 4  // between text and scroll bar
)

// menurect returns the rectangle holding menu element i.
// textr is the rectangle holding all text elements.
func menurect(textr Rectangle, i int, fontheight int) Rectangle {
	if i < 0 {
		return ZR
	}
	r := textr
	r.Min.Y += (fontheight + MenuVspacing) * i
	r.Max.Y = r.Min.Y + fontheight + MenuVspacing
	return r.Inset(MenuBorder - MenuMargin)
}

// menusel returns the element number containing point p, or -1.
func menusel(textr Rectangle, p Point, fontheight int) int {
	if !p.In(textr) {
		return -1
	}
	return (p.Y - textr.Min.Y) / (fontheight + MenuVspacing)
}

// Menuhit displays a popup menu and tracks the mouse until the button
// is released. Returns the selected item index, or -1 if nothing selected.
// This is a port of 9front's menuhit().
//
// but is the button number (1=left, 2=middle, 3=right).
// mc is the mouse controller.
// menu is the menu to display.
// scr is an optional Screen for allocating a window (may be nil).
func (mc *Mousectl) Menuhit(but int, scr *Image, menu *Menu) int {
	if menu == nil || mc == nil {
		return -1
	}

	d := mc.Display
	if d == nil {
		return -1
	}

	screen := scr
	if screen == nil {
		screen = d.ScreenImage
	}
	if screen == nil {
		return -1
	}

	f := d.DefaultFont
	if f == nil {
		return -1
	}

	// Count items and find max width
	var items []string
	maxwid := 0
	for nitem := 0; ; nitem++ {
		var item string
		if menu.Item != nil {
			if nitem >= len(menu.Item) {
				break
			}
			item = menu.Item[nitem]
		} else if menu.Gen != nil {
			item = menu.Gen(nitem)
			if item == "" {
				break
			}
		} else {
			break
		}
		items = append(items, item)
		w := f.StringWidth(item)
		if w > maxwid {
			maxwid = w
		}
	}
	nitem := len(items)
	if nitem == 0 {
		return -1
	}

	if menu.Lasthit < 0 || menu.Lasthit >= nitem {
		menu.Lasthit = 0
	}

	// Determine scrolling parameters
	screenitem := (screen.R.Dy() - 10) / (f.Height + MenuVspacing)
	var scrolling bool
	var nitemdrawn, wid, off, lasti int

	if nitem > MenuMaxunscroll || nitem > screenitem {
		scrolling = true
		nitemdrawn = MenuNscroll
		if nitemdrawn > screenitem {
			nitemdrawn = screenitem
		}
		wid = maxwid + MenuGap + MenuScrollwid
		off = menu.Lasthit - nitemdrawn/2
		if off < 0 {
			off = 0
		}
		if off > nitem-nitemdrawn {
			off = nitem - nitemdrawn
		}
		lasti = menu.Lasthit - off
	} else {
		scrolling = false
		nitemdrawn = nitem
		wid = maxwid
		off = 0
		lasti = menu.Lasthit
	}

	// Calculate menu rectangle
	r := Rect(0, 0, wid, nitemdrawn*(f.Height+MenuVspacing)).Inset(-MenuMargin)
	r = r.Sub(Pt(wid/2, lasti*(f.Height+MenuVspacing)+f.Height/2))
	r = r.Add(mc.Point)

	// Keep on screen
	var pt Point
	if r.Max.X > screen.R.Max.X {
		pt.X = screen.R.Max.X - r.Max.X
	}
	if r.Max.Y > screen.R.Max.Y {
		pt.Y = screen.R.Max.Y - r.Max.Y
	}
	if r.Min.X < screen.R.Min.X {
		pt.X = screen.R.Min.X - r.Min.X
	}
	if r.Min.Y < screen.R.Min.Y {
		pt.Y = screen.R.Min.Y - r.Min.Y
	}
	menur := r.Add(pt)

	// Compute text rectangle
	var textr Rectangle
	textr.Max.X = menur.Max.X - MenuMargin
	textr.Min.X = textr.Max.X - maxwid
	textr.Min.Y = menur.Min.Y + MenuMargin
	textr.Max.Y = textr.Min.Y + nitemdrawn*(f.Height+MenuVspacing)

	// Draw menu background
	screen.Draw(menur, d.White, ZP)
	screen.Border(menur, MenuBlackborder, d.Black, ZP)

	// Draw items
	for i := 0; i < nitemdrawn; i++ {
		itemr := menurect(textr, i, f.Height)
		item := items[i+off]
		ptx := (textr.Min.X + textr.Max.X - f.StringWidth(item)) / 2
		pty := textr.Min.Y + i*(f.Height+MenuVspacing)
		screen.String(Pt(ptx, pty), d.Black, ZP, f, item)
		_ = itemr
	}

	// Highlight last item
	if lasti >= 0 && lasti < nitemdrawn {
		itemr := menurect(textr, lasti, f.Height)
		screen.Draw(itemr, d.Black, ZP)
		item := items[lasti+off]
		ptx := (textr.Min.X + textr.Max.X - f.StringWidth(item)) / 2
		pty := textr.Min.Y + lasti*(f.Height+MenuVspacing)
		screen.String(Pt(ptx, pty), d.White, ZP, f, item)
	}

	d.Flush()

	// Track mouse
	sel := lasti
	for {
		m := mc.Read()
		if m.Buttons&(1<<uint(but-1)) == 0 {
			// Button released
			break
		}

		i := menusel(textr, m.Point, f.Height)
		if i != sel {
			// Unhighlight old
			if sel >= 0 && sel < nitemdrawn {
				itemr := menurect(textr, sel, f.Height)
				screen.Draw(itemr, d.White, ZP)
				item := items[sel+off]
				ptx := (textr.Min.X + textr.Max.X - f.StringWidth(item)) / 2
				pty := textr.Min.Y + sel*(f.Height+MenuVspacing)
				screen.String(Pt(ptx, pty), d.Black, ZP, f, item)
			}
			// Highlight new
			if i >= 0 && i < nitemdrawn {
				itemr := menurect(textr, i, f.Height)
				screen.Draw(itemr, d.Black, ZP)
				item := items[i+off]
				ptx := (textr.Min.X + textr.Max.X - f.StringWidth(item)) / 2
				pty := textr.Min.Y + i*(f.Height+MenuVspacing)
				screen.String(Pt(ptx, pty), d.White, ZP, f, item)
			}
			sel = i
			d.Flush()
		}

		// Handle scrolling
		if scrolling && sel < 0 {
			// Scroll position
			_ = off // TODO: implement scroll tracking
		}
	}

	if sel >= 0 && sel < nitemdrawn {
		menu.Lasthit = sel + off
		return menu.Lasthit
	}
	return -1
}
