package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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

// === Encounter Identity Tests ===

func TestTruthEncounterIdentityVisible(t *testing.T) {
	pr, er, inv, _, ar, _, _, _ := combat1v1ActiveSnapshot()
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "enc:") {
		t.Fatalf("combat panel should show encounter identity prefix, got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "enc:enc-co..") {
		t.Fatalf("combat panel should show truncated encounter ID, got:\n%s", stripped)
	}
}

func TestTruthEncounterIdentityStableOnVictory(t *testing.T) {
	pr, er, inv, _, ar, _, _, _ := combat1v1VictorySnapshot()
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "enc:") {
		t.Fatalf("combat panel should show encounter identity after completion, got:\n%s", stripped)
	}
}

// === Timeline Ordering Tests ===

func TestTruthTimelineOrdering(t *testing.T) {
	pr, er, _, _, _, _, _, _ := combat1v1ActiveSnapshot()
	events := eventLog{}
	events.append(eventEntry{kind: "enc", text: "enc:opened"})
	events.append(eventEntry{kind: "combat", text: "orc-1>p1 hit 15"})
	events.append(eventEntry{kind: "combat", text: "mob defeated"})
	events.append(eventEntry{kind: "enc", text: "VICTORY"})

	panel := renderEventPanel(sidePanelWidth, events, pr, er)
	stripped := stripANSI(panel)

	for _, want := range []string{"enc:opened", "hit 15", "mob defeated", "VICTORY"} {
		if !strings.Contains(stripped, want) {
			t.Fatalf("event panel should contain %q, got:\n%s", want, stripped)
		}
	}

	openIdx := strings.Index(stripped, "enc:opened")
	hitIdx := strings.Index(stripped, "hit 15")
	victoryIdx := strings.Index(stripped, "VICTORY")
	if openIdx >= hitIdx || hitIdx >= victoryIdx {
		t.Fatalf("events should be in chronological order, got positions: opened=%d hit=%d victory=%d", openIdx, hitIdx, victoryIdx)
	}
}

// === Retaliation Visibility Tests ===

func TestTruthRetaliationEventVisible(t *testing.T) {
	text := eventTextForCombatResult("damage_applied", 12, "orc-1", "p1")
	if !strings.Contains(text, "orc-1") {
		t.Fatalf("retaliation event should contain attacker, got: %s", text)
	}
	if !strings.Contains(text, "p1") {
		t.Fatalf("retaliation event should contain target, got: %s", text)
	}
	if !strings.Contains(text, "hit") {
		t.Fatalf("retaliation event should contain hit label, got: %s", text)
	}
	if !strings.Contains(text, "12") {
		t.Fatalf("retaliation event should contain damage value, got: %s", text)
	}
}

func TestTruthRetaliationFallbackWithoutActor(t *testing.T) {
	text := eventTextForCombatResult("damage_applied", 12, "", "")
	if text != "hit 12" {
		t.Fatalf("without actor/target should fall back to simple format, got: %s", text)
	}
}

func TestTruthRetaliationInEventPanel(t *testing.T) {
	pr, er, _, _, _, _, _, _ := combat1v1ActiveSnapshot()
	events := eventLog{}
	events.append(eventEntry{kind: "combat", text: "orc-1>p1 hit 12"})

	panel := renderEventPanel(sidePanelWidth, events, pr, er)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "orc-1") {
		t.Fatalf("event panel should show mob attacker identity, got:\n%s", stripped)
	}
}

func TestTruthMobEngagementInRoster(t *testing.T) {
	_, er, _, _, ar, _, _, _ := group2p1mActiveSnapshot()
	enc := findPlayerEncounter(er.Encounters, "enc-group-2p1m")
	lines := renderCombatMobRoster(enc, ar, "p1", sidePanelWidth, "")
	joined := stripANSI(strings.Join(lines, "\n"))
	if !strings.Contains(joined, "<-") {
		t.Fatalf("mob roster should show targeting indicator, got:\n%s", joined)
	}
}

func TestTruthMultiplayerNoGhostParticipants(t *testing.T) {
	pr, er, _, _, _, focus, _, _ := group2p1mActiveSnapshot()
	panel := renderEncounterPanel(sidePanelWidth, pr, er, focus, "p1")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "p1") || !strings.Contains(stripped, "p2") {
		t.Fatalf("encounter panel should show both real players, got:\n%s", stripped)
	}
	if strings.Contains(stripped, "p3") {
		t.Fatalf("encounter panel should not contain ghost participant p3, got:\n%s", stripped)
	}
}

// --- Spatial helpers ---

// spatialCanvas returns a blank canvas and simple bounds for spatial overlay tests.
// Bounds cover world coordinates (0,0) to (50,50) — large enough for snapshot positions.
func spatialCanvas(w, h int) (string, mapBounds) {
	bounds := mapBounds{MinX: 0, MaxX: 50, MinZ: 0, MaxZ: 50, SpanX: 50, SpanZ: 50}
	var sb strings.Builder
	for r := range h {
		if r > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(strings.Repeat(" ", w))
	}
	return sb.String(), bounds
}

// spatialMapFixture returns a mapReadResult with simple 4-wall room geometry.
// Used for renderMapPanel integration tests that need the adaptive viewport path.
func spatialMapFixture() mapReadResult {
	lines := []mapLine{
		{From: mapVec3{X: 0, Y: 0, Z: 0}, To: mapVec3{X: 50, Y: 0, Z: 0}},
		{From: mapVec3{X: 50, Y: 0, Z: 0}, To: mapVec3{X: 50, Y: 0, Z: 50}},
		{From: mapVec3{X: 50, Y: 0, Z: 50}, To: mapVec3{X: 0, Y: 0, Z: 50}},
		{From: mapVec3{X: 0, Y: 0, Z: 50}, To: mapVec3{X: 0, Y: 0, Z: 0}},
	}
	bounds := computeBounds(lines)
	width, height := 30, 15
	ascii, _ := projectAndRasterize(lines, width, height)
	return mapReadResult{
		State:     mapReadOK,
		MapText:   ascii,
		MapWidth:  width,
		MapHeight: height,
		Bounds:    bounds,
		Lines:     lines,
	}
}

// === Spatial Truth Tests ===

func TestTruthSpatialPlayerMarkerOnMap(t *testing.T) {
	pr, _, _, _, _, _, _, _ := combat1v1ActiveSnapshot()
	canvas, bounds := spatialCanvas(20, 10)
	result := overlayPlayer(canvas, pr.Position, bounds, 20, 10, false)
	if !strings.Contains(result, "@") {
		t.Fatalf("map should show player marker '@' at backend position, got:\n%s", result)
	}
}

func TestTruthSpatialPlayerCombatGlyphOnMap(t *testing.T) {
	pr, _, _, _, _, _, _, _ := combat1v1ActiveSnapshot()
	canvas, bounds := spatialCanvas(20, 10)
	result := overlayPlayer(canvas, pr.Position, bounds, 20, 10, true)
	if !strings.Contains(result, "*") {
		t.Fatalf("map should show combat marker '*' for player in combat, got:\n%s", result)
	}
	if strings.Contains(result, "@") {
		t.Fatalf("map should not show idle marker '@' when player is in combat, got:\n%s", result)
	}
}

func TestTruthSpatialMobMarkerOnMap(t *testing.T) {
	_, _, _, mobr, _, _, _, _ := combat1v1ActiveSnapshot()
	canvas, bounds := spatialCanvas(20, 10)
	result := overlayMobs(canvas, mobr.Mobs, bounds, 20, 10)
	if !strings.Contains(result, "m") {
		t.Fatalf("map should show mob marker 'm' at backend position, got:\n%s", result)
	}
}

func TestTruthSpatialFocusedMobTargetMarker(t *testing.T) {
	_, _, _, mobr, _, focus, entries, _ := combat1v1ActiveSnapshot()
	canvas, bounds := spatialCanvas(20, 10)
	canvas = overlayMobs(canvas, mobr.Mobs, bounds, 20, 10)
	fe := focusedEntry(focus, entries)
	if fe == nil || fe.kind != "mb" {
		t.Fatal("expected focused entry to be a mob")
	}
	canvas = overlayFocusedMob(canvas, mobr.Mobs, fe.id, bounds, 20, 10)
	if !strings.Contains(canvas, "M") {
		t.Fatalf("map should show target marker 'M' for focused mob, got:\n%s", canvas)
	}
}

func TestTruthSpatialAttackTargetGlyphOnMap(t *testing.T) {
	_, _, _, mobr, ar, _, _, _ := combat1v1ActiveSnapshot()
	canvas, bounds := spatialCanvas(20, 10)
	canvas = overlayMobs(canvas, mobr.Mobs, bounds, 20, 10)
	canvas = overlayAttackTarget(canvas, mobr.Mobs, ar.TargetID, bounds, 20, 10, 'X')
	if !strings.Contains(canvas, "X") {
		t.Fatalf("map should show attack target glyph 'X', got:\n%s", canvas)
	}
}

func TestTruthSpatialMultipleMobsDistinct(t *testing.T) {
	mobs := []mobPosition{
		{MobID: "orc-1", LiveMobID: "live-orc-1", Position: mobPosVec3{X: 10, Y: 10, Z: 0}},
		{MobID: "orc-2", LiveMobID: "live-orc-2", Position: mobPosVec3{X: 40, Y: 40, Z: 0}},
	}
	canvas, bounds := spatialCanvas(20, 10)
	result := overlayMobs(canvas, mobs, bounds, 20, 10)
	if strings.Count(result, "m") < 2 {
		t.Fatalf("map should show 2 distinct mob markers 'm', got %d in:\n%s", strings.Count(result, "m"), result)
	}
}

func TestTruthSpatialMultipleMobsSameCellShowCount(t *testing.T) {
	mobs := []mobPosition{
		{MobID: "orc-1", LiveMobID: "live-orc-1", Position: mobPosVec3{X: 25, Y: 0, Z: 0}},
		{MobID: "orc-2", LiveMobID: "live-orc-2", Position: mobPosVec3{X: 25, Y: 0, Z: 0}},
	}
	canvas, bounds := spatialCanvas(20, 10)
	result := overlayMobs(canvas, mobs, bounds, 20, 10)
	if !strings.Contains(result, "2") {
		t.Fatalf("map should show count '2' for overlapping mobs, got:\n%s", result)
	}
}

func TestTruthSpatialPlayerAndMobCoexist(t *testing.T) {
	pr, _, _, mobr, _, _, _, _ := combat1v1ActiveSnapshot()
	canvas, bounds := spatialCanvas(20, 10)
	canvas = overlayMobs(canvas, mobr.Mobs, bounds, 20, 10)
	canvas = overlayPlayer(canvas, pr.Position, bounds, 20, 10, false)
	if !strings.Contains(canvas, "@") {
		t.Fatalf("map should show player marker '@', got:\n%s", canvas)
	}
	if !strings.Contains(canvas, "m") {
		t.Fatalf("map should show mob marker 'm' alongside player, got:\n%s", canvas)
	}
}

func TestTruthSpatialOutOfRangeLabel(t *testing.T) {
	pr, _, _, mobr, ar, focus, entries, _ := combat1v1ActiveSnapshot()
	tc := targetConfirmResult{
		State:           targetConfirmOK,
		TargetKind:      "mb",
		TargetID:        "live-orc-1",
		Found:           true,
		WithinProximity: false,
		Distance:        50,
	}
	panel := renderProximityPanel(sidePanelWidth, tc, ar, mobr, pr, focus, entries)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "OOR") {
		t.Fatalf("proximity panel should show OOR for out-of-range target, got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "dist:50") {
		t.Fatalf("proximity panel should show dist:50 for out-of-range target, got:\n%s", stripped)
	}
}

func TestTruthSpatialFocusedPlayerGlyph(t *testing.T) {
	pr, _, _, _, _, _, _, _ := combat1v1ActiveSnapshot()
	canvas, bounds := spatialCanvas(20, 10)
	canvas = overlayPlayer(canvas, pr.Position, bounds, 20, 10, false)
	canvas = overlayFocusedPlayer(canvas, pr.Position, bounds, 20, 10)
	if !strings.Contains(canvas, "&") {
		t.Fatalf("map should show focused player marker '&', got:\n%s", canvas)
	}
	if strings.Contains(canvas, "@") {
		t.Fatalf("focused player '&' should replace idle '@', got:\n%s", canvas)
	}
}

func TestTruthSpatialHitImpactGlyph(t *testing.T) {
	_, _, _, mobr, ar, _, _, _ := combat1v1ActiveSnapshot()
	canvas, bounds := spatialCanvas(20, 10)
	canvas = overlayMobs(canvas, mobr.Mobs, bounds, 20, 10)
	canvas = overlayAttackTarget(canvas, mobr.Mobs, ar.TargetID, bounds, 20, 10, '!')
	if !strings.Contains(canvas, "!") {
		t.Fatalf("map should show hit impact glyph '!' for damage result, got:\n%s", canvas)
	}
}

func TestTruthSpatialMapPanelIntegration(t *testing.T) {
	pr, _, _, mobr, ar, focus, entries, _ := combat1v1ActiveSnapshot()
	mr := spatialMapFixture()
	panel := renderMapPanel(mr, mobr, pr, focus, entries, 80, 30, ar, "damage_applied")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "*") {
		t.Fatalf("map panel should show combat player marker '*', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "!") {
		t.Fatalf("map panel should show hit impact glyph '!' for attack target, got:\n%s", stripped)
	}
}

// --- Temporal playback infrastructure ---

// scenarioFrame holds one step of backend-owned state for playback validation.
type scenarioFrame struct {
	pr     playerReadResult
	er     encounterReadResult
	inv    inventoryReadResult
	mobr   mobReadResult
	ar     attackResult
	tc     targetConfirmResult
	events eventLog
}

// === Temporal Validation Tests ===

func TestTruthTemporalMovementProgression(t *testing.T) {
	canvas, bounds := spatialCanvas(20, 10)
	w, h := 20, 10

	posA := playerPosResult{X: 10, Y: 10}
	posB := playerPosResult{X: 40, Y: 40}

	frameA := overlayPlayer(canvas, posA, bounds, w, h, false)
	frameB := overlayPlayer(canvas, posB, bounds, w, h, false)

	// Both frames must have player marker
	if !strings.Contains(frameA, "@") {
		t.Fatalf("frame A should show player marker, got:\n%s", frameA)
	}
	if !strings.Contains(frameB, "@") {
		t.Fatalf("frame B should show player marker, got:\n%s", frameB)
	}
	// Player position should differ between frames
	if frameA == frameB {
		t.Fatalf("player marker should move between frames A and B")
	}
}

func TestTruthTemporalDistanceTransition(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, Position: playerPosResult{X: 20, Y: 0}, HasPos: true}
	mobr := mobReadResult{
		State: mobReadOK,
		Mobs: []mobPosition{{MobID: "orc-1", LiveMobID: "live-orc-1", Position: mobPosVec3{X: 25, Y: 0, Z: 0}}},
		Count: 1,
	}
	ar := attackResult{}
	entries := []rosterEntry{{kind: "mb", id: "live-orc-1", mobID: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}

	// Frame 1: out of range
	tc1 := targetConfirmResult{State: targetConfirmOK, TargetKind: "mb", TargetID: "live-orc-1", Found: true, WithinProximity: false, Distance: 50}
	panel1 := renderProximityPanel(sidePanelWidth, tc1, ar, mobr, pr, focus, entries)
	stripped1 := stripANSI(panel1)

	// Frame 2: in range
	tc2 := targetConfirmResult{State: targetConfirmOK, TargetKind: "mb", TargetID: "live-orc-1", Found: true, WithinProximity: true, Distance: 5}
	panel2 := renderProximityPanel(sidePanelWidth, tc2, ar, mobr, pr, focus, entries)
	stripped2 := stripANSI(panel2)

	if !strings.Contains(stripped1, "OOR") {
		t.Fatalf("frame 1 should show OOR, got:\n%s", stripped1)
	}
	if !strings.Contains(stripped2, "IN RANGE") {
		t.Fatalf("frame 2 should show IN RANGE, got:\n%s", stripped2)
	}
	if strings.Contains(stripped1, "IN RANGE") {
		t.Fatalf("frame 1 should not show IN RANGE, got:\n%s", stripped1)
	}
}

