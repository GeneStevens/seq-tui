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

	panel := renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobr, pr, focus, entries, 0, false)
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

	panel := renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobr, pr, rosterFocus{index: -1}, nil, 0, false)
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
	panel := renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobr, pr, focus, entries, 0, false)
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
	panel := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobr, pr, focus, entries, 0, false)
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

// --- M20260427-28 range/target/map legibility tests ---

func TestTargetPanelShowsMobNameAfterFocus(t *testing.T) {
	// When a mob is focused, renderProximityPanel output contains the mob's ID/name.
	mobs := []mobPosition{{MobID: "orc-alpha", Position: mobPosVec3{X: 30, Y: 0}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	entries := []rosterEntry{{kind: "mb", id: "orc-alpha", canonicalMobID: true}}
	focus := rosterFocus{index: 0}

	panel := renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobr, pr, focus, entries, 0, false)
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "orc-alpha") {
		t.Fatalf("panel should contain focused mob name 'orc-alpha', got:\n%s", stripped)
	}
}

func TestProximityPanelInRangeHasBoldStyle(t *testing.T) {
	// When backend confirms within proximity, the IN RANGE line uses bold ANSI styling.
	mobs := []mobPosition{{MobID: "orc-1", LiveMobID: "live-orc-1", Position: mobPosVec3{X: 5, Y: 0}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	entries := []rosterEntry{{kind: "mb", id: "live-orc-1", mobID: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}

	tc := targetConfirmResult{
		State:           targetConfirmOK,
		TargetID:        "live-orc-1",
		Found:           true,
		WithinProximity: true,
		Distance:        5,
	}
	panel := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobr, pr, focus, entries, 0, false)

	// Find the line containing "IN RANGE" in the raw (non-stripped) output
	found := false
	for _, line := range strings.Split(panel, "\n") {
		if strings.Contains(stripANSI(line), "IN RANGE") {
			// Bold ANSI escape is \033[1m (or part of a combined SGR sequence with ;1)
			if strings.Contains(line, "\033[1m") || strings.Contains(line, "\033[1;") || strings.Contains(line, ";1m") {
				found = true
			}
			break
		}
	}
	if !found {
		t.Fatalf("IN RANGE line should contain bold ANSI code, got raw panel:\n%s", panel)
	}
}

func TestProximityPanelOORShowsCloserHint(t *testing.T) {
	// When OOR, panel shows "g:closer" actionable hint.
	mobs := []mobPosition{{MobID: "orc-1", LiveMobID: "live-orc-1", Position: mobPosVec3{X: 100, Y: 0}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	entries := []rosterEntry{{kind: "mb", id: "live-orc-1", mobID: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}

	tc := targetConfirmResult{
		State:           targetConfirmOK,
		TargetID:        "live-orc-1",
		Found:           true,
		WithinProximity: false,
		Distance:        100,
	}
	panel := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobr, pr, focus, entries, 0, false)
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "g:closer") {
		t.Fatalf("OOR panel should show 'g:closer' hint, got:\n%s", stripped)
	}
}

func TestProximityPanelNoTargetShowsTabHint(t *testing.T) {
	// When no target is focused and no mobs are visible, panel shows "tab" hint.
	mobr := mobReadResult{State: mobReadOK, Mobs: nil, Count: 0}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}

	panel := renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobr, pr, rosterFocus{index: -1}, nil, 0, false)
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "tab") {
		t.Fatalf("no-target panel should show 'tab' hint, got:\n%s", stripped)
	}
}

func TestMapFocusedMobNotOverwrittenByAttackGlyph(t *testing.T) {
	// When attack target == focused mob, map should show 'M' (focused), not 'X' (attack).
	// This validates the fix: renderMapPanel skips attack overlay when target is focused mob.
	lines := []mapLine{
		{From: mapVec3{X: 0, Y: 0, Z: 0}, To: mapVec3{X: 100, Y: 0, Z: 100}},
	}
	bounds := computeBounds(lines)
	width, height := 30, 15
	ascii, _ := projectAndRasterize(lines, width, height)

	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   ascii,
		MapWidth:  width,
		MapHeight: height,
		Bounds:    bounds,
		Lines:     lines,
	}

	// Place mob near the player so it falls within the adaptive viewport.
	// mob.Position.Y maps to world Z for projection (same as player.Position.Y).
	mobs := []mobPosition{{MobID: "orc-1", LiveMobID: "live-orc-1", Position: mobPosVec3{X: 53, Y: 48}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 50, Y: 50}}
	entries := []rosterEntry{{kind: "mb", id: "live-orc-1", mobID: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}
	ar := attackResult{State: attackStateSent, TargetID: "live-orc-1"}

	panel := renderMapPanel(mr, mobr, pr, focus, entries, 34, 17, ar, "")
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "M") {
		t.Fatalf("focused mob should show as 'M' on map, got:\n%s", stripped)
	}
	if strings.Contains(stripped, "X") {
		t.Fatalf("attack glyph 'X' should not overwrite focused mob 'M', got:\n%s", stripped)
	}
}

