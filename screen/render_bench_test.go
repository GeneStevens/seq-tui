package screen_test

import (
	"fmt"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// fullLogModel is a model in play whose event log has been full for a while.
func fullLogModel() *model.Model {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{
		Space: "z", Seq: 1,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Level: 1, Hp: 10, MaxHp: 10, Pos: &pb.Pos{}}},
	}}})
	for i := range 3 * model.DefaultLogCap {
		for _, b := range []*pb.Event{
			{Body: &pb.Event_Combat{Combat: &pb.Combat{Kind: pb.CombatKind_COMBAT_KIND_HIT, Attacker: "mob", Target: "me", Amount: 3}}},
			{Body: &pb.Event_EntityMoved{EntityMoved: &pb.EntityMoved{Id: "mob", Pos: &pb.Pos{X: int64(i)}}}},
		} {
			b.Seq, b.GameTimeMs = m.Last()+1, int64(i)*100
			m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: b}})
		}
	}
	return m
}

// BenchmarkRenderFullLog is one frame at 200x50 with a full event log (T2).
func BenchmarkRenderFullLog(b *testing.B) {
	m := fullLogModel()
	b.ReportAllocs()
	for b.Loop() {
		_ = screen.Render(m, screen.View{}, 200, 50, screen.Plain())
	}
}

// worldModel is a model in play in a zone with ground, walls, camps and forty
// mobs about the character: what the map has to draw at 200x50 (T5).
func worldModel() *model.Model {
	m := model.New()
	poly := func(x0, y0, x1, y1 int64) []*pb.Pos {
		return []*pb.Pos{{X: x0, Y: y0}, {X: x1, Y: y0}, {X: x1, Y: y1}, {X: x0, Y: y1}}
	}
	zm := &pb.ZoneMap{Space: "z", Name: "Z", ViewRadiusMm: 40000,
		Bounds: &pb.Rect{MinX: -200000, MinY: -200000, MaxX: 200000, MaxY: 200000},
		Walls:  []*pb.Wall{{A: &pb.Pos{X: -30000, Y: 10000}, B: &pb.Pos{X: 30000, Y: 10000}}},
		Camps:  []*pb.Camp{{Pos: &pb.Pos{X: 15000, Y: -15000}}},
		Areas: []*pb.Area{
			{Id: "b", Kind: "building", Poly: poly(-20000, 12000, -8000, 24000)},
			{Id: "w", Kind: "water", Poly: poly(10000, 12000, 40000, 30000)},
			{Id: "f", Kind: "forest", Poly: poly(-40000, -30000, -10000, -10000)},
			{Id: "r", Kind: "road", Poly: poly(-60000, -2000, 60000, 2000)},
		},
	}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: zm}})
	snap := &pb.Snapshot{Space: "z", Seq: 1,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Level: 1, Hp: 10, MaxHp: 10, Pos: &pb.Pos{}}}}
	for i := range 40 {
		snap.Entities = append(snap.Entities, &pb.EntityState{Id: fmt.Sprintf("m%d", i), Name: "Mob", Kind: pb.EntityKind_ENTITY_KIND_MOB, Level: 1, Hp: 5, MaxHp: 5,
			Pos: &pb.Pos{X: int64(i%8-4) * 7000, Y: int64(i/8-2) * 9000}})
	}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: snap}})
	return m
}

// BenchmarkRenderMap200x50 is one frame of the world at 200x50 with the
// character standing still (T5).
func BenchmarkRenderMap200x50(b *testing.B) {
	m := worldModel()
	v := screen.View{Hits: &screen.Hits{}}
	b.ReportAllocs()
	for b.Loop() {
		_ = screen.Render(m, v, 200, 50, screen.Plain())
	}
}

// BenchmarkRenderMapMoving is the same frame after the character has moved a
// cell: the map's static layer is drawn again.
func BenchmarkRenderMapMoving(b *testing.B) {
	m := worldModel()
	v := screen.View{Hits: &screen.Hits{}}
	b.ReportAllocs()
	x := int64(0)
	for b.Loop() {
		x += 700
		m.Self.Pos = &pb.Pos{X: x % 100000}
		_ = screen.Render(m, v, 200, 50, screen.Plain())
	}
}
