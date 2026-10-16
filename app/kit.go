package app

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
)

// createKey handles a key on the character creation screen. The name takes
// letters; Tab or Down moves to the class list, where Up and Down or 1 to 6
// choose; Enter makes the character from either.
func (a *App) createKey(k string) Action {
	c := a.create
	if c.Busy {
		if k == "ctrl+c" {
			return Action{Kind: Quit}
		}
		return Action{}
	}
	c.Err = ""
	switch k {
	case "ctrl+c":
		return Action{Kind: Quit}
	case "enter":
		if n := len([]rune(c.Name)); n < 3 {
			c.Err = "a name is 3 to 16 letters"
			return Action{}
		}
		return Action{Kind: CreateCharacter}
	case "tab":
		c.Classes = !c.Classes
		return Action{}
	case "esc":
		c.Classes = false
		return Action{}
	}
	if !c.Classes {
		switch {
		case k == "down":
			c.Classes = true
		case k == "backspace":
			if r := []rune(c.Name); len(r) > 0 {
				c.Name = string(r[:len(r)-1])
			}
		case len(k) == 1 && (k[0] >= 'a' && k[0] <= 'z' || k[0] >= 'A' && k[0] <= 'Z'):
			if len(c.Name) < screen.MaxName {
				c.Name += k
			}
		}
		return Action{}
	}
	switch {
	case k == "up":
		if c.Class == 0 {
			c.Classes = false
		} else {
			c.Class--
		}
	case k == "down":
		c.Class = min(c.Class+1, len(screen.Classes)-1)
	case len(k) == 1 && k[0] >= '1' && k[0] <= byte('0'+len(screen.Classes)):
		c.Class = int(k[0] - '1')
	}
	return Action{}
}

// createCmd makes the character on the server, then chooses it: the first
// snapshot is in the model when it returns.
func (a *App) createCmd(name string, class string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*commandTimeout)
		defer cancel()
		id, err := a.sess.CreateCharacter(ctx, name, class)
		if err == nil {
			err = a.sess.Select(ctx, id)
		}
		return created{err}
	}
}

// createError is why a character was not made, in words.
func createError(err error) string {
	var r client.Refused
	if errors.As(err, &r) {
		return model.Words(r.Reason)
	}
	return err.Error()
}

// slotKey handles a key on the slot screen: it chooses a slot, then an ability
// for it, and sends set_slot. The server refuses it in combat.
func (a *App) slotKey(k string, m *model.Model) Action {
	s := a.slots
	n := len(m.Self.Abilities)
	if s.Pick {
		switch k {
		case "up", "w", "k":
			s.Cur = max(s.Cur-1, 0)
		case "down", "s", "j":
			s.Cur = min(s.Cur+1, max(n-1, 0))
		case "esc", "left":
			s.Pick = false
		case "enter", "space", " ":
			s.Pick = false
			if s.Cur < n {
				return Action{Kind: SetSlot, Slot: s.Slot + 1, ID: m.Self.Abilities[s.Cur].GetId()}
			}
		case "ctrl+c":
			return Action{Kind: Quit}
		}
		return Action{}
	}
	switch k {
	case "up", "w", "k":
		s.Slot = max(s.Slot-1, 0)
	case "down", "s", "j":
		s.Slot = min(s.Slot+1, model.Slots-1)
	case "esc", "b", "q":
		a.slots = nil
	case "enter", "space", " ", "right":
		if n == 0 {
			return Action{Note: "you know no abilities yet"}
		}
		s.Pick, s.Cur = true, 0
		for i, ab := range m.Self.Abilities {
			if ab.GetId() == m.Self.Slots[s.Slot].Ability {
				s.Cur = i
			}
		}
	case "backspace", "delete", "x":
		return Action{Kind: SetSlot, Slot: s.Slot + 1}
	case "ctrl+c":
		return Action{Kind: Quit}
	default:
		if i := slotKey(k); i > 0 {
			s.Slot = i - 1
		}
	}
	return Action{}
}