func TestTruthTemporalCombatProgression(t *testing.T) {
	frames := []scenarioFrame{
		// Frame 0: encounter just opened, full HP
		{
			pr: playerReadResult{State: playerReadOK, Position: playerPosResult{X: 20, Y: 0}, HasPos: true, ActiveEncounterID: "enc-prog", HasActiveEncounter: true},
			er: encounterReadResult{State: encounterReadOK, Encounters: []encounterSummary{{
				EncounterID: "enc-prog", State: "Active", PlayerIDs: []string{"p1"}, MobIDs: []string{"live-orc-1"},
				PlayerCount: 1, MobCount: 1, MobsAlive: 1, MobsDead: 0, ActionIndex: 0,
				MobThreat: []mobThreatEntry{{MobID: "live-orc-1", LiveMobID: "live-orc-1", HPCurrent: 100, HPMax: 100}},
			}}, Count: 1},
			inv: inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100},
			ar:  attackResult{State: attackStateSent, TargetID: "live-orc-1"},
		},
		// Frame 1: mid-combat, HP reduced
		{
			pr: playerReadResult{State: playerReadOK, Position: playerPosResult{X: 20, Y: 0}, HasPos: true, ActiveEncounterID: "enc-prog", HasActiveEncounter: true},
			er: encounterReadResult{State: encounterReadOK, Encounters: []encounterSummary{{
				EncounterID: "enc-prog", State: "Active", PlayerIDs: []string{"p1"}, MobIDs: []string{"live-orc-1"},
				PlayerCount: 1, MobCount: 1, MobsAlive: 1, MobsDead: 0, ActionIndex: 5,
				LatestResultKind: "damage_applied", LatestResultValue: 20,
				MobThreat: []mobThreatEntry{{MobID: "live-orc-1", LiveMobID: "live-orc-1", HPCurrent: 40, HPMax: 100}},
			}}, Count: 1},
			inv: inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 70, HPMax: 100},
			ar:  attackResult{State: attackStateSent, TargetID: "live-orc-1"},
		},
		// Frame 2: victory, mob dead
		{
			pr: playerReadResult{State: playerReadOK, Position: playerPosResult{X: 20, Y: 0}, HasPos: true, ActiveEncounterID: "enc-prog", HasActiveEncounter: true},
			er: encounterReadResult{State: encounterReadOK, Encounters: []encounterSummary{{
				EncounterID: "enc-prog", State: "Completed", CompletedReason: "all_mobs_dead",
				PlayerIDs: []string{"p1"}, MobIDs: []string{"live-orc-1"},
				PlayerCount: 1, MobCount: 1, MobsAlive: 0, MobsDead: 1, ActionIndex: 10,
				XPAwarded: 150, DropsGenerated: true, Drops: []string{"loot-1"},
				MobThreat: []mobThreatEntry{{MobID: "live-orc-1", LiveMobID: "live-orc-1", HPCurrent: 0, HPMax: 100}},
			}}, Count: 1},
			inv: inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 55, HPMax: 100},
			ar:  attackResult{State: attackStateSent, TargetID: "live-orc-1"},
		},
	}

	for i, f := range frames {
		combat := renderCombatPanel(sidePanelWidth, f.ar, f.pr, f.er, defaultTarget(), f.inv, "")
		stripped := stripANSI(combat)

		switch i {
		case 0:
			if !strings.Contains(stripped, "[active]") {
				t.Fatalf("frame 0: should show [active], got:\n%s", stripped)
			}
			if !strings.Contains(stripped, "100/100") {
				t.Fatalf("frame 0: should show mob HP 100/100, got:\n%s", stripped)
			}
		case 1:
			if !strings.Contains(stripped, "[active]") {
				t.Fatalf("frame 1: should show [active], got:\n%s", stripped)
			}
			if !strings.Contains(stripped, "40/100") {
				t.Fatalf("frame 1: should show mob HP 40/100, got:\n%s", stripped)
			}
		case 2:
			if !strings.Contains(stripped, "VICTORY") {
				t.Fatalf("frame 2: should show VICTORY, got:\n%s", stripped)
			}
		}
	}

	// Verify player HP progression through player panel
	player0 := stripANSI(renderPlayerPanel(sidePanelWidth, frames[0].pr, frames[0].inv, respawnResult{}))
	player1 := stripANSI(renderPlayerPanel(sidePanelWidth, frames[1].pr, frames[1].inv, respawnResult{}))
	player2 := stripANSI(renderPlayerPanel(sidePanelWidth, frames[2].pr, frames[2].inv, respawnResult{}))

	if !strings.Contains(player0, "100/100") {
		t.Fatalf("frame 0: player HP should be 100/100, got:\n%s", player0)
	}
	if !strings.Contains(player1, "70/100") {
		t.Fatalf("frame 1: player HP should be 70/100, got:\n%s", player1)
	}
	if !strings.Contains(player2, "55/100") {
		t.Fatalf("frame 2: player HP should be 55/100, got:\n%s", player2)
	}

	// Verify outcome panel on victory frame
	outcome := stripANSI(renderOutcomePanel(sidePanelWidth, frames[2].pr, frames[2].er))
	if !strings.Contains(outcome, "xp +150") {
		t.Fatalf("frame 2: outcome should show xp +150, got:\n%s", outcome)
	}
}

func TestTruthTemporalPlayerDeathProgression(t *testing.T) {
	// Frame 0: alive, low HP
	inv0 := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 10, HPMax: 100}
	pr0 := playerReadResult{State: playerReadOK, Position: playerPosResult{X: 20, Y: 0}, HasPos: true}
	panel0 := stripANSI(renderPlayerPanel(sidePanelWidth, pr0, inv0, respawnResult{}))

	// Frame 1: dead
	inv1 := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, BlockedReason: "player_dead", HPCurrent: 0, HPMax: 100}
	panel1 := stripANSI(renderPlayerPanel(sidePanelWidth, pr0, inv1, respawnResult{}))

	// Frame 2: respawn sent
	panel2 := stripANSI(renderPlayerPanel(sidePanelWidth, pr0, inv1, respawnResult{State: respawnStateSent}))

	if strings.Contains(panel0, "DEAD") {
		t.Fatalf("frame 0: should not show DEAD at 10 HP, got:\n%s", panel0)
	}
	if !strings.Contains(panel0, "10/100") {
		t.Fatalf("frame 0: should show 10/100, got:\n%s", panel0)
	}
	if !strings.Contains(panel1, "DEAD") {
		t.Fatalf("frame 1: should show DEAD, got:\n%s", panel1)
	}
	if !strings.Contains(panel1, "0/100") {
		t.Fatalf("frame 1: should show 0/100, got:\n%s", panel1)
	}
	_ = panel2 // respawn sent frame compiled and rendered
}

func TestTruthTemporalTargetingChange(t *testing.T) {
	mobs := []mobPosition{
		{MobID: "orc-1", LiveMobID: "live-orc-1", Position: mobPosVec3{X: 25, Y: 0, Z: 0}},
		{MobID: "orc-2", LiveMobID: "live-orc-2", Position: mobPosVec3{X: 30, Y: 0, Z: 0}},
	}
	canvas, bounds := spatialCanvas(20, 10)
	w, h := 20, 10

	// Frame 0: orc-1 focused
	map0 := overlayMobs(canvas, mobs, bounds, w, h)
	map0 = overlayFocusedMob(map0, mobs, "live-orc-1", bounds, w, h)

	// Frame 1: orc-2 focused
	map1 := overlayMobs(canvas, mobs, bounds, w, h)
	map1 = overlayFocusedMob(map1, mobs, "live-orc-2", bounds, w, h)

	// Both frames should have 'M' marker
	if !strings.Contains(map0, "M") {
		t.Fatalf("frame 0: should show target marker M for orc-1, got:\n%s", map0)
	}
	if !strings.Contains(map1, "M") {
		t.Fatalf("frame 1: should show target marker M for orc-2, got:\n%s", map1)
	}
	// Target should move between frames
	if map0 == map1 {
		t.Fatalf("target marker T should be at different position between frames")
	}
}

func TestTruthTemporalMultiPlayerProgression(t *testing.T) {
	frames := []scenarioFrame{
		// Frame 0: p1 alone in encounter
		{
			pr: playerReadResult{State: playerReadOK, Position: playerPosResult{X: 20, Y: 0}, HasPos: true, ActiveEncounterID: "enc-mp", HasActiveEncounter: true},
			er: encounterReadResult{State: encounterReadOK, Encounters: []encounterSummary{{
				EncounterID: "enc-mp", State: "Active", PlayerIDs: []string{"p1"}, MobIDs: []string{"live-orc-1"},
				PlayerCount: 1, MobCount: 1, MobsAlive: 1, MobsDead: 0, ActionIndex: 1,
				MobThreat: []mobThreatEntry{{MobID: "live-orc-1", LiveMobID: "live-orc-1", SelectedTargetPlayerID: "p1", HPCurrent: 100, HPMax: 100}},
			}}, Count: 1},
			inv: inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 90, HPMax: 100},
			ar:  attackResult{State: attackStateSent, TargetID: "live-orc-1"},
		},
		// Frame 1: p2 joins
		{
			pr: playerReadResult{State: playerReadOK, Position: playerPosResult{X: 20, Y: 0}, HasPos: true, ActiveEncounterID: "enc-mp", HasActiveEncounter: true},
			er: encounterReadResult{State: encounterReadOK, Encounters: []encounterSummary{{
				EncounterID: "enc-mp", State: "Active", PlayerIDs: []string{"p1", "p2"}, MobIDs: []string{"live-orc-1"},
				PlayerCount: 2, MobCount: 1, MobsAlive: 1, MobsDead: 0, ActionIndex: 3,
				MobThreat: []mobThreatEntry{{MobID: "live-orc-1", LiveMobID: "live-orc-1", SelectedTargetPlayerID: "p1", HPCurrent: 60, HPMax: 100}},
			}}, Count: 1},
			inv: inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 80, HPMax: 100},
			ar:  attackResult{State: attackStateSent, TargetID: "live-orc-1"},
		},
		// Frame 2: mob dies, victory
		{
			pr: playerReadResult{State: playerReadOK, Position: playerPosResult{X: 20, Y: 0}, HasPos: true, ActiveEncounterID: "enc-mp", HasActiveEncounter: true},
			er: encounterReadResult{State: encounterReadOK, Encounters: []encounterSummary{{
				EncounterID: "enc-mp", State: "Completed", CompletedReason: "all_mobs_dead",
				PlayerIDs: []string{"p1", "p2"}, MobIDs: []string{"live-orc-1"},
				PlayerCount: 2, MobCount: 1, MobsAlive: 0, MobsDead: 1, ActionIndex: 8,
				XPAwarded: 200,
				MobThreat: []mobThreatEntry{{MobID: "live-orc-1", LiveMobID: "live-orc-1", HPCurrent: 0, HPMax: 100}},
			}}, Count: 1},
			inv: inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 75, HPMax: 100},
			ar:  attackResult{State: attackStateSent, TargetID: "live-orc-1"},
		},
	}

	// Frame 0: only p1 in encounter panel
	focus0 := rosterFocus{index: 0}
	enc0 := stripANSI(renderEncounterPanel(sidePanelWidth, frames[0].pr, frames[0].er, focus0, "p1"))
	if !strings.Contains(enc0, "p1") {
		t.Fatalf("frame 0: encounter panel should show p1, got:\n%s", enc0)
	}
	if strings.Contains(enc0, "p2") {
		t.Fatalf("frame 0: encounter panel should not show p2 yet, got:\n%s", enc0)
	}

	// Frame 1: p1 and p2 both visible
	enc1 := stripANSI(renderEncounterPanel(sidePanelWidth, frames[1].pr, frames[1].er, focus0, "p1"))
	if !strings.Contains(enc1, "p1") || !strings.Contains(enc1, "p2") {
		t.Fatalf("frame 1: encounter panel should show both p1 and p2, got:\n%s", enc1)
	}

	// Frame 1: mob HP decreased
	combat1 := stripANSI(renderCombatPanel(sidePanelWidth, frames[1].ar, frames[1].pr, frames[1].er, defaultTarget(), frames[1].inv, ""))
	if !strings.Contains(combat1, "60/100") {
		t.Fatalf("frame 1: should show mob HP 60/100, got:\n%s", combat1)
	}

	// Frame 2: victory
	combat2 := stripANSI(renderCombatPanel(sidePanelWidth, frames[2].ar, frames[2].pr, frames[2].er, defaultTarget(), frames[2].inv, ""))
	if !strings.Contains(combat2, "VICTORY") {
		t.Fatalf("frame 2: should show VICTORY, got:\n%s", combat2)
	}

	// Frame 2: mob dead in roster
	enc2 := findPlayerEncounter(frames[2].er.Encounters, "enc-mp")
	roster2 := renderCombatMobRoster(enc2, frames[2].ar, "p1", sidePanelWidth, "")
	joined2 := stripANSI(strings.Join(roster2, "\n"))
	if !strings.Contains(joined2, "(dead)") {
		t.Fatalf("frame 2: mob roster should show (dead), got:\n%s", joined2)
	}
}

func TestTruthTemporalMobRetarget(t *testing.T) {
	// Frame 0: mob targeting p1
	enc0 := encounterSummary{
		EncounterID: "enc-retarget", State: "Active", PlayerIDs: []string{"p1", "p2"}, MobIDs: []string{"live-orc-1"},
		PlayerCount: 2, MobCount: 1, MobsAlive: 1, MobsDead: 0,
		MobThreat: []mobThreatEntry{{MobID: "live-orc-1", LiveMobID: "live-orc-1", SelectedTargetPlayerID: "p1", HPCurrent: 80, HPMax: 100}},
	}
	ar := attackResult{State: attackStateSent, TargetID: "live-orc-1"}
	roster0 := renderCombatMobRoster(&enc0, ar, "p1", sidePanelWidth, "")
	joined0 := stripANSI(strings.Join(roster0, "\n"))

	// Frame 1: mob retargets to p2
	enc1 := enc0
	enc1.MobThreat = []mobThreatEntry{{MobID: "live-orc-1", LiveMobID: "live-orc-1", SelectedTargetPlayerID: "p2", HPCurrent: 60, HPMax: 100}}
	roster1 := renderCombatMobRoster(&enc1, ar, "p1", sidePanelWidth, "")
	joined1 := stripANSI(strings.Join(roster1, "\n"))

	// Both frames should show mob identity
	if !strings.Contains(joined0, "orc-1") {
		t.Fatalf("frame 0: roster should show orc-1, got:\n%s", joined0)
	}
	if !strings.Contains(joined1, "orc-1") {
		t.Fatalf("frame 1: roster should show orc-1, got:\n%s", joined1)
	}
	// Targeting indicator should change
	if joined0 == joined1 {
		t.Fatalf("mob roster should differ when target changes from p1 to p2")
	}
}

// --- Deferred backend facts ---
// The following backend-owned facts are not currently surfaced in the TUI
// and are intentionally not tested here. If they are added in future missions,
// corresponding truth validation tests should be added.
//
// - per-mob threat table values (numeric threat amounts)
// - per-player XP breakdown in multiplayer encounters
// - encounter timeline / action history details beyond latest result
// - other-player position overlay in multiplayer map viewport
// - mob name labels on map viewport
// - respawn confirmation rendering (respawnStateOK transition)

// --- E2E trace ingestion types ---

type traceFile struct {
	ScenarioName string       `json:"scenario_name"`
	Frames       []traceFrame `json:"frames"`
}

type traceFrame struct {
	StepIndex   int             `json:"step_index"`
	StepLabel   string          `json:"step_label"`
	Players     []tracePlayer   `json:"players"`
	Mobs        []traceMob      `json:"mobs"`
	Encounter   *traceEncounter `json:"encounter,omitempty"`
	Events      []traceEvent    `json:"events,omitempty"`
	ActiveCasts []traceCast     `json:"active_casts,omitempty"`
}

type tracePlayer struct {
	PlayerID      string  `json:"player_id"`
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
	Z             float64 `json:"z"`
	HPCurrent     int     `json:"hp_current"`
	HPMax         int     `json:"hp_max"`
	CanAct        bool    `json:"can_act"`
	BlockedReason string  `json:"blocked_reason"`
	XPCurrent     int     `json:"xp_current"`
}

