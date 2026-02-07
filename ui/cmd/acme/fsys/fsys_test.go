package acmefsys

import (
	"bytes"
	"strings"
	"testing"

	p9 "github.com/elizafairlady/go-libui/ui/fsys"

	"github.com/elizafairlady/go-libui/ui/cmd/acme/window"
)

// pipeConn is an in-memory bidirectional pipe for testing.
type pipeConn struct {
	r *bytes.Buffer
	w *bytes.Buffer
}

func (p *pipeConn) Read(b []byte) (int, error)  { return p.r.Read(b) }
func (p *pipeConn) Write(b []byte) (int, error) { return p.w.Write(b) }
func (p *pipeConn) Close() error                { return nil }

func TestVersionAttach(t *testing.T) {
	row := window.NewRow()
	s := NewServer(row)

	var req bytes.Buffer
	var resp bytes.Buffer
	conn := &pipeConn{r: &req, w: &resp}

	// Send Tversion
	ver := &p9.Fcall{Type: p9.Tversion, Tag: p9.NOTAG, Msize: 8192, Version: "9P2000"}
	p9.WriteFcall(&req, ver)

	fc, _ := p9.ReadFcall(&req)
	r := s.handle(fc)
	p9.WriteFcall(conn, r)

	rr, _ := p9.ReadFcall(&resp)
	if rr.Type != p9.Rversion {
		t.Fatalf("got type %d, want Rversion", rr.Type)
	}
	if rr.Version != "9P2000" {
		t.Fatalf("version = %q", rr.Version)
	}

	// Send Tattach
	resp.Reset()
	att := &p9.Fcall{Type: p9.Tattach, Tag: 1, Fid: 0}
	r = s.handle(att)
	p9.WriteFcall(conn, r)

	rr, _ = p9.ReadFcall(&resp)
	if rr.Type != p9.Rattach {
		t.Fatalf("got type %d, want Rattach", rr.Type)
	}
}

func TestWalkAndReadBody(t *testing.T) {
	row := window.NewRow()
	col := row.NewColumn()
	w := row.NewWindow(col)
	w.Body.SetAll("hello from body")
	w.Tag.SetAll("test.txt Del")

	s := NewServer(row)

	// Attach
	att := &p9.Fcall{Type: p9.Tattach, Tag: 1, Fid: 0}
	s.handle(att)

	// Walk to /<id>/body
	walk := &p9.Fcall{
		Type:   p9.Twalk,
		Tag:    2,
		Fid:    0,
		Newfid: 1,
		Wname:  []string{strings.TrimSpace(string(rune('0' + w.ID))), "body"},
	}
	// Use proper ID string
	walk.Wname[0] = "1"
	wr := s.handle(walk)
	if wr.Type == p9.Rerror {
		t.Fatalf("walk error: %s", wr.Ename)
	}
	if len(wr.Wqid) != 2 {
		t.Fatalf("wqid len = %d, want 2", len(wr.Wqid))
	}

	// Open
	open := &p9.Fcall{Type: p9.Topen, Tag: 3, Fid: 1}
	or := s.handle(open)
	if or.Type == p9.Rerror {
		t.Fatalf("open error: %s", or.Ename)
	}

	// Read
	rd := &p9.Fcall{Type: p9.Tread, Tag: 4, Fid: 1, Offset: 0, Count: 4096}
	rr := s.handle(rd)
	if rr.Type == p9.Rerror {
		t.Fatalf("read error: %s", rr.Ename)
	}
	if string(rr.Data) != "hello from body" {
		t.Fatalf("data = %q", string(rr.Data))
	}
}

func TestWalkNewWindow(t *testing.T) {
	row := window.NewRow()
	s := NewServer(row)

	// Attach
	att := &p9.Fcall{Type: p9.Tattach, Tag: 1, Fid: 0}
	s.handle(att)

	// Walk to /new/ctl
	walk := &p9.Fcall{
		Type:   p9.Twalk,
		Tag:    2,
		Fid:    0,
		Newfid: 1,
		Wname:  []string{"new", "ctl"},
	}
	wr := s.handle(walk)
	if wr.Type == p9.Rerror {
		t.Fatalf("walk error: %s", wr.Ename)
	}

	// Should have created a window
	if len(row.Windows) != 1 {
		t.Fatalf("windows = %d, want 1", len(row.Windows))
	}
}

func TestWriteBody(t *testing.T) {
	row := window.NewRow()
	col := row.NewColumn()
	w := row.NewWindow(col)

	s := NewServer(row)

	// Attach
	s.handle(&p9.Fcall{Type: p9.Tattach, Tag: 1, Fid: 0})

	// Walk to /1/body
	walk := &p9.Fcall{Type: p9.Twalk, Tag: 2, Fid: 0, Newfid: 1, Wname: []string{"1", "body"}}
	wr := s.handle(walk)
	if wr.Type == p9.Rerror {
		t.Fatalf("walk: %s", wr.Ename)
	}

	// Open
	s.handle(&p9.Fcall{Type: p9.Topen, Tag: 3, Fid: 1})

	// Write
	data := []byte("hello world")
	ww := s.handle(&p9.Fcall{Type: p9.Twrite, Tag: 4, Fid: 1, Data: data, Count: uint32(len(data))})
	if ww.Type == p9.Rerror {
		t.Fatalf("write: %s", ww.Ename)
	}

	if w.Body.ReadAll() != "hello world" {
		t.Fatalf("body = %q", w.Body.ReadAll())
	}
}

func TestWriteCtl(t *testing.T) {
	row := window.NewRow()
	col := row.NewColumn()
	w := row.NewWindow(col)

	s := NewServer(row)
	s.handle(&p9.Fcall{Type: p9.Tattach, Tag: 1, Fid: 0})

	// Walk to /1/ctl
	walk := &p9.Fcall{Type: p9.Twalk, Tag: 2, Fid: 0, Newfid: 1, Wname: []string{"1", "ctl"}}
	s.handle(walk)
	s.handle(&p9.Fcall{Type: p9.Topen, Tag: 3, Fid: 1})

	// Write name command
	data := []byte("name /tmp/test.go\n")
	s.handle(&p9.Fcall{Type: p9.Twrite, Tag: 4, Fid: 1, Data: data, Count: uint32(len(data))})

	if w.Name != "/tmp/test.go" {
		t.Fatalf("name = %q", w.Name)
	}
}

func TestReadIndex(t *testing.T) {
	row := window.NewRow()
	col := row.NewColumn()
	w := row.NewWindow(col)
	w.Tag.SetAll("test.go Del")
	w.Body.SetAll("package main")

	s := NewServer(row)
	s.handle(&p9.Fcall{Type: p9.Tattach, Tag: 1, Fid: 0})

	// Walk to /index
	s.handle(&p9.Fcall{Type: p9.Twalk, Tag: 2, Fid: 0, Newfid: 1, Wname: []string{"index"}})
	s.handle(&p9.Fcall{Type: p9.Topen, Tag: 3, Fid: 1})

	rr := s.handle(&p9.Fcall{Type: p9.Tread, Tag: 4, Fid: 1, Offset: 0, Count: 4096})
	if rr.Type == p9.Rerror {
		t.Fatalf("read: %s", rr.Ename)
	}
	idx := string(rr.Data)
	if !strings.Contains(idx, "test.go Del") {
		t.Fatalf("index = %q", idx)
	}
}
