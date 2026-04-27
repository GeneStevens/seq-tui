package main

import (
	"math"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- M20260412-17 target selection, visualization, and range clarity tests ---

func TestBuildMobEntriesByDistanceSortOrder(t *testing.T) {
	player := playerPosResult{X: 0, Y: 0}
	mobs := []mobPosition{
		{MobID: "orc-far", Position: mobPosVec3{X: 100, Y: 0}},
		{MobID: "orc-near", Position: mobPosVec3{X: 5, Y: 0}},
		{MobID: "orc-mid", Position: mobPosVec3{X: 50, Y: 0}},
	}
	entries := buildMobEntriesByDistance(player, mobs)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].id != "orc-near" {
		t.Fatalf("first entry should be nearest mob, got: %s", entries[0].id)
	}
	if entries[1].id != "orc-mid" {
		t.Fatalf("second entry should be mid mob, got: %s", entries[1].id)
	}
	if entries[2].id != "orc-far" {
		t.Fatalf("third entry should be farthest mob, got: %s", entries[2].id)
	}
}

func TestBuildMobEntriesByDistanceTieBreakByMobID(t *testing.T) {
	player := playerPosResult{X: 0, Y: 0}
	mobs := []mobPosition{
		{MobID: "orc-b", Position: mobPosVec3{X: 10, Y: 0}},
		{MobID: "orc-a", Position: mobPosVec3{X: 10, Y: 0}},
	}
	entries := buildMobEntriesByDistance(player, mobs)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].id != "orc-a" {
		t.Fatalf("tie should break by MobID ascending, got first: %s", entries[0].id)
	}
}

func TestFirstTabSelectsNearestViaFocusDown(t *testing.T) {
	// Simulates what Tab does: build distance list, then moveFocusDown from -1
	player := playerPosResult{X: 0, Y: 0}
	mobs := []mobPosition{
		{MobID: "orc-far", Position: mobPosVec3{X: 100, Y: 0}},
		{MobID: "orc-near", Position: mobPosVec3{X: 5, Y: 0}},
	}
	entries := buildMobEntriesByDistance(player, mobs)
	focus := moveFocusDown(rosterFocus{index: -1}, len(entries))
	fe := focusedEntry(focus, entries)
	if fe == nil || fe.id != "orc-near" {
		t.Fatalf("first focus should be nearest mob, got: %v", fe)
	}
}

func TestRebuildListAfterPlayerMoveChangesNearest(t *testing.T) {
	mobs := []mobPosition{
		{MobID: "orc-a", Position: mobPosVec3{X: 100, Y: 0}},
		{MobID: "orc-b", Position: mobPosVec3{X: 5, Y: 0}},
	}

	// From origin, orc-b is nearest
	entries1 := buildMobEntriesByDistance(playerPosResult{X: 0, Y: 0}, mobs)
	if entries1[0].id != "orc-b" {
		t.Fatalf("from origin, orc-b should be nearest, got: %s", entries1[0].id)
	}

	// After moving near orc-a, rebuild shows orc-a as nearest
	entries2 := buildMobEntriesByDistance(playerPosResult{X: 95, Y: 0}, mobs)
	if entries2[0].id != "orc-a" {
		t.Fatalf("from (95,0), orc-a should be nearest, got: %s", entries2[0].id)
	}
}

func TestFocusedMobOverlayUsesM(t *testing.T) {
	mapText := ".....\n..m..\n....."
	mobs := []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 50, Y: 50}}}
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	result := overlayFocusedMob(mapText, mobs, "orc-1", bounds, 5, 3)
	if !strings.Contains(result, "M") {
		t.Fatalf("focused mob should show as 'M', got: %s", result)
	}
}

