package app

import (
	"context"
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	"github.com/GeneStevens/seq/agentkit"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The keys of the house companion and of a second character on a stance
// (SEQ-049, D35, D42, D46). The companion's are intents the server judges (the
// quest, the role, a free party slot); the second character is driven from this
// client by a stance (client.Duo), so its keys change that driver's state and
// send nothing themselves.
//
//	P  call the helper to your side, or send it away
//	H  the next stance
//	Y  attack my target      U  come here      J  stand and fight nothing
//	Z  drop the order, back to the stance      .  talk to the other helper

// SetDuo gives the program a second character to show and drive. The driver
// itself is run by whoever owns the Duo: the program's main, or a scenario.
func (a *App) SetDuo(d *client.Duo) { a.duo = d }

// helperKeys are the keys that mean something to a helper.
func helperKey(k string) bool {
	switch k {
	case "p", "h", "y", "u", "j", "z", ".":
		return true
	}
	return false
}

// keysToDuo says whether the helper keys talk to the second character: it is
// what they talk to when the character has no house companion, or has said so.
func (a *App) keysToDuo(m *model.Model) bool {
	if a.duo == nil {
		return false
	}
	return a.toDuo || !companionReady(m)
}

// companionReady says the character has earned a companion and chosen its role.
func companionReady(m *model.Model) bool {
	c := m.Companion()
	return c.GetUnlocked() && c.GetRole() != ""
}

// nextIn is the entry after cur in list, round again; the first if cur is not in it.
func nextIn(list []string, cur string) string {
	i := slices.Index(list, cur)
	return list[(i+1)%len(list)]
}

// helperAction decides what a helper key does against the model and the draft.
func (a *App) helperAction(k string, m *model.Model) Action {
	duo := a.keysToDuo(m)
	if !duo && !companionReady(m) && k != "." {
		return Action{Note: "no companion yet: the Watch Captain's quest (G) earns one"}
	}
	turn := m.TurnBased()
	if k == "." {
		switch {
		case a.duo == nil || !companionReady(m):
			return Action{Note: "there is only one helper to command"}
		}
		a.toDuo = !a.toDuo
		if a.toDuo {
			return Action{Note: "the companion keys now command your second character"}
		}
		return Action{Note: "the companion keys now command your recruit"}
	}
	c := m.Companion()
	step := func(a Action) Action {
		if turn && !duo { // the second character is not planned for: its stance makes its plan each round

			return Action{Kind: PlanAdd, Step: &a}
		}
		return a
	}
	switch k {
	case "p":
		switch {
		case duo && a.duo.Active():
			return Action{Kind: DuoDismiss}
		case duo:
			return Action{Kind: DuoSummon}
		case c.GetPresent() || c.GetSummoned():
			return Action{Kind: CompanionDismiss}
		}
		return Action{Kind: CompanionSummon}
	case "h":
		if duo {
			return Action{Kind: DuoStance, ID: nextIn(agentkit.CompanionStances, a.duo.Stance())}
		}
		return step(Action{Kind: CompanionStance, ID: nextIn(agentkit.CompanionStances, c.GetStance())})
	case "y":
		t := m.Self.Target
		if turn && !duo {
			t = a.draft.target(m)
		}
		if t == "" {
			return Action{Note: "no target: Tab picks one, then Y sends your helper at it"}
		}
		if duo {
			return Action{Kind: DuoOrder, ID: "attack", Target: t}
		}
		return step(Action{Kind: CompanionOrder, ID: "attack", Target: t, Note: nameOf(m, t)})
	case "u":
		at := m.Self.Pos
		if turn && !duo {
			at = a.draft.Pos(m)
		}
		if duo {
			return Action{Kind: DuoOrder, ID: "move", X: at.GetX(), Y: at.GetY()}
		}
		return step(Action{Kind: CompanionOrder, ID: "move", X: at.GetX(), Y: at.GetY()})
	case "j":
		if duo {
			return Action{Kind: DuoOrder, ID: "stop"}
		}
		return step(Action{Kind: CompanionOrder, ID: "stop"})
	case "z":
		if duo {
			return Action{Kind: DuoOrder, ID: "clear"}
		}
		return step(Action{Kind: CompanionOrder, ID: "clear"})
	}
	return Action{}
}

// companion sends the intent of a companion action and says in a word what it
// was for the status line.
func (a *App) companion(ctx context.Context, act Action) (what, id string, err error) {
	switch act.Kind {
	case CompanionSummon:
		what = "call the companion"
		id, err = a.sess.CompanionSummon(ctx)
	case CompanionDismiss:
		what = "send the companion away"
		id, err = a.sess.CompanionDismiss(ctx)
	case CompanionStance:
		what = "companion stance " + act.ID
		id, err = a.sess.CompanionStance(ctx, act.ID)
	case CompanionRole:
		what = "recruit a " + act.ID
		id, err = a.sess.CompanionRole(ctx, act.NPC, act.ID)
	case CompanionOrder:
		what = "companion order: " + act.ID
		id, err = a.sess.CompanionOrder(ctx, act.ID, act.Target, act.X, act.Y)
	}
	return what, id, err
}

// duoDo does an action on the second character: a stance or an order changes
// the driver, summoning and dismissing log it in and out. It returns the
// command to run, if any, and the note for the status line.
func (a *App) duoDo(act Action) (tea.Cmd, string) {
	switch act.Kind {
	case DuoStance:
		a.duo.SetStance(act.ID)
		return nil, "second character: stance " + act.ID
	case DuoOrder:
		switch act.ID {
		case "attack":
			a.duo.SetOrder(client.Order{Kind: "attack", Target: act.Target})
		case "move":
			a.duo.SetOrder(client.Order{Kind: "move", Dest: posOf(act.X, act.Y)})
		case "stop":
			a.duo.SetOrder(client.Order{Kind: "stop"})
		default:
			a.duo.SetOrder(client.Order{})
			return nil, "second character: back to its stance"
		}
		return nil, "second character ordered: " + act.ID
	}
	a.inflight++
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		defer cancel()
		if act.Kind == DuoDismiss {
			if err := a.duo.Dismiss(ctx); err != nil {
				return resultMsg{note: "second character not sent away: " + err.Error(), bad: true}
			}
			return resultMsg{note: "second character logged out"}
		}
		if err := a.duo.Summon(ctx); err != nil {
			return resultMsg{note: "second character not brought in: " + err.Error(), bad: true}
		}
		return resultMsg{note: "second character in play"}
	}, ""
}

