package scenario_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/content"
	"github.com/GeneStevens/seq/core"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
	"github.com/GeneStevens/seq/testserver"
)

// The scenarios of D55 (SEQ-054): each presses its own keys.

// A duo in turn-based play: the second character, on the assist stance with an
// agent credential, sends a stand-in plan each round, and the Round panel shows
// what it planned and, once the round has run, that it played a stand-in.
func TestDuoOnAStanceStandsInEachRound(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds},
		Accounts: []testserver.Account{{Name: "ann", Characters: []string{"aria", "bram"}}}})
	w.Fixture("duo_stance_turns.jsonl")
	w.Place("ann", "aria", gateX, gateY)
	w.Place("ann", "bram", gateX, gateY)
	p := w.LoginWithDuoAgent("ann", "aria", "bram")
	p.Press("p")
	notice(t, p, "second character in play", false)
	openPreset(t, p, "duo-turns", true)
	w.Advance(time.Second)
	p.Press("h")
	notice(t, p, "second character: stance assist", false)

	// Both walk in by the same E; the round waits for the two.
	p.Press("e")
	p.Eventually("round 1", openRound)
	p.Eventually("two in round 1", func(m *model.Model) bool { return m.Round != nil && m.Round.Participants == 2 })
	p.DuoEventually("the second character in the instance", func(m *model.Model) bool { return m.InInstance() && m.TurnBased() && m.Round != nil && m.Round.Open })

	// Round 1: Bram has planned for himself by now, and the panel says what. A
	// stand-in is not a submission, so the round still waits for one plan of two.
	p.Press("down", "tab", "f", "enter")
	notice(t, p, "plan accepted", false)
	var planned []string
	var round uint64
	p.Eventually("the second character's plan", func(m *model.Model) bool {
		planned, round = p.DuoPlan()
		return len(planned) > 0
	})
	p.Model(func(m *model.Model) {
		if round != m.Round.Number || len(m.Round.Submitted) != 1 {
			t.Fatalf("round %+v, bram planned %v for round %d", m.Round, planned, round)
		}
	})
	p.GoldenWithout("duo_stance_planned", arrival, "entered Testgrounds")

	// The deadline passes: Aria played her own plan, Bram his stand-in.
	w.Advance(7 * time.Second)
	p.Model(func(m *model.Model) {
		r := m.Round
		if r.Resolved != 1 || r.ResolvedBy != "deadline" || r.PlayedBy("aria") != "own_plan" || r.PlayedBy("bram") != "stand_in" {
			t.Fatalf("round %+v", r)
		}
	})
	p.Golden("duo_stance_ran")
}

// The status line gives the server's word on how fast the character moves
// (D55): slowed in a forest, with the percent and the speed it brings, kept only
// while the zone map says the character still stands on that ground, and fresh
// at the next snapshot.
func TestTheServersSpeedInAForestAndOnTheRoad(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: warrior(scenario.Downs), Zone: noSpawns})
	w.Place("ann", "hale", 140000, 145000) // in Greywood
	p := w.Login("ann", "hale")
	var pct int64
	p.Model(func(m *model.Model) {
		s := m.Self
		pct = s.SpeedPct
		if s.Ground != "forest" || s.SpeedPct >= 100 || s.EffectiveSpeedMmS != s.Stats.GetSpeedMmS()*s.SpeedPct/100 || m.Underfoot() != model.Forest {
			t.Fatalf("the server says %q at %d%% and %d mm/s of %d", s.Ground, s.SpeedPct, s.EffectiveSpeedMmS, s.Stats.GetSpeedMmS())
		}
	})
	if got := status(p); !strings.Contains(got, fmt.Sprintf("forest: slowed to %d%%", pct)) {
		t.Fatalf("the status line says %q", got)
	}
	p.Golden("speed_forest")

	// Out of the wood the server has not spoken again: the line names the open
	// ground and claims no speed.
	walkVia(t, w, p, [2]int64{112000, 145000})
	if got := status(p); strings.Contains(got, "slowed") || strings.Contains(got, "faster") {
		t.Fatalf("out of the forest the status line still says %q", got)
	}
	// A fresh snapshot tells the speed on the road.
	p.Fresh()
	p.Eventually("the snapshot", func(m *model.Model) bool { return m.Ready() && m.Map != nil })
	p.Model(func(m *model.Model) {
		if m.Self.Ground != "open" || m.Self.SpeedPct != 100 {
			t.Fatalf("the server says %q at %d%%", m.Self.Ground, m.Self.SpeedPct)
		}
	})
}