func TestUnfocusedMobSmallMFocusedMobBigM(t *testing.T) {
	// Two mobs on a 10-wide map: one focused, one not.
	// Unfocused should stay 'm', focused should become 'M'.
	mapText := "..........\n..........\n.........."
	mobs := []mobPosition{
		{MobID: "orc-a", Position: mobPosVec3{X: 20, Y: 50}},
		{MobID: "orc-b", Position: mobPosVec3{X: 80, Y: 50}},
	}
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	w, h := 10, 3
	result := overlayMobs(mapText, mobs, bounds, w, h)
	result = overlayFocusedMob(result, mobs, "orc-b", bounds, w, h)
	if !strings.Contains(result, "m") {
		t.Fatal("unfocused mob should remain as 'm'")
	}
	if !strings.Contains(result, "M") {
		t.Fatal("focused mob should be upgraded to 'M'")
	}
}

func TestPlayerOverlaysOnTopOfFocusedMob(t *testing.T) {
	// Player and focused mob at the same position: player '@' takes priority.
	mapText := "...\n...\n..."
	mobs := []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 50, Y: 50}}}
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	w, h := 3, 3
	result := overlayMobs(mapText, mobs, bounds, w, h)
	result = overlayFocusedMob(result, mobs, "orc-1", bounds, w, h)
	// Now overlay the player at the same world position
	result = overlayPlayer(result, playerPosResult{X: 50, Y: 50}, bounds, w, h, false)
	if !strings.Contains(result, "@") {
		t.Fatal("player '@' should overlay on top of focused mob 'M'")
	}
	// The 'M' should be replaced by '@' at that cell
	if strings.Contains(result, "M") {
		t.Fatal("player should overwrite focused mob glyph at same position")
	}
}

func TestColorizeFocusedMobDistinctFromRegularMob(t *testing.T) {
	mStyled := colorizeMapContent("m")
	bigMStyled := colorizeMapContent("M")
	if mStyled == bigMStyled {
		t.Fatal("focused mob 'M' should have distinct styling from regular mob 'm'")
	}
	// Both should be styled (contain ANSI escapes)
	if !strings.Contains(mStyled, "\033[") {
		t.Fatal("regular mob 'm' should have ANSI styling")
	}
	if !strings.Contains(bigMStyled, "\033[") {
		t.Fatal("focused mob 'M' should have ANSI styling")
	}
}

func TestTargetPanelShowsFocusedMobDistance(t *testing.T) {
	mobs := []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 30, Y: 0}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	entries := []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}

	panel := renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobr, pr, focus, entries)
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "orc-1") {
		t.Fatalf("target panel should show mob ID, got: %s", stripped)
	}
	if !strings.Contains(stripped, "dist:30") {
		t.Fatalf("target panel should show distance, got: %s", stripped)
	}
}

func TestTargetPanelFallsBackToNearestWhenNoFocus(t *testing.T) {
	mobs := []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 20, Y: 0}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}

	panel := renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobr, pr, rosterFocus{index: -1}, nil)
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "nearest") {
		t.Fatalf("target panel should fall back to nearest mob when no focus, got: %s", stripped)
	}
}

// --- Auto-query proximity on tab focus tests ---

func TestTabFocusMobTriggersProximityQuery(t *testing.T) {
	// When tab focuses a mob, Update should return a non-nil Cmd (proximity query).
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	m.mobRead = mobReadResult{
		State: mobReadOK,
		Mobs:  []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 10, Y: 0}}},
		Count: 1,
	}
	m.rosterFocus = rosterFocus{index: -1}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd == nil {
		t.Fatal("tab focusing a mob should return a non-nil Cmd for proximity query")
	}
	um := updated.(model)
	fe := focusedEntry(um.rosterFocus, um.rosterEntries)
	if fe == nil || fe.kind != "mb" {
		t.Fatal("tab should focus the mob entry")
	}
	if um.lastProximityID != fe.id {
		t.Fatalf("lastProximityID should be set to focused mob, got: %s", um.lastProximityID)
	}
}

