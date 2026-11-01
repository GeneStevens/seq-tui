package screen

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func rectPoly(x0, y0, x1, y1 int64) []*pb.Pos {
	return []*pb.Pos{{X: x0, Y: y0}, {X: x1, Y: y0}, {X: x1, Y: y1}, {X: x0, Y: y1}}
}

// overviewWorld is a 400 m square zone with a river (north-south) and a road
// (east-west) crossing in the middle, a lake, a meadow of four pieces whose
// middle is the crossing, and a hut of three small pieces. The character stands
// on the crossing, with an objective beside it.
func overviewWorld(objective *pb.Pos) *model.Model {
	zm := &pb.ZoneMap{Space: "z", Name: "Z", ViewRadiusMm: 40000,
		Bounds: &pb.Rect{MinX: 0, MinY: 0, MaxX: 400000, MaxY: 400000},
		Areas: []*pb.Area{
			{Id: "river", Name: "The River", Kind: "water", Poly: rectPoly(194000, 0, 206000, 400000)},
			{Id: "road", Name: "Long Road", Kind: "road", Poly: rectPoly(0, 196000, 400000, 204000)},
			{Id: "lake", Name: "Mere", Kind: "water", Poly: rectPoly(240000, 240000, 340000, 340000)},
			{Id: "m1", Name: "Meadow", Kind: "forest", Poly: rectPoly(150000, 150000, 190000, 190000)},
			{Id: "m2", Name: "Meadow", Kind: "forest", Poly: rectPoly(210000, 150000, 250000, 190000)},
			{Id: "m3", Name: "Meadow", Kind: "forest", Poly: rectPoly(150000, 210000, 190000, 250000)},
			{Id: "m4", Name: "Meadow", Kind: "forest", Poly: rectPoly(210000, 210000, 250000, 250000)},
			{Id: "h1", Name: "Hut", Kind: "building", Poly: rectPoly(130000, 170000, 136000, 176000)},
			{Id: "h2", Name: "Hut", Kind: "building", Poly: rectPoly(140000, 170000, 146000, 176000)},
			{Id: "h3", Name: "Hut", Kind: "building", Poly: rectPoly(130000, 180000, 136000, 186000)},
			{Id: "s1", Name: "Stones", Kind: "rock", Poly: rectPoly(130000, 150000, 144000, 164000)},
		},
		Doors: []*pb.Door{{Id: "gate", Kind: pb.DoorKind_DOOR_KIND_ZONE_LINE, Name: "Gate", Pos: &pb.Pos{X: 4000, Y: 200000}, LeadsTo: "x"}},
		Camps: []*pb.Camp{{Id: "camp", Mob: "Wolf", Level: 2, Count: 2, Pos: &pb.Pos{X: 270000, Y: 140000}}},
	}
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: zm}})
	self := &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Pos: &pb.Pos{X: 200000, Y: 200000}, Hp: 10, MaxHp: 10}}
	if objective != nil {
		self.Hints = []*pb.Hint{{Kind: "objective", Id: "obj", Label: "Go there", Pos: objective}}
	}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 1, GameTimeMs: 1000, Self: self}}})
	return m
}

// names are the place names written on the grid, in reading order.
func names(g *grid) []string {
	var out []string
	for _, row := range g.cells {
		run := ""
		for _, c := range row {
			if c.k == kPlace {
				run += string(c.r)
				continue
			}
			if run != "" {
				out = append(out, run)
			}
			run = ""
		}
		if run != "" {
			out = append(out, run)
		}
	}
	return out
}

func placed(m *model.Model, v View, cw, ch int) (before []cell, g *grid) {
	g = overviewGrid(m, v, cw, ch)
	c, r := g.cellOf(m.Self.Pos)
	g.put(c, r, glyphSelf, kSelf)
	before = append([]cell(nil), g.flat...)
	paintPlaces(g, m, v)
	return before, g
}

