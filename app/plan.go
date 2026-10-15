// Package app is the Bubble Tea program: it turns keys into the client's
// commands and the model into the screen. It holds no game rule; where a key
// needs a choice (the next target, the nearest drop) it only picks among
// things the model shows, and the server decides whether the intent is allowed.
package app

import (
	"sort"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// StepMm is how far one press of a move key aims: the destination sent is this
// far from where the character stands.
const StepMm = 3000

// Kind says what a key does.
type Kind int

const (
	None Kind = iota
	Move
	Target // set the target to ID ("" clears it)
	Attack // turn auto-attack On
	Loot
	UseDoor
	Logout
	Quit
	ToggleHelp
	ToggleVerbose
)

// Action is a key's meaning, with the choice it needs already made.
type Action struct {
	Kind Kind
	X, Y int64  // Move: the destination
	ID   string // Target, Loot, UseDoor: what it names
	On   bool   // Attack
	Note string // None: why nothing happens, for the status line
}

// Plan decides what a key does against the model. It is pure.
func Plan(key string, m *model.Model) Action {
	sp := m.Self.Pos
	switch key {
	case "up", "w":
		return Action{Kind: Move, X: sp.GetX(), Y: sp.GetY() + StepMm}
	case "down", "s":
		return Action{Kind: Move, X: sp.GetX(), Y: sp.GetY() - StepMm}
	case "left", "a":
		return Action{Kind: Move, X: sp.GetX() - StepMm, Y: sp.GetY()}
	case "right", "d":
		return Action{Kind: Move, X: sp.GetX() + StepMm, Y: sp.GetY()}
	case "tab":
		if id := nextTarget(m); id != "" {
			return Action{Kind: Target, ID: id}
		}
		return Action{Note: "no target in view"}
	case "esc":
		if m.Self.Target != "" {
			return Action{Kind: Target}
		}
	case "f", "space", " ":
		return Action{Kind: Attack, On: !m.Self.AutoAttack}
	case "l":
		if id := nearestDrop(m); id != "" {
			return Action{Kind: Loot, ID: id}
		}
		return Action{Note: "nothing to loot in view"}
	case "e", "enter":
		if id := nearestDoor(m); id != "" {
			return Action{Kind: UseDoor, ID: id}
		}
		return Action{Note: "no door on this map"}
	case "o":
		return Action{Kind: Logout}
	case "q", "ctrl+c":
		return Action{Kind: Quit}
	case "?":
		return Action{Kind: ToggleHelp}
	case "v":
		return Action{Kind: ToggleVerbose}
	}
	return Action{}
}

func d2(a, b *pb.Pos) int64 {
	dx, dy := a.GetX()-b.GetX(), a.GetY()-b.GetY()
	return dx*dx + dy*dy
}

// nextTarget is the living mob after the current target, nearest first, and
// round again.
func nextTarget(m *model.Model) string {
	var ids []string
	for _, id := range m.EntityIDs() {
		if e := m.Entities[id]; e.GetKind() == pb.EntityKind_ENTITY_KIND_MOB && !e.GetDead() {
			ids = append(ids, id)
		}
	}
	sort.SliceStable(ids, func(i, j int) bool {
		return d2(m.Entities[ids[i]].GetPos(), m.Self.Pos) < d2(m.Entities[ids[j]].GetPos(), m.Self.Pos)
	})
	for i, id := range ids {
		if id == m.Self.Target {
			return ids[(i+1)%len(ids)]
		}
	}
	if len(ids) > 0 {
		return ids[0]
	}
	return ""
}

// nearestDrop prefers a drop this character may take.
func nearestDrop(m *model.Model) string {
	best, bestOK, bestD := "", false, int64(0)
	for _, id := range m.DropIDs() {
		d := m.Drops[id]
		ok, dd := d.GetCanLoot(), d2(d.GetPos(), m.Self.Pos)
		if best == "" || (ok && !bestOK) || (ok == bestOK && dd < bestD) {
			best, bestOK, bestD = id, ok, dd
		}
	}
	return best
}

func nearestDoor(m *model.Model) string {
	best, bestD := "", int64(0)
	for _, d := range m.Map.GetDoors() {
		if dd := d2(d.GetPos(), m.Self.Pos); best == "" || dd < bestD {
			best, bestD = d.GetId(), dd
		}
	}
	return best
}
