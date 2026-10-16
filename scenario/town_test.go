package scenario_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/content"
	"github.com/GeneStevens/seq/core"
	"github.com/GeneStevens/seq/testserver"
)

// The scenarios of Thornwick and the Thornwick Downs (SEQ-045) on the town
// server: each presses its own keys, the way a player does.

// townAccount is one account with no characters: the scenario makes one on the
// creation screen.
func townAccount() []testserver.Account { return []testserver.Account{{Name: "ann"}} }

// makeWarrior makes a Warrior called name on the creation screen (the class menu's
// first entry) and waits for it to be playing.
func makeWarrior(p *scenario.Player, name string) {
	p.Type(name)
	p.Press("tab", "1", "enter")
	p.Created()
}

// offered finds the contract with that name on the board the player stands at,
// as the position in the board's list; fails the test if it is not offered.
func offered(t *testing.T, p *scenario.Player, board, name string) int {
	t.Helper()
	idx := -1
	p.Model(func(m *model.Model) {
		for i, o := range m.NPC(board).GetOffers() {
			if o.GetName() == name {
				idx = i
			}
		}
	})
	if idx < 0 {
		t.Fatalf("the board does not offer %q", name)
	}
	return idx
}

// walk presses the arrow n times, a second of game time after each: 3 m a press.
func walk(w *scenario.World, p *scenario.Player, key string, n int) {
	for range n {
		p.Press(key)
		w.Advance(time.Second)
	}
}

func inSpace(name string) func(*model.Model) bool {
	return func(m *model.Model) bool {
		return m.Ready() && m.Map != nil && m.Map.GetSpace() == m.Space && m.Map.GetName() == name
	}
}

// A new character begins in Hawthorn Square; G beside the contract board answers
// with its lines and two options; the first opens the board, which offers four
// contracts for level 1; Enter takes the Hunt. The board holds three, refuses a
// fourth with its reason, and a contract is given up with X.
func TestThornwickTalkToTheBoardAndTakeAHunt(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})
	w.Fixture("thornwick_board.jsonl")
	p := w.NewCharacter("ann")
	makeWarrior(p, "Hale")
	p.Model(func(m *model.Model) {
		if m.Map.GetName() != "Thornwick" || m.Self.Pos.GetX() != 60000 || m.Self.Pos.GetY() != 45000 || len(m.NPCs) < 3 {
			t.Fatalf("a new character: in %q at %v with %d people in view", m.Map.GetName(), m.Self.Pos, len(m.NPCs))
		}
	})
	p.Golden("thornwick_square")

	// The board is 6 m from the start point: out of the 5 m reach.
	p.Press("g")
	notice(t, p, "out_of_range", true)
	p.Press("up")
	w.Advance(time.Second)
	p.Press("g")
	notice(t, p, "talk to Thornwick Contract Board ok", false)
	p.Model(func(m *model.Model) {
		d := m.Said("board")
		if d == nil || len(d.Lines) != 2 || len(d.Options) != 2 || d.Options[0].GetAction() != "contract_accept" {
			t.Fatalf("the board said %+v", d)
		}
	})
	p.Golden("talk_board")
	p.Press("1")
	p.Golden("board_offers")
	hunt := offered(t, p, "board", "Thin the Meadow Wolves")
	for range hunt {
		p.Press("down")
	}
	p.Press("enter")
	notice(t, p, "take Thin the Meadow Wolves ok", false)
	p.Model(func(m *model.Model) {
		c := m.Contract("hunt-meadow-wolves")
		if c == nil || c.GetKind() != "hunt" || c.GetHave() != 0 || c.GetNeed() != 3 || !hasLog(m, "reward", "you took the contract Thin the Meadow Wolves") {
			t.Fatalf("after taking it: %+v", m.Self.Contracts)
		}
	})
	p.Golden("board_taken")

	// Two more, and the fourth is refused with its reason.
	p.Press("down", "enter", "down", "enter")
	p.Model(func(m *model.Model) {
		if len(m.Self.Contracts) != 3 {
			t.Fatalf("three contracts held, have %d", len(m.Self.Contracts))
		}
	})
	p.Press("down", "enter")
	notice(t, p, "contract_limit", true)
	// Tab to the contracts held; X gives the third up.
	p.Press("tab", "down", "down", "x")
	notice(t, p, "give up", false)
	p.Model(func(m *model.Model) {
		if len(m.Self.Contracts) != 2 || !hasLog(m, "reward", "gave up the contract") {
			t.Fatalf("after giving one up: %+v", m.Self.Contracts)
		}
	})
	p.Press("esc", "esc", "k")
	p.Golden("contracts_held")
	p.Press("esc", "m")
	p.Golden("town_overview")
}

