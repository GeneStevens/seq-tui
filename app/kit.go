package app

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq/agentkit"
	pb "github.com/GeneStevens/seq/protocol/seqv1"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
)

// createKey handles a key on the character creation screen. The name takes
// letters; Tab or Down moves on to the body line, when the server lists more
// than one body (D202), and then to the class list (the server's menu), where
// Up and Down or the number choose; Enter makes the character from any of them.
func (a *App) createKey(k string, m *model.Model) Action {
	c := a.create
	if c.Busy {
		if k == "ctrl+c" {
			return Action{Kind: Quit}
		}
		return Action{}
	}
	c.Err = ""
	bodies := m.Bodies()
	if len(bodies) < 2 {
		bodies = nil
	}
	switch k {
	case "ctrl+c":
		return Action{Kind: Quit}
	case "ctrl+l":
		return Action{Kind: OpenSessions}
	case "enter":
		if n := len([]rune(c.Name)); n < 3 {
			c.Err = "a name is 3 to 16 letters"
			return Action{}
		}
		if bodies != nil && c.Body == pb.BodyType_BODY_TYPE_UNSPECIFIED {
			c.Err = "choose a body: Female or Male"
			return Action{}
		}
		menu := m.Classes()
		if c.Class >= len(menu) {
			c.Err = "the server has sent no class menu to choose from"
			return Action{}
		}
		act := Action{Kind: CreateCharacter, ID: menu[c.Class].GetId(), Note: c.Name}
		if bodies != nil {
			act.Body = agentkit.BodyWord(c.Body)
		}
		return act
	case "tab":
		switch {
		case !c.Bodies && !c.Classes && bodies != nil:
			c.Bodies = true
		case c.Bodies:
			c.Bodies, c.Classes = false, true
		default:
			c.Classes = !c.Classes
		}
		return Action{}
	case "esc":
		if !c.Classes && !c.Bodies && c.Name == "" && a.back != nil {
			a.picker, a.back, a.create = a.back, nil, nil // back to the characters
			return Action{}
		}
		c.Classes, c.Bodies = false, false
		return Action{}
	}
	if c.Bodies {
		switch k {
		case "up":
			c.Bodies = false
		case "down":
			c.Bodies, c.Classes = false, true
		case "right", "left":
			c.Body = nextBody(bodies, c.Body, k == "right")
		case "f", "F":
			c.Body = chooseBody(bodies, pb.BodyType_BODY_TYPE_FEMALE, c.Body)
		case "m", "M":
			c.Body = chooseBody(bodies, pb.BodyType_BODY_TYPE_MALE, c.Body)
		}
		return Action{}
	}
	if !c.Classes {
		switch {
		case k == "down":
			c.Bodies, c.Classes = bodies != nil, bodies == nil
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
			c.Classes, c.Bodies = false, bodies != nil
		} else {
			c.Class--
		}
	case k == "down":
		c.Class = min(c.Class+1, max(len(m.Classes())-1, 0))
	case len(k) == 1 && k[0] >= '1' && k[0] <= byte('0'+min(len(m.Classes()), 9)):
		c.Class = int(k[0] - '1')
	}
	return Action{}
}

// nextBody is the body after (or before) the chosen one among the listed; with
// none chosen, Right takes the first and Left the last.
func nextBody(bodies []pb.BodyType, cur pb.BodyType, forward bool) pb.BodyType {
	at := -1
	for i, b := range bodies {
		if b == cur {
			at = i
		}
	}
	switch {
	case at < 0 && forward:
		return bodies[0]
	case at < 0:
		return bodies[len(bodies)-1]
	case forward:
		return bodies[(at+1)%len(bodies)]
	}
	return bodies[(at+len(bodies)-1)%len(bodies)]
}

// chooseBody is want when the server lists it, else what was chosen.
func chooseBody(bodies []pb.BodyType, want, cur pb.BodyType) pb.BodyType {
	for _, b := range bodies {
		if b == want {
			return want
		}
	}
	return cur
}

// createCmd makes the character on the server, then chooses it: the first
// snapshot is in the model when it returns. When the character is made and the
// choosing fails, the answer carries its id, so the picker offers it and
// nothing makes it a second time.
func (a *App) createCmd(name, class, body string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*commandTimeout)
		defer cancel()
		id, err := a.sess.CreateCharacterBody(ctx, name, class, body)
		if err != nil {
			return created{err: err}
		}
		if err = a.sess.Select(ctx, id); err != nil {
			return created{err: err, id: id}
		}
		return created{}
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
