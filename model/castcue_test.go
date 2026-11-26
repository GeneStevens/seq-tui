package model_test

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
	"google.golang.org/protobuf/encoding/protojson"
)

// castableChief is a copy of seq's castable-chief wire records (testserver/example/
// testdata/castable-chief.txt): the four ways a Warren Chief's Brow Breaker
// ends, as the server sent them, one case to a "## " heading.
const castableChief = "../testdata/castable/castable-chief.txt"

type wireCase struct {
	name   string
	events []*pb.Event
}

var wireLine = regexp.MustCompile(`^\+\d+ ms (\{.*\})$`)

func wireCases(t *testing.T) []wireCase {
	t.Helper()
	f, err := os.Open(castableChief)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []wireCase
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<16), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if name, ok := strings.CutPrefix(line, "## "); ok {
			out = append(out, wireCase{name: strings.Fields(name)[0]})
			continue
		}
		if mm := wireLine.FindStringSubmatch(line); mm != nil {
			e := &pb.Event{}
			if err := protojson.Unmarshal([]byte(mm[1]), e); err != nil {
				t.Fatalf("%s: %v", line, err)
			}
			out[len(out)-1].events = append(out[len(out)-1].events, e)
		}
	}
	if len(out) != 4 {
		t.Fatalf("%d cases in %s", len(out), castableChief)
	}
	return out
}

// chiefWorld is a character (id) 1.8 m from a Warren Chief, in a snapshot at
// game time at.
func chiefWorld(id string, at int64) (*model.Model, uint64) {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "warren", Seq: 1, GameTimeMs: at,
		Self:     &pb.SelfState{Entity: &pb.EntityState{Id: id, Name: "Hero", Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, Level: 5, Hp: 90, MaxHp: 90, Pos: &pb.Pos{}}},
		Entities: []*pb.EntityState{{Id: "mob-chief-1", Name: "Warren Chief", Kind: pb.EntityKind_ENTITY_KIND_MOB, Level: 7, Hp: 300, MaxHp: 300, Pos: &pb.Pos{X: 1800}}},
	}}})
	return m, 1
}

func feed(m *model.Model, seq uint64, events ...*pb.Event) uint64 {
	for _, e := range events {
		e.Seq = seq + 1
		seq++
		m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: e}})
	}
	return seq
}

func heroOf(c wireCase) string { return c.events[0].GetCastStarted().GetTarget() }

// Each wire case ends in its own way, read from the event and its reason.
func TestEachEndingOfTheCastIsItsOwn(t *testing.T) {
	want := map[string]string{
		"lands":   model.EndLanded,
		"kicked":  model.EndInterrupt,
		"stunned": model.EndStunned,
		"stepped": model.EndOutOfReach,
	}
	for _, c := range wireCases(t) {
		first := c.events[0]
		m, seq := chiefWorld(heroOf(c), first.GetGameTimeMs()-100)
		seq = feed(m, seq, first)
		cast := m.CastOf("mob-chief-1")
		if cast == nil || cast.EndsAtMs != first.GetGameTimeMs()+2000 || cast.Target != heroOf(c) || m.CastName(cast.Ability) != "Brow Breaker" {
			t.Fatalf("%s: after the start: %+v", c.name, cast)
		}
		if m.CastEndOf("mob-chief-1") != nil {
			t.Fatalf("%s: an ending before the end", c.name)
		}
		feed(m, seq, c.events[1:]...)
		end := m.CastEndOf("mob-chief-1")
		if m.CastOf("mob-chief-1") != nil || end == nil || end.Kind != want[c.name] {
			t.Errorf("%s: cast %+v end %+v, want %s", c.name, m.CastOf("mob-chief-1"), end, want[c.name])
		}
	}
}

// A miss after the landing says so.
func TestALandedCastThatMissesIsMissed(t *testing.T) {
	c := wireCases(t)[0]
	m, seq := chiefWorld(heroOf(c), 8000)
	seq = feed(m, seq, c.events[0], c.events[1])
	feed(m, seq, &pb.Event{GameTimeMs: 10200, Body: &pb.Event_Combat{Combat: &pb.Combat{Kind: pb.CombatKind_COMBAT_KIND_MISS, Attacker: "mob-chief-1", Target: heroOf(c), Ability: "chief-brow-breaker"}}})
	if end := m.CastEndOf("mob-chief-1"); end == nil || end.Kind != model.EndMissed {
		t.Fatalf("end %+v", end)
	}
}

// The ending comes from the reason: other words in the text change nothing, and an
// unknown reason is a cancel that says why.
func TestAnEndingIsReadFromItsReasonNotItsWords(t *testing.T) {
	for _, tc := range []struct{ reason, kind string }{{"stunned", model.EndStunned}, {"out_of_range", model.EndOutOfReach}, {"target_gone", model.EndCancelled}} {
		m, seq := chiefWorld("rex1", 100)
		seq = feed(m, seq, &pb.Event{GameTimeMs: 1000, Body: &pb.Event_CastStarted{CastStarted: &pb.CastStarted{Actor: "mob-chief-1", Target: "rex1", Ability: "chief-brow-breaker", CastMs: 2000}}})
		feed(m, seq, &pb.Event{GameTimeMs: 1500, Text: "wholly new words", Body: &pb.Event_CastCancelled{CastCancelled: &pb.CastEnded{Actor: "mob-chief-1", Target: "rex1", Ability: "chief-brow-breaker", Reason: tc.reason}}})
		if end := m.CastEndOf("mob-chief-1"); end == nil || end.Kind != tc.kind || end.Reason != tc.reason {
			t.Errorf("%s: %+v", tc.reason, end)
		}
	}
}