type traceEncounter struct {
	EncounterID     string            `json:"encounter_id"`
	State           string            `json:"state"`
	CompletedReason string            `json:"completed_reason"`
	PlayerIDs       []string          `json:"player_ids"`
	MobIDs          []string          `json:"mob_ids"`
	MobsAlive       int               `json:"mobs_alive"`
	MobsDead        int               `json:"mobs_dead"`
	ActionIndex     uint64            `json:"action_index"`
	XPAwarded       int               `json:"xp_awarded"`
	MobThreat       []traceMobThreat  `json:"mob_threat"`
	PlayerGCD       []tracePlayerGCD  `json:"player_gcd,omitempty"`
	PlayerMana      []tracePlayerMana `json:"player_mana,omitempty"`
	MobCasts        []traceMobCast    `json:"mob_casts,omitempty"`
	MobLockouts     []traceMobLockout `json:"mob_lockouts,omitempty"`
}

type traceMobCast struct {
	MobID       string  `json:"mob_id"`
	AbilityID   string  `json:"ability_id"`
	ElapsedMs   int64   `json:"elapsed_ms"`
	DurationMs  int64   `json:"duration_ms"`
	ProgressPct float64 `json:"progress_pct"`
}

type traceMobLockout struct {
	MobID       string `json:"mob_id"`
	RemainingMs int64  `json:"remaining_ms"`
}

type tracePlayerMana struct {
	PlayerID    string `json:"player_id"`
	ManaCurrent int    `json:"mana_current"`
	ManaMax     int    `json:"mana_max"`
}

type tracePlayerGCD struct {
	PlayerID          string `json:"player_id"`
	QueuedAbilityID   string `json:"queued_ability_id"`
	QueuedTargetMobID string `json:"queued_target_mob_id"`
	GCDRemainingMs    int64  `json:"gcd_remaining_ms"`
}

type traceMobThreat struct {
	MobID                  string `json:"mob_id"`
	SelectedTargetPlayerID string `json:"selected_target_player_id"`
	HPCurrent              int    `json:"hp_current"`
	HPMax                  int    `json:"hp_max"`
}

type traceMob struct {
	MobID     string  `json:"mob_id"`
	MobName   string  `json:"mob_name"`
	State     string  `json:"state"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Z         float64 `json:"z"`
	HPCurrent int     `json:"hp_current"`
	HPMax     int     `json:"hp_max"`
}

type traceEvent struct {
	Kind          string `json:"kind"`
	Actor         string `json:"actor"`
	Target        string `json:"target"`
	Damage        int    `json:"damage,omitempty"`
	Heal          int    `json:"heal,omitempty"`
	Amount        int    `json:"amount,omitempty"`
	ActionIndex   uint64 `json:"action_index,omitempty"`
	BlockedReason string `json:"blocked_reason,omitempty"`
	EncounterID   string `json:"encounter_id,omitempty"`
	AbilityID     string `json:"ability_id,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

type traceCast struct {
	PlayerID       string  `json:"player_id"`
	AbilityID      string  `json:"ability_id"`
	TargetMobID    string  `json:"target_mob_id"`
	CastDurationMs int64   `json:"cast_duration_ms"`
	ElapsedMs      int64   `json:"elapsed_ms"`
	ProgressPct    float64 `json:"progress_pct"`
}

// --- Trace loader + adapter ---

func loadScenarioTrace(t *testing.T, path string) traceFile {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read trace file %s: %v", path, err)
	}
	var tf traceFile
	if err := json.Unmarshal(data, &tf); err != nil {
		t.Fatalf("failed to decode trace file %s: %v", path, err)
	}
	return tf
}

func traceFrameToScenarioFrame(tf traceFrame, playerID string) scenarioFrame {
	// Find the player in the frame
	var player *tracePlayer
	for i := range tf.Players {
		if tf.Players[i].PlayerID == playerID {
			player = &tf.Players[i]
			break
		}
	}

	// Build playerReadResult
	pr := playerReadResult{State: playerReadOK}
	if player != nil {
		pr.Position = playerPosResult{X: player.X, Y: player.Y}
		pr.HasPos = true
	}

	// Build encounterReadResult
	er := encounterReadResult{State: encounterReadOK}
	ar := attackResult{}
	if tf.Encounter != nil {
		enc := tf.Encounter
		pr.ActiveEncounterID = enc.EncounterID
		pr.HasActiveEncounter = true

		// Build mob threat from encounter mob_threat; if absent, synthesize
		// from trace mob data so dead mobs have HP 0 entries for rendering.
		var threat []mobThreatEntry
		if len(enc.MobThreat) > 0 {
			for _, mt := range enc.MobThreat {
				threat = append(threat, mobThreatEntry{
					MobID:                  mt.MobID,
					LiveMobID:              mt.MobID,
					SelectedTargetPlayerID: mt.SelectedTargetPlayerID,
					HPCurrent:              mt.HPCurrent,
					HPMax:                  mt.HPMax,
				})
			}
		} else {
			// Synthesize from trace mobs for encounter mob IDs
			for _, mid := range enc.MobIDs {
				for _, m := range tf.Mobs {
					if m.MobID == mid {
						threat = append(threat, mobThreatEntry{
							MobID:     m.MobID,
							LiveMobID: m.MobID,
							HPCurrent: m.HPCurrent,
							HPMax:     m.HPMax,
						})
						break
					}
				}
			}
		}

		es := encounterSummary{
			EncounterID:     enc.EncounterID,
			State:           enc.State,
			CompletedReason: enc.CompletedReason,
			PlayerIDs:       enc.PlayerIDs,
			MobIDs:          enc.MobIDs,
			PlayerCount:     len(enc.PlayerIDs),
			MobCount:        len(enc.MobIDs),
			MobsAlive:       enc.MobsAlive,
			MobsDead:        enc.MobsDead,
			ActionIndex:     enc.ActionIndex,
			XPAwarded:       enc.XPAwarded,
			MobThreat:       threat,
		}

		// Map active casts from trace frame to encounter summary
		if len(tf.ActiveCasts) > 0 {
			var casts []activeCastEntry
			for _, ac := range tf.ActiveCasts {
				casts = append(casts, activeCastEntry{
					PlayerID:       ac.PlayerID,
					AbilityID:      ac.AbilityID,
					TargetMobID:    ac.TargetMobID,
					CastDurationMs: ac.CastDurationMs,
					ElapsedMs:      ac.ElapsedMs,
					ProgressPct:    ac.ProgressPct,
				})
			}
			es.ActiveCasts = casts
		}

		// Map player GCD queue data and GCD remaining from trace encounter
		for _, pg := range enc.PlayerGCD {
			if pg.QueuedAbilityID != "" {
				es.QueuedAbilityID = pg.QueuedAbilityID
				es.QueuedTargetID = pg.QueuedTargetMobID
			}
			if pg.GCDRemainingMs > 0 && es.GCDRemainingMs == 0 {
				es.GCDRemainingMs = pg.GCDRemainingMs
			}
		}

		// Map player mana from trace encounter
		if len(enc.PlayerMana) > 0 {
			var mana []playerManaEntry
			for _, pm := range enc.PlayerMana {
				mana = append(mana, playerManaEntry{
					PlayerID:    pm.PlayerID,
					ManaCurrent: pm.ManaCurrent,
					ManaMax:     pm.ManaMax,
				})
			}
			es.PlayerMana = mana
		}

		// Map mob casts from trace encounter
		if len(enc.MobCasts) > 0 {
			var casts []mobCastEntry
			for _, mc := range enc.MobCasts {
				casts = append(casts, mobCastEntry{
					MobID:       mc.MobID,
					AbilityID:   mc.AbilityID,
					ElapsedMs:   mc.ElapsedMs,
					DurationMs:  mc.DurationMs,
					ProgressPct: mc.ProgressPct,
				})
			}
			es.MobCasts = casts
		}

		// Map mob lockouts from trace encounter
		if len(enc.MobLockouts) > 0 {
			var lockouts []mobLockoutEntry
			for _, ml := range enc.MobLockouts {
				lockouts = append(lockouts, mobLockoutEntry{
					MobID:       ml.MobID,
					RemainingMs: ml.RemainingMs,
				})
			}
			es.MobLockouts = lockouts
		}

		er.Encounters = []encounterSummary{es}
		er.Count = 1

		// If there are mobs in the encounter, set attack target to first mob
		if len(enc.MobIDs) > 0 {
			ar = attackResult{State: attackStateSent, TargetID: enc.MobIDs[0]}
		}
	}

	// Build inventoryReadResult from player lifecycle
	inv := inventoryReadResult{State: inventoryReadOK}
	if player != nil {
		inv.HasLifecycle = true
		inv.CanAct = player.CanAct
		inv.BlockedReason = player.BlockedReason
		inv.HPCurrent = player.HPCurrent
		inv.HPMax = player.HPMax
	}

	// Build mobReadResult
	mobs := make([]mobPosition, 0, len(tf.Mobs))
	for _, m := range tf.Mobs {
		mobs = append(mobs, mobPosition{
			MobID:     m.MobID,
			LiveMobID: m.MobID,
			Position:  mobPosVec3{X: m.X, Y: m.Y, Z: m.Z},
		})
	}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: len(mobs)}

	// Build targetConfirmResult (for first mob in encounter, if any)
	tc := targetConfirmResult{}
	if tf.Encounter != nil && len(tf.Encounter.MobIDs) > 0 {
		tc = targetConfirmResult{
			State:           targetConfirmOK,
			TargetKind:      "mb",
			TargetID:        tf.Encounter.MobIDs[0],
			Found:           true,
			WithinProximity: true,
			Distance:        5,
		}
	}

	var events eventLog
	for _, te := range tf.Events {
		text := traceEventToText(te, playerID)
		if text != "" {
			events.append(eventEntry{kind: traceEventKind(te), text: text})
		}
	}

	return scenarioFrame{pr: pr, er: er, inv: inv, mobr: mobr, ar: ar, tc: tc, events: events}
}

func traceEventToText(te traceEvent, playerID string) string {
	switch te.Kind {
	case "player_hit":
		return eventTextForCombatResult("damage_applied", te.Damage, te.Actor, te.Target)
	case "mob_hit":
		return eventTextForCombatResult("damage_applied", te.Damage, te.Actor, te.Target)
	case "damage_applied":
		return eventTextForCombatResult("damage_applied", te.Damage, te.Actor, te.Target)
	case "heal_applied":
		return eventTextForCombatResult("heal_applied", te.Heal, te.Actor, te.Target)
	case "mob_death":
		return "mob defeated"
	case "player_death":
		return "DEFEAT"
	case "cooldown_blocked":
		return eventTextForResult("cooldown_blocked", 0)
	case "cast_start":
		return eventTextForResult("cast_start", 0)
	case "cast_complete":
		return eventTextForResult("cast_complete", 0)
	case "cast_denied":
		return eventTextForResult("cast_denied", 0)
	case "cast_cancelled":
		return eventTextForResult("cast_cancelled", 0)
	case "interrupt_applied":
		return eventTextForResult("interrupt_applied", 0)
	case "interrupt_denied":
		return eventTextForResult("interrupt_denied", 0)
	case "interrupt_lockout_started":
		return eventTextForResult("interrupt_lockout_started", 0)
	case "interrupt_lockout_blocked":
		return eventTextForResult("interrupt_lockout_blocked", 0)
	case "gcd_started":
		return eventTextForResult("gcd_started", 0)
	case "gcd_blocked":
		return eventTextForResult("gcd_blocked", 0)
	case "mob_cast_start":
		return "casting"
	case "ability_queued":
		if te.AbilityID != "" {
			return "queued " + te.AbilityID
		}
		return eventTextForResult("ability_queued", 0)
	case "ability_executed_from_queue":
		if te.AbilityID != "" {
			return "from queue " + te.AbilityID
		}
		return eventTextForResult("ability_executed_from_queue", 0)
	case "resource_blocked":
		return eventTextForResult("resource_blocked", 0)
	case "resource_spent":
		return eventTextForResult("resource_spent", te.Amount)
	case "resource_regen":
		return eventTextForResult("resource_regen", te.Amount)
	default:
		return te.Kind
	}
}

func traceEventKind(te traceEvent) string {
	switch te.Kind {
	case "player_hit", "mob_hit", "damage_applied", "heal_applied", "mob_death", "player_death":
		return "combat"
	case "cooldown_blocked", "cast_denied", "cast_cancelled", "gcd_blocked", "resource_blocked", "interrupt_denied", "interrupt_lockout_blocked":
		return "denial"
	case "cast_start", "cast_complete", "mob_cast_start":
		return "cast"
	case "gcd_started", "resource_spent", "resource_regen", "interrupt_applied", "interrupt_lockout_started":
		return "resource"
	case "ability_queued", "ability_executed_from_queue":
		return "queue"
	default:
		return "enc"
	}
}

// === E2E Validation Tests ===

func TestE2ECombat1v1Lifecycle(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_combat_1v1.json"))
	if len(trace.Frames) < 6 {
		t.Fatalf("expected at least 6 frames, got %d", len(trace.Frames))
	}

	// Frame 0: player joined, no encounter
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	player0 := stripANSI(renderPlayerPanel(sidePanelWidth, f0.pr, f0.inv, respawnResult{}))
	if !strings.Contains(player0, "100/100") {
		t.Fatalf("frame 0: player should have full HP 100/100, got:\n%s", player0)
	}

	// Frame 4: combat completed (attack_until_complete runs to completion)
	f4 := traceFrameToScenarioFrame(trace.Frames[4], "p1")
	combat4 := stripANSI(renderCombatPanel(sidePanelWidth, f4.ar, f4.pr, f4.er, defaultTarget(), f4.inv, ""))
	if !strings.Contains(combat4, "VICTORY") {
		t.Fatalf("frame 4: should show VICTORY after combat, got:\n%s", combat4)
	}

	// Frame 4: player HP should be reduced (took damage during combat)
	player4 := stripANSI(renderPlayerPanel(sidePanelWidth, f4.pr, f4.inv, respawnResult{}))
	if strings.Contains(player4, "100/100") {
		t.Fatalf("frame 4: player HP should be reduced after combat, got:\n%s", player4)
	}

	// Frame 4: mob should be dead in roster
	if f4.er.Count > 0 {
		enc := findPlayerEncounter(f4.er.Encounters, f4.pr.ActiveEncounterID)
		if enc != nil {
			roster := renderCombatMobRoster(enc, f4.ar, "p1", sidePanelWidth, "")
			joined := stripANSI(strings.Join(roster, "\n"))
			if !strings.Contains(joined, "(dead)") {
				t.Fatalf("frame 4: mob roster should show (dead), got:\n%s", joined)
			}
		}
	}

	// Frame 5: same state (assert frame) — XP awarded
	f5 := traceFrameToScenarioFrame(trace.Frames[5], "p1")
	outcome5 := stripANSI(renderOutcomePanel(sidePanelWidth, f5.pr, f5.er))
	if !strings.Contains(outcome5, "xp +50") {
		t.Fatalf("frame 5: outcome should show xp +50, got:\n%s", outcome5)
	}
}

func TestE2ECombat1v1SpatialProgression(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_combat_1v1.json"))

	// Frame 0: player at origin
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	canvas0, bounds := spatialCanvas(20, 10)
	map0 := overlayPlayer(canvas0, f0.pr.Position, bounds, 20, 10, false)
	if !strings.Contains(map0, "@") {
		t.Fatalf("frame 0: should show player marker @, got:\n%s", map0)
	}

	// Frame 0: mobs visible
	mob0 := overlayMobs(canvas0, f0.mobr.Mobs, bounds, 20, 10)
	if !strings.Contains(mob0, "m") {
		t.Fatalf("frame 0: should show mob marker m, got:\n%s", mob0)
	}

	// Frame 2: player moved to range (position changed)
	f2 := traceFrameToScenarioFrame(trace.Frames[2], "p1")
	canvas2, _ := spatialCanvas(20, 10)
	map2 := overlayPlayer(canvas2, f2.pr.Position, bounds, 20, 10, false)
	if !strings.Contains(map2, "@") {
		t.Fatalf("frame 2: should show player marker @ after move, got:\n%s", map2)
	}

	// Player position should differ between frame 0 and frame 2
	if f0.pr.Position.X == f2.pr.Position.X && f0.pr.Position.Y == f2.pr.Position.Y {
		t.Fatalf("player position should change between frame 0 and frame 2")
	}
}

