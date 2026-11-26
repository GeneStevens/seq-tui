package screen_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// chiefFight is Rex facing a Warren Chief (the target), the chief's Brow Breaker
// begun at game time 8200 on Rex, who stands chiefX mm from it.
func chiefFight(chiefX int64) (*model.Model, uint64) {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: "warren", Name: "Warren", ViewRadiusMm: 20000, Bounds: &pb.Rect{MaxX: 40000, MaxY: 40000}}}})
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "warren", Seq: 1, GameTimeMs: 8000,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "rex1", Name: "Rex", Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, Level: 5, Hp: 90, MaxHp: 90, Pos: &pb.Pos{X: 10000, Y: 10000}}},
		Entities: []*pb.EntityState{
			{Id: "mob-chief-1", Name: "Warren Chief", Kind: pb.EntityKind_ENTITY_KIND_MOB, Level: 7, Hp: 300, MaxHp: 300, Pos: &pb.Pos{X: 10000 + chiefX, Y: 10000}},
			{Id: "pat1", Name: "Pat", Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, Level: 5, Hp: 90, MaxHp: 90, Pos: &pb.Pos{X: 10000, Y: 12000}},
		}}}})
	m.Self.Target = "mob-chief-1"
	seq := send(m, 1, &pb.Event{GameTimeMs: 8200, Body: &pb.Event_CastStarted{CastStarted: &pb.CastStarted{Actor: "mob-chief-1", Target: "rex1", Ability: "chief-brow-breaker", CastMs: 2000}}})
	return m, seq
}

func send(m *model.Model, seq uint64, e *pb.Event) uint64 {
	e.Seq = seq + 1
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: e}})
	return e.Seq
}

func targetPanel(t *testing.T, m *model.Model, gameNow int64, w, h int) string {
	t.Helper()
	out := screen.Render(m, screen.View{Conn: screen.ConnOnline, GameNow: gameNow}, w, h, screen.Plain())
	var rows []string
	for _, l := range strings.Split(out, "\n") {
		rows = append(rows, strings.TrimRight(l, " "))
	}
	return strings.Join(rows, "\n")
}

// The cue names the move (never its owner-prefixed id), who it is on, and the
// time left, from the game time it began at.
func TestTheTargetPanelShowsTheMoveItsTargetAndTheTimeLeft(t *testing.T) {
	m, _ := chiefFight(1800)
	out := targetPanel(t, m, 0, 140, 45)
	for _, s := range []string{"Casting Brow Breaker on you", "2.1s"} {
		if !strings.Contains(out, s) {
			t.Errorf("missing %q in\n%s", s, out)
		}
	}
	if strings.Contains(out, "chief-brow-breaker") {
		t.Error("the owner-prefixed id is shown")
	}
	// The local clock runs the countdown on between the server's messages.
	if out := targetPanel(t, m, 8200+1500, 140, 45); !strings.Contains(out, "0.6s") {
		t.Errorf("no 0.5s with the local clock at +1.5 s:\n%s", out)
	}
}

// A cast on someone else names them.
func TestTheCueNamesAnotherTarget(t *testing.T) {
	m, seq := chiefFight(1800)
	send(m, seq, &pb.Event{GameTimeMs: 8300, Body: &pb.Event_CastStarted{CastStarted: &pb.CastStarted{Actor: "mob-chief-1", Target: "pat1", Ability: "chief-brow-breaker", CastMs: 2000}}})
	if out := targetPanel(t, m, 0, 140, 45); !strings.Contains(out, "Casting Brow Breaker on Pat") {
		t.Errorf("not on Pat:\n%s", out)
	}
}

// The cue's countdown follows the server's game time: a newer game time from an
// event puts a drifted local clock right.
func TestTheCountdownIsPutRightByANewerGameTime(t *testing.T) {
	m, seq := chiefFight(1800)
	if out := targetPanel(t, m, 8200+900, 140, 45); !strings.Contains(out, "1.2s") { // the local clock reads 0.9 s in
		t.Fatalf("not 1.1s left:\n%s", out)
	}
	send(m, seq, &pb.Event{GameTimeMs: 9400, Body: &pb.Event_EntityMoved{EntityMoved: &pb.EntityMoved{Id: "pat1", Pos: &pb.Pos{X: 10000, Y: 12000}}}})
	if out := targetPanel(t, m, 0, 140, 45); !strings.Contains(out, "0.9s") { // the server says 1.2 s in
		t.Errorf("not 0.8s left:\n%s", out)
	}
}

// Brow Breaker's reach is stated, labelled as the client's own, with where the
// character stands to it.
func TestTheReachAndTheCharactersRelationToIt(t *testing.T) {
	in, _ := chiefFight(1800)
	if out := targetPanel(t, in, 0, 140, 45); !strings.Contains(out, "Reach 2.5 m (client-side figure)") || !strings.Contains(out, "You: 1.8 m, in reach") {
		t.Errorf("in reach:\n%s", out)
	}
	far, _ := chiefFight(3100)
	if out := targetPanel(t, far, 0, 140, 45); !strings.Contains(out, "You: 3.1 m, OUT of reach (predicted)") {
		t.Errorf("out of reach:\n%s", out)
	}
}

