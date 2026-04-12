package main

import (
	"strings"
	"testing"
)

// --- M20260412-18 live mob identity unification tests ---

func TestDuplicateMobIDsDistinguishedByProcessID(t *testing.T) {
	// Two live mobs with same mob_id but different ProcessIDs (respawn scenario)
	player := playerPosResult{X: 0, Y: 0}
	mobs := []mobPosition{
		{ProcessID: "pid-1", MobID: "orc_abc123", Position: mobPosVec3{X: 10, Y: 0}},
		{ProcessID: "pid-2", MobID: "orc_abc123", Position: mobPosVec3{X: 50, Y: 0}},
	}
	entries := buildMobEntriesByDistance(player, mobs)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries for 2 live mobs, got %d", len(entries))
	}
	// Nearest (pid-1 at x=10) should be first
	if entries[0].id != "pid-1" {
		t.Fatalf("nearest mob should be pid-1, got: %s", entries[0].id)
	}
	if entries[1].id != "pid-2" {
		t.Fatalf("farther mob should be pid-2, got: %s", entries[1].id)
	}
	// Both should share the same mobID for attack submission
	if entries[0].mobID != "orc_abc123" || entries[1].mobID != "orc_abc123" {
		t.Fatal("both entries should have the same canonical mobID")
	}
}

func TestFindMobPositionByProcessID(t *testing.T) {
	mobs := []mobPosition{
		{ProcessID: "pid-1", MobID: "orc_abc123", Position: mobPosVec3{X: 10, Y: 20}},
		{ProcessID: "pid-2", MobID: "orc_abc123", Position: mobPosVec3{X: 50, Y: 60}},
	}
	// Lookup by ProcessID should find the exact live mob
	pos1 := findMobPosition(mobs, "pid-1")
	if pos1 == nil || pos1.X != 10 || pos1.Y != 20 {
		t.Fatalf("pid-1 should resolve to (10,20), got: %v", pos1)
	}
	pos2 := findMobPosition(mobs, "pid-2")
	if pos2 == nil || pos2.X != 50 || pos2.Y != 60 {
		t.Fatalf("pid-2 should resolve to (50,60), got: %v", pos2)
	}
}

func TestFindMobPositionFallsBackToMobID(t *testing.T) {
	// Encounter context: only mob_id available, no ProcessID match
	mobs := []mobPosition{
		{ProcessID: "pid-1", MobID: "orc_abc123", Position: mobPosVec3{X: 10, Y: 20}},
	}
	pos := findMobPosition(mobs, "orc_abc123")
	if pos == nil || pos.X != 10 || pos.Y != 20 {
		t.Fatalf("mob_id fallback should find mob, got: %v", pos)
	}
}

func TestTargetStableAcrossRefresh(t *testing.T) {
	// Simulate: target selected by ProcessID, then mob_positions refreshes
	// with the SAME ProcessID at a slightly different position.
	// Target should still resolve to the same live mob.
	mobs2 := []mobPosition{
		{ProcessID: "pid-1", MobID: "orc_abc123", Position: mobPosVec3{X: 11, Y: 21}}, // moved slightly
		{ProcessID: "pid-2", MobID: "orc_def456", Position: mobPosVec3{X: 50, Y: 60}},
	}

	// Select pid-1
	targetID := "pid-1"

	// After refresh, pid-1 still resolves correctly
	pos := findMobPosition(mobs2, targetID)
	if pos == nil || pos.X != 11 || pos.Y != 21 {
		t.Fatalf("target should track the same live mob after refresh, got: %v", pos)
	}
}

func TestTargetClearsWhenMobDies(t *testing.T) {
	// Simulate: target pid-1 dies, removed from mob_positions. A new mob
	// spawns at the same spawn point with pid-3 and same mob_id.
	// Target lookup by ProcessID should NOT find the new mob.
	mobs := []mobPosition{
		{ProcessID: "pid-3", MobID: "orc_abc123", Position: mobPosVec3{X: 80, Y: 90}}, // new spawn
		{ProcessID: "pid-2", MobID: "orc_def456", Position: mobPosVec3{X: 50, Y: 60}},
	}

	// Old target was pid-1 — should not match pid-3 even though mob_id matches
	pos := findMobPosition(mobs, "pid-1")
	if pos != nil {
		t.Fatalf("dead mob's ProcessID should not match new spawn, got: %v", pos)
	}
}

func TestAttackUsesCanonicalMobIDNotProcessID(t *testing.T) {
	// Verify that rosterEntry stores ProcessID as id but mobID for attack
	player := playerPosResult{X: 0, Y: 0}
	mobs := []mobPosition{
		{ProcessID: "pid-1", MobID: "orc_abc123", Position: mobPosVec3{X: 10, Y: 0}},
	}
	entries := buildMobEntriesByDistance(player, mobs)
	if len(entries) != 1 {
		t.Fatal("expected 1 entry")
	}
	// id should be ProcessID for live lookup
	if entries[0].id != "pid-1" {
		t.Fatalf("entry.id should be ProcessID, got: %s", entries[0].id)
	}
	// mobID should be canonical for attack submission
	if entries[0].mobID != "orc_abc123" {
		t.Fatalf("entry.mobID should be canonical mob_id, got: %s", entries[0].mobID)
	}
}

