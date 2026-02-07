package draw

import (
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
)

// InitMouse opens the mouse device and returns a Mousectl.
// If file is empty, it defaults to /dev/mouse.
// The image i is the associated display image (used for flushing).
func InitMouse(file string, i *Image) (*Mousectl, error) {
	if file == "" {
		file = "/dev/mouse"
	}

	mfd, err := os.OpenFile(file, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("initmouse: %v", err)
	}

	// Derive cursor file path from mouse file path
	cursorfile := path.Join(path.Dir(file), "cursor")
	cfd, err := os.OpenFile(cursorfile, os.O_RDWR, 0)
	if err != nil {
		// non-fatal: cursor control may not be available
		cfd = nil
	}

	var d *Display
	if i != nil {
		d = i.Display
	}

	mc := &Mousectl{
		C:       make(chan Mouse, 0),
		Resize:  make(chan bool, 2),
		Display: d,
		file:    mfd,
		cfd:     cfd,
		image:   i,
	}

	go mc.readproc()
	return mc, nil
}

// readproc reads mouse events in a goroutine.
// The mouse message format is: type[1] x[12] y[12] buttons[12] msec[12]
// where type is 'm' for mouse or 'r' for resize.
func (mc *Mousectl) readproc() {
	buf := make([]byte, 1+5*12)
	nerr := 0
	for {
		n, err := mc.file.Read(buf)
		if n != 1+4*12 {
			if err != nil || mc.file == nil {
				break
			}
			nerr++
			if nerr > 10 {
				break
			}
			continue
		}
		nerr = 0

		var m Mouse
		switch buf[0] {
		case 'r':
			// Resize event - also contains mouse data
			select {
			case mc.Resize <- true:
			default:
			}
			fallthrough
		case 'm':
			m.X = atoiField(buf[1 : 1+12])
			m.Y = atoiField(buf[1+12 : 1+2*12])
			m.Buttons = atoiField(buf[1+2*12 : 1+3*12])
			m.Msec = uint32(atoiField(buf[1+3*12 : 1+4*12]))
			select {
			case mc.C <- m:
			default:
			}
			// Update after send so readmouse() gets the right value
			mc.Mouse = m
		}
	}
}

// ReadMouse reads the next mouse event, flushing the display buffer first.
// This is the synchronous read matching 9front readmouse().
func (mc *Mousectl) ReadMouse() error {
	if mc.image != nil {
		d := mc.image.Display
		if d != nil && d.bufp > 0 {
			d.Flush()
		}
	}
	m, ok := <-mc.C
	if !ok {
		return fmt.Errorf("readmouse: channel closed")
	}
	mc.Mouse = m
	return nil
}

// Read reads a mouse event, blocking until one is available.
func (mc *Mousectl) Read() Mouse {
	return <-mc.C
}

// MoveTo moves the mouse cursor to point p.
func (mc *Mousectl) MoveTo(p Point) {
	fmt.Fprintf(mc.file, "m%d %d", p.X, p.Y)
	mc.Point = p
}

// SetCursor sets the mouse cursor shape.
// Pass nil to reset to default cursor.
func (mc *Mousectl) SetCursor(c *Cursor) {
	if mc.cfd == nil {
		return
	}
	if c == nil {
		mc.cfd.Write([]byte{})
		return
	}
	// Format: offset.x[4] offset.y[4] clr[2*16] set[2*16]
	var buf [2*4 + 2*2*16]byte
	binary.LittleEndian.PutUint32(buf[0:], uint32(c.Offset.X))
	binary.LittleEndian.PutUint32(buf[4:], uint32(c.Offset.Y))
	copy(buf[8:], c.Clr[:])
	copy(buf[8+2*16:], c.Set[:])
	mc.cfd.Write(buf[:])
}

// Close closes the mouse connection.
func (mc *Mousectl) Close() {
	if mc.cfd != nil {
		mc.cfd.Close()
		mc.cfd = nil
	}
	if mc.file != nil {
		mc.file.Close()
		mc.file = nil
	}
}

// atoiField parses a whitespace-padded decimal field from a Plan 9 mouse message.
func atoiField(b []byte) int {
	s := strings.TrimSpace(string(b))
	n, _ := strconv.Atoi(s)
	return n
}
