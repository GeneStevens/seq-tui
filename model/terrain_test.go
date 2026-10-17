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

type fixed struct{ t *model.Terrain }

func (f fixed) Terrain(space string) *model.Terrain {
	if space == "Here" {
		return f.t
	}
	return nil
}

// The model finds the terrain of the space it is in, by the space's name.
func TestTheModelFindsTheGroundOfItsSpace(t *testing.T) {
	m := model.New()
	m.UseTerrain(fixed{&model.Terrain{Areas: []model.Area{model.NewArea("r", model.Road, rect(0, 0, 10, 10))}}})
	m.Space, m.Self.Pos = "Here", &pb.Pos{X: 5, Y: 5}
	if m.Underfoot() != model.Road {
		t.Errorf("underfoot %q", m.Underfoot())
	}
	m.Space = "Elsewhere"
	if m.Ground() != nil || m.Underfoot() != "" {
		t.Error("another space has no terrain")
	}
}
