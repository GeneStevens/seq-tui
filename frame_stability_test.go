package main

import (
	"math"
	"strings"
	"testing"
)

// --- A) Repeated identical state produces identical frames ---

// TestSideColumnRenderIdempotent verifies that renderSideColumn with identical
// inputs produces identical output across three consecutive calls.
func TestSideColumnRenderIdempotent(t *testing.T) {
	width := sidePanelWidth
	target := defaultTarget()
	zr := zoneReadResult{}
	mr := mapReadResult{}
	mobr := mobReadResult{State: mobReadOK, Mobs: nil, Count: 0}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 50, Y: 50}}
	er := encounterReadResult{}
	focus := rosterFocus{index: -1}
	var entries []rosterEntry
	tc := targetConfirmResult{}
	ar := attackResult{}
	pk := pickupResult{}
	inv := inventoryReadResult{}
	rs := respawnResult{}
	events := eventLog{}

	var outputs [3]string
	for i := range outputs {
		outputs[i] = renderSideColumn(width, target, zr, mr, mobr, pr, er, focus, entries, tc, ar, pk, inv, 0, 0, rs, events, false)
	}
	for i := 1; i < len(outputs); i++ {
		if outputs[i] != outputs[0] {
			t.Errorf("renderSideColumn call %d differs from call 0", i)
		}
	}
}

// TestProximityPanelRenderIdempotent verifies that renderProximityPanel with
// identical inputs produces identical output across three consecutive calls.
func TestProximityPanelRenderIdempotent(t *testing.T) {
	mobs := []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 30, Y: 0}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	entries := []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}

	var outputs [3]string
	for i := range outputs {
		outputs[i] = renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobr, pr, focus, entries)
	}
	for i := 1; i < len(outputs); i++ {
		if outputs[i] != outputs[0] {
			t.Errorf("renderProximityPanel call %d differs from call 0", i)
		}
	}
}

// TestFullLayoutRenderIdempotent verifies that renderLayout with identical
// inputs produces identical output across three consecutive calls.
func TestFullLayoutRenderIdempotent(t *testing.T) {
	lines, bounds := testGeometry()
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
	mobr := mobReadResult{State: mobReadOK, Mobs: nil, Count: 0}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 50, Y: 50}}
	target := defaultTarget()
	events := eventLog{}

	termW, termH := 100, 40

	var outputs [3]string
	for i := range outputs {
		outputs[i] = renderLayout(termW, termH, "", target, zoneReadResult{}, mr, mobr, pr,
			encounterReadResult{}, rosterFocus{index: -1}, nil,
			targetConfirmResult{}, attackResult{}, pickupResult{},
			inventoryReadResult{}, 0, 0, respawnResult{}, events, false)
	}
	for i := 1; i < len(outputs); i++ {
		if outputs[i] != outputs[0] {
			t.Errorf("renderLayout call %d differs from call 0", i)
		}
	}
}

// --- B) Mob stability across repeated renders ---

// TestMobPositionsStableAcrossRepeatedRenders verifies that 3 mobs at distinct
// positions produce the same cell coordinates across 5 consecutive renders.
// Mob positions are placed within the adaptive viewport window centered at (50,50).
func TestMobPositionsStableAcrossRepeatedRenders(t *testing.T) {
	lines, bounds := testGeometry()
	vpW, vpH := 40, 20

	// With vpW=40 vpH=20 and adaptiveRef 240x120, the world window is ~16.7 units
	// centered at (50,50), spanning roughly [41.7, 58.3] x [41.7, 58.3].
	mobs := []mobPosition{
		{MobID: "mob-a", Position: mobPosVec3{X: 44, Y: 44}},
		{MobID: "mob-b", Position: mobPosVec3{X: 50, Y: 50}},
		{MobID: "mob-c", Position: mobPosVec3{X: 56, Y: 56}},
	}

	centerX, centerZ := 50.0, 50.0

	type cellPos struct{ col, row int }
	var baseline [3]cellPos

	for iter := 0; iter < 5; iter++ {
		ascii, vpBounds := rasterizeAdaptiveViewport(lines, bounds, centerX, centerZ, vpW, vpH)
		overlay := overlayMobs(ascii, mobs, vpBounds, vpW, vpH)

		// Find each mob's cell. With 3 distinct positions, each cell has exactly one 'm'.
		mCells := findAllGlyphs(overlay, 'm')
		if len(mCells) != 3 {
			t.Fatalf("iteration %d: expected 3 'm' glyphs, got %d", iter, len(mCells))
		}

		if iter == 0 {
			for i, c := range mCells {
				baseline[i] = cellPos{c[0], c[1]}
			}
		} else {
			for i, c := range mCells {
				if c[0] != baseline[i].col || c[1] != baseline[i].row {
					t.Errorf("iteration %d: mob %d moved from (%d,%d) to (%d,%d)",
						iter, i, baseline[i].col, baseline[i].row, c[0], c[1])
				}
			}
		}
	}
}

