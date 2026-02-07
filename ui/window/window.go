package window

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Window models an acme window. Each window has:
//   - A tag (editable text bar with commands)
//   - A body (multi-line editable text area)
//   - An address register (q0, q1 into body)
//   - Control state (name, dirty, isdir, etc.)
//   - An event channel (for external programs that open /event)
//
// In real acme (dat.h), Window has: Text tag, Text body, where each
// Text embeds a Frame and points to a File (which embeds a Buffer).
// We flatten this: Window directly owns body and tag Buffers.
//
// Each Window is identified by an integer ID, chosen sequentially.
// The filesystem namespace is:
//
//	/<id>/addr    — address register (read/write)
//	/<id>/body    — body text (read: full text; write: append)
//	/<id>/ctl     — control messages
//	/<id>/data    — addressed read/write at addr
//	/<id>/event   — event stream
//	/<id>/rdsel   — read current selection
//	/<id>/wrsel   — write replaces selection
//	/<id>/tag     — tag text (read: full text; write: append)
//	/<id>/xdata   — bounded read at addr range
type Window struct {
	ID   int
	Tag  Buffer
	Body Buffer

	// Addr is the address register, set by writing to addr file,
	// used by data/xdata reads and writes.
	Addr Range

	// Sel is the user-visible selection in the body.
	Sel Range

	// Name is the file name associated with this window.
	Name string

	// IsDir indicates this window shows a directory listing.
	IsDir bool

	// IsScratch marks the window as a scratch buffer.
	IsScratch bool

	// EventOpen tracks how many readers have the event file open.
	// When > 0, user actions are sent as events instead of being
	// handled directly, matching acme's nopen[QWevent] mechanism.
	EventOpen int

	// Events is the pending event text (acme's w->events).
	Events string

	// Col is the column index this window belongs to (-1 if none).
	Col int

	// Owner is the last mouse button owner character (acme's w->owner).
	Owner byte
}

// Range is a text range [Q0, Q1), matching acme's Range struct.
type Range struct {
	Q0 int
	Q1 int
}

// Row manages all columns and windows, like acme's Row.
// It serves as the top-level container for the window filesystem.
type Row struct {
	mu       sync.Mutex
	Tag      Buffer          // row-level tag
	Cols     []*Column       // columns
	Windows  map[int]*Window // all windows by ID
	nextID   int             // next window ID
	SnarfBuf Buffer          // global snarf buffer (acme's snarfbuf)
}

// Column models an acme column. It has a tag and a list of windows.
type Column struct {
	ID      int
	Tag     Buffer
	Windows []*Window
}

// NewRow creates a new empty Row.
func NewRow() *Row {
	r := &Row{
		Windows: make(map[int]*Window),
	}
	r.Tag.SetAll("Newcol Kill Putall Dump Load Exit")
	return r
}

// NewColumn adds a new column to the row. Returns the column.
func (r *Row) NewColumn() *Column {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := &Column{
		ID: len(r.Cols),
	}
	c.Tag.SetAll("New Cut Paste Snarf Zerox Delcol")
	r.Cols = append(r.Cols, c)
	return c
}

// NewWindow creates a new window in the given column.
// Like acme's coladd(). Returns the window.
func (r *Row) NewWindow(col *Column) *Window {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	w := &Window{
		ID:  r.nextID,
		Col: col.ID,
	}
	r.Windows[w.ID] = w
	col.Windows = append(col.Windows, w)
	return w
}

// CloseWindow removes a window from its column and the row.
func (r *Row) CloseWindow(w *Window) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.Cols {
		for i, cw := range c.Windows {
			if cw.ID == w.ID {
				c.Windows = append(c.Windows[:i], c.Windows[i+1:]...)
				break
			}
		}
	}
	delete(r.Windows, w.ID)
}

// CloseColumn removes a column and all its windows.
func (r *Row) CloseColumn(col *Column) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Remove all windows in this column
	for _, w := range col.Windows {
		delete(r.Windows, w.ID)
	}
	// Remove the column
	for i, c := range r.Cols {
		if c == col {
			r.Cols = append(r.Cols[:i], r.Cols[i+1:]...)
			break
		}
	}
}

// LookID finds a window by ID.
func (r *Row) LookID(id int) *Window {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.Windows[id]
}

