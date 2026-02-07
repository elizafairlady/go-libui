package draw

import "testing"

// TestMenuConstants verifies menu layout constants match menuhit.c.
func TestMenuConstants(t *testing.T) {
	if MenuMargin != 4 {
		t.Errorf("MenuMargin = %d, want 4", MenuMargin)
	}
	if MenuBorder != 2 {
		t.Errorf("MenuBorder = %d, want 2", MenuBorder)
	}
	if MenuBlackborder != 2 {
		t.Errorf("MenuBlackborder = %d, want 2", MenuBlackborder)
	}
	if MenuVspacing != 2 {
		t.Errorf("MenuVspacing = %d, want 2", MenuVspacing)
	}
	if MenuMaxunscroll != 25 {
		t.Errorf("MenuMaxunscroll = %d, want 25", MenuMaxunscroll)
	}
	if MenuNscroll != 20 {
		t.Errorf("MenuNscroll = %d, want 20", MenuNscroll)
	}
	if MenuScrollwid != 14 {
		t.Errorf("MenuScrollwid = %d, want 14", MenuScrollwid)
	}
	if MenuGap != 4 {
		t.Errorf("MenuGap = %d, want 4", MenuGap)
	}
}

// TestMenurect tests the menu item rectangle calculation.
func TestMenurect(t *testing.T) {
	fontheight := 16
	textr := Rect(100, 200, 300, 200+3*(fontheight+MenuVspacing))

	// Negative index returns zero rect
	r := menurect(textr, -1, fontheight)
	if !r.Eq(ZR) {
		t.Errorf("menurect(-1) = %v, want ZR", r)
	}

	// First item
	r = menurect(textr, 0, fontheight)
	if r.Min.Y != 200+(MenuBorder-MenuMargin) {
		t.Errorf("menurect(0).Min.Y = %d", r.Min.Y)
	}
	// Height should be fontheight+Vspacing - 2*(Border-Margin) inset
	expectedH := fontheight + MenuVspacing - 2*(MenuBorder-MenuMargin)
	if r.Dy() != expectedH {
		t.Errorf("menurect(0).Dy() = %d, want %d", r.Dy(), expectedH)
	}

	// Second item should be displaced by (fontheight+Vspacing)
	r1 := menurect(textr, 1, fontheight)
	if r1.Min.Y != r.Min.Y+fontheight+MenuVspacing {
		t.Errorf("menurect(1).Min.Y = %d, want %d", r1.Min.Y, r.Min.Y+fontheight+MenuVspacing)
	}
}

// TestMenusel tests hit detection in menu items.
func TestMenusel(t *testing.T) {
	fontheight := 16
	textr := Rect(100, 200, 300, 200+3*(fontheight+MenuVspacing))

	// Point outside rectangle
	if got := menusel(textr, Pt(50, 210), fontheight); got != -1 {
		t.Errorf("menusel outside = %d, want -1", got)
	}
	if got := menusel(textr, Pt(150, 100), fontheight); got != -1 {
		t.Errorf("menusel above = %d, want -1", got)
	}

	// First item
	if got := menusel(textr, Pt(150, 200), fontheight); got != 0 {
		t.Errorf("menusel first = %d, want 0", got)
	}

	// Second item
	y := 200 + fontheight + MenuVspacing
	if got := menusel(textr, Pt(150, y), fontheight); got != 1 {
		t.Errorf("menusel second = %d, want 1", got)
	}

	// Third item
	y = 200 + 2*(fontheight+MenuVspacing)
	if got := menusel(textr, Pt(150, y), fontheight); got != 2 {
		t.Errorf("menusel third = %d, want 2", got)
	}
}

// TestMenuStruct tests the Menu struct.
func TestMenuStruct(t *testing.T) {
	m := &Menu{
		Item:    []string{"Open", "Save", "Quit"},
		Lasthit: 0,
	}
	if len(m.Item) != 3 {
		t.Errorf("len(Item) = %d, want 3", len(m.Item))
	}
	if m.Lasthit != 0 {
		t.Errorf("Lasthit = %d, want 0", m.Lasthit)
	}

	// Test Gen function
	gen := func(i int) string {
		items := []string{"Alpha", "Beta", "Gamma"}
		if i >= len(items) {
			return ""
		}
		return items[i]
	}
	mg := &Menu{Gen: gen}
	if mg.Gen(0) != "Alpha" {
		t.Errorf("Gen(0) = %q, want Alpha", mg.Gen(0))
	}
	if mg.Gen(3) != "" {
		t.Errorf("Gen(3) = %q, want empty", mg.Gen(3))
	}
}