// --- M20260412-19 encounter identity continuity tests ---

func TestBuildRosterEntriesUsesProcessIDFromMobThreat(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{"orc_abc123"},
		MobThreat: []mobThreatEntry{
			{MobID: "orc_abc123", ProcessID: "pid-1", HPCurrent: 80, HPMax: 100},
		},
	}
	entries := buildRosterEntries(enc)
	var mobEntry *rosterEntry
	for i := range entries {
		if entries[i].kind == "mb" {
			mobEntry = &entries[i]
			break
		}
	}
	if mobEntry == nil {
		t.Fatal("expected a mob entry")
	}
	if mobEntry.id != "pid-1" {
		t.Fatalf("encounter mob entry.id should be ProcessID 'pid-1', got: %s", mobEntry.id)
	}
	if mobEntry.mobID != "orc_abc123" {
		t.Fatalf("encounter mob entry.mobID should be canonical 'orc_abc123', got: %s", mobEntry.mobID)
	}
}

func TestBuildRosterEntriesFallsBackToMobIDWhenNoProcessID(t *testing.T) {
	enc := &encounterSummary{
		MobIDs:    []string{"orc_abc123"},
		MobThreat: []mobThreatEntry{
			{MobID: "orc_abc123", ProcessID: "", HPCurrent: 80, HPMax: 100},
		},
	}
	entries := buildRosterEntries(enc)
	var mobEntry *rosterEntry
	for i := range entries {
		if entries[i].kind == "mb" {
			mobEntry = &entries[i]
			break
		}
	}
	if mobEntry == nil {
		t.Fatal("expected a mob entry")
	}
	if mobEntry.id != "orc_abc123" {
		t.Fatalf("should fall back to mob_id when no ProcessID, got: %s", mobEntry.id)
	}
}

func TestEncounterTargetContinuityWithPreEncounterSelection(t *testing.T) {
	// Pre-encounter: target selected by ProcessID "pid-1"
	// Encounter opens: mob_threat carries process_id "pid-1" for the same mob
	// Roster entry should still have id="pid-1" for continuity

	// Pre-encounter selection
	preEncEntries := buildMobEntriesByDistance(
		playerPosResult{X: 0, Y: 0},
		[]mobPosition{{ProcessID: "pid-1", MobID: "orc_abc123", Position: mobPosVec3{X: 10, Y: 0}}},
	)
	if preEncEntries[0].id != "pid-1" {
		t.Fatalf("pre-encounter entry.id should be ProcessID, got: %s", preEncEntries[0].id)
	}

	// Encounter opens — same ProcessID propagated
	encEntries := buildRosterEntries(&encounterSummary{
		MobIDs: []string{"orc_abc123"},
		MobThreat: []mobThreatEntry{
			{MobID: "orc_abc123", ProcessID: "pid-1", HPCurrent: 100, HPMax: 100},
		},
	})
	var mobEntry *rosterEntry
	for i := range encEntries {
		if encEntries[i].kind == "mb" {
			mobEntry = &encEntries[i]
			break
		}
	}
	if mobEntry == nil {
		t.Fatal("expected a mob entry in encounter roster")
	}
	// Same ProcessID should survive the encounter transition
	if mobEntry.id != "pid-1" {
		t.Fatalf("encounter entry.id should match pre-encounter ProcessID 'pid-1', got: %s", mobEntry.id)
	}
}

func TestMobThreatEntryCarriesProcessID(t *testing.T) {
	// Verify that mobThreatEntry struct can carry ProcessID
	mt := mobThreatEntry{
		MobID:     "orc_abc123",
		ProcessID: "pid-1",
		HPCurrent: 80,
		HPMax:     100,
	}
	if mt.ProcessID != "pid-1" {
		t.Fatalf("expected ProcessID 'pid-1', got: %s", mt.ProcessID)
	}
}

// --- M20260412-20 sticky target identity persistence tests ---

func TestMergeRosterIdentityPreservesExistingProcessID(t *testing.T) {
	oldEntries := []rosterEntry{
		{kind: "pc", id: "p1"},
		{kind: "mb", id: "pid-1", mobID: "orc_abc123", canonicalMobID: true},
	}
	// New encounter refresh resolves a DIFFERENT process_id for same mob_id
	newEntries := []rosterEntry{
		{kind: "pc", id: "p1"},
		{kind: "mb", id: "pid-2", mobID: "orc_abc123", canonicalMobID: true},
	}
	merged := mergeRosterIdentity(oldEntries, newEntries)

	var mobEntry *rosterEntry
	for i := range merged {
		if merged[i].kind == "mb" {
			mobEntry = &merged[i]
			break
		}
	}
	if mobEntry == nil {
		t.Fatal("expected mob entry")
	}
	// Should preserve the ORIGINAL ProcessID, not the newly resolved one
	if mobEntry.id != "pid-1" {
		t.Fatalf("mergeRosterIdentity should preserve existing ProcessID 'pid-1', got: %s", mobEntry.id)
	}
	if mobEntry.mobID != "orc_abc123" {
		t.Fatalf("mobID should remain canonical, got: %s", mobEntry.mobID)
	}
}

