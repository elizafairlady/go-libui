// Package fsys implements a 9P2000 file server for the acme window
// namespace, directly modeled on /sys/src/cmd/acme/fsys.c.
//
// The protocol implementation follows the 9P2000 specification:
// http://man.cat-v.org/plan_9/5/intro
package fsys

import (
	"encoding/binary"
	"fmt"
	"io"
)

// 9P2000 message types
const (
	Tversion = 100
	Rversion = 101
	Tauth    = 102
	Rauth    = 103
	Tattach  = 104
	Rattach  = 105
	Rerror   = 107
	Tflush   = 108
	Rflush   = 109
	Twalk    = 110
	Rwalk    = 111
	Topen    = 112
	Ropen    = 113
	Tcreate  = 114
	Rcreate  = 115
	Tread    = 116
	Rread    = 117
	Twrite   = 118
	Rwrite   = 119
	Tclunk   = 120
	Rclunk   = 121
	Tremove  = 122
	Rremove  = 123
	Tstat    = 124
	Rstat    = 125
	Twstat   = 126
	Rwstat   = 127
)

// Open modes
const (
	OREAD  = 0
	OWRITE = 1
	ORDWR  = 2
	OEXEC  = 3

	OTRUNC  = 0x10
	ORCLOSE = 0x40
)

// Qid types
const (
	QTDIR    = 0x80
	QTAPPEND = 0x40
	QTFILE   = 0x00
)

// Dir mode bits
const (
	DMDIR    = 0x80000000
	DMAPPEND = 0x40000000
)

// IOHDRSZ is the overhead for a 9P message header
const IOHDRSZ = 24

// Qid is the server's unique identification for a file
type Qid struct {
	Type uint8
	Vers uint32
	Path uint64
}

// Fcall is a 9P protocol message
type Fcall struct {
	Type uint8
	Tag  uint16
	Fid  uint32

	// Tversion, Rversion
	Msize   uint32
	Version string

	// Rerror
	Ename string

	// Tattach
	Afid  uint32
	Uname string
	Aname string

	// Twalk
	Newfid uint32
	Wname  []string

	// Rwalk
	Wqid []Qid

	// Topen
	Mode uint8

	// Ropen, Rcreate, Rattach
	Qid    Qid
	Iounit uint32

	// Tread
	Offset uint64
	Count  uint32

	// Rread
	Data []byte

	// Twrite
	// uses Offset, Count, Data

	// Rwrite
	// uses Count

	// Tflush
	Oldtag uint16

	// Rstat, Twstat
	Stat []byte
}

// ReadFcall reads one 9P message from r
func ReadFcall(r io.Reader) (*Fcall, error) {
	// Read size[4]
	var sizeBuf [4]byte
	if _, err := io.ReadFull(r, sizeBuf[:]); err != nil {
		return nil, err
	}
	size := binary.LittleEndian.Uint32(sizeBuf[:])
	if size < 4 || size > 1024*1024 {
		return nil, fmt.Errorf("bad 9P message size: %d", size)
	}

	buf := make([]byte, size)
	copy(buf[:4], sizeBuf[:])
	if _, err := io.ReadFull(r, buf[4:]); err != nil {
		return nil, err
	}

	return unmarshal(buf)
}

// WriteFcall writes one 9P message to w
func WriteFcall(w io.Writer, fc *Fcall) error {
	buf := marshal(fc)
	_, err := w.Write(buf)
	return err
}

func gstring(buf []byte, off int) (string, int) {
	if off+2 > len(buf) {
		return "", off
	}
	n := int(binary.LittleEndian.Uint16(buf[off:]))
	off += 2
	if off+n > len(buf) {
		return "", off
	}
	return string(buf[off : off+n]), off + n
}

func gqid(buf []byte, off int) (Qid, int) {
	if off+13 > len(buf) {
		return Qid{}, off
	}
	q := Qid{
		Type: buf[off],
		Vers: binary.LittleEndian.Uint32(buf[off+1:]),
		Path: binary.LittleEndian.Uint64(buf[off+5:]),
	}
	return q, off + 13
}

