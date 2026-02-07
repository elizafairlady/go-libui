package fsys

import (
	"strings"
	"testing"
)

// mockProvider implements StateProvider for testing.
type mockProvider struct {
	state  map[string]string
	focus  string
	bodies map[string]string
	tags   map[string]string
	tree   string
	acts   []string
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		state:  make(map[string]string),
		bodies: make(map[string]string),
		tags:   make(map[string]string),
	}
}

func (m *mockProvider) GetState(path string) string { return m.state[path] }
func (m *mockProvider) SetState(path, value string) { m.state[path] = value }
func (m *mockProvider) ListState(dir string) []string {
	var keys []string
	for k := range m.state {
		keys = append(keys, k)
	}
	return keys
}
func (m *mockProvider) TreeText() string { return m.tree }
func (m *mockProvider) ProcessAction(line string) error {
	m.acts = append(m.acts, line)
	return nil
}
func (m *mockProvider) GetFocus() string          { return m.focus }
func (m *mockProvider) SetFocus(id string)        { m.focus = id }
func (m *mockProvider) BodyText(id string) string { return m.bodies[id] }
func (m *mockProvider) SetBodyText(id, t string)  { m.bodies[id] = t }
func (m *mockProvider) BodyIDs() []string {
	var ids []string
	for k := range m.bodies {
		ids = append(ids, k)
	}
	return ids
}
func (m *mockProvider) TagText(id string) string { return m.tags[id] }
func (m *mockProvider) TagIDs() []string {
	var ids []string
	for k := range m.tags {
		ids = append(ids, k)
	}
	return ids
}

func TestStateServerVersion(t *testing.T) {
	prov := newMockProvider()
	s := NewStateServer(prov)

	ver := &Fcall{Type: Tversion, Tag: NOTAG, Msize: 8192, Version: "9P2000"}
	r := s.handle(ver)
	if r.Type != Rversion {
		t.Fatalf("got type %d, want Rversion", r.Type)
	}
	if r.Version != "9P2000" {
		t.Fatalf("version = %q", r.Version)
	}
}

func TestStateServerWalkReadState(t *testing.T) {
	prov := newMockProvider()
	prov.state["count"] = "42"
	s := NewStateServer(prov)

	// Attach
	s.handle(&Fcall{Type: Tattach, Tag: 1, Fid: 0})

	// Walk to /state/count
	walk := &Fcall{Type: Twalk, Tag: 2, Fid: 0, Newfid: 1, Wname: []string{"state", "count"}}
	wr := s.handle(walk)
	if wr.Type == Rerror {
		t.Fatalf("walk: %s", wr.Ename)
	}
	if len(wr.Wqid) != 2 {
		t.Fatalf("wqid len = %d, want 2", len(wr.Wqid))
	}

	// Open
	s.handle(&Fcall{Type: Topen, Tag: 3, Fid: 1})

	// Read
	rd := s.handle(&Fcall{Type: Tread, Tag: 4, Fid: 1, Offset: 0, Count: 4096})
	if rd.Type == Rerror {
		t.Fatalf("read: %s", rd.Ename)
	}
	if string(rd.Data) != "42" {
		t.Fatalf("data = %q, want %q", string(rd.Data), "42")
	}
}

func TestStateServerWriteState(t *testing.T) {
	prov := newMockProvider()
	s := NewStateServer(prov)

	s.handle(&Fcall{Type: Tattach, Tag: 1, Fid: 0})

	// Walk to /state/name
	s.handle(&Fcall{Type: Twalk, Tag: 2, Fid: 0, Newfid: 1, Wname: []string{"state", "name"}})
	s.handle(&Fcall{Type: Topen, Tag: 3, Fid: 1})

	// Write
	data := []byte("Ada")
	wr := s.handle(&Fcall{Type: Twrite, Tag: 4, Fid: 1, Data: data, Count: uint32(len(data))})
	if wr.Type == Rerror {
		t.Fatalf("write: %s", wr.Ename)
	}

	if prov.state["name"] != "Ada" {
		t.Fatalf("state[name] = %q", prov.state["name"])
	}
}

func TestStateServerReadTree(t *testing.T) {
	prov := newMockProvider()
	prov.tree = "rev 1\nroot root\n"
	s := NewStateServer(prov)

	s.handle(&Fcall{Type: Tattach, Tag: 1, Fid: 0})
	s.handle(&Fcall{Type: Twalk, Tag: 2, Fid: 0, Newfid: 1, Wname: []string{"tree"}})
	s.handle(&Fcall{Type: Topen, Tag: 3, Fid: 1})

	rd := s.handle(&Fcall{Type: Tread, Tag: 4, Fid: 1, Offset: 0, Count: 4096})
	if rd.Type == Rerror {
		t.Fatalf("read: %s", rd.Ename)
	}
	if !strings.Contains(string(rd.Data), "rev 1") {
		t.Fatalf("tree data = %q", string(rd.Data))
	}
}

