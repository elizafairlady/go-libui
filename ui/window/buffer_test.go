package window

import "testing"

func TestBufferInsertDelete(t *testing.T) {
	var b Buffer
	b.Insert(0, []rune("hello"))
	if got := b.ReadAll(); got != "hello" {
		t.Fatalf("got %q, want %q", got, "hello")
	}
	if b.Nc() != 5 {
		t.Fatalf("nc = %d, want 5", b.Nc())
	}

	// Insert in middle: "hello" → "hel cruello"
	b.Insert(3, []rune(" cruel"))
	if got := b.ReadAll(); got != "hel cruello" {
		t.Fatalf("got %q, want %q", got, "hel cruello")
	}

	// Delete [3,9) removes " cruel" → "hello"
	b.Delete(3, 9)
	if got := b.ReadAll(); got != "hello" {
		t.Fatalf("got %q, want %q", got, "hello")
	}
}

func TestBufferAppend(t *testing.T) {
	var b Buffer
	b.Insert(0, []rune("hello"))
	b.Insert(b.Nc(), []rune(" world"))
	if got := b.ReadAll(); got != "hello world" {
		t.Fatalf("got %q, want %q", got, "hello world")
	}
}

func TestBufferReadRange(t *testing.T) {
	var b Buffer
	b.SetAll("abcdefghij")
	if got := b.ReadRange(2, 5); got != "cde" {
		t.Fatalf("got %q, want %q", got, "cde")
	}
	// Clamp
	if got := b.ReadRange(-1, 100); got != "abcdefghij" {
		t.Fatalf("got %q, want %q", got, "abcdefghij")
	}
}

func TestBufferDirtyClean(t *testing.T) {
	var b Buffer
	if b.Dirty() {
		t.Fatal("new buffer should not be dirty")
	}
	b.Insert(0, []rune("x"))
	if !b.Dirty() {
		t.Fatal("should be dirty after insert")
	}
	b.Clean()
	if b.Dirty() {
		t.Fatal("should be clean after Clean()")
	}
	b.Delete(0, 1)
	if !b.Dirty() {
		t.Fatal("should be dirty after delete")
	}
}

func TestBufferSeq(t *testing.T) {
	var b Buffer
	s0 := b.Seq()
	b.Insert(0, []rune("a"))
	s1 := b.Seq()
	if s1 <= s0 {
		t.Fatal("seq should increase after insert")
	}
	b.Delete(0, 1)
	s2 := b.Seq()
	if s2 <= s1 {
		t.Fatal("seq should increase after delete")
	}
}

func TestBufferReset(t *testing.T) {
	var b Buffer
	b.SetAll("hello world")
	b.Reset()
	if b.Nc() != 0 {
		t.Fatalf("nc = %d after reset, want 0", b.Nc())
	}
}
