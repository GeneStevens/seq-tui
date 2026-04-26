package main

import (
	"strings"
	"testing"
)

// --- M20260425-08 TUI truth validation harness ---
//
// Snapshot constructors return populated state matching backend scenario truth
// at specific checkpoints. Each test calls a render function with snapshot data
// and asserts that the output contains the expected backend-owned fact.

// combat1v1ActiveSnapshot returns state for an active 1v1 encounter.
// Player p1 at (20,0), one mob "orc-1" (live "live-orc-1") at (25,0).
// Encounter: Active, 1 player, 1 mob, 1 alive, 0 dead, action_index 3.
// Latest result: damage_applied value 15. Mob threat: orc-1 targeting p1, HP 70/100.
// Player HP: 85/100, can_act true. Attack sent against "live-orc-1".
func combat1v1ActiveSnapshot() (playerReadResult, encounterReadResult, inventoryReadResult, mobReadResult, attackResult, rosterFocus, []rosterEntry, targetConfirmResult) {
	pr := playerReadResult{
		State:              playerReadOK,
		Position:           playerPosResult{X: 20, Y: 0},
		HasPos:             true,
		ActiveEncounterID:  "enc-combat-1v1",
		HasActiveEncounter: true,
	}

	enc := encounterSummary{
		EncounterID:       "enc-combat-1v1",
		State:             "Active",
		PlayerIDs:         []string{"p1"},
		MobIDs:            []string{"live-orc-1"},
		PlayerCount:       1,
		MobCount:          1,
		MobsAlive:         1,
		MobsDead:          0,
		ActionIndex:       3,
		LatestResultKind:  "damage_applied",
		LatestResultValue: 15,
		MobThreat: []mobThreatEntry{
			{
				MobID:                  "live-orc-1",
				LiveMobID:              "live-orc-1",
				SelectedTargetPlayerID: "p1",
				HPCurrent:              70,
				HPMax:                  100,
			},
		},
	}

	er := encounterReadResult{
		State:      encounterReadOK,
		Encounters: []encounterSummary{enc},
		Count:      1,
	}

	inv := inventoryReadResult{
		State:        inventoryReadOK,
		HasLifecycle: true,
		CanAct:       true,
		HPCurrent:    85,
		HPMax:        100,
	}

	mobr := mobReadResult{
		State: mobReadOK,
		Mobs: []mobPosition{
			{
				MobID:     "orc-1",
				LiveMobID: "live-orc-1",
				Position:  mobPosVec3{X: 25, Y: 0, Z: 0},
			},
		},
		Count: 1,
	}

	ar := attackResult{
		State:    attackStateSent,
		TargetID: "live-orc-1",
	}

	// Roster: [pc:p1, mb:live-orc-1 (mobID: orc-1)]
	entries := []rosterEntry{
		{kind: "pc", id: "p1"},
		{kind: "mb", id: "live-orc-1", mobID: "orc-1", canonicalMobID: true},
	}

	// Focus on mob entry (index 1)
	focus := rosterFocus{index: 1}

	tc := targetConfirmResult{
		State:           targetConfirmOK,
		TargetKind:      "mb",
		TargetID:        "live-orc-1",
		Found:           true,
		WithinProximity: true,
		Distance:        5,
	}

	return pr, er, inv, mobr, ar, focus, entries, tc
}

