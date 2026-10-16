package app_test

import (
	"testing"

	"github.com/GeneStevens/seq-tui/app"
	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func standing(npcs ...*pb.NpcInfo) *model.Model {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "town", Seq: 1,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Pos: &pb.Pos{X: 1000, Y: 1000}}}, Npcs: npcs}}})
	return m
}

// G talks to the nearest person the character has seen, whatever they do; with
// nobody seen, it says so and sends nothing. K, M open screens and send nothing.
func TestTalkKeyPicksTheNearestPerson(t *testing.T) {
	if a := app.Plan("g", standing()); a.Kind != app.None || a.Note != "nobody here to talk to" {
		t.Errorf("nobody: %+v", a)
	}
	m := standing(
		&pb.NpcInfo{Id: "far", Name: "Far", Role: "vendor", Pos: &pb.Pos{X: 9000, Y: 1000}},
		&pb.NpcInfo{Id: "near", Name: "Near", Role: "quest_giver", Pos: &pb.Pos{X: 2000, Y: 1000}},
	)
	if a := app.Plan("g", m); a.Kind != app.Talk || a.NPC != "near" || a.Note != "Near" {
		t.Errorf("two people: %+v", a)
	}
	if a := app.Plan("k", m); a.Kind != app.OpenContracts {
		t.Errorf("k: %+v", a)
	}
	if a := app.Plan("m", m); a.Kind != app.OpenOverview {
		t.Errorf("m: %+v", a)
	}
}
