package scenario_test

import (
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/core"
	"github.com/GeneStevens/seq/testserver"
)

// learnAt changes the level a class gets an ability at, in the test server's
// copy of the zone (a class schedule is data). It lets a scenario that starts a
// character at level 1 have the ability it is about, with no hours of play.
func learnAt(class, ability string, level int) func(string, *core.Zone) {
	return func(_ string, z *core.Zone) {
		for i := range z.Classes {
			if z.Classes[i].ID != class {
				continue
			}
			for j := range z.Classes[i].Abilities {
				if z.Classes[i].Abilities[j].Ability == ability {
					z.Classes[i].Abilities[j].Level = level
				}
			}
		}
	}
}

// tough gives every mob of the zone this many hit points, so a fight lasts.
func tough(hp int) func(string, *core.Zone) {
	return func(_ string, z *core.Zone) {
		for i := range z.MobTemplates {
			z.MobTemplates[i].Stats.MaxHP = hp
		}
	}
}

func both(fs ...func(string, *core.Zone)) func(string, *core.Zone) {
	return func(s string, z *core.Zone) {
		for _, f := range fs {
			f(s, z)
		}
	}
}

func mana(p *scenario.Player) (n int64) {
	p.Model(func(m *model.Model) { n = m.Self.Mana })
	return
}

func slotsOf(m *model.Model) string {
	var out []string
	for _, s := range m.Self.Slots {
		out = append(out, s.Ability)
	}
	return strings.Join(out, ",")
}

func hasLog(m *model.Model, kind, contains string) bool {
	for _, e := range m.Log {
		if e.Kind == kind && strings.Contains(e.Text, contains) {
			return true
		}
	}
	return false
}

// Make a Wizard with the creation screen, put Root on key 1 and Flame Dart on
// key 3 with the slot screen, root the scuttler and burn it down, watching the
// mana drop and come back.
func TestWizardRootsAndNukes(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{{Name: "ann"}},
		Zone: learnAt("wizard", "wizard-root", 1)})
	w.Fixture("wizard.jsonl")
	p := w.NewCharacter("ann")
	p.Golden("create_name")
	p.Type("Zedd")
	p.Press("tab", "4")
	p.Golden("create_class")
	p.Press("enter")
	p.Created()
	p.Model(func(m *model.Model) {
		if m.Self.ID != "zedd" || m.Self.Class != "wizard" || m.Self.Level != 1 || m.Self.MaxMana != 150 || slotsOf(m) != "wizard-flame-dart,wizard-frost-bolt,wizard-root,,,,," {
			t.Fatalf("a new wizard: %+v slots %s", m.Self, slotsOf(m))
		}
	})
	p.Golden("kit_new")

	// B opens the slot screen: key 1 gets Root, key 3 gets Flame Dart.
	p.Press("b", "enter", "down", "down")
	p.Golden("slots_pick")
	p.Press("enter", "3", "enter", "up", "up", "enter")
	w.Advance(200 * time.Millisecond)
	p.Model(func(m *model.Model) {
		if slotsOf(m) != "wizard-root,wizard-frost-bolt,wizard-flame-dart,,,,," {
			t.Fatalf("slots %s", slotsOf(m))
		}
	})
	p.Golden("slots_set")
	p.Press("esc")
	p.Golden("kit_slotted")

	// Tab picks the scuttler; key 1 casts Root. Half way through, the bar shows the cast.
	p.Press("tab", "1")
	notice(t, p, "key 1 Root ok", false)
	w.Advance(800 * time.Millisecond)
	p.Model(func(m *model.Model) {
		if c := m.Self.Cast; c == nil || c.Ability != "wizard-root" {
			t.Fatalf("cast %+v", c)
		}
	})
	p.Golden("cast_root")
	w.Advance(800 * time.Millisecond)
	start := mana(p)
	p.Model(func(m *model.Model) {
		fx := m.EffectsOn(m.Self.Target)
		if m.Self.Cast != nil || len(fx) != 1 || fx[0].Kind != "root" || start != 140 {
			t.Fatalf("after the cast: cast %+v, effects %+v, mana %d", m.Self.Cast, fx, start)
		}
	})
	p.Golden("rooted")

	// Flame Dart, then Frost Bolt again and again, each after the global
	// cooldown, until the held mob is dead. Spells always land (D50): what the
	// mob loses is the spell's, less its resist.
	low := mana(p)
	for i := 0; i < 8; i++ {
		key := "2"
		if i == 0 {
			key = "3"
		}
		p.Press(key)
		w.Advance(2500 * time.Millisecond)
		low = mana(p)
		var dead bool
		p.Model(func(m *model.Model) { dead = m.Self.Target == "" })
		if dead {
			break
		}
	}
	p.Model(func(m *model.Model) {
		if low >= start || m.Self.XP != 20 || !hasLog(m, "combat_death", "Zedd killed Plain Scuttler") || m.Self.Target != "" {
			t.Fatalf("after the kill: mana %d (was %d), xp %d, target %q", low, start, m.Self.XP, m.Self.Target)
		}
	})
	p.Golden("nuked")
	w.Advance(30 * time.Second)
	if back := mana(p); back <= low || back > 150 {
		t.Fatalf("mana did not come back: %d after %d", back, low)
	}
	p.Golden("mana_back")
}

