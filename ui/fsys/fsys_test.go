package fsys

import (
	"io"
	"strings"
	"testing"

	"github.com/elizafairlady/go-libui/ui/window"
)

// pipePair creates a connected in-memory pipe pair for testing 9P.
func pipePair() (io.ReadWriteCloser, io.ReadWriteCloser) {
	cr, sw := io.Pipe()
	sr, cw := io.Pipe()
	return &pipeRWC{sr, sw}, &pipeRWC{cr, cw}
}

type pipeRWC struct {
	r *io.PipeReader
	w *io.PipeWriter
}

func (p *pipeRWC) Read(b []byte) (int, error)  { return p.r.Read(b) }
func (p *pipeRWC) Write(b []byte) (int, error) { return p.w.Write(b) }
func (p *pipeRWC) Close() error {
	p.r.Close()
	return p.w.Close()
}

// send/recv helpers
func send(t *testing.T, c io.ReadWriteCloser, fc *Fcall) {
	t.Helper()
	if err := WriteFcall(c, fc); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func recv(t *testing.T, c io.ReadWriteCloser) *Fcall {
	t.Helper()
	fc, err := ReadFcall(c)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return fc
}

func sendRecv(t *testing.T, c io.ReadWriteCloser, tx *Fcall) *Fcall {
	t.Helper()
	send(t, c, tx)
	return recv(t, c)
}

func TestVersion(t *testing.T) {
	row := window.NewRow()
	srv := NewServer(row)
	serverSide, clientSide := pipePair()
	go srv.Serve(serverSide)

	r := sendRecv(t, clientSide, &Fcall{Type: Tversion, Tag: 0xFFFF, Msize: 8192, Version: "9P2000"})
	if r.Type != Rversion {
		t.Fatalf("got type %d, want %d (Rversion)", r.Type, Rversion)
	}
	if r.Version != "9P2000" {
		t.Fatalf("version = %q, want 9P2000", r.Version)
	}
	clientSide.Close()
}

func TestAttachAndWalk(t *testing.T) {
	row := window.NewRow()
	col := row.NewColumn()
	w := row.NewWindow(col)
	w.Name = "test.txt"
	w.Body.SetAll("hello from body")
	w.Tag.SetAll("test.txt Del Put")

	srv := NewServer(row)
	serverSide, clientSide := pipePair()
	go srv.Serve(serverSide)
	defer clientSide.Close()

	// Version
	sendRecv(t, clientSide, &Fcall{Type: Tversion, Tag: 0xFFFF, Msize: 8192, Version: "9P2000"})

	// Attach
	r := sendRecv(t, clientSide, &Fcall{Type: Tattach, Tag: 1, Fid: 0, Afid: 0xFFFFFFFF, Uname: "test", Aname: ""})
	if r.Type != Rattach {
		t.Fatalf("got type %d, want Rattach", r.Type)
	}

	// Walk to window 1
	r = sendRecv(t, clientSide, &Fcall{Type: Twalk, Tag: 2, Fid: 0, Newfid: 1, Wname: []string{"1"}})
	if r.Type != Rwalk {
		t.Fatalf("got type %d (%s), want Rwalk", r.Type, errMsg(r))
	}
	if len(r.Wqid) != 1 {
		t.Fatalf("nwqid = %d, want 1", len(r.Wqid))
	}

	// Walk to body
	r = sendRecv(t, clientSide, &Fcall{Type: Twalk, Tag: 3, Fid: 1, Newfid: 2, Wname: []string{"body"}})
	if r.Type != Rwalk {
		t.Fatalf("walk body: type %d (%s), want Rwalk", r.Type, errMsg(r))
	}

	// Open body
	r = sendRecv(t, clientSide, &Fcall{Type: Topen, Tag: 4, Fid: 2, Mode: OREAD})
	if r.Type != Ropen {
		t.Fatalf("open body: type %d (%s), want Ropen", r.Type, errMsg(r))
	}

	// Read body
	r = sendRecv(t, clientSide, &Fcall{Type: Tread, Tag: 5, Fid: 2, Offset: 0, Count: 1024})
	if r.Type != Rread {
		t.Fatalf("read body: type %d (%s), want Rread", r.Type, errMsg(r))
	}
	if string(r.Data) != "hello from body" {
		t.Fatalf("body = %q, want %q", string(r.Data), "hello from body")
	}

	// Clunk body fid
	sendRecv(t, clientSide, &Fcall{Type: Tclunk, Tag: 6, Fid: 2})
}

func TestWriteCtl(t *testing.T) {
	row := window.NewRow()
	col := row.NewColumn()
	w := row.NewWindow(col)
	w.Name = "test.txt"
	w.Body.SetAll("original")

	srv := NewServer(row)
	serverSide, clientSide := pipePair()
	go srv.Serve(serverSide)
	defer clientSide.Close()

	// Version + Attach
	sendRecv(t, clientSide, &Fcall{Type: Tversion, Tag: 0xFFFF, Msize: 8192, Version: "9P2000"})
	sendRecv(t, clientSide, &Fcall{Type: Tattach, Tag: 1, Fid: 0, Afid: 0xFFFFFFFF, Uname: "test", Aname: ""})

	// Walk to 1/ctl
	r := sendRecv(t, clientSide, &Fcall{Type: Twalk, Tag: 2, Fid: 0, Newfid: 1, Wname: []string{"1", "ctl"}})
	if r.Type != Rwalk {
		t.Fatalf("walk: %s", errMsg(r))
	}

	// Open ctl for write
	sendRecv(t, clientSide, &Fcall{Type: Topen, Tag: 3, Fid: 1, Mode: OWRITE})

	// Write "name /tmp/newname.txt\n"
	msg := []byte("name /tmp/newname.txt\n")
	r = sendRecv(t, clientSide, &Fcall{Type: Twrite, Tag: 4, Fid: 1, Offset: 0, Count: uint32(len(msg)), Data: msg})
	if r.Type != Rwrite {
		t.Fatalf("write ctl: %s", errMsg(r))
	}

	if w.Name != "/tmp/newname.txt" {
		t.Fatalf("name = %q, want /tmp/newname.txt", w.Name)
	}

	sendRecv(t, clientSide, &Fcall{Type: Tclunk, Tag: 5, Fid: 1})
}

func TestWriteBody(t *testing.T) {
	row := window.NewRow()
	col := row.NewColumn()
	w := row.NewWindow(col)
	w.Body.SetAll("")

	srv := NewServer(row)
	serverSide, clientSide := pipePair()
	go srv.Serve(serverSide)
	defer clientSide.Close()

	// Version + Attach
	sendRecv(t, clientSide, &Fcall{Type: Tversion, Tag: 0xFFFF, Msize: 8192, Version: "9P2000"})
	sendRecv(t, clientSide, &Fcall{Type: Tattach, Tag: 1, Fid: 0, Afid: 0xFFFFFFFF, Uname: "test", Aname: ""})

	// Walk to 1/body
	sendRecv(t, clientSide, &Fcall{Type: Twalk, Tag: 2, Fid: 0, Newfid: 1, Wname: []string{"1", "body"}})
	sendRecv(t, clientSide, &Fcall{Type: Topen, Tag: 3, Fid: 1, Mode: OWRITE})

	// Write (appends)
	msg := []byte("line 1\n")
	sendRecv(t, clientSide, &Fcall{Type: Twrite, Tag: 4, Fid: 1, Offset: 0, Count: uint32(len(msg)), Data: msg})
	msg = []byte("line 2\n")
	sendRecv(t, clientSide, &Fcall{Type: Twrite, Tag: 5, Fid: 1, Offset: 0, Count: uint32(len(msg)), Data: msg})

	got := w.Body.ReadAll()
	if got != "line 1\nline 2\n" {
		t.Fatalf("body = %q, want %q", got, "line 1\nline 2\n")
	}

	sendRecv(t, clientSide, &Fcall{Type: Tclunk, Tag: 6, Fid: 1})
}

func TestWalkNew(t *testing.T) {
	row := window.NewRow()
	row.NewColumn()

	srv := NewServer(row)
	serverSide, clientSide := pipePair()
	go srv.Serve(serverSide)
	defer clientSide.Close()

	sendRecv(t, clientSide, &Fcall{Type: Tversion, Tag: 0xFFFF, Msize: 8192, Version: "9P2000"})
	sendRecv(t, clientSide, &Fcall{Type: Tattach, Tag: 1, Fid: 0, Afid: 0xFFFFFFFF, Uname: "test", Aname: ""})

	// Walk to new/ctl — should create a window
	r := sendRecv(t, clientSide, &Fcall{Type: Twalk, Tag: 2, Fid: 0, Newfid: 1, Wname: []string{"new", "ctl"}})
	if r.Type != Rwalk {
		t.Fatalf("walk new/ctl: %s", errMsg(r))
	}

	// The new window should exist
	if len(row.Cols[0].Windows) != 1 {
		t.Fatalf("windows = %d, want 1", len(row.Cols[0].Windows))
	}

	// Read ctl to get the ID
	sendRecv(t, clientSide, &Fcall{Type: Topen, Tag: 3, Fid: 1, Mode: OREAD})
	r = sendRecv(t, clientSide, &Fcall{Type: Tread, Tag: 4, Fid: 1, Offset: 0, Count: 1024})
	if r.Type != Rread {
		t.Fatalf("read ctl: %s", errMsg(r))
	}
	// Should start with the window ID
	ctl := string(r.Data)
	if !strings.Contains(ctl, "1") {
		t.Fatalf("ctl = %q, should contain window id", ctl)
	}

	sendRecv(t, clientSide, &Fcall{Type: Tclunk, Tag: 5, Fid: 1})
}

func TestReadIndex(t *testing.T) {
	row := window.NewRow()
	col := row.NewColumn()
	w := row.NewWindow(col)
	w.Tag.SetAll("hello.txt Del Put")
	w.Body.SetAll("contents")

	srv := NewServer(row)
	serverSide, clientSide := pipePair()
	go srv.Serve(serverSide)
	defer clientSide.Close()

	sendRecv(t, clientSide, &Fcall{Type: Tversion, Tag: 0xFFFF, Msize: 8192, Version: "9P2000"})
	sendRecv(t, clientSide, &Fcall{Type: Tattach, Tag: 1, Fid: 0, Afid: 0xFFFFFFFF, Uname: "test", Aname: ""})

	// Walk to index
	sendRecv(t, clientSide, &Fcall{Type: Twalk, Tag: 2, Fid: 0, Newfid: 1, Wname: []string{"index"}})
	sendRecv(t, clientSide, &Fcall{Type: Topen, Tag: 3, Fid: 1, Mode: OREAD})

	r := sendRecv(t, clientSide, &Fcall{Type: Tread, Tag: 4, Fid: 1, Offset: 0, Count: 4096})
	if r.Type != Rread {
		t.Fatalf("read index: %s", errMsg(r))
	}
	idx := string(r.Data)
	if !strings.Contains(idx, "hello.txt Del Put") {
		t.Fatalf("index = %q, should contain tag text", idx)
	}

	sendRecv(t, clientSide, &Fcall{Type: Tclunk, Tag: 5, Fid: 1})
}

func errMsg(fc *Fcall) string {
	if fc.Type == Rerror {
		return fc.Ename
	}
	return ""
}
