package window

import "testing"

func TestRowNewColumnWindow(t *testing.T) {
	r := NewRow()
	c := r.NewColumn()
	if len(r.Cols) != 1 {
		t.Fatalf("ncol = %d, want 1", len(r.Cols))
	}
	w := r.NewWindow(c)
	if w.ID != 1 {
		t.Fatalf("id = %d, want 1", w.ID)
	}
	if len(c.Windows) != 1 {
		t.Fatalf("nw = %d, want 1", len(c.Windows))
	}
	if r.LookID(1) != w {
		t.Fatal("LookID failed")
	}

	w2 := r.NewWindow(c)
	if w2.ID != 2 {
		t.Fatalf("id = %d, want 2", w2.ID)
	}

	r.CloseWindow(w)
	if len(c.Windows) != 1 {
		t.Fatalf("nw = %d after close, want 1", len(c.Windows))
	}
	if r.LookID(1) != nil {
		t.Fatal("window 1 should be gone")
	}
}

func TestRowCloseColumn(t *testing.T) {
	r := NewRow()
	c1 := r.NewColumn()
	c2 := r.NewColumn()
	w1 := r.NewWindow(c1)
	r.NewWindow(c2)
	r.CloseColumn(c1)
	if len(r.Cols) != 1 {
		t.Fatalf("ncol = %d, want 1", len(r.Cols))
	}
	if r.LookID(w1.ID) != nil {
		t.Fatal("window from closed column should be gone")
	}
}

func TestWindowCtl(t *testing.T) {
	w := &Window{ID: 1}
	w.Body.SetAll("hello world")
	w.Body.dirty = true

	if err := w.Ctl("clean"); err != nil {
		t.Fatal(err)
	}
	if w.Body.Dirty() {
		t.Fatal("should be clean after ctl clean")
	}

	if err := w.Ctl("dirty"); err != nil {
		t.Fatal(err)
	}
	if !w.Body.Dirty() {
		t.Fatal("should be dirty after ctl dirty")
	}

	if err := w.Ctl("name /tmp/foo.txt"); err != nil {
		t.Fatal(err)
	}
	if w.Name != "/tmp/foo.txt" {
		t.Fatalf("name = %q, want /tmp/foo.txt", w.Name)
	}

	if err := w.Ctl("scratch"); err != nil {
		t.Fatal(err)
	}
	if !w.IsScratch {
		t.Fatal("should be scratch")
	}

	w.Addr = Range{0, 5}
	if err := w.Ctl("dot=addr"); err != nil {
		t.Fatal(err)
	}
	if w.Sel.Q0 != 0 || w.Sel.Q1 != 5 {
		t.Fatalf("sel = %v, want {0, 5}", w.Sel)
	}
}

func TestWindowParseAddr(t *testing.T) {
	w := &Window{ID: 1}
	w.Body.SetAll("hello world")

	if err := w.ParseAddr("#3,#8"); err != nil {
		t.Fatal(err)
	}
	if w.Addr.Q0 != 3 || w.Addr.Q1 != 8 {
		t.Fatalf("addr = %v, want {3, 8}", w.Addr)
	}

	// Read addressed range
	got := w.Body.ReadRange(w.Addr.Q0, w.Addr.Q1)
	if got != "lo wo" {
		t.Fatalf("got %q, want %q", got, "lo wo")
	}
}

func TestWindowIndex(t *testing.T) {
	w := &Window{ID: 42}
	w.Tag.SetAll("scratch Del Get Put")
	w.Body.SetAll("some text")
	idx := w.Index()
	if len(idx) == 0 {
		t.Fatal("empty index")
	}
	// Should contain window ID and tag text
	if idx[0:11] != "         42" {
		t.Fatalf("index doesn't start with id 42: %q", idx[:20])
	}
}

func TestSnarfCutPaste(t *testing.T) {
	r := NewRow()
	c := r.NewColumn()
	w := r.NewWindow(c)
	w.Body.SetAll("hello world")
	w.Sel = Range{6, 11} // "world"

	// Snarf (copy)
	r.Snarf(w)
	if got := r.SnarfBuf.ReadAll(); got != "world" {
		t.Fatalf("snarf = %q, want %q", got, "world")
	}
	// Body unchanged
	if got := w.Body.ReadAll(); got != "hello world" {
		t.Fatalf("body = %q, want %q", got, "hello world")
	}

	// Cut
	w.Sel = Range{5, 11} // " world"
	r.Cut(w)
	if got := r.SnarfBuf.ReadAll(); got != " world" {
		t.Fatalf("snarf = %q, want %q", got, " world")
	}
	if got := w.Body.ReadAll(); got != "hello" {
		t.Fatalf("body = %q, want %q", got, "hello")
	}

	// Paste
	w.Sel = Range{5, 5} // cursor at end
	r.Paste(w)
	if got := w.Body.ReadAll(); got != "hello world" {
		t.Fatalf("body = %q, want %q", got, "hello world")
	}
}
