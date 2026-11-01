package scenario_test

import (
	"math"
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

// North is up on the real packs (O28, D125): the pinned seq-world lays Thornwick
// out with +y to the north, as the TUI has always drawn and walked it. Each test
// reads the zones through the test server, like the other world scenarios.

// rowOf is the first screen row that holds text, or -1.
func rowOf(screen, text string) int {
	for i, row := range strings.Split(screen, "\n") {
		if strings.Contains(row, text) {
			return i
		}
	}
	return -1
}

// In the Downs overview Highfold, in the north of the dale, is drawn above Eelmere
// and Eel Mere, in the south.
func TestTheDownsOverviewDrawsHighfoldAboveEelMere(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: warrior(scenario.Downs), Zone: noSpawns})
	p := w.Login("ann", "hale")
	p.Resize(200, 50)
	p.Press("m", "ctrl+a") // the large map, with every name written
	s := p.Screen(200, 50)
	high, eel := rowOf(s, "Highfold"), rowOf(s, "Eelmere")
	if high < 0 || eel < 0 || high >= eel {
		t.Errorf("Highfold is on row %d and Eelmere on row %d of the Downs overview:\n%s", high, eel, s)
	}
	if mere := rowOf(s, "Eel Mere"); mere >= 0 && high >= mere {
		t.Errorf("Highfold is on row %d, not above Eel Mere on row %d", high, mere)
	}
}

// In Thornwick the large map draws the North Wall at the top and the South Wall at
// the bottom.
func TestThornwickOverviewDrawsTheNorthWallAtTheTop(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: warrior(scenario.Thornwick), Zone: noSpawns})
	p := w.Login("ann", "hale")
	p.Resize(200, 50)
	p.Press("m", "ctrl+a")
	s := p.Screen(200, 50)
	north, south, square := rowOf(s, "North Wall"), rowOf(s, "South Wall"), rowOf(s, "Hawthorn Square")
	if north < 0 || south < 0 || square < 0 || north >= square || square >= south {
		t.Errorf("North Wall is on row %d, Hawthorn Square on row %d, South Wall on row %d:\n%s", north, square, south, s)
	}
	// Row 0 is the frame and row 1 the clear row above the zone's edge; the edge is row 2.
	if north != 3 {
		t.Errorf("the North Wall is on row %d, not at the top of the map (row 3):\n%s", north, s)
	}
}

// Up and W in Hawthorn Square walk toward the North Wall: y grows.
func TestUpAndWWalkFromHawthornSquareToTheNorthWall(t *testing.T) {
	z, err := content.Zone(scenario.Thornwick)
	if err != nil {
		t.Fatal(err)
	}
	var wall *core.Area
	for i := range z.Areas {
		if z.Areas[i].ID == "wall-north" {
			wall = &z.Areas[i]
		}
	}
	if wall == nil {
		t.Fatal("Thornwick has no wall-north area")
	}
	wallY := wall.Poly[0].Y
	for _, c := range []struct{ name, key string }{{"up", "up"}, {"w", "w"}} {
		t.Run(c.name, func(t *testing.T) {
			w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: warrior(scenario.Thornwick), Zone: noSpawns})
			p := w.Login("ann", "hale")
			from := p.Pos()
			if from[1] != 45000 {
				t.Fatalf("the character did not start in the square: %v", from)
			}
			walk(w, p, c.key, 5)
			to := p.Pos()
			if to[1] <= from[1] {
				t.Errorf("%s took the character from y %d to y %d: not toward the North Wall", c.key, from[1], to[1])
			}
			if d0, d1 := int64(wallY)-from[1], int64(wallY)-to[1]; d1 >= d0 {
				t.Errorf("%s left the character %d mm from the North Wall, from %d mm", c.key, d1, d0)
			}
		})
	}
}

// A click on a landmark of the large map walks to that landmark: the screen cell
// the program draws it at maps back to the same place in the world.
func TestAClickOnALandmarkTargetsThatLandmark(t *testing.T) {
	for _, id := range []string{"highfold", "eel-mere-shore"} {
		t.Run(id, func(t *testing.T) {
			w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: warrior(scenario.Downs), Zone: noSpawns})
			p := w.Login("ann", "hale")
			p.Resize(200, 50)
			p.Press("m")
			var mk *pb.Marker
			var from *pb.Pos
			p.Model(func(m *model.Model) {
				from = m.Self.Pos
				for _, k := range m.Map.GetMarkers() {
					if k.GetId() == id {
						mk = k
					}
				}
			})
			if mk == nil {
				t.Fatalf("the Downs have no %s marker", id)
			}
			x, y := at(t, p, mk.GetPos())
			p.AltClick(x, y)
			notice(t, p, "move ok", false)
			for range 8 {
				w.Advance(time.Second)
			}
			p.Model(func(m *model.Model) {
				// Highfold is north of the character and Eelmere south of it: the walk goes
				// the way the landmark is drawn, and gets nearer to it.
				dy, want := m.Self.Pos.GetY()-from.GetY(), mk.GetPos().GetY()-from.GetY()
				d0 := math.Hypot(float64(mk.GetPos().GetX()-from.GetX()), float64(want))
				d1 := math.Hypot(float64(mk.GetPos().GetX()-m.Self.Pos.GetX()), float64(mk.GetPos().GetY()-m.Self.Pos.GetY()))
				if dy*want <= 0 || d1 >= d0 {
					t.Errorf("the click on %s at (%d, %d) walked from %v to %v, not toward %v", mk.GetName(), x, y, from, m.Self.Pos, mk.GetPos())
				}
			})
		})
	}
}
