// Synfs is a very simple synthetic 9P2000 file server.
//
// It serves a root directory containing a single read-only file called
// "hello" whose contents are "hello, world\n".
//
// Usage:
//
//	synfs [-a addr]
//
// The default listen address is localhost:5640. Connect with any 9P
// client, for example:
//
//	9p -a localhost:5640 read hello
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"9fans.net/go/plan9"
)

var addr = flag.String("a", ":5640", "listen address")

// fileContent is what the synthetic file "hello" contains.
var fileContent = []byte("hello, world\n")

// Qid paths for our two nodes.
const (
	qidRoot  = 0
	qidHello = 1
)

// Pre-built Qids.
var (
	rootQid  = plan9.Qid{Path: qidRoot, Vers: 0, Type: plan9.QTDIR}
	helloQid = plan9.Qid{Path: qidHello, Vers: 0, Type: plan9.QTFILE}
)

// now returns a fixed timestamp for directory entries.
func now() uint32 { return uint32(time.Now().Unix()) }

// dirBytes marshals a Dir into the wire format used in Rstat and Rread
// of directories (the stat(5) encoding with the leading 2-byte size).
func dirBytes(d *plan9.Dir) []byte {
	b, _ := d.Bytes()
	return b
}

// rootDir returns the Dir for "/".
func rootDir() *plan9.Dir {
	return &plan9.Dir{
		Qid:   rootQid,
		Mode:  plan9.Perm(plan9.DMDIR | 0555),
		Atime: now(),
		Mtime: now(),
		Name:  "/",
		Uid:   "none",
		Gid:   "none",
		Muid:  "none",
	}
}

// helloDir returns the Dir for "hello".
func helloDir() *plan9.Dir {
	return &plan9.Dir{
		Qid:    helloQid,
		Mode:   0444,
		Atime:  now(),
		Mtime:  now(),
		Length: uint64(len(fileContent)),
		Name:   "hello",
		Uid:    "none",
		Gid:    "none",
		Muid:   "none",
	}
}

// fidState tracks the server-side state of a fid.
type fidState struct {
	qid plan9.Qid
}

// conn handles a single 9P connection.
type conn struct {
	rwc   io.ReadWriteCloser
	msize uint32

	mu   sync.Mutex
	fids map[uint32]*fidState
}

func newConn(rwc io.ReadWriteCloser) *conn {
	return &conn{
		rwc:  rwc,
		fids: make(map[uint32]*fidState),
	}
}

func (c *conn) getFid(fid uint32) *fidState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fids[fid]
}

func (c *conn) setFid(fid uint32, f *fidState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fids[fid] = f
}

func (c *conn) delFid(fid uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.fids, fid)
}

func (c *conn) serve() {
	defer c.rwc.Close()
	for {
		tx, err := plan9.ReadFcall(c.rwc)
		if err != nil {
			if err != io.EOF {
				log.Printf("read fcall: %v", err)
			}
			return
		}
		rx := c.handle(tx)
		rx.Tag = tx.Tag
		if err := plan9.WriteFcall(c.rwc, rx); err != nil {
			log.Printf("write fcall: %v", err)
			return
		}
	}
}

func (c *conn) handle(tx *plan9.Fcall) *plan9.Fcall {
	switch tx.Type {
	case plan9.Tversion:
		return c.tversion(tx)
	case plan9.Tauth:
		return rerror("authentication not required")
	case plan9.Tattach:
		return c.tattach(tx)
	case plan9.Tflush:
		return &plan9.Fcall{Type: plan9.Rflush}
	case plan9.Twalk:
		return c.twalk(tx)
	case plan9.Topen:
		return c.topen(tx)
	case plan9.Tcreate:
		return rerror("create prohibited")
	case plan9.Tread:
		return c.tread(tx)
	case plan9.Twrite:
		return rerror("write prohibited")
	case plan9.Tclunk:
		return c.tclunk(tx)
	case plan9.Tremove:
		return rerror("remove prohibited")
	case plan9.Tstat:
		return c.tstat(tx)
	case plan9.Twstat:
		return rerror("wstat prohibited")
	default:
		return rerror(fmt.Sprintf("unknown message type %d", tx.Type))
	}
}