// fightUntil fights what is in view until done holds: Tab picks a mob, F
// attacks, and the clock runs ten seconds at a time (a camp's mob comes back 45
// seconds after it dies). It gives up after limit steps.
func fightUntil(t *testing.T, w *scenario.World, p *scenario.Player, done func(*model.Model) bool, limit int) {
	t.Helper()
	for range limit {
		finished, attacking := false, false
		p.Model(func(m *model.Model) { finished, attacking = done(m), m.Self.AutoAttack })
		if finished {
			return
		}
		if !attacking {
			p.Press("tab")
			target := false
			p.Model(func(m *model.Model) { target = m.Self.Target != "" })
			if target {
				p.Press("f")
			}
		}
		w.Advance(10 * time.Second)
	}
	t.Fatalf("the fight did not finish in %d steps", limit)
}

// Take the Hunt at the board, walk out of the east gate, kill the wolves of the
// Downs, come back and hand it in.
func TestThornwickHuntOutTheGateAndBack(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})
	w.Fixture("thornwick_hunt.jsonl")
	p := w.NewCharacter("ann")
	makeWarrior(p, "Hale")
	p.Press("up")
	w.Advance(time.Second)
	p.Press("g", "1", "enter", "esc", "esc")
	p.Model(func(m *model.Model) {
		if m.Contract("hunt-meadow-wolves") == nil {
			t.Fatalf("the hunt was not taken: %v", m.Self.Contracts)
		}
	})

	// East along the square to the gate, 54 m.
	walk(w, p, "right", 18)
	walk(w, p, "down", 1)
	p.Golden("gate_before")
	p.Press("e")
	notice(t, p, "use door ok", false)
	p.Eventually("the Downs", inSpace("Thornwick Downs"))
	p.Golden("downs_gate")

	// To the wolves at (32, 118): they notice a character 8 m off.
	walk(w, p, "right", 7)
	walk(w, p, "down", 9)
	p.Golden("downs_camp")
	fightUntil(t, w, p, func(m *model.Model) bool { c := m.Contract("hunt-meadow-wolves"); return c != nil && c.GetReady() }, 25)
	p.Model(func(m *model.Model) {
		if c := m.Contract("hunt-meadow-wolves"); c == nil || c.GetHave() != 3 || !hasLog(m, "reward", "contract Thin the Meadow Wolves: 3 of 3") {
			t.Fatalf("after the hunt: %v", m.Self.Contracts)
		}
	})
	p.Golden("hunt_ready")

	// Home: back to the door, through it, and across the square to the board.
	walk(w, p, "left", 7)
	walk(w, p, "up", 9)
	p.Press("e")
	notice(t, p, "use door ok", false)
	p.Eventually("Thornwick", inSpace("Thornwick"))
	walk(w, p, "left", 18)
	walk(w, p, "up", 1)
	p.Press("g")
	p.Press("2")
	p.Golden("board_turn_in")
	p.Press("enter")
	notice(t, p, "hand in Thin the Meadow Wolves ok", false)
	p.Model(func(m *model.Model) {
		if len(m.Self.Contracts) != 0 || m.Self.Crowns < 12 || !hasLog(m, "reward", "handed in Thin the Meadow Wolves") {
			t.Fatalf("after the turn-in: contracts %v, %d crowns", m.Self.Contracts, m.Self.Crowns)
		}
	})
	p.Golden("hunt_turned_in")
}

