package frame

import (
	"unicode/utf8"
)

const boxSlop = 25

// addbox inserts n empty boxes after position bn, shifting existing
// boxes up. After the call, box[bn+n] == old box[bn].
func (f *Frame) addbox(bn, n int) {
	if bn > f.nbox {
		panic("frame: addbox: bn > nbox")
	}
	if f.nbox+n > f.nalloc {
		f.growbox(n + boxSlop)
	}
	for i := f.nbox - 1; i >= bn; i-- {
		f.box[i+n] = f.box[i]
	}
	f.nbox += n
}

// closebox removes boxes n0..n1 (inclusive) by shifting later boxes down.
// It does NOT free box contents; call freebox first if needed.
func (f *Frame) closebox(n0, n1 int) {
	if n0 >= f.nbox || n1 >= f.nbox || n1 < n0 {
		panic("frame: closebox")
	}
	n1++
	for i := n1; i < f.nbox; i++ {
		f.box[i-(n1-n0)] = f.box[i]
	}
	f.nbox -= n1 - n0
}

// delbox frees and removes boxes n0..n1 (inclusive).
func (f *Frame) delbox(n0, n1 int) {
	if n0 >= f.nbox || n1 >= f.nbox || n1 < n0 {
		panic("frame: delbox")
	}
	f.freebox(n0, n1)
	f.closebox(n0, n1)
}

// freebox frees the text contents of boxes n0..n1 (inclusive).
func (f *Frame) freebox(n0, n1 int) {
	if n1 < n0 {
		return
	}
	if n0 >= f.nbox || n1 >= f.nbox {
		panic("frame: freebox")
	}
	for i := n0; i <= n1; i++ {
		if f.box[i].nrune >= 0 {
			f.box[i].ptr = nil
		}
	}
}

// growbox grows the box array by delta slots.
func (f *Frame) growbox(delta int) {
	f.nalloc += delta
	newbox := make([]frbox, f.nalloc)
	copy(newbox, f.box[:f.nbox])
	f.box = newbox
}

// dupbox duplicates box at bn, inserting a copy at bn+1.
func (f *Frame) dupbox(bn int) {
	if f.box[bn].nrune < 0 {
		panic("frame: dupbox on break box")
	}
	f.addbox(bn, 1)
	if f.box[bn].nrune >= 0 {
		p := make([]byte, len(f.box[bn].ptr))
		copy(p, f.box[bn].ptr)
		f.box[bn+1].ptr = p
	}
}

// runeIndex returns the byte offset of the n-th rune in p.
func runeIndex(p []byte, n int) int {
	offset := 0
	for i := 0; i < n; i++ {
		_, size := utf8.DecodeRune(p[offset:])
		offset += size
	}
	return offset
}

// truncatebox drops the last n runes from a text box.
func (f *Frame) truncatebox(b *frbox, n int) {
	if b.nrune < 0 || b.nrune < n {
		panic("frame: truncatebox")
	}
	b.nrune -= n
	idx := runeIndex(b.ptr, b.nrune)
	b.ptr = b.ptr[:idx]
	b.wid = f.Font.StringWidth(string(b.ptr))
}

// chopbox drops the first n runes from a text box.
func (f *Frame) chopbox(b *frbox, n int) {
	if b.nrune < 0 || b.nrune < n {
		panic("frame: chopbox")
	}
	idx := runeIndex(b.ptr, n)
	newptr := make([]byte, len(b.ptr)-idx)
	copy(newptr, b.ptr[idx:])
	b.ptr = newptr
	b.nrune -= n
	b.wid = f.Font.StringWidth(string(b.ptr))
}

// splitbox splits box bn at rune position n: box[bn] keeps the
// first n runes, box[bn+1] gets the rest.
func (f *Frame) splitbox(bn, n int) {
	f.dupbox(bn)
	f.truncatebox(&f.box[bn], f.box[bn].nrune-n)
	f.chopbox(&f.box[bn+1], n)
}

// mergebox merges box bn and bn+1 into a single text box.
func (f *Frame) mergebox(bn int) {
	b := &f.box[bn]
	b1 := &f.box[bn+1]
	newptr := make([]byte, len(b.ptr)+len(b1.ptr))
	copy(newptr, b.ptr)
	copy(newptr[len(b.ptr):], b1.ptr)
	b.ptr = newptr
	b.wid += b1.wid
	b.nrune += b1.nrune
	f.delbox(bn+1, bn+1)
}

// findbox finds the box containing character position q, starting
// from box bn where character position p is known. If q falls in
// the middle of a box, the box is split so that q lands on a boundary.
// Returns the box index of the box starting at q.
func (f *Frame) findbox(bn int, p, q uint32) int {
	for bn < f.nbox {
		nr := uint32(f.box[bn].nRune())
		if p+nr > q {
			break
		}
		p += nr
		bn++
	}
	if p != q {
		f.splitbox(bn, int(q-p))
		bn++
	}
	return bn
}

// strlen returns the total rune count of boxes starting at index nb.
func (f *Frame) strlen(nb int) int {
	n := 0
	for ; nb < f.nbox; nb++ {
		n += f.box[nb].nRune()
	}
	return n
}