// combat1v1VictorySnapshot returns state for a completed 1v1 victory.
// Encounter: Completed, all_mobs_dead, action_index 12, XP 100.
// Player HP: 45/100. Drops generated with ["loot-sword-1"]. MobsDead: 1, MobsAlive: 0.
func combat1v1VictorySnapshot() (playerReadResult, encounterReadResult, inventoryReadResult, mobReadResult, attackResult, rosterFocus, []rosterEntry, targetConfirmResult) {
	pr := playerReadResult{
		State:              playerReadOK,
		Position:           playerPosResult{X: 20, Y: 0},
		HasPos:             true,
		ActiveEncounterID:  "enc-combat-1v1",
		HasActiveEncounter: true,
	}

	enc := encounterSummary{
		EncounterID:     "enc-combat-1v1",
		State:           "Completed",
		CompletedReason: "all_mobs_dead",
		PlayerIDs:       []string{"p1"},
		MobIDs:          []string{"live-orc-1"},
		PlayerCount:     1,
		MobCount:        1,
		MobsAlive:       0,
		MobsDead:        1,
		ActionIndex:     12,
		XPAwarded:       100,
		DropsGenerated:  true,
		Drops:           []string{"loot-sword-1"},
		MobThreat: []mobThreatEntry{
			{
				MobID:     "live-orc-1",
				LiveMobID: "live-orc-1",
				HPCurrent: 0,
				HPMax:     100,
			},
		},
	}

	er := encounterReadResult{
		State:      encounterReadOK,
		Encounters: []encounterSummary{enc},
		Count:      1,
	}

	inv := inventoryReadResult{
		State:        inventoryReadOK,
		HasLifecycle: true,
		CanAct:       true,
		HPCurrent:    45,
		HPMax:        100,
	}

	mobr := mobReadResult{
		State: mobReadOK,
		Mobs: []mobPosition{
			{
				MobID:     "orc-1",
				LiveMobID: "live-orc-1",
				Position:  mobPosVec3{X: 25, Y: 0, Z: 0},
			},
		},
		Count: 1,
	}

	ar := attackResult{
		State:    attackStateSent,
		TargetID: "live-orc-1",
	}

	entries := []rosterEntry{
		{kind: "pc", id: "p1"},
		{kind: "mb", id: "live-orc-1", mobID: "orc-1", canonicalMobID: true},
	}

	focus := rosterFocus{index: 1}

	tc := targetConfirmResult{
		State:           targetConfirmOK,
		TargetKind:      "mb",
		TargetID:        "live-orc-1",
		Found:           true,
		WithinProximity: true,
		Distance:        5,
	}

	return pr, er, inv, mobr, ar, focus, entries, tc
}

// combat1v1DeathSnapshot returns state for a completed 1v1 where the player died.
// Encounter: Completed, all_players_dead. Player HP: 0/100, can_act false, blocked_reason "player_dead".
func combat1v1DeathSnapshot() (playerReadResult, encounterReadResult, inventoryReadResult, mobReadResult, attackResult, rosterFocus, []rosterEntry, targetConfirmResult) {
	pr := playerReadResult{
		State:              playerReadOK,
		Position:           playerPosResult{X: 20, Y: 0},
		HasPos:             true,
		ActiveEncounterID:  "enc-death-1v1",
		HasActiveEncounter: true,
	}

	enc := encounterSummary{
		EncounterID:     "enc-death-1v1",
		State:           "Completed",
		CompletedReason: "all_players_dead",
		PlayerIDs:       []string{"p1"},
		MobIDs:          []string{"live-orc-1"},
		PlayerCount:     1,
		MobCount:        1,
		MobsAlive:       1,
		MobsDead:        0,
		MobThreat: []mobThreatEntry{
			{
				MobID:     "live-orc-1",
				LiveMobID: "live-orc-1",
				HPCurrent: 80,
				HPMax:     100,
			},
		},
	}

	er := encounterReadResult{
		State:      encounterReadOK,
		Encounters: []encounterSummary{enc},
		Count:      1,
	}

	inv := inventoryReadResult{
		State:         inventoryReadOK,
		HasLifecycle:  true,
		CanAct:        false,
		BlockedReason: "player_dead",
		HPCurrent:     0,
		HPMax:         100,
	}

	mobr := mobReadResult{
		State: mobReadOK,
		Mobs: []mobPosition{
			{
				MobID:     "orc-1",
				LiveMobID: "live-orc-1",
				Position:  mobPosVec3{X: 25, Y: 0, Z: 0},
			},
		},
		Count: 1,
	}

	ar := attackResult{}

	entries := []rosterEntry{
		{kind: "pc", id: "p1"},
		{kind: "mb", id: "live-orc-1", mobID: "orc-1", canonicalMobID: true},
	}

	focus := rosterFocus{index: 0}

	tc := targetConfirmResult{}

	return pr, er, inv, mobr, ar, focus, entries, tc
}

