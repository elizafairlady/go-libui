package fsys

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/elizafairlady/go-libui/ui/window"
)

// File IDs within a window directory, matching acme's dat.h enum
const (
	Qdir   = iota // .
	Qcons         // cons
	Qindex        // index
	Qlog          // log
	Qnew          // new

	QWaddr   // addr
	QWbody   // body
	QWctl    // ctl
	QWdata   // data
	QWevent  // event
	QWerrors // errors
	QWrdsel  // rdsel
	QWwrsel  // wrsel
	QWtag    // tag
)

// QID encodes window ID and file type into a qid path
func qidPath(winid, file int) uint64 {
	return uint64(winid)<<8 | uint64(file)
}

func qidWin(path uint64) int  { return int(path >> 8) }
func qidFile(path uint64) int { return int(path & 0xFF) }

// dirtab is one entry in a directory listing
type dirtab struct {
	name string
	qtyp uint8
	qid  int
	perm uint32
}

// Top-level directory entries (the root /mnt/acme/)
var rootDir = []dirtab{
	{"cons", QTFILE, Qcons, 0600},
	{"index", QTFILE, Qindex, 0400},
	{"log", QTFILE, Qlog, 0400},
	{"new", QTDIR, Qnew, DMDIR | 0500},
}

// Per-window directory entries (/mnt/acme/N/)
var winDir = []dirtab{
	{"addr", QTFILE, QWaddr, 0600},
	{"body", QTAPPEND, QWbody, DMAPPEND | 0600},
	{"ctl", QTFILE, QWctl, 0600},
	{"data", QTFILE, QWdata, 0600},
	{"event", QTFILE, QWevent, 0600},
	{"errors", QTFILE, QWerrors, 0200},
	{"rdsel", QTFILE, QWrdsel, 0400},
	{"wrsel", QTFILE, QWwrsel, 0200},
	{"tag", QTAPPEND, QWtag, DMAPPEND | 0600},
}

// fid tracks the state of an open file handle
type fid struct {
	fid  uint32
	busy bool
	open bool
	qid  Qid
	w    *window.Window // nil for root-level files
	dir  *dirtab
}

// Server is a 9P2000 file server for the acme window namespace
type Server struct {
	row   *window.Row
	mu    sync.Mutex
	fids  map[uint32]*fid
	msize uint32
}

// NewServer creates a 9P server for the given Row
func NewServer(row *window.Row) *Server {
	return &Server{
		row:   row,
		fids:  make(map[uint32]*fid),
		msize: 8192 + IOHDRSZ,
	}
}

// Serve handles 9P messages on the given ReadWriteCloser
func (s *Server) Serve(rwc io.ReadWriteCloser) {
	defer rwc.Close()
	for {
		fc, err := ReadFcall(rwc)
		if err != nil {
			return
		}
		resp := s.handle(fc)
		if err := WriteFcall(rwc, resp); err != nil {
			return
		}
	}
}

// ListenAndServe starts a Unix socket listener at the given path
func (s *Server) ListenAndServe(path string) error {
	os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.Serve(conn)
		}
	}()
	return nil
}

func (s *Server) lookFid(id uint32) *fid {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fids[id]
}

func (s *Server) newFid(id uint32) *fid {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.fids[id]
	if f != nil {
		return f
	}
	f = &fid{fid: id}
	s.fids[id] = f
	return f
}

func (s *Server) delFid(id uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.fids, id)
}

func respond(tx *Fcall, err string) *Fcall {
	r := &Fcall{Tag: tx.Tag}
	if err != "" {
		r.Type = Rerror
		r.Ename = err
	} else {
		r.Type = tx.Type + 1
	}
	return r
}

func (s *Server) handle(tx *Fcall) *Fcall {
	switch tx.Type {
	case Tversion:
		return s.sVersion(tx)
	case Tauth:
		return respond(tx, "authentication not required")
	case Tattach:
		return s.sAttach(tx)
	case Tflush:
		return respond(tx, "")
	case Twalk:
		return s.sWalk(tx)
	case Topen:
		return s.sOpen(tx)
	case Tcreate:
		return respond(tx, "permission denied")
	case Tread:
		return s.sRead(tx)
	case Twrite:
		return s.sWrite(tx)
	case Tclunk:
		return s.sClunk(tx)
	case Tremove:
		return respond(tx, "permission denied")
	case Tstat:
		return s.sStat(tx)
	case Twstat:
		return respond(tx, "permission denied")
	default:
		return respond(tx, "bad fcall type")
	}
}

