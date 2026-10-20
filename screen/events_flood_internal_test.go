package screen

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The Events panel keeps its visible lines (D85): a death, then 2,000 routine
// events (a mob walking, a rules tick), and the blows and "X killed you" are still
// there to read, however long the player stays dead; verbose mode merges the
// routine ones back by time.
func TestTheDeathStaysInEventsAfterAFloodOfQuietEvents(t *testing.T) {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{
		Space: "z", Seq: 1,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Level: 1, Hp: 10, MaxHp: 10, Pos: &pb.Pos{}}},
	}}})
	ev := func(ms int64, e *pb.Event) {
		e.Seq, e.GameTimeMs = m.Last()+1, ms
		m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: e}})
	}
	for i := range 3 {
		ev(int64(i)*100, &pb.Event{Body: &pb.Event_Combat{Combat: &pb.Combat{Kind: pb.CombatKind_COMBAT_KIND_HIT, Attacker: "mob", Target: "me", Amount: 3}}})
	}
	ev(400, &pb.Event{Body: &pb.Event_Combat{Combat: &pb.Combat{Kind: pb.CombatKind_COMBAT_KIND_DEATH, Attacker: "mob", Target: "me"}}})
	for i := range 2000 {
		ev(1000+int64(i)*100, &pb.Event{Body: &pb.Event_EntityMoved{EntityMoved: &pb.EntityMoved{Id: "mob", Pos: &pb.Pos{X: int64(i)}}}})
		if i%10 == 0 {
			ev(1000+int64(i)*100, &pb.Event{Body: &pb.Event_RulesTick{RulesTick: &pb.RulesTick{}}})
		}
	}

	rows := strings.Join(logLines(m, View{}, 70, 12, Plain()), "\n")
	for _, want := range []string{"hit Me for 3", "killed Me"} {
		if !strings.Contains(rows, want) {
			t.Errorf("the panel lost %q after 2,000 quiet events:\n%s", want, rows)
		}
	}
	if strings.Contains(rows, "nothing has happened yet") || strings.Contains(rows, "moved to") {
		t.Errorf("the short panel shows the quiet lines or is empty:\n%s", rows)
	}
	var shown, quiet int
	for _, e := range m.Log {
		if e.Hidden() {
			quiet++
		} else {
			shown++
		}
	}
	if shown != 4 || quiet > model.DefaultLogCap {
		t.Errorf("the log holds %d visible and %d quiet lines", shown, quiet)
	}
	all := strings.Join(logLines(m, View{Verbose: true}, 70, 12, Plain()), "\n")
	if !strings.Contains(all, "moved to") {
		t.Errorf("verbose mode shows no quiet line:\n%s", all)
	}
	// The visible ring keeps the last 200 lines when it is full of them.
	for i := range 450 {
		ev(500000+int64(i), &pb.Event{Body: &pb.Event_Combat{Combat: &pb.Combat{Kind: pb.CombatKind_COMBAT_KIND_HIT, Attacker: "mob", Target: "me", Amount: int64(i)}}})
		ev(500000+int64(i), &pb.Event{Body: &pb.Event_EntityMoved{EntityMoved: &pb.EntityMoved{Id: "mob", Pos: &pb.Pos{X: int64(i)}}}})
	}
	shown = 0
	for _, e := range m.Log {
		if !e.Hidden() {
			shown++
		}
	}
	if shown != model.DefaultLogCap {
		t.Errorf("a full log keeps %d visible lines, want %d", shown, model.DefaultLogCap)
	}
}
