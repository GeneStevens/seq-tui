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
// toTheBoard walks from the start to the contract board, round the Watch's
// greeter, who stands between: G there talks to the nearest person, the board.
func toTheBoard(w *scenario.World, p *scenario.Player) {
	p.Step("left")
	walk(w, p, "up", 2)
}

func walk(w *scenario.World, p *scenario.Player, key string, n int) {
	for range n {
		p.Step(key)
	}
}

// walkVia walks the character leg by leg to each point (in mm), pressing the
// arrows a second of game time apart: along x first, then along y, so a leg
// follows the street a reader of the map would. It stops within a step of the
// point, and fails the test if a press moves the character nowhere.
func walkVia(t *testing.T, w *scenario.World, p *scenario.Player, pts ...[2]int64) {
	t.Helper()
	const step = 3000
	pos := func() (x, y int64) {
		p.Model(func(m *model.Model) { x, y = m.Self.Pos.GetX(), m.Self.Pos.GetY() })
		return
	}
	for _, pt := range pts {
		for range 400 {
			x, y := pos()
			dx, dy := pt[0]-x, pt[1]-y
			key, along := "", int64(0)
			switch {
			case dx >= step:
				key, along = "right", dx
			case dx <= -step:
				key, along = "left", -dx
			case dy >= step:
				key, along = "up", dy
			case dy <= -step:
				key, along = "down", -dy
			default:
				key = ""
			}
			if key == "" {
				break
			}
			// A leg is walked in holds of up to 12 m, none past the point.
			hold := min(along/step*step, 12000)
			if !p.TryStepBy(key, hold) {
				// A mob that engages the character in that step stops the walk
				// (D66); a later press is obeyed.
				if !p.TryStepBy(key, hold) {
					t.Fatalf("a step %s at (%d, %d) on the way to (%d, %d) moves nowhere", key, x, y, pt[0], pt[1])
				}
			}
		}
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

	// The greeter, Tamsin Hale, stands 2 m from the start point (D62): G talks to
	// the nearest person. The board is 6 m away, out of the 5 m reach.
	p.Press("g")
	notice(t, p, "talk to Tamsin Hale ok", false)
	p.Press("esc")
	toTheBoard(w, p)
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
	hunt := offered(t, p, "board", "Wolves at Wickett Mill")
	for range hunt {
		p.Press("down")
	}
	p.Press("enter")
	notice(t, p, "take Wolves at Wickett Mill ok", false)
	p.Model(func(m *model.Model) {
		c := m.Contract("hunt-mill-wolves")
		if c == nil || c.GetKind() != "hunt" || c.GetHave() != 0 || c.GetNeed() != 4 || !hasLog(m, "reward", "You take the contract Wolves at Wickett Mill") {
			t.Fatalf("after taking it: %+v", m.Self.Contracts)
		}
	})
	p.Golden("board_taken")

	// Two more, and the fourth is refused with its reason.
	// (Contracts of later levels are listed below the open ones now: Enter on the
	// row the cursor stays on takes the next, where a Down would step onto one.)
	p.Press("enter", "enter")
	p.Model(func(m *model.Model) {
		if len(m.Self.Contracts) != 3 {
			t.Fatalf("three contracts held, have %d", len(m.Self.Contracts))
		}
	})
	p.Press("up", "enter")
	notice(t, p, "as many contracts as you can", true)
	// Tab to the contracts held; X gives the third up.
	p.Press("tab", "down", "down", "x")
	notice(t, p, "give up", false)
	p.Model(func(m *model.Model) {
		if len(m.Self.Contracts) != 2 || !hasLog(m, "reward", "You give up the contract") {
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
	toTheBoard(w, p)
	p.Press("g", "1")
	for range offered(t, p, "board", "Wolves at Wickett Mill") {
		p.Press("down")
	}
	p.Press("enter", "esc", "esc")
	p.Model(func(m *model.Model) {
		if m.Contract("hunt-mill-wolves") == nil {
			t.Fatalf("the hunt was not taken: %v", m.Self.Contracts)
		}
	})

	// East along the square and Gatehouse Way to the gate, 54 m.
	walkVia(t, w, p, [2]int64{114000, 45000})
	p.Golden("gate_before")
	p.Press("e")
	notice(t, p, "use door ok", false)
	p.Eventually("the Downs", inSpace("Thornwick Downs"))
	p.Model(func(m *model.Model) {
		if c := m.Camps(); len(c) == 0 || c[0].GetMob() == "" || c[0].GetCount() == 0 {
			t.Fatalf("the Downs' map has camps %v", c)
		}
	})
	p.Golden("downs_gate")

	// Across the fields to the wolves at (45, 94): they notice a character 8 m off.
	walkVia(t, w, p, [2]int64{30000, 150000}, [2]int64{30000, 104000}, [2]int64{45000, 104000})
	p.Golden("downs_camp")
	walkVia(t, w, p, [2]int64{45000, 98000})
	fightUntil(t, w, p, func(m *model.Model) bool { c := m.Contract("hunt-mill-wolves"); return c != nil && c.GetReady() }, 25)
	p.Model(func(m *model.Model) {
		if c := m.Contract("hunt-mill-wolves"); c == nil || c.GetHave() != 4 || !hasLog(m, "reward", "Wolves at Wickett Mill: 4 of 4") {
			t.Fatalf("after the hunt: %v", m.Self.Contracts)
		}
	})
	p.Golden("hunt_ready")

	// Home: back to the door, through it, and across the square to the board.
	walkVia(t, w, p, [2]int64{30000, 104000}, [2]int64{30000, 150000}, [2]int64{8000, 150000})
	p.Press("e")
	notice(t, p, "use door ok", false)
	p.Eventually("Thornwick", inSpace("Thornwick"))
	walkVia(t, w, p, [2]int64{57000, 45000}, [2]int64{57000, 51000}) // by the board, off the greeter
	p.Press("g")
	p.Press("2")
	for _, sz := range [][2]int{{140, 45}, {80, 24}} {
		// A finished contract says where it is handed in (D76).
		if s := p.Screen(sz[0], sz[1]); !strings.Contains(s, "hand it in at Thornwick Contract Board") {
			t.Errorf("%v: the READY contract does not say where to hand it in:\n%s", sz, s)
		}
	}
	p.Golden("board_turn_in")
	p.Press("enter")
	notice(t, p, "hand in Wolves at Wickett Mill ok", false)
	for _, sz := range [][2]int{{140, 45}, {80, 24}} {
		// The board panel shows what the hand-in paid (D76).
		if s := p.Screen(sz[0], sz[1]); !strings.Contains(s, "Handed in the contract Wolves at Wickett Mill: 22 XP, 10 crowns") {
			t.Errorf("%v: no hand-in summary in the board panel:\n%s", sz, s)
		}
	}
	p.Model(func(m *model.Model) { noIDs(t, m) })
	p.Model(func(m *model.Model) {
		if len(m.Self.Contracts) != 0 || m.Self.Crowns < 10 || !hasLog(m, "reward", "You hand in Wolves at Wickett Mill") {
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
	w.Place("ann", "hale", 28000, 64000)
	z, _ := content.Zone(scenario.Thornwick)
	w.Srv.Give("hale", func(cs *core.CharacterSave) {
		cl := z.Class("warrior")
		cs.Stats, cs.Known, cs.Slots, cs.XP = core.ClassCharacter(cl, *z.CharacterTemplate("warrior"), 3)
		cs.Known, cs.Slots = cl.KnownAt(1), core.DefaultSlots(cl.KnownAt(1))
		cs.Crowns = 200
	})
	p := w.Login("ann", "hale")

	// The vendor is 8 m off, up the row: close the gap, then talk.
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

	// To the warrior trainer: up Tanner's Row to Candlemakers' Lane, then west.
	walk(w, p, "down", 14)
	walk(w, p, "left", 4)
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
	toTheBoard(w, p)
	p.Press("g", "1")
	for range offered(t, p, "board", "Word from Wickett Mill") {
		p.Press("down")
	}
	p.Press("enter")
	notice(t, p, "take Word from Wickett Mill ok", false)
	p.Press("esc", "esc")

	walkVia(t, w, p, [2]int64{114000, 45000})
	p.Press("e")
	p.Eventually("the Downs", inSpace("Thornwick Downs"))
	p.Model(func(m *model.Model) {
		c := m.Contract("scout-wickett-mill")
		if c == nil || c.GetMarker() != "wickett-mill" || c.GetMarkerPos() == nil || c.GetName() != "Word from Wickett Mill" {
			t.Fatalf("in the Downs the scout contract is %v", c)
		}
	})
	p.Model(func(m *model.Model) {
		// The spot is the zone map's (D52), not something the client worked out.
		if k := m.Map.GetMarkers(); len(k) == 0 || m.Marker(m.Contract("scout-wickett-mill")) == nil {
			t.Fatalf("the Downs' map has markers %v; the contract's spot is %v", k, m.Marker(m.Contract("scout-wickett-mill")))
		}
	})
	if s := p.Screen(140, 45); !strings.Contains(s, "scout marker") || !strings.Contains(s, "%") {
		t.Fatalf("the marker is not on the map or in Nearby:\n%s", s)
	}
	p.Golden("scout_marker_in_view")

	// Down to the mill's latitude, then east to it.
	walk(w, p, "down", 5)
	walk(w, p, "right", 13)
	p.Model(func(m *model.Model) {
		if c := m.Contract("scout-wickett-mill"); c == nil || !c.GetReady() || !hasLog(m, "reward", "Word from Wickett Mill: 1 of 1") {
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
	walkVia(t, w, p, [2]int64{57000, 45000}, [2]int64{57000, 51000}) // by the board, off the greeter
	p.Press("g", "2", "enter")
	notice(t, p, "hand in Word from Wickett Mill ok", false)
	p.Model(func(m *model.Model) {
		if len(m.Self.Contracts) != 0 || m.Self.Crowns != 8 || m.Self.XP < 20 {
			t.Fatalf("after the turn-in: %v, %d crowns, %d xp", m.Self.Contracts, m.Self.Crowns, m.Self.XP)
		}
	})
}

// The Harvest Raid starts: the whole zone is told, the banner is up, and a level 8
// warrior who stands in the raiders' way fights them (a level 12 one would find
// the level 4 raiders trivial, and they would walk past it: D70). When it ends the banner says
// how, and the character, whose account is attended, is paid for taking part.
func TestThornwickHarvestRaidAnnouncedAndJoined(t *testing.T) {
	w := scenario.New(t, testserver.Options{
		Zones:    scenario.Town,
		Accounts: []testserver.Account{{Name: "ann", Home: scenario.Downs, Classed: map[string]string{"hale": "warrior"}}},
		Zone: func(space string, z *core.Zone) {
			for i := range z.WorldEvents {
				z.WorldEvents[i].EveryMs, z.WorldEvents[i].OffsetMs = 20_000, 0 // the raid begins 20 s in, not an hour and a half
			}
		},
	})
	w.Fixture("thornwick_raid.jsonl")
	w.Place("ann", "hale", 110000, 131000)
	z, _ := content.Zone(scenario.Thornwick)
	w.Srv.Give("hale", func(cs *core.CharacterSave) {
		cl := z.Class("warrior")
		cs.Stats, cs.Known, cs.Slots, cs.XP = core.ClassCharacter(cl, *z.CharacterTemplate("warrior"), 8)
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
		if e == nil || !e.Active || e.Kind != "started" || e.Name != "Harvest Raid" || !strings.Contains(e.Text, "Defend the farms") || !hasLog(m, "world_event", "Goblins pour out of the east") {
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
		if !hasLog(m, "reward", "You are paid for Harvest Raid") {
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
		Accounts: []testserver.Account{{Name: "ann", Home: scenario.Downs, Classed: map[string]string{"hale": "warrior"}}},
		Zone: func(space string, z *core.Zone) {
			if space == scenario.Downs { // one place for it to appear, 10 m from the character
				z.Rares[0].Points = []core.Pos{{X: 215000, Y: 70000}}
				z.Rares[0].MinMs, z.Rares[0].MaxMs = 10_000, 10_000 // it comes at 10 s
			}
		},
	})
	w.Fixture("thornwick_rare.jsonl")
	w.Place("ann", "hale", 205000, 70000)
	z, _ := content.Zone(scenario.Thornwick)
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
		if len(r) != 1 || r[0].GetName() != "Old Grimtusk" || !r[0].GetRare() || !hasLog(m, "rare_appeared", "Old Grimtusk comes into view") {
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

// A trainer is not in view from 25 m; walking up to it, it appears by event
// (npc_appeared) in Nearby and the log, and the traders' list greys it out as too
// far until the character stands within its reach, which is the NPC's own.
func TestThornwickATrainerAppearsAsYouWalkUp(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: []testserver.Account{{Name: "ann", Classed: map[string]string{"hale": "warrior"}}}})
	w.Fixture("thornwick_trainer.jsonl")
	w.Place("ann", "hale", 28000, 40000)
	p := w.Login("ann", "hale")
	p.Model(func(m *model.Model) {
		if m.NPC("trainer-warrior") != nil {
			t.Fatalf("the trainer, 25 m off, is in view already: %v", m.NPCs)
		}
	})
	p.Golden("trainer_out_of_view")
	walk(w, p, "down", 3)
	p.Model(func(m *model.Model) {
		n := m.NPC("trainer-warrior")
		if n == nil || n.GetReachMm() <= 0 || m.InReach(n) || !hasLog(m, "npc_appeared", "Gorran Stoutarm <Warrior Trainer> comes into view") {
			t.Fatalf("after three steps: %v; log ends %+v", n, m.Log[len(m.Log)-2:])
		}
	})
	// The short log keeps the NPCs coming into view out (the model has the line, above).
	if s := p.Screen(140, 45); strings.Contains(s, "comes into view") {
		t.Fatalf("the short log has a person coming into view:\n%s", s)
	}
	p.Golden("trainer_appears")
	p.Press("n")
	if s := p.Screen(140, 45); !strings.Contains(rowWith(s, "Gorran Stoutarm"), "too far") {
		t.Fatalf("a trainer beyond its reach is not marked:\n%s", s)
	}
	p.Golden("trainer_too_far")
	p.Press("esc")
	walkVia(t, w, p, [2]int64{28000, 19000}, [2]int64{16000, 19000})
	p.Model(func(m *model.Model) {
		if n := m.NPC("trainer-warrior"); n == nil || !m.InReach(n) {
			t.Fatalf("the trainer is not in reach after the walk: %v at %v", n, m.Self.Pos)
		}
	})
	p.Press("n")
	if s := p.Screen(140, 45); strings.Contains(rowWith(s, "Gorran Stoutarm"), "too far") {
		t.Fatalf("a trainer in reach is marked too far:\n%s", s)
	}
	p.Press("esc")
}

// A warrior bound in Thornwick dies in the Downs, releases, and wakes in
// Thornwick at the binder: the map, the NPCs in Nearby and the panels are the
// town's.
func TestThornwickDieInTheDownsAndWakeInThornwick(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: []testserver.Account{{Name: "ann", Home: scenario.Downs, Classed: map[string]string{"hale": "warrior"}}}})
	w.Fixture("thornwick_respawn.jsonl")
	z, _ := content.Zone(scenario.Thornwick)
	bind := z.NPC("binder").Pos
	w.Place("ann", "hale", 20000, 150000)
	w.Srv.Give("hale", func(cs *core.CharacterSave) {
		cs.Dead, cs.HP = true, 0
		cs.BindZone, cs.BindPos = z.Name, &bind
	})
	p := w.Login("ann", "hale")
	p.Model(func(m *model.Model) {
		if !m.Self.Dead || m.Map.GetName() != "Thornwick Downs" {
			t.Fatalf("dead %v in %q", m.Self.Dead, m.Map.GetName())
		}
	})
	p.Golden("downs_dead")
	p.Press("r")
	notice(t, p, "release ok", false)
	w.Advance(core.DefaultRespawnDelayMs*time.Millisecond + 2*time.Second)
	p.Eventually("Thornwick", inSpace("Thornwick"))
	p.Model(func(m *model.Model) {
		if m.Self.Dead || m.Self.HP != m.Self.MaxHP || m.Self.Pos.GetX() != int64(bind.X) || m.Self.Pos.GetY() != int64(bind.Y) || m.NPC("binder") == nil {
			t.Fatalf("after waking: dead %v, hp %d/%d at %v, binder %v", m.Self.Dead, m.Self.HP, m.Self.MaxHP, m.Self.Pos, m.NPC("binder"))
		}
	})
	if s := p.Screen(140, 45); !strings.Contains(s, "Thornwick") || !strings.Contains(s, "Binder") {
		t.Fatalf("the screen does not follow:\n%s", s)
	}
	p.Golden("woke_in_thornwick")
}

// rowWith is the row of the screen that holds text, "" if none does.
func rowWith(screen, text string) string {
	for _, l := range strings.Split(screen, "\n") {
		if strings.Contains(l, text) {
			return l
		}
	}
	return ""
}
