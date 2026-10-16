// Package app is the Bubble Tea program: it turns keys into the client's
// commands and the model into the screen. It holds no game rule; where a key
// needs a choice (the next target, the nearest drop) it only picks among
// things the model shows, and the server decides whether the intent is allowed.
package app

import (
	"fmt"
	"sort"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq/agentkit"
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
	Release
	Logout
	Quit
	ToggleHelp
	ToggleVerbose
	Stop       // a plan step: stop where you are
	OpenMenu   // the instance menu
	PlanAdd    // add Step to the plan
	PlanUndo   // remove the last step
	PlanClear  // remove every step
	PlanSubmit // send the plan
	OpenInstance
	UseAbility // use the ability in Slot (1 to 8); ID is a target, "" the current one
	SetSlot    // put the ability ID in Slot ("" empties it)
	OpenSlots  // the slot screen
	CreateCharacter
)

// Action is a key's meaning, with the choice it needs already made.
type Action struct {
	Kind Kind
	X, Y int64   // Move: the destination
	ID   string  // Target, Loot, UseDoor: what it names
	On   bool    // Attack
	Slot int     // UseAbility, SetSlot: 1 to 8
	Note string  // None: why nothing happens, for the status line
	Step *Action // PlanAdd: the step to add

	// PlanSubmit: the plan as the toolkit takes it, and the round and edit it
	// is a copy of.
	Steps []agentkit.Step
	Round uint64
	Ver   int
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
	case "r":
		return Action{Kind: Release} // the server denies it unless the character is dead
	}
	if n := slotKey(key); n > 0 {
		return Action{Kind: UseAbility, Slot: n, Note: slotLabel(m, n)}
	}
	return common(key)
}

// common are the keys that mean the same in real time and in a plan.
func common(key string) Action {
	switch key {
	case "o":
		return Action{Kind: Logout}
	case "q", "ctrl+c":
		return Action{Kind: Quit}
	case "?":
		return Action{Kind: ToggleHelp}
	case "v":
		return Action{Kind: ToggleVerbose}
	case "i":
		return Action{Kind: OpenMenu}
	case "b":
		return Action{Kind: OpenSlots}
	}
	return Action{}
}

// slotKey is the slot a number key names (1 to 8), 0 for any other key.
func slotKey(key string) int {
	if len(key) == 1 && key[0] >= '1' && key[0] <= '8' {
		return int(key[0] - '0')
	}
	return 0
}

// slotLabel names a slot for the status line and the plan: "key 1 Flame Dart".
func slotLabel(m *model.Model, n int) string {
	if id := m.Self.Slots[n-1].Ability; id != "" {
		return fmt.Sprintf("key %d %s", n, m.AbilityName(id))
	}
	return fmt.Sprintf("key %d", n)
}

func d2(a, b *pb.Pos) int64 {
	dx, dy := a.GetX()-b.GetX(), a.GetY()-b.GetY()
	return dx*dx + dy*dy
}

// nextTarget is the living mob after the current target, nearest first, and
// round again.
func nextTarget(m *model.Model) string { return nextTargetFrom(m, m.Self.Target) }

// nextTargetFrom is the living mob after cur, nearest first.
func nextTargetFrom(m *model.Model, cur string) string {
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
		if id == cur {
			return ids[(i+1)%len(ids)]
		}
	}
	if len(ids) > 0 {
		return ids[0]
	}
	return ""
}

// nearestDrop prefers a drop this character may take.
func nearestDrop(m *model.Model) string { return nearestDropTo(m, m.Self.Pos) }

func nearestDropTo(m *model.Model, from *pb.Pos) string {
	best, bestOK, bestD := "", false, int64(0)
	for _, id := range m.DropIDs() {
		d := m.Drops[id]
		ok, dd := d.GetCanLoot(), d2(d.GetPos(), from)
		if best == "" || (ok && !bestOK) || (ok == bestOK && dd < bestD) {
			best, bestOK, bestD = id, ok, dd
		}
	}
	return best
}

func nearestDoor(m *model.Model) string { return nearestDoorTo(m, m.Self.Pos) }

func nearestDoorTo(m *model.Model, from *pb.Pos) string {
	best, bestD := "", int64(0)
	for _, d := range m.Map.GetDoors() {
		if dd := d2(d.GetPos(), from); best == "" || dd < bestD {
			best, bestD = d.GetId(), dd
		}
	}
	return best
}
