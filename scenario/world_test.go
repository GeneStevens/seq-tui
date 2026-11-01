package scenario_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/core"
	"github.com/GeneStevens/seq/testserver"
)

// The scenarios of the redrawn zones (D53, WLD-002): Thornwick's streets, the
// Downs' road and ford, a chase round a building, the Warren's tunnels. Each
// presses its own keys; the zones are seq-world's, loaded through the test
// server, and the ground the screen draws is read from the same packs.

// warrior is an account with one warrior, hale, at the start of home.
func warrior(home string) []testserver.Account {
	return []testserver.Account{{Name: "ann", Home: home, Classed: map[string]string{"hale": "warrior"}}}
}

// noSpawns clears every zone's mobs, so a walk is not a fight.
func noSpawns(_ string, z *core.Zone) { z.Spawns = nil }

// onGround says the character stands on that ground.
func onGround(t *testing.T, p *scenario.Player, want model.AreaKind) {
	t.Helper()
	p.Model(func(m *model.Model) {
		if got := m.Underfoot(); got != want {
			t.Fatalf("at %v the ground is %q; want %q", m.Self.Pos, got, want)
		}
	})
}

// status is the status line (the last row) of the screen at 80x24.
func status(p *scenario.Player) string {
	rows := strings.Split(strings.TrimRight(p.Screen(80, 24), "\n"), "\n")
	return rows[len(rows)-1]
}

// From Hawthorn Square along Gatehouse Way to the east gate: the streets are
// roads (the status line says so), the buildings stand either side, and a town
// guard walks its patrol along the way.
func TestThornwickWalkTheStreetsFromTheSquareToTheGate(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})
	p := w.NewCharacter("ann")
	makeWarrior(p, "Hale")
	onGround(t, p, model.Road)
	if s := status(p); !strings.Contains(s, "road: faster") {
		t.Fatalf("on a road the status line says %q", s)
	}
	p.Golden("thornwick_streets")

	// East along the way, to where a town guard walks.
	walkVia(t, w, p, [2]int64{100000, 45000})
	// The guard at the gate is an NPC (D55) that walks (78, 45) to (112, 45) and
	// back. The server streams its moves (D56); the zone map tells no path.
	p.Eventually("the gate guard in view", func(m *model.Model) bool { return m.NPC("guard-gate") != nil })

	walkVia(t, w, p, [2]int64{114000, 45000})
	p.Golden("thornwick_gate")
	p.Press("e")
	notice(t, p, "use door ok", false)
	p.Eventually("the Downs", inSpace("Thornwick Downs"))
}

// The Downs by the road from the town gate to the river, then the ford: the road
// is fast, the river is not walked into, the ford is crossed on foot.
func TestDownsCrossByTheRoadAndFordTheRiver(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: warrior(scenario.Downs), Zone: noSpawns})
	p := w.Login("ann", "hale")
	onGround(t, p, model.Road)

	// East along the road for 138 m, then south across the fields to the river bank.
	walkVia(t, w, p, [2]int64{150000, 225000})
	onGround(t, p, model.Road)
	walkVia(t, w, p, [2]int64{150000, 210000})
	walkVia(t, w, p, [2]int64{162000, 210000})
	p.Golden("downs_riverbank")

	// The water is not walked into.
	for range 3 {
		p.Press("right")
		w.Advance(time.Second)
	}
	p.Model(func(m *model.Model) {
		if x := m.Self.Pos.GetX(); x > 170000 {
			t.Fatalf("the character walked into the river: %v", m.Self.Pos)
		}
	})

	// The ford, 10 m south, is crossed.
	walkVia(t, w, p, [2]int64{162000, 198000}, [2]int64{170000, 198000})
	onGround(t, p, model.Ford)
	p.Golden("downs_ford")
	walkVia(t, w, p, [2]int64{186000, 198000})
	p.Model(func(m *model.Model) {
		if m.Self.Pos.GetX() < 183000 {
			t.Fatalf("the ford was not crossed: %v", m.Self.Pos)
		}
	})
	p.Golden("downs_far_bank")
	p.Press("m")
	p.Golden("downs_overview")
}