func TestStateServerWriteAction(t *testing.T) {
	prov := newMockProvider()
	s := NewStateServer(prov)

	s.handle(&Fcall{Type: Tattach, Tag: 1, Fid: 0})
	s.handle(&Fcall{Type: Twalk, Tag: 2, Fid: 0, Newfid: 1, Wname: []string{"actions"}})
	s.handle(&Fcall{Type: Topen, Tag: 3, Fid: 1})

	line := []byte("click id=btn button=1\n")
	wr := s.handle(&Fcall{Type: Twrite, Tag: 4, Fid: 1, Data: line, Count: uint32(len(line))})
	if wr.Type == Rerror {
		t.Fatalf("write: %s", wr.Ename)
	}

	if len(prov.acts) != 1 || prov.acts[0] != "click id=btn button=1" {
		t.Fatalf("acts = %v", prov.acts)
	}
}

func TestStateServerReadWriteFocus(t *testing.T) {
	prov := newMockProvider()
	prov.focus = "textbox1"
	s := NewStateServer(prov)

	s.handle(&Fcall{Type: Tattach, Tag: 1, Fid: 0})
	s.handle(&Fcall{Type: Twalk, Tag: 2, Fid: 0, Newfid: 1, Wname: []string{"focus"}})
	s.handle(&Fcall{Type: Topen, Tag: 3, Fid: 1})

	// Read
	rd := s.handle(&Fcall{Type: Tread, Tag: 4, Fid: 1, Offset: 0, Count: 4096})
	if string(rd.Data) != "textbox1\n" {
		t.Fatalf("focus = %q", string(rd.Data))
	}

	// Write new focus
	data := []byte("btn2\n")
	s.handle(&Fcall{Type: Twrite, Tag: 5, Fid: 1, Data: data, Count: uint32(len(data))})
	if prov.focus != "btn2" {
		t.Fatalf("focus = %q after write", prov.focus)
	}
}

func TestStateServerReadBody(t *testing.T) {
	prov := newMockProvider()
	prov.bodies["wb-0-1"] = "hello world"
	s := NewStateServer(prov)

	s.handle(&Fcall{Type: Tattach, Tag: 1, Fid: 0})
	s.handle(&Fcall{Type: Twalk, Tag: 2, Fid: 0, Newfid: 1, Wname: []string{"body", "wb-0-1"}})
	s.handle(&Fcall{Type: Topen, Tag: 3, Fid: 1})

	rd := s.handle(&Fcall{Type: Tread, Tag: 4, Fid: 1, Offset: 0, Count: 4096})
	if string(rd.Data) != "hello world" {
		t.Fatalf("body = %q", string(rd.Data))
	}
}

func TestStateServerWriteBody(t *testing.T) {
	prov := newMockProvider()
	s := NewStateServer(prov)

	s.handle(&Fcall{Type: Tattach, Tag: 1, Fid: 0})
	s.handle(&Fcall{Type: Twalk, Tag: 2, Fid: 0, Newfid: 1, Wname: []string{"body", "wb-0-1"}})
	s.handle(&Fcall{Type: Topen, Tag: 3, Fid: 1})

	data := []byte("new content")
	s.handle(&Fcall{Type: Twrite, Tag: 4, Fid: 1, Data: data, Count: uint32(len(data))})
	if prov.bodies["wb-0-1"] != "new content" {
		t.Fatalf("body = %q", prov.bodies["wb-0-1"])
	}
}

func TestStateServerReadTag(t *testing.T) {
	prov := newMockProvider()
	prov.tags["tag-1"] = "File Edit Del"
	s := NewStateServer(prov)

	s.handle(&Fcall{Type: Tattach, Tag: 1, Fid: 0})
	s.handle(&Fcall{Type: Twalk, Tag: 2, Fid: 0, Newfid: 1, Wname: []string{"tag", "tag-1"}})
	s.handle(&Fcall{Type: Topen, Tag: 3, Fid: 1})

	rd := s.handle(&Fcall{Type: Tread, Tag: 4, Fid: 1, Offset: 0, Count: 4096})
	if string(rd.Data) != "File Edit Del" {
		t.Fatalf("tag = %q", string(rd.Data))
	}
}
