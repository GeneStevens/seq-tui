package screen

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// A run of cells of one kind is styled once: the text is the cells', and the
// styler is asked for each run, not each cell (T5).
func TestStyledWritesRuns(t *testing.T) {
	g := newGrid(scale{col: 1, row: 2}, 8, 2, &pb.Pos{})
	row := g.cells[0]
	for i, c := range []cell{
		{}, {}, {r: glyphWall, k: kWall}, {r: glyphWall, k: kWall}, {r: glyphMob, k: kMob},
		{r: glyphWater, k: kGround, a: model.Water}, {r: glyphWater, k: kGround, a: model.Water}, {r: glyphRoad, k: kGround, a: model.Road},
	} {
		row[i] = c
	}
	calls := 0
	st := Plain()
	st.Red = func(s string) string { calls++; return "<" + s + ">" }
	got := g.styled(st)
	if want := "  ##<m>≈≈·"; got[0] != want {
		t.Errorf("row 0 is %q, want %q", got[0], want)
	}
	if got[1] != strings.Repeat(" ", 8) {
		t.Errorf("row 1 is %q", got[1])
	}
	if calls != 1 {
		t.Errorf("the styler was asked %d times for the red run, want 1", calls)
	}
}

// The grid of the drawing before is the next one's: the same memory, blank, and
// the picture is the picture a new grid would have (T5).
func TestAGridIsReusedAndBlank(t *testing.T) {
	m := worldModelInternal()
	h := &Hits{}
	v := View{Hits: h}
	draw := func(x int64) string {
		m.Self.Pos = &pb.Pos{X: x}
		h.begin()
		return strings.Join(mapRows(m, v, 60, 20, Plain()), "\n")
	}
	draw(0)
	first := h.g
	moved := draw(7000)
	if h.g != first {
		t.Error("the second drawing did not reuse the first one's grid")
	}
	if fresh := (func() string {
		f := View{Hits: &Hits{}}
		m.Self.Pos = &pb.Pos{X: 7000}
		return strings.Join(mapRows(m, f, 60, 20, Plain()), "\n")
	})(); moved != fresh {
		t.Errorf("a reused grid draws another picture:\n%s\nfresh:\n%s", moved, fresh)
	}
}

func worldModelInternal() *model.Model {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: "z", ViewRadiusMm: 40000,
		Bounds: &pb.Rect{MinX: -50000, MinY: -50000, MaxX: 50000, MaxY: 50000},
		Walls:  []*pb.Wall{{A: &pb.Pos{X: -30000, Y: 10000}, B: &pb.Pos{X: 30000, Y: 10000}}},
		Areas:  []*pb.Area{{Id: "w", Kind: "water", Poly: []*pb.Pos{{X: 10000, Y: 12000}, {X: 40000, Y: 12000}, {X: 40000, Y: 30000}}}}}}})
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 1,
		Self:     &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Hp: 1, MaxHp: 1, Pos: &pb.Pos{}}},
		Entities: []*pb.EntityState{{Id: "m", Name: "Mob", Kind: pb.EntityKind_ENTITY_KIND_MOB, Pos: &pb.Pos{X: 3000, Y: 3000}}}}}})
	return m
}
