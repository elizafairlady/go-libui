package draw

import (
	"fmt"
	"os"
)

// InitMouse opens the mouse device and returns a Mousectl.
func (d *Display) InitMouse() (*Mousectl, error) {
	f, err := os.Open("/dev/mouse")
	if err != nil {
		return nil, fmt.Errorf("initmouse: %v", err)
	}

	mc := &Mousectl{
		Display: d,
		file:    f,
		C:       make(chan Mouse, 16),
		Resize:  make(chan bool, 2),
	}

	go mc.readproc()
	return mc, nil
}

// readproc reads mouse events in a goroutine.
func (mc *Mousectl) readproc() {
	buf := make([]byte, 1+4*12)
	for {
		n, err := mc.file.Read(buf)
		if err != nil {
			close(mc.C)
			return
		}
		if n < 1 {
			continue
		}

		switch buf[0] {
		case 'm':
			// Mouse move/button event
			// Format: 'm' x[12] y[12] buttons[12] msec[12]
			if n >= 1+4*12 {
				m := Mouse{}
				m.X = atoi(string(buf[1:13]))
				m.Y = atoi(string(buf[13:25]))
				m.Buttons = atoi(string(buf[25:37]))
				m.Msec = uint32(atoi(string(buf[37:49])))
				mc.Mouse = m
				select {
				case mc.C <- m:
				default:
				}
			}
		case 'r':
			// Resize event
			select {
			case mc.Resize <- true:
			default:
			}
		}
	}
}

// Read reads a mouse event, blocking until one is available.
func (mc *Mousectl) Read() Mouse {
	return <-mc.C
}

// Close closes the mouse connection.
func (mc *Mousectl) Close() {
	if mc.file != nil {
		mc.file.Close()
	}
}

// MoveTo moves the mouse cursor to point p.
func (mc *Mousectl) MoveTo(p Point) {
	if mc.Display == nil {
		return
	}
	wctl, err := os.OpenFile("/dev/wctl", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer wctl.Close()
	fmt.Fprintf(wctl, "ptr %d %d", p.X, p.Y)
}
