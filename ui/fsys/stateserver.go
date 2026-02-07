// stateserver.go implements a generic 9P2000 server that exposes the
// UIFS state tree, view tree, actions, and body/tag text as files.
//
// Namespace:
//
//	/             directory (root)
//	/tree         read: serialized view tree
//	/actions      write: process action line
//	/focus        read/write: focused node ID
//	/state/       directory: state keys
//	/state/<key>  read: state.Get(key); write: state.Set(key, value)
//	/body/        directory: body node IDs
//	/body/<id>    read: body text; write: set body text
//	/tag/         directory: tag node IDs
//	/tag/<id>     read: tag text
package fsys

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// StateProvider is the interface the state server needs from the UIFS
// and renderer. This decouples the server from concrete types.
type StateProvider interface {
	// State access
	GetState(path string) string
	SetState(path, value string)
	ListState(dir string) []string

	// Tree
	TreeText() string

	// Actions
	ProcessAction(line string) error

	// Focus
	GetFocus() string
	SetFocus(id string)

	// Body text (from renderer)
	BodyText(id string) string
	SetBodyText(id, text string)
	BodyIDs() []string

	// Tag text (from renderer)
	TagText(id string) string
	TagIDs() []string
}

// File qid paths for the state server namespace
const (
	qRoot    = iota
	qTree    // /tree
	qActions // /actions
	qFocus   // /focus
	qStateD  // /state/
	qBodyD   // /body/
	qTagD    // /tag/

	qStateBase = 0x1000 // /state/<key> start at this offset
	qBodyBase  = 0x2000 // /body/<id>
	qTagBase   = 0x3000 // /tag/<id>
)

type stFid struct {
	busy bool
	open bool
	qid  Qid
	path string // the resolved path (e.g. "count" for /state/count)
}

// StateServer is a 9P2000 file server for the UIFS state tree.
type StateServer struct {
	prov  StateProvider
	mu    sync.Mutex
	fids  map[uint32]*stFid
	msize uint32
}

// NewStateServer creates a state server backed by the given provider.
func NewStateServer(prov StateProvider) *StateServer {
	return &StateServer{
		prov:  prov,
		fids:  make(map[uint32]*stFid),
		msize: 8192 + IOHDRSZ,
	}
}

// Serve handles 9P messages on the given ReadWriteCloser.
func (s *StateServer) Serve(rwc io.ReadWriteCloser) {
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

// Post posts the 9P server to /srv/<name> so clients can mount it.
// This is the Plan 9 way: create a pipe, post one end to /srv,
// serve 9P on the other end.
func (s *StateServer) Post(name string) error {
	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("pipe: %w", err)
	}

	// Post the read end to /srv
	srvPath := "/srv/" + name
	os.Remove(srvPath)
	f, err := os.Create(srvPath)
	if err != nil {
		r.Close()
		w.Close()
		return fmt.Errorf("create %s: %w", srvPath, err)
	}
	_, err = fmt.Fprintf(f, "%d", r.Fd())
	f.Close()
	if err != nil {
		r.Close()
		w.Close()
		return fmt.Errorf("post %s: %w", srvPath, err)
	}
	r.Close() // kernel has the fd now

	// Serve on the write end
	go s.Serve(w)
	return nil
}

func (s *StateServer) lookFid(id uint32) *stFid {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fids[id]
}

func (s *StateServer) newFid(id uint32) *stFid {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.fids[id]
	if f != nil {
		return f
	}
	f = &stFid{}
	s.fids[id] = f
	return f
}

func (s *StateServer) delFid(id uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.fids, id)
}

func stRespond(tx *Fcall, errStr string) *Fcall {
	r := &Fcall{Tag: tx.Tag}
	if errStr != "" {
		r.Type = Rerror
		r.Ename = errStr
	} else {
		r.Type = tx.Type + 1
	}
	return r
}

func (s *StateServer) handle(tx *Fcall) *Fcall {
	switch tx.Type {
	case Tversion:
		return s.sVersion(tx)
	case Tauth:
		return stRespond(tx, "authentication not required")
	case Tattach:
		return s.sAttach(tx)
	case Tflush:
		return stRespond(tx, "")
	case Twalk:
		return s.sWalk(tx)
	case Topen:
		return s.sOpen(tx)
	case Tcreate:
		return stRespond(tx, "permission denied")
	case Tread:
		return s.sRead(tx)
	case Twrite:
		return s.sWrite(tx)
	case Tclunk:
		return s.sClunk(tx)
	case Tremove:
		return stRespond(tx, "permission denied")
	case Tstat:
		return s.sStat(tx)
	case Twstat:
		return stRespond(tx, "permission denied")
	default:
		return stRespond(tx, "bad fcall type")
	}
}