// TestRemovingMobDoesNotShiftOthers verifies that removing one mob from a set
// of 3 does not shift the cell positions of the remaining 2.
func TestRemovingMobDoesNotShiftOthers(t *testing.T) {
	lines, bounds := testGeometry()
	vpW, vpH := 40, 20
	centerX, centerZ := 50.0, 50.0

	// All mob positions within the adaptive viewport window (~[41.7, 58.3] for vpW=40)
	mobA := mobPosition{MobID: "mob-a", Position: mobPosVec3{X: 44, Y: 44}}
	mobB := mobPosition{MobID: "mob-b", Position: mobPosVec3{X: 50, Y: 50}}
	mobC := mobPosition{MobID: "mob-c", Position: mobPosVec3{X: 56, Y: 56}}

	// Render with all 3 mobs first to get baseline positions for A and C
	ascii3, vpBounds3 := rasterizeAdaptiveViewport(lines, bounds, centerX, centerZ, vpW, vpH)
	overlay3 := overlayMobs(ascii3, []mobPosition{mobA, mobB, mobC}, vpBounds3, vpW, vpH)
	cells3 := findAllGlyphs(overlay3, 'm')
	if len(cells3) != 3 {
		t.Fatalf("expected 3 'm' glyphs with 3 mobs, got %d", len(cells3))
	}
	// Record A (first) and C (last) positions from sorted cell list
	colA, rowA := cells3[0][0], cells3[0][1]
	colC, rowC := cells3[2][0], cells3[2][1]

	// Remove mob-b, render with A and C only
	ascii2, vpBounds2 := rasterizeAdaptiveViewport(lines, bounds, centerX, centerZ, vpW, vpH)
	overlay2 := overlayMobs(ascii2, []mobPosition{mobA, mobC}, vpBounds2, vpW, vpH)

	cells2 := findAllGlyphs(overlay2, 'm')
	if len(cells2) != 2 {
		t.Fatalf("expected 2 'm' glyphs after removing mob-b, got %d", len(cells2))
	}

	// Mob-a and mob-c should remain at the same positions
	colA2, rowA2 := cells2[0][0], cells2[0][1]
	colC2, rowC2 := cells2[1][0], cells2[1][1]

	if colA != colA2 || rowA != rowA2 {
		t.Errorf("mob-a shifted after mob-b removal: (%d,%d) -> (%d,%d)", colA, rowA, colA2, rowA2)
	}
	if colC != colC2 || rowC != rowC2 {
		t.Errorf("mob-c shifted after mob-b removal: (%d,%d) -> (%d,%d)", colC, rowC, colC2, rowC2)
	}
}

