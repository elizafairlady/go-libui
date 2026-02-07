package draw

import (
	"time"
)

// Event types for einit/eread.
const (
	Emouse    = 1
	Ekeyboard = 2
)

// Event system limits from event.h.
const (
	MAXSLAVE = 32
	EMAXMSG  = 128 + 8192 // size of 9p header + data
)

// Event structure matching 9front event.h.
type Event struct {
	Kbdc  rune
	Mouse Mouse
	N     int           // number of characters in message
	V     interface{}   // data unpacked by general event-handling function
	Data  [EMAXMSG]byte // message from an arbitrary file descriptor
}

// Eventctl manages the event system
type Eventctl struct {
	Display  *Display
	Mouse    *Mousectl
	Keyboard *Keyboardctl
	Screen   *Image
}

// Einit initializes the event system.
// keys is a mask of Emouse and/or Ekeyboard.
func (d *Display) Einit(keys int) (*Eventctl, error) {
	ec := &Eventctl{
		Display: d,
		Screen:  d.Image,
	}

	var err error
	if keys&Emouse != 0 {
		ec.Mouse, err = InitMouse("", d.Image)
		if err != nil {
			return nil, err
		}
	}
	if keys&Ekeyboard != 0 {
		ec.Keyboard, err = InitKeyboard("")
		if err != nil {
			if ec.Mouse != nil {
				ec.Mouse.Close()
			}
			return nil, err
		}
	}

	return ec, nil
}

// Eread waits for an event and returns its type.
func (ec *Eventctl) Eread(keys int, ev *Event) int {
	for {
		select {
		case m, ok := <-ec.Mouse.C:
			if !ok {
				return 0
			}
			if keys&Emouse != 0 {
				ev.Mouse = m
				return Emouse
			}
		case r, ok := <-ec.Keyboard.C:
			if !ok {
				return 0
			}
			if keys&Ekeyboard != 0 {
				ev.Kbdc = r
				return Ekeyboard
			}
		case <-ec.Mouse.Resize:
			// Handle resize
			ec.Display.GetWindow(Refnone)
			ec.Screen = ec.Display.Image
		}
	}
}

// Emouse reads a mouse event.
func (ec *Eventctl) Emouse() Mouse {
	return ec.Mouse.Read()
}

// Ekbd reads a keyboard event.
func (ec *Eventctl) Ekbd() rune {
	return ec.Keyboard.Read()
}

// Ecanread returns true if an event of the given type is ready.
func (ec *Eventctl) Ecanread(keys int) bool {
	if keys&Emouse != 0 && len(ec.Mouse.C) > 0 {
		return true
	}
	if keys&Ekeyboard != 0 && len(ec.Keyboard.C) > 0 {
		return true
	}
	return false
}

// Ecanmouse returns true if a mouse event is ready.
func (ec *Eventctl) Ecanmouse() bool {
	return len(ec.Mouse.C) > 0
}

// Ecankbd returns true if a keyboard event is ready.
func (ec *Eventctl) Ecankbd() bool {
	return len(ec.Keyboard.C) > 0
}

// Etimer creates a timer channel that sends periodically.
func Etimer(period time.Duration) <-chan time.Time {
	return time.Tick(period)
}

// Close closes all event resources.
func (ec *Eventctl) Close() {
	if ec.Mouse != nil {
		ec.Mouse.Close()
	}
	if ec.Keyboard != nil {
		ec.Keyboard.Close()
	}
}