func (s *StateServer) sVersion(tx *Fcall) *Fcall {
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

func (s *StateServer) sAttach(tx *Fcall) *Fcall {
	f := s.newFid(tx.Fid)
	f.busy = true
	f.qid = Qid{Type: QTDIR, Path: qRoot}
	r := &Fcall{Type: Rattach, Tag: tx.Tag, Qid: f.qid}
	return r
}

func (s *StateServer) sWalk(tx *Fcall) *Fcall {
	f := s.lookFid(tx.Fid)
	if f == nil || !f.busy {
		return stRespond(tx, "fid not in use")
	}

	var nf *stFid
	if tx.Fid != tx.Newfid {
		nf = s.newFid(tx.Newfid)
		nf.busy = true
		nf.qid = f.qid
		nf.path = f.path
		f = nf
	}

	r := &Fcall{Type: Rwalk, Tag: tx.Tag}
	q := f.qid
	path := f.path

	for _, name := range tx.Wname {
		if q.Type&QTDIR == 0 {
			if nf != nil {
				nf.busy = false
			}
			return stRespond(tx, "not a directory")
		}

		if name == ".." {
			q = Qid{Type: QTDIR, Path: qRoot}
			path = ""
			r.Wqid = append(r.Wqid, q)
			continue
		}

		switch q.Path {
		case qRoot:
			switch name {
			case "tree":
				q = Qid{Type: QTFILE, Path: qTree}
				path = ""
			case "actions":
				q = Qid{Type: QTFILE, Path: qActions}
				path = ""
			case "focus":
				q = Qid{Type: QTFILE, Path: qFocus}
				path = ""
			case "state":
				q = Qid{Type: QTDIR, Path: qStateD}
				path = ""
			case "body":
				q = Qid{Type: QTDIR, Path: qBodyD}
				path = ""
			case "tag":
				q = Qid{Type: QTDIR, Path: qTagD}
				path = ""
			default:
				if nf != nil && len(r.Wqid) == 0 {
					nf.busy = false
				}
				if len(r.Wqid) == 0 {
					return stRespond(tx, "file does not exist")
				}
				goto done
			}

		case qStateD:
			// Walking into /state/<key>
			q = Qid{Type: QTFILE, Path: qStateBase}
			path = name

		case qBodyD:
			q = Qid{Type: QTFILE, Path: qBodyBase}
			path = name

		case qTagD:
			q = Qid{Type: QTFILE, Path: qTagBase}
			path = name

		default:
			if nf != nil && len(r.Wqid) == 0 {
				nf.busy = false
			}
			if len(r.Wqid) == 0 {
				return stRespond(tx, "file does not exist")
			}
			goto done
		}
		r.Wqid = append(r.Wqid, q)
	}

done:
	if len(r.Wqid) == len(tx.Wname) {
		f.qid = q
		f.path = path
	}
	return r
}

func (s *StateServer) sOpen(tx *Fcall) *Fcall {
	f := s.lookFid(tx.Fid)
	if f == nil || !f.busy {
		return stRespond(tx, "fid not in use")
	}
	f.open = true
	r := &Fcall{Type: Ropen, Tag: tx.Tag, Qid: f.qid, Iounit: s.msize - IOHDRSZ}
	return r
}

func (s *StateServer) sRead(tx *Fcall) *Fcall {
	f := s.lookFid(tx.Fid)
	if f == nil || !f.busy {
		return stRespond(tx, "fid not in use")
	}

	r := &Fcall{Type: Rread, Tag: tx.Tag}

	// Directory reads
	if f.qid.Type&QTDIR != 0 {
		r.Data = s.readDir(f, tx.Offset, tx.Count)
		return r
	}

	var data []byte
	switch f.qid.Path {
	case qTree:
		data = []byte(s.prov.TreeText())
	case qActions:
		data = nil // actions is write-only
	case qFocus:
		data = []byte(s.prov.GetFocus() + "\n")
	case qStateBase:
		data = []byte(s.prov.GetState(f.path))
	case qBodyBase:
		data = []byte(s.prov.BodyText(f.path))
	case qTagBase:
		data = []byte(s.prov.TagText(f.path))
	}

	r.Data = stSliceRead(data, tx.Offset, tx.Count)
	return r
}

func (s *StateServer) sWrite(tx *Fcall) *Fcall {
	f := s.lookFid(tx.Fid)
	if f == nil || !f.busy {
		return stRespond(tx, "fid not in use")
	}

	r := &Fcall{Type: Rwrite, Tag: tx.Tag, Count: tx.Count}

	switch f.qid.Path {
	case qActions:
		line := strings.TrimRight(string(tx.Data), "\n")
		if err := s.prov.ProcessAction(line); err != nil {
			return stRespond(tx, err.Error())
		}
	case qFocus:
		id := strings.TrimRight(string(tx.Data), "\n")
		s.prov.SetFocus(id)
	case qStateBase:
		s.prov.SetState(f.path, string(tx.Data))
	case qBodyBase:
		s.prov.SetBodyText(f.path, string(tx.Data))
	default:
		return stRespond(tx, "write not allowed")
	}

	return r
}

func (s *StateServer) sClunk(tx *Fcall) *Fcall {
	s.delFid(tx.Fid)
	return &Fcall{Type: Rclunk, Tag: tx.Tag}
}

func (s *StateServer) sStat(tx *Fcall) *Fcall {
	f := s.lookFid(tx.Fid)
	if f == nil || !f.busy {
		return stRespond(tx, "fid not in use")
	}

	var name string
	var perm uint32
	var qtyp uint8

	switch f.qid.Path {
	case qRoot:
		name = "."
		qtyp = QTDIR
		perm = DMDIR | 0500
	case qTree:
		name = "tree"
		perm = 0400
	case qActions:
		name = "actions"
		perm = 0200
	case qFocus:
		name = "focus"
		perm = 0600
	case qStateD:
		name = "state"
		qtyp = QTDIR
		perm = DMDIR | 0700
	case qBodyD:
		name = "body"
		qtyp = QTDIR
		perm = DMDIR | 0700
	case qTagD:
		name = "tag"
		qtyp = QTDIR
		perm = DMDIR | 0500
	case qStateBase:
		name = f.path
		perm = 0600
	case qBodyBase:
		name = f.path
		perm = 0600
	case qTagBase:
		name = f.path
		perm = 0400
	}

	stat := stMakeStat(name, qtyp, f.qid.Path, perm)
	r := &Fcall{Type: Rstat, Tag: tx.Tag, Stat: stat}
	return r
}

// readDir generates directory listing data for given directory fid.
func (s *StateServer) readDir(f *stFid, offset uint64, count uint32) []byte {
	type entry struct {
		name string
		qtyp uint8
		path uint64
		perm uint32
	}

	var entries []entry

	switch f.qid.Path {
	case qRoot:
		entries = []entry{
			{"tree", QTFILE, qTree, 0400},
			{"actions", QTFILE, qActions, 0200},
			{"focus", QTFILE, qFocus, 0600},
			{"state", QTDIR, qStateD, DMDIR | 0700},
			{"body", QTDIR, qBodyD, DMDIR | 0700},
			{"tag", QTDIR, qTagD, DMDIR | 0500},
		}

	case qStateD:
		keys := s.prov.ListState("")
		sort.Strings(keys)
		for _, k := range keys {
			entries = append(entries, entry{k, QTFILE, qStateBase, 0600})
		}

	case qBodyD:
		ids := s.prov.BodyIDs()
		sort.Strings(ids)
		for _, id := range ids {
			entries = append(entries, entry{id, QTFILE, qBodyBase, 0600})
		}

	case qTagD:
		ids := s.prov.TagIDs()
		sort.Strings(ids)
		for _, id := range ids {
			entries = append(entries, entry{id, QTFILE, qTagBase, 0400})
		}
	}

	var buf []byte
	for _, e := range entries {
		stat := stMakeStat(e.name, e.qtyp, e.path, e.perm)
		buf = append(buf, stat...)
	}

	return stSliceRead(buf, offset, count)
}

func stMakeStat(name string, qtyp uint8, qpath uint64, perm uint32) []byte {
	uid := "ui"
	gid := "ui"
	muid := "ui"
	now := uint32(time.Now().Unix())

	fixedLen := 2 + 4 + 13 + 4 + 4 + 4 + 8 + 2 + 2 + 2 + 2
	strLen := len(name) + len(uid) + len(gid) + len(muid)
	statLen := fixedLen + strLen

	buf := make([]byte, 2+statLen)
	off := 0

	binary.LittleEndian.PutUint16(buf[off:], uint16(statLen))
	off += 2
	binary.LittleEndian.PutUint16(buf[off:], 0) // type
	off += 2
	binary.LittleEndian.PutUint32(buf[off:], 0) // dev
	off += 4

	buf[off] = qtyp
	binary.LittleEndian.PutUint32(buf[off+1:], 0)
	binary.LittleEndian.PutUint64(buf[off+5:], qpath)
	off += 13

	binary.LittleEndian.PutUint32(buf[off:], perm)
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], now)
	off += 4
	binary.LittleEndian.PutUint32(buf[off:], now)
	off += 4
	binary.LittleEndian.PutUint64(buf[off:], 0) // length
	off += 8

	off = stPstring(buf, off, name)
	off = stPstring(buf, off, uid)
	off = stPstring(buf, off, gid)
	off = stPstring(buf, off, muid)

	return buf[:off]
}

func stPstring(buf []byte, off int, s string) int {
	binary.LittleEndian.PutUint16(buf[off:], uint16(len(s)))
	off += 2
	copy(buf[off:], s)
	return off + len(s)
}

func stSliceRead(data []byte, offset uint64, count uint32) []byte {
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
