package model

import (
	"testing"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Areas with one name are one place (D120): one label, in the middle of their
// combined bounding box.
func TestPlaceLabelsGroupAreasByName(t *testing.T) {
	sq := func(x, y, s int64) []*pb.Pos {
		return []*pb.Pos{{X: x, Y: y}, {X: x + s, Y: y}, {X: x + s, Y: y + s}, {X: x, Y: y + s}}
	}
	m := New()
	m.Map = &pb.ZoneMap{Areas: []*pb.Area{
		{Id: "a", Name: "Hut", Poly: sq(0, 0, 10)},
		{Id: "b", Name: "Lake", Poly: sq(100, 100, 50)},
		{Id: "c", Name: "Hut", Poly: sq(30, 0, 10)},
		{Id: "d", Name: "", Poly: sq(0, 0, 10)},
		{Id: "e", Name: "Hut", Poly: sq(0, 20, 10)},
		{Id: "f", Name: "Empty"},
	}}
	got := m.PlaceLabels()
	if len(got) != 2 || got[0].Name != "Hut" || got[1].Name != "Lake" {
		t.Fatalf("places %+v, want Hut and Lake once each, in order", got)
	}
	hut := got[0]
	if hut.Pos.GetX() != 20 || hut.Pos.GetY() != 15 || hut.DX != 40 || hut.DY != 30 || hut.Areas != 3 || hut.Size != 1200 {
		t.Errorf("the hut is %+v %v", hut, hut.Pos)
	}
	if got[1].Areas != 1 || got[1].Pos.GetX() != 125 {
		t.Errorf("the lake is %+v", got[1])
	}
}
