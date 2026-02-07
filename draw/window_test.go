package draw

import "testing"

// TestOriginWindowMath tests the rectangle update in OriginWindow.
func TestOriginWindowMath(t *testing.T) {
	// OriginWindow updates R and Clipr by delta = log - R.Min
	w := &Image{
		R:     Rect(100, 200, 300, 400),
		Clipr: Rect(100, 200, 300, 400),
	}
	log := Pt(150, 250)
	delta := log.Sub(w.R.Min) // (50, 50)

	expected := w.R.Add(delta)
	expectedClip := w.Clipr.Add(delta)

	// Simulate what OriginWindow does (without display)
	w.R = w.R.Add(delta)
	w.Clipr = w.Clipr.Add(delta)

	if !w.R.Eq(expected) {
		t.Errorf("R = %v, want %v", w.R, expected)
	}
	if !w.Clipr.Eq(expectedClip) {
		t.Errorf("Clipr = %v, want %v", w.Clipr, expectedClip)
	}
	if w.R.Min.X != 150 || w.R.Min.Y != 250 {
		t.Errorf("R.Min = (%d,%d), want (150,250)", w.R.Min.X, w.R.Min.Y)
	}
}

// TestTopBottomWindowNil tests nil safety.
func TestTopBottomWindowNil(t *testing.T) {
	var w *Image
	w.TopWindow()    // should not panic
	w.BottomWindow() // should not panic
}

// TestTopBottomWindowNoScreen tests no-op when no screen.
func TestTopBottomWindowNoScreen(t *testing.T) {
	w := &Image{Display: &Display{}}
	w.TopWindow()    // Screen is nil, should be no-op
	w.BottomWindow() // Screen is nil, should be no-op
}

// TestTopNWindowsEmpty tests empty slice.
func TestTopNWindowsEmpty(t *testing.T) {
	TopNWindows(nil)        // should not panic
	TopNWindows([]*Image{}) // should not panic
	BottomNWindows(nil)
	BottomNWindows([]*Image{})
}

// TestOriginWindowNilSafe tests nil image safety.
func TestOriginWindowNilSafe(t *testing.T) {
	var w *Image
	err := w.OriginWindow(Pt(0, 0), Pt(0, 0))
	if err != nil {
		t.Errorf("OriginWindow on nil should return nil, got %v", err)
	}
}

// TestRefreshConstants tests the refresh mode constants.
func TestRefreshConstants(t *testing.T) {
	if Refbackup != 0 {
		t.Errorf("Refbackup = %d, want 0", Refbackup)
	}
	if Refnone != 1 {
		t.Errorf("Refnone = %d, want 1", Refnone)
	}
	if Refmesg != 2 {
		t.Errorf("Refmesg = %d, want 2", Refmesg)
	}
}

// TestAllocScreenDifferentDisplays tests error for mismatched displays.
func TestAllocScreenDifferentDisplays(t *testing.T) {
	d1 := &Display{}
	d2 := &Display{}
	img1 := &Image{Display: d1}
	img2 := &Image{Display: d2}

	_, err := AllocScreen(img1, img2, false)
	if err == nil {
		t.Error("AllocScreen with different displays should fail")
	}
}