func rerror(msg string) *plan9.Fcall {
	return &plan9.Fcall{Type: plan9.Rerror, Ename: msg}
}

func (c *conn) tversion(tx *plan9.Fcall) *plan9.Fcall {
	c.msize = tx.Msize
	if c.msize > 65536 {
		c.msize = 65536
	}
	return &plan9.Fcall{
		Type:    plan9.Rversion,
		Msize:   c.msize,
		Version: plan9.VERSION9P,
	}
}

func (c *conn) tattach(tx *plan9.Fcall) *plan9.Fcall {
	c.setFid(tx.Fid, &fidState{qid: rootQid})
	return &plan9.Fcall{
		Type: plan9.Rattach,
		Qid:  rootQid,
	}
}

func (c *conn) twalk(tx *plan9.Fcall) *plan9.Fcall {
	f := c.getFid(tx.Fid)
	if f == nil {
		return rerror("unknown fid")
	}

	cur := f.qid
	wqid := make([]plan9.Qid, 0, len(tx.Wname))

	for _, name := range tx.Wname {
		if cur.Type&plan9.QTDIR == 0 {
			break // can't walk into a file
		}
		switch {
		case cur.Path == qidRoot && name == "hello":
			cur = helloQid
		case name == "..":
			cur = rootQid
		default:
			// Name not found — stop walking.
			if len(wqid) == 0 {
				return rerror("file not found")
			}
			goto done
		}
		wqid = append(wqid, cur)
	}
done:
	// If the full walk succeeded (or wname was empty), assign newfid.
	if len(wqid) == len(tx.Wname) {
		c.setFid(tx.Newfid, &fidState{qid: cur})
	}
	return &plan9.Fcall{
		Type: plan9.Rwalk,
		Wqid: wqid,
	}
}

func (c *conn) topen(tx *plan9.Fcall) *plan9.Fcall {
	f := c.getFid(tx.Fid)
	if f == nil {
		return rerror("unknown fid")
	}
	return &plan9.Fcall{
		Type:   plan9.Ropen,
		Qid:    f.qid,
		Iounit: c.msize - plan9.IOHDRSIZE,
	}
}

func (c *conn) tread(tx *plan9.Fcall) *plan9.Fcall {
	f := c.getFid(tx.Fid)
	if f == nil {
		return rerror("unknown fid")
	}

	var data []byte
	switch f.qid.Path {
	case qidRoot:
		// Reading a directory: return the encoded Dir for "hello".
		all := dirBytes(helloDir())
		if tx.Offset >= uint64(len(all)) {
			data = nil
		} else {
			data = all[tx.Offset:]
		}
	case qidHello:
		if tx.Offset >= uint64(len(fileContent)) {
			data = nil
		} else {
			data = fileContent[tx.Offset:]
		}
	default:
		return rerror("unknown qid")
	}

	if uint32(len(data)) > tx.Count {
		data = data[:tx.Count]
	}
	return &plan9.Fcall{
		Type: plan9.Rread,
		Data: data,
	}
}

func (c *conn) tclunk(tx *plan9.Fcall) *plan9.Fcall {
	c.delFid(tx.Fid)
	return &plan9.Fcall{Type: plan9.Rclunk}
}

func (c *conn) tstat(tx *plan9.Fcall) *plan9.Fcall {
	f := c.getFid(tx.Fid)
	if f == nil {
		return rerror("unknown fid")
	}

	var d *plan9.Dir
	switch f.qid.Path {
	case qidRoot:
		d = rootDir()
	case qidHello:
		d = helloDir()
	default:
		return rerror("unknown qid")
	}

	return &plan9.Fcall{
		Type: plan9.Rstat,
		Stat: dirBytes(d),
	}
}

func main() {
	flag.Parse()
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("synfs: listening on %s", *addr)
	for {
		nc, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go newConn(nc).serve()
	}
}