// A place of several areas has one label, wherever it is drawn (D120).
func TestOnePlaceHasOneLabel(t *testing.T) {
	m := overviewWorld(nil)
	for _, sz := range [][2]int{{60, 20}, {90, 30}, {160, 48}} {
		for _, v := range []View{{}, {Labels: true}, {BigZoom: 1}, {BigZoom: 2}} {
			_, g := placed(m, v, sz[0], sz[1])
			seen := map[string]int{}
			for _, n := range names(g) {
				seen[n]++
			}
			for n, k := range seen {
				if k > 1 {
					t.Errorf("%v %+v: %q is written %d times", sz, v, n, k)
				}
			}
		}
	}
	if got := len(m.PlaceLabels()); got != 6 {
		t.Errorf("%d places, want 6 (river, road, lake, meadow, hut, stones)", got)
	}
}

// A label never lies over a road, water, a ford, a bridge, a door, the character,
// the objective, a mark or another label (D120); a place that finds no room is
// left out.
func TestLabelsNeverCoverWhatMatters(t *testing.T) {
	for _, obj := range []*pb.Pos{nil, {X: 190000, Y: 200000}, {X: 160000, Y: 160000}} {
		m := overviewWorld(obj)
		for _, sz := range [][2]int{{60, 20}, {120, 40}, {160, 48}} {
			for _, v := range []View{{}, {Labels: true}, {BigZoom: 1}} {
				before, g := placed(m, v, sz[0], sz[1])
				for i, c := range g.flat {
					if c.k != kPlace {
						continue
					}
					b := before[i]
					if !b.nameable() {
						t.Fatalf("%v %+v obj %v: a label covers %q (kind %d, ground %q)", sz, v, obj, b.r, b.k, b.a)
					}
				}
				if obj != nil {
					oc, or := g.cellOf(obj)
					if g.cells[or][oc].k == kPlace {
						t.Errorf("%v %+v: a label covers the objective", sz, v)
					}
				}
			}
		}
	}
}

// The name of a lake or a river goes beside it, not on it.
func TestLabelOfWaterGoesBesideIt(t *testing.T) {
	m := overviewWorld(nil)
	_, g := placed(m, View{}, 120, 40)
	got := strings.Join(names(g), "|")
	for _, want := range []string{"Mere", "Meadow", "The River", "Long Road"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q is not named at the widest zoom: %s", want, got)
		}
	}
}

// A place too small to matter is named only when zoomed in or with everything
// named (Alt); the Stones span 14 m and the widest map has a cell of several metres.
// The Hut is as small, but a building matters (D121): it is named at any zoom.
func TestSmallPlacesAreNamedOnlyWhenZoomedInOrWithAlt(t *testing.T) {
	m := overviewWorld(nil)
	has := func(v View, name string) bool {
		_, g := placed(m, v, 120, 40)
		for _, n := range names(g) {
			if n == name || strings.HasPrefix(n, name+" +") { // a name may carry the count under it
				return true
			}
		}
		return false
	}
	if has(View{}, "Stones") {
		t.Error("the stones are named at the widest zoom")
	}
	if !has(View{BigZoom: 2}, "Stones") {
		t.Error("the stones are not named when zoomed in")
	}
	if !has(View{Labels: true}, "Stones") {
		t.Error("the stones are not named with Alt held")
	}
	if !has(View{}, "Hut") {
		t.Error("the hut, a building, is not named at the widest zoom")
	}
	if !has(View{}, "Meadow") {
		t.Error("the meadow, four pieces spanning 100 m, is not named at the widest zoom")
	}
}

// The ground (D120): open ground is blank, a road stays and is bold, a forest is
// a stipple, camps are left out at the widest zoom and shown zoomed in.
func TestOverviewGround(t *testing.T) {
	m := overviewWorld(nil)
	count := func(g *grid, k cellKind, r rune) (n int) {
		for _, c := range g.flat {
			if c.k == k && c.r == r {
				n++
			}
		}
		return
	}
	g := overviewGrid(m, View{}, 120, 40)
	if count(g, kGround, glyphFloor) != 0 {
		t.Error("open ground is drawn")
	}
	if count(g, kGround, glyphRoad) == 0 {
		t.Error("the road is not drawn")
	}
	if count(g, kCamp, glyphCamp) != 0 {
		t.Error("a camp is drawn at the widest zoom")
	}
	if g := overviewGrid(m, View{BigZoom: 1}, 120, 40); count(g, kCamp, glyphCamp) == 0 {
		t.Error("the camp is not drawn zoomed in")
	}
	// A stipple: no two forest cells side by side.
	forest := 0
	for _, row := range g.cells {
		for c, cell := range row {
			if cell.r == glyphForest {
				forest++
				if c > 0 && row[c-1].r == glyphForest {
					t.Fatal("the forest is solid quotes, not a stipple")
				}
			}
		}
	}
	if forest == 0 {
		t.Error("the forest is not drawn")
	}
}

