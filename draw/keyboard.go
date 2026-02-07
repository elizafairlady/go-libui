package draw

import (
	"fmt"
	"os"
	"unicode/utf8"
)

// InitKeyboard opens the keyboard device and returns a Keyboardctl.
func (d *Display) InitKeyboard() (*Keyboardctl, error) {
	// Try cons first, then kbd
	f, err := os.Open("/dev/cons")
	if err != nil {
		f, err = os.Open("/dev/kbd")
		if err != nil {
			return nil, fmt.Errorf("initkeyboard: %v", err)
		}
	}

	// Set console to raw mode
	ctl, err := os.OpenFile("/dev/consctl", os.O_WRONLY, 0)
	if err == nil {
		ctl.Write([]byte("rawon"))
		ctl.Close()
	}

	kc := &Keyboardctl{
		file: f,
		C:    make(chan rune, 32),
	}

	go kc.readproc()
	return kc, nil
}

// readproc reads keyboard events in a goroutine.
func (kc *Keyboardctl) readproc() {
	buf := make([]byte, 128)
	for {
		n, err := kc.file.Read(buf)
		if err != nil {
			close(kc.C)
			return
		}
		if n == 0 {
			continue
		}

		// Decode UTF-8 runes
		data := buf[:n]
		for len(data) > 0 {
			r, size := utf8.DecodeRune(data)
			if r == utf8.RuneError && size == 1 {
				// Invalid UTF-8, skip byte
				data = data[1:]
				continue
			}
			select {
			case kc.C <- r:
			default:
			}
			data = data[size:]
		}
	}
}

// Read reads a rune from the keyboard, blocking until one is available.
func (kc *Keyboardctl) Read() rune {
	return <-kc.C
}

// Close closes the keyboard connection.
func (kc *Keyboardctl) Close() {
	if kc.file != nil {
		kc.file.Close()
	}
	// Restore console mode
	ctl, err := os.OpenFile("/dev/consctl", os.O_WRONLY, 0)
	if err == nil {
		ctl.Write([]byte("rawoff"))
		ctl.Close()
	}
}

// Special key constants
const (
	KeyHome      = 0xF000
	KeyUp        = 0xF001
	KeyPgup      = 0xF002
	KeyPrint     = 0xF003
	KeyLeft      = 0xF004
	KeyRight     = 0xF005
	KeyDown      = 0xF006
	KeyView      = 0xF007
	KeyPgdown    = KeyView
	KeyEnd       = 0xF008
	KeyInsert    = 0xF009
	KeyAlt       = 0xF00A
	KeyShift     = 0xF00B
	KeyCtl       = 0xF00C
	KeyBackspace = 0x08
	KeyDelete    = 0x7F
	KeyEscape    = 0x1B
	KeyEOF       = 0x04
	KeyCmd       = 0xF00D
)
