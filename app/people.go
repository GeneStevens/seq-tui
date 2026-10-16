package app

import (
	"context"
	"fmt"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The keys of talking, the contract board, the Contracts screen and the zone
// overview (SEQ-044). None of them decides a rule: a talk, a contract taken,
// handed in or given up is an intent, and the server says whether it holds.

// talkAction is the key G: talk to the NPC nearest the character. The server
// decides whether it is within reach.
func talkAction(m *model.Model) Action {
	n := m.NearestNPC()
	if n == nil {
		return Action{Note: "nobody here to talk to"}
	}
	return Action{Kind: Talk, NPC: n.GetId(), Note: n.GetName()}
}

// people sends the intent for an action of talk or contracts, and says in a
// word what it was for the status line.
func (a *App) people(ctx context.Context, act Action) (what, id string, err error) {
	switch act.Kind {
	case Talk:
		what = "talk to " + act.Note
		id, err = a.sess.Talk(ctx, act.NPC)
	case ContractAccept:
		what = "take " + act.Note
		id, err = a.sess.ContractAccept(ctx, act.NPC, act.ID)
	case ContractTurnIn:
		what = "hand in " + act.Note
		id, err = a.sess.ContractTurnIn(ctx, act.NPC, act.ID)
	case ContractAbandon:
		what = "give up " + act.Note
		id, err = a.sess.ContractAbandon(ctx, act.ID)
	}
	return what, id, err
}

// talkKey handles a key on the conversation screen.
func (a *App) talkKey(k string, m *model.Model) Action {
	t := a.talk
	d := m.Said(t.NPC)
	n := 0
	if d != nil {
		n = len(d.Options)
	}
	t.Sel = max(min(t.Sel, n-1), 0)
	choose := func(i int) Action {
		if d == nil || i >= len(d.Options) {
			return Action{}
		}
		t.Sel = i
		return a.option(m, t.NPC, d.Options[i])
	}
	switch {
	case k == "ctrl+c":
		return Action{Kind: Quit}
	case k == "esc" || k == "q":
		a.talk = nil
	case k == "g":
		return Action{Kind: Talk, NPC: t.NPC, Note: m.NPCName(t.NPC)}
	case up(k) || down(k):
		t.Sel = step(t.Sel, k, n)
	case enter(k):
		if n == 0 {
			return Action{Note: "they have nothing to offer"}
		}
		return choose(t.Sel)
	case len(k) == 1 && k[0] >= '1' && k[0] <= '9':
		if i := int(k[0] - '1'); i < n {
			return choose(i)
		}
	}
	return Action{}
}

// option does what a dialogue option says: the trading, teaching, binding and
// banking ones open the screen of that trader on this NPC, the contract ones the
// board, and one with no action says so.
func (a *App) option(m *model.Model, npc string, o *pb.DialogueOption) Action {
	switch o.GetAction() {
	case "buy", "sell", "train", "bind", "bank_deposit", "bank_withdraw":
		return Action{Kind: OpenTradersAt, NPC: npc}
	case "contract_accept":
		return Action{Kind: OpenBoard, NPC: npc, Count: 0}
	case "contract_turn_in":
		return Action{Kind: OpenBoard, NPC: npc, Count: 1}
	case "companion_role":
		return Action{Kind: OpenRole, NPC: npc}
	}
	return Action{Note: fmt.Sprintf("%s has nothing for you to do here yet", m.NPCName(npc))}
}

// boardKey handles a key on the contract board's screen.
func (a *App) boardKey(k string, m *model.Model) Action {
	b := a.board
	n := m.NPC(b.NPC)
	var offers []*pb.ContractOffer
	if n != nil {
		offers = n.GetOffers()
	}
	held := m.Self.Contracts
	b.Cur[0], b.Cur[1] = max(min(b.Cur[0], len(offers)-1), 0), max(min(b.Cur[1], len(held)-1), 0)
	switch {
	case k == "ctrl+c":
		return Action{Kind: Quit}
	case k == "esc" || k == "q":
		a.board, a.talk = nil, &screen.Talk{NPC: b.NPC}
	case k == "tab" || k == "left" || k == "right":
		b.Pane = 1 - b.Pane
	case up(k) || down(k):
		if b.Pane == 0 {
			b.Cur[0] = step(b.Cur[0], k, len(offers))
		} else {
			b.Cur[1] = step(b.Cur[1], k, len(held))
		}
	case k == "x" && b.Pane == 1:
		if len(held) == 0 {
			return Action{Note: "you hold no contracts"}
		}
		c := held[b.Cur[1]]
		return Action{Kind: ContractAbandon, ID: c.GetId(), Note: c.GetName()}
	case enter(k) && b.Pane == 0:
		if len(offers) == 0 {
			return Action{Note: "the board has nothing for you now"}
		}
		o := offers[b.Cur[0]]
		return Action{Kind: ContractAccept, NPC: b.NPC, ID: o.GetId(), Note: o.GetName()}
	case enter(k):
		if len(held) == 0 {
			return Action{Note: "you hold no contracts to hand in"}
		}
		c := held[b.Cur[1]]
		return Action{Kind: ContractTurnIn, NPC: b.NPC, ID: c.GetId(), Note: c.GetName()}
	}
	return Action{}
}

// contractsKey handles a key on the Contracts screen.
func (a *App) contractsKey(k string, m *model.Model) Action {
	c := a.cons
	held := m.Self.Contracts
	c.Sel = max(min(c.Sel, len(held)-1), 0)
	switch {
	case k == "ctrl+c":
		return Action{Kind: Quit}
	case k == "esc" || k == "q":
		a.cons = nil
	case up(k) || down(k):
		c.Sel = step(c.Sel, k, len(held))
	case k == "x":
		if len(held) == 0 {
			return Action{Note: "you hold no contracts"}
		}
		h := held[c.Sel]
		return Action{Kind: ContractAbandon, ID: h.GetId(), Note: h.GetName()}
	}
	return Action{}
}

// overviewKey handles a key on the zone overview.
func (a *App) overviewKey(k string) Action {
	switch k {
	case "ctrl+c":
		return Action{Kind: Quit}
	case "esc", "m", "q":
		a.overview = false
	}
	return Action{}
}

// tradersAt is the place of an NPC in the traders' list, -1 if it is not in view.
func tradersAt(m *model.Model, npc string) int {
	for i, n := range screen.TradersInView(m) {
		if n.GetId() == npc {
			return i
		}
	}
	return -1
}