// A death inside an instance wakes the character at the instance's entrance, in
// the instance, not at the bind point in the zone (D55): the map and the panels
// follow.
func TestAnInstanceDeathWakesAtItsEntrance(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria"),
		Zone: func(_ string, z *core.Zone) { // a frail character, so the scuttler can kill it
			for i := range z.Characters {
				z.Characters[i].Stats.MaxHP = 6
			}
		}})
	w.Fixture("instance_death.jsonl")
	w.Place("ann", "aria", gateX, gateY)
	p := w.Login("ann", "aria")
	openPreset(t, p, "solo-trial", true)
	w.Advance(time.Second)
	p.Press("e")
	p.Eventually("the instance", inside)
	var entrance *pb.Pos
	p.Model(func(m *model.Model) { entrance = m.Self.Pos })

	// One step toward the scuttler wakes it; it kills the character.
	p.Press("down")
	advanceUntil(t, w, p, "the character to die", time.Second, 60, func(m *model.Model) bool { return m.Self.Dead })
	p.Golden("instance_death_dead")
	p.Press("r")
	notice(t, p, "release ok", false)
	advanceUntil(t, w, p, "the respawn", time.Second, 20, func(m *model.Model) bool { return !m.Self.Dead && !m.Self.Released })
	p.Eventually("the new snapshot", inside)
	p.Model(func(m *model.Model) {
		if !m.InInstance() || m.Instance.GetPreset() != "solo-trial" || m.Space == "Testgrounds" {
			t.Fatalf("after the death: space %q, instance %v", m.Space, m.Instance)
		}
		if m.Self.Pos.GetX() != entrance.GetX() || m.Self.Pos.GetY() != entrance.GetY() {
			t.Fatalf("woke at %v; the entrance is %v", m.Self.Pos, entrance)
		}
	})
	if c, _ := w.Srv.Character("aria"); c.Space == testserver.Testgrounds {
		t.Fatalf("the server has aria in %q", c.Space)
	}
	p.Golden("instance_death_entrance")
}

// Pay that waited for a join (D55): the character takes part in the Harvest
// Raid, leaves the Downs by the town gate, the raid ends without it, and when it
// comes back the log says that owed pay has arrived.
func TestOwedPayArrivesAtTheNextJoin(t *testing.T) {
	w := scenario.New(t, testserver.Options{
		Zones:    scenario.Town,
		Accounts: []testserver.Account{{Name: "ann", Home: scenario.Downs, Classed: map[string]string{"hale": "warrior"}}},
		Zone: func(space string, z *core.Zone) {
			if space != scenario.Downs {
				return
			}
			for i := range z.WorldEvents {
				z.WorldEvents[i].EveryMs, z.WorldEvents[i].OffsetMs = 20_000, 0
				z.WorldEvents[i].DurationMs = 50_000
			}
			for i := range z.Markers { // the raid comes down to the town gate, where the character stands
				switch z.Markers[i].ID {
				case "raid-edge":
					z.Markers[i].Pos = core.Pos{X: 60000, Y: 225000}
				case "barleycross":
					z.Markers[i].Pos = core.Pos{X: 12000, Y: 225000}
				}
			}
		},
	})
	w.Fixture("owed_pay.jsonl")
	w.Place("ann", "hale", 34000, 225000)
	z, _ := content.Zone(scenario.Thornwick)
	w.Srv.Give("hale", func(cs *core.CharacterSave) {
		cl := z.Class("warrior")
		cs.Stats, cs.Known, cs.Slots, cs.XP = core.ClassCharacter(cl, *z.CharacterTemplate("warrior"), 12)
		cs.Known, cs.Slots = cl.KnownAt(1), core.DefaultSlots(cl.KnownAt(1))
	})
	w.Srv.Attend("ann")
	p := w.Login("ann", "hale")
	w.Advance(21 * time.Second)

	// The character kills a raider, so it has taken part; then it leaves by the gate.
	fightUntil(t, w, p, func(m *model.Model) bool { return hasLog(m, "combat_death", "killed Raiding Goblin") }, 20)
	p.Press("f") // auto-attack off: the way out is not a fight
	walk(w, p, "left", 9)
	p.Press("e")
	p.Eventually("Thornwick", inSpace("Thornwick"))

	// The raid ends in the Downs while the character is in town.
	w.Advance(60 * time.Second)
	p.Model(func(m *model.Model) {
		if hasLog(m, "reward", "owed pay") || hasLog(m, "reward", "paid for") {
			t.Fatalf("paid before the character came back: %v", m.Log)
		}
	})
	p.Press("e")
	p.Eventually("the Downs", inSpace("Thornwick Downs"))
	p.Eventually("owed pay in the log", func(m *model.Model) bool { return hasLog(m, "reward", "owed pay arrived with the join") })
	p.Model(func(m *model.Model) {
		// Half of the 120 XP and 60 crowns: the raid was lost, and an account's pay
		// owed to an absent character is the XP and crowns only. The character is
		// at the level cap (SEQ-094), which earns no XP, so the pay is the crowns.
		if m.Self.Crowns != 30 || !hasLog(m, "reward", "owed pay arrived with the join: 30 crowns") {
			t.Fatalf("crowns %d; the log: %v", m.Self.Crowns, m.Log)
		}
	})
	p.Golden("owed_pay")
}

