// Package acmefsys implements the 9P2000 file server for the acme
// window namespace, modeled on /sys/src/cmd/acme/fsys.c.
package acmefsys

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

	p9 "github.com/elizafairlady/go-libui/ui/fsys"

	"github.com/elizafairlady/go-libui/ui/cmd/acme/window"
)

// File IDs within a window directory
const (
	Qdir = iota
	Qcons
	Qindex
	Qlog
	Qnew

	QWaddr
	QWbody
	QWctl
	QWdata
	QWevent
	QWerrors
	QWrdsel
	QWwrsel
	QWtag
)

func qidPath(winid, file int) uint64 {
	return uint64(winid)<<8 | uint64(file)
}

func qidWin(path uint64) int  { return int(path >> 8) }
func qidFile(path uint64) int { return int(path & 0xFF) }

type dirtab struct {
	name string
	qtyp uint8
	qid  int
	perm uint32
}

var rootDir = []dirtab{
	{"cons", p9.QTFILE, Qcons, 0600},
	{"index", p9.QTFILE, Qindex, 0400},
	{"log", p9.QTFILE, Qlog, 0400},
	{"new", p9.QTDIR, Qnew, p9.DMDIR | 0500},
}

var winDir = []dirtab{
	{"addr", p9.QTFILE, QWaddr, 0600},
	{"body", p9.QTAPPEND, QWbody, p9.DMAPPEND | 0600},
	{"ctl", p9.QTFILE, QWctl, 0600},
	{"data", p9.QTFILE, QWdata, 0600},
	{"event", p9.QTFILE, QWevent, 0600},
	{"errors", p9.QTFILE, QWerrors, 0200},
	{"rdsel", p9.QTFILE, QWrdsel, 0400},
	{"wrsel", p9.QTFILE, QWwrsel, 0200},
	{"tag", p9.QTAPPEND, QWtag, p9.DMAPPEND | 0600},
}

type fid struct {
	fid  uint32
	busy bool
	open bool
	qid  p9.Qid
	w    *window.Window
	dir  *dirtab
}

// Server is a 9P2000 file server for the acme window namespace.
type Server struct {
	row   *window.Row
	mu    sync.Mutex
	fids  map[uint32]*fid
	msize uint32
}

// NewServer creates a 9P server for the given Row.
func NewServer(row *window.Row) *Server {
	return &Server{
		row:   row,
		fids:  make(map[uint32]*fid),
		msize: 8192 + p9.IOHDRSZ,
	}
}

// Serve handles 9P messages on the given ReadWriteCloser.
func (s *Server) Serve(rwc io.ReadWriteCloser) {
	defer rwc.Close()
	for {
		fc, err := p9.ReadFcall(rwc)
		if err != nil {
			return
		}
		resp := s.handle(fc)
		if err := p9.WriteFcall(rwc, resp); err != nil {
			return
		}
	}
}

// ListenAndServe starts a Unix socket listener.
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

func respond(tx *p9.Fcall, err string) *p9.Fcall {
	r := &p9.Fcall{Tag: tx.Tag}
	if err != "" {
		r.Type = p9.Rerror
		r.Ename = err
	} else {
		r.Type = tx.Type + 1
	}
	return r
}

func (s *Server) handle(tx *p9.Fcall) *p9.Fcall {
	switch tx.Type {
	case p9.Tversion:
		return s.sVersion(tx)
	case p9.Tauth:
		return respond(tx, "authentication not required")
	case p9.Tattach:
		return s.sAttach(tx)
	case p9.Tflush:
		return respond(tx, "")
	case p9.Twalk:
		return s.sWalk(tx)
	case p9.Topen:
		return s.sOpen(tx)
	case p9.Tcreate:
		return respond(tx, "permission denied")
	case p9.Tread:
		return s.sRead(tx)
	case p9.Twrite:
		return s.sWrite(tx)
	case p9.Tclunk:
		return s.sClunk(tx)
	case p9.Tremove:
		return respond(tx, "permission denied")
	case p9.Tstat:
		return s.sStat(tx)
	case p9.Twstat:
		return respond(tx, "permission denied")
	default:
		return respond(tx, "bad fcall type")
	}
}