// A start that comes late is shortened to what is left, never negative, and never
// restarted by the clock it came at.
func TestADelayedStartIsShortenedNeverRestarted(t *testing.T) {
	m, seq := chiefWorld("rex1", 9000) // the snapshot already says 9000
	feed(m, seq, &pb.Event{GameTimeMs: 8200, Body: &pb.Event_CastStarted{CastStarted: &pb.CastStarted{Actor: "mob-chief-1", Target: "rex1", Ability: "chief-brow-breaker", CastMs: 2000}}})
	c := m.CastOf("mob-chief-1")
	if c == nil {
		t.Fatal("no cast")
	}
	for _, tc := range []struct{ now, left int64 }{{8200, 2000}, {9000, 1200}, {10200, 0}, {11000, 0}} {
		if got := c.Left(tc.now); got != tc.left {
			t.Errorf("left at %d: %d, want %d", tc.now, got, tc.left)
		}
	}
	if !c.Live(10200) || c.Live(10200+model.CastGraceMs) {
		t.Error("a cast is live to its deadline and its grace, and no longer")
	}
}

// A start already past its deadline when it is read has no cue and no ending.
func TestAStaleStartHasNoCue(t *testing.T) {
	m, seq := chiefWorld("rex1", 12000)
	feed(m, seq, &pb.Event{GameTimeMs: 8200, Body: &pb.Event_CastStarted{CastStarted: &pb.CastStarted{Actor: "mob-chief-1", Target: "rex1", Ability: "chief-brow-breaker", CastMs: 2000}}})
	if m.CastOf("mob-chief-1") != nil || m.CastEndOf("mob-chief-1") != nil {
		t.Fatalf("cast %+v end %+v", m.CastOf("mob-chief-1"), m.CastEndOf("mob-chief-1"))
	}
}

// A lost ending does not leave a cue: the next event past the deadline and grace
// drops it.
func TestACastPastItsDeadlineIsDroppedByTheNextEvent(t *testing.T) {
	m, seq := chiefWorld("rex1", 100)
	seq = feed(m, seq, &pb.Event{GameTimeMs: 1000, Body: &pb.Event_CastStarted{CastStarted: &pb.CastStarted{Actor: "mob-chief-1", Target: "rex1", Ability: "chief-brow-breaker", CastMs: 2000}}})
	feed(m, seq, &pb.Event{GameTimeMs: 3000 + model.CastGraceMs, Body: &pb.Event_EntityMoved{EntityMoved: &pb.EntityMoved{Id: "mob-chief-1", Pos: &pb.Pos{X: 1800}}}})
	if m.CastOf("mob-chief-1") != nil {
		t.Fatal("still casting")
	}
}

// A cast of the character's own is the You panel's, not the cue's.
func TestTheCharactersOwnCastIsNotACue(t *testing.T) {
	m, seq := chiefWorld("rex1", 100)
	feed(m, seq, &pb.Event{GameTimeMs: 1000, Body: &pb.Event_CastStarted{CastStarted: &pb.CastStarted{Actor: "rex1", Target: "mob-chief-1", Ability: "wizard-frost-bolt", CastMs: 1500}}})
	if m.CastOf("rex1") != nil {
		t.Fatal("a cue for the character's own cast")
	}
}

func TestACasterThatDiesOrLeavesTakesTheCueWithIt(t *testing.T) {
	for _, body := range []any{
		&pb.Event_Combat{Combat: &pb.Combat{Kind: pb.CombatKind_COMBAT_KIND_DEATH, Target: "mob-chief-1"}},
		&pb.Event_EntityLeft{EntityLeft: &pb.EntityLeft{Id: "mob-chief-1"}},
	} {
		m, seq := chiefWorld("rex1", 100)
		seq = feed(m, seq, &pb.Event{GameTimeMs: 1000, Body: &pb.Event_CastStarted{CastStarted: &pb.CastStarted{Actor: "mob-chief-1", Target: "rex1", Ability: "chief-brow-breaker", CastMs: 2000}}})
		e := &pb.Event{GameTimeMs: 1200}
		switch b := body.(type) {
		case *pb.Event_Combat:
			e.Body = b
		case *pb.Event_EntityLeft:
			e.Body = b
		}
		feed(m, seq, e)
		if m.CastOf("mob-chief-1") != nil {
			t.Errorf("%T: still casting", body)
		}
	}
}

func TestEndingWordsDifferForEachEnding(t *testing.T) {
	seen := map[string]bool{}
	for _, k := range []string{model.EndLanded, model.EndMissed, model.EndInterrupt, model.EndStunned, model.EndOutOfReach, model.EndCancelled} {
		w := model.CastEndWords(&model.CastEndInfo{Kind: k, Reason: "died"}, "Brow Breaker")
		if seen[w] || !strings.Contains(w, "Brow Breaker") {
			t.Errorf("%s: %q", k, w)
		}
		seen[w] = true
	}
}