func TestE2EMultiplayerLifecycle(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_group_2p_1m.json"))
	if len(trace.Frames) < 11 {
		t.Fatalf("expected at least 11 frames, got %d", len(trace.Frames))
	}

	// Frame 0: only p1 exists
	if len(trace.Frames[0].Players) != 1 {
		t.Fatalf("frame 0: expected 1 player, got %d", len(trace.Frames[0].Players))
	}

	// Frame 1: p2 joins
	if len(trace.Frames[1].Players) != 2 {
		t.Fatalf("frame 1: expected 2 players, got %d", len(trace.Frames[1].Players))
	}

	// Frame 6: encounter_join — both players in encounter
	f6 := traceFrameToScenarioFrame(trace.Frames[6], "p1")
	if f6.er.Count == 0 {
		t.Fatalf("frame 6: should have encounter after encounter_join")
	}
	enc6 := f6.er.Encounters[0]
	if len(enc6.PlayerIDs) != 2 {
		t.Fatalf("frame 6: encounter should have 2 players, got %d", len(enc6.PlayerIDs))
	}
	focus6 := rosterFocus{index: 0}
	encPanel6 := stripANSI(renderEncounterPanel(sidePanelWidth, f6.pr, f6.er, focus6, "p1"))
	if !strings.Contains(encPanel6, "p1") || !strings.Contains(encPanel6, "p2") {
		t.Fatalf("frame 6: encounter panel should show both p1 and p2, got:\n%s", encPanel6)
	}

	// Frame 9: combat complete, victory
	f9 := traceFrameToScenarioFrame(trace.Frames[9], "p1")
	combat9 := stripANSI(renderCombatPanel(sidePanelWidth, f9.ar, f9.pr, f9.er, defaultTarget(), f9.inv, ""))
	if !strings.Contains(combat9, "VICTORY") {
		t.Fatalf("frame 9: should show VICTORY, got:\n%s", combat9)
	}

	// Frame 9: both players took damage (HP < 100)
	f9p1 := traceFrameToScenarioFrame(trace.Frames[9], "p1")
	f9p2 := traceFrameToScenarioFrame(trace.Frames[9], "p2")
	player9p1 := stripANSI(renderPlayerPanel(sidePanelWidth, f9p1.pr, f9p1.inv, respawnResult{}))
	player9p2 := stripANSI(renderPlayerPanel(sidePanelWidth, f9p2.pr, f9p2.inv, respawnResult{}))
	if strings.Contains(player9p1, "100/100") {
		t.Fatalf("frame 9: p1 HP should be reduced, got:\n%s", player9p1)
	}
	if strings.Contains(player9p2, "100/100") {
		t.Fatalf("frame 9: p2 HP should be reduced, got:\n%s", player9p2)
	}

	// Frame 10: XP awarded
	f10 := traceFrameToScenarioFrame(trace.Frames[10], "p1")
	outcome10 := stripANSI(renderOutcomePanel(sidePanelWidth, f10.pr, f10.er))
	if !strings.Contains(outcome10, "xp +50") {
		t.Fatalf("frame 10: outcome should show xp +50, got:\n%s", outcome10)
	}

	// No ghost players — only p1 and p2 in frame 10
	if len(trace.Frames[10].Players) != 2 {
		t.Fatalf("frame 10: expected 2 players, got %d", len(trace.Frames[10].Players))
	}
}

func TestE2ETraceAdapterRoundtrip(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_combat_1v1.json"))

	for i, frame := range trace.Frames {
		sf := traceFrameToScenarioFrame(frame, "p1")

		// Every frame should produce a renderable player panel
		panel := renderPlayerPanel(sidePanelWidth, sf.pr, sf.inv, respawnResult{})
		if panel == "" {
			t.Fatalf("frame %d: player panel should not be empty", i)
		}

		// If encounter exists, combat panel should render
		if sf.er.Count > 0 {
			combat := renderCombatPanel(sidePanelWidth, sf.ar, sf.pr, sf.er, defaultTarget(), sf.inv, "")
			if combat == "" {
				t.Fatalf("frame %d: combat panel should not be empty when encounter exists", i)
			}
		}
	}
}

func TestE2ETraceDeterminism(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_combat_1v1.json"))

	// Convert and render same frame twice — output must be identical
	for i, frame := range trace.Frames {
		sf1 := traceFrameToScenarioFrame(frame, "p1")
		sf2 := traceFrameToScenarioFrame(frame, "p1")

		panel1 := renderPlayerPanel(sidePanelWidth, sf1.pr, sf1.inv, respawnResult{})
		panel2 := renderPlayerPanel(sidePanelWidth, sf2.pr, sf2.inv, respawnResult{})
		if panel1 != panel2 {
			t.Fatalf("frame %d: deterministic render failed — player panel differs between runs", i)
		}

		if sf1.er.Count > 0 {
			c1 := renderCombatPanel(sidePanelWidth, sf1.ar, sf1.pr, sf1.er, defaultTarget(), sf1.inv, "")
			c2 := renderCombatPanel(sidePanelWidth, sf2.ar, sf2.pr, sf2.er, defaultTarget(), sf2.inv, "")
			if c1 != c2 {
				t.Fatalf("frame %d: deterministic render failed — combat panel differs between runs", i)
			}
		}
	}
}

func TestE2ECombat1v1MobStateTracking(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_combat_1v1.json"))

	// Frame 0: mob should be alive
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	if len(f0.mobr.Mobs) == 0 {
		t.Fatalf("frame 0: should have mobs")
	}

	// Frame 0: mob visible on map
	canvas, bounds := spatialCanvas(20, 10)
	mobMap := overlayMobs(canvas, f0.mobr.Mobs, bounds, 20, 10)
	if !strings.Contains(mobMap, "m") {
		t.Fatalf("frame 0: should show mob marker, got:\n%s", mobMap)
	}

	// Frame 4: encounter completed — check mob dead in encounter data
	f4 := traceFrameToScenarioFrame(trace.Frames[4], "p1")
	if f4.er.Count > 0 {
		enc := f4.er.Encounters[0]
		if enc.MobsDead != 1 {
			t.Fatalf("frame 4: expected 1 dead mob, got %d", enc.MobsDead)
		}
		if enc.MobsAlive != 0 {
			t.Fatalf("frame 4: expected 0 alive mobs, got %d", enc.MobsAlive)
		}
	}
}

// === Wipe/Death/Respawn/Re-engage E2E Tests ===

func TestE2EWipeReengageLifecycle(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_wipe_reengage.json"))

	// Find wipe frame: encounter Completed with all_players_dead
	wipeIdx := -1
	for i, f := range trace.Frames {
		if f.Encounter != nil && f.Encounter.State == "Completed" && f.Encounter.CompletedReason == "all_players_dead" {
			wipeIdx = i
			break
		}
	}
	if wipeIdx < 0 {
		t.Fatal("no wipe frame found (encounter Completed with all_players_dead)")
	}

	sf := traceFrameToScenarioFrame(trace.Frames[wipeIdx], "p1")

	// Combat panel should show DEFEAT
	combat := stripANSI(renderCombatPanel(sidePanelWidth, sf.ar, sf.pr, sf.er, defaultTarget(), sf.inv, ""))
	if !strings.Contains(combat, "DEFEAT") {
		t.Fatalf("wipe frame: combat panel should contain DEFEAT, got:\n%s", combat)
	}

	// Player panel should show DEAD
	player := stripANSI(renderPlayerPanel(sidePanelWidth, sf.pr, sf.inv, respawnResult{}))
	if !strings.Contains(player, "DEAD") {
		t.Fatalf("wipe frame: player panel should contain DEAD, got:\n%s", player)
	}

	// Outcome panel should show DEFEAT
	outcome := stripANSI(renderOutcomePanel(sidePanelWidth, sf.pr, sf.er))
	if !strings.Contains(outcome, "DEFEAT") {
		t.Fatalf("wipe frame: outcome panel should contain DEFEAT, got:\n%s", outcome)
	}
}

func TestE2EWipeReengageDeathState(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_wipe_reengage.json"))

	// Find frame where p1 has can_act=false and hp=0
	deathIdx := -1
	for i, f := range trace.Frames {
		for _, p := range f.Players {
			if p.PlayerID == "p1" && !p.CanAct && p.HPCurrent <= 0 {
				deathIdx = i
				break
			}
		}
		if deathIdx >= 0 {
			break
		}
	}
	if deathIdx < 0 {
		t.Fatal("no death frame found for p1 (can_act=false, HP<=0)")
	}

	sf := traceFrameToScenarioFrame(trace.Frames[deathIdx], "p1")

	// Player panel should show DEAD with 0/
	player := stripANSI(renderPlayerPanel(sidePanelWidth, sf.pr, sf.inv, respawnResult{}))
	if !strings.Contains(player, "DEAD") {
		t.Fatalf("death frame: player panel should contain DEAD, got:\n%s", player)
	}
	if !strings.Contains(player, "0/") {
		t.Fatalf("death frame: player panel should contain 0/ (zero HP), got:\n%s", player)
	}

	// Verify blocked_reason mapped through adapter
	if sf.inv.BlockedReason != "player_dead" {
		t.Fatalf("death frame: inv.BlockedReason should be player_dead, got %q", sf.inv.BlockedReason)
	}
}

func TestE2EWipeReengageRespawnTransition(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_wipe_reengage.json"))

	// Find death frame (p1 can_act=false, HP<=0)
	deathIdx := -1
	for i, f := range trace.Frames {
		for _, p := range f.Players {
			if p.PlayerID == "p1" && !p.CanAct && p.HPCurrent <= 0 {
				deathIdx = i
				break
			}
		}
		if deathIdx >= 0 {
			break
		}
	}
	if deathIdx < 0 {
		t.Fatal("no death frame found for p1")
	}

	// Find respawn frame (p1 can_act=true AFTER death, HP > 0)
	respawnIdx := -1
	for i := deathIdx + 1; i < len(trace.Frames); i++ {
		for _, p := range trace.Frames[i].Players {
			if p.PlayerID == "p1" && p.CanAct && p.HPCurrent > 0 {
				respawnIdx = i
				break
			}
		}
		if respawnIdx >= 0 {
			break
		}
	}
	if respawnIdx < 0 {
		t.Fatal("no respawn frame found for p1 (can_act=true after death)")
	}

	// Death frame should show DEAD
	sfDead := traceFrameToScenarioFrame(trace.Frames[deathIdx], "p1")
	deadPanel := stripANSI(renderPlayerPanel(sidePanelWidth, sfDead.pr, sfDead.inv, respawnResult{}))
	if !strings.Contains(deadPanel, "DEAD") {
		t.Fatalf("death frame should show DEAD, got:\n%s", deadPanel)
	}

	// Respawn frame should NOT show DEAD and should show HP
	sfAlive := traceFrameToScenarioFrame(trace.Frames[respawnIdx], "p1")
	alivePanel := stripANSI(renderPlayerPanel(sidePanelWidth, sfAlive.pr, sfAlive.inv, respawnResult{}))
	if strings.Contains(alivePanel, "DEAD") {
		t.Fatalf("respawn frame should NOT show DEAD, got:\n%s", alivePanel)
	}
	if !strings.Contains(alivePanel, "/") {
		t.Fatalf("respawn frame should show HP fraction, got:\n%s", alivePanel)
	}
}

func TestE2EWipeReengageNewEncounterIdentity(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_wipe_reengage.json"))

	// Find wipe encounter ID
	wipeEncID := ""
	for _, f := range trace.Frames {
		if f.Encounter != nil && f.Encounter.CompletedReason == "all_players_dead" {
			wipeEncID = f.Encounter.EncounterID
			break
		}
	}
	if wipeEncID == "" {
		t.Fatal("no wipe encounter found")
	}

	// Find re-engage encounter ID (Active encounter with different ID, after the wipe)
	reengageEncID := ""
	foundWipe := false
	for _, f := range trace.Frames {
		if f.Encounter != nil && f.Encounter.CompletedReason == "all_players_dead" {
			foundWipe = true
			continue
		}
		if foundWipe && f.Encounter != nil && f.Encounter.State == "Active" && f.Encounter.EncounterID != wipeEncID {
			reengageEncID = f.Encounter.EncounterID
			break
		}
	}
	if reengageEncID == "" {
		t.Fatal("no re-engage encounter found with different ID")
	}

	// IDs must differ
	if wipeEncID == reengageEncID {
		t.Fatalf("wipe and re-engage encounter IDs should differ: %s", wipeEncID)
	}
}

func TestE2EWipeReengageCompletedReason(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_wipe_reengage.json"))

	// Find wipe frame
	wipeIdx := -1
	for i, f := range trace.Frames {
		if f.Encounter != nil && f.Encounter.State == "Completed" && f.Encounter.CompletedReason == "all_players_dead" {
			wipeIdx = i
			break
		}
	}
	if wipeIdx < 0 {
		t.Fatal("no wipe frame found")
	}

	sf := traceFrameToScenarioFrame(trace.Frames[wipeIdx], "p1")
	outcome := stripANSI(renderOutcomePanel(sidePanelWidth, sf.pr, sf.er))

	// DEFEAT should be visible (from completed_reason=all_players_dead)
	if !strings.Contains(outcome, "DEFEAT") {
		t.Fatalf("outcome panel should contain DEFEAT for wipe, got:\n%s", outcome)
	}

	// VICTORY should NOT be present
	if strings.Contains(outcome, "VICTORY") {
		t.Fatalf("outcome panel should NOT contain VICTORY for wipe, got:\n%s", outcome)
	}
}

// === E2E Pacing Readability Tests ===

func TestE2EPacingCombat1v1HPVisibility(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_combat_1v1.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Frame 0 (join): full HP should be visible
	sf0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	panel0 := stripANSI(renderPlayerPanel(sidePanelWidth, sf0.pr, sf0.inv, respawnResult{}))
	if !strings.Contains(panel0, "100/100") {
		t.Fatalf("frame 0: player panel should show full HP 100/100, got:\n%s", panel0)
	}

	// Frame 4 (attack_until_complete): HP should be reduced
	sf4 := traceFrameToScenarioFrame(trace.Frames[4], "p1")
	panel4 := stripANSI(renderPlayerPanel(sidePanelWidth, sf4.pr, sf4.inv, respawnResult{}))
	if strings.Contains(panel4, "100/100") {
		t.Fatalf("frame 4: player panel should NOT show full HP after combat, got:\n%s", panel4)
	}
	if !strings.Contains(panel4, "/") {
		t.Fatalf("frame 4: player panel should contain HP fraction, got:\n%s", panel4)
	}

	// Frame 4: mob roster should show (dead) since encounter completed with all_mobs_dead
	if sf4.er.Count > 0 {
		enc := findPlayerEncounter(sf4.er.Encounters, sf4.pr.ActiveEncounterID)
		if enc != nil {
			roster := renderCombatMobRoster(enc, sf4.ar, "p1", sidePanelWidth, "")
			rosterStr := stripANSI(strings.Join(roster, "\n"))
			if !strings.Contains(rosterStr, "(dead)") {
				t.Fatalf("frame 4: mob roster should contain (dead), got:\n%s", rosterStr)
			}
		}
	}
}

func TestE2EPacingWipeHPProgression(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_wipe_reengage.json"))

	// Collect rendered player panels for all frames where p1 exists and encounter is Active
	var renderedPanels []string
	for _, f := range trace.Frames {
		if f.Encounter == nil || f.Encounter.State != "Active" {
			continue
		}
		// Check that p1 exists in this frame
		hasP1 := false
		for _, p := range f.Players {
			if p.PlayerID == "p1" {
				hasP1 = true
				break
			}
		}
		if !hasP1 {
			continue
		}

		sf := traceFrameToScenarioFrame(f, "p1")
		panel := stripANSI(renderPlayerPanel(sidePanelWidth, sf.pr, sf.inv, respawnResult{}))
		renderedPanels = append(renderedPanels, panel)
	}

	if len(renderedPanels) < 2 {
		t.Fatalf("expected at least 2 active combat frames with p1, got %d", len(renderedPanels))
	}

	// Verify that at least 2 different HP values appear across active combat frames
	// (HP decline is visible, not a single jump)
	unique := make(map[string]bool)
	for _, p := range renderedPanels {
		unique[p] = true
	}
	if len(unique) < 2 {
		t.Fatalf("expected at least 2 distinct player panel renderings across active combat frames, got %d unique out of %d total:\n%s",
			len(unique), len(renderedPanels), renderedPanels[0])
	}
}