func unmarshal(buf []byte) (*Fcall, error) {
	if len(buf) < 7 {
		return nil, fmt.Errorf("short 9P message")
	}
	fc := &Fcall{
		Type: buf[4],
		Tag:  binary.LittleEndian.Uint16(buf[5:]),
	}
	off := 7

	switch fc.Type {
	case Tversion:
		fc.Msize = binary.LittleEndian.Uint32(buf[off:])
		off += 4
		fc.Version, off = gstring(buf, off)

	case Tauth:
		fc.Afid = binary.LittleEndian.Uint32(buf[off:])
		off += 4
		fc.Uname, off = gstring(buf, off)
		fc.Aname, off = gstring(buf, off)

	case Tattach:
		fc.Fid = binary.LittleEndian.Uint32(buf[off:])
		off += 4
		fc.Afid = binary.LittleEndian.Uint32(buf[off:])
		off += 4
		fc.Uname, off = gstring(buf, off)
		fc.Aname, off = gstring(buf, off)

	case Tflush:
		fc.Oldtag = binary.LittleEndian.Uint16(buf[off:])

	case Twalk:
		fc.Fid = binary.LittleEndian.Uint32(buf[off:])
		off += 4
		fc.Newfid = binary.LittleEndian.Uint32(buf[off:])
		off += 4
		nwname := int(binary.LittleEndian.Uint16(buf[off:]))
		off += 2
		fc.Wname = make([]string, nwname)
		for i := 0; i < nwname; i++ {
			fc.Wname[i], off = gstring(buf, off)
		}

	case Topen:
		fc.Fid = binary.LittleEndian.Uint32(buf[off:])
		off += 4
		fc.Mode = buf[off]

	case Tcreate:
		fc.Fid = binary.LittleEndian.Uint32(buf[off:])
		off += 4
		// name, perm, mode — we reject creates

	case Tread:
		fc.Fid = binary.LittleEndian.Uint32(buf[off:])
		off += 4
		fc.Offset = binary.LittleEndian.Uint64(buf[off:])
		off += 8
		fc.Count = binary.LittleEndian.Uint32(buf[off:])

	case Twrite:
		fc.Fid = binary.LittleEndian.Uint32(buf[off:])
		off += 4
		fc.Offset = binary.LittleEndian.Uint64(buf[off:])
		off += 8
		fc.Count = binary.LittleEndian.Uint32(buf[off:])
		off += 4
		fc.Data = make([]byte, fc.Count)
		copy(fc.Data, buf[off:])

	case Tclunk, Tremove:
		fc.Fid = binary.LittleEndian.Uint32(buf[off:])

	case Tstat:
		fc.Fid = binary.LittleEndian.Uint32(buf[off:])

	case Twstat:
		fc.Fid = binary.LittleEndian.Uint32(buf[off:])
		off += 4
		// stat data follows

	// --- R-messages (server → client) ---
	case Rversion:
		fc.Msize = binary.LittleEndian.Uint32(buf[off:])
		off += 4
		fc.Version, off = gstring(buf, off)

	case Rerror:
		fc.Ename, off = gstring(buf, off)

	case Rauth:
		fc.Qid, off = gqid(buf, off)

	case Rattach:
		fc.Qid, off = gqid(buf, off)

	case Rflush, Rclunk, Rremove, Rwstat:
		// no additional data

	case Rwalk:
		nwqid := int(binary.LittleEndian.Uint16(buf[off:]))
		off += 2
		fc.Wqid = make([]Qid, nwqid)
		for i := 0; i < nwqid; i++ {
			fc.Wqid[i], off = gqid(buf, off)
		}

	case Ropen, Rcreate:
		fc.Qid, off = gqid(buf, off)
		fc.Iounit = binary.LittleEndian.Uint32(buf[off:])
		off += 4

	case Rread:
		fc.Count = binary.LittleEndian.Uint32(buf[off:])
		off += 4
		fc.Data = make([]byte, fc.Count)
		copy(fc.Data, buf[off:])
		off += int(fc.Count)

	case Rwrite:
		fc.Count = binary.LittleEndian.Uint32(buf[off:])
		off += 4

	case Rstat:
		n := int(binary.LittleEndian.Uint16(buf[off:]))
		off += 2
		fc.Stat = make([]byte, n)
		copy(fc.Stat, buf[off:])
		off += n

	default:
		return nil, fmt.Errorf("unknown 9P type %d", fc.Type)
	}

	_ = off
	return fc, nil
}

func pstring(buf []byte, off int, s string) int {
	binary.LittleEndian.PutUint16(buf[off:], uint16(len(s)))
	off += 2
	copy(buf[off:], s)
	return off + len(s)
}