// roleKey handles a key on the screen for choosing the recruit's role.
func (a *App) roleKey(k string, m *model.Model) Action {
	r := a.role
	n := len(agentkit.CompanionRoles)
	r.Sel = max(min(r.Sel, n-1), 0)
	switch {
	case k == "ctrl+c":
		return Action{Kind: Quit}
	case k == "esc" || k == "q":
		a.role, a.talk = nil, &screen.Talk{NPC: r.NPC}
	case up(k) || down(k):
		r.Sel = step(r.Sel, k, n)
	case enter(k):
		role := agentkit.CompanionRoles[r.Sel]
		npc := r.NPC
		a.role, a.talk = nil, &screen.Talk{NPC: npc}
		return Action{Kind: CompanionRole, NPC: npc, ID: role}
	case len(k) == 1 && k[0] >= '1' && int(k[0]-'1') < n:
		r.Sel = int(k[0] - '1')
	}
	return Action{}
}

// companionStepText is how the plan lists a companion step.
func companionStepText(s Action, m *model.Model) string {
	switch s.Kind {
	case CompanionStance:
		return "companion: stance " + s.ID
	}
	switch s.ID {
	case "attack":
		return "companion: attack " + nameOf(m, s.Target)
	case "move":
		return fmt.Sprintf("companion: go to (%d,%d)", s.X, s.Y)
	case "stop":
		return "companion: stand"
	}
	return "companion: back to its stance"
}

func posOf(x, y int64) *pb.Pos { return &pb.Pos{X: x, Y: y} }
