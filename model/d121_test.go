package model_test

import (
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func sq(x, y, s int64) []*pb.Pos {
	return []*pb.Pos{{X: x, Y: y}, {X: x + s, Y: y}, {X: x + s, Y: y + s}, {X: x, Y: y + s}}
}

func area(id, name, kind string, poly []*pb.Pos) *pb.Area {
	return &pb.Area{Id: id, Name: name, Kind: kind, Poly: poly}
}

// The cluster rule (D121): pieces within model.ClusterGapMm of one another, hop by
// hop, are one place named at the middle of all; otherwise it is named beside its
// largest piece.
func TestClusterRule(t *testing.T) {
	m := 1000
	for _, tc := range []struct {
		name      string
		areas     []*pb.Area
		scattered bool
		x, y      int64 // where the name goes, metres
	}{
		{"a ring of four banks, 21 to 33 m apart", []*pb.Area{
			area("1", "Ring", "rock", sq(40000, 300000, 3000)), area("2", "Ring", "rock", sq(100000, 304000, 3000)),
			area("3", "Ring", "rock", sq(76000, 332000, 3000)), area("4", "Ring", "rock", sq(52000, 322000, 3000))}, false, 71, 317},
		{"one piece", []*pb.Area{area("1", "Hut", "building", sq(10000, 10000, 8000))}, false, 14, 14},
		{"two copses a field apart", []*pb.Area{
			area("1", "Copse", "forest", sq(0, 0, 10000)), area("2", "Copse", "forest", sq(200000, 0, 30000))}, true, 215, 15},
		{"a chain whose ends are far apart", []*pb.Area{
			area("1", "Wall", "rock", sq(0, 0, 4000)), area("2", "Wall", "rock", sq(40000, 0, 4000)),
			area("3", "Wall", "rock", sq(80000, 0, 4000)), area("4", "Wall", "rock", sq(120000, 0, 4000))}, false, 62, 2},
		{"just over the gap", []*pb.Area{
			area("1", "Pair", "rock", sq(0, 0, 4000)), area("2", "Pair", "rock", sq(4000+model.ClusterGapMm+1, 0, 5000))}, true, 46, 2},
	} {
		mod := model.New()
		mod.Map = &pb.ZoneMap{Areas: tc.areas}
		got := mod.PlaceLabels()
		if len(got) != 1 {
			t.Fatalf("%s: %d places", tc.name, len(got))
		}
		p := got[0]
		if p.Scattered != tc.scattered || p.Pos.GetX()/int64(m) != tc.x || p.Pos.GetY()/int64(m) != tc.y {
			t.Errorf("%s: scattered %v, named at %d,%d m; want %v at %d,%d", tc.name, p.Scattered, p.Pos.GetX()/int64(m), p.Pos.GetY()/int64(m), tc.scattered, tc.x, tc.y)
		}
	}
}

// What makes a place matter (D121), and the order of it.
func TestPlaceSignificance(t *testing.T) {
	zm := &pb.ZoneMap{
		Areas: []*pb.Area{
			area("a", "Stones", "rock", sq(0, 0, 3000)),
			area("b", "Hut", "building", sq(20000, 0, 6000)),
			area("c", "Gate Yard", "town", sq(40000, 0, 10000)),
			area("d", "Hamlet", "town", sq(60000, 0, 10000)),
			area("e", "Well", "town", sq(80000, 0, 10000)),
			area("f", "Field", "town", sq(100000, 0, 10000)),
			area("g", "Lane", "road", sq(120000, 0, 10000)),
		},
		Doors: []*pb.Door{{Id: "gate", Name: "Gate", Pos: &pb.Pos{X: 45000, Y: 5000}}},
		Npcs: []*pb.NpcPlace{
			{Id: "q", Name: "Quentin", Role: "quest_giver", Pos: &pb.Pos{X: 65000, Y: 5000}},
			{Id: "v", Name: "Vera", Role: "vendor", Pos: &pb.Pos{X: 85000, Y: 5000}},
			{Id: "r", Name: "Rolf", Role: "vendor", Pos: &pb.Pos{X: 125000, Y: 5000}}, // on a road: the road does not matter for it
		},
	}
	mod := model.New()
	mod.Map = zm
	mod.Self.Hints = []*pb.Hint{{Kind: "objective", Id: "o", Label: "Go", Pos: &pb.Pos{X: 105000, Y: 5000}}}
	sig := map[string]int{}
	for _, p := range mod.PlaceLabels() {
		sig[p.Name] = p.Sig
	}
	want := map[string]int{"Stones": model.SigSize, "Hut": model.SigDoorOrBuilding, "Gate Yard": model.SigDoorOrBuilding, "Hamlet": model.SigQuestPerson,
		"Well": model.SigNPC, "Field": model.SigObjective, "Lane": model.SigSize}
	for name, w := range want {
		if sig[name] != w {
			t.Errorf("%s matters %d, want %d", name, sig[name], w)
		}
	}
	if !(model.SigObjective > model.SigQuestPerson && model.SigQuestPerson > model.SigNPC && model.SigNPC > model.SigDoorOrBuilding && model.SigDoorOrBuilding > model.SigSize) {
		t.Error("the order is not objective, quest person, other NPC, door or building, size")
	}
	places := mod.PlaceLabels()
	model.RankPlaces(places)
	if places[0].Name != "Field" || places[1].Name != "Hamlet" || places[2].Name != "Well" {
		t.Errorf("ranked %s, %s, %s", places[0].Name, places[1].Name, places[2].Name)
	}
}

// A cell says it is over a place by the place's ground, not its bounding box.
func TestPlaceTouchesACellByItsGround(t *testing.T) {
	l := model.PlaceLabel{Polys: [][]*pb.Pos{{{X: 0, Y: 0}, {X: 100, Y: 0}, {X: 0, Y: 100}}}} // a triangle
	for _, tc := range []struct {
		x0, y0, x1, y1 int64
		want           bool
	}{
		{10, 10, 20, 20, true},  // inside
		{-5, -5, 5, 5, true},    // over a corner
		{60, 60, 80, 80, false}, // in the box, beyond the long edge
		{45, 45, 55, 55, true},  // the edge crosses it
		{200, 200, 210, 210, false},
	} {
		if got := l.Touches(tc.x0, tc.y0, tc.x1, tc.y1); got != tc.want {
			t.Errorf("%v: %v", tc, got)
		}
	}
}

// Lines are stamped with the local wall-clock time, 24-hour to the tenth (D121).
func TestStampFormat(t *testing.T) {
	at := time.Date(2026, 11, 1, 16, 4, 3, 270_000_000, time.Local)
	for _, tc := range []struct {
		t      time.Time
		approx bool
		want   string
	}{
		{at, false, "16:04:03.2"},
		{time.Date(2026, 11, 1, 0, 5, 9, 0, time.Local), false, "00:05:09.0"},
		{at, true, "~16:04:03 "},
		{time.Time{}, false, "--:--:--.-"},
	} {
		got := model.Stamp(tc.t, tc.approx)
		if got != tc.want || len(got) != model.StampWidth {
			t.Errorf("Stamp(%v, %v) = %q, want %q", tc.t, tc.approx, got, tc.want)
		}
	}
	if got := model.StampGame(at, false, 127445600); got != "16:04:03.2 (game 127445.6s)" {
		t.Errorf("StampGame %q", got)
	}
}

// A line is stamped with the time the client received it; one replayed at login
// gets the time worked out from the game time and is marked; with no clock it has
// no time.
func TestLinesAreStampedWithTheLocalTime(t *testing.T) {
	defer model.FixClock()
	now := model.FixedTime()
	model.Clock = func() time.Time { return now }
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 1, GameTimeMs: 100_000,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me"}}}}})
	ev := func(seq uint64, ms int64) {
		m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: seq, GameTimeMs: ms, Text: "x", Body: &pb.Event_TargetCleared{TargetCleared: &pb.TargetCleared{Actor: "me"}}}}})
	}
	ev(2, 100_500) // live
	ev(3, 40_000)  // replayed: 60.5 s of game time before the newest
	last := func() model.LogEntry { return m.Log[len(m.Log)-1] }
	if e := m.Log[len(m.Log)-2]; e.Approx || !e.At.Equal(now) {
		t.Errorf("a live line is stamped %v approx %v", e.At, e.Approx)
	}
	if e := last(); !e.Approx || !e.At.Equal(now.Add(-60_500*time.Millisecond)) {
		t.Errorf("a replayed line is stamped %v approx %v", e.At, e.Approx)
	}
	model.Clock = nil
	m.Notice("no clock")
	if e := last(); !e.At.IsZero() || model.Stamp(e.At, e.Approx) != "--:--:--.-" {
		t.Errorf("without a clock the stamp is %q", model.Stamp(e.At, e.Approx))
	}
}
