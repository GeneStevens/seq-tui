package main

import (
	"strings"
	"testing"
)

// --- M20260412-26 encounter lifecycle and combat legibility tests ---

func TestEncounterOpenedEventText(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.playerRead = playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}

	enc := encounterSummary{EncounterID: "enc-1", State: "Active", ActionIndex: 0, TimelineLength: 0}
	updated, _ := m.Update(encounterReadResultMsg{result: encounterReadResult{
		State: encounterReadOK, Count: 1, Encounters: []encounterSummary{enc},
	}})
	um := updated.(model)

	entries := um.events.recent()
	if len(entries) == 0 || entries[0].text != "enc:opened" {
		t.Fatalf("first encounter event should be 'enc:opened', got: %v", entries)
	}
}

func TestVictoryEventText(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.playerRead = playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	m.lastSeenEncID = "enc-1"
	m.lastSeenEncState = "Active"
	m.lastSeenActIdx = 5

	enc := encounterSummary{EncounterID: "enc-1", State: "Completed", CompletedReason: "all_mobs_dead", ActionIndex: 5}
	updated, _ := m.Update(encounterReadResultMsg{result: encounterReadResult{
		State: encounterReadOK, Count: 1, Encounters: []encounterSummary{enc},
	}})
	um := updated.(model)

	entries := um.events.recent()
	found := false
	for _, e := range entries {
		if e.text == "VICTORY" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected VICTORY event, got: %v", entries)
	}
}

func TestDefeatEventText(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.playerRead = playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	m.lastSeenEncID = "enc-1"
	m.lastSeenEncState = "Active"
	m.lastSeenActIdx = 3

	enc := encounterSummary{EncounterID: "enc-1", State: "Completed", CompletedReason: "all_players_dead", ActionIndex: 3}
	updated, _ := m.Update(encounterReadResultMsg{result: encounterReadResult{
		State: encounterReadOK, Count: 1, Encounters: []encounterSummary{enc},
	}})
	um := updated.(model)

	entries := um.events.recent()
	if len(entries) == 0 || entries[0].text != "DEFEAT" {
		t.Fatalf("expected DEFEAT event, got: %v", entries)
	}
}

func TestMobDefeatedEvent(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.playerRead = playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	m.lastSeenEncID = "enc-1"
	m.lastSeenEncState = "Active"
	m.lastSeenActIdx = 2
	m.lastSeenMobsDead = 0

	enc := encounterSummary{EncounterID: "enc-1", State: "Active", ActionIndex: 2, MobsDead: 1, TimelineLength: 3}
	updated, _ := m.Update(encounterReadResultMsg{result: encounterReadResult{
		State: encounterReadOK, Count: 1, Encounters: []encounterSummary{enc},
	}})
	um := updated.(model)

	entries := um.events.recent()
	found := false
	for _, e := range entries {
		if e.text == "mob defeated" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'mob defeated' event when MobsDead increases, got: %v", entries)
	}
}

func TestDenialDedup(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.playerRead = playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	m.lastSeenEncID = "enc-1"
	m.lastSeenEncState = "Active"
	m.lastSeenActIdx = 0
	m.lastSeenTimelineLen = 1
	m.lastDenialKind = ""

	// First denial — should appear
	enc1 := encounterSummary{EncounterID: "enc-1", State: "Active", ActionIndex: 0, TimelineLength: 2, LatestDenialKind: "out_of_range"}
	updated, _ := m.Update(encounterReadResultMsg{result: encounterReadResult{
		State: encounterReadOK, Count: 1, Encounters: []encounterSummary{enc1},
	}})
	m = updated.(model)

	// Same denial again — should be deduped
	enc2 := encounterSummary{EncounterID: "enc-1", State: "Active", ActionIndex: 0, TimelineLength: 3, LatestDenialKind: "out_of_range"}
	updated, _ = m.Update(encounterReadResultMsg{result: encounterReadResult{
		State: encounterReadOK, Count: 1, Encounters: []encounterSummary{enc2},
	}})
	m = updated.(model)

	entries := m.events.recent()
	denialCount := 0
	for _, e := range entries {
		if e.text == "deny range" {
			denialCount++
		}
	}
	if denialCount != 1 {
		t.Fatalf("repeated identical denials should be deduped, got %d denial events", denialCount)
	}
}

func TestCombatPanelShowsEncounterState(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1, ActionIndex: 0,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "[active]") {
		t.Fatalf("combat panel should show [active] for active encounter, got: %s", stripped)
	}
}

func TestCombatPanelIdleState(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, encounterReadResult{}, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "[idle]") {
		t.Fatalf("combat panel should show [idle] when no encounter, got: %s", stripped)
	}
}

func TestDeadMobShownInRoster(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{"orc-1"},
		MobThreat: []mobThreatEntry{
			{MobID: "orc-1", HPCurrent: 0, HPMax: 100},
		},
	}
	lines := renderCombatMobRoster(enc, attackResult{}, "p1", 20, "")
	if len(lines) == 0 {
		t.Fatal("roster should have entries")
	}
	found := false
	for _, l := range lines {
		if strings.Contains(stripANSI(l), "(dead)") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("dead mob should show (dead) in roster, got: %v", lines)
	}
}