// Out of reach now is a prediction: the cue stays, and goes only with the real
// ending; stepping back in clears the warning.
func TestOutOfReachNowKeepsTheCueUntilTheRealEnd(t *testing.T) {
	m, seq := chiefFight(1800)
	seq = send(m, seq, &pb.Event{GameTimeMs: 8600, Body: &pb.Event_EntityMoved{EntityMoved: &pb.EntityMoved{Id: "rex1", Pos: &pb.Pos{X: 6900, Y: 10000}}}})
	out := targetPanel(t, m, 0, 140, 45)
	if !strings.Contains(out, "OUT of reach (predicted)") || !strings.Contains(out, "Casting Brow Breaker on you") || strings.Contains(out, "OUT OF REACH: it fell short") {
		t.Errorf("out of reach now:\n%s", out)
	}
	send(m, seq, &pb.Event{GameTimeMs: 9000, Body: &pb.Event_EntityMoved{EntityMoved: &pb.EntityMoved{Id: "rex1", Pos: &pb.Pos{X: 10000, Y: 10000}}}})
	if out := targetPanel(t, m, 0, 140, 45); strings.Contains(out, "OUT of reach (predicted)") || !strings.Contains(out, "in reach") {
		t.Errorf("back in reach:\n%s", out)
	}
	// Past the deadline (10200) and before the ending: the cue is still there.
	if out := targetPanel(t, m, 10200+200, 140, 45); !strings.Contains(out, "Casting Brow Breaker on you") || !strings.Contains(out, "0.1s") {
		t.Errorf("at the deadline:\n%s", out)
	}
	// Past the grace with no ending: it goes.
	if out := targetPanel(t, m, 10200+model.CastGraceMs+100, 140, 45); strings.Contains(out, "Casting Brow Breaker") {
		t.Errorf("after the grace:\n%s", out)
	}
}

// Each ending reads differently on the panel.
func TestEachEndingReadsDifferently(t *testing.T) {
	end := func(body any, at int64) string {
		m, seq := chiefFight(1800)
		e := &pb.Event{GameTimeMs: at}
		switch b := body.(type) {
		case *pb.Event_AbilityUsed:
			e.Body = b
		case *pb.Event_CastInterrupted:
			e.Body = b
		case *pb.Event_CastCancelled:
			e.Body = b
		}
		send(m, seq, e)
		out := targetPanel(t, m, 0, 140, 45)
		if strings.Contains(out, "Casting Brow Breaker") {
			t.Errorf("%T: the cue is still up after the end:\n%s", body, out)
		}
		return out
	}
	ended := func(r string) *pb.CastEnded {
		return &pb.CastEnded{Actor: "mob-chief-1", Target: "rex1", Ability: "chief-brow-breaker", Reason: r}
	}
	for _, tc := range []struct {
		name string
		body any
		at   int64
		want string
	}{
		{"landed", &pb.Event_AbilityUsed{AbilityUsed: &pb.AbilityUsed{Actor: "mob-chief-1", Target: "rex1", Ability: "chief-brow-breaker"}}, 10200, "Brow Breaker LANDED"},
		{"interrupted", &pb.Event_CastInterrupted{CastInterrupted: ended("interrupt")}, 8500, "Brow Breaker INTERRUPTED"},
		{"stunned", &pb.Event_CastCancelled{CastCancelled: ended("stunned")}, 8500, "Brow Breaker STUNNED: cast broken"},
		{"out of reach", &pb.Event_CastCancelled{CastCancelled: ended("out_of_range")}, 10200, "Brow Breaker OUT OF REACH"},
	} {
		if out := end(tc.body, tc.at); !strings.Contains(out, tc.want) {
			t.Errorf("%s: missing %q:\n%s", tc.name, tc.want, out)
		}
	}
}

// The ending is shown for a few seconds, then the plain target.
func TestAnEndingGoesAfterAFewSeconds(t *testing.T) {
	m, seq := chiefFight(1800)
	send(m, seq, &pb.Event{GameTimeMs: 8500, Body: &pb.Event_CastInterrupted{CastInterrupted: &pb.CastEnded{Actor: "mob-chief-1", Target: "rex1", Ability: "chief-brow-breaker", Reason: "interrupt"}}})
	if out := targetPanel(t, m, 8500+1000, 140, 45); !strings.Contains(out, "INTERRUPTED") {
		t.Error("the ending is gone at once")
	}
	if out := targetPanel(t, m, 8500+model.CastEndShownMs+1, 140, 45); strings.Contains(out, "INTERRUPTED") {
		t.Error("the ending stays")
	}
}

// The cue fits the small screen too, in the room the Target panel has.
func TestTheCueFitsTheSmallScreen(t *testing.T) {
	m, _ := chiefFight(3100)
	out := targetPanel(t, m, 0, 80, 24)
	if !strings.Contains(out, "Brow Breaker on you") {
		t.Errorf("no cue at 80x24:\n%s", out)
	}
	for _, l := range strings.Split(out, "\n") {
		if n := len([]rune(l)); n > 80 {
			t.Errorf("a line is %d cells: %q", n, l)
		}
	}
}