func (s *Server) sVersion(tx *Fcall) *Fcall {
	r := &Fcall{Type: Rversion, Tag: tx.Tag}
	r.Msize = tx.Msize
	if r.Msize > 65536 {
		r.Msize = 65536
	}
	s.msize = r.Msize
	if strings.HasPrefix(tx.Version, "9P") {
		r.Version = "9P2000"
	} else {
		r.Version = "unknown"
	}
	return r
}

func (s *Server) sAttach(tx *Fcall) *Fcall {
	f := s.newFid(tx.Fid)
	f.busy = true
	f.qid = Qid{Type: QTDIR, Path: qidPath(0, Qdir)}
	r := &Fcall{Type: Rattach, Tag: tx.Tag, Qid: f.qid}
	return r
}

func (s *Server) sWalk(tx *Fcall) *Fcall {
	f := s.lookFid(tx.Fid)
	if f == nil || !f.busy {
		return respond(tx, "fid not in use")
	}

	var nf *fid
	if tx.Fid != tx.Newfid {
		nf = s.newFid(tx.Newfid)
		nf.busy = true
		nf.qid = f.qid
		nf.w = f.w
		nf.dir = f.dir
		f = nf
	}

	r := &Fcall{Type: Rwalk, Tag: tx.Tag}
	q := f.qid
	w := f.w

	for _, name := range tx.Wname {
		if q.Type&QTDIR == 0 {
			if nf != nil {
				nf.busy = false
			}
			return respond(tx, "not a directory")
		}

		if name == ".." {
			q = Qid{Type: QTDIR, Path: qidPath(0, Qdir)}
			w = nil
			r.Wqid = append(r.Wqid, q)
			continue
		}

		winid := qidWin(q.Path)

		// Try numeric name (window directory)
		if id, err := strconv.Atoi(name); err == nil {
			ww := s.row.LookID(id)
			if ww != nil {
				w = ww
				q = Qid{Type: QTDIR, Path: qidPath(id, Qdir)}
				r.Wqid = append(r.Wqid, q)
				continue
			}
		}

		// "new" — create a new window
		if name == "new" && winid == 0 {
			if len(s.row.Cols) == 0 {
				s.row.NewColumn()
			}
			ww := s.row.NewWindow(s.row.Cols[0])
			ww.Tag.SetAll("scratch Del Snarf Get Put Look |")
			w = ww
			q = Qid{Type: QTDIR, Path: qidPath(ww.ID, Qdir)}
			r.Wqid = append(r.Wqid, q)
			continue
		}

		// Look in appropriate directory table
		var dirs []dirtab
		if winid == 0 {
			dirs = rootDir
		} else {
			dirs = winDir
		}

		found := false
		for _, d := range dirs {
			if d.name == name {
				q = Qid{Type: d.qtyp, Path: qidPath(winid, d.qid)}
				r.Wqid = append(r.Wqid, q)
				found = true
				break
			}
		}
		if !found {
			if nf != nil && len(r.Wqid) == 0 {
				nf.busy = false
			}
			if len(r.Wqid) == 0 {
				return respond(tx, "file does not exist")
			}
			break // partial walk
		}
	}

	if len(r.Wqid) == len(tx.Wname) {
		f.qid = q
		f.w = w
	}
	return r
}

func (s *Server) sOpen(tx *Fcall) *Fcall {
	f := s.lookFid(tx.Fid)
	if f == nil || !f.busy {
		return respond(tx, "fid not in use")
	}
	f.open = true
	r := &Fcall{Type: Ropen, Tag: tx.Tag, Qid: f.qid, Iounit: s.msize - IOHDRSZ}
	return r
}

