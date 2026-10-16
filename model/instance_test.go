package model_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func readLog(t *testing.T, name string) []*pb.ServerMessage {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "testdata", "logs", name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	msgs, err := record.Read(f)
	if err != nil {
		t.Fatal(err)
	}
	return msgs
}

// replayTo applies the first n messages of a recorded log.
func replayTo(t *testing.T, name string, n int) *model.Model {
	t.Helper()
	m := model.New()
	for i, msg := range readLog(t, name) {
		if n > 0 && i >= n {
			break
		}
		m.Apply(msg)
	}
	return m
}

func TestMenuIsKeptFromTheWorldZoneMap(t *testing.T) {
	m := replayTo(t, "live_solo_trial.jsonl", 3)
	if len(m.Presets) != 5 || m.Preset("quick-turns").GetTimeMode() != "turn_based" || m.Preset("quick-turns").GetTurnDeadlineMs() != 20000 {
		t.Fatalf("presets %v", m.Presets)
	}
	if m.InInstance() || m.TurnBased() || m.Opened != nil || m.Round != nil {
		t.Error("the world zone is not an instance")
	}
}

func TestSoloTrialOpenedEnteredClearedHandedBack(t *testing.T) {
	m := replayTo(t, "live_solo_trial.jsonl", 386)
	if m.Opened == nil || m.Opened.Preset != "solo-trial" || m.Opened.PartyCap != 1 || m.InInstance() {
		t.Fatalf("opened, not entered: %+v in %v", m.Opened, m.InInstance())
	}
	m = replayTo(t, "live_solo_trial.jsonl", 395)
	if !m.InInstance() || m.InstanceName() != "Solo Trial" || m.TurnBased() || m.Current().GetTimeMode() != "realtime" {
		t.Errorf("inside: %q %v", m.Space, m.Current())
	}
	if len(m.Presets) != 5 {
		t.Error("the menu was lost when the character went in")
	}
	m = replayTo(t, "live_solo_trial.jsonl", 0)
	if m.InInstance() || m.Opened != nil || m.Ended == nil || m.Ended.Reason != "cleared" || m.Ended.Preset != "solo-trial" {
		t.Errorf("after the hand-back: ended %+v, in instance %v", m.Ended, m.InInstance())
	}
	if m.Space != "Testgrounds" || len(m.Faults) != 0 {
		t.Errorf("space %q faults %v", m.Space, m.Faults)
	}
	last := m.Log[len(m.Log)-2]
	if last.Kind != "instance_ended" || last.Reason != "cleared" || !strings.Contains(last.Text, "cleared") {
		t.Errorf("log: %+v", last)
	}
}

func TestRoundsOfQuickTurns(t *testing.T) {
	m := replayTo(t, "live_quick_turns.jsonl", 391)
	r := m.Round
	if r == nil || !r.Open || r.Number != 2 || len(r.Submitted) != 0 || !m.TurnBased() || m.InstanceName() != "Quick Turns" {
		t.Fatalf("round %+v", r)
	}
	if r.DeadlineMs == 0 {
		t.Error("no deadline")
	}
	// Round 2 ran on its plan, round 3 opened.
	m = replayTo(t, "live_quick_turns.jsonl", 433)
	if r = m.Round; !r.Open || r.Number != 3 || r.Resolved != 2 || r.ResolvedBy != "all_submitted" {
		t.Errorf("round %+v", r)
	}
	if !contains(r.ResolvedSubmitted, "me-1") {
		t.Errorf("own plan not seen: %+v", r)
	}
	// Round 5 was missed: resolved by the deadline with no plan of ours.
	m = replayTo(t, "live_quick_turns.jsonl", 478)
	r = m.Round
	if r.Open || r.Number != 5 || r.ResolvedBy != "deadline" || contains(r.ResolvedSubmitted, "me-1") {
		t.Errorf("missed round %+v", r)
	}
	if e := m.Log[len(m.Log)-1]; e.Kind != "round_resolved" || e.Reason != "deadline" || !strings.Contains(e.Text, "safe default") {
		t.Errorf("log: %+v", e)
	}
	// All of it, to the end: cleared, handed back, no round left.
	m = replayTo(t, "live_quick_turns.jsonl", 0)
	if m.Round != nil || m.InInstance() || m.TurnBased() || m.Ended == nil || m.Ended.Reason != "cleared" || len(m.Faults) != 0 {
		t.Errorf("end: round %+v ended %+v faults %v", m.Round, m.Ended, m.Faults)
	}
}

func contains(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// A fresh login into the open round (the instance may have slept): the snapshot
// does not carry the round, the repeated round_started does, and it is logged once.
func TestReconnectRestoresTheRound(t *testing.T) {
	m := replayTo(t, "live_quick_turns.jsonl", 459) // the second welcome, before its snapshot
	n := 0
	for _, e := range m.Log {
		if e.Kind == "round_started" {
			n++
		}
	}
	m = replayTo(t, "live_quick_turns.jsonl", 462)
	if r := m.Round; r == nil || !r.Open || r.Number != 4 || !m.TurnBased() || m.Connections != 3 {
		t.Fatalf("round %+v connections %d", r, m.Connections)
	}
	k := 0
	for _, e := range m.Log {
		if e.Kind == "round_started" {
			k++
		}
	}
	if k != n {
		t.Errorf("the repeated round_started was logged again: %d then %d", n, k)
	}
}

// A plan already accepted shows as submitted when the round is told again.
func TestAcceptedPlanShowsAsSubmittedAfterReconnect(t *testing.T) {
	m := replayTo(t, "live_quick_turns.jsonl", 391)
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: m.Last() + 1, Body: &pb.Event_RoundStarted{RoundStarted: &pb.RoundStarted{
		Round: m.Round.Number, DeadlineMs: m.Round.DeadlineMs, Submitted: []string{"me-1"}}}}}})
	if !m.Round.Open || !m.Round.HasSubmitted(m.Self.ID) {
		t.Errorf("round %+v", m.Round)
	}
}

func TestInstanceOpenFailedIsLoggedInWords(t *testing.T) {
	m := replayTo(t, "live_quick_turns.jsonl", 391)
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: m.Last() + 1, Body: &pb.Event_InstanceOpenFailed{InstanceOpenFailed: &pb.InstanceOpenFailed{Reason: "instance_exists", Instance: "inst-1"}}}}})
	e := m.Log[len(m.Log)-1]
	if e.Kind != "instance_open_failed" || e.Reason != "instance_exists" || !strings.Contains(e.Text, "already has an instance open") || !strings.Contains(e.Text, "inst-1") {
		t.Errorf("%+v", e)
	}
}

func TestWords(t *testing.T) {
	for code, want := range map[string]string{
		"party_too_large": "larger than this instance allows",
		"plan_required":   "build a plan",
		"round_closed":    "plan for the next round",
		"no_instance":     "open one first",
		"not_a_code":      "",
	} {
		got := model.Words(code)
		if want == "" {
			if got != code {
				t.Errorf("%s: %q", code, got)
			}
			continue
		}
		if !strings.Contains(got, want) || !strings.HasSuffix(got, "("+code+")") {
			t.Errorf("%s: %q", code, got)
		}
	}
}
