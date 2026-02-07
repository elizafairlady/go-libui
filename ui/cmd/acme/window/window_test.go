package window

import "testing"

func TestNewRow(t *testing.T) {
	r := NewRow()
	if len(r.Cols) != 0 {
		t.Fatalf("cols = %d, want 0", len(r.Cols))
	}
	if r.Tag.ReadAll() != "Newcol Kill Putall Dump Load Exit" {
		t.Fatalf("tag = %q", r.Tag.ReadAll())
	}
}

func TestNewColumnAndWindow(t *testing.T) {
	r := NewRow()
	col := r.NewColumn()
	if col == nil {
		t.Fatal("col is nil")
	}
	if len(r.Cols) != 1 {
		t.Fatalf("cols = %d, want 1", len(r.Cols))
	}
	w := r.NewWindow(col)
	if w == nil {
		t.Fatal("window is nil")
	}
	if w.ID != 1 {
		t.Fatalf("id = %d, want 1", w.ID)
	}
	if len(r.Windows) != 1 {
		t.Fatalf("windows = %d, want 1", len(r.Windows))
	}
}

func TestCloseWindow(t *testing.T) {
	r := NewRow()
	col := r.NewColumn()
	w := r.NewWindow(col)
	r.CloseWindow(w)
	if len(r.Windows) != 0 {
		t.Fatalf("windows = %d after close", len(r.Windows))
	}
	if len(col.Windows) != 0 {
		t.Fatalf("col windows = %d after close", len(col.Windows))
	}
}

func TestCloseColumn(t *testing.T) {
	r := NewRow()
	col := r.NewColumn()
	r.NewWindow(col)
	r.NewWindow(col)
	r.CloseColumn(col)
	if len(r.Cols) != 0 {
		t.Fatalf("cols = %d after close", len(r.Cols))
	}
	if len(r.Windows) != 0 {
		t.Fatalf("windows = %d after close", len(r.Windows))
	}
}

func TestCtlName(t *testing.T) {
	r := NewRow()
	col := r.NewColumn()
	w := r.NewWindow(col)
	if err := w.Ctl("name /tmp/test.txt\n"); err != nil {
		t.Fatal(err)
	}
	if w.Name != "/tmp/test.txt" {
		t.Fatalf("name = %q", w.Name)
	}
}

func TestCtlCleanDirty(t *testing.T) {
	r := NewRow()
	col := r.NewColumn()
	w := r.NewWindow(col)
	w.Body.SetAll("hello")
	w.Body.Insert(5, []rune("!")) // makes dirty
	if !w.Body.Dirty() {
		t.Fatal("should be dirty after insert")
	}
	if err := w.Ctl("clean\n"); err != nil {
		t.Fatal(err)
	}
	if w.Body.Dirty() {
		t.Fatal("should be clean after ctl clean")
	}
}

func TestParseAddr(t *testing.T) {
	r := NewRow()
	col := r.NewColumn()
	w := r.NewWindow(col)
	w.Body.SetAll("hello world")

	if err := w.ParseAddr("#3,#8"); err != nil {
		t.Fatal(err)
	}
	if w.Addr.Q0 != 3 || w.Addr.Q1 != 8 {
		t.Fatalf("addr = %d,%d, want 3,8", w.Addr.Q0, w.Addr.Q1)
	}
}

func TestSnarfCutPaste(t *testing.T) {
	r := NewRow()
	col := r.NewColumn()
	w := r.NewWindow(col)
	w.Body.SetAll("hello world")
	w.Sel = Range{6, 11} // "world"

	r.Snarf(w)
	if r.SnarfBuf.ReadAll() != "world" {
		t.Fatalf("snarf = %q", r.SnarfBuf.ReadAll())
	}

	r.Cut(w)
	if w.Body.ReadAll() != "hello " {
		t.Fatalf("after cut: %q", w.Body.ReadAll())
	}

	w.Sel = Range{6, 6}
	r.Paste(w)
	if w.Body.ReadAll() != "hello world" {
		t.Fatalf("after paste: %q", w.Body.ReadAll())
	}
}

func TestIndex(t *testing.T) {
	r := NewRow()
	col := r.NewColumn()
	w := r.NewWindow(col)
	w.Tag.SetAll("test.txt Del")
	w.Body.SetAll("contents")
	idx := w.Index()
	if idx == "" {
		t.Fatal("empty index")
	}
}

func TestLookID(t *testing.T) {
	r := NewRow()
	col := r.NewColumn()
	w := r.NewWindow(col)
	found := r.LookID(w.ID)
	if found != w {
		t.Fatal("LookID failed")
	}
	if r.LookID(999) != nil {
		t.Fatal("LookID should return nil for unknown ID")
	}
}
