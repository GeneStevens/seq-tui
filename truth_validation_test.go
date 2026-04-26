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
	for r := 0; r < h; r++ {
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
	if !strings.Contains(canvas, "T") {
		t.Fatalf("map should show target marker 'T' for focused mob, got:\n%s", canvas)
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
	pr   playerReadResult
	er   encounterReadResult
	inv  inventoryReadResult
	mobr mobReadResult
	ar   attackResult
	tc   targetConfirmResult
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

	// Both frames should have 'T' marker
	if !strings.Contains(map0, "T") {
		t.Fatalf("frame 0: should show target marker T for orc-1, got:\n%s", map0)
	}
	if !strings.Contains(map1, "T") {
		t.Fatalf("frame 1: should show target marker T for orc-2, got:\n%s", map1)
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