// A cast is cancelled by moving (nothing is spent), and another is broken by
// the damage of the scuttler it was aimed at.
func TestCastCancelledByMovingAndInterruptedByDamage(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{{Name: "ann", Classed: map[string]string{"zedd": "wizard"}}},
		Zone: tough(400)})
	w.Fixture("casts.jsonl")
	p := w.Login("ann", "zedd")

	// Frost Bolt takes 2 s; a step half way through cancels it.
	p.Press("tab", "2")
	w.Advance(500 * time.Millisecond)
	p.Press("left")
	w.Advance(200 * time.Millisecond)
	p.Model(func(m *model.Model) {
		e := m.Self.LastCastEnd
		if m.Self.Cast != nil || e == nil || e.Broken != "cancelled" || e.Reason != "moved" || m.Self.Mana != 150 {
			t.Fatalf("after moving: cast %+v, ended %+v, mana %d", m.Self.Cast, e, m.Self.Mana)
		}
		if !hasLog(m, "cast_cancelled", "Frost Bolt was cancelled: you moved") {
			t.Fatalf("the log ends %+v", m.Log[len(m.Log)-4:])
		}
	})
	p.Golden("cast_moved")

	// A cast that lands starts the fight, and the scuttler comes. Cast again and
	// again until its damage breaks one; the roll is the server's, seeded, so the run is the same each time.
	broken := false
	for range 8 {
		w.Advance(1600 * time.Millisecond) // past the global cooldown
		p.Press("2")
		w.Advance(2300 * time.Millisecond)
		p.Model(func(m *model.Model) { broken = m.Self.LastCastEnd != nil && m.Self.LastCastEnd.Broken == "interrupted" })
		if broken {
			break
		}
	}
	p.Model(func(m *model.Model) {
		e := m.Self.LastCastEnd
		if !broken || e.Reason != "damage" || m.Self.Cast != nil {
			t.Fatalf("no cast was interrupted by damage: ended %+v", e)
		}
		if !hasLog(m, "cast_interrupted", "Frost Bolt was interrupted: damage broke it") {
			t.Fatalf("the log ends %+v", m.Log[len(m.Log)-4:])
		}
	})
	p.Golden("cast_interrupted")
}

