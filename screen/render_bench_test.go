package screen_test

import (
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
