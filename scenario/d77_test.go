package scenario_test

import (
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
	"github.com/GeneStevens/seq/testserver"
)

// D77, Diablo-style mouse play: the mouse events go to the real program as the
// SGR reports a terminal sends, and the scenario reads what the server and the
// screen did. Every mouse action also has a key (see TestEveryMouseActionHasAKey).

// mobIn is a live mob in view.
func mobIn(t *testing.T, p *scenario.Player) *pb.EntityState {
	t.Helper()
	var mob *pb.EntityState
	p.Model(func(m *model.Model) {
		for _, id := range m.EntityIDs() {
			if e := m.Entities[id]; e.GetKind() == pb.EntityKind_ENTITY_KIND_MOB && !e.GetDead() {
				mob = e
				return
			}
		}
	})
	if mob == nil {
		t.Fatal("no mob in view")
	}
	return mob
}

// mapOf is the map part of a 140-column screen: what is left of the panels.
func mapOf(s string) string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if r := []rune(l); len(r) > 98 {
			l = string(r[:98])
		}
		out = append(out, l)
	}
	return strings.Join(out, "\n")
}

func at(t *testing.T, p *scenario.Player, pos *pb.Pos) (x, y int) {
	t.Helper()
	x, y, ok := p.Where(pos)
	if !ok {
		t.Fatalf("the map does not show %v", pos)
	}
	return x, y
}

// Click the ground to walk; the first click on a mob selects it and opens its
// card and never attacks; the second attacks; hover names it; Alt names
// everything; the wheel zooms.
func TestMouseWalkSelectAttackHoverAndZoom(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria")})
	w.Place("ann", "aria", 84000, 50000) // 16 m from the trophy scuttler: in view, not yet roused
	p := w.Login("ann", "aria")
	p.Resize(140, 45)

	// Click the ground ten metres west, away from the mob: the program sends a move
	// to that point.
	var from, dest *pb.Pos
	p.Model(func(m *model.Model) { from = m.Self.Pos })
	dest = &pb.Pos{X: from.GetX() - 10000, Y: from.GetY()}
	x, y := at(t, p, dest)
	p.Click(x, y)
	notice(t, p, "move ok", false)
	for range 12 {
		w.Advance(time.Second)
	}
	p.Model(func(m *model.Model) {
		if m.Self.Pos.GetX() > from.GetX()-8000 {
			t.Errorf("the click did not walk the character west: %v to %v", from, m.Self.Pos)
		}
	})

	// The first click on the mob selects it and opens its card; nothing attacks.
	mob := mobIn(t, p)
	x, y = at(t, p, mob.GetPos())
	p.Click(x, y)
	p.Model(func(m *model.Model) {
		if m.Self.Target != mob.GetId() || m.Self.AutoAttack {
			t.Errorf("a first click selects and never attacks: target %q, attack %v", m.Self.Target, m.Self.AutoAttack)
		}
	})
	if _, _, card, _, _, _ := p.Mouse(); card != mob.GetId() {
		t.Errorf("the card is %q, want the mob's", card)
	}
	s := p.Screen(140, 45)
	for _, want := range []string{mob.GetName(), "Level", "Click again or F: attack"} {
		if !strings.Contains(s, want) {
			t.Errorf("the info card lacks %q:\n%s", want, s)
		}
	}
	p.Golden("mouse_info_card")

	// The second click attacks.
	p.Click(x, y)
	p.Model(func(m *model.Model) {
		if !m.Self.AutoAttack {
			t.Error("a second click on the selected mob attacks it")
		}
	})
	p.Press("f") // F turns it off again: the key does what the click did
	p.Press("esc")
	if _, _, card, _, _, _ := p.Mouse(); card != "" {
		t.Errorf("Esc leaves the card open: %q", card)
	}

	// Hovering names what is under the pointer, by it.
	label := func() bool { return strings.Contains(mapOf(p.Screen(140, 45)), mob.GetName()+"  Lv") }
	p.Hover(x, y, false)
	if !label() {
		t.Errorf("hovering the mob shows no label:\n%s", p.Screen(140, 45))
	}
	p.Hover(x, y+3, false)
	if label() {
		t.Error("the label stays after the pointer moved away")
	}

	// Alt held labels everything; letting go takes the labels away.
	p.Hover(x, y+3, true)
	if !label() {
		t.Errorf("Alt over the map does not name everything:\n%s", p.Screen(140, 45))
	}
	p.Golden("mouse_alt_labels")
	p.Hover(x, y+3, false)
	if label() {
		t.Error("the labels stay after Alt is let go")
	}
	// Where a terminal sends no Alt with the mouse, Ctrl+A is the toggle.
	p.Press("ctrl+a")
	if !label() {
		t.Error("Ctrl+A does not label everything")
	}
	p.Press("ctrl+a")

	// The wheel zooms the map: a step in shows less ground, so the mob moves
	// away from the character on the screen.
	before := p.Screen(140, 45)
	p.Wheel(x, y, true)
	if z, _, _, _, _, _ := p.Mouse(); z != 1 {
		t.Errorf("the wheel up zoomed to %d, want 1", z)
	}
	if p.Screen(140, 45) == before {
		t.Error("zooming changed nothing on the map")
	}
	p.Wheel(x, y, false)
	p.Wheel(x, y, false)
	if z, _, _, _, _, _ := p.Mouse(); z != -1 {
		t.Errorf("two steps out from 1 gave %d, want -1", z)
	}
	p.Press("+")
	if z, _, _, _, _, _ := p.Mouse(); z != 0 {
		t.Errorf("the + key gave zoom %d, want 0", z)
	}
}

