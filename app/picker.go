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

// pickerRows is the server's list of characters as the picker's rows.
func pickerRows(chars []*pb.CharacterEntry) []screen.PickerRow {
	var rows []screen.PickerRow
	for _, c := range chars {
		row := screen.PickerRow{ID: c.GetCharacter(), Name: c.GetName(), Class: className(c.GetClass()), Level: int64(c.GetLevel()), Zone: c.GetZone(), Online: c.GetOnline()}
		if ms := c.GetLastPlayedMs(); ms > 0 {
			row.Played = time.UnixMilli(ms)
		}
		rows = append(rows, row)
	}
	return rows
}

// NewPicker returns the program's model for a session that is connected and has
// not chosen a character, with the account's characters as the server listed
// them (newest played first): it opens on the picker with the newest played
// chosen. An empty list has nothing to pick; use NewCreator.
func NewPicker(sess *client.Session, chars []*pb.CharacterEntry) *App {
	a := New(sess)
	p := &screen.Picker{Rows: pickerRows(chars)}
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
func (a *App) pickedMsg(msg picked) tea.Cmd {
	a.inflight--
	p := a.picker
	if msg.err == nil {
		a.picker = nil
		a.view.Notice, a.view.NoticeBad = "", false
		return nil
	}
	if p == nil {
		return nil
	}
	p.Busy = false
	text := msg.err.Error()
	if m := refusalMessage.FindStringSubmatch(text); m != nil {
		text = model.Words(m[1])
	}
	p.Err = text
	if msg.err != nil && strings.Contains(msg.err.Error(), "character_in_use") {
		return a.refreshPicker() // the list is stale: the character is in play somewhere
	}
	return nil
}

// pickerRefreshEvery is how often an open picker asks the server for the
// characters again, so that a character another connection has left stops
// showing "online (in use)". One small request per interval, and none once a
// character is chosen.
const pickerRefreshEvery = 5 * time.Second

// listed answers a refresh of the picker's list; err is not nil when the server
// could not say (the rows stay as they were).
type listed struct {
	chars []*pb.CharacterEntry
	err   error
}

// pickerTick is called on each redraw: it asks for a fresh list when the picker
// is open and idle and the interval has passed.
func (a *App) pickerTick() tea.Cmd {
	p := a.picker
	if p == nil || p.Busy || a.refreshing {
		return nil
	}
	if a.now().Sub(a.refreshed) < pickerRefreshEvery {
		return nil
	}
	return a.refreshPicker()
}

// refreshPicker asks the server for the account's characters.
func (a *App) refreshPicker() tea.Cmd {
	if a.refreshing {
		return nil
	}
	a.refreshing, a.refreshed = true, a.now()
	a.inflight++
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		defer cancel()
		chars, err := a.sess.ListCharacters(ctx)
		return listed{chars, err}
	}
}

// listedMsg takes a fresh list into the open picker, keeping the chosen
// character chosen. A failed or empty answer changes nothing.
func (a *App) listedMsg(msg listed) {
	a.inflight--
	a.refreshing = false
	p := a.picker
	if p == nil || msg.err != nil || len(msg.chars) == 0 {
		return
	}
	var chosen string
	if c := p.Chosen(); c != nil {
		chosen = c.ID
	}
	create := p.Create()
	p.Rows = pickerRows(msg.chars)
	if create {
		p.Sel = len(p.Rows)
		return
	}
	p.Sel = 0
	for i, r := range p.Rows {
		if r.ID == chosen {
			p.Sel = i
		}
	}
}
