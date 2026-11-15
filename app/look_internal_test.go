package app

import (
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The look key reads the figure nearest the large map's cursor, else nearest the
// character; with nobody in view it only says so.
func TestLookReadsTheNearestFigure(t *testing.T) {
	m := model.New()
	m.Self.Pos = &pb.Pos{X: 0, Y: 0}
	if a := lookAction(m, nil); a.Kind == Look || a.Note == "" {
		t.Errorf("nobody in view: %+v", a)
	}
	m.Entities["near"] = &pb.EntityState{Id: "near", Pos: &pb.Pos{X: 3000, Y: 0}}
	m.Entities["far"] = &pb.EntityState{Id: "far", Pos: &pb.Pos{X: 30000, Y: 0}}
	for _, c := range []struct {
		at   *pb.Pos
		want string
	}{{nil, "near"}, {&pb.Pos{X: 29000, Y: 0}, "far"}} {
		if a := lookAction(m, c.at); a.Kind != Look || a.ID != c.want {
			t.Errorf("at %v: %+v, want %s", c.at, a, c.want)
		}
	}
}