// npcNamed is the NPC the character has seen with that name.
func npcNamed(t *testing.T, p *scenario.Player, name string) *pb.NpcInfo {
	t.Helper()
	var npc *pb.NpcInfo
	p.Model(func(m *model.Model) {
		for _, n := range m.KnownNPCs() {
			if n.GetName() == name {
				npc = n
			}
		}
	})
	if npc == nil {
		t.Fatalf("no NPC called %q", name)
	}
	return npc
}

// In Thornwick: hover names a person with the title, the first click on them opens
// the card and nothing else, the second click talks; M opens the large map with
// the places' names, and a click on it, or the keyboard cursor and Enter, walks.
func TestMouseNPCAndTheLargeMap(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: []testserver.Account{{Name: "ann", Classed: map[string]string{"hale": "warrior"}}}})
	p := w.Login("ann", "hale")
	p.Resize(140, 45)

	tamsin := npcNamed(t, p, "Tamsin Hale")
	x, y := at(t, p, tamsin.GetPos())
	p.Hover(x, y, false)
	if s := mapOf(p.Screen(140, 45)); !strings.Contains(s, model.NPCLabel(tamsin)) {
		t.Errorf("hovering a person names them with their title:\n%s", s)
	}
	p.Click(x, y)
	if _, _, card, _, _, _ := p.Mouse(); card != tamsin.GetId() {
		t.Errorf("the card is %q, want Tamsin's", card)
	}
	s := p.Screen(140, 45)
	for _, want := range []string{"Offers:", "new for you", "Click again or G: talk"} {
		if !strings.Contains(mapOf(s), want) {
			t.Errorf("the NPC card lacks %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "Talking") {
		t.Error("a first click talked")
	}
	p.Golden("mouse_npc_card")
	p.Click(x, y)
	if s := p.Screen(140, 45); !strings.Contains(s, "Talking") {
		t.Errorf("the second click does not open the conversation:\n%s", s)
	}
	p.Press("esc")

	// M opens the large map: the legend, the places' names, the vitals strip.
	p.Press("m")
	s = p.Screen(140, 45)
	for _, want := range []string{"Zone overview", "Hawthorn Square", "T trainer", "HP "} {
		if !strings.Contains(s, want) {
			t.Errorf("the large map lacks %q:\n%s", want, s)
		}
	}
	p.Golden("large_map")

	// Click a spot on it: the character walks there.
	var from *pb.Pos
	p.Model(func(m *model.Model) { from = m.Self.Pos })
	dest := &pb.Pos{X: from.GetX() - 9000, Y: from.GetY()}
	x, y = at(t, p, dest)
	p.Click(x, y)
	notice(t, p, "move ok", false)
	for range 6 {
		w.Advance(time.Second)
	}
	p.Model(func(m *model.Model) {
		if m.Self.Pos.GetX() > from.GetX()-6000 {
			t.Errorf("the click on the large map did not walk west: %v to %v", from, m.Self.Pos)
		}
		from = m.Self.Pos
	})
	// The same with the keyboard: the cursor moves a cell at a time, Enter walks.
	p.Press("right", "right", "right", "right", "right", "right", "right", "right")
	if s := p.Screen(140, 45); !strings.Contains(s, "+") {
		t.Error("the cursor is not shown")
	}
	p.Press("enter")
	notice(t, p, "move ok", false)
	for range 6 {
		w.Advance(time.Second)
	}
	p.Model(func(m *model.Model) {
		if m.Self.Pos.GetX() <= from.GetX()+3000 {
			t.Errorf("the cursor and Enter did not walk east: %v to %v", from, m.Self.Pos)
		}
	})
	// The wheel zooms the large map in on the character.
	p.Wheel(x, y, true)
	if _, z, _, _, _, _ := p.Mouse(); z != 1 {
		t.Errorf("the large map's zoom is %d, want 1", z)
	}
	p.Press("m")
	if s := p.Screen(140, 45); strings.Contains(s, "Zone overview") {
		t.Error("M did not close the large map")
	}
}

// Click loot: one click on a drop takes everything within reach, and the log says
// so in one line.
func TestClickingLootTakesAllOfIt(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{{Name: "ann", Characters: []string{"c1"}}}})
	w.Place("ann", "c1", 98000, 50000)
	w.Srv.Attend("ann")
	p := w.Login("ann", "c1")
	p.Resize(140, 45)
	p.Press("tab", "f")
	for range 60 {
		var drops int
		p.Model(func(m *model.Model) { drops = len(m.Drops) })
		if drops >= 3 {
			break
		}
		w.Advance(time.Second)
	}
	var drop *pb.LootDropState
	p.Model(func(m *model.Model) {
		if len(m.Drops) < 3 {
			t.Fatalf("%d drops", len(m.Drops))
		}
		drop = m.Drops[m.DropIDs()[0]]
	})
	x, y := at(t, p, drop.GetPos())
	p.Hover(x, y, false)
	if s := mapOf(p.Screen(140, 45)); !strings.Contains(s, "Scuttler Blade") && !strings.Contains(s, "Worn Cap") && !strings.Contains(s, "crowns") {
		t.Errorf("hovering loot names it:\n%s", s)
	}
	p.Click(x, y)
	p.Model(func(m *model.Model) {
		if len(m.Drops) != 0 || len(m.Self.Inventory) != 2 {
			t.Errorf("one click on loot should take all of it: %d drops left, bags %v", len(m.Drops), m.Self.Inventory)
		}
		n := 0
		for _, e := range m.Log {
			if e.Kind == "reward" && strings.HasPrefix(e.Text, "You took ") {
				n++
				for _, want := range []string{"Worn Cap", "Scuttler Blade", "5 crowns"} {
					if !strings.Contains(e.Text, want) {
						t.Errorf("the loot-all line %q lacks %q", e.Text, want)
					}
				}
			}
		}
		if n != 1 {
			t.Errorf("the log has %d loot-all lines, want 1: %+v", n, m.Log)
		}
	})
}