// The companion's mana and the party's (D55): a healer recruit and a second
// character, a cleric, each show a mana bar in their panel, from the server's
// word (the group's mana in the snapshot).
func TestCompanionAndPartyManaAreShown(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: []testserver.Account{{Name: "ann", Home: scenario.Downs,
		Classed: map[string]string{"hale": "warrior", "bram": "cleric"}}}})
	w.Fixture("companion_mana.jsonl")
	w.Place("ann", "hale", 146000, 170000)
	w.Place("ann", "bram", 144000, 170000)
	w.Srv.Give("hale", levelUp(5, nil))
	w.Srv.Give("bram", levelUpAs("cleric", 5, nil))
	w.Srv.GiveCompanion("hale", companionRecord("healer", "follow"))
	p := w.LoginWithDuo("ann", "hale", "bram")
	p.Model(func(m *model.Model) {
		cur, most := m.CompanionMana()
		if g := m.CompanionMember(); g == nil || most <= 0 || cur <= 0 || cur > most || g.MaxMana != most {
			t.Fatalf("the healer recruit's mana is %d of %d (member %+v)", cur, most, g)
		}
		if m.Self.MaxMana != 0 {
			t.Fatalf("a warrior has no mana: %d", m.Self.MaxMana)
		}
	})
	if s := p.Screen(140, 45); !strings.Contains(s, "MP ") {
		t.Fatalf("the Companion panel shows no mana:\n%s", s)
	}
	p.Golden("companion_mana")

	// Point the helper keys at the second character and bring it in.
	p.Press(".", "p")
	notice(t, p, "second character in play", false)
	w.Advance(time.Second)
	p.Model(func(m *model.Model) {
		g := m.InView("bram")
		if g == nil {
			t.Fatalf("the second character is not in view: %v", m.Entities)
		}
	})
	p.Duo(func(m *model.Model) {
		if m.Self.MaxMana <= 0 || m.Self.Mana <= 0 {
			t.Fatalf("the cleric's own mana: %d of %d", m.Self.Mana, m.Self.MaxMana)
		}
	})
	if s := p.Screen(140, 45); strings.Count(s, "MP ") != 2 { // the warrior has none: the companion's and the party's
		t.Fatalf("the panels show no mana for both helpers:\n%s", s)
	}
	p.Golden("party_mana")
}

// Every marker of the zone map is on the map (D55): the zone line, the scout
// markers, and the Harvest Raid's spawn and goal, each in its own glyph, and
// the zone overview names them in its legend.
func TestAllTheZonesMarkersAreOnTheMap(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: warrior(scenario.Downs), Zone: noSpawns})
	w.Place("ann", "hale", 60000, 210000) // between the town gate and Barleycross
	p := w.Login("ann", "hale")
	p.Eventually("the zone map", func(m *model.Model) bool { return m.Map != nil && m.Map.GetSpace() == m.Space })
	p.Model(func(m *model.Model) {
		kinds := map[string]int{}
		for _, k := range m.Map.GetMarkers() {
			kinds[k.GetKind()]++
		}
		if kinds["zone_line"] < 2 || kinds["scout"] < 3 || kinds["event_spawn"] != 1 || kinds["event_goal"] != 1 {
			t.Fatalf("the zone map's markers: %v", kinds)
		}
	})
	// The spots are named as the pack names them (D56), not by their ids.
	near := p.Screen(140, 45)
	// (Nearby lists the nearest dozen of the Downs' many places now; the farther
	// Goblin Lookout is named by the overview below, and Brigands' Rise, which
	// the overview's crowded labels leave out, is left to the markers' kinds above.)
	for _, name := range []string{"Barleycross", "Wickett Mill"} {
		if !strings.Contains(near, name) {
			t.Errorf("the screen does not name %q:\n%s", name, near)
		}
	}
	for _, id := range []string{"barleycross", "raid-edge", "wickett-mill", "goblin-lookout"} {
		if strings.Contains(near, id) {
			t.Errorf("the screen shows the marker's id %q:\n%s", id, near)
		}
	}
	p.Press("m")
	s := p.Screen(140, 45)
	// The Lookout is too small a place to be named at the widest zoom (D120);
	// naming everything (Ctrl+A, as Alt) does.
	p.Press("ctrl+a")
	if all := p.Screen(140, 45); !strings.Contains(all, "Goblin Lookout") {
		t.Errorf("the overview does not name the Goblin Lookout:\n%s", all)
	}
	p.Press("ctrl+a")
	for _, glyph := range []string{";", "v", "G"} {
		// The overview draws the markers; the legend names them (it has the glyph
		// too, so the picture must hold at least one more).
		if strings.Count(s, glyph) < 2 {
			t.Errorf("no %q on the overview:\n%s", glyph, s)
		}
	}
	p.Golden("markers_overview")
}