func TestE2EPacingSpatialRangeTransition(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_combat_1v1.json"))
	if len(trace.Frames) < 3 {
		t.Fatalf("expected at least 3 frames, got %d", len(trace.Frames))
	}

	// Frame 0 (join): p1 at (0,0)
	sf0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	// Frame 2 (move_to_range): p1 at (6,0)
	sf2 := traceFrameToScenarioFrame(trace.Frames[2], "p1")

	canvas0, bounds0 := spatialCanvas(20, 10)
	map0 := overlayPlayer(canvas0, sf0.pr.Position, bounds0, 20, 10, false)
	canvas2, bounds2 := spatialCanvas(20, 10)
	map2 := overlayPlayer(canvas2, sf2.pr.Position, bounds2, 20, 10, false)

	// Both should contain player marker
	if !strings.Contains(map0, "@") {
		t.Fatalf("frame 0: map should contain player marker @, got:\n%s", map0)
	}
	if !strings.Contains(map2, "@") {
		t.Fatalf("frame 2: map should contain player marker @, got:\n%s", map2)
	}

	// Maps should differ (position changed from (0,0) to (6,0))
	if map0 == map2 {
		t.Fatalf("frame 0 and frame 2 maps should differ after move, both:\n%s", map0)
	}

	// Render proximity panel for frame 2 (within range after move)
	// renderProximityPanel needs a focused roster entry with a mob to show range info
	var entries []rosterEntry
	if len(sf2.mobr.Mobs) > 0 {
		entries = append(entries, rosterEntry{kind: "mb", id: sf2.mobr.Mobs[0].MobID, mobID: sf2.mobr.Mobs[0].MobID})
	}
	focus := rosterFocus{index: 0}
	proxPanel := stripANSI(renderProximityPanel(sidePanelWidth, sf2.tc, sf2.ar, sf2.mobr, sf2.pr, focus, entries))
	// Should contain either "IN RANGE" or a distance value
	if !strings.Contains(proxPanel, "IN RANGE") && !strings.Contains(proxPanel, "dist:") {
		t.Fatalf("frame 2: proximity panel should show range info after move, got:\n%s", proxPanel)
	}
}

func TestE2EPacingEncounterStateProgression(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_combat_1v1.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Frame 0 (join): no encounter
	sf0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	combat0 := stripANSI(renderCombatPanel(sidePanelWidth, sf0.ar, sf0.pr, sf0.er, defaultTarget(), sf0.inv, ""))
	combat0Lower := strings.ToLower(combat0)
	// No encounter visible — should NOT contain "active" or "VICTORY"
	if strings.Contains(combat0Lower, "active") {
		t.Fatalf("frame 0: combat panel should not show active state (no encounter), got:\n%s", combat0)
	}
	if strings.Contains(combat0Lower, "victory") {
		t.Fatalf("frame 0: combat panel should not show VICTORY (no encounter), got:\n%s", combat0)
	}

	// Frame 4 (completed): encounter Completed with all_mobs_dead
	sf4 := traceFrameToScenarioFrame(trace.Frames[4], "p1")
	combat4 := stripANSI(renderCombatPanel(sidePanelWidth, sf4.ar, sf4.pr, sf4.er, defaultTarget(), sf4.inv, ""))
	if !strings.Contains(combat4, "VICTORY") {
		t.Fatalf("frame 4: combat panel should contain VICTORY for completed encounter, got:\n%s", combat4)
	}
}

func TestE2EPacingPanelConsistency(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_combat_1v1.json"))

	for i, f := range trace.Frames {
		sf := traceFrameToScenarioFrame(f, "p1")

		// Player panel should always be non-empty
		playerPanel := stripANSI(renderPlayerPanel(sidePanelWidth, sf.pr, sf.inv, respawnResult{}))
		if strings.TrimSpace(playerPanel) == "" {
			t.Fatalf("frame %d: player panel should not be empty", i)
		}

		// If encounter exists, combat panel should be non-empty
		if sf.er.Count > 0 {
			combatPanel := stripANSI(renderCombatPanel(sidePanelWidth, sf.ar, sf.pr, sf.er, defaultTarget(), sf.inv, ""))
			if strings.TrimSpace(combatPanel) == "" {
				t.Fatalf("frame %d: combat panel should not be empty when encounter exists", i)
			}
		}
	}
}

// === E2E Event Stream Tests ===

func TestE2EEventStreamCombat1v1OrderedMessages(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_combat_1v1.json"))

	combatFrame := trace.Frames[4]
	sf := traceFrameToScenarioFrame(combatFrame, "p1")

	entries := sf.events.recent()
	if len(entries) == 0 {
		t.Fatal("combat frame should have event log entries from trace events")
	}

	for i, e := range entries {
		if strings.TrimSpace(e.text) == "" {
			t.Fatalf("event %d has empty text", i)
		}
	}

	panel := stripANSI(renderEventPanel(sidePanelWidth, sf.events, sf.pr, sf.er))
	if !strings.Contains(panel, "hit") {
		t.Fatal("event panel should contain 'hit' from combat events")
	}
}

func TestE2EEventStreamHitDirectionVisible(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_combat_1v1.json"))
	combatFrame := trace.Frames[4]
	sf := traceFrameToScenarioFrame(combatFrame, "p1")

	entries := sf.events.recent()

	foundPlayerHit := false
	foundMobHit := false
	for _, e := range entries {
		if strings.Contains(e.text, "p1") && strings.Contains(e.text, "m1") && strings.Contains(e.text, "hit") {
			foundPlayerHit = true
		}
		if strings.Contains(e.text, "m1") && strings.Contains(e.text, "p1") && strings.Contains(e.text, "hit") {
			foundMobHit = true
		}
	}
	if !foundPlayerHit {
		t.Fatal("expected player>mob hit event with actor>target direction visible")
	}
	if !foundMobHit {
		t.Fatal("expected mob>player hit event with actor>target direction visible")
	}
}

func TestE2EEventStreamNoDuplicateSpam(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_combat_1v1.json"))
	combatFrame := trace.Frames[4]
	sf := traceFrameToScenarioFrame(combatFrame, "p1")

	entries := sf.events.recent()
	if len(entries) < 2 {
		t.Fatal("expected multiple event entries")
	}

	for i := 1; i < len(entries); i++ {
		if entries[i].text == entries[i-1].text {
			t.Fatalf("duplicate consecutive events at %d: %q", i, entries[i].text)
		}
	}
}

func TestE2EEventStreamOrderMatchesTrace(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_combat_1v1.json"))
	combatFrame := trace.Frames[4]
	sf := traceFrameToScenarioFrame(combatFrame, "p1")

	entries := sf.events.recent()

	if len(entries) < 4 {
		t.Fatalf("expected at least 4 events, got %d", len(entries))
	}

	if !strings.Contains(entries[0].text, "p1") || !strings.Contains(entries[0].text, "hit") {
		t.Fatalf("first event should be player hit, got %q", entries[0].text)
	}

	if !strings.Contains(entries[1].text, "m1") || !strings.Contains(entries[1].text, "hit") {
		t.Fatalf("second event should be mob hit, got %q", entries[1].text)
	}
}

func TestE2EEventStreamWipeDeathVisibleBeforeDeathFrame(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_wipe_reengage.json"))

	deathFrameIdx := -1
	for i, f := range trace.Frames {
		for _, p := range f.Players {
			if p.PlayerID == "p1" && p.HPCurrent <= 0 {
				deathFrameIdx = i
				break
			}
		}
		if deathFrameIdx >= 0 {
			break
		}
	}
	if deathFrameIdx < 0 {
		t.Fatal("no death frame found for p1")
	}

	var allEvents []traceEvent
	for i := 0; i <= deathFrameIdx && i < len(trace.Frames); i++ {
		allEvents = append(allEvents, trace.Frames[i].Events...)
	}

	foundDeath := false
	for _, e := range allEvents {
		if e.Kind == "player_death" && e.Target == "p1" {
			foundDeath = true
			break
		}
	}
	if !foundDeath {
		t.Fatal("expected player_death event for p1 at or before death frame")
	}

	foundMobHit := false
	for _, e := range allEvents {
		if e.Kind == "mob_hit" && e.Target == "p1" {
			foundMobHit = true
			break
		}
		if e.Kind == "player_death" && e.Target == "p1" {
			break
		}
	}
	if !foundMobHit {
		t.Fatal("expected mob_hit events for p1 before player_death")
	}
}