func (s *Server) sRead(tx *Fcall) *Fcall {
	f := s.lookFid(tx.Fid)
	if f == nil || !f.busy {
		return respond(tx, "fid not in use")
	}

	q := qidFile(f.qid.Path)
	winid := qidWin(f.qid.Path)
	r := &Fcall{Type: Rread, Tag: tx.Tag}

	// Directory read
	if f.qid.Type&QTDIR != 0 {
		r.Data = s.readDir(winid, tx.Offset, tx.Count)
		return r
	}

	w := f.w
	if w == nil && winid > 0 {
		w = s.row.LookID(winid)
	}

	switch q {
	case Qcons:
		r.Data = nil

	case Qindex:
		r.Data = s.readIndex(tx.Offset, tx.Count)

	case QWbody:
		if w != nil {
			data := []byte(w.Body.ReadAll())
			r.Data = sliceRead(data, tx.Offset, tx.Count)
		}

	case QWtag:
		if w != nil {
			data := []byte(w.Tag.ReadAll())
			r.Data = sliceRead(data, tx.Offset, tx.Count)
		}

	case QWctl:
		if w != nil {
			data := []byte(w.CtlPrint())
			r.Data = sliceRead(data, tx.Offset, tx.Count)
		}

	case QWaddr:
		if w != nil {
			data := []byte(fmt.Sprintf("%11d %11d ", w.Addr.Q0, w.Addr.Q1))
			r.Data = sliceRead(data, tx.Offset, tx.Count)
		}

	case QWdata:
		if w != nil {
			text := w.Body.ReadRange(w.Addr.Q0, w.Body.Nc())
			data := []byte(text)
			r.Data = sliceRead(data, tx.Offset, tx.Count)
		}

	case QWrdsel:
		if w != nil {
			text := w.Body.ReadRange(w.Sel.Q0, w.Sel.Q1)
			data := []byte(text)
			r.Data = sliceRead(data, tx.Offset, tx.Count)
		}

	case QWevent:
		if w != nil {
			data := []byte(w.Events)
			r.Data = sliceRead(data, tx.Offset, tx.Count)
			// Consume read events
			n := int(tx.Offset) + len(r.Data)
			if n >= len(w.Events) {
				w.Events = ""
			} else {
				w.Events = w.Events[n:]
			}
		}

	default:
		r.Data = nil
	}

	return r
}

func (s *Server) sWrite(tx *Fcall) *Fcall {
	f := s.lookFid(tx.Fid)
	if f == nil || !f.busy {
		return respond(tx, "fid not in use")
	}

	q := qidFile(f.qid.Path)
	winid := qidWin(f.qid.Path)
	r := &Fcall{Type: Rwrite, Tag: tx.Tag, Count: tx.Count}

	w := f.w
	if w == nil && winid > 0 {
		w = s.row.LookID(winid)
	}

	switch q {
	case Qcons:
		// Write to cons → TODO: append to +Errors
		os.Stderr.Write(tx.Data)

	case QWbody:
		if w != nil {
			// Append to body (DMAPPEND mode)
			w.Body.Insert(w.Body.Nc(), []rune(string(tx.Data)))
		}

	case QWtag:
		if w != nil {
			// Append to tag (DMAPPEND mode)
			w.Tag.Insert(w.Tag.Nc(), []rune(string(tx.Data)))
		}

	case QWctl:
		if w != nil {
			if err := w.Ctl(string(tx.Data)); err != nil {
				return respond(tx, err.Error())
			}
		}

	case QWaddr:
		if w != nil {
			if err := w.ParseAddr(string(tx.Data)); err != nil {
				return respond(tx, err.Error())
			}
		}

	case QWdata:
		if w != nil {
			// Write at addr, replacing addr range, like acme's xfidwrite QWdata
			runes := []rune(string(tx.Data))
			if w.Addr.Q1 > w.Addr.Q0 {
				w.Body.Delete(w.Addr.Q0, w.Addr.Q1)
			}
			w.Body.Insert(w.Addr.Q0, runes)
			w.Addr.Q0 += len(runes)
			w.Addr.Q1 = w.Addr.Q0
		}

	case QWwrsel:
		if w != nil {
			// Write replaces selection
			runes := []rune(string(tx.Data))
			if w.Sel.Q1 > w.Sel.Q0 {
				w.Body.Delete(w.Sel.Q0, w.Sel.Q1)
			}
			w.Body.Insert(w.Sel.Q0, runes)
			w.Sel.Q1 = w.Sel.Q0 + len(runes)
		}

	case QWevent:
		if w != nil {
			// Write events back — the program wants acme to handle them
			w.Events += string(tx.Data)
		}

	case QWerrors:
		// TODO: append to +Errors window
		os.Stderr.Write(tx.Data)

	default:
		return respond(tx, "write not allowed")
	}

	return r
}

func (s *Server) sClunk(tx *Fcall) *Fcall {
	s.delFid(tx.Fid)
	return &Fcall{Type: Rclunk, Tag: tx.Tag}
}