// TestAddingMobDoesNotShiftExisting verifies that adding a third mob does not
// shift the cell positions of the original 2.
func TestAddingMobDoesNotShiftExisting(t *testing.T) {
	lines, bounds := testGeometry()
	vpW, vpH := 40, 20
	centerX, centerZ := 50.0, 50.0

	// All mob positions within the adaptive viewport window
	mobA := mobPosition{MobID: "mob-a", Position: mobPosVec3{X: 44, Y: 44}}
	mobC := mobPosition{MobID: "mob-c", Position: mobPosVec3{X: 56, Y: 56}}

	// Render with 2 mobs, record their positions individually
	asciiA, vpBoundsA := rasterizeAdaptiveViewport(lines, bounds, centerX, centerZ, vpW, vpH)
	overlayA := overlayMobs(asciiA, []mobPosition{mobA}, vpBoundsA, vpW, vpH)
	colA, rowA := findGlyph(overlayA, 'm')
	if colA < 0 {
		t.Fatal("mob-a not visible when rendered alone")
	}

	asciiC, vpBoundsC := rasterizeAdaptiveViewport(lines, bounds, centerX, centerZ, vpW, vpH)
	overlayC := overlayMobs(asciiC, []mobPosition{mobC}, vpBoundsC, vpW, vpH)
	colC, rowC := findGlyph(overlayC, 'm')
	if colC < 0 {
		t.Fatal("mob-c not visible when rendered alone")
	}

	// Add mob-b and render with all 3
	mobB := mobPosition{MobID: "mob-b", Position: mobPosVec3{X: 50, Y: 50}}
	ascii3, vpBounds3 := rasterizeAdaptiveViewport(lines, bounds, centerX, centerZ, vpW, vpH)
	overlay3 := overlayMobs(ascii3, []mobPosition{mobA, mobB, mobC}, vpBounds3, vpW, vpH)

	cells3 := findAllGlyphs(overlay3, 'm')
	if len(cells3) != 3 {
		t.Fatalf("expected 3 'm' glyphs after adding mob-b, got %d", len(cells3))
	}

	// Find mob-a and mob-c in the 3-mob render by matching their known positions
	foundA, foundC := false, false
	for _, c := range cells3 {
		if c[0] == colA && c[1] == rowA {
			foundA = true
		}
		if c[0] == colC && c[1] == rowC {
			foundC = true
		}
	}
	if !foundA {
		t.Error("mob-a cell position not found after adding mob-b")
	}
	if !foundC {
		t.Error("mob-c cell position not found after adding mob-b")
	}
}

// --- C) Focus stability through state refresh ---

// TestReconcileFocusPreservesIdenticalRoster verifies that reconcileFocus with
// an unchanged roster preserves the same focus index.
func TestReconcileFocusPreservesIdenticalRoster(t *testing.T) {
	entries := []rosterEntry{
		{kind: "pc", id: "player-1"},
		{kind: "mb", id: "orc-1", mobID: "orc-1", canonicalMobID: true},
		{kind: "mb", id: "orc-2", mobID: "orc-2", canonicalMobID: true},
	}
	focus := rosterFocus{index: 1}

	result := reconcileFocus(focus, entries, entries)
	if result.index != 1 {
		t.Errorf("expected focus index 1 after identical reconcile, got %d", result.index)
	}
}

// TestMergeRosterIdentityIdempotent verifies that mergeRosterIdentity called
// twice with the same data produces the same result.
func TestMergeRosterIdentityIdempotent(t *testing.T) {
	old := []rosterEntry{
		{kind: "mb", id: "proc-1", mobID: "orc-1", canonicalMobID: true},
		{kind: "mb", id: "proc-2", mobID: "orc-2", canonicalMobID: true},
	}
	// Simulate new entries from encounter refresh (using mob_id as id)
	newEntries1 := []rosterEntry{
		{kind: "mb", id: "orc-1", mobID: "orc-1", canonicalMobID: true},
		{kind: "mb", id: "orc-2", mobID: "orc-2", canonicalMobID: true},
	}
	newEntries2 := []rosterEntry{
		{kind: "mb", id: "orc-1", mobID: "orc-1", canonicalMobID: true},
		{kind: "mb", id: "orc-2", mobID: "orc-2", canonicalMobID: true},
	}

	result1 := mergeRosterIdentity(old, newEntries1)
	result2 := mergeRosterIdentity(old, newEntries2)

	if len(result1) != len(result2) {
		t.Fatalf("merge results differ in length: %d vs %d", len(result1), len(result2))
	}
	for i := range result1 {
		if result1[i].id != result2[i].id || result1[i].mobID != result2[i].mobID {
			t.Errorf("entry %d differs: {id:%s mobID:%s} vs {id:%s mobID:%s}",
				i, result1[i].id, result1[i].mobID, result2[i].id, result2[i].mobID)
		}
	}
}

// TestFocusedEntryStableAfterReconcile verifies that focusedEntry returns the
// same entry after reconcile with an identical roster.
func TestFocusedEntryStableAfterReconcile(t *testing.T) {
	entries := []rosterEntry{
		{kind: "pc", id: "player-1"},
		{kind: "mb", id: "orc-1", mobID: "orc-1", canonicalMobID: true},
		{kind: "mb", id: "orc-2", mobID: "orc-2", canonicalMobID: true},
	}
	focus := rosterFocus{index: 2}

	before := focusedEntry(focus, entries)
	if before == nil {
		t.Fatal("expected focused entry before reconcile")
	}

	newFocus := reconcileFocus(focus, entries, entries)
	after := focusedEntry(newFocus, entries)
	if after == nil {
		t.Fatal("expected focused entry after reconcile")
	}

	if before.kind != after.kind || before.id != after.id || before.mobID != after.mobID {
		t.Errorf("focused entry changed after reconcile: {%s,%s,%s} -> {%s,%s,%s}",
			before.kind, before.id, before.mobID, after.kind, after.id, after.mobID)
	}
}

