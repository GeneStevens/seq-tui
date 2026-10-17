package model_test

import (
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func rect(x0, y0, x1, y1 int64) []*pb.Pos {
	return []*pb.Pos{{X: x0, Y: y0}, {X: x1, Y: y0}, {X: x1, Y: y1}, {X: x0, Y: y1}}
}

// The ground at a point follows doc 02 §4a: a building beats everything, a
// bridge or ford beats the water under it, a road beats the forest it runs
// through, and the edge of an area is inside it.
func TestGroundAtAPoint(t *testing.T) {
	tr := &model.Terrain{Areas: []model.Area{
		model.NewArea("wood", model.Forest, rect(0, 0, 100, 100)),
		model.NewArea("lane", model.Road, rect(40, 0, 60, 100)),
		model.NewArea("river", model.Water, rect(0, 120, 100, 160)),
		model.NewArea("ford", model.Ford, rect(40, 120, 60, 160)),
		model.NewArea("hut", model.Building, rect(45, 45, 55, 55)),
	}}
	for _, tc := range []struct {
		x, y int64
		want model.AreaKind
	}{
		{10, 10, model.Forest},
		{50, 10, model.Road},
		{50, 50, model.Building},
		{40, 10, model.Road}, // on the edge
		{10, 140, model.Water},
		{50, 140, model.Ford},
		{200, 200, ""},
	} {
		if got := tr.Ground(tc.x, tc.y); got != tc.want {
			t.Errorf("ground at (%d, %d) is %q; want %q", tc.x, tc.y, got, tc.want)
		}
	}
	var none *model.Terrain
	if none.Ground(1, 1) != "" || none.HasFloor() {
		t.Error("no terrain is open ground")
	}
}

func TestAConcavePolygonIsNotItsBoundingBox(t *testing.T) {
	l := model.NewArea("l", model.Forest, []*pb.Pos{{X: 0, Y: 0}, {X: 100, Y: 0}, {X: 100, Y: 40}, {X: 40, Y: 40}, {X: 40, Y: 100}, {X: 0, Y: 100}})
	if !l.Contains(10, 90) || !l.Contains(90, 10) || l.Contains(90, 90) {
		t.Error("an L-shaped area holds its two arms and not the corner between them")
	}
}

// The model reads the terrain out of the zone map the server sent, and gives
// none for a map without areas.
func TestTheModelReadsTheGroundFromTheZoneMap(t *testing.T) {
	m := model.New()
	m.Self.Pos = &pb.Pos{X: 5, Y: 5}
	if m.Ground() != nil || m.Underfoot() != "" {
		t.Error("no map, no terrain")
	}
	m.Map = &pb.ZoneMap{
		Areas: []*pb.Area{{Id: "r", Kind: "road", Poly: rect(0, 0, 10, 10)}},
	}
	if m.Underfoot() != model.Road {
		t.Errorf("underfoot %q", m.Underfoot())
	}
	m.Map = &pb.ZoneMap{Walls: []*pb.Wall{{}}}
	if m.Ground() != nil || m.Underfoot() != "" {
		t.Error("a map without areas has no terrain")
	}
}

// The server's word on speed (D55) is used while the map's areas still put the
// character on the ground it named, else only the ground is named.
func TestSpeedWordsFollowTheServerAndStopClaimingOnOtherGround(t *testing.T) {
	m := model.New()
	m.Map = &pb.ZoneMap{Areas: []*pb.Area{{Id: "wood", Kind: "forest", Poly: rect(0, 0, 10, 10)}, {Id: "way", Kind: "road", Poly: rect(20, 0, 30, 10)}}}
	m.Self.Pos, m.Self.Ground, m.Self.SpeedPct = &pb.Pos{X: 5, Y: 5}, "forest", 75
	if got := m.SpeedWords(); got != "forest: slowed to 75%" {
		t.Errorf("in the forest: %q", got)
	}
	m.Self.Pos = &pb.Pos{X: 25, Y: 5} // on the road since the snapshot: the server has said nothing of it
	if got := m.SpeedWords(); got != "road" {
		t.Errorf("walked onto the road: %q", got)
	}
	m.Self.Pos = &pb.Pos{X: 50, Y: 50}
	if got := m.SpeedWords(); got != "" {
		t.Errorf("out in the open: %q", got)
	}
	m.Self.Ground, m.Self.SpeedPct = "road", 125
	m.Self.Pos = &pb.Pos{X: 25, Y: 5}
	if got := m.SpeedWords(); got != "road: faster, 125%" {
		t.Errorf("on the road, told: %q", got)
	}
	m.Self.Ground, m.Self.SpeedPct = "open", 100
	m.Self.Pos = &pb.Pos{X: 50, Y: 50}
	if got := m.SpeedWords(); got != "" {
		t.Errorf("normal speed: %q", got)
	}
}
