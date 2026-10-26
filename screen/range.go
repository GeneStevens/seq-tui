package screen

import (
	"fmt"
	"math"

	"github.com/GeneStevens/seq-tui/model"
)

// DefaultMeleeMm is a character's melee reach when the server has not said (2.5 m).
const DefaultMeleeMm = 2500

// MeleeMm is the character's melee reach, as the server's stats give it.
func MeleeMm(m *model.Model) int64 {
	if r := m.Self.Stats.GetMeleeRangeMm(); r > 0 {
		return int64(r)
	}
	return DefaultMeleeMm
}

// DistMm is how far the entity id stands from the character, from the positions
// the client already has; ok is false when either is not known.
func DistMm(m *model.Model, id string) (mm int64, ok bool) {
	e := m.Entities[id]
	if e == nil || e.GetPos() == nil || m.Self.Pos == nil {
		return 0, false
	}
	dx, dy := float64(e.GetPos().GetX()-m.Self.Pos.GetX()), float64(e.GetPos().GetY()-m.Self.Pos.GetY())
	return int64(math.Round(math.Hypot(dx, dy))), true
}

// AttackTooFar says auto-attack is on for a living target that stands beyond
// melee reach, and how far it is. The engine accepts the attack and waits
// silently (D75), so the client says so.
func AttackTooFar(m *model.Model) (mm int64, far bool) {
	s := &m.Self
	if !s.AutoAttack || s.Target == "" {
		return 0, false
	}
	if e := m.Entities[s.Target]; e == nil || e.GetDead() {
		return 0, false
	}
	d, ok := DistMm(m, s.Target)
	return d, ok && d > MeleeMm(m)
}

// FarWords is the line for an attack on a target out of reach, d mm away.
func FarWords(d int64) string {
	return fmt.Sprintf("Too far: %d m. Option-click the target to close in", (d+500)/1000)
}