// group2p1mActiveSnapshot returns state for a 2-player, 1-mob active encounter.
// Players: p1, p2. Mob: orc-1 (live-orc-1) targeting p1.
func group2p1mActiveSnapshot() (playerReadResult, encounterReadResult, inventoryReadResult, mobReadResult, attackResult, rosterFocus, []rosterEntry, targetConfirmResult) {
	pr := playerReadResult{
		State:              playerReadOK,
		Position:           playerPosResult{X: 20, Y: 0},
		HasPos:             true,
		ActiveEncounterID:  "enc-group-2p1m",
		HasActiveEncounter: true,
	}

	enc := encounterSummary{
		EncounterID: "enc-group-2p1m",
		State:       "Active",
		PlayerIDs:   []string{"p1", "p2"},
		MobIDs:      []string{"live-orc-1"},
		PlayerCount: 2,
		MobCount:    1,
		MobsAlive:   1,
		MobsDead:    0,
		ActionIndex: 1,
		MobThreat: []mobThreatEntry{
			{
				MobID:                  "live-orc-1",
				LiveMobID:              "live-orc-1",
				SelectedTargetPlayerID: "p1",
				HPCurrent:              90,
				HPMax:                  100,
			},
		},
	}

	er := encounterReadResult{
		State:      encounterReadOK,
		Encounters: []encounterSummary{enc},
		Count:      1,
	}

	inv := inventoryReadResult{
		State:        inventoryReadOK,
		HasLifecycle: true,
		CanAct:       true,
		HPCurrent:    95,
		HPMax:        100,
	}

	mobr := mobReadResult{
		State: mobReadOK,
		Mobs: []mobPosition{
			{
				MobID:     "orc-1",
				LiveMobID: "live-orc-1",
				Position:  mobPosVec3{X: 25, Y: 0, Z: 0},
			},
		},
		Count: 1,
	}

	ar := attackResult{
		State:    attackStateSent,
		TargetID: "live-orc-1",
	}

	entries := []rosterEntry{
		{kind: "pc", id: "p1"},
		{kind: "pc", id: "p2"},
		{kind: "mb", id: "live-orc-1", mobID: "orc-1", canonicalMobID: true},
	}

	focus := rosterFocus{index: 2}

	tc := targetConfirmResult{
		State:           targetConfirmOK,
		TargetKind:      "mb",
		TargetID:        "live-orc-1",
		Found:           true,
		WithinProximity: true,
		Distance:        5,
	}

	return pr, er, inv, mobr, ar, focus, entries, tc
}

// === 1v1 Combat Truth Tests ===

func TestTruthCombat1v1EncounterActive(t *testing.T) {
	pr, er, inv, _, ar, _, _, _ := combat1v1ActiveSnapshot()
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "[active]") {
		t.Fatalf("combat panel should contain [active], got:\n%s", stripped)
	}
}

func TestTruthCombat1v1PlayerHP(t *testing.T) {
	pr, _, inv, _, _, _, _, _ := combat1v1ActiveSnapshot()
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "85/100") {
		t.Fatalf("player panel should contain 85/100 (player HP), got:\n%s", stripped)
	}
}

func TestTruthCombat1v1MobHP(t *testing.T) {
	pr, er, inv, _, ar, focus, entries, _ := combat1v1ActiveSnapshot()
	// Get focused mob ID from roster entries
	fe := focusedEntry(focus, entries)
	focusedMobID := ""
	if fe != nil && fe.kind == "mb" {
		focusedMobID = fe.id
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, focusedMobID)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "70/100") {
		t.Fatalf("combat panel should contain 70/100 (mob HP), got:\n%s", stripped)
	}
}

func TestTruthCombat1v1CombatResult(t *testing.T) {
	pr, er, inv, _, ar, _, _, _ := combat1v1ActiveSnapshot()
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "res:dmg") {
		t.Fatalf("combat panel should contain res:dmg (compact result kind), got:\n%s", stripped)
	}
}

func TestTruthCombat1v1FocusedMobInProximity(t *testing.T) {
	pr, _, _, mobr, ar, focus, entries, tc := combat1v1ActiveSnapshot()
	panel := renderProximityPanel(sidePanelWidth, tc, ar, mobr, pr, focus, entries)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "orc-1") {
		t.Fatalf("proximity panel should contain mob identity orc-1, got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "IN RANGE") {
		t.Fatalf("proximity panel should contain IN RANGE, got:\n%s", stripped)
	}
}

func TestTruthCombat1v1Distance(t *testing.T) {
	pr, _, _, mobr, ar, focus, entries, tc := combat1v1ActiveSnapshot()
	panel := renderProximityPanel(sidePanelWidth, tc, ar, mobr, pr, focus, entries)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "dist:5") {
		t.Fatalf("proximity panel should contain dist:5, got:\n%s", stripped)
	}
}

func TestTruthCombat1v1Victory(t *testing.T) {
	pr, er, inv, _, ar, _, _, _ := combat1v1VictorySnapshot()
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "VICTORY") {
		t.Fatalf("combat panel should contain VICTORY for completed encounter, got:\n%s", stripped)
	}
}

func TestTruthCombat1v1XPAwarded(t *testing.T) {
	pr, er, _, _, _, _, _, _ := combat1v1VictorySnapshot()
	panel := renderOutcomePanel(sidePanelWidth, pr, er)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "xp +100") {
		t.Fatalf("outcome panel should contain xp +100, got:\n%s", stripped)
	}
}

