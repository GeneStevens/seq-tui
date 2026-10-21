package model

import (
	"testing"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// D86: an xp reward the kill cap cut is marked on its log entry.
func TestACappedXPRewardIsMarked(t *testing.T) {
	m := New()
	rw := func(seq uint64, paid int64, text string) {
		m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: seq, Text: text,
			Body: &pb.Event_Reward{Reward: &pb.Reward{Recipient: "me", Kind: "xp", Amount: 22, PaidAsLevel: paid}}}}})
	}
	rw(1, 5, "You gain 22 experience (capped: paid as a level 5 creature)")
	rw(2, 0, "You gain 40 experience")
	rw(3, 4, "")
	if len(m.Log) != 3 || !m.Log[0].Capped || m.Log[1].Capped || !m.Log[2].Capped {
		t.Fatalf("capped marks wrong: %+v", m.Log)
	}
	if m.Log[2].Text == "" || m.Log[2].Text[len(m.Log[2].Text)-len("level 4 creature)"):] != "level 4 creature)" {
		t.Errorf("the capped part is not added when the server leaves it out: %q", m.Log[2].Text)
	}
}
