package ui

import (
	"fmt"
	"strings"

	"github.com/elizafairlady/go-libui/ui/fsys"
	"github.com/elizafairlady/go-libui/ui/render"
	"github.com/elizafairlady/go-libui/ui/uifs"
)

// uiSrvName returns the /srv name for the 9P state server.
// The name is "ui.<title>" posted to /srv.
func uiSrvName(title string) string {
	// Sanitize title for use as /srv filename
	safe := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, title)
	if safe == "" {
		safe = "app"
	}
	return fmt.Sprintf("ui.%s", strings.ToLower(safe))
}

// stateProvider adapts the UIFS and Renderer into the
// fsys.StateProvider interface for the 9P state server.
type stateProvider struct {
	u *uifs.UIFS
	r *render.Renderer
}

var _ fsys.StateProvider = (*stateProvider)(nil)

func (p *stateProvider) GetState(path string) string {
	return p.u.GetState(path)
}

func (p *stateProvider) SetState(path, value string) {
	p.u.SetState(path, value)
}

func (p *stateProvider) ListState(dir string) []string {
	return p.u.State().Keys()
}

func (p *stateProvider) TreeText() string {
	return p.u.TreeText()
}

func (p *stateProvider) ProcessAction(line string) error {
	return p.u.ProcessAction(line)
}

func (p *stateProvider) GetFocus() string {
	return p.u.Focus
}

func (p *stateProvider) SetFocus(id string) {
	p.u.SetFocus(id)
	p.r.Focus = id
}

func (p *stateProvider) BodyText(id string) string {
	return p.r.BodyText(id)
}

func (p *stateProvider) SetBodyText(id, text string) {
	if bs, ok := p.r.Bodies[id]; ok {
		bs.Buf.SetAll(text)
	}
}

func (p *stateProvider) BodyIDs() []string {
	ids := make([]string, 0, len(p.r.Bodies))
	for id := range p.r.Bodies {
		ids = append(ids, id)
	}
	return ids
}

func (p *stateProvider) TagText(id string) string {
	return p.r.TagText(id)
}

func (p *stateProvider) TagIDs() []string {
	ids := make([]string, 0, len(p.r.Tags))
	for id := range p.r.Tags {
		ids = append(ids, id)
	}
	return ids
}
