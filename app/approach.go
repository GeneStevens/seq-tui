package app

import (
	"fmt"
	"math"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Option-click (D109). A plain click never moves the character; Option (Alt) is
// the walking click, the same in the terminal and in the browser. The engine has
// no follow or approach intent, so the client closes in itself: it sends a Move
// towards the thing, again as the thing moves, and acts once the character is in
// reach (arms auto-attack on a mob, talks to a person, loots a drop). Esc, a
// movement key or a click on the ground ends it.

const (
	// approachEvery is how often an approach looks again (the redraw tick).
	approachStall  = 600 * time.Millisecond // a walk that has stopped short is begun again after this
	approachGiveUp = 45 * time.Second
	// approachMoved is how far the thing may move from the last destination sent
	// before the Move is sent again.
	approachMoved = 1500
	dropReachMm   = 5000 // a drop is looted from within this (as a click always did)
)

// approach is the thing the character is closing in on.
type approach struct {
	id    string
	kind  screen.PickKind
	name  string
	since time.Time
	sent  time.Time // when the last Move went
	dest  *pb.Pos   // where it went
}

// selection is the mob, person or drop last clicked: W walks to it.
type selection struct {
	id   string
	kind screen.PickKind
	name string
}

// thingPos is where the thing is now; nil when it is gone.
func thingPos(m *model.Model, kind screen.PickKind, id string) *pb.Pos {
	switch kind {
	case screen.PickMob, screen.PickPlayer:
		if e := m.Entities[id]; e != nil {
			return e.GetPos()
		}
	case screen.PickNPC:
		if n := m.NPC(id); n != nil {
			return n.GetPos()
		}
	case screen.PickDrop:
		if d := m.Drops[id]; d != nil {
			return d.GetPos()
		}
	}
	return nil
}

// reachOf is how close the character must be to act on the thing.
func reachOf(m *model.Model, kind screen.PickKind, id string) int64 {
	switch kind {
	case screen.PickNPC:
		if n := m.NPC(id); n != nil && n.GetReachMm() > 0 {
			return n.GetReachMm()
		}
		return dropReachMm
	case screen.PickDrop:
		return dropReachMm
	}
	return screen.MeleeMm(m)
}

// mmTo is how far the character is from p.
func mmTo(m *model.Model, p *pb.Pos) int64 {
	if p == nil || m.Self.Pos == nil {
		return 0
	}
	return int64(math.Round(math.Hypot(float64(p.GetX()-m.Self.Pos.GetX()), float64(p.GetY()-m.Self.Pos.GetY()))))
}

// metresWords is a distance in whole metres.
func metresWords(mm int64) int64 { return (mm + 500) / 1000 }

// tooFar is the note for a click on a thing out of reach.
func tooFar(m *model.Model, p *pb.Pos) string {
	return fmt.Sprintf("Too far (%d m): Option-click to walk there", metresWords(mmTo(m, p)))
}

// standAt is where to send the character to reach the thing at p: a little short
// of it, so that the walk ends inside reach and not on top of it.
func standAt(m *model.Model, p *pb.Pos, reach int64) (x, y int64) {
	d := mmTo(m, p)
	if d == 0 {
		return p.GetX(), p.GetY()
	}
	short := float64(reach) * 0.6
	if float64(d) <= short {
		return m.Self.Pos.GetX(), m.Self.Pos.GetY()
	}
	k := (float64(d) - short) / float64(d)
	return m.Self.Pos.GetX() + int64(math.Round(float64(p.GetX()-m.Self.Pos.GetX())*k)),
		m.Self.Pos.GetY() + int64(math.Round(float64(p.GetY()-m.Self.Pos.GetY())*k))
}

// attackingIt says auto-attack is already on for the mob id.
func attackingIt(m *model.Model, id string) bool { return m.Self.AutoAttack && m.Self.Target == id }

// approachSteps is what to send now to close in on the thing: the intents, whether
// the approach is over, and a note for the status line when it ended without
// acting. It reads the model only.
func (a *App) approachSteps(m *model.Model, ap *approach) (acts []Action, done bool, note string) {
	now := a.now()
	pos := thingPos(m, ap.kind, ap.id)
	if e := m.Entities[ap.id]; pos == nil || (ap.kind == screen.PickMob && e.GetDead()) {
		return nil, true, ap.name + " is gone"
	}
	if now.Sub(ap.since) > approachGiveUp {
		return nil, true, "gave up closing in on " + ap.name
	}
	if mmTo(m, pos) <= reachOf(m, ap.kind, ap.id) {
		if m.Walking() {
			acts = append(acts, Action{Kind: Stop})
		}
		switch ap.kind {
		case screen.PickMob:
			if m.Self.Target != ap.id {
				acts = append(acts, Action{Kind: Target, ID: ap.id})
			}
			if !attackingIt(m, ap.id) {
				acts = append(acts, Action{Kind: Attack, On: true, Note: "attack " + ap.name})
			}
		case screen.PickNPC:
			acts = append(acts, Action{Kind: Talk, NPC: ap.id, Note: ap.name})
		case screen.PickDrop:
			acts = append(acts, Action{Kind: Loot, ID: ap.id})
		}
		return acts, true, ""
	}
	moved := ap.dest == nil || math.Hypot(float64(pos.GetX()-ap.dest.GetX()), float64(pos.GetY()-ap.dest.GetY())) > approachMoved
	if moved || (!m.Walking() && now.Sub(ap.sent) >= approachStall) {
		x, y := standAt(m, pos, reachOf(m, ap.kind, ap.id))
		ap.dest, ap.sent = &pb.Pos{X: pos.GetX(), Y: pos.GetY()}, now
		acts = append(acts, Action{Kind: Move, X: x, Y: y, Quiet: true})
	}
	return acts, false, ""
}

// startApproach begins closing in on a pick; its first intents go at once: the
// target of a mob, and the first step.
func (a *App) startApproach(m *model.Model, act Action) tea.Cmd {
	ap := &approach{id: act.ID, kind: screen.PickKind(act.Count), name: act.Note, since: a.now()}
	a.approach = ap
	var acts []Action
	if ap.kind == screen.PickMob && m.Self.Target != ap.id {
		acts = append(acts, Action{Kind: Target, ID: ap.id, Quiet: true})
	}
	more, done, note := a.approachSteps(m, ap)
	acts = append(acts, more...)
	if done {
		a.approach = nil
		a.view.Notice = note
	} else {
		a.view.Notice = "Closing in on " + ap.name + " (Esc cancels)"
	}
	return a.sendInOrder(acts)
}

// approachTick runs on the redraw tick: it looks at the thing again.
func (a *App) approachTick() tea.Cmd {
	ap := a.approach
	if ap == nil {
		return nil
	}
	if a.focus || a.replay != nil || a.menu != nil {
		a.approach = nil
		return nil
	}
	var (
		acts []Action
		done bool
		note string
	)
	a.sess.View(func(m *model.Model) { acts, done, note = a.approachSteps(m, ap) })
	if done {
		a.approach = nil
		if note != "" {
			a.view.Notice, a.view.NoticeBad = note, false
		}
		a.paint.dirty = true
	}
	return a.sendInOrder(acts)
}

// sendInOrder sends the intents one after another, each waiting for the last
// one's answer (a target must be set before the attack that names none).
func (a *App) sendInOrder(acts []Action) tea.Cmd {
	var cmds []tea.Cmd
	for _, act := range acts {
		a.inflight++
		send := a.send(act)
		if act.Quiet {
			cmds = append(cmds, quiet(send))
			continue
		}
		cmds = append(cmds, send)
	}
	switch len(cmds) {
	case 0:
		return nil
	case 1:
		return cmds[0]
	}
	return tea.Sequence(cmds...)
}

// quiet makes a command's answer leave the status line alone, unless it is a
// refusal.
func quiet(c tea.Cmd) tea.Cmd {
	return func() tea.Msg {
		msg := c()
		if r, ok := msg.(resultMsg); ok && !r.bad {
			r.quiet = true
			return r
		}
		return msg
	}
}

// walkToAction is the key W (Option-click's fallback): walk to the thing last
// clicked, else the mob targeted, else the ground last clicked.
func (a *App) walkToAction(m *model.Model) Action {
	ms := &a.mouse
	if s := ms.sel; s != nil {
		if p := thingPos(m, s.kind, s.id); p != nil {
			return Action{Kind: Approach, ID: s.id, Count: int(s.kind), Note: s.name, X: p.GetX(), Y: p.GetY()}
		}
	}
	if id := m.Self.Target; id != "" {
		if e := m.Entities[id]; e != nil && !e.GetDead() {
			return Action{Kind: Approach, ID: id, Count: int(screen.PickMob), Note: e.GetName(), X: e.GetPos().GetX(), Y: e.GetPos().GetY()}
		}
	}
	if p := ms.spot; p != nil {
		return Action{Kind: Move, X: p.GetX(), Y: p.GetY(), Note: "walk to the spot you clicked"}
	}
	return Action{Note: "Click a mob, person, drop or spot first: W walks to it"}
}
