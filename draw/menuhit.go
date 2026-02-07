package draw

// Menuhit displays a menu and returns the selected item.
// Returns -1 if no item was selected.
func (mc *Mousectl) Menuhit(but int, scr *Image, menu *Menu) int {
	if menu == nil || mc == nil {
		return -1
	}

	d := mc.Display
	if d == nil {
		return -1
	}

	// Get menu items
	items := menu.Item
	if menu.Gen != nil && len(items) == 0 {
		// Generate items
		for i := 0; ; i++ {
			s := menu.Gen(i)
			if s == "" {
				break
			}
			items = append(items, s)
		}
	}

	if len(items) == 0 {
		return -1
	}

	// Calculate menu dimensions
	f := d.DefaultFont
	maxw := 0
	for _, item := range items {
		w := f.StringWidth(item)
		if w > maxw {
			maxw = w
		}
	}

	border := 2
	margin := 4
	itemHeight := f.Height + 2
	menuWidth := maxw + 2*margin + 2*border
	menuHeight := len(items)*itemHeight + 2*border

	// Position menu at mouse location
	m := mc.Mouse
	r := Rect(m.X, m.Y, m.X+menuWidth, m.Y+menuHeight)

	// Keep menu on screen
	if scr != nil {
		if r.Max.X > scr.R.Max.X {
			r = r.Sub(Pt(r.Max.X-scr.R.Max.X, 0))
		}
		if r.Max.Y > scr.R.Max.Y {
			r = r.Sub(Pt(0, r.Max.Y-scr.R.Max.Y))
		}
		if r.Min.X < scr.R.Min.X {
			r = r.Add(Pt(scr.R.Min.X-r.Min.X, 0))
		}
		if r.Min.Y < scr.R.Min.Y {
			r = r.Add(Pt(0, scr.R.Min.Y-r.Min.Y))
		}
	}

	// Save background (in a real implementation)
	// For now, we just draw and let caller redraw

	// Draw menu
	scr.Draw(r, d.White, ZP)
	scr.Border(r, border, d.Black, ZP)

	// Draw items
	sel := menu.Lasthit
	if sel < 0 || sel >= len(items) {
		sel = 0
	}

	for i, item := range items {
		itemr := Rect(r.Min.X+border, r.Min.Y+border+i*itemHeight,
			r.Max.X-border, r.Min.Y+border+(i+1)*itemHeight)
		if i == sel {
			scr.Draw(itemr, d.Black, ZP)
			scr.String(Pt(itemr.Min.X+margin, itemr.Min.Y+1),
				d.White, ZP, f, item)
		} else {
			scr.String(Pt(itemr.Min.X+margin, itemr.Min.Y+1),
				d.Black, ZP, f, item)
		}
	}
	d.Flush()

	// Track mouse
	for {
		m = mc.Read()
		if m.Buttons == 0 {
			// Button released
			break
		}

		// Check which item mouse is over
		newsel := -1
		if m.Point.In(r) {
			y := m.Y - r.Min.Y - border
			newsel = y / itemHeight
			if newsel < 0 || newsel >= len(items) {
				newsel = -1
			}
		}

		if newsel != sel {
			// Unhighlight old
			if sel >= 0 && sel < len(items) {
				itemr := Rect(r.Min.X+border, r.Min.Y+border+sel*itemHeight,
					r.Max.X-border, r.Min.Y+border+(sel+1)*itemHeight)
				scr.Draw(itemr, d.White, ZP)
				scr.String(Pt(itemr.Min.X+margin, itemr.Min.Y+1),
					d.Black, ZP, f, items[sel])
			}
			// Highlight new
			if newsel >= 0 && newsel < len(items) {
				itemr := Rect(r.Min.X+border, r.Min.Y+border+newsel*itemHeight,
					r.Max.X-border, r.Min.Y+border+(newsel+1)*itemHeight)
				scr.Draw(itemr, d.Black, ZP)
				scr.String(Pt(itemr.Min.X+margin, itemr.Min.Y+1),
					d.White, ZP, f, items[newsel])
			}
			sel = newsel
			d.Flush()
		}
	}

	if sel >= 0 && sel < len(items) {
		menu.Lasthit = sel
		return sel
	}
	return -1
}
