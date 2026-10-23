package app_test

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/app"
	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// benchProgram is the program at 200x50 in a zone with ground, walls and forty
// mobs about the character (T4, T5): the screen a player with a big terminal sees.
func benchProgram() (*app.App, *client.Session) {
	poly := func(x0, y0, x1, y1 int64) []*pb.Pos {
		return []*pb.Pos{{X: x0, Y: y0}, {X: x1, Y: y0}, {X: x1, Y: y1}, {X: x0, Y: y1}}
	}
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: "z", Name: "Z", ViewRadiusMm: 40000,
		Bounds: &pb.Rect{MinX: -200000, MinY: -200000, MaxX: 200000, MaxY: 200000},
		Walls:  []*pb.Wall{{A: &pb.Pos{X: -30000, Y: 10000}, B: &pb.Pos{X: 30000, Y: 10000}}},
		Areas: []*pb.Area{
			{Id: "b", Kind: "building", Poly: poly(-20000, 12000, -8000, 24000)},
			{Id: "w", Kind: "water", Poly: poly(10000, 12000, 40000, 30000)},
			{Id: "r", Kind: "road", Poly: poly(-60000, -2000, 60000, 2000)},
		}}}})
	snap := &pb.Snapshot{Space: "z", Seq: 1,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Level: 1, Hp: 10, MaxHp: 10, Pos: &pb.Pos{}}}}
	for i := range 40 {
		snap.Entities = append(snap.Entities, &pb.EntityState{Id: fmt.Sprintf("m%d", i), Name: "Mob", Kind: pb.EntityKind_ENTITY_KIND_MOB, Level: 1, Hp: 5, MaxHp: 5,
			Pos: &pb.Pos{X: int64(i%8-4) * 7000, Y: int64(i/8-2) * 9000}})
	}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: snap}})
	s := client.NewSession(client.Config{Model: m})
	a := app.New(s)
	a.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	a.View()
	return a, s
}

// BenchmarkMoveThenView is a moved event for the character, and the repaint it
// causes (T5).
func BenchmarkMoveThenView(b *testing.B) {
	a, s := benchProgram()
	b.ReportAllocs()
	x := int64(0)
	for b.Loop() {
		x += 700
		s.View(func(m *model.Model) { m.Self.Pos = &pb.Pos{X: x % 100000} })
		a.Update(tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'}))
		_ = a.View()
	}
}

// BenchmarkHover is the pointer moving over the map, a cell at a time then a
// few motions inside each (T4): what the program does for each motion message.
func BenchmarkHover(b *testing.B) {
	a, _ := benchProgram()
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		i++
		a.Update(tea.MouseMotionMsg{X: 20 + (i/4)%100, Y: 10 + (i/4)%30})
		_ = a.View()
	}
}

// BenchmarkClick is a left click on the ground (T4): the click and the repaint after it.
func BenchmarkClick(b *testing.B) {
	a, _ := benchProgram()
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		i++
		a.Update(tea.MouseClickMsg{X: 60 + i%20, Y: 12, Button: tea.MouseLeft})
		_ = a.View()
	}
}
