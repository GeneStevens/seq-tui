package model_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// replayTo applies a recorded log, or the part of it before a mark ("file@mark").
func replayTo(t *testing.T, spec string) *model.Model {
	t.Helper()
	msgs, err := record.Load(filepath.Join("..", "testdata", "logs"), spec)
	if err != nil {
		t.Fatal(err)
	}
	m := model.New()
	for _, msg := range msgs {
		m.Apply(msg)
	}
	return m
}

func TestMenuIsKeptFromTheWorldZoneMap(t *testing.T) {
	m := replayTo(t, "solo_trial.jsonl@instance_menu")
	if len(m.Presets) != 5 || m.Preset("quick-turns").GetTimeMode() != "turn_based" || m.Preset("quick-turns").GetTurnDeadlineMs() != 20000 {
		t.Fatalf("presets %v", m.Presets)
	}
	if m.Preset("quick-turns").GetSleepLimitMs() == 0 || m.Preset("solo-trial").GetSleepLimitMs() != 0 {
		t.Errorf("sleep limits: turn-based %d, real-time %d", m.Preset("quick-turns").GetSleepLimitMs(), m.Preset("solo-trial").GetSleepLimitMs())
	}
	if m.InInstance() || m.TurnBased() || m.Opened != nil || m.Round != nil || m.Instance != nil {
		t.Error("the world zone is not an instance")
	}
}

func TestSoloTrialOpenedEnteredClearedHandedBack(t *testing.T) {
	m := replayTo(t, "solo_trial.jsonl@instance_opened")
	if m.Opened == nil || m.Opened.Preset != "solo-trial" || m.Opened.PartyCap != 1 || m.InInstance() {
		t.Fatalf("opened, not entered: %+v in %v", m.Opened, m.InInstance())
	}
	m = replayTo(t, "solo_trial.jsonl@instance_inside")
	if !m.InInstance() || m.InstanceName() != "Solo Trial" || m.TurnBased() || m.Current().GetTimeMode() != "realtime" {
		t.Errorf("inside: %q %v", m.Space, m.Current())
	}
	// What the instance is comes from the server, not from the space's name (D49).
	if in := m.Instance; in.GetPreset() != "solo-trial" || in.GetPartyCap() != 1 || in.GetTimeMode() != "realtime" || in.GetDifficulty() != "heroic" || in.GetSleepLimitMs() != 0 {
		t.Errorf("instance info %v", in)
	}
	m.Space = "Somewhere else entirely"
	if m.InstanceName() != "Solo Trial" || !m.InInstance() {
		t.Error("the instance was read from the space's name")
	}
	if len(m.Presets) != 5 {
		t.Error("the menu was lost when the character went in")
	}
	m = replayTo(t, "solo_trial.jsonl")
	if m.InInstance() || m.Opened != nil || m.Ended == nil || m.Ended.Reason != "cleared" || m.Ended.Preset != "solo-trial" || m.Instance != nil {
		t.Errorf("after the hand-back: ended %+v, in instance %v", m.Ended, m.InInstance())
	}
	if m.Space != "Testgrounds" || len(m.Faults) != 0 {
		t.Errorf("space %q faults %v", m.Space, m.Faults)
	}
	found := false
	for _, e := range m.Log {
		if e.Kind == "instance_ended" && e.Reason == "cleared" && strings.Contains(e.Text, "cleared") {
			found = true
		}
	}
	if !found {
		t.Errorf("no instance_ended in the log: %+v", m.Log)
	}
}

func TestRoundsOfQuickTurns(t *testing.T) {
	m := replayTo(t, "quick_turns.jsonl@round_plan")
	r := m.Round
	if r == nil || !r.Open || r.Number != 1 || r.Participants != 1 || len(r.Submitted) != 0 || !m.TurnBased() || m.InstanceName() != "Quick Turns" {
		t.Fatalf("round %+v", r)
	}
	if in := m.Instance; in.GetTimeMode() != "turn_based" || in.GetPartyCap() != 1 || in.GetSleepLimitMs() != 7*24*3600*1000 {
		t.Errorf("instance info %v", in)
	}
	if r.DeadlineMs == 0 {
		t.Error("no deadline")
	}
	// Round 1 ran on our plan, round 2 opened; the round log says what we played.
	m = replayTo(t, "quick_turns.jsonl@round_ran")
	if r = m.Round; !r.Open || r.Number != 2 || r.Resolved != 1 || r.ResolvedBy != "all_submitted" {
		t.Errorf("round %+v", r)
	}
	if !contains(r.ResolvedSubmitted, "aria") || r.PlayedBy("aria") != "own_plan" || r.ResolvedParticipants != 1 {
		t.Errorf("own plan not seen: %+v", r)
	}
	if e := m.Log[len(m.Log)-2]; e.Kind != "round_resolved" || !strings.Contains(e.Text, "1 of 1 submitted") || !strings.Contains(e.Text, "you played your own plan") {
		t.Errorf("log: %+v", e)
	}
	// Round 2 was missed: resolved by the deadline with no plan of ours.
	m = replayTo(t, "quick_turns.jsonl@round_missed")
	r = m.Round
	if !r.Open || r.Number != 3 || r.Resolved != 2 || r.ResolvedBy != "deadline" || contains(r.ResolvedSubmitted, "aria") || r.PlayedBy("aria") != "safe_default" {
		t.Errorf("missed round %+v", r)
	}
	if e := m.Log[len(m.Log)-2]; e.Kind != "round_resolved" || e.Reason != "deadline" || !strings.Contains(e.Text, "0 of 1 submitted") || !strings.Contains(e.Text, "you played the safe default") {
		t.Errorf("log: %+v", e)
	}
	// All of it, to the end: cleared, handed back, no round left.
	m = replayTo(t, "quick_turns.jsonl")
	if m.Round != nil || m.InInstance() || m.TurnBased() || m.Ended == nil || m.Ended.Reason != "cleared" || len(m.Faults) != 0 {
		t.Errorf("end: round %+v ended %+v faults %v", m.Round, m.Ended, m.Faults)
	}
}

