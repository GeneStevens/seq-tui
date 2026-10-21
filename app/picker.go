package app

import (
	"context"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	pb "github.com/GeneStevens/seq/protocol/seqv1"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
)

// picked answers a character chosen on the picker: err is nil when it is
// playing.
type picked struct {
	id  string
	err error
}

// className is a class id as a name for the table: the server lists the id
// ("warrior"), and the class definitions come only with a snapshot.
func className(id string) string {
	if id == "" {
		return ""
	}
	return strings.ToUpper(id[:1]) + id[1:]
}

// NewPicker returns the program's model for a session that is connected and has
// not chosen a character, with the account's characters as the server listed
// them (newest played first): it opens on the picker with the newest played
// chosen. An empty list has nothing to pick; use NewCreator.
func NewPicker(sess *client.Session, chars []*pb.CharacterEntry) *App {
	a := New(sess)
	p := &screen.Picker{}
	for _, c := range chars {
		row := screen.PickerRow{ID: c.GetCharacter(), Name: c.GetName(), Class: className(c.GetClass()), Level: int64(c.GetLevel()), Zone: c.GetZone(), Online: c.GetOnline()}
		if ms := c.GetLastPlayedMs(); ms > 0 {
			row.Played = time.UnixMilli(ms)
		}
		p.Rows = append(p.Rows, row)
	}
	// The server sorts them, but the choice does not depend on it.
	for i, r := range p.Rows {
		if r.Played.After(p.Rows[p.Sel].Played) {
			p.Sel = i
		}
	}
	a.picker = p
	return a
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
// refusal it says why. A character that is online on another connection is
// refused by the server (character_in_use) and stays on the list.
func (a *App) pickedMsg(msg picked) {
	a.inflight--
	p := a.picker
	if msg.err == nil {
		a.picker = nil
		a.view.Notice, a.view.NoticeBad = "", false
		return
	}
	if p == nil {
		return
	}
	p.Busy = false
	text := msg.err.Error()
	if m := refusalMessage.FindStringSubmatch(text); m != nil {
		text = model.Words(m[1])
	}
	p.Err = text
}
