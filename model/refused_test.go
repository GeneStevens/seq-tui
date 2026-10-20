package model_test

import (
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

const gateShut = "Gate North is shut: no expedition is open beyond it. Start one first."

func refusedModel() *model.Model {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "Z", Seq: 1, GameTimeMs: 5000,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Aria", Hp: 10, MaxHp: 10}}}}})
	return m
}

func denied(m *model.Model, id string) {
	m.IntentSent(id, 1, "use_door")
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_IntentResult{IntentResult: &pb.IntentResult{IntentId: id, Reason: "no_instance", Text: gateShut}}})
}

func refusals(m *model.Model) (out []model.LogEntry) {
	for _, e := range m.Log {
		if e.Refused() {
			out = append(out, e)
		}
	}
	return out
}

func TestARefusedIntentIsALogLineInTheServersWords(t *testing.T) {
	m := refusedModel()
	denied(m, "i1")
	got := refusals(m)
	if len(got) != 1 || got[0].Text != gateShut || got[0].Hidden() {
		t.Fatalf("log: %+v", m.Log)
	}
	// No code, no internal prefix, and an accepted intent adds nothing.
	m.IntentSent("i2", 2, "move")
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_IntentResult{IntentResult: &pb.IntentResult{IntentId: "i2", Accepted: true}}})
	if len(refusals(m)) != 1 {
		t.Errorf("an accepted intent is in the log: %+v", m.Log)
	}
	// Without the server's text the reason's words are used.
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_IntentResult{IntentResult: &pb.IntentResult{IntentId: "i3", Reason: "no_such_door"}}})
	if got := refusals(m); len(got) != 2 || got[1].Text == "" {
		t.Errorf("a refusal with no text: %+v", got)
	}
}

func TestAnIdenticalRefusalWithinTwoSecondsIsOneLine(t *testing.T) {
	m := refusedModel()
	denied(m, "i1")
	denied(m, "i2") // the same moment: the same line
	if n := len(refusals(m)); n != 1 {
		t.Fatalf("%d lines: %+v", n, m.Log)
	}
	// The denial event that says the same is the same line, before or after.
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: 2, GameTimeMs: 5500, Text: gateShut,
		Body: &pb.Event_Denied{Denied: &pb.Denied{Actor: "me", Intent: "use_door", Reason: "no_instance"}}}}})
	if n := len(refusals(m)); n != 1 {
		t.Fatalf("the denial event repeats the line: %+v", m.Log)
	}
	// Past two seconds it is told again.
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: 3, GameTimeMs: 9000, Text: gateShut,
		Body: &pb.Event_Denied{Denied: &pb.Denied{Actor: "me", Intent: "use_door", Reason: "no_instance"}}}}})
	if n := len(refusals(m)); n != 2 {
		t.Fatalf("a later refusal was dropped: %+v", m.Log)
	}
}

func TestDoorStatesSayWhichDoorsAreUsable(t *testing.T) {
	m := refusedModel()
	if u, _ := m.DoorUsable("gate-north"); !u {
		t.Error("a door the server has said nothing of is taken as usable")
	}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_DoorStates{DoorStates: &pb.DoorStates{Doors: []*pb.DoorState{{Id: "gate-north", Why: gateShut}, {Id: "line-east", Usable: true}}}}})
	if u, why := m.DoorUsable("gate-north"); u || why != gateShut {
		t.Errorf("gate-north: %v %q", u, why)
	}
	if u, _ := m.DoorUsable("line-east"); !u {
		t.Error("line-east")
	}
	// A new message replaces the old; a new space forgets them.
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_DoorStates{DoorStates: &pb.DoorStates{Doors: []*pb.DoorState{{Id: "gate-north", Usable: true}}}}})
	if u, _ := m.DoorUsable("gate-north"); !u {
		t.Error("the door opened")
	}
}
