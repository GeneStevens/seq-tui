package app

import (
	"context"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/roster"
	"github.com/GeneStevens/seq-tui/screen"
)

// picked answers a character chosen on the picker: err is nil when it is
// playing.
type picked struct {
	id  string
	err error
}

// NewPicker returns the program's model for a session that is connected and has
// not chosen a character, with characters to choose from (the roster's): it
// opens on the picker, the last played first and chosen. An empty roster has
// nothing to pick; use NewCreator.
func NewPicker(sess *client.Session, ros *roster.Roster, path string) *App {
	a := New(sess)
	a.roster, a.rosterPath = ros, path
	p := &screen.Picker{}
	for _, e := range ros.List() {
		p.Rows = append(p.Rows, screen.PickerRow{ID: e.ID, Name: e.Name, Class: e.Class, Level: e.Level, Zone: e.Zone, Played: e.Played})
	}
	a.picker = p
	return a
}

// SetRoster gives the app the roster it notes the characters it plays in, and
// the file it keeps it in (a roster is only written once a character is
// playing).
func (a *App) SetRoster(r *roster.Roster, path string) { a.roster, a.rosterPath = r, path }

// NoteRoster records the character now playing (its class, level and zone as
// the model has them) in the roster file; it does nothing before a character is
// chosen or without a roster.
func (a *App) NoteRoster() {
	if a.roster == nil || a.rosterPath == "" {
		return
	}
	var e roster.Entry
	a.sess.View(func(m *model.Model) {
		if !m.Ready() || m.Self.ID == "" {
			return
		}
		class := m.Self.Class
		if c := m.Class(class); c != nil {
			class = c.GetName()
		}
		zone := m.Space
		if m.Map != nil && m.Map.GetName() != "" {
			zone = m.Map.GetName()
		}
		e = roster.Entry{ID: m.Self.ID, Name: m.Self.Name, Class: class, Level: m.Self.Level, Zone: zone, Played: a.now()}
	})
	if e.ID == "" {
		return
	}
	a.roster.Note(e)
	if err := a.roster.Save(a.rosterPath); err != nil {
		a.view.Notice, a.view.NoticeBad = "could not save the character list: "+err.Error(), true
	}
}

// pickerKey handles a key on the picker.
func (a *App) pickerKey(k string) Action {
	p := a.picker
	if k == "ctrl+c" {
		return Action{Kind: Quit}
	}
	if p.Busy {
		return Action{}
	}
	p.Err = ""
	switch k {
	case "ctrl+l":
		return Action{Kind: OpenSessions}
	case "up", "w", "k":
		p.Move(-1)
	case "down", "s", "j":
		p.Move(1)
	case "n":
		p.Sel = len(p.Rows)
		return a.pickerEnter()
	case "enter", "space", " ":
		return a.pickerEnter()
	default:
		if len(k) == 1 && k[0] >= '1' && k[0] <= '9' {
			if i := int(k[0] - '1'); i < len(p.Rows) {
				p.Sel = i
			}
		}
	}
	return Action{}
}

func (a *App) pickerEnter() Action {
	p := a.picker
	if c := p.Chosen(); c != nil {
		return Action{Kind: PickCharacter, ID: c.ID}
	}
	a.back, a.picker = p, nil
	a.create = &screen.Create{}
	return Action{}
}

// pickCmd chooses the character on the server: the first snapshot is in the
// model when it returns.
func (a *App) pickCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*commandTimeout)
		defer cancel()
		return picked{id, a.sess.Select(ctx, id)}
	}
}

// refusalMessage finds the reason in a refusal's text (select refused:
// code:ERROR_CODE_SELECT_REFUSED message:"not_permitted").
var refusalMessage = regexp.MustCompile(`message:"([^"]*)"`)

// pickedMsg takes the answer to a pick: the picker closes on success; on a
// refusal it says why, and drops a character the server does not know (the
// world was reset under the list).
func (a *App) pickedMsg(msg picked) {
	a.inflight--
	p := a.picker
	if msg.err == nil {
		a.picker = nil
		a.view.Notice, a.view.NoticeBad = "", false
		a.NoteRoster()
		return
	}
	if p == nil {
		return
	}
	p.Busy = false
	text, code := msg.err.Error(), ""
	if m := refusalMessage.FindStringSubmatch(text); m != nil {
		code = m[1]
		text = model.Words(code)
	}
	p.Err = text
	if a.roster != nil && (code == "not_permitted" || strings.Contains(code, "unknown")) {
		a.roster.Forget(msg.id)
		if a.rosterPath != "" {
			_ = a.roster.Save(a.rosterPath)
		}
		for i, r := range p.Rows {
			if r.ID == msg.id {
				p.Rows = append(p.Rows[:i], p.Rows[i+1:]...)
				p.Sel = min(p.Sel, len(p.Rows))
				p.Err = "that character is not in this world (it may have been reset); it is off the list"
				break
			}
		}
	}
}
