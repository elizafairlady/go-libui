// Package window provides acme-style windows where body and tag are
// files backed by rune buffers, following the Plan 9 acme model.
//
// In real acme (see /sys/src/cmd/acme/dat.h), a Buffer is a
// disk-backed block cache. We use an in-memory rune slice for now,
// but the interface is designed so we can swap in disk backing later.
package window

// Buffer is a text buffer that stores runes and supports insert,
// delete, and read operations. It models acme's Buffer type.
//
// In acme, Buffer has: nc (char count), cache, and disk-backed Block
// array. We simplify to in-memory but keep the same operation set.
type Buffer struct {
	r     []rune // the data
	seq   int    // modification sequence number
	dirty bool   // modified since last clean
}

// Nc returns the number of runes in the buffer.
func (b *Buffer) Nc() int {
	return len(b.r)
}

// Runes returns the underlying rune slice. The caller must not modify it.
// This is used by the frame renderer which needs direct rune access.
func (b *Buffer) Runes() []rune {
	return b.r
}

// Read reads n runes starting at position q into dst.
// Returns the number of runes actually read.
func (b *Buffer) Read(q int, dst []rune) int {
	if q < 0 || q >= len(b.r) {
		return 0
	}
	n := copy(dst, b.r[q:])
	return n
}

// ReadAll returns all runes in the buffer as a string.
func (b *Buffer) ReadAll() string {
	return string(b.r)
}

// ReadRange returns runes [q0, q1) as a string.
func (b *Buffer) ReadRange(q0, q1 int) string {
	if q0 < 0 {
		q0 = 0
	}
	if q1 > len(b.r) {
		q1 = len(b.r)
	}
	if q0 >= q1 {
		return ""
	}
	return string(b.r[q0:q1])
}

// Insert inserts runes at position q.
func (b *Buffer) Insert(q int, r []rune) {
	if q < 0 {
		q = 0
	}
	if q > len(b.r) {
		q = len(b.r)
	}
	// Make room
	b.r = append(b.r, make([]rune, len(r))...)
	copy(b.r[q+len(r):], b.r[q:])
	copy(b.r[q:], r)
	b.dirty = true
	b.seq++
}

// Delete deletes runes in range [q0, q1).
func (b *Buffer) Delete(q0, q1 int) {
	if q0 < 0 {
		q0 = 0
	}
	if q1 > len(b.r) {
		q1 = len(b.r)
	}
	if q0 >= q1 {
		return
	}
	copy(b.r[q0:], b.r[q1:])
	b.r = b.r[:len(b.r)-(q1-q0)]
	b.dirty = true
	b.seq++
}

// Reset clears the buffer.
func (b *Buffer) Reset() {
	b.r = b.r[:0]
	b.seq++
}

// SetAll replaces the entire buffer contents.
func (b *Buffer) SetAll(s string) {
	b.r = []rune(s)
	b.seq++
}

// Dirty returns whether the buffer has been modified since last Clean().
func (b *Buffer) Dirty() bool {
	return b.dirty
}

// Clean marks the buffer as unmodified.
func (b *Buffer) Clean() {
	b.dirty = false
}

// Seq returns the modification sequence number.
func (b *Buffer) Seq() int {
	return b.seq
}
