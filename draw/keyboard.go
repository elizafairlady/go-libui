package draw

import (
	"fmt"
	"os"
	"unicode/utf8"
)

// Key constants from keyboard.h.
// KF is the start of the function key range in private Unicode space.
const (
	KF   = 0xF000 // beginning of private Unicode space
	Spec = 0xF800
	PF   = Spec | 0x20 // num pad function key
)

// Function keys and navigation keys.
const (
	Kview          = Spec | 0x00 // view (shift window up)
	KF1            = KF | 1
	KF2            = KF | 2
	KF3            = KF | 3
	KF4            = KF | 4
	KF5            = KF | 5
	KF6            = KF | 6
	KF7            = KF | 7
	KF8            = KF | 8
	KF9            = KF | 9
	KF10           = KF | 0xA
	KF11           = KF | 0xB
	KF12           = KF | 0xC
	Khome          = KF | 0x0D
	Kup            = KF | 0x0E
	Kdown          = Kview
	Kpgup          = KF | 0x0F
	Kprint         = KF | 0x10
	Kleft          = KF | 0x11
	Kright         = KF | 0x12
	Kpgdown        = KF | 0x13
	Kins           = KF | 0x14
	Kalt           = KF | 0x15
	Kshift         = KF | 0x16
	Kctl           = KF | 0x17
	Kend           = KF | 0x18
	Kscroll        = KF | 0x19
	Kscrolloneup   = KF | 0x20
	Kscrollonedown = KF | 0x21
)

// Multimedia keys.
const (
	Ksbwd  = KF | 0x22 // skip backwards
	Ksfwd  = KF | 0x23 // skip forward
	Kpause = KF | 0x24 // play/pause
	Kvoldn = KF | 0x25 // volume decrement
	Kvolup = KF | 0x26 // volume increment
	Kmute  = KF | 0x27 // (un)mute
	Kbrtdn = KF | 0x28 // brightness decrement
	Kbrtup = KF | 0x29 // brightness increment
)

// Control characters.
const (
	Ksoh  = 0x01
	Kstx  = 0x02
	Ketx  = 0x03
	Keof  = 0x04
	Kenq  = 0x05
	Kack  = 0x06
	Kbs   = 0x08
	Knack = 0x15
	Ketb  = 0x17
	Kdel  = 0x7f
	Kesc  = 0x1b
)

// Special keys.
const (
	Kbreak  = Spec | 0x61
	Kcaps   = Spec | 0x64
	Knum    = Spec | 0x65
	Kmiddle = Spec | 0x66
	Kaltgr  = Spec | 0x67
	Kmod4   = Spec | 0x68
	Kmouse  = Spec | 0x100
)

// InitKeyboard opens the keyboard device and returns a Keyboardctl.
// If file is empty, it defaults to /dev/cons.
func InitKeyboard(file string) (*Keyboardctl, error) {
	if file == "" {
		file = "/dev/cons"
	}

	consfd, err := os.OpenFile(file, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("initkeyboard: %v", err)
	}

	ctlfile := file + "ctl"
	ctlfd, err := os.OpenFile(ctlfile, os.O_WRONLY, 0)
	if err != nil {
		consfd.Close()
		return nil, fmt.Errorf("initkeyboard: can't open %s: %v", ctlfile, err)
	}

	kc := &Keyboardctl{
		C:     make(chan rune, 20),
		file:  consfd,
		ctlfd: ctlfd,
	}

	if err := kc.Ctl("rawon"); err != nil {
		ctlfd.Close()
		consfd.Close()
		return nil, fmt.Errorf("initkeyboard: can't turn on raw mode: %v", err)
	}

	go kc.readproc()
	return kc, nil
}

// readproc reads keyboard input in a goroutine, decoding UTF-8 runes
// and sending them on kc.C.
func (kc *Keyboardctl) readproc() {
	buf := make([]byte, 20)
	n := 0
	for {
		m, err := kc.file.Read(buf[n:])
		if err != nil || m <= 0 {
			close(kc.C)
			return
		}
		n += m
		for n > 0 && utf8.FullRune(buf[:n]) {
			r, size := utf8.DecodeRune(buf[:n])
			n -= size
			copy(buf, buf[size:size+n])
			select {
			case kc.C <- r:
			default:
				// drop if channel full
			}
		}
	}
}

// Read reads a rune from the keyboard, blocking until one is available.
func (kc *Keyboardctl) Read() rune {
	return <-kc.C
}

// Ctl writes a control message to the keyboard ctl file.
func (kc *Keyboardctl) Ctl(msg string) error {
	_, err := kc.ctlfd.Write([]byte(msg))
	return err
}

// Close closes the keyboard connection.
func (kc *Keyboardctl) Close() {
	if kc.ctlfd != nil {
		kc.ctlfd.Close()
		kc.ctlfd = nil
	}
	if kc.file != nil {
		kc.file.Close()
		kc.file = nil
	}
}
