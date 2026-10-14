package model

import (
	"fmt"
	"strings"
)

// Text renders the model as plain text: what cmd/seq-tui prints until the
// screen exists. It reads the model and changes nothing.
func (m *Model) Text() string {
	var b strings.Builder
	s := m.Self
	fmt.Fprintf(&b, "space %s  tick %d  game time %d ms  stream %d  connections %d\n", m.Space, m.Tick, m.GameTimeMs, m.last, m.Connections)
	fmt.Fprintf(&b, "you: %s (%s) level %d  hp %d/%d  xp %d  at (%d, %d)  target %q  auto-attack %v\n",
		s.Name, s.ID, s.Level, s.HP, s.MaxHP, s.XP, s.Pos.GetX(), s.Pos.GetY(), s.Target, s.AutoAttack)
	if len(s.Inventory) > 0 {
		fmt.Fprintf(&b, "inventory: %s\n", strings.Join(s.Inventory, ", "))
	}
	if m.Map != nil {
		fmt.Fprintf(&b, "map: %s, %d walls, %d doors, view radius %d mm\n", m.Map.GetName(), len(m.Map.GetWalls()), len(m.Map.GetDoors()), m.Map.GetViewRadiusMm())
		for _, d := range m.Map.GetDoors() {
			fmt.Fprintf(&b, "  door %s (%s) at (%d, %d)\n", d.GetId(), d.GetKind(), d.GetPos().GetX(), d.GetPos().GetY())
		}
	}
	for _, id := range m.GroupIDs() {
		g := m.Group[id]
		fmt.Fprintf(&b, "group: %s hp %d/%d target %q\n", id, g.Entity.GetHp(), g.Entity.GetMaxHp(), g.Target)
	}
	fmt.Fprintf(&b, "in view (%d):\n", len(m.Entities))
	for _, id := range m.EntityIDs() {
		e := m.Entities[id]
		dead := ""
		if e.GetDead() {
			dead = " dead"
		}
		fmt.Fprintf(&b, "  %s %s hp %d/%d at (%d, %d)%s\n", id, e.GetName(), e.GetHp(), e.GetMaxHp(), e.GetPos().GetX(), e.GetPos().GetY(), dead)
	}
	for _, id := range m.DropIDs() {
		d := m.Drops[id]
		fmt.Fprintf(&b, "drop: %s %s at (%d, %d)\n", id, d.GetItem(), d.GetPos().GetX(), d.GetPos().GetY())
	}
	for _, id := range m.EncounterIDs() {
		e := m.Encounters[id]
		fmt.Fprintf(&b, "encounter: %s %s owned by %s\n", id, e.GetMob(), e.GetOwner())
	}
	fmt.Fprintf(&b, "intents (%d):\n", len(m.Intents))
	for _, i := range m.Intents {
		fmt.Fprintf(&b, "  %s %s %s %s\n", i.ID, i.Kind, i.State, i.Reason)
	}
	fmt.Fprintf(&b, "events (%d):\n", len(m.Log))
	for _, e := range m.Log {
		fmt.Fprintf(&b, "  %d %s\n", e.Seq, e.Text)
	}
	for _, e := range m.Errors {
		fmt.Fprintf(&b, "error: %s: %s\n", e.GetCode(), e.GetMessage())
	}
	return b.String()
}