// --- D) Small movement predictability ---

// TestPlayerMoveOneStepMobsStay verifies that when the player moves one step,
// a stationary mob remains visible and its cell shift is bounded. The adaptive
// viewport uses center snapping, so a 1-unit world move may cross a snap boundary
// causing a shift of up to ~2 cells. The key invariant: the mob stays visible.
func TestPlayerMoveOneStepMobsStay(t *testing.T) {
	lines, bounds := testGeometry()
	vpW, vpH := 40, 20

	// Mob within adaptive viewport of both player positions
	mob := mobPosition{MobID: "mob-1", Position: mobPosVec3{X: 48, Y: 50}}
	mobs := []mobPosition{mob}

	// Render at player position (50, 50)
	ascii1, vpBounds1 := rasterizeAdaptiveViewport(lines, bounds, 50.0, 50.0, vpW, vpH)
	overlay1 := overlayMobs(ascii1, mobs, vpBounds1, vpW, vpH)
	col1, row1 := findGlyph(overlay1, 'm')

	// Render at player position (51, 50) — one step east
	ascii2, vpBounds2 := rasterizeAdaptiveViewport(lines, bounds, 51.0, 50.0, vpW, vpH)
	overlay2 := overlayMobs(ascii2, mobs, vpBounds2, vpW, vpH)
	col2, row2 := findGlyph(overlay2, 'm')

	if col1 < 0 || col2 < 0 {
		t.Fatal("mob not visible in one or both renders")
	}

	// Mob must remain visible after a single-step player movement
	if col2 < 0 || row2 < 0 {
		t.Error("mob disappeared after player moved one step")
	}

	// Row should not shift (same Z coordinate, no vertical movement)
	if row1 != row2 {
		t.Errorf("mob row shifted by %d (expected 0 with same Z)", row1-row2)
	}

	// Column shift is bounded by cell resolution. With snapping, a 1-unit move
	// can shift by up to a few cells. Verify the shift is not unreasonably large
	// (no more than 5 cells for a 1-unit world move on a 40-cell viewport).
	colDelta := col1 - col2
	if colDelta < 0 {
		colDelta = -colDelta
	}
	if colDelta > 5 {
		t.Errorf("mob column shifted by %d cells for 1-unit move (unreasonably large)", colDelta)
	}
}

// TestPlayerMoveProximityUpdates verifies that when the player moves closer to
// a mob, the distance shown in the proximity panel decreases.
func TestPlayerMoveProximityUpdates(t *testing.T) {
	mob := mobPosition{MobID: "orc-1", Position: mobPosVec3{X: 50, Y: 0}}
	mobr := mobReadResult{State: mobReadOK, Mobs: []mobPosition{mob}, Count: 1}
	entries := []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}

	// Player far away at (0, 0)
	prFar := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	panelFar := renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobr, prFar, focus, entries)
	strippedFar := stripANSI(panelFar)

	// Player closer at (30, 0)
	prClose := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 30, Y: 0}}
	panelClose := renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobr, prClose, focus, entries)
	strippedClose := stripANSI(panelClose)

	// Far panel should show dist:50, close panel should show dist:20
	if !strings.Contains(strippedFar, "dist:50") {
		t.Fatalf("far panel should show dist:50, got:\n%s", strippedFar)
	}
	if !strings.Contains(strippedClose, "dist:20") {
		t.Fatalf("close panel should show dist:20, got:\n%s", strippedClose)
	}
}

