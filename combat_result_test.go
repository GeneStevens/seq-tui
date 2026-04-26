package main

import (
	"testing"
)

// --- M20260412-24 combat result and denial truth surfacing tests ---

func TestEventTextForResultDenialKinds(t *testing.T) {
	cases := map[string]string{
		"out_of_range":     "deny range",
		"los_blocked":      "deny los",
		"invalid_target":   "deny target",
		"cooldown_blocked": "deny cooldown",
		"blocked_wounded":  "blocked",
		"damage_applied":   "hit 42",
		"attack_miss":      "miss",
	}
	for kind, expected := range cases {
		value := 0
		if kind == "damage_applied" {
			value = 42
		}
		got := eventTextForResult(kind, value)
		if got != expected {
			t.Errorf("eventTextForResult(%q, %d) = %q, want %q", kind, value, got, expected)
		}
	}
}

func TestDenialEventDetectedByTimelineGrowth(t *testing.T) {
	// Simulate: encounter with ActionIndex=0, TimelineLength=1 (initial)
	// Then TimelineLength grows to 2 while ActionIndex stays 0 → denial
	m := model{slog: &sessionLogger{}}
	m.lastSeenEncID = "enc-1"
	m.lastSeenActIdx = 0
	m.lastSeenTimelineLen = 1
	m.lastSeenEncState = "Active"
	m.playerRead = playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}

	// Simulate encounter refresh with a denial event
	enc := encounterSummary{
		EncounterID:      "enc-1",
		State:            "Active",
		ActionIndex:      0, // unchanged — no applied action
		TimelineLength:   2, // grew — denial event was added
		LatestDenialKind: "out_of_range",
	}
	encounters := encounterReadResult{
		State:      encounterReadOK,
		Encounters: []encounterSummary{enc},
		Count:      1,
	}
	m.encounterRead = encounters

	// Process the encounter refresh
	updated, _ := m.Update(encounterReadResultMsg{result: encounters})
	um := updated.(model)

	// The event stream should contain the denial
	entries := um.events.recent()
	found := false
	for _, e := range entries {
		if e.text == "deny range" {
			found = true
			break
		}
	}
	if !found {
		texts := make([]string, len(entries))
		for i, e := range entries {
			texts[i] = e.text
		}
		t.Fatalf("expected 'deny range' in event stream, got: %v", texts)
	}
}

func TestCombatResultEventDetectedByActionIndexAdvance(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.lastSeenEncID = "enc-1"
	m.lastSeenActIdx = 0
	m.lastSeenTimelineLen = 0
	m.lastSeenEncState = "Active"
	m.playerRead = playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}

	enc := encounterSummary{
		EncounterID:      "enc-1",
		State:            "Active",
		ActionIndex:      1, // advanced
		TimelineLength:   1,
		LatestResultKind: "damage_applied",
		LatestResultValue: 25,
	}
	encounters := encounterReadResult{
		State:      encounterReadOK,
		Encounters: []encounterSummary{enc},
		Count:      1,
	}

	updated, _ := m.Update(encounterReadResultMsg{result: encounters})
	um := updated.(model)

	entries := um.events.recent()
	found := false
	for _, e := range entries {
		if e.text == "hit 25" {
			found = true
			break
		}
	}
	if !found {
		texts := make([]string, len(entries))
		for i, e := range entries {
			texts[i] = e.text
		}
		t.Fatalf("expected 'hit 25' in event stream, got: %v", texts)
	}
}