// A Warrior taunts the scuttler from across the field, takes its blows, binds
// his wounds with Second Wind, bashes it and cuts it down.
func TestWarriorTauntsHealsAndKills(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{{Name: "ann", Classed: map[string]string{"gorm": "warrior"}}},
		Zone: learnAt("warrior", "warrior-second-wind", 1)})
	w.Fixture("warrior.jsonl")
	p := w.Login("ann", "gorm")
	p.Model(func(m *model.Model) {
		if m.Self.Class != "warrior" || m.Self.MaxMana != 0 || slotsOf(m) != "warrior-taunt,warrior-bash,warrior-second-wind,,,,," {
			t.Fatalf("a new warrior: %+v slots %s", m.Self, slotsOf(m))
		}
	})
	p.Golden("warrior_new")

	// Taunt reaches 20 m: the scuttler, 13 m off, turns on the warrior.
	p.Press("tab", "1")
	notice(t, p, "key 1 Taunt ok", false)
	w.Advance(500 * time.Millisecond)
	p.Golden("warrior_taunt")
	w.Advance(6 * time.Second)
	p.Model(func(m *model.Model) {
		if m.Self.HP >= m.Self.MaxHP || !hasLog(m, "ability_used", "gorm used Taunt on Plain Scuttler") {
			t.Fatalf("hp %d; log ends %+v", m.Self.HP, m.Log[len(m.Log)-4:])
		}
	})

	// Second Wind: for twelve seconds a share of what he deals comes back as
	// health, and ninety seconds pass before the next. F swings at the scuttler.
	p.Press("f", "3")
	notice(t, p, "key 3 Second Wind ok", false)
	w.Advance(1200 * time.Millisecond)
	p.Model(func(m *model.Model) {
		g, left := m.Slot(3, m.GameTimeMs)
		fx := m.Self.Effects
		if len(fx) != 1 || fx[0].Kind != "lifesteal" || g != model.SlotCooldown || left < 85_000 || !m.Self.AutoAttack {
			t.Fatalf("after Second Wind: effects %+v, slot %v %d ms, auto-attack %v", fx, g, left, m.Self.AutoAttack)
		}
	})
	p.Golden("warrior_second_wind")

	// Bash once the global cooldown is over, and the scuttler dies.
	w.Advance(time.Second)
	p.Press("2")
	notice(t, p, "key 2 Bash ok", false)
	w.Advance(60 * time.Second)
	p.Model(func(m *model.Model) {
		if m.Self.XP != 20 || m.Self.Target != "" || !hasLog(m, "combat_death", "gorm killed Plain Scuttler") || !hasLog(m, "ability_used", "gorm used Bash") {
			t.Fatalf("after the fight: xp %d target %q", m.Self.XP, m.Self.Target)
		}
	})
	p.Golden("warrior_killed")
}

// worth makes every mob of the zone give this much XP.
func worth(xp int) func(string, *core.Zone) {
	return func(_ string, z *core.Zone) {
		for i := range z.MobTemplates {
			z.MobTemplates[i].Stats.XPValue = xp
		}
	}
}

// A kill that is worth a level: the log says so. A level no longer grants
// abilities (SEQ-040): they are bought from the class trainer, so the kit is the
// same and the trainer's scenario is where the new one is learned.
func TestLevelUpIsInTheLog(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{{Name: "ann", Classed: map[string]string{"zedd": "wizard"}}},
		Zone: worth(60)})
	w.Fixture("levelup.jsonl")
	p := w.Login("ann", "zedd")
	p.Model(func(m *model.Model) {
		if m.Self.Level != 1 || m.Self.XPNext != 40 {
			t.Fatalf("a new wizard: level %d, xp to %d", m.Self.Level, m.Self.XPNext)
		}
	})
	// Frost Bolt until it is dead; the scuttler's blows may break a cast, and
	// then the wizard casts again.
	p.Press("tab")
	for range 10 {
		var xp int64
		p.Model(func(m *model.Model) { xp = m.Self.XP })
		if xp > 0 {
			break
		}
		p.Press("2")
		w.Advance(2500 * time.Millisecond)
	}
	p.Model(func(m *model.Model) {
		if m.Self.Level != 2 || m.Self.XP != 60 || len(m.Self.Abilities) != 2 {
			t.Fatalf("after the kill: level %d, xp %d, slots %s", m.Self.Level, m.Self.XP, slotsOf(m))
		}
		if !hasLog(m, "level_up", "you reached level 2") || hasLog(m, "level_up", "new ability") {
			t.Fatalf("the level-up line: %+v", m.Log[len(m.Log)-4:])
		}
	})
	p.Golden("levelup")
}