// A warrior of level 3 with crowns stands in Tanner's Row. G talks to the armour
// vendor and its option opens the vendor's window, where Enter buys; then G beside
// the warrior trainer opens the trainer's, where Enter teaches an ability.
func TestThornwickBuyFromAVendorAndTrainAnAbility(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: []testserver.Account{{Name: "ann", Classed: map[string]string{"hale": "warrior"}}}})
	w.Fixture("thornwick_shops.jsonl")
	w.Place("ann", "hale", 30000, 64000)
	z, _ := content.Town()
	w.Srv.Give("hale", func(cs *core.CharacterSave) {
		cl := z.Class("warrior")
		cs.Stats, cs.Known, cs.Slots, cs.XP = core.ClassCharacter(cl, *z.CharacterTemplate("warrior"), 3)
		cs.Known, cs.Slots = cl.KnownAt(1), core.DefaultSlots(cl.KnownAt(1))
		cs.Crowns = 200
	})
	p := w.Login("ann", "hale")

	// The vendor is 12 m off: close the gap, then talk.
	walk(w, p, "right", 2)
	walk(w, p, "down", 2)
	p.Press("g")
	p.Model(func(m *model.Model) {
		if d := m.Said("vendor-armour"); d == nil || len(d.Options) != 2 || d.Options[0].GetAction() != "buy" {
			t.Fatalf("the armour vendor said %+v", d)
		}
	})
	p.Golden("talk_vendor")
	p.Press("1")
	p.Golden("talk_vendor_window")
	p.Press("enter")
	notice(t, p, "buy Tanned Cap ok", false)
	items, crowns := inventoryOf(p)
	if !slices.Equal(items, []string{"tanned-cap"}) || crowns >= 200 {
		t.Fatalf("after buying: %q, %d crowns", items, crowns)
	}
	p.Press("esc", "esc")

	// To the warrior trainer in the north-west corner of the row.
	walk(w, p, "left", 5)
	walk(w, p, "up", 5)
	p.Press("g")
	p.Model(func(m *model.Model) {
		if d := m.Said("trainer-warrior"); d == nil || d.Options[0].GetAction() != "train" {
			t.Fatalf("the warrior trainer said %+v", d)
		}
	})
	p.Press("1")
	learn, row := "", 0
	p.Model(func(m *model.Model) {
		for i, tr := range m.NPC("trainer-warrior").GetTrains() {
			if !tr.GetKnown() && tr.GetLevel() <= m.Self.Level && learn == "" {
				learn, row = tr.GetAbility(), i
			}
		}
	})
	if learn == "" {
		t.Fatal("the trainer has nothing to teach a level 3 warrior")
	}
	for range row {
		p.Press("down")
	}
	p.Golden("trainer_window")
	p.Press("enter")
	notice(t, p, "train ", false)
	p.Model(func(m *model.Model) {
		if m.AbilityByID(learn) == nil {
			t.Fatalf("%s was not learned: %v", learn, m.Self.Abilities)
		}
	})
	p.Golden("trainer_learned")
}

// A Scout contract: the marker shows on the map (%) once the character is in the
// Downs, the Contracts panel counts it reached, and the board pays it.
func TestThornwickScoutToAMarker(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})
	w.Fixture("thornwick_scout.jsonl")
	p := w.NewCharacter("ann")
	makeWarrior(p, "Hale")
	p.Press("up")
	w.Advance(time.Second)
	p.Press("g", "1")
	for range offered(t, p, "board", "Word from Wickett Mill") {
		p.Press("down")
	}
	p.Press("enter")
	notice(t, p, "take Word from Wickett Mill ok", false)
	p.Press("esc", "esc")

	walk(w, p, "right", 18)
	walk(w, p, "down", 1)
	p.Press("e")
	p.Eventually("the Downs", inSpace("Thornwick Downs"))
	p.Model(func(m *model.Model) {
		c := m.Contract("scout-wickett-mill")
		if c == nil || c.GetMarker() != "wickett-mill" || c.GetMarkerPos() == nil || c.GetName() != "Word from Wickett Mill" {
			t.Fatalf("in the Downs the scout contract is %v", c)
		}
	})
	p.Golden("scout_marker_in_view")

	// Down to the mill's latitude, then east to it.
	walk(w, p, "down", 5)
	walk(w, p, "right", 13)
	p.Model(func(m *model.Model) {
		if c := m.Contract("scout-wickett-mill"); c == nil || !c.GetReady() || !hasLog(m, "reward", "the marker is reached") {
			t.Fatalf("at the marker: %v", c)
		}
	})
	p.Golden("scout_reached")
	p.Press("m")
	p.Golden("zone_overview")
	p.Press("esc")

	// Home and hand it in.
	walk(w, p, "left", 13)
	walk(w, p, "up", 5)
	p.Press("e")
	p.Eventually("Thornwick", inSpace("Thornwick"))
	walk(w, p, "left", 18)
	walk(w, p, "up", 1)
	p.Press("g", "2", "enter")
	notice(t, p, "hand in Word from Wickett Mill ok", false)
	p.Model(func(m *model.Model) {
		if len(m.Self.Contracts) != 0 || m.Self.Crowns != 8 || m.Self.XP < 20 {
			t.Fatalf("after the turn-in: %v, %d crowns, %d xp", m.Self.Contracts, m.Self.Crowns, m.Self.XP)
		}
	})
}