// Ctl handles control file writes for a window, matching the
// commands in acme's xfidctlwrite() from xfid.c.
//
// Supported commands:
//   - name <path>\n    — set file name
//   - clean            — mark clean
//   - dirty            — mark dirty
//   - del              — delete (check dirty)
//   - delete           — delete (force)
//   - get              — reload from disk
//   - put              — write to disk
//   - show             — show dot
//   - dot=addr         — set selection to addr
//   - addr=dot         — set addr to selection
//   - scratch          — mark as scratch
//   - mark             — mark for undo
//   - nomark           — disable auto-mark
func (w *Window) Ctl(msg string) error {
	for len(msg) > 0 {
		var cmd string
		if i := strings.IndexByte(msg, '\n'); i >= 0 {
			cmd = msg[:i]
			msg = msg[i+1:]
		} else {
			cmd = msg
			msg = ""
		}
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}

		switch {
		case cmd == "clean":
			w.Body.Clean()
		case cmd == "dirty":
			w.Body.dirty = true
		case cmd == "scratch":
			w.IsScratch = true
		case cmd == "show":
			// TODO: make dot visible
		case cmd == "dot=addr":
			w.Sel = w.Addr
		case cmd == "addr=dot":
			w.Addr = w.Sel
		case strings.HasPrefix(cmd, "name "):
			w.Name = strings.TrimSpace(cmd[5:])
		default:
			return fmt.Errorf("unknown ctl: %s", cmd)
		}
	}
	return nil
}

// Index returns the index line for this window, matching acme's format:
// five 11-char decimal fields (id, nchars_tag, nchars_body, isdir, dirty)
// followed by the tag up to first newline.
func (w *Window) Index() string {
	isdir := 0
	if w.IsDir {
		isdir = 1
	}
	dirty := 0
	if w.Body.Dirty() {
		dirty = 1
	}
	tag := w.Tag.ReadAll()
	if i := strings.IndexByte(tag, '\n'); i >= 0 {
		tag = tag[:i]
	}
	return fmt.Sprintf("%11d %11d %11d %11d %11d %s\n",
		w.ID, w.Tag.Nc(), w.Body.Nc(), isdir, dirty, tag)
}

// CtlPrint returns the ctl file contents for reading, matching
// acme's winctlprint() format.
func (w *Window) CtlPrint() string {
	isdir := 0
	if w.IsDir {
		isdir = 1
	}
	dirty := 0
	if w.Body.Dirty() {
		dirty = 1
	}
	return fmt.Sprintf("%11d %11d %11d %11d %11d ",
		w.ID, w.Tag.Nc(), w.Body.Nc(), isdir, dirty)
}

// WinEvent appends an event string, like acme's winevent().
// Events accumulate until read from the event file.
func (w *Window) WinEvent(format string, args ...any) {
	w.Events += fmt.Sprintf(format, args...)
}

// ParseAddr parses an address string and sets w.Addr.
// For now, supports simple forms:
//   - #n       — character position n
//   - #n,#m    — range [n, m)
//   - empty    — whole file (0, nc)
func (w *Window) ParseAddr(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		w.Addr = Range{0, w.Body.Nc()}
		return nil
	}
	if s[0] == '#' {
		parts := strings.SplitN(s, ",", 2)
		q0, err := parseCharAddr(parts[0])
		if err != nil {
			return err
		}
		q1 := q0
		if len(parts) == 2 {
			q1, err = parseCharAddr(parts[1])
			if err != nil {
				return err
			}
		}
		w.Addr = Range{q0, q1}
		return nil
	}
	return fmt.Errorf("unsupported address: %s", s)
}

// Snarf copies the selection from w.Body into the global snarf buffer.
// Like acme's cut() with dosnarf=TRUE, docut=FALSE (the "Snarf" command).
func (r *Row) Snarf(w *Window) {
	if w.Sel.Q0 >= w.Sel.Q1 {
		return
	}
	text := w.Body.ReadRange(w.Sel.Q0, w.Sel.Q1)
	r.SnarfBuf.Reset()
	r.SnarfBuf.Insert(0, []rune(text))
}

// Cut copies the selection into snarf and deletes it from the body.
// Like acme's cut() with dosnarf=TRUE, docut=TRUE (the "Cut" command).
func (r *Row) Cut(w *Window) {
	if w.Sel.Q0 >= w.Sel.Q1 {
		return
	}
	r.Snarf(w)
	w.Body.Delete(w.Sel.Q0, w.Sel.Q1)
	w.Sel.Q1 = w.Sel.Q0
}

// Paste inserts the snarf buffer at the selection, replacing it.
// Like acme's paste() from exec.c.
func (r *Row) Paste(w *Window) {
	if r.SnarfBuf.Nc() == 0 {
		return
	}
	// Delete current selection first
	if w.Sel.Q0 < w.Sel.Q1 {
		w.Body.Delete(w.Sel.Q0, w.Sel.Q1)
		w.Sel.Q1 = w.Sel.Q0
	}
	text := []rune(r.SnarfBuf.ReadAll())
	w.Body.Insert(w.Sel.Q0, text)
	w.Sel.Q1 = w.Sel.Q0 + len(text)
}

func parseCharAddr(s string) (int, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") {
		n, err := strconv.Atoi(s[1:])
		if err != nil {
			return 0, fmt.Errorf("bad address %s: %w", s, err)
		}
		return n, nil
	}
	return 0, fmt.Errorf("unsupported address form: %s", s)
}