func pqid(buf []byte, off int, q Qid) int {
	buf[off] = q.Type
	binary.LittleEndian.PutUint32(buf[off+1:], q.Vers)
	binary.LittleEndian.PutUint64(buf[off+5:], q.Path)
	return off + 13
}

func marshal(fc *Fcall) []byte {
	// Generous allocation
	buf := make([]byte, 8192)
	off := 4 // skip size
	buf[off] = fc.Type
	off++
	binary.LittleEndian.PutUint16(buf[off:], fc.Tag)
	off += 2

	switch fc.Type {
	// --- T-messages (client → server) ---
	case Tversion:
		binary.LittleEndian.PutUint32(buf[off:], fc.Msize)
		off += 4
		off = pstring(buf, off, fc.Version)

	case Tauth:
		binary.LittleEndian.PutUint32(buf[off:], fc.Afid)
		off += 4
		off = pstring(buf, off, fc.Uname)
		off = pstring(buf, off, fc.Aname)

	case Tattach:
		binary.LittleEndian.PutUint32(buf[off:], fc.Fid)
		off += 4
		binary.LittleEndian.PutUint32(buf[off:], fc.Afid)
		off += 4
		off = pstring(buf, off, fc.Uname)
		off = pstring(buf, off, fc.Aname)

	case Tflush:
		binary.LittleEndian.PutUint16(buf[off:], fc.Oldtag)
		off += 2

	case Twalk:
		binary.LittleEndian.PutUint32(buf[off:], fc.Fid)
		off += 4
		binary.LittleEndian.PutUint32(buf[off:], fc.Newfid)
		off += 4
		binary.LittleEndian.PutUint16(buf[off:], uint16(len(fc.Wname)))
		off += 2
		for _, name := range fc.Wname {
			off = pstring(buf, off, name)
		}

	case Topen:
		binary.LittleEndian.PutUint32(buf[off:], fc.Fid)
		off += 4
		buf[off] = fc.Mode
		off++

	case Tread:
		binary.LittleEndian.PutUint32(buf[off:], fc.Fid)
		off += 4
		binary.LittleEndian.PutUint64(buf[off:], fc.Offset)
		off += 8
		binary.LittleEndian.PutUint32(buf[off:], fc.Count)
		off += 4

	case Twrite:
		binary.LittleEndian.PutUint32(buf[off:], fc.Fid)
		off += 4
		binary.LittleEndian.PutUint64(buf[off:], fc.Offset)
		off += 8
		binary.LittleEndian.PutUint32(buf[off:], fc.Count)
		off += 4
		copy(buf[off:], fc.Data)
		off += len(fc.Data)

	case Tclunk, Tremove, Tstat:
		binary.LittleEndian.PutUint32(buf[off:], fc.Fid)
		off += 4

	// --- R-messages (server → client) ---
	case Rversion:
		binary.LittleEndian.PutUint32(buf[off:], fc.Msize)
		off += 4
		off = pstring(buf, off, fc.Version)

	case Rerror:
		off = pstring(buf, off, fc.Ename)

	case Rattach, Ropen, Rcreate:
		off = pqid(buf, off, fc.Qid)
		if fc.Type != Rattach {
			binary.LittleEndian.PutUint32(buf[off:], fc.Iounit)
			off += 4
		}

	case Rflush, Rclunk, Rremove, Rwstat:
		// no additional data

	case Rwalk:
		binary.LittleEndian.PutUint16(buf[off:], uint16(len(fc.Wqid)))
		off += 2
		for _, q := range fc.Wqid {
			off = pqid(buf, off, q)
		}

	case Rread:
		binary.LittleEndian.PutUint32(buf[off:], uint32(len(fc.Data)))
		off += 4
		copy(buf[off:], fc.Data)
		off += len(fc.Data)

	case Rwrite:
		binary.LittleEndian.PutUint32(buf[off:], fc.Count)
		off += 4

	case Rstat:
		binary.LittleEndian.PutUint16(buf[off:], uint16(len(fc.Stat)))
		off += 2
		copy(buf[off:], fc.Stat)
		off += len(fc.Stat)

	case Rauth:
		off = pqid(buf, off, fc.Qid)
	}

	// Write size
	binary.LittleEndian.PutUint32(buf[:], uint32(off))
	return buf[:off]
}
