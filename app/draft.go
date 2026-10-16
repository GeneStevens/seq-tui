package app

import (
	"fmt"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	"github.com/GeneStevens/seq/agentkit"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Draft is the plan being built for the open round of a turn-based instance:
// an ordered list of the same actions the keys make in real time (D30). It
// holds no rule; the server judges the plan when it runs.
type Draft struct {
	Steps []Action
	// Sent: the server has taken exactly these steps as the character's plan.
	Sent bool
	// Round is the round the draft is for.
	Round uint64
	// Ver counts edits, so a receipt for an older version is not taken for
	// the current one.
	Ver int
}

func (d *Draft) edited() { d.Sent = false; d.Ver++ }

// movement and action are the two kinds of step a plan holds one of (doc 05
// §4.6): a move or a door, and an ability or an auto-attack toggle.
func movement(k Kind) bool { return k == Move || k == UseDoor }
func action(k Kind) bool   { return k == UseAbility || k == Attack }

// companionStep says a step is a word to the house companion: a plan holds one
// stance and one order for it.
func companionStep(k Kind) bool { return k == CompanionStance || k == CompanionOrder }

// Add appends a step. It says why not, in words, when the plan is full or
// already holds the one movement or the one action it may; the server judges
// the plan again when it runs.
func (d *Draft) Add(a Action) string {
	if len(d.Steps) >= screen.MaxPlanSteps {
		return fmt.Sprintf("the plan is full (%d steps): Enter submits it, Backspace removes a step", screen.MaxPlanSteps)
	}
	for _, s := range d.Steps {
		switch {
		case movement(a.Kind) && movement(s.Kind):
			return "a plan holds one move or door: Backspace removes the one it has"
		case action(a.Kind) && action(s.Kind):
			return "a plan holds one ability or attack toggle: Backspace removes the one it has"
		case companionStep(a.Kind) && a.Kind == s.Kind:
			return "a plan holds one companion stance and one companion order: Backspace removes the one it has"
		}
	}
	d.Steps = append(d.Steps, a)
	d.edited()
	return ""
}

// Undo removes the last step.
func (d *Draft) Undo() bool {
	if len(d.Steps) == 0 {
		return false
	}
	d.Steps = d.Steps[:len(d.Steps)-1]
	d.edited()
	return true
}

// Clear removes every step.
func (d *Draft) Clear() bool {
	if len(d.Steps) == 0 {
		return false
	}
	d.Steps = nil
	d.edited()
	return true
}

// Pos is where the character would stand after the plan's moves: the last
// move's destination, else where it stands now.
func (d *Draft) Pos(m *model.Model) *pb.Pos {
	for i := len(d.Steps) - 1; i >= 0; i-- {
		if d.Steps[i].Kind == Move {
			return &pb.Pos{X: d.Steps[i].X, Y: d.Steps[i].Y}
		}
	}
	return m.Self.Pos
}

// Dest is the end of the plan's last move, for the map, or nil.
func (d *Draft) Dest() *pb.Pos {
	for i := len(d.Steps) - 1; i >= 0; i-- {
		if d.Steps[i].Kind == Move {
			return &pb.Pos{X: d.Steps[i].X, Y: d.Steps[i].Y}
		}
	}
	return nil
}

// target is the target the plan would leave selected.
func (d *Draft) target(m *model.Model) string {
	for i := len(d.Steps) - 1; i >= 0; i-- {
		if d.Steps[i].Kind == Target {
			return d.Steps[i].ID
		}
	}
	return m.Self.Target
}

// attack is whether the plan would leave auto-attack on.
func (d *Draft) attack(m *model.Model) bool {
	for i := len(d.Steps) - 1; i >= 0; i-- {
		if d.Steps[i].Kind == Attack {
			return d.Steps[i].On
		}
	}
	return m.Self.AutoAttack
}

// Slots are the slot changes the plan holds, slot (1 to 8) to the ability it
// will hold ("" emptied); the last step for a slot wins.
func (d *Draft) Slots() map[int]string {
	var out map[int]string
	for _, s := range d.Steps {
		if s.Kind == SetSlot {
			if out == nil {
				out = map[int]string{}
			}
			out[s.Slot] = s.ID
		}
	}
	return out
}

// Lines is the plan as the words the Round panel lists.
func (d *Draft) Lines(m *model.Model) []string {
	out := make([]string, len(d.Steps))
	for i, s := range d.Steps {
		out[i] = stepText(s, m)
	}
	return out
}

func stepText(s Action, m *model.Model) string {
	switch s.Kind {
	case Move:
		return fmt.Sprintf("move (%d,%d)", s.X, s.Y)
	case Target:
		if s.ID == "" {
			return "clear target"
		}
		return "target " + nameOf(m, s.ID)
	case Attack:
		if s.On {
			return "attack on"
		}
		return "attack off"
	case Loot:
		if d := m.Drops[s.ID]; d != nil {
			return "loot " + d.GetItem()
		}
		return "loot " + s.ID
	case UseDoor:
		return "use door " + s.ID
	case Stop:
		return "stop"
	case Release:
		return "release"
	case UseAbility:
		return "use " + slotWords(m, s.Slot)
	case CompanionStance, CompanionOrder:
		return companionStepText(s, m)
	case SetSlot:
		if s.ID == "" {
			return fmt.Sprintf("empty key %d", s.Slot)
		}
		return fmt.Sprintf("key %d: %s", s.Slot, m.AbilityName(s.ID))
	}
	return "?"
}

// slotWords is "Flame Dart (key 1)", or "key 1" for an empty slot.
func slotWords(m *model.Model, n int) string {
	if n < 1 || n > model.Slots {
		return fmt.Sprintf("key %d", n)
	}
	if id := m.Self.Slots[n-1].Ability; id != "" {
		return fmt.Sprintf("%s (key %d)", m.AbilityName(id), n)
	}
	return fmt.Sprintf("key %d (empty)", n)
}

func nameOf(m *model.Model, id string) string {
	if e := m.Entities[id]; e != nil {
		return e.GetName()
	}
	if g := m.Group[id]; g != nil {
		return g.Entity.GetName()
	}
	return id
}

// Agent turns the steps into the toolkit's plan steps.
func (d *Draft) Agent() []agentkit.Step {
	out := make([]agentkit.Step, 0, len(d.Steps))
	for _, s := range d.Steps {
		switch s.Kind {
		case Move:
			out = append(out, agentkit.StepMove(s.X, s.Y))
		case Target:
			out = append(out, agentkit.StepTarget(s.ID))
		case Attack:
			out = append(out, agentkit.StepAttack(s.On))
		case Loot:
			out = append(out, agentkit.StepLoot(s.ID))
		case UseDoor:
			out = append(out, agentkit.StepUseDoor(s.ID))
		case Stop:
			out = append(out, agentkit.StepStop())
		case Release:
			out = append(out, agentkit.StepRelease())
		case UseAbility:
			out = append(out, agentkit.StepUseAbility(s.Slot, s.ID))
		case SetSlot:
			out = append(out, agentkit.StepSetSlot(s.Slot, s.ID))
		case CompanionStance:
			out = append(out, agentkit.StepCompanionStance(s.ID))
		case CompanionOrder:
			var dest *pb.Pos
			if s.ID == "move" {
				dest = &pb.Pos{X: s.X, Y: s.Y}
			}
			out = append(out, agentkit.StepCompanionOrder(s.ID, s.Target, dest, 0))
		}
	}
	return out
}

// TurnKey decides what a key does in a turn-based instance, where bare intents
// are refused and the keys build the plan (D30). It is pure.
func TurnKey(key string, m *model.Model, d *Draft) Action {
	pos := d.Pos(m)
	step := func(a Action) Action { return Action{Kind: PlanAdd, Step: &a} }
	switch key {
	case "up", "w":
		return step(Action{Kind: Move, X: pos.GetX(), Y: pos.GetY() + StepMm})
	case "down", "s":
		return step(Action{Kind: Move, X: pos.GetX(), Y: pos.GetY() - StepMm})
	case "left", "a":
		return step(Action{Kind: Move, X: pos.GetX() - StepMm, Y: pos.GetY()})
	case "right", "d":
		return step(Action{Kind: Move, X: pos.GetX() + StepMm, Y: pos.GetY()})
	case "tab":
		if id := nextTargetFrom(m, d.target(m)); id != "" {
			return step(Action{Kind: Target, ID: id})
		}
		return Action{Note: "no target in view"}
	case "f", "space", " ":
		return step(Action{Kind: Attack, On: !d.attack(m)})
	case "l":
		a := lootAction(m, pos)
		if a.Kind == Loot {
			return step(a)
		}
		return a
	case "e":
		if id := nearestDoorTo(m, pos); id != "" {
			return step(Action{Kind: UseDoor, ID: id})
		}
		return Action{Note: "no door on this map"}
	case "x":
		return step(Action{Kind: Stop})
	case "r":
		return step(Action{Kind: Release})
	case "backspace":
		return Action{Kind: PlanUndo}
	case "esc":
		return Action{Kind: PlanClear}
	case "enter":
		return Action{Kind: PlanSubmit}
	}
	if n := slotKey(key); n > 0 {
		return step(Action{Kind: UseAbility, Slot: n})
	}
	return common(key)
}