func TestMergeRosterIdentityAllowsNewMobs(t *testing.T) {
	oldEntries := []rosterEntry{
		{kind: "mb", id: "pid-1", mobID: "orc_abc123", canonicalMobID: true},
	}
	// New encounter adds a second mob
	newEntries := []rosterEntry{
		{kind: "mb", id: "pid-1", mobID: "orc_abc123", canonicalMobID: true},
		{kind: "mb", id: "pid-3", mobID: "orc_def456", canonicalMobID: true},
	}
	merged := mergeRosterIdentity(oldEntries, newEntries)
	if len(merged) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(merged))
	}
	if merged[0].id != "pid-1" {
		t.Fatalf("existing mob should keep pid-1, got: %s", merged[0].id)
	}
	if merged[1].id != "pid-3" {
		t.Fatalf("new mob should use its own ProcessID, got: %s", merged[1].id)
	}
}

func TestStickyTargetSurvivesEncounterRefresh(t *testing.T) {
	// Simulates the encounter refresh path: select target, then
	// reconcileFocus against refreshed roster. ProcessID must persist.
	oldEntries := []rosterEntry{
		{kind: "pc", id: "p1"},
		{kind: "mb", id: "pid-1", mobID: "orc_abc123", canonicalMobID: true},
	}
	focus := rosterFocus{index: 1} // focused on the mob

	// Encounter refresh: mob_threat returns different process_id for same mob_id
	newRaw := []rosterEntry{
		{kind: "pc", id: "p1"},
		{kind: "mb", id: "pid-99", mobID: "orc_abc123", canonicalMobID: true},
	}
	merged := mergeRosterIdentity(oldEntries, newRaw)
	newFocus := reconcileFocus(focus, oldEntries, merged)

	fe := focusedEntry(newFocus, merged)
	if fe == nil {
		t.Fatal("focus should survive encounter refresh")
	}
	if fe.id != "pid-1" {
		t.Fatalf("sticky target should remain pid-1 after refresh, got: %s", fe.id)
	}
}

func TestTargetClearedWhenProcessIDDisappears(t *testing.T) {
	// Target pid-1 disappears from mob_positions.
	// findMobPosition("pid-1") returns nil → target should be cleared.
	mobs := []mobPosition{
		{ProcessID: "pid-2", MobID: "orc_def456", Position: mobPosVec3{X: 50, Y: 0}},
	}
	// pid-1 is not in this list
	pos := findMobPosition(mobs, "pid-1")
	if pos != nil {
		t.Fatal("pid-1 should not be found after disappearing")
	}
}

func TestTabExplicitlyChangesTarget(t *testing.T) {
	// When user presses Tab, the target should change to the new selection.
	// This is NOT a refresh — it's an explicit user action.
	player := playerPosResult{X: 0, Y: 0}
	mobs := []mobPosition{
		{ProcessID: "pid-1", MobID: "orc_abc", Position: mobPosVec3{X: 5, Y: 0}},
		{ProcessID: "pid-2", MobID: "orc_def", Position: mobPosVec3{X: 10, Y: 0}},
	}
	entries := buildMobEntriesByDistance(player, mobs)
	focus := moveFocusDown(rosterFocus{index: -1}, len(entries)) // first tab → nearest
	fe := focusedEntry(focus, entries)
	if fe == nil || fe.id != "pid-1" {
		t.Fatalf("first tab should select pid-1 (nearest), got: %v", fe)
	}
	focus = moveFocusDown(focus, len(entries)) // second tab → next
	fe = focusedEntry(focus, entries)
	if fe == nil || fe.id != "pid-2" {
		t.Fatalf("second tab should select pid-2, got: %v", fe)
	}
}

func TestOverlayFocusedMobByProcessID(t *testing.T) {
	// Two mobs with same MobID — overlay should match the correct one by ProcessID
	mapText := ".....\n.....\n....."
	mobs := []mobPosition{
		{ProcessID: "pid-1", MobID: "orc_abc123", Position: mobPosVec3{X: 25, Y: 75}},
		{ProcessID: "pid-2", MobID: "orc_abc123", Position: mobPosVec3{X: 75, Y: 25}},
	}
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}

	result1 := overlayFocusedMob(mapText, mobs, "pid-1", bounds, 5, 3)
	result2 := overlayFocusedMob(mapText, mobs, "pid-2", bounds, 5, 3)

	if !strings.Contains(result1, "T") {
		t.Fatal("overlay by ProcessID pid-1 should place T glyph")
	}
	if !strings.Contains(result2, "T") {
		t.Fatal("overlay by ProcessID pid-2 should place T glyph")
	}
	// They should be at different positions
	if result1 == result2 {
		t.Fatal("overlay for different PIDs should be at different positions")
	}
}
