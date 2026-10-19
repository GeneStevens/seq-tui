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

// The mobs of D70 on the pinned Thornwick packs (SEQ-072, WLD-010): the TUI
// shows each mob's difficulty band and says in the log what the mobs do. The
// server sends the band (con) and the ordinary events; the words are the TUI's.

// The camps of the Downs the scenarios use (seq-world's zones.yaml), by id.
var d70Camps = map[string]core.Pos{
	"goblin-scrappers-a": {X: 245000, Y: 105000},
	"goblin-hexers-a":    {X: 355000, Y: 90000},
}

// d70Hero logs in a warrior of the level standing dx mm east of the camp's
// first mob, who is in view.
func d70Hero(t *testing.T, level int, camp string, dx int, edit func(*core.CharacterSave)) (*scenario.World, *scenario.Player, string) {
	t.Helper()
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: []testserver.Account{{Name: "ann", Home: scenario.Downs, Classed: map[string]string{"hale": "warrior"}}}})
	at := d70Camps[camp]
	w.Place("ann", "hale", at.X+dx, at.Y)
	w.Srv.Give("hale", levelUp(level, edit))
	return w, w.Login("ann", "hale"), string(core.MobID(camp, 1))
}

// heavy gives a character the heavy hand the delvers have, so a fight the
// scenario watches is over in game seconds.
func heavy(cs *core.CharacterSave) {
	cs.Stats.WeaponMin += 8
	cs.Stats.WeaponMax += 10
}

// A mob in view carries its band: Nearby and the Target panel say it in words.
func TestAMobInViewShowsItsDifficultyBand(t *testing.T) {
	w, p, id := d70Hero(t, 8, "goblin-scrappers-a", 6000, nil)
	_ = w
	p.Model(func(m *model.Model) {
		e := m.Entities[id]
		if e == nil || e.GetCon() != core.Con(int(e.GetLevel()), 8) || e.GetCon() == "" {
			t.Fatalf("the scrapper in view: %v", e)
		}
	})
	p.Press("tab")
	s := p.Screen(140, 45)
	for _, want := range []string{"easy", "Goblin Scrapper"} {
		if !strings.Contains(s, want) {
			t.Fatalf("the screen lacks %q:\n%s", want, s)
		}
	}
	p.Golden("d70_band")
}

// A goblin flees once it is hurt, and the log says so, then says it turns back.
func TestAGoblinFleesAndTurnsBackInTheLog(t *testing.T) {
	w, p, id := d70Hero(t, 8, "goblin-scrappers-a", 9000, heavy)
	p.Press("tab", "f")
	var fled, back bool
	for range 60 {
		w.Advance(time.Second)
		attacking := false
		p.Model(func(m *model.Model) {
			fled, attacking = hasLog(m, "mob_flees", "Goblin Scrapper flees"), m.Self.AutoAttack
		})
		if !attacking && !fled && !back {
			p.Press("f") // the goblin was out of reach when the swing was asked for; it has come to us
		}
		if fled {
			break
		}
	}
	if !fled {
		p.Model(func(m *model.Model) {
			for _, l := range m.Log {
				if l.Kind != "rules_tick" && !strings.HasPrefix(l.Kind, "combat_") {
					t.Logf("%d %s %s", l.GameTimeMs, l.Kind, l.Text)
				}
			}
			t.Fatal("the goblin never fled")
		})
	}
	p.Model(func(m *model.Model) {
		if m.Entities[id] == nil || m.Entities[id].GetDead() {
			t.Fatalf("the goblin is gone after it fled")
		}
	})
	for range 40 {
		w.Advance(time.Second)
		p.Model(func(m *model.Model) { back = hasLog(m, "mob_returns", "Goblin Scrapper turns back to fight") })
		if back {
			break
		}
	}
	if !back {
		p.Model(func(m *model.Model) { t.Fatalf("the goblin never turned back: %v", m.Log) })
	}
	p.Model(func(m *model.Model) {
		n := 0
		for _, l := range m.Log {
			if l.Kind == "mob_flees" {
				n++
			}
		}
		if n != 1 {
			t.Fatalf("it fled %d times; the server lets a goblin flee once", n)
		}
	})
}

// A Bash on a hexer's bolt breaks it off, and the log says whose blow did.
func TestABashBreaksOffAHexersBoltInTheLog(t *testing.T) {
	w, p, _ := d70Hero(t, 8, "goblin-hexers-a", 1500, nil)
	p.Press("tab", "f")
	var broke bool
	pressed := 0 // casts answered with a Bash
	for range 600 {
		casts := 0
		p.Model(func(m *model.Model) {
			broke = hasLog(m, "cast_cancelled", "your Bash breaks off Goblin Hexer's Cinder Bolt")
			for _, l := range m.Log {
				if l.Kind == "cast_started" && strings.Contains(l.Text, "Goblin Hexer begins casting Cinder Bolt") {
					casts++
				}
			}
		})
		if broke {
			break
		}
		if casts > pressed {
			pressed = casts
			p.Press("2") // Bash, in slot 2
		}
		w.Advance(100 * time.Millisecond)
	}
	if !broke {
		p.Model(func(m *model.Model) {
			for _, l := range m.Log {
				if l.Kind != "entity_moved" && l.Kind != "rules_tick" && !strings.HasPrefix(l.Kind, "combat_") {
					t.Logf("%s", l.Text)
				}
			}
		})
		t.Fatal("no bolt was broken off")
	}
}