func (s *Server) sVersion(tx *p9.Fcall) *p9.Fcall {
	r := &p9.Fcall{Type: p9.Rversion, Tag: tx.Tag}
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

func (s *Server) sAttach(tx *p9.Fcall) *p9.Fcall {
	f := s.newFid(tx.Fid)
	f.busy = true
	f.qid = p9.Qid{Type: p9.QTDIR, Path: qidPath(0, Qdir)}
	r := &p9.Fcall{Type: p9.Rattach, Tag: tx.Tag, Qid: f.qid}
	return r
}

func (s *Server) sWalk(tx *p9.Fcall) *p9.Fcall {
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

	r := &p9.Fcall{Type: p9.Rwalk, Tag: tx.Tag}
	q := f.qid
	w := f.w

	for _, name := range tx.Wname {
		if q.Type&p9.QTDIR == 0 {
			if nf != nil {
				nf.busy = false
			}
			return respond(tx, "not a directory")
		}

		if name == ".." {
			q = p9.Qid{Type: p9.QTDIR, Path: qidPath(0, Qdir)}
			w = nil
			r.Wqid = append(r.Wqid, q)
			continue
		}

		winid := qidWin(q.Path)

		if id, err := strconv.Atoi(name); err == nil {
			ww := s.row.LookID(id)
			if ww != nil {
				w = ww
				q = p9.Qid{Type: p9.QTDIR, Path: qidPath(id, Qdir)}
				r.Wqid = append(r.Wqid, q)
				continue
			}
		}

		if name == "new" && winid == 0 {
			if len(s.row.Cols) == 0 {
				s.row.NewColumn()
			}
			ww := s.row.NewWindow(s.row.Cols[0])
			ww.Tag.SetAll("scratch Del Snarf Get Put Look |")
			w = ww
			q = p9.Qid{Type: p9.QTDIR, Path: qidPath(ww.ID, Qdir)}
			r.Wqid = append(r.Wqid, q)
			continue
		}

		var dirs []dirtab
		if winid == 0 {
			dirs = rootDir
		} else {
			dirs = winDir
		}

		found := false
		for _, d := range dirs {
			if d.name == name {
				q = p9.Qid{Type: d.qtyp, Path: qidPath(winid, d.qid)}
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
			break
		}
	}

	if len(r.Wqid) == len(tx.Wname) {
		f.qid = q
		f.w = w
	}
	return r
}

func (s *Server) sOpen(tx *p9.Fcall) *p9.Fcall {
	f := s.lookFid(tx.Fid)
	if f == nil || !f.busy {
		return respond(tx, "fid not in use")
	}
	f.open = true
	r := &p9.Fcall{Type: p9.Ropen, Tag: tx.Tag, Qid: f.qid, Iounit: s.msize - p9.IOHDRSZ}
	return r
}

func (s *Server) sRead(tx *p9.Fcall) *p9.Fcall {
	f := s.lookFid(tx.Fid)
	if f == nil || !f.busy {
		return respond(tx, "fid not in use")
	}

	q := qidFile(f.qid.Path)
	winid := qidWin(f.qid.Path)
	r := &p9.Fcall{Type: p9.Rread, Tag: tx.Tag}

	if f.qid.Type&p9.QTDIR != 0 {
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

func (s *Server) sWrite(tx *p9.Fcall) *p9.Fcall {
	f := s.lookFid(tx.Fid)
	if f == nil || !f.busy {
		return respond(tx, "fid not in use")
	}

	q := qidFile(f.qid.Path)
	winid := qidWin(f.qid.Path)
	r := &p9.Fcall{Type: p9.Rwrite, Tag: tx.Tag, Count: tx.Count}

	w := f.w
	if w == nil && winid > 0 {
		w = s.row.LookID(winid)
	}

	switch q {
	case Qcons:
		os.Stderr.Write(tx.Data)
	case QWbody:
		if w != nil {
			w.Body.Insert(w.Body.Nc(), []rune(string(tx.Data)))
		}
	case QWtag:
		if w != nil {
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
			runes := []rune(string(tx.Data))
			if w.Sel.Q1 > w.Sel.Q0 {
				w.Body.Delete(w.Sel.Q0, w.Sel.Q1)
			}
			w.Body.Insert(w.Sel.Q0, runes)
			w.Sel.Q1 = w.Sel.Q0 + len(runes)
		}
	case QWevent:
		if w != nil {
			w.Events += string(tx.Data)
		}
	case QWerrors:
		os.Stderr.Write(tx.Data)
	default:
		return respond(tx, "write not allowed")
	}

	return r
}

func (s *Server) sClunk(tx *p9.Fcall) *p9.Fcall {
	s.delFid(tx.Fid)
	return &p9.Fcall{Type: p9.Rclunk, Tag: tx.Tag}
}

func (s *Server) sStat(tx *p9.Fcall) *p9.Fcall {
	f := s.lookFid(tx.Fid)
	if f == nil || !f.busy {
		return respond(tx, "fid not in use")
	}

	winid := qidWin(f.qid.Path)
	file := qidFile(f.qid.Path)

	var d dirtab
	if f.qid.Type&p9.QTDIR != 0 {
		if winid == 0 {
			d = dirtab{".", p9.QTDIR, Qdir, p9.DMDIR | 0500}
		} else {
			d = dirtab{strconv.Itoa(winid), p9.QTDIR, Qdir, p9.DMDIR | 0500}
		}
	} else {
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
	r := &p9.Fcall{Type: p9.Rstat, Tag: tx.Tag, Stat: stat}
	return r
}

func (s *Server) readDir(winid int, offset uint64, count uint32) []byte {
	var entries []dirtab

	if winid == 0 {
		entries = append(entries, rootDir...)
		for _, c := range s.row.Cols {
			for _, w := range c.Windows {
				entries = append(entries, dirtab{
					strconv.Itoa(w.ID), p9.QTDIR, Qdir, p9.DMDIR | 0700,
				})
			}
		}
	} else {
		entries = winDir
	}

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

func makeStat(winid int, d dirtab) []byte {
	name := d.name
	uid := "acme"
	gid := "acme"
	muid := "acme"

	now := uint32(time.Now().Unix())

	fixedLen := 2 + 4 + 13 + 4 + 4 + 4 + 8 + 2 + 2 + 2 + 2
	strLen := len(name) + len(uid) + len(gid) + len(muid)
	statLen := fixedLen + strLen

	buf := make([]byte, 2+statLen)
	off := 0

	binary.LittleEndian.PutUint16(buf[off:], uint16(statLen))
	off += 2
	binary.LittleEndian.PutUint16(buf[off:], 0)
	off += 2
	binary.LittleEndian.PutUint32(buf[off:], 0)
	off += 4

	var qpath uint64
	if d.name != "." {
		qpath = qidPath(winid, d.qid)
	}
	buf[off] = d.qtyp
	binary.LittleEndian.PutUint32(buf[off+1:], 0)
	binary.LittleEndian.PutUint64(buf[off+5:], qpath)
	off += 13

	binary.LittleEndian.PutUint32(buf[off:], d.perm)
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], now)
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], now)
	off += 4
	binary.LittleEndian.PutUint64(buf[off:], 0)
	off += 8

	off = pstring(buf, off, name)
	off = pstring(buf, off, uid)
	off = pstring(buf, off, gid)
	off = pstring(buf, off, muid)

	return buf[:off]
}

func pstring(buf []byte, off int, s string) int {
	binary.LittleEndian.PutUint16(buf[off:], uint16(len(s)))
	off += 2
	copy(buf[off:], s)
	return off + len(s)
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