// A wolf takes the character in sight, the character goes round Wickett Mill
// and the wolf follows round the building (D53 pathfinding) instead of giving up.
func TestDownsGetChasedRoundABuilding(t *testing.T) {
	w := scenario.New(t, testserver.Options{
		Zones: scenario.Town, Accounts: warrior(scenario.Downs),
		Zone: func(space string, z *core.Zone) {
			if space != scenario.Downs {
				return
			}
			kept := z.Spawns[:0]
			for _, s := range z.Spawns {
				if s.ID == "meadow-wolves-a" || strings.HasPrefix(s.ID, "meadow-wolves-a-") {
					s.Pos = core.Pos{X: 45000, Y: 270000} // 13 m north of the mill's north wall, 7 m from the character: inside its 8 m aggro range
					kept = append(kept, s)
				}
			}
			z.Spawns = kept
		},
	})
	w.Place("ann", "hale", 45000, 263000)
	p := w.Login("ann", "hale")
	wolf := func() (id string, x, y int64) {
		p.Model(func(m *model.Model) {
			for _, e := range m.EntityIDs() {
				if strings.HasPrefix(e, "mob-meadow-wolves-a") {
					id, x, y = e, m.Entities[e].GetPos().GetX(), m.Entities[e].GetPos().GetY()
				}
			}
		})
		return
	}
	if id, _, _ := wolf(); id == "" {
		t.Fatal("no wolf in view of the mill's north side")
	}

	// West along the north wall, then south along the west wall, then east along
	// the south wall: the mill is between the character and where the wolf began.
	walkVia(t, w, p, [2]int64{36000, 263000})
	p.Golden("chase_north_side")
	walkVia(t, w, p, [2]int64{36000, 243000}, [2]int64{45000, 243000})
	_, wx, wy := wolf()
	p.Model(func(m *model.Model) {
		d := math.Hypot(float64(wx-m.Self.Pos.GetX()), float64(wy-m.Self.Pos.GetY()))
		if wx == 0 || d > 8000 || wy > 257000 {
			t.Fatalf("the wolf is at (%d, %d), %.0f mm from the character at %v: it did not follow round the mill", wx, wy, d, m.Self.Pos)
		}
		if m.Self.HP <= 0 {
			t.Fatalf("the character died")
		}
	})
	p.Golden("chase_south_side")
}

// The Warren: the mouth, then the gallery through the dig hall and the neck, to
// the head of the den's wing. The floor is drawn, the rock beside it is walls.
func TestWarrenWalkTheTunnelsToTheDen(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{scenario.Warren}, Accounts: warrior(scenario.Warren), Zone: noSpawns})
	p := w.Login("ann", "hale")
	onGround(t, p, model.Floor)
	p.Golden("warren_mouth")
	walkVia(t, w, p, [2]int64{60000, 70000})
	p.Golden("warren_tunnels")

	// Rock is not walked into: the gallery is 8 m wide.
	for range 4 {
		p.Press("up")
		w.Advance(time.Second)
	}
	p.Model(func(m *model.Model) {
		if y := m.Self.Pos.GetY(); y > 74000 && !(y < 82000 && m.Self.Pos.GetX() >= 46000 && m.Self.Pos.GetX() <= 82000) {
			t.Fatalf("the character walked into the rock: %v", m.Self.Pos)
		}
	})
	walkVia(t, w, p, [2]int64{60000, 70000}, [2]int64{120000, 70000}, [2]int64{130000, 70000})
	p.Model(func(m *model.Model) {
		if m.Self.Pos.GetX() < 128000 {
			t.Fatalf("the den was not reached: %v", m.Self.Pos)
		}
	})
	p.Golden("warren_den")
	p.Press("m")
	p.Golden("warren_overview")
}
