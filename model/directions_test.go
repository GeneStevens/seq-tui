package model_test

import (
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func asking() *model.Model {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "town", Seq: 1,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me"}},
		Npcs: []*pb.NpcInfo{{Id: "guard", Name: "Town Guard", Role: "guard"}}}}})
	return m
}

// The list is the asker's, by NPC; Directions is offered only when the NPC's
// last words carried the option.
func TestDirectionsListsAndTheOptionThatLeadsToThem(t *testing.T) {
	m := asking()
	event(m, 100, &pb.Event{Body: &pb.Event_Directions{Directions: &pb.DirectionsList{Actor: "bram", Npc: "guard", Entries: []*pb.DirectionEntry{{Id: "x"}}}}})
	if m.DirectionsOf("guard") != nil {
		t.Error("another character's list was kept")
	}
	event(m, 200, &pb.Event{Body: &pb.Event_Directions{Directions: &pb.DirectionsList{Actor: "me", Npc: "guard", Entries: []*pb.DirectionEntry{{Id: "trainers:t1", Group: "trainers", Name: "Gorran"}}}}})
	if got := m.DirectionsOf("guard").GetEntries(); len(got) != 1 || got[0].GetId() != "trainers:t1" {
		t.Errorf("the list: %v", got)
	}
	for _, tc := range []struct {
		options []*pb.DialogueOption
		want    bool
	}{
		{nil, false},
		{[]*pb.DialogueOption{{Action: "buy"}}, false},
		{[]*pb.DialogueOption{{Action: "buy"}, {Action: "directions", Label: "Directions"}}, true},
	} {
		event(m, 300, &pb.Event{Body: &pb.Event_Dialogue{Dialogue: &pb.Dialogue{Actor: "me", Npc: "guard", Options: tc.options}}})
		if got := m.GivesDirections("guard"); got != tc.want {
			t.Errorf("options %v: gives directions %v, want %v", tc.options, got, tc.want)
		}
	}
}

// The destination is two kinds of hint: it is there while they say so, the door
// stands for it in another zone, and the hints that follow take it away.
func TestTheDestinationIsTheHintsAndArrivalEndsIt(t *testing.T) {
	m := asking()
	hints := func(h ...*pb.Hint) {
		event(m, 100, &pb.Event{Body: &pb.Event_ObjectiveHints{ObjectiveHints: &pb.ObjectiveHints{Actor: "me", Hints: h}}})
	}
	if m.Destination() != nil || m.DestinationWaypoint() != nil {
		t.Fatal("a destination from nothing")
	}
	near := &pb.Hint{Kind: "destination", Id: "n1", Label: "Ottilie Penn", Pos: &pb.Pos{X: 5}}
	hints(near, &pb.Hint{Kind: "objective", Label: "Talk to Gorran"})
	if m.Destination() != near || m.DestinationWaypoint() != near {
		t.Errorf("near: %v, %v", m.Destination(), m.DestinationWaypoint())
	}
	door := &pb.Hint{Kind: "destination_waypoint", Label: "East Gate — to the Thornwick Downs", Pos: &pb.Pos{X: 9}}
	hints(&pb.Hint{Kind: "destination", Label: "Barleycross", Zone: "Thornwick Downs", Pos: &pb.Pos{X: 9}}, door)
	if m.DestinationWaypoint() != door || model.DoorName(door) != "East Gate" {
		t.Errorf("another zone: %v, door %q", m.DestinationWaypoint(), model.DoorName(door))
	}
	event(m, 200, &pb.Event{Text: "You have reached Ottilie Penn.", Body: &pb.Event_DestinationEnded{DestinationEnded: &pb.DestinationEnded{Actor: "me", Reason: "reached", Name: "Ottilie Penn", Text: "You have reached Ottilie Penn."}}})
	if got := lastText(m); got != "You have reached Ottilie Penn." {
		t.Errorf("the arrival line: %q", got)
	}
	hints()
	if m.Destination() != nil {
		t.Error("the destination outlived the hints that dropped it")
	}
}