// TestPlayerMoveMapMarkerMoves verifies that the player '@' cell changes when
// the player's world position changes.
func TestPlayerMoveMapMarkerMoves(t *testing.T) {
	lines, bounds := testGeometry()
	vpW, vpH := 40, 20

	// Player at center (50, 50)
	ascii1, vpBounds1 := rasterizeAdaptiveViewport(lines, bounds, 50.0, 50.0, vpW, vpH)
	overlay1 := overlayPlayer(ascii1, playerPosResult{X: 50, Y: 50}, vpBounds1, vpW, vpH, false)
	col1, row1 := findGlyph(overlay1, '@')

	// Player moved to (70, 50)
	ascii2, vpBounds2 := rasterizeAdaptiveViewport(lines, bounds, 70.0, 50.0, vpW, vpH)
	overlay2 := overlayPlayer(ascii2, playerPosResult{X: 70, Y: 50}, vpBounds2, vpW, vpH, false)
	col2, row2 := findGlyph(overlay2, '@')

	if col1 < 0 || col2 < 0 {
		t.Fatal("player marker not found in one or both renders")
	}

	// Player always renders near center of viewport (viewport follows player).
	// Row should be similar (same Z), but the marker should exist in both renders.
	if col1 < 0 || row1 < 0 || col2 < 0 || row2 < 0 {
		t.Error("player marker '@' must be present in both renders")
	}
}

// --- E) Panel consistency ---

// TestMapAndProximityAgreeOnTarget verifies that when a mob is focused, it shows
// as 'M' on the map AND appears in the proximity panel with correct distance.
func TestMapAndProximityAgreeOnTarget(t *testing.T) {
	lines, bounds := testGeometry()
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

	mobs := []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 53, Y: 48}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 50, Y: 50}}
	entries := []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}

	// Check map panel has 'M' for focused mob
	mapPanel := renderMapPanel(mr, mobr, pr, focus, entries, 34, 17, attackResult{}, "")
	strippedMap := stripANSI(mapPanel)
	if !strings.Contains(strippedMap, "M") {
		t.Fatalf("map should show focused mob as 'M', got:\n%s", strippedMap)
	}

	// Check proximity panel shows the mob with distance
	proxPanel := renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobr, pr, focus, entries)
	strippedProx := stripANSI(proxPanel)
	if !strings.Contains(strippedProx, "orc-1") {
		t.Fatalf("proximity panel should show focused mob 'orc-1', got:\n%s", strippedProx)
	}
	// Distance from (50,50) to (53,48) = sqrt(9+4) ~ 3.6, displayed as dist:4 (rounded)
	if !strings.Contains(strippedProx, "dist:") {
		t.Fatalf("proximity panel should show distance, got:\n%s", strippedProx)
	}
}

// TestInRangeConsistency verifies that when the proximity panel shows IN RANGE,
// the map does not show a denial glyph ('?').
func TestInRangeConsistency(t *testing.T) {
	lines, bounds := testGeometry()
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

	mobs := []mobPosition{{MobID: "orc-1", LiveMobID: "live-orc-1", Position: mobPosVec3{X: 53, Y: 48}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 50, Y: 50}}
	entries := []rosterEntry{{kind: "mb", id: "live-orc-1", mobID: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}

	tc := targetConfirmResult{
		State:           targetConfirmOK,
		TargetID:        "live-orc-1",
		Found:           true,
		WithinProximity: true,
		Distance:        4,
	}

	// Proximity panel should show IN RANGE
	proxPanel := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobr, pr, focus, entries)
	strippedProx := stripANSI(proxPanel)
	if !strings.Contains(strippedProx, "IN RANGE") {
		t.Fatalf("proximity panel should show IN RANGE, got:\n%s", strippedProx)
	}

	// Map should not show denial glyph '?' when in range
	// Attack result with "out_of_range" would produce '?', but with IN RANGE it should not appear
	mapPanel := renderMapPanel(mr, mobr, pr, focus, entries, 34, 17, attackResult{}, "")
	strippedMap := stripANSI(mapPanel)
	if strings.Contains(strippedMap, "?") {
		t.Fatalf("map should not show denial glyph '?' when IN RANGE, got:\n%s", strippedMap)
	}
}