// atGate starts new characters at the gate of Testgrounds, where its instances
// are entered.
func atGate(_ string, z *core.Zone) { z.Start = core.Pos{X: gateX, Y: gateY} }

// A Wizard in quick-turns plans a cast: a target and Frost Bolt on key 2. The
// plan holds one ability, so a second is refused with its reason; the cast
// plays through the round, and lands in the next.
func TestTurnBasedRoundWithACastInThePlan(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{{Name: "ann", Classed: map[string]string{"zedd": "wizard"}}},
		Zone: atGate})
	w.Fixture("turn_cast.jsonl")
	p := w.Login("ann", "zedd")
	openPreset(t, p, "quick-turns", true)
	w.Advance(time.Second)
	p.Press("e")
	p.Eventually("round 1", openRound)

	p.Press("tab", "2")
	if got := p.Plan(); len(got) != 2 || got[0] != "target Plain Scuttler" || got[1] != "use Frost Bolt (key 2)" {
		t.Fatalf("plan %q", got)
	}
	// One ability or attack toggle a plan, and one move: the rest are refused and say why.
	p.Press("1")
	notice(t, p, "a plan holds one ability or attack toggle", true)
	p.Press("f")
	notice(t, p, "a plan holds one ability or attack toggle", true)
	p.Press("down", "left")
	notice(t, p, "a plan holds one move or door", true)
	if got := p.Plan(); len(got) != 3 || got[2] != "move (100000,87000)" {
		t.Fatalf("plan %q", got)
	}
	// A move would cancel the cast: take it out again.
	p.Press("backspace")
	p.Golden("turn_cast_plan")

	p.Press("enter")
	notice(t, p, "plan accepted", false)
	w.Advance(100 * time.Millisecond)
	p.Model(func(m *model.Model) {
		r := m.Round
		if r.Number != 2 || r.Resolved != 1 || r.PlayedBy("zedd") != "own_plan" {
			t.Fatalf("round %+v", r)
		}
		if !hasLog(m, "cast_started", "zedd began casting Frost Bolt") {
			t.Fatalf("log ends %+v", m.Log[len(m.Log)-5:])
		}
	})
	p.Golden("turn_cast_round")
}

// The creation screen: a name too short is not sent, a taken name is refused by
// the server with its reason and the screen stays, and another name goes through.
func TestCreationRefusals(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{{Name: "ann"}}})
	first := w.NewCharacter("ann")
	first.Type("Zedd")
	first.Press("tab", "4", "enter")
	first.Created()

	p := w.NewCharacter("ann")
	p.Type("Ab")
	p.Press("enter")
	if s := p.Screen(80, 24); !strings.Contains(s, "! a name is 3 to 16 letters") {
		t.Fatalf("a short name:\n%s", s)
	}
	p.Press("backspace", "backspace")
	p.Type("zEDD")
	p.Press("tab", "2", "enter")
	if s := p.Screen(80, 24); !strings.Contains(s, "! someone already has that name (name_taken)") || !strings.Contains(s, "Name   zEDD") {
		t.Fatalf("a taken name:\n%s", s)
	}
	p.Golden("create_refused")
	p.Press("tab", "backspace", "backspace", "backspace", "backspace")
	p.Type("Mira")
	p.Press("tab", "6", "enter")
	p.Created()
	p.Model(func(m *model.Model) {
		if m.Self.ID != "mira" || m.Self.Class != "druid" || !strings.Contains(strings.ToLower(slotsOf(m)), "druid-thorn-bolt") {
			t.Fatalf("the second character: %+v", m.Self)
		}
	})
}