// A duo entering together (D49): the round waits for one, then for two, and the
// round log says what each character played.
func TestGatheringRoundOfADuo(t *testing.T) {
	m := replayTo(t, "duo_turns.jsonl@duo_gathered")
	if r := m.Round; r == nil || r.Number != 1 || r.Participants != 2 || len(r.Submitted) != 0 {
		t.Fatalf("round %+v", m.Round)
	}
	var started []string
	for _, e := range m.Log {
		if e.Kind == "round_started" {
			started = append(started, e.Text)
		}
	}
	if len(started) != 2 || !strings.Contains(started[0], "0 of 1 submitted") || !strings.Contains(started[1], "0 of 2 submitted") {
		t.Errorf("round_started was sent again as the second character joined, and logged each time: %q", started)
	}
	m = replayTo(t, "duo_turns.jsonl")
	r := m.Round
	if r.Resolved != 1 || r.ResolvedParticipants != 2 || len(r.ResolvedSubmitted) != 2 || r.PlayedBy("aria") != "own_plan" || r.PlayedBy("bram") != "own_plan" {
		t.Fatalf("round %+v", r)
	}
	var ran string
	for _, e := range m.Log {
		if e.Kind == "round_resolved" {
			ran = e.Text
		}
	}
	if !strings.Contains(ran, "2 of 2 submitted") || !strings.Contains(ran, "you played your own plan") || !strings.Contains(ran, "played its own plan") {
		t.Errorf("round log %q", ran)
	}
}

func TestStandInAndSafeDefaultAreSaidExactly(t *testing.T) {
	m := replayTo(t, "duo_turns.jsonl@duo_gathered")
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: m.Last() + 1, Body: &pb.Event_RoundResolved{RoundResolved: &pb.RoundResolved{
		Round: 1, Reason: "deadline", Submitted: []string{"aria"}, Participants: 3,
		Plays: []*pb.RoundPlay{{Character: "aria", Source: "own_plan"}, {Character: "bram", Source: "stand_in"}, {Character: "cleo", Source: "safe_default"}}}}}}})
	e := m.Log[len(m.Log)-1]
	for _, want := range []string{"1 of 3 submitted", "you played your own plan", "played a stand-in plan", "cleo played the safe default"} {
		if !strings.Contains(e.Text, want) {
			t.Errorf("%q lacks %q", e.Text, want)
		}
	}
	if m.Round.PlayedBy("bram") != "stand_in" || m.Round.PlayedBy("cleo") != "safe_default" || m.Round.PlayedBy("nobody") != "" {
		t.Errorf("plays %+v", m.Round.Plays)
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
	count := func(m *model.Model) (n int) {
		for _, e := range m.Log {
			if e.Kind == "round_started" {
				n++
			}
		}
		return
	}
	m := replayTo(t, "quick_turns.jsonl@round_resumed")
	n := count(m)
	m = replayTo(t, "quick_turns.jsonl@round_fresh")
	if r := m.Round; r == nil || !r.Open || r.Number != 3 || !m.TurnBased() || m.Connections != 3 || m.Instance.GetPreset() != "quick-turns" {
		t.Fatalf("round %+v connections %d", r, m.Connections)
	}
	if k := count(m); k != n {
		t.Errorf("the repeated round_started was logged again: %d then %d", n, k)
	}
}

// A plan already accepted shows as submitted when the round is told again.
func TestAcceptedPlanShowsAsSubmittedAfterReconnect(t *testing.T) {
	m := replayTo(t, "quick_turns.jsonl@round_plan")
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: m.Last() + 1, Body: &pb.Event_RoundStarted{RoundStarted: &pb.RoundStarted{
		Round: m.Round.Number, DeadlineMs: m.Round.DeadlineMs, Submitted: []string{"aria"}, Participants: 1}}}}})
	if !m.Round.Open || !m.Round.HasSubmitted(m.Self.ID) {
		t.Errorf("round %+v", m.Round)
	}
}

func TestInstanceOpenFailedIsLoggedInWords(t *testing.T) {
	m := replayTo(t, "quick_turns.jsonl@round_plan")
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