func TestMapPlayerNotOverwrittenByAttackFocusOverlays(t *testing.T) {
	// Player, focused mob, and attack target all at the same cell: player '@' always wins.
	lines := []mapLine{
		{From: mapVec3{X: 0, Y: 0, Z: 0}, To: mapVec3{X: 100, Y: 0, Z: 100}},
	}
	bounds := computeBounds(lines)
	width, height := 30, 15
	ascii, _ := projectAndRasterize(lines, width, height)

	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   ascii,
		MapWidth:  width,
		MapHeight: height,
		Bounds:    bounds,
		Lines:     lines,
	}

	// Player at (50,50), focused mob at same position, attack on a different mob also at (50,50)
	mobs := []mobPosition{
		{MobID: "orc-1", LiveMobID: "live-orc-1", Position: mobPosVec3{X: 50, Y: 0, Z: 50}},
		{MobID: "orc-2", LiveMobID: "live-orc-2", Position: mobPosVec3{X: 50, Y: 0, Z: 50}},
	}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 2}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 50, Y: 50}}
	entries := []rosterEntry{
		{kind: "mb", id: "live-orc-1", mobID: "orc-1", canonicalMobID: true},
		{kind: "mb", id: "live-orc-2", mobID: "orc-2", canonicalMobID: true},
	}
	focus := rosterFocus{index: 0} // focus on orc-1
	ar := attackResult{State: attackStateSent, TargetID: "live-orc-2"}

	panel := renderMapPanel(mr, mobr, pr, focus, entries, 34, 17, ar, "")
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "@") {
		t.Fatalf("player '@' should always be visible, got:\n%s", stripped)
	}
}

func TestMapRenderIdempotent(t *testing.T) {
	// Same state rendered twice produces identical output.
	lines := []mapLine{
		{From: mapVec3{X: 0, Y: 0, Z: 0}, To: mapVec3{X: 100, Y: 0, Z: 100}},
	}
	bounds := computeBounds(lines)
	width, height := 30, 15
	ascii, _ := projectAndRasterize(lines, width, height)

	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   ascii,
		MapWidth:  width,
		MapHeight: height,
		Bounds:    bounds,
		Lines:     lines,
	}

	mobs := []mobPosition{
		{MobID: "orc-1", LiveMobID: "live-orc-1", Position: mobPosVec3{X: 30, Y: 0, Z: 30}},
		{MobID: "orc-2", LiveMobID: "live-orc-2", Position: mobPosVec3{X: 70, Y: 0, Z: 70}},
	}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 2}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 50, Y: 50}}
	entries := []rosterEntry{
		{kind: "mb", id: "live-orc-1", mobID: "orc-1", canonicalMobID: true},
	}
	focus := rosterFocus{index: 0}
	ar := attackResult{State: attackStateSent, TargetID: "live-orc-2"}

	out1 := renderMapPanel(mr, mobr, pr, focus, entries, 34, 17, ar, "")
	out2 := renderMapPanel(mr, mobr, pr, focus, entries, 34, 17, ar, "")

	if out1 != out2 {
		t.Fatalf("renderMapPanel should be idempotent.\nRender 1:\n%s\nRender 2:\n%s", out1, out2)
	}
}

func TestMapDistinctGlyphsForPlayerFocusedOther(t *testing.T) {
	// Three entities at different positions all show their correct glyphs.
	lines := []mapLine{
		{From: mapVec3{X: 0, Y: 0, Z: 0}, To: mapVec3{X: 100, Y: 0, Z: 100}},
	}
	bounds := computeBounds(lines)
	width, height := 30, 15
	ascii, _ := projectAndRasterize(lines, width, height)

	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   ascii,
		MapWidth:  width,
		MapHeight: height,
		Bounds:    bounds,
		Lines:     lines,
	}

	// All three entities near each other so they fall within the adaptive viewport.
	// Player at (50,50), focused mob slightly offset, unfocused mob also nearby.
	// mob.Position.Y maps to world Z for projection (same as player.Position.Y).
	mobs := []mobPosition{
		{MobID: "orc-focused", LiveMobID: "live-orc-focused", Position: mobPosVec3{X: 53, Y: 48}},
		{MobID: "orc-other", LiveMobID: "live-orc-other", Position: mobPosVec3{X: 47, Y: 52}},
	}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 2}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 50, Y: 50}}
	entries := []rosterEntry{
		{kind: "mb", id: "live-orc-focused", mobID: "orc-focused", canonicalMobID: true},
		{kind: "mb", id: "live-orc-other", mobID: "orc-other", canonicalMobID: true},
	}
	focus := rosterFocus{index: 0} // focus on orc-focused

	panel := renderMapPanel(mr, mobr, pr, focus, entries, 34, 17, attackResult{}, "")
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "@") {
		t.Fatalf("map should contain player '@', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "M") {
		t.Fatalf("map should contain focused mob 'M', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "m") {
		t.Fatalf("map should contain unfocused mob 'm', got:\n%s", stripped)
	}
}