// The character is in reverse video, not by colour alone (D120).
func TestSelfIsReverseVideo(t *testing.T) {
	st := ANSI()
	got := cellStyle(kSelf, "", st)("@")
	if !strings.Contains(got, "\x1b[7m") {
		t.Errorf("the character is drawn %q", got)
	}
	if cellStyle(kGround, model.Road, st)("·") == cellStyle(kGround, model.Floor, st)(".") {
		t.Error("a road is drawn like floor")
	}
}

// The legend lists the marks on the map being shown, and no others (D120).
func TestLegendListsOnlyPresentMarks(t *testing.T) {
	st := Plain()
	all := strings.Join(legendLines(nil, 120, st, nil), "\n")
	if !strings.Contains(all, "c camp") || !strings.Contains(all, "= bridge") {
		t.Fatalf("the full legend is missing marks:\n%s", all)
	}
	m := overviewWorld(nil)
	for _, v := range []View{{}, {BigZoom: 1}} {
		g := overviewGrid(m, v, 120, 40)
		c, r := g.cellOf(m.Self.Pos)
		g.put(c, r, glyphSelf, kSelf)
		present := g.marksShown()
		for name, text := range map[string]string{
			"below": strings.Join(legendLines(m, 118, st, present), "\n"),
			"side":  strings.Join(legendSide(sideLegendW, st, present), "\n"),
		} {
			wants := []string{"@ you", "≈ water", "· road"}
			if v.BigZoom == 0 { // zoomed in, the door and the zone's edge are out of the picture
				wants = append(wants, "D door", "░ zone edge")
			}
			for _, want := range wants {
				if !strings.Contains(text, want) {
					t.Errorf("%s %+v lacks %q:\n%s", name, v, want, text)
				}
			}
			for _, not := range []string{"= bridge", ": ford", "$ vendor", "mob", "loot", "floor", "target"} {
				if strings.Contains(text, not) {
					t.Errorf("%s %+v lists %q, which is not on the map:\n%s", name, v, not, text)
				}
			}
			if camp := strings.Contains(text, "c camp"); camp != (v.BigZoom > 0) {
				t.Errorf("%s %+v: camp listed %v", name, v, camp)
			}
		}
	}
}

// The window is used (D120): wide, the legend is beside the map and the map is as
// tall as the box allows; narrow, the legend is below.
func TestOverviewLayout(t *testing.T) {
	m := overviewWorld(nil)
	wide := overviewLines(m, View{}, 198, 44, Plain())
	if len(wide) != 44 {
		t.Fatalf("%d rows", len(wide))
	}
	if !strings.Contains(wide[0], "You") || !strings.Contains(wide[1], "@ you") || !strings.Contains(wide[1], "░") {
		t.Errorf("the legend is not beside the map: %q", wide[1])
	}
	// The map reaches the row above the two lines of text.
	if !strings.Contains(wide[41], "░░░") || !strings.Contains(wide[42], "Z, 400m by 400m") {
		t.Errorf("the map does not fill the height:\n%s", strings.Join(wide, "\n"))
	}
	narrow := overviewLines(m, View{}, 78, 20, Plain())
	if len(narrow) != 20 {
		t.Errorf("%d rows", len(narrow))
	}
	if !strings.Contains(strings.Join(narrow, "\n"), "You       @ you") {
		t.Errorf("the narrow window has no legend below:\n%s", strings.Join(narrow, "\n"))
	}
	// A window with no room for the legend keeps its map and points to the help.
	short := overviewLines(m, View{}, 78, 12, Plain())
	if len(short) != 12 || !strings.Contains(strings.Join(short, "\n"), "page 2 of the help") {
		t.Errorf("the short window:\n%s", strings.Join(short, "\n"))
	}
}
