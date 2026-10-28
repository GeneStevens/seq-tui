package model_test

import (
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The ferry's words (SEQ-193): a trip within the zone is travel, a failed
// crossing's fare is returned, and a ferry that cannot cross is refused.
func TestFerryTravelRefundAndRefusalAreInTheLog(t *testing.T) {
	const (
		sail     = "You sail with the Ferryman to the Far Shore"
		returned = "The ferry can't cross right now; your fare is returned."
	)
	tests := []struct {
		name, kind, text string
		msg              *pb.ServerMessage
	}{
		{"travel in the server's words", "traveled", sail, ferryEvent(&pb.Event{Text: sail, Body: &pb.Event_Traveled{Traveled: &pb.Traveled{Actor: "me", Pos: &pb.Pos{X: 5}}}})},
		{"travel with no words names the place", "traveled", "Aria travelled to Far Shore", ferryEvent(&pb.Event{Body: &pb.Event_Traveled{Traveled: &pb.Traveled{Actor: "me", Place: "Far Shore"}}})},
		{"a refunded fare in the server's words", "reward", returned, ferryEvent(&pb.Event{Text: returned, Body: &pb.Event_Reward{Reward: &pb.Reward{Recipient: "me", Kind: "refund", Amount: 10}}})},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := refusedModel()
			m.Apply(tc.msg)
			last := m.Log[len(m.Log)-1]
			if last.Kind != tc.kind || last.Text != tc.text || last.Hidden() {
				t.Fatalf("log: %+v", m.Log)
			}
			if m.Self.Dead || m.Self.HP != 10 {
				t.Errorf("travel changed life: dead %v hp %d", m.Self.Dead, m.Self.HP)
			}
		})
	}
	if m := refusedModel(); true {
		m.IntentSent("f", 1, "travel")
		m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_IntentResult{IntentResult: &pb.IntentResult{IntentId: "f", Reason: "ferry_down", Text: "The ferry can't cross right now"}}})
		if got := refusals(m); len(got) != 1 || got[0].Text != "The ferry can't cross right now" {
			t.Fatalf("refusal: %+v", m.Log)
		}
	}
	if got := model.Words("ferry_down"); got == "ferry_down" {
		t.Errorf("no words for ferry_down: %q", got)
	}
}

func ferryEvent(e *pb.Event) *pb.ServerMessage {
	e.Seq, e.GameTimeMs = 2, 6000
	return &pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: e}}
}