// The Harvest Raid starts: the whole zone is told, the banner is up, and a level 12
// warrior who stands in the raiders' way fights them. When it ends the banner says
// how, and the character, whose account is attended, is paid for taking part.
func TestThornwickHarvestRaidAnnouncedAndJoined(t *testing.T) {
	w := scenario.New(t, testserver.Options{
		Zones:    scenario.Town,
		Accounts: []testserver.Account{{Name: "ann", Home: testserver.ThornwickDowns, Classed: map[string]string{"hale": "warrior"}}},
		Zone: func(space string, z *core.Zone) {
			for i := range z.WorldEvents {
				z.WorldEvents[i].EveryMs, z.WorldEvents[i].OffsetMs = 20_000, 0 // the raid begins 20 s in, not an hour and a half
			}
		},
	})
	w.Fixture("thornwick_raid.jsonl")
	w.Place("ann", "hale", 110000, 131000)
	z, _ := content.Town()
	w.Srv.Give("hale", func(cs *core.CharacterSave) {
		cl := z.Class("warrior")
		cs.Stats, cs.Known, cs.Slots, cs.XP = core.ClassCharacter(cl, *z.CharacterTemplate("warrior"), 12)
		cs.Known, cs.Slots = cl.KnownAt(1), core.DefaultSlots(cl.KnownAt(1))
	})
	w.Srv.Attend("ann")
	p := w.Login("ann", "hale")
	p.Model(func(m *model.Model) {
		if len(m.ActiveEvents()) != 0 {
			t.Fatalf("the raid has begun already: %v", m.WorldEvents)
		}
	})
	w.Advance(21 * time.Second)
	p.Model(func(m *model.Model) {
		e := m.WorldEvents["harvest-raid"]
		if e == nil || !e.Active || e.Kind != "started" || !strings.Contains(e.Text, "Defend the farms") || !hasLog(m, "world_event", "Harvest Raid") {
			t.Fatalf("the announcement: %+v", e)
		}
	})
	p.Golden("event_banner")

	over := func(m *model.Model) bool { e := m.WorldEvents["harvest-raid"]; return e != nil && !e.Active }
	fightUntil(t, w, p, over, 40)
	p.Model(func(m *model.Model) {
		e := m.WorldEvents["harvest-raid"]
		if e.Kind != "won" && e.Kind != "lost" {
			t.Fatalf("it ended as %q", e.Kind)
		}
		if !hasLog(m, "reward", "was paid for the Harvest Raid") {
			t.Fatalf("the character was not paid for taking part: %v", m.Log)
		}
	})
	p.Golden("event_over")
}

// Old Grimtusk, the Downs' roaming rare, walks into view: the log says so, the map
// marks it (r), Nearby and the Target panel call it a rare.
func TestThornwickARareComesIntoView(t *testing.T) {
	w := scenario.New(t, testserver.Options{
		Zones:    scenario.Town,
		Accounts: []testserver.Account{{Name: "ann", Home: testserver.ThornwickDowns, Classed: map[string]string{"hale": "warrior"}}},
		Zone: func(space string, z *core.Zone) {
			if space == testserver.ThornwickDowns { // one place for it to appear, 10 m from the character
				z.Rares[0].Points = []core.Pos{{X: 220000, Y: 150000}}
				z.Rares[0].MinMs, z.Rares[0].MaxMs = 10_000, 10_000 // it comes at 10 s
			}
		},
	})
	w.Fixture("thornwick_rare.jsonl")
	w.Place("ann", "hale", 210000, 150000)
	z, _ := content.Town()
	w.Srv.Give("hale", func(cs *core.CharacterSave) {
		cl := z.Class("warrior")
		cs.Stats, cs.Known, cs.Slots, cs.XP = core.ClassCharacter(cl, *z.CharacterTemplate("warrior"), 12)
		cs.Known, cs.Slots = cl.KnownAt(1), core.DefaultSlots(cl.KnownAt(1))
	})
	p := w.Login("ann", "hale")
	p.Model(func(m *model.Model) {
		if len(m.RaresInView()) != 0 {
			t.Fatal("a rare is in view before its time")
		}
	})
	w.Advance(11 * time.Second)
	p.Model(func(m *model.Model) {
		r := m.RaresInView()
		if len(r) != 1 || r[0].GetName() != "Old Grimtusk" || !hasLog(m, "rare_appeared", "a rare, Old Grimtusk") {
			t.Fatalf("rares in view: %v", r)
		}
	})
	p.Press("tab")
	notice(t, p, "target ok", false)
	p.Golden("rare_in_view")
	if s := p.Screen(80, 24); !strings.Contains(s, "RARE") {
		t.Fatalf("the rare is not marked:\n%s", s)
	}
}