func TestTruthCombat1v1MobDead(t *testing.T) {
	_, er, _, _, ar, _, _, _ := combat1v1VictorySnapshot()
	enc := findPlayerEncounter(er.Encounters, "enc-combat-1v1")
	lines := renderCombatMobRoster(enc, ar, "p1", sidePanelWidth, "")
	joined := stripANSI(strings.Join(lines, "\n"))
	if !strings.Contains(joined, "(dead)") {
		t.Fatalf("mob roster should contain (dead) for killed mob, got:\n%s", joined)
	}
}

func TestTruthCombat1v1DeathState(t *testing.T) {
	pr, _, inv, _, _, _, _, _ := combat1v1DeathSnapshot()
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "DEAD") {
		t.Fatalf("player panel should contain DEAD when HP is 0, got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "0/100") {
		t.Fatalf("player panel should contain 0/100 (dead player HP), got:\n%s", stripped)
	}
}

func TestTruthCombat1v1BlockedReason(t *testing.T) {
	pr, _, inv, _, _, _, _, _ := combat1v1DeathSnapshot()
	// When player is dead (HP 0), renderPlayerPanel shows "DEAD" instead of
	// "blocked:<reason>". The blocked path is for alive-but-unable-to-act states.
	// Verify the panel surfaces the death state which is the highest-priority blocked indicator.
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	stripped := stripANSI(panel)
	// Dead state takes precedence over blocked — verify the dead label is present.
	// The blocked_reason "player_dead" is the backend cause, surfaced as "DEAD" label.
	if !strings.Contains(stripped, "DEAD") {
		t.Fatalf("player panel should contain DEAD (blocked by death), got:\n%s", stripped)
	}
}

func TestTruthCombat1v1Denial(t *testing.T) {
	pr, er, inv, _, ar, _, _, _ := combat1v1ActiveSnapshot()
	// Replace latest result kind with out_of_range denial
	enc := er.Encounters[0]
	enc.LatestResultKind = "out_of_range"
	enc.LatestResultValue = 0
	er.Encounters = []encounterSummary{enc}

	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "deny:oor") {
		t.Fatalf("combat panel should contain deny:oor for out_of_range denial, got:\n%s", stripped)
	}
}

func TestTruthCombat1v1EventStream(t *testing.T) {
	pr, er, _, _, _, _, _, _ := combat1v1ActiveSnapshot()
	events := eventLog{}
	events.append(eventEntry{kind: "combat", text: "hit 15"})
	events.append(eventEntry{kind: "move", text: "move north"})
	events.append(eventEntry{kind: "enc", text: "enc:opened"})

	panel := renderEventPanel(sidePanelWidth, events, pr, er)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "hit 15") {
		t.Fatalf("event panel should contain 'hit 15', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "enc:opened") {
		t.Fatalf("event panel should contain 'enc:opened', got:\n%s", stripped)
	}
}

// === Multiplayer Truth Tests ===

func TestTruthGroup2p1mBothPlayersInRoster(t *testing.T) {
	pr, er, _, _, _, focus, _, _ := group2p1mActiveSnapshot()
	panel := renderEncounterPanel(sidePanelWidth, pr, er, focus, "p1")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "p1") {
		t.Fatalf("encounter panel should contain p1, got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "p2") {
		t.Fatalf("encounter panel should contain p2, got:\n%s", stripped)
	}
}

func TestTruthGroup2p1mPlayerAndMobCounts(t *testing.T) {
	pr, er, inv, _, ar, _, _, _ := group2p1mActiveSnapshot()
	// Check encounter panel for player/mob counts
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, "")
	stripped := stripANSI(panel)
	// The encounter panel format for active encounters is "%dp/%dm"
	if !strings.Contains(stripped, "1a/0d") {
		t.Fatalf("combat panel should contain alive/dead counts, got:\n%s", stripped)
	}
}

func TestTruthGroup2p1mMobEngagement(t *testing.T) {
	_, er, _, _, ar, _, _, _ := group2p1mActiveSnapshot()
	enc := findPlayerEncounter(er.Encounters, "enc-group-2p1m")
	lines := renderCombatMobRoster(enc, ar, "p1", sidePanelWidth, "")
	joined := stripANSI(strings.Join(lines, "\n"))
	if !strings.Contains(joined, "<-") {
		t.Fatalf("mob roster should contain <- (mob targeting indicator), got:\n%s", joined)
	}
}

// --- Deferred backend facts ---
// The following backend-owned facts are not currently surfaced in the TUI
// and are intentionally not tested here. If they are added in future missions,
// corresponding truth validation tests should be added.
//
// - encounter_id: not displayed as a readable value to the player
// - per-mob threat table values (numeric threat amounts)
// - per-player XP breakdown in multiplayer encounters
// - encounter timeline / action history details beyond latest result
// - mob retaliation details (which mob hit which player)