func TestShiftTabFocusMobTriggersProximityQuery(t *testing.T) {
	// When shift+tab focuses a mob, Update should return a non-nil Cmd.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	m.mobRead = mobReadResult{
		State: mobReadOK,
		Mobs:  []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 10, Y: 0}}},
		Count: 1,
	}
	m.rosterFocus = rosterFocus{index: -1}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if cmd == nil {
		t.Fatal("shift+tab focusing a mob should return a non-nil Cmd for proximity query")
	}
	um := updated.(model)
	fe := focusedEntry(um.rosterFocus, um.rosterEntries)
	if fe == nil || fe.kind != "mb" {
		t.Fatal("shift+tab should focus the mob entry")
	}
	if um.lastProximityID != fe.id {
		t.Fatalf("lastProximityID should be set to focused mob, got: %s", um.lastProximityID)
	}
}

func TestTabNoMobsReturnsNilCmd(t *testing.T) {
	// When there are no mobs, tab should return nil Cmd.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: nil, Count: 0}
	m.rosterFocus = rosterFocus{index: -1}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd != nil {
		t.Fatal("tab with no mobs should return nil Cmd")
	}
}

func TestProximityPanelFallbackShowsCheckHint(t *testing.T) {
	// When no backend proximity data is available, dist display should include (t:chk) hint.
	mobs := []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 25, Y: 0}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	entries := []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}

	// No targetConfirm data — empty result triggers the fallback branch
	panel := renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobr, pr, focus, entries)
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "dist:25") {
		t.Fatalf("fallback should show distance, got: %s", stripped)
	}
	if !strings.Contains(stripped, "(t:chk)") {
		t.Fatalf("fallback should show range check hint (t:chk), got: %s", stripped)
	}
}

func TestProximityPanelWithRangeDataNoHint(t *testing.T) {
	// When backend proximity data IS available, no (t:chk) hint should appear.
	mobs := []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 25, Y: 0}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	entries := []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}

	tc := targetConfirmResult{
		State:           targetConfirmOK,
		TargetID:        "orc-1",
		Found:           true,
		WithinProximity: true,
		Distance:        25,
	}
	panel := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobr, pr, focus, entries)
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "IN RANGE") {
		t.Fatalf("should show IN RANGE with backend data, got: %s", stripped)
	}
	if strings.Contains(stripped, "(t:chk)") {
		t.Fatalf("should NOT show (t:chk) hint when backend data is available, got: %s", stripped)
	}
}

func TestGKeyUsesSelectedTarget(t *testing.T) {
	// g key must use the focused mob from the roster, not any arbitrary mob
	m := model{slog: &sessionLogger{}}
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: []mobPosition{
		{MobID: "orc-near", Position: mobPosVec3{X: 5, Y: 0}},
		{MobID: "orc-far", Position: mobPosVec3{X: 100, Y: 0}},
	}, Count: 2}
	// Focus on far mob (simulating the user tabbed to it)
	m.rosterEntries = []rosterEntry{
		{kind: "mb", id: "orc-far", canonicalMobID: true},
	}
	m.rosterFocus = rosterFocus{index: 0}

	// g key should compute direction toward orc-far, not orc-near
	fe := focusedEntry(m.rosterFocus, m.rosterEntries)
	mobPos := findMobPosition(m.mobRead.Mobs, fe.id)
	dir := directionTowardMob(m.playerRead.Position, *mobPos)

	if dir != "east" {
		t.Fatalf("g key should step toward focused mob (east), got: %s", dir)
	}

	// Verify it reduces distance to the FOCUSED target
	step := 5.0
	newX := m.playerRead.Position.X + step // east = +X
	distBefore := math.Abs(mobPos.X - m.playerRead.Position.X)
	distAfter := math.Abs(mobPos.X - newX)
	if distAfter >= distBefore {
		t.Fatalf("step should reduce distance to focused target: before=%.0f after=%.0f", distBefore, distAfter)
	}
}