func (s *Server) sStat(tx *Fcall) *Fcall {
	f := s.lookFid(tx.Fid)
	if f == nil || !f.busy {
		return respond(tx, "fid not in use")
	}

	winid := qidWin(f.qid.Path)
	file := qidFile(f.qid.Path)

	var d dirtab
	if f.qid.Type&QTDIR != 0 {
		if winid == 0 {
			d = dirtab{".", QTDIR, Qdir, DMDIR | 0500}
		} else {
			d = dirtab{strconv.Itoa(winid), QTDIR, Qdir, DMDIR | 0500}
		}
	} else {
		// Find in directory table
		dirs := rootDir
		if winid > 0 {
			dirs = winDir
		}
		for _, dd := range dirs {
			if dd.qid == file {
				d = dd
				break
			}
		}
	}

	stat := makeStat(winid, d)
	r := &Fcall{Type: Rstat, Tag: tx.Tag, Stat: stat}
	return r
}

// readDir generates a directory listing
func (s *Server) readDir(winid int, offset uint64, count uint32) []byte {
	var entries []dirtab

	if winid == 0 {
		// Root directory: top-level files + window directories
		entries = append(entries, rootDir...)
		for _, c := range s.row.Cols {
			for _, w := range c.Windows {
				entries = append(entries, dirtab{
					strconv.Itoa(w.ID), QTDIR, Qdir, DMDIR | 0700,
				})
			}
		}
	} else {
		// Window directory
		entries = winDir
	}

	// Generate stat entries
	var buf []byte
	for _, d := range entries {
		stat := makeStat(winid, d)
		buf = append(buf, stat...)
	}

	return sliceRead(buf, offset, count)
}

func (s *Server) readIndex(offset uint64, count uint32) []byte {
	var sb strings.Builder
	for _, c := range s.row.Cols {
		for _, w := range c.Windows {
			sb.WriteString(w.Index())
		}
	}
	return sliceRead([]byte(sb.String()), offset, count)
}

// makeStat creates a 9P stat entry
func makeStat(winid int, d dirtab) []byte {
	// Plan 9 stat format:
	// size[2] type[2] dev[4] qid[13] mode[4] atime[4] mtime[4]
	// length[8] name[s] uid[s] gid[s] muid[s]

	name := d.name
	uid := "acme"
	gid := "acme"
	muid := "acme"

	now := uint32(time.Now().Unix())

	// Calculate size
	fixedLen := 2 + 4 + 13 + 4 + 4 + 4 + 8 + 2 + 2 + 2 + 2 // size fields (excl size[2] itself)
	strLen := len(name) + len(uid) + len(gid) + len(muid)
	statLen := fixedLen + strLen

	buf := make([]byte, 2+statLen)
	off := 0

	// size[2] — does not include itself
	binary.LittleEndian.PutUint16(buf[off:], uint16(statLen))
	off += 2

	// type[2]
	binary.LittleEndian.PutUint16(buf[off:], 0)
	off += 2

	// dev[4]
	binary.LittleEndian.PutUint32(buf[off:], 0)
	off += 4

	// qid[13]
	var qpath uint64
	if d.name != "." {
		qpath = qidPath(winid, d.qid)
	}
	buf[off] = d.qtyp
	binary.LittleEndian.PutUint32(buf[off+1:], 0) // vers
	binary.LittleEndian.PutUint64(buf[off+5:], qpath)
	off += 13

	// mode[4]
	binary.LittleEndian.PutUint32(buf[off:], d.perm)
	off += 4

	// atime[4]
	binary.LittleEndian.PutUint32(buf[off:], now)
	off += 4

	// mtime[4]
	binary.LittleEndian.PutUint32(buf[off:], now)
	off += 4

	// length[8]
	binary.LittleEndian.PutUint64(buf[off:], 0)
	off += 8

	// name[s]
	off = pstring(buf, off, name)

	// uid[s]
	off = pstring(buf, off, uid)

	// gid[s]
	off = pstring(buf, off, gid)

	// muid[s]
	off = pstring(buf, off, muid)

	return buf[:off]
}

func sliceRead(data []byte, offset uint64, count uint32) []byte {
	off := int(offset)
	if off >= len(data) {
		return nil
	}
	end := off + int(count)
	if end > len(data) {
		end = len(data)
	}
	return data[off:end]
}