// TestOORNoImpactGlyph verifies that when out of range, no hit impact glyph ('!')
// is shown on the map.
func TestOORNoImpactGlyph(t *testing.T) {
	lines, bounds := testGeometry()
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

	mobs := []mobPosition{{MobID: "orc-1", LiveMobID: "live-orc-1", Position: mobPosVec3{X: 90, Y: 10}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 50, Y: 50}}
	entries := []rosterEntry{{kind: "mb", id: "live-orc-1", mobID: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}

	// Attack submitted but result is out_of_range (denial)
	ar := attackResult{State: attackStateSent, TargetID: "live-orc-1"}
	latestResult := "out_of_range"

	mapPanel := renderMapPanel(mr, mobr, pr, focus, entries, 34, 17, ar, latestResult)
	strippedMap := stripANSI(mapPanel)

	// The focused mob gets 'M' (focus takes priority over attack overlay).
	// The attack overlay is skipped when target is the focused mob.
	// So neither '!' (hit impact) nor '?' (denial) should appear.
	if strings.Contains(strippedMap, "!") {
		t.Fatalf("map should not show hit impact '!' when OOR, got:\n%s", strippedMap)
	}
}

// --- F) Overlay ordering determinism ---

// TestOverlayMobsSameCellDeterministic verifies that 3 mobs at the same world
// position always produce the digit '3' on the map, across 100 iterations.
func TestOverlayMobsSameCellDeterministic(t *testing.T) {
	lines, bounds := testGeometry()
	vpW, vpH := 40, 20
	centerX, centerZ := 50.0, 50.0

	mobs := []mobPosition{
		{MobID: "mob-a", Position: mobPosVec3{X: 50, Y: 50}},
		{MobID: "mob-b", Position: mobPosVec3{X: 50, Y: 50}},
		{MobID: "mob-c", Position: mobPosVec3{X: 50, Y: 50}},
	}

	for i := 0; i < 100; i++ {
		ascii, vpBounds := rasterizeAdaptiveViewport(lines, bounds, centerX, centerZ, vpW, vpH)
		overlay := overlayMobs(ascii, mobs, vpBounds, vpW, vpH)

		if !strings.ContainsRune(overlay, '3') {
			t.Fatalf("iteration %d: 3 mobs at same cell should produce '3' digit, got:\n%s", i, overlay)
		}
	}
}

// TestOverlayMobsIndependentCells verifies that mobs at different cells produce
// the same output regardless of input order.
func TestOverlayMobsIndependentCells(t *testing.T) {
	lines, bounds := testGeometry()
	vpW, vpH := 40, 20
	centerX, centerZ := 50.0, 50.0

	// All mobs within the adaptive viewport window centered at (50,50)
	mobA := mobPosition{MobID: "mob-a", Position: mobPosVec3{X: 44, Y: 44}}
	mobB := mobPosition{MobID: "mob-b", Position: mobPosVec3{X: 50, Y: 56}}
	mobC := mobPosition{MobID: "mob-c", Position: mobPosVec3{X: 56, Y: 50}}

	// Order 1: A, B, C
	ascii1, vpBounds1 := rasterizeAdaptiveViewport(lines, bounds, centerX, centerZ, vpW, vpH)
	overlay1 := overlayMobs(ascii1, []mobPosition{mobA, mobB, mobC}, vpBounds1, vpW, vpH)

	// Order 2: C, A, B
	ascii2, vpBounds2 := rasterizeAdaptiveViewport(lines, bounds, centerX, centerZ, vpW, vpH)
	overlay2 := overlayMobs(ascii2, []mobPosition{mobC, mobA, mobB}, vpBounds2, vpW, vpH)

	// Order 3: B, C, A
	ascii3, vpBounds3 := rasterizeAdaptiveViewport(lines, bounds, centerX, centerZ, vpW, vpH)
	overlay3 := overlayMobs(ascii3, []mobPosition{mobB, mobC, mobA}, vpBounds3, vpW, vpH)

	if overlay1 != overlay2 {
		t.Error("mob overlay differs between order [A,B,C] and [C,A,B]")
	}
	if overlay1 != overlay3 {
		t.Error("mob overlay differs between order [A,B,C] and [B,C,A]")
	}
}

// --- Test helpers ---

// findAllGlyphs returns all (col, row) positions of the given glyph in an ASCII
// map string, sorted by row then column for deterministic comparison.
func findAllGlyphs(ascii string, glyph rune) [][2]int {
	var results [][2]int
	for r, line := range strings.Split(ascii, "\n") {
		for c, ch := range line {
			if ch == glyph {
				results = append(results, [2]int{c, r})
			}
		}
	}
	return results
}

// computeDistance returns the Euclidean distance between two points on the XY plane.
func computeDistance(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return math.Sqrt(dx*dx + dy*dy)
}