func TestE2EEventStreamWipeDeathRendersInPanel(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_wipe_reengage.json"))

	var deathSF scenarioFrame
	found := false
	for _, f := range trace.Frames {
		for _, e := range f.Events {
			if e.Kind == "player_death" {
				deathSF = traceFrameToScenarioFrame(f, "p1")
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("no frame with player_death events")
	}

	entries := deathSF.events.recent()
	foundDeathEvent := false
	for _, e := range entries {
		if e.text == "DEFEAT" {
			foundDeathEvent = true
			break
		}
	}
	if !foundDeathEvent {
		t.Fatal("expected DEFEAT event in event log for death frame")
	}

	panel := stripANSI(renderEventPanel(sidePanelWidth, deathSF.events, deathSF.pr, deathSF.er))
	if !strings.Contains(panel, "DEFEAT") {
		t.Fatal("event panel should show DEFEAT for death frame")
	}
}

func TestE2EEventStreamMobDefeatedVisible(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_combat_1v1.json"))
	combatFrame := trace.Frames[4]
	sf := traceFrameToScenarioFrame(combatFrame, "p1")

	entries := sf.events.recent()
	foundDefeated := false
	for _, e := range entries {
		if e.text == "mob defeated" {
			foundDefeated = true
			break
		}
	}
	if !foundDefeated {
		t.Fatal("expected 'mob defeated' event in combat log")
	}

	panel := stripANSI(renderEventPanel(sidePanelWidth, sf.events, sf.pr, sf.er))
	if !strings.Contains(panel, "mob defeated") {
		t.Fatal("event panel should show 'mob defeated'")
	}
}

// === M20260426-16 Ability Cooldown E2E Tests ===

func TestE2EAbilityCooldownHitVisible(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_ability_cooldown.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Frame 1: heavy_strike success — should show player hit in event panel
	f1 := traceFrameToScenarioFrame(trace.Frames[1], "p1")
	panel1 := stripANSI(renderEventPanel(sidePanelWidth, f1.events, f1.pr, f1.er))
	if !strings.Contains(panel1, "hit") {
		t.Fatalf("frame 1: ability hit should be visible in event panel, got:\n%s", panel1)
	}
	if !strings.Contains(panel1, "38") {
		t.Fatalf("frame 1: ability damage value 38 should be visible, got:\n%s", panel1)
	}
}

func TestE2EAbilityCooldownDenialVisible(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_ability_cooldown.json"))

	// Frame 2: cooldown_blocked — should show denial in event panel
	f2 := traceFrameToScenarioFrame(trace.Frames[2], "p1")
	panel2 := stripANSI(renderEventPanel(sidePanelWidth, f2.events, f2.pr, f2.er))
	if !strings.Contains(panel2, "deny cooldown") {
		t.Fatalf("frame 2: cooldown denial should show 'deny cooldown' in event panel, got:\n%s", panel2)
	}
}

func TestE2EAbilityCooldownDenialNoActionIndexAdvance(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_ability_cooldown.json"))

	// Frame 1: action_index = 1 after successful hit
	f1 := traceFrameToScenarioFrame(trace.Frames[1], "p1")
	if f1.er.Encounters[0].ActionIndex != 1 {
		t.Fatalf("frame 1: expected action_index 1, got %d", f1.er.Encounters[0].ActionIndex)
	}

	// Frame 2: action_index still 1 after cooldown denial
	f2 := traceFrameToScenarioFrame(trace.Frames[2], "p1")
	if f2.er.Encounters[0].ActionIndex != 1 {
		t.Fatalf("frame 2: expected action_index 1 (unchanged after denial), got %d", f2.er.Encounters[0].ActionIndex)
	}

	// Frame 4: action_index = 2 after second success
	f4 := traceFrameToScenarioFrame(trace.Frames[4], "p1")
	if f4.er.Encounters[0].ActionIndex != 2 {
		t.Fatalf("frame 4: expected action_index 2 after second success, got %d", f4.er.Encounters[0].ActionIndex)
	}
}

func TestE2EAbilityCooldownNoDuplicateSpam(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_ability_cooldown.json"))

	// Frame 2: should have exactly one denial entry, not repeated
	f2 := traceFrameToScenarioFrame(trace.Frames[2], "p1")
	entries := f2.events.recent()
	denialCount := 0
	for _, e := range entries {
		if e.text == "deny cooldown" {
			denialCount++
		}
	}
	if denialCount != 1 {
		t.Fatalf("expected exactly 1 'deny cooldown' event, got %d", denialCount)
	}
}

func TestE2EAbilityCooldownSuccessAfterExpiry(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_ability_cooldown.json"))

	// Frame 4: second success after cooldown expires
	f4 := traceFrameToScenarioFrame(trace.Frames[4], "p1")
	panel4 := stripANSI(renderEventPanel(sidePanelWidth, f4.events, f4.pr, f4.er))
	if !strings.Contains(panel4, "hit") {
		t.Fatalf("frame 4: second ability hit should be visible after cooldown expires, got:\n%s", panel4)
	}
	if !strings.Contains(panel4, "38") {
		t.Fatalf("frame 4: damage value should be visible after cooldown expires, got:\n%s", panel4)
	}

	// Should NOT contain denial in this frame
	if strings.Contains(panel4, "deny cooldown") {
		t.Fatalf("frame 4: should not show denial in success frame, got:\n%s", panel4)
	}
}

// === Cast Lifecycle E2E Tests ===

func TestCastStartVisibleInEventStream(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_cast_lifecycle.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Frame 0: cast_start — should show "casting" in event panel
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	panel0 := stripANSI(renderEventPanel(sidePanelWidth, f0.events, f0.pr, f0.er))
	if !strings.Contains(panel0, "casting") {
		t.Fatalf("frame 0: cast_start should show 'casting' in event panel, got:\n%s", panel0)
	}
}

func TestCastCompleteVisibleInEventStream(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_cast_lifecycle.json"))

	// Frame 2: cast_complete + damage — should show both
	f2 := traceFrameToScenarioFrame(trace.Frames[2], "p1")
	panel2 := stripANSI(renderEventPanel(sidePanelWidth, f2.events, f2.pr, f2.er))
	if !strings.Contains(panel2, "cast complete") {
		t.Fatalf("frame 2: cast_complete should show 'cast complete' in event panel, got:\n%s", panel2)
	}
	if !strings.Contains(panel2, "hit") {
		t.Fatalf("frame 2: damage after cast_complete should show 'hit' in event panel, got:\n%s", panel2)
	}
	if !strings.Contains(panel2, "18") {
		t.Fatalf("frame 2: damage value 18 should be visible after cast_complete, got:\n%s", panel2)
	}
}

func TestCastDeniedVisibleInEventStream(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_cast_lifecycle.json"))

	// Frame 4: cast_denied — should show "deny cast" in event panel
	f4 := traceFrameToScenarioFrame(trace.Frames[4], "p1")
	panel4 := stripANSI(renderEventPanel(sidePanelWidth, f4.events, f4.pr, f4.er))
	if !strings.Contains(panel4, "deny cast") {
		t.Fatalf("frame 4: cast_denied should show 'deny cast' in event panel, got:\n%s", panel4)
	}
}

func TestCastLifecycleProgressionAcrossFrames(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_cast_lifecycle.json"))

	// Process frames 0 through 2 and verify ordering: cast_start appears first, then cast_complete later
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	f2 := traceFrameToScenarioFrame(trace.Frames[2], "p1")

	// Frame 0 should have casting but not cast complete
	entries0 := f0.events.recent()
	hasCasting := false
	hasCastComplete := false
	for _, e := range entries0 {
		if e.text == "casting" {
			hasCasting = true
		}
		if e.text == "cast complete" {
			hasCastComplete = true
		}
	}
	if !hasCasting {
		t.Fatalf("frame 0: should have 'casting' event")
	}
	if hasCastComplete {
		t.Fatalf("frame 0: should NOT have 'cast complete' event yet")
	}

	// Frame 2 should have cast complete
	entries2 := f2.events.recent()
	hasCastComplete = false
	for _, e := range entries2 {
		if e.text == "cast complete" {
			hasCastComplete = true
		}
	}
	if !hasCastComplete {
		t.Fatalf("frame 2: should have 'cast complete' event")
	}
}

func TestCastNoDuplicateSpam(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_cast_lifecycle.json"))

	// Process all frames and verify no duplicate consecutive identical event lines
	var allEvents []eventEntry
	for _, frame := range trace.Frames {
		sf := traceFrameToScenarioFrame(frame, "p1")
		entries := sf.events.recent()
		allEvents = append(allEvents, entries...)
	}

	for i := 1; i < len(allEvents); i++ {
		if allEvents[i].text == allEvents[i-1].text && allEvents[i].text != "" {
			// Allow combat events like mob_hit to repeat (they are distinct combat rounds)
			if allEvents[i].kind == "combat" {
				continue
			}
			t.Fatalf("duplicate consecutive event at index %d: %q (kind=%s)", i, allEvents[i].text, allEvents[i].kind)
		}
	}
}

// --- Cast Interrupt / Movement Cancel Tests (trace_cast_interrupt.json) ---

func TestCastCancelledByMovementVisible(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_cast_interrupt.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Frame 1: cast_cancelled by movement — should show "cast cancelled" in event panel
	f1 := traceFrameToScenarioFrame(trace.Frames[1], "p1")
	panel1 := stripANSI(renderEventPanel(sidePanelWidth, f1.events, f1.pr, f1.er))
	if !strings.Contains(panel1, "cast cancelled") {
		t.Fatalf("frame 1: cast_cancelled should show 'cast cancelled' in event panel, got:\n%s", panel1)
	}
	// Should NOT contain damage text
	if strings.Contains(panel1, "hit") {
		t.Fatalf("frame 1: cast_cancelled frame should NOT show 'hit' (no damage), got:\n%s", panel1)
	}
}

func TestCastRecastAfterCancelVisible(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_cast_interrupt.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Process frames 1 (cancel), 2 (recast), 3 (complete+damage)
	f1 := traceFrameToScenarioFrame(trace.Frames[1], "p1")
	f2 := traceFrameToScenarioFrame(trace.Frames[2], "p1")
	f3 := traceFrameToScenarioFrame(trace.Frames[3], "p1")

	// Frame 1: cast_cancelled appears
	entries1 := f1.events.recent()
	hasCancelled := false
	for _, e := range entries1 {
		if e.text == "cast cancelled" {
			hasCancelled = true
		}
	}
	if !hasCancelled {
		t.Fatalf("frame 1: should have 'cast cancelled' event")
	}

	// Frame 2: cast_start appears again (recast)
	entries2 := f2.events.recent()
	hasCasting := false
	for _, e := range entries2 {
		if e.text == "casting" {
			hasCasting = true
		}
	}
	if !hasCasting {
		t.Fatalf("frame 2: should have 'casting' event after recast")
	}

	// Frame 3: cast_complete + damage
	entries3 := f3.events.recent()
	hasCastComplete := false
	hasDamage := false
	for _, e := range entries3 {
		if e.text == "cast complete" {
			hasCastComplete = true
		}
		if strings.Contains(e.text, "hit") && strings.Contains(e.text, "18") {
			hasDamage = true
		}
	}
	if !hasCastComplete {
		t.Fatalf("frame 3: should have 'cast complete' event")
	}
	if !hasDamage {
		t.Fatalf("frame 3: should have damage event with value 18")
	}
}

func TestCastInterruptReasonReadable(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_cast_interrupt.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Frame 1: reason=movement — renders as "cast cancelled"
	f1 := traceFrameToScenarioFrame(trace.Frames[1], "p1")
	panel1 := stripANSI(renderEventPanel(sidePanelWidth, f1.events, f1.pr, f1.er))
	if !strings.Contains(panel1, "cast cancelled") {
		t.Fatalf("frame 1 (reason=movement): should render 'cast cancelled', got:\n%s", panel1)
	}

	// Frame 4: reason=interrupted — also renders as "cast cancelled"
	f4 := traceFrameToScenarioFrame(trace.Frames[4], "p1")
	panel4 := stripANSI(renderEventPanel(sidePanelWidth, f4.events, f4.pr, f4.er))
	if !strings.Contains(panel4, "cast cancelled") {
		t.Fatalf("frame 4 (reason=interrupted): should render 'cast cancelled', got:\n%s", panel4)
	}
}

func TestCastInterruptNoDuplicateSpam(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_cast_interrupt.json"))

	// Process all frames and verify no duplicate consecutive identical event lines
	var allEvents []eventEntry
	for _, frame := range trace.Frames {
		sf := traceFrameToScenarioFrame(frame, "p1")
		entries := sf.events.recent()
		allEvents = append(allEvents, entries...)
	}

	for i := 1; i < len(allEvents); i++ {
		if allEvents[i].text == allEvents[i-1].text && allEvents[i].text != "" {
			// Allow combat events to repeat (distinct rounds)
			if allEvents[i].kind == "combat" {
				continue
			}
			t.Fatalf("duplicate consecutive event at index %d: %q (kind=%s)", i, allEvents[i].text, allEvents[i].kind)
		}
	}
}

func TestCastInterruptEventOrdering(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_cast_interrupt.json"))
	if len(trace.Frames) < 2 {
		t.Fatalf("expected at least 2 frames, got %d", len(trace.Frames))
	}

	// Process frames 0 and 1 and verify cast_start text appears BEFORE cast_cancelled
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	f1 := traceFrameToScenarioFrame(trace.Frames[1], "p1")

	// Collect events in order across both frames
	var allEvents []eventEntry
	allEvents = append(allEvents, f0.events.recent()...)
	allEvents = append(allEvents, f1.events.recent()...)

	castingIdx := -1
	cancelledIdx := -1
	for i, e := range allEvents {
		if e.text == "casting" && castingIdx == -1 {
			castingIdx = i
		}
		if e.text == "cast cancelled" && cancelledIdx == -1 {
			cancelledIdx = i
		}
	}

	if castingIdx == -1 {
		t.Fatalf("expected 'casting' event in frames 0-1")
	}
	if cancelledIdx == -1 {
		t.Fatalf("expected 'cast cancelled' event in frames 0-1")
	}
	if castingIdx >= cancelledIdx {
		t.Fatalf("'casting' (index %d) should appear before 'cast cancelled' (index %d)", castingIdx, cancelledIdx)
	}
}

// --- Cast Progress Visibility Tests (trace_cast_progress.json) ---

func TestCastProgressVisibleDuringCast(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_cast_progress.json"))
	if len(trace.Frames) < 4 {
		t.Fatalf("expected at least 4 frames, got %d", len(trace.Frames))
	}

	// Frame 0: cast_start with active cast — combat panel should show "casting:" and "power_shot"
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	panel0 := stripANSI(renderCombatPanel(sidePanelWidth, f0.ar, f0.pr, f0.er, defaultTarget(), f0.inv, ""))
	if !strings.Contains(panel0, "casting:") {
		t.Fatalf("frame 0: combat panel should contain 'casting:' during active cast, got:\n%s", panel0)
	}
	if !strings.Contains(panel0, "power_shot") {
		t.Fatalf("frame 0: combat panel should contain 'power_shot' ability name, got:\n%s", panel0)
	}

	// Frame 1: cast still in progress — combat panel should still show "casting:" and "power_shot"
	f1 := traceFrameToScenarioFrame(trace.Frames[1], "p1")
	panel1 := stripANSI(renderCombatPanel(sidePanelWidth, f1.ar, f1.pr, f1.er, defaultTarget(), f1.inv, ""))
	if !strings.Contains(panel1, "casting:") {
		t.Fatalf("frame 1: combat panel should contain 'casting:' during active cast, got:\n%s", panel1)
	}
	if !strings.Contains(panel1, "power_shot") {
		t.Fatalf("frame 1: combat panel should contain 'power_shot' ability name, got:\n%s", panel1)
	}
}

func TestCastProgressPercentageAdvances(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_cast_progress.json"))
	if len(trace.Frames) < 2 {
		t.Fatalf("expected at least 2 frames, got %d", len(trace.Frames))
	}

	// Frame 0: 0% progress
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	panel0 := stripANSI(renderCombatPanel(sidePanelWidth, f0.ar, f0.pr, f0.er, defaultTarget(), f0.inv, ""))
	if !strings.Contains(panel0, "0%") {
		t.Fatalf("frame 0: combat panel should contain '0%%' progress, got:\n%s", panel0)
	}

	// Frame 1: 50% progress
	f1 := traceFrameToScenarioFrame(trace.Frames[1], "p1")
	panel1 := stripANSI(renderCombatPanel(sidePanelWidth, f1.ar, f1.pr, f1.er, defaultTarget(), f1.inv, ""))
	if !strings.Contains(panel1, "50%") {
		t.Fatalf("frame 1: combat panel should contain '50%%' progress, got:\n%s", panel1)
	}
}

func TestCastProgressClearedAfterCompletion(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_cast_progress.json"))
	if len(trace.Frames) < 3 {
		t.Fatalf("expected at least 3 frames, got %d", len(trace.Frames))
	}

	// Frame 2: cast_complete — no active casts, combat panel should NOT contain "casting:"
	f2 := traceFrameToScenarioFrame(trace.Frames[2], "p1")
	panel2 := stripANSI(renderCombatPanel(sidePanelWidth, f2.ar, f2.pr, f2.er, defaultTarget(), f2.inv, ""))
	if strings.Contains(panel2, "casting:") {
		t.Fatalf("frame 2: combat panel should NOT contain 'casting:' after cast completion, got:\n%s", panel2)
	}
}

func TestCastProgressClearedAfterCancellation(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_cast_progress.json"))
	if len(trace.Frames) < 4 {
		t.Fatalf("expected at least 4 frames, got %d", len(trace.Frames))
	}

	// Frame 3: cast_cancelled — no active casts, combat panel should NOT contain "casting:"
	f3 := traceFrameToScenarioFrame(trace.Frames[3], "p1")
	panel3 := stripANSI(renderCombatPanel(sidePanelWidth, f3.ar, f3.pr, f3.er, defaultTarget(), f3.inv, ""))
	if strings.Contains(panel3, "casting:") {
		t.Fatalf("frame 3: combat panel should NOT contain 'casting:' after cancellation, got:\n%s", panel3)
	}

	// Event panel should contain "cast cancelled"
	eventPanel3 := stripANSI(renderEventPanel(sidePanelWidth, f3.events, f3.pr, f3.er))
	if !strings.Contains(eventPanel3, "cast cancelled") {
		t.Fatalf("frame 3: event panel should contain 'cast cancelled', got:\n%s", eventPanel3)
	}
}

func TestCastCancellationReasonStillVisible(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_cast_progress.json"))
	if len(trace.Frames) < 4 {
		t.Fatalf("expected at least 4 frames, got %d", len(trace.Frames))
	}

	// Frame 3: cast_cancelled with reason=movement — event panel should show "cast cancelled"
	f3 := traceFrameToScenarioFrame(trace.Frames[3], "p1")
	eventPanel3 := stripANSI(renderEventPanel(sidePanelWidth, f3.events, f3.pr, f3.er))
	if !strings.Contains(eventPanel3, "cast cancelled") {
		t.Fatalf("frame 3: event panel should contain 'cast cancelled' for cancellation event, got:\n%s", eventPanel3)
	}
}

// === GCD Queue E2E Tests ===

func TestGCDStartedVisibleInEventStream(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_gcd_queue.json"))
	if len(trace.Frames) < 4 {
		t.Fatalf("expected at least 4 frames, got %d", len(trace.Frames))
	}

	// Frame 0: gcd_started — should show "gcd" in event panel
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	panel0 := stripANSI(renderEventPanel(sidePanelWidth, f0.events, f0.pr, f0.er))
	if !strings.Contains(panel0, "gcd") {
		t.Fatalf("frame 0: gcd_started should show 'gcd' in event panel, got:\n%s", panel0)
	}
}

func TestAbilityQueuedVisibleInEventStream(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_gcd_queue.json"))

	// Frame 1: ability_queued — should show "queued" and ability name (may be truncated) in event panel
	f1 := traceFrameToScenarioFrame(trace.Frames[1], "p1")
	panel1 := stripANSI(renderEventPanel(sidePanelWidth, f1.events, f1.pr, f1.er))
	if !strings.Contains(panel1, "queued") {
		t.Fatalf("frame 1: ability_queued should show 'queued' in event panel, got:\n%s", panel1)
	}
	// Ability name may be truncated in narrow panel — check the underlying event entry
	entries1 := f1.events.recent()
	hasQueuedAbility := false
	for _, e := range entries1 {
		if strings.Contains(e.text, "queued") && strings.Contains(e.text, "heavy_strike") {
			hasQueuedAbility = true
		}
	}
	if !hasQueuedAbility {
		t.Fatalf("frame 1: ability_queued event entry should contain 'queued heavy_strike', got entries: %v", entries1)
	}
}

func TestAbilityExecutedFromQueueVisible(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_gcd_queue.json"))

	// Frame 2: ability_executed_from_queue — should show "from queue" in event panel
	f2 := traceFrameToScenarioFrame(trace.Frames[2], "p1")
	panel2 := stripANSI(renderEventPanel(sidePanelWidth, f2.events, f2.pr, f2.er))
	if !strings.Contains(panel2, "from queue") {
		t.Fatalf("frame 2: ability_executed_from_queue should show 'from queue' in event panel, got:\n%s", panel2)
	}
	// Ability name may be truncated in narrow panel — check the underlying event entry
	entries2 := f2.events.recent()
	hasFromQueueAbility := false
	for _, e := range entries2 {
		if strings.Contains(e.text, "from queue") && strings.Contains(e.text, "heavy_strike") {
			hasFromQueueAbility = true
		}
	}
	if !hasFromQueueAbility {
		t.Fatalf("frame 2: from_queue event entry should contain 'from queue heavy_strike', got entries: %v", entries2)
	}
}

func TestGCDQueueEventOrdering(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_gcd_queue.json"))

	// Process frames 0-2, verify events appear in order: hit → gcd → queued → from queue → hit
	var allEvents []eventEntry
	for i := 0; i <= 2; i++ {
		sf := traceFrameToScenarioFrame(trace.Frames[i], "p1")
		allEvents = append(allEvents, sf.events.recent()...)
	}

	hitIdx := -1
	gcdIdx := -1
	queuedIdx := -1
	fromQueueIdx := -1
	for i, e := range allEvents {
		if strings.Contains(e.text, "hit") && hitIdx == -1 {
			hitIdx = i
		}
		if e.text == "gcd" && gcdIdx == -1 {
			gcdIdx = i
		}
		if strings.Contains(e.text, "queued") && !strings.Contains(e.text, "from queue") && queuedIdx == -1 {
			queuedIdx = i
		}
		if strings.Contains(e.text, "from queue") && fromQueueIdx == -1 {
			fromQueueIdx = i
		}
	}

	if hitIdx == -1 {
		t.Fatalf("expected 'hit' event in frames 0-2")
	}
	if gcdIdx == -1 {
		t.Fatalf("expected 'gcd' event in frames 0-2")
	}
	if queuedIdx == -1 {
		t.Fatalf("expected 'queued' event in frames 0-2")
	}
	if fromQueueIdx == -1 {
		t.Fatalf("expected 'from queue' event in frames 0-2")
	}

	if hitIdx >= gcdIdx {
		t.Fatalf("'hit' (index %d) should appear before 'gcd' (index %d)", hitIdx, gcdIdx)
	}
	if gcdIdx >= queuedIdx {
		t.Fatalf("'gcd' (index %d) should appear before 'queued' (index %d)", gcdIdx, queuedIdx)
	}
	if queuedIdx >= fromQueueIdx {
		t.Fatalf("'queued' (index %d) should appear before 'from queue' (index %d)", queuedIdx, fromQueueIdx)
	}
}

func TestGCDQueueNoDuplicateSpam(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_gcd_queue.json"))

	// Process all frames and verify no duplicate consecutive identical event lines
	var allEvents []eventEntry
	for _, frame := range trace.Frames {
		sf := traceFrameToScenarioFrame(frame, "p1")
		allEvents = append(allEvents, sf.events.recent()...)
	}

	for i := 1; i < len(allEvents); i++ {
		if allEvents[i].text == allEvents[i-1].text && allEvents[i].text != "" {
			// Allow combat events to repeat (distinct rounds)
			if allEvents[i].kind == "combat" {
				continue
			}
			t.Fatalf("duplicate consecutive event at index %d: %q (kind=%s)", i, allEvents[i].text, allEvents[i].kind)
		}
	}
}

func TestQueuedAbilityVisibleInCombatPanel(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_gcd_queue.json"))

	// Modify frame 1 encounter to include player_gcd data for combat panel test
	frame1 := trace.Frames[1]
	frame1.Encounter.PlayerGCD = []tracePlayerGCD{
		{PlayerID: "p1", QueuedAbilityID: "heavy_strike", QueuedTargetMobID: "live-orc-1"},
	}

	f1 := traceFrameToScenarioFrame(frame1, "p1")
	panel1 := stripANSI(renderCombatPanel(sidePanelWidth, f1.ar, f1.pr, f1.er, defaultTarget(), f1.inv, ""))
	if !strings.Contains(panel1, "queued:") {
		t.Fatalf("frame 1: combat panel should contain 'queued:' when ability is queued, got:\n%s", panel1)
	}
	if !strings.Contains(panel1, "heavy_strike") {
		t.Fatalf("frame 1: combat panel should contain 'heavy_strike' ability name, got:\n%s", panel1)
	}
}

func TestQueuedAbilityNotVisibleWhenEmpty(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_gcd_queue.json"))

	// Frame 0 has no player_gcd — combat panel should NOT show "queued:"
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	panel0 := stripANSI(renderCombatPanel(sidePanelWidth, f0.ar, f0.pr, f0.er, defaultTarget(), f0.inv, ""))
	if strings.Contains(panel0, "queued:") {
		t.Fatalf("frame 0: combat panel should NOT contain 'queued:' when no ability is queued, got:\n%s", panel0)
	}
}

// === Resource System E2E Tests ===

func TestResourceSpentEventVisibility(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_resource.json"))
	if len(trace.Frames) < 4 {
		t.Fatalf("expected at least 4 frames, got %d", len(trace.Frames))
	}

	// Frame 0: resource_spent event should show "mana -10" in event stream
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	panel0 := stripANSI(renderEventPanel(sidePanelWidth, f0.events, f0.pr, f0.er))
	if !strings.Contains(panel0, "mana -10") {
		t.Fatalf("frame 0: resource_spent should show 'mana -10' in event panel, got:\n%s", panel0)
	}

	// Frame 1: second resource_spent also shows "mana -10"
	f1 := traceFrameToScenarioFrame(trace.Frames[1], "p1")
	entries1 := f1.events.recent()
	hasSpent := false
	for _, e := range entries1 {
		if strings.Contains(e.text, "mana -10") {
			hasSpent = true
		}
	}
	if !hasSpent {
		t.Fatalf("frame 1: resource_spent event entry should contain 'mana -10', got entries: %v", entries1)
	}
}

func TestResourceBlockedEventVisibility(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_resource.json"))

	// Frame 2: resource_blocked — should show "deny mana" in event panel
	f2 := traceFrameToScenarioFrame(trace.Frames[2], "p1")
	panel2 := stripANSI(renderEventPanel(sidePanelWidth, f2.events, f2.pr, f2.er))
	if !strings.Contains(panel2, "deny mana") {
		t.Fatalf("frame 2: resource_blocked should show 'deny mana' in event panel, got:\n%s", panel2)
	}

	// Verify it classifies as denial kind
	entries2 := f2.events.recent()
	hasDenial := false
	for _, e := range entries2 {
		if e.text == "deny mana" && e.kind == "denial" {
			hasDenial = true
		}
	}
	if !hasDenial {
		t.Fatalf("frame 2: resource_blocked should classify as 'denial' kind, got entries: %v", entries2)
	}
}

func TestResourceRegenEventVisibility(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_resource.json"))

	// Frame 3: resource_regen — should show "mana +20" in event panel
	f3 := traceFrameToScenarioFrame(trace.Frames[3], "p1")
	panel3 := stripANSI(renderEventPanel(sidePanelWidth, f3.events, f3.pr, f3.er))
	if !strings.Contains(panel3, "mana +20") {
		t.Fatalf("frame 3: resource_regen should show 'mana +20' in event panel, got:\n%s", panel3)
	}

	// Verify it classifies as resource kind
	entries3 := f3.events.recent()
	hasRegen := false
	for _, e := range entries3 {
		if e.text == "mana +20" && e.kind == "resource" {
			hasRegen = true
		}
	}
	if !hasRegen {
		t.Fatalf("frame 3: resource_regen should classify as 'resource' kind, got entries: %v", entries3)
	}
}

// === Interrupt System E2E Tests ===

func TestInterruptAppliedEventVisibility(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_interrupt.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Frame 0: interrupt_applied should show "interrupted" in event panel
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	panel0 := stripANSI(renderEventPanel(sidePanelWidth, f0.events, f0.pr, f0.er))
	if !strings.Contains(panel0, "interrupted") {
		t.Fatalf("frame 0: interrupt_applied should show 'interrupted' in event panel, got:\n%s", panel0)
	}

	// Verify it classifies as resource kind
	entries0 := f0.events.recent()
	hasResource := false
	for _, e := range entries0 {
		if e.text == "interrupted" && e.kind == "resource" {
			hasResource = true
		}
	}
	if !hasResource {
		t.Fatalf("frame 0: interrupt_applied should classify as 'resource' kind, got entries: %v", entries0)
	}
}

func TestInterruptDeniedEventVisibility(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_interrupt.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Frame 3: interrupt_denied should show "deny interrupt" in event panel
	f3 := traceFrameToScenarioFrame(trace.Frames[3], "p1")
	panel3 := stripANSI(renderEventPanel(sidePanelWidth, f3.events, f3.pr, f3.er))
	if !strings.Contains(panel3, "deny interrupt") {
		t.Fatalf("frame 3: interrupt_denied should show 'deny interrupt' in event panel, got:\n%s", panel3)
	}

	// Verify it classifies as denial kind
	entries3 := f3.events.recent()
	hasDenial := false
	for _, e := range entries3 {
		if e.text == "deny interrupt" && e.kind == "denial" {
			hasDenial = true
		}
	}
	if !hasDenial {
		t.Fatalf("frame 3: interrupt_denied should classify as 'denial' kind, got entries: %v", entries3)
	}
}

func TestInterruptLockoutBlockedEventVisibility(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_interrupt.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Frame 1: interrupt_lockout_blocked should show "lockout blocked" in event panel
	f1 := traceFrameToScenarioFrame(trace.Frames[1], "p1")
	panel1 := stripANSI(renderEventPanel(sidePanelWidth, f1.events, f1.pr, f1.er))
	if !strings.Contains(panel1, "lockout blocked") {
		t.Fatalf("frame 1: interrupt_lockout_blocked should show 'lockout blocked' in event panel, got:\n%s", panel1)
	}

	// Verify it classifies as denial kind
	entries1 := f1.events.recent()
	hasDenial := false
	for _, e := range entries1 {
		if e.text == "lockout blocked" && e.kind == "denial" {
			hasDenial = true
		}
	}
	if !hasDenial {
		t.Fatalf("frame 1: interrupt_lockout_blocked should classify as 'denial' kind, got entries: %v", entries1)
	}
}

func TestInterruptEventOrderReadability(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_interrupt.json"))
	if len(trace.Frames) < 1 {
		t.Fatalf("expected at least 1 frame, got %d", len(trace.Frames))
	}

	// Frame 0 has cast_cancelled + interrupt_applied + interrupt_lockout_started
	// Verify they appear in readable order: cast cancelled before interrupted before lockout
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	entries0 := f0.events.recent()

	cancelledIdx := -1
	interruptedIdx := -1
	lockoutIdx := -1
	for i, e := range entries0 {
		if e.text == "cast cancelled" && cancelledIdx == -1 {
			cancelledIdx = i
		}
		if e.text == "interrupted" && interruptedIdx == -1 {
			interruptedIdx = i
		}
		if e.text == "lockout" && lockoutIdx == -1 {
			lockoutIdx = i
		}
	}

	if cancelledIdx == -1 {
		t.Fatalf("expected 'cast cancelled' event in frame 0")
	}
	if interruptedIdx == -1 {
		t.Fatalf("expected 'interrupted' event in frame 0")
	}
	if lockoutIdx == -1 {
		t.Fatalf("expected 'lockout' event in frame 0")
	}
	if cancelledIdx >= interruptedIdx {
		t.Fatalf("'cast cancelled' (index %d) should appear before 'interrupted' (index %d)", cancelledIdx, interruptedIdx)
	}
	if interruptedIdx >= lockoutIdx {
		t.Fatalf("'interrupted' (index %d) should appear before 'lockout' (index %d)", interruptedIdx, lockoutIdx)
	}
}

func TestManaDisplayInCombatPanel(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_resource.json"))

	// Frame 0: encounter has player_mana [p1: 90/100]
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	panel0 := stripANSI(renderCombatPanel(sidePanelWidth, f0.ar, f0.pr, f0.er, defaultTarget(), f0.inv, ""))
	if !strings.Contains(panel0, "mana: 90/100") {
		t.Fatalf("frame 0: combat panel should display 'mana: 90/100', got:\n%s", panel0)
	}

	// Frame 2: low mana [p1: 5/100]
	f2 := traceFrameToScenarioFrame(trace.Frames[2], "p1")
	panel2 := stripANSI(renderCombatPanel(sidePanelWidth, f2.ar, f2.pr, f2.er, defaultTarget(), f2.inv, ""))
	if !strings.Contains(panel2, "mana: 5/100") {
		t.Fatalf("frame 2: combat panel should display 'mana: 5/100', got:\n%s", panel2)
	}

	// Frame 3: mana after regen [p1: 25/100]
	f3 := traceFrameToScenarioFrame(trace.Frames[3], "p1")
	panel3 := stripANSI(renderCombatPanel(sidePanelWidth, f3.ar, f3.pr, f3.er, defaultTarget(), f3.inv, ""))
	if !strings.Contains(panel3, "mana: 25/100") {
		t.Fatalf("frame 3: combat panel should display 'mana: 25/100', got:\n%s", panel3)
	}
}

// === Mob Casting E2E Tests ===

func TestMobCastStartVisibility(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_mob_casting.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Frame 0: cast_start with mob actor should show "casting" in event panel
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	panel0 := stripANSI(renderEventPanel(sidePanelWidth, f0.events, f0.pr, f0.er))
	if !strings.Contains(panel0, "casting") {
		t.Fatalf("frame 0: cast_start should show 'casting' in event panel, got:\n%s", panel0)
	}

	// Verify it classifies as cast kind
	entries0 := f0.events.recent()
	hasCast := false
	for _, e := range entries0 {
		if e.text == "casting" && e.kind == "cast" {
			hasCast = true
		}
	}
	if !hasCast {
		t.Fatalf("frame 0: cast_start should classify as 'cast' kind, got entries: %v", entries0)
	}
}

func TestMobCastDamageVisibility(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_mob_casting.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Frame 1: cast_complete then damage_applied with correct value
	f1 := traceFrameToScenarioFrame(trace.Frames[1], "p1")
	panel1 := stripANSI(renderEventPanel(sidePanelWidth, f1.events, f1.pr, f1.er))
	if !strings.Contains(panel1, "cast complete") {
		t.Fatalf("frame 1: should show 'cast complete' in event panel, got:\n%s", panel1)
	}

	// Verify damage event via entries (panel may truncate long actor>target text)
	entries1 := f1.events.recent()
	hasDamage := false
	for _, e := range entries1 {
		if strings.Contains(e.text, "hit 25") && e.kind == "combat" {
			hasDamage = true
		}
	}
	if !hasDamage {
		t.Fatalf("frame 1: should have damage event containing 'hit 25', got entries: %v", entries1)
	}

	// Verify player HP reflects the damage (75/100)
	player1 := stripANSI(renderPlayerPanel(sidePanelWidth, f1.pr, f1.inv, respawnResult{}))
	if !strings.Contains(player1, "75/100") {
		t.Fatalf("frame 1: player HP should be 75/100 after taking 25 damage, got:\n%s", player1)
	}

	// Verify event ordering: cast_complete appears before hit
	castCompleteIdx := -1
	hitIdx := -1
	for i, e := range entries1 {
		if e.text == "cast complete" && castCompleteIdx == -1 {
			castCompleteIdx = i
		}
		if strings.Contains(e.text, "hit 25") && hitIdx == -1 {
			hitIdx = i
		}
	}
	if castCompleteIdx == -1 {
		t.Fatalf("frame 1: expected 'cast complete' event")
	}
	if hitIdx == -1 {
		t.Fatalf("frame 1: expected 'hit 25' event")
	}
	if castCompleteIdx >= hitIdx {
		t.Fatalf("frame 1: 'cast complete' (index %d) should appear before 'hit 25' (index %d)", castCompleteIdx, hitIdx)
	}
}

func TestMobCastInterruptedNoDamage(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_mob_casting.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Frame 2: interrupt_applied + cast_cancelled, no damage event
	f2 := traceFrameToScenarioFrame(trace.Frames[2], "p1")
	panel2 := stripANSI(renderEventPanel(sidePanelWidth, f2.events, f2.pr, f2.er))
	if !strings.Contains(panel2, "interrupted") {
		t.Fatalf("frame 2: should show 'interrupted' in event panel, got:\n%s", panel2)
	}
	if !strings.Contains(panel2, "cast cancelled") {
		t.Fatalf("frame 2: should show 'cast cancelled' in event panel, got:\n%s", panel2)
	}

	// No damage event in this frame (interrupt prevented cast completion)
	entries2 := f2.events.recent()
	for _, e := range entries2 {
		if strings.Contains(e.text, "hit") && e.kind == "combat" {
			t.Fatalf("frame 2: should NOT have damage event after interrupt, got: %v", e)
		}
	}

	// Player HP should still be 75 (no damage taken)
	player2 := stripANSI(renderPlayerPanel(sidePanelWidth, f2.pr, f2.inv, respawnResult{}))
	if !strings.Contains(player2, "75/100") {
		t.Fatalf("frame 2: player HP should remain 75/100 (interrupt prevented damage), got:\n%s", player2)
	}

	// Verify event ordering: cast_start before interrupted before cast_cancelled
	castStartIdx := -1
	interruptedIdx := -1
	cancelledIdx := -1
	for i, e := range entries2 {
		if e.text == "casting" && castStartIdx == -1 {
			castStartIdx = i
		}
		if e.text == "interrupted" && interruptedIdx == -1 {
			interruptedIdx = i
		}
		if e.text == "cast cancelled" && cancelledIdx == -1 {
			cancelledIdx = i
		}
	}
	if castStartIdx == -1 {
		t.Fatalf("frame 2: expected 'casting' event")
	}
	if interruptedIdx == -1 {
		t.Fatalf("frame 2: expected 'interrupted' event")
	}
	if cancelledIdx == -1 {
		t.Fatalf("frame 2: expected 'cast cancelled' event")
	}
	if castStartIdx >= interruptedIdx {
		t.Fatalf("frame 2: 'casting' (index %d) should appear before 'interrupted' (index %d)", castStartIdx, interruptedIdx)
	}
	if interruptedIdx >= cancelledIdx {
		t.Fatalf("frame 2: 'interrupted' (index %d) should appear before 'cast cancelled' (index %d)", interruptedIdx, cancelledIdx)
	}
}

func TestMobCastLockoutBlockedVisibility(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_mob_casting.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Frame 3: interrupt_lockout_blocked should show "lockout blocked"
	f3 := traceFrameToScenarioFrame(trace.Frames[3], "p1")
	panel3 := stripANSI(renderEventPanel(sidePanelWidth, f3.events, f3.pr, f3.er))
	if !strings.Contains(panel3, "lockout blocked") {
		t.Fatalf("frame 3: interrupt_lockout_blocked should show 'lockout blocked' in event panel, got:\n%s", panel3)
	}

	// Verify it classifies as denial kind
	entries3 := f3.events.recent()
	hasDenial := false
	for _, e := range entries3 {
		if e.text == "lockout blocked" && e.kind == "denial" {
			hasDenial = true
		}
	}
	if !hasDenial {
		t.Fatalf("frame 3: interrupt_lockout_blocked should classify as 'denial' kind, got entries: %v", entries3)
	}

	// Player HP should still be 75 (lockout prevented cast)
	player3 := stripANSI(renderPlayerPanel(sidePanelWidth, f3.pr, f3.inv, respawnResult{}))
	if !strings.Contains(player3, "75/100") {
		t.Fatalf("frame 3: player HP should remain 75/100 (lockout blocked cast), got:\n%s", player3)
	}
}

func TestMobCastingEncounterFlow(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_mob_casting.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Verify HP progression across all frames:
	// Frame 0: 100/100 (cast started but not complete)
	// Frame 1: 75/100 (took 25 damage from completed cast)
	// Frame 2: 75/100 (interrupt prevented damage)
	// Frame 3: 75/100 (lockout blocked cast)
	// Frame 4: 50/100 (took 25 damage from second completed cast)
	expectedHP := []int{100, 75, 75, 75, 50}
	for i, expected := range expectedHP {
		sf := traceFrameToScenarioFrame(trace.Frames[i], "p1")
		if sf.inv.HPCurrent != expected {
			t.Fatalf("frame %d: expected player HP %d, got %d", i, expected, sf.inv.HPCurrent)
		}
	}

	// Verify damage only occurs on frames with completed casts (1 and 4)
	for i, frame := range trace.Frames {
		sf := traceFrameToScenarioFrame(frame, "p1")
		entries := sf.events.recent()
		hasDamage := false
		for _, e := range entries {
			if strings.Contains(e.text, "hit") && e.kind == "combat" {
				hasDamage = true
			}
		}
		if i == 1 || i == 4 {
			if !hasDamage {
				t.Fatalf("frame %d: expected damage event on completed cast frame", i)
			}
		} else {
			if hasDamage {
				t.Fatalf("frame %d: should NOT have damage event (cast not completed or blocked)", i)
			}
		}
	}

	// Frame 4: lockout expired, full cast+damage cycle works again
	f4 := traceFrameToScenarioFrame(trace.Frames[4], "p1")
	panel4 := stripANSI(renderEventPanel(sidePanelWidth, f4.events, f4.pr, f4.er))
	if !strings.Contains(panel4, "casting") {
		t.Fatalf("frame 4: should show 'casting' for new cast_start, got:\n%s", panel4)
	}
	if !strings.Contains(panel4, "cast complete") {
		t.Fatalf("frame 4: should show 'cast complete', got:\n%s", panel4)
	}
	// Verify damage event via entries (panel may truncate long actor>target text)
	entries4 := f4.events.recent()
	hasDamageF4 := false
	for _, e := range entries4 {
		if strings.Contains(e.text, "hit 25") && e.kind == "combat" {
			hasDamageF4 = true
		}
	}
	if !hasDamageF4 {
		t.Fatalf("frame 4: should have damage event containing 'hit 25', got entries: %v", entries4)
	}

	// Verify no duplicate/spam events — each frame should have reasonable event count
	for i, frame := range trace.Frames {
		sf := traceFrameToScenarioFrame(frame, "p1")
		entries := sf.events.recent()
		if len(entries) > 10 {
			t.Fatalf("frame %d: too many events (%d), possible spam", i, len(entries))
		}
	}
}

func TestE2ECombatClarity(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_combat_clarity.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Frame 0: GCD active — combat panel should contain "gcd:"
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	combat0 := stripANSI(renderCombatPanel(sidePanelWidth, f0.ar, f0.pr, f0.er, defaultTarget(), f0.inv, "mob_caster_1"))
	if !strings.Contains(combat0, "gcd:") {
		t.Fatalf("frame 0: combat panel should contain 'gcd:', got:\n%s", combat0)
	}
	// GCD remaining > 0 should show time
	if !strings.Contains(combat0, "gcd: 0.8s") {
		t.Fatalf("frame 0: combat panel should show gcd remaining '0.8s', got:\n%s", combat0)
	}

	// Frame 1: Mob starts casting — combat panel should contain mob casting info
	// Use wider panel width to avoid line wrapping in narrow panels
	f1 := traceFrameToScenarioFrame(trace.Frames[1], "p1")
	combat1 := stripANSI(renderCombatPanel(40, f1.ar, f1.pr, f1.er, defaultTarget(), f1.inv, "mob_caster_1"))
	if !strings.Contains(combat1, "mob casting: mob_firebolt") {
		t.Fatalf("frame 1: combat panel should contain 'mob casting: mob_firebolt', got:\n%s", combat1)
	}

	// Frame 1: mob_cast_start event should produce "casting" text with strong emphasis
	entries1 := f1.events.recent()
	foundCasting := false
	for _, e := range entries1 {
		if e.text == "casting" {
			foundCasting = true
			emphasis := eventEmphasis(e.text)
			if emphasis != "strong" {
				t.Fatalf("frame 1: 'casting' event should have strong emphasis, got %s", emphasis)
			}
		}
	}
	if !foundCasting {
		t.Fatalf("frame 1: events should contain 'casting', got: %v", entries1)
	}

	// Frame 2: Interrupt applied, lockout visible — combat panel should contain "lockout:"
	f2 := traceFrameToScenarioFrame(trace.Frames[2], "p1")
	combat2 := stripANSI(renderCombatPanel(sidePanelWidth, f2.ar, f2.pr, f2.er, defaultTarget(), f2.inv, "mob_caster_1"))
	if !strings.Contains(combat2, "lockout:") {
		t.Fatalf("frame 2: combat panel should contain 'lockout:', got:\n%s", combat2)
	}

	// Frame 2: "interrupted" event should have strong emphasis
	entries2 := f2.events.recent()
	foundInterrupted := false
	for _, e := range entries2 {
		if e.text == "interrupted" {
			foundInterrupted = true
			emphasis := eventEmphasis(e.text)
			if emphasis != "strong" {
				t.Fatalf("frame 2: 'interrupted' event should have strong emphasis, got %s", emphasis)
			}
		}
	}
	if !foundInterrupted {
		t.Fatalf("frame 2: events should contain 'interrupted', got: %v", entries2)
	}

	// Frame 3: GCD expired (remaining = 0), lockout still active
	f3 := traceFrameToScenarioFrame(trace.Frames[3], "p1")
	combat3 := stripANSI(renderCombatPanel(sidePanelWidth, f3.ar, f3.pr, f3.er, defaultTarget(), f3.inv, "mob_caster_1"))
	if !strings.Contains(combat3, "gcd: ready") {
		t.Fatalf("frame 3: combat panel should show 'gcd: ready' when GCD expired, got:\n%s", combat3)
	}
	if !strings.Contains(combat3, "lockout:") {
		t.Fatalf("frame 3: combat panel should still show 'lockout:' (lockout active), got:\n%s", combat3)
	}

	// Frame 4: Lockout expired — no lockout line
	f4 := traceFrameToScenarioFrame(trace.Frames[4], "p1")
	combat4 := stripANSI(renderCombatPanel(sidePanelWidth, f4.ar, f4.pr, f4.er, defaultTarget(), f4.inv, "mob_caster_1"))
	if strings.Contains(combat4, "lockout:") {
		t.Fatalf("frame 4: combat panel should NOT show lockout after expiry, got:\n%s", combat4)
	}
	if !strings.Contains(combat4, "gcd: ready") {
		t.Fatalf("frame 4: combat panel should show 'gcd: ready', got:\n%s", combat4)
	}
}

func TestE2EHealLifecycle(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_heal.json"))
	if len(trace.Frames) < 5 {
		t.Fatalf("expected at least 5 frames, got %d", len(trace.Frames))
	}

	// Frame 0: Player damaged, ready to heal — HP 75/100, mana 100/100
	f0 := traceFrameToScenarioFrame(trace.Frames[0], "p1")
	player0 := stripANSI(renderPlayerPanel(sidePanelWidth, f0.pr, f0.inv, respawnResult{}))
	if !strings.Contains(player0, "75/100") {
		t.Fatalf("frame 0: player should show HP 75/100, got:\n%s", player0)
	}

	// Frame 1: Heal cast started — event stream contains "casting"
	f1 := traceFrameToScenarioFrame(trace.Frames[1], "p1")
	panel1 := stripANSI(renderEventPanel(sidePanelWidth, f1.events, f1.pr, f1.er))
	if !strings.Contains(panel1, "casting") {
		t.Fatalf("frame 1: event panel should contain 'casting', got:\n%s", panel1)
	}

	// Frame 2: Heal completes — event panel has "heal 25", HP=100/100, mana=75/100
	f2 := traceFrameToScenarioFrame(trace.Frames[2], "p1")
	panel2 := stripANSI(renderEventPanel(sidePanelWidth, f2.events, f2.pr, f2.er))
	if !strings.Contains(panel2, "heal 25") {
		t.Fatalf("frame 2: event panel should contain 'heal 25', got:\n%s", panel2)
	}
	player2 := stripANSI(renderPlayerPanel(sidePanelWidth, f2.pr, f2.inv, respawnResult{}))
	if !strings.Contains(player2, "100/100") {
		t.Fatalf("frame 2: player should show HP 100/100 after heal, got:\n%s", player2)
	}
	combat2 := stripANSI(renderCombatPanel(sidePanelWidth, f2.ar, f2.pr, f2.er, defaultTarget(), f2.inv, "mob_caster_1"))
	if !strings.Contains(combat2, "75/100") {
		t.Fatalf("frame 2: combat panel should show mana 75/100, got:\n%s", combat2)
	}

	// Frame 3: Heal interrupted — event panel has "cast cancelled", HP unchanged at 80
	f3 := traceFrameToScenarioFrame(trace.Frames[3], "p1")
	panel3 := stripANSI(renderEventPanel(sidePanelWidth, f3.events, f3.pr, f3.er))
	if !strings.Contains(panel3, "cast cancelled") {
		t.Fatalf("frame 3: event panel should contain 'cast cancelled', got:\n%s", panel3)
	}
	player3 := stripANSI(renderPlayerPanel(sidePanelWidth, f3.pr, f3.inv, respawnResult{}))
	if !strings.Contains(player3, "80/100") {
		t.Fatalf("frame 3: player HP should remain 80/100 after cancelled heal, got:\n%s", player3)
	}

	// Frame 4: Resource denial — event panel has "deny mana"
	f4 := traceFrameToScenarioFrame(trace.Frames[4], "p1")
	panel4 := stripANSI(renderEventPanel(sidePanelWidth, f4.events, f4.pr, f4.er))
	if !strings.Contains(panel4, "deny mana") {
		t.Fatalf("frame 4: event panel should contain 'deny mana', got:\n%s", panel4)
	}
}

// --- M20260427-26 Multi-Mob Encounter Pressure and Priority Validation ---

func TestE2EMultiMobBothVisible(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_multi_mob_pressure.json"))
	if len(trace.Frames) < 6 {
		t.Fatalf("expected at least 6 frames, got %d", len(trace.Frames))
	}

	// Frame 1: encounter opened with 2 mobs
	f1 := traceFrameToScenarioFrame(trace.Frames[1], "p1")

	// Render combat panel — attack target is m1 (first mob in encounter)
	combat := stripANSI(renderCombatPanel(sidePanelWidth, f1.ar, f1.pr, f1.er, defaultTarget(), f1.inv, ""))
	// Both mob IDs should appear in the roster
	if !strings.Contains(combat, "m1") {
		t.Fatalf("frame 1: combat panel should contain mob 'm1', got:\n%s", combat)
	}
	if !strings.Contains(combat, "m2") {
		t.Fatalf("frame 1: combat panel should contain mob 'm2', got:\n%s", combat)
	}

	// Encounter panel should show 1p/2m participant counts
	focus := rosterFocus{index: 0}
	encPanel := stripANSI(renderEncounterPanel(sidePanelWidth, f1.pr, f1.er, focus, "p1"))
	if !strings.Contains(encPanel, "1p/2m") {
		t.Fatalf("frame 1: encounter panel should show '1p/2m', got:\n%s", encPanel)
	}
}

func TestE2EMultiMobCasterCasting(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_multi_mob_pressure.json"))
	if len(trace.Frames) < 6 {
		t.Fatalf("expected at least 6 frames, got %d", len(trace.Frames))
	}

	// Frame 2: caster m2 begins casting mob_firebolt
	f2 := traceFrameToScenarioFrame(trace.Frames[2], "p1")

	// Combat panel shows casting info for the targeted mob. The trace adapter
	// sets attack target to m1 (first encounter mob). To verify the caster's
	// cast bar, target m2 directly — this simulates the player tabbing to m2.
	arM2 := attackResult{State: attackStateSent, TargetID: "m2"}
	combat := stripANSI(renderCombatPanel(40, arM2, f2.pr, f2.er, defaultTarget(), f2.inv, "m2"))
	if !strings.Contains(combat, "mob casting: mob_firebolt") {
		t.Fatalf("frame 2: combat panel should contain 'mob casting: mob_firebolt', got:\n%s", combat)
	}
	if !strings.Contains(combat, "25%") {
		t.Fatalf("frame 2: combat panel should show cast progress '25%%', got:\n%s", combat)
	}

	// Event log should contain a "casting" entry from the cast_start event
	entries := f2.events.recent()
	foundCasting := false
	for _, e := range entries {
		if e.text == "casting" {
			foundCasting = true
			break
		}
	}
	if !foundCasting {
		t.Fatalf("frame 2: events should contain 'casting' for cast_start, got: %v", entries)
	}
}

func TestE2EMultiMobInterleavedEvents(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_multi_mob_pressure.json"))
	if len(trace.Frames) < 6 {
		t.Fatalf("expected at least 6 frames, got %d", len(trace.Frames))
	}

	// Frame 3: combined damage — melee m1 hits and caster m2 cast completes with damage
	f3 := traceFrameToScenarioFrame(trace.Frames[3], "p1")
	panel := stripANSI(renderEventPanel(sidePanelWidth, f3.events, f3.pr, f3.er))

	// Event panel should contain melee damage from m1
	if !strings.Contains(panel, "hit 12") {
		t.Fatalf("frame 3: event panel should contain melee 'hit 12', got:\n%s", panel)
	}
	// Event panel should contain caster damage from m2
	if !strings.Contains(panel, "hit 30") {
		t.Fatalf("frame 3: event panel should contain caster 'hit 30', got:\n%s", panel)
	}
	// Event panel should contain cast completion
	if !strings.Contains(panel, "cast complete") {
		t.Fatalf("frame 3: event panel should contain 'cast complete', got:\n%s", panel)
	}
}

func TestE2EMultiMobActorTargetClarity(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_multi_mob_pressure.json"))
	if len(trace.Frames) < 6 {
		t.Fatalf("expected at least 6 frames, got %d", len(trace.Frames))
	}

	// Frame 3: combined damage — verify actor>target labels distinguish sources
	f3 := traceFrameToScenarioFrame(trace.Frames[3], "p1")
	panel := stripANSI(renderEventPanel(sidePanelWidth, f3.events, f3.pr, f3.er))

	// Melee damage should show m1>p1 actor>target prefix
	if !strings.Contains(panel, "m1>p1 hit 12") {
		t.Fatalf("frame 3: event panel should contain 'm1>p1 hit 12' for melee damage, got:\n%s", panel)
	}
	// Caster damage should show m2>p1 actor>target prefix
	if !strings.Contains(panel, "m2>p1 hit 30") {
		t.Fatalf("frame 3: event panel should contain 'm2>p1 hit 30' for caster damage, got:\n%s", panel)
	}
}

func TestE2EMultiMobRosterEngagement(t *testing.T) {
	trace := loadScenarioTrace(t, filepath.Join("testdata", "trace_multi_mob_pressure.json"))
	if len(trace.Frames) < 6 {
		t.Fatalf("expected at least 6 frames, got %d", len(trace.Frames))
	}

	// Frame 1: both mobs targeting p1 — roster should show engagement markers
	f1 := traceFrameToScenarioFrame(trace.Frames[1], "p1")
	enc := findPlayerEncounter(f1.er.Encounters, f1.pr.ActiveEncounterID)
	if enc == nil {
		t.Fatalf("frame 1: expected encounter summary, got nil")
	}

	roster := renderCombatMobRoster(enc, f1.ar, "p1", sidePanelWidth, "")
	joined := stripANSI(strings.Join(roster, "\n"))

	// Both mobs should have the "<-" engagement marker since both target p1
	engagementCount := strings.Count(joined, "<-")
	if engagementCount < 2 {
		t.Fatalf("frame 1: roster should show '<-' engagement marker for both mobs targeting p1, found %d, got:\n%s", engagementCount, joined)
	}

	// Both mob IDs should appear in the roster
	if !strings.Contains(joined, "m1") {
		t.Fatalf("frame 1: roster should contain mob 'm1', got:\n%s", joined)
	}
	if !strings.Contains(joined, "m2") {
		t.Fatalf("frame 1: roster should contain mob 'm2', got:\n%s", joined)
	}
}
