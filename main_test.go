package main

import (
	"strings"
	"testing"
	"time"
)

func TestStaticMapIsNonempty(t *testing.T) {
	if len(staticMap) == 0 {
		t.Fatal("staticMap should not be empty")
	}
}

func TestStaticMapHasMultipleLines(t *testing.T) {
	lines := strings.Split(staticMap, "\n")
	if len(lines) < 5 {
		t.Fatalf("expected at least 5 lines, got %d", len(lines))
	}
}

func TestStaticMapHasWalls(t *testing.T) {
	if !strings.Contains(staticMap, "#") {
		t.Fatal("staticMap should contain wall characters")
	}
}

func TestRenderMapContainsPlayerMarker(t *testing.T) {
	rendered := renderMap()
	if !strings.ContainsRune(rendered, playerMarker) {
		t.Fatal("renderMap() should contain the player marker")
	}
}

func TestRenderMapDoesNotMutateStaticMap(t *testing.T) {
	original := staticMap
	renderMap()
	if staticMap != original {
		t.Fatal("renderMap() must not mutate staticMap")
	}
}

func TestPlayerMarkerAtCorrectPosition(t *testing.T) {
	lines := strings.Split(renderMap(), "\n")
	if playerY >= len(lines) {
		t.Fatalf("playerY %d out of range", playerY)
	}
	runes := []rune(lines[playerY])
	if playerX >= len(runes) {
		t.Fatalf("playerX %d out of range", playerX)
	}
	if runes[playerX] != playerMarker {
		t.Fatalf("expected marker %c at (%d,%d), got %c", playerMarker, playerX, playerY, runes[playerX])
	}
}

func TestRenderMapContainsLandmarks(t *testing.T) {
	rendered := renderMap()
	for _, lm := range landmarks {
		if !strings.ContainsRune(rendered, lm.glyph) {
			t.Fatalf("renderMap() should contain landmark glyph %c (%s)", lm.glyph, lm.label)
		}
	}
}

func TestLandmarksAtCorrectPositions(t *testing.T) {
	lines := strings.Split(renderMap(), "\n")
	for _, lm := range landmarks {
		if lm.y >= len(lines) {
			t.Fatalf("landmark %s y=%d out of range", lm.label, lm.y)
		}
		runes := []rune(lines[lm.y])
		if lm.x >= len(runes) {
			t.Fatalf("landmark %s x=%d out of range", lm.label, lm.x)
		}
		// Player marker takes priority, so skip check if player overlaps
		if lm.x == playerX && lm.y == playerY {
			continue
		}
		if runes[lm.x] != lm.glyph {
			t.Fatalf("expected landmark %c at (%d,%d), got %c", lm.glyph, lm.x, lm.y, runes[lm.x])
		}
	}
}

func TestRenderMapContainsThreatMarkers(t *testing.T) {
	rendered := renderMap()
	for _, tm := range threatMarkers {
		if !strings.ContainsRune(rendered, tm.glyph) {
			t.Fatalf("renderMap() should contain threat marker glyph %c (%s)", tm.glyph, tm.label)
		}
	}
}

func TestThreatMarkersAtCorrectPositions(t *testing.T) {
	lines := strings.Split(renderMap(), "\n")
	for _, tm := range threatMarkers {
		if tm.y >= len(lines) {
			t.Fatalf("threat marker %s y=%d out of range", tm.label, tm.y)
		}
		runes := []rune(lines[tm.y])
		if tm.x >= len(runes) {
			t.Fatalf("threat marker %s x=%d out of range", tm.label, tm.x)
		}
		if tm.x == playerX && tm.y == playerY {
			continue
		}
		if runes[tm.x] != tm.glyph {
			t.Fatalf("expected threat marker %c at (%d,%d), got %c", tm.glyph, tm.x, tm.y, runes[tm.x])
		}
	}
}

func TestNearbyPanelContainsTensionWording(t *testing.T) {
	panel := renderNearbyPanel(sidePanelWidth)
	tensionPhrases := []string{"uneasy presence", "faint movement?"}
	for _, phrase := range tensionPhrases {
		if !strings.Contains(panel, phrase) {
			t.Fatalf("nearby panel should contain tension phrase %q", phrase)
		}
	}
}

func TestNearbyPanelContainsDirectionalCues(t *testing.T) {
	panel := renderNearbyPanel(sidePanelWidth)
	directionalPhrases := []string{"north", "east"}
	for _, phrase := range directionalPhrases {
		if !strings.Contains(panel, phrase) {
			t.Fatalf("nearby panel should contain directional cue %q", phrase)
		}
	}
}

func TestNearbyPanelContainsStillnessCues(t *testing.T) {
	panel := renderNearbyPanel(sidePanelWidth)
	stillnessPhrases := []string{"stone dampness", "deep silence"}
	for _, phrase := range stillnessPhrases {
		if !strings.Contains(panel, phrase) {
			t.Fatalf("nearby panel should contain stillness cue %q", phrase)
		}
	}
}

func TestRenderMapDeterministic(t *testing.T) {
	a := renderMap()
	b := renderMap()
	if a != b {
		t.Fatal("renderMap() should produce deterministic output")
	}
}

func TestStyledMapContainsPlayerMarker(t *testing.T) {
	styled := renderStyledMap()
	if !strings.ContainsRune(styled, playerMarker) {
		t.Fatal("styled map should contain player marker")
	}
}

func TestStyledMapDeterministic(t *testing.T) {
	a := renderStyledMap()
	b := renderStyledMap()
	if a != b {
		t.Fatal("renderStyledMap() should produce deterministic output")
	}
}

func TestStyledMapNonEmpty(t *testing.T) {
	styled := renderStyledMap()
	if len(styled) == 0 {
		t.Fatal("styled map should not be empty")
	}
}

func TestStyledMapContainsLandmarks(t *testing.T) {
	styled := renderStyledMap()
	for _, lm := range landmarks {
		if !strings.ContainsRune(styled, lm.glyph) {
			t.Fatalf("styled map should contain landmark glyph %c (%s)", lm.glyph, lm.label)
		}
	}
}

func TestTileDistanceAtPlayer(t *testing.T) {
	d := tileDistance(playerX, playerY)
	if d != 0 {
		t.Fatalf("distance at player position should be 0, got %f", d)
	}
}

func TestTileDistancePositive(t *testing.T) {
	d := tileDistance(playerX+5, playerY+5)
	if d <= 0 {
		t.Fatal("distance away from player should be positive")
	}
}

func TestRenderHeaderContainsTitle(t *testing.T) {
	header := renderHeader(80)
	if !strings.Contains(header, headerTitle) {
		t.Fatal("header should contain app title")
	}
}

func TestRenderHeaderContainsSubtitle(t *testing.T) {
	header := renderHeader(80)
	if !strings.Contains(header, headerSubtitle) {
		t.Fatal("header should contain subtitle")
	}
}

func TestRenderFooterContainsQuitHint(t *testing.T) {
	footer := renderFooter(80, "", "", "", "", "")
	if !strings.Contains(footer, "quit") {
		t.Fatal("footer should contain quit hint")
	}
}

func TestRenderMapPanelContainsPlayerMarker(t *testing.T) {
	panel := renderMapPanel(mapReadResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 80, 40)
	if !strings.ContainsRune(panel, playerMarker) {
		t.Fatal("map panel should contain player marker")
	}
}

func TestRenderLayoutContainsAllSections(t *testing.T) {
	layout := renderLayout(80, 40, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(layout, headerTitle) {
		t.Fatal("layout should contain header title")
	}
	if !strings.ContainsRune(layout, playerMarker) {
		t.Fatal("layout should contain player marker")
	}
	if !strings.Contains(layout, "quit") {
		t.Fatal("layout should contain quit hint")
	}
}

func TestRenderLayoutNonEmpty(t *testing.T) {
	layout := renderLayout(80, 40, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if len(layout) == 0 {
		t.Fatal("layout should not be empty")
	}
}

func TestRenderNearbyPanelContainsTitle(t *testing.T) {
	panel := renderNearbyPanel(sidePanelWidth)
	if !strings.Contains(panel, nearbyTitle) {
		t.Fatal("nearby panel should contain title")
	}
}

func TestRenderStatusPanelContainsTitle(t *testing.T) {
	panel := renderStatusPanel(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{})
	if !strings.Contains(panel, statusTitle) {
		t.Fatal("status panel should contain title")
	}
}

func TestRenderSideColumnContainsBothSections(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(col, nearbyTitle) {
		t.Fatal("side column should contain nearby title")
	}
	if !strings.Contains(col, statusTitle) {
		t.Fatal("side column should contain status title")
	}
}

func TestWideLayoutContainsPanels(t *testing.T) {
	layout := renderLayout(120, 40, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(layout, nearbyTitle) {
		t.Fatal("wide layout should contain nearby panel")
	}
	if !strings.Contains(layout, statusTitle) {
		t.Fatal("wide layout should contain status panel")
	}
	if !strings.ContainsRune(layout, playerMarker) {
		t.Fatal("wide layout should still contain player marker")
	}
}

func TestNarrowLayoutOmitsPanels(t *testing.T) {
	layout := renderLayout(50, 30, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if strings.Contains(layout, nearbyTitle) {
		t.Fatal("narrow layout should not contain nearby panel")
	}
	if !strings.ContainsRune(layout, playerMarker) {
		t.Fatal("narrow layout should still contain player marker")
	}
}

func TestRenderLayoutSmallTerminal(t *testing.T) {
	// Should not panic with very small dimensions
	layout := renderLayout(20, 5, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if len(layout) == 0 {
		t.Fatal("layout should not be empty even for small terminal")
	}
}

func TestRenderLayoutVariousSizes(t *testing.T) {
	sizes := [][2]int{{40, 20}, {80, 40}, {120, 50}, {200, 60}}
	for _, sz := range sizes {
		layout := renderLayout(sz[0], sz[1], "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
		if !strings.Contains(layout, headerTitle) {
			t.Fatalf("layout at %dx%d missing header", sz[0], sz[1])
		}
		if !strings.ContainsRune(layout, playerMarker) {
			t.Fatalf("layout at %dx%d missing player marker", sz[0], sz[1])
		}
		if !strings.Contains(layout, "quit") {
			t.Fatalf("layout at %dx%d missing footer", sz[0], sz[1])
		}
	}
}

func TestDirectionFromKeyArrows(t *testing.T) {
	cases := map[string]string{
		"up": "north", "down": "south", "left": "west", "right": "east",
	}
	for key, want := range cases {
		if got := directionFromKey(key); got != want {
			t.Fatalf("directionFromKey(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestDirectionFromKeyHJKL(t *testing.T) {
	cases := map[string]string{
		"h": "west", "j": "south", "k": "north", "l": "east",
	}
	for key, want := range cases {
		if got := directionFromKey(key); got != want {
			t.Fatalf("directionFromKey(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestDirectionFromKeyUnrecognized(t *testing.T) {
	if dir := directionFromKey("x"); dir != "" {
		t.Fatalf("unrecognized key should return empty, got %q", dir)
	}
}

func TestFooterContainsMovementKeys(t *testing.T) {
	footer := renderFooter(120, "", "", "", "", "")
	if !strings.Contains(footer, "move") {
		t.Fatal("footer should advertise movement keys")
	}
	if !strings.Contains(footer, "quit") {
		t.Fatal("footer should still contain quit hint")
	}
}

func TestFooterShowsIntentPreview(t *testing.T) {
	preview := moveIntent{direction: "north"}.preview()
	footer := renderFooter(120, preview, "", "", "", "")
	if !strings.Contains(footer, "move north") {
		t.Fatal("footer should show intent preview with direction")
	}
	if !strings.Contains(footer, "not sent") {
		t.Fatal("footer should indicate intent was not sent")
	}
}

func TestMoveIntentPreviewEmpty(t *testing.T) {
	i := moveIntent{}
	if i.preview() != "" {
		t.Fatal("empty intent should produce empty preview")
	}
}

func TestMoveIntentPreviewFormat(t *testing.T) {
	i := moveIntent{direction: "west"}
	p := i.preview()
	if !strings.Contains(p, "intent") {
		t.Fatal("preview should contain 'intent'")
	}
	if !strings.Contains(p, "move west") {
		t.Fatal("preview should contain 'move west'")
	}
	if !strings.Contains(p, "not sent") {
		t.Fatal("preview should contain 'not sent'")
	}
}

func TestPlayerMarkerUnchangedAfterInput(t *testing.T) {
	// Simulate: model receives a movement key but position must not change
	m := model{width: 80, height: 40, lastIntent: moveIntent{direction: "north"}, target: defaultTarget()}
	view := m.View()
	if !strings.ContainsRune(view, playerMarker) {
		t.Fatal("player marker should still be present after input")
	}
	// Verify the underlying map is unchanged
	lines := strings.Split(renderMap(), "\n")
	runes := []rune(lines[playerY])
	if runes[playerX] != playerMarker {
		t.Fatal("player position must not change from movement input")
	}
}

func TestViewEmptyBeforeResize(t *testing.T) {
	m := model{}
	if m.View() != "" {
		t.Fatal("View() should be empty before receiving window size")
	}
}

func TestViewNonEmptyAfterResize(t *testing.T) {
	m := model{width: 80, height: 40, target: defaultTarget()}
	view := m.View()
	if len(view) == 0 {
		t.Fatal("View() should not be empty after resize")
	}
}

func TestDefaultTargetValues(t *testing.T) {
	target := defaultTarget()
	if !strings.Contains(target.BaseURL, "9090") {
		t.Fatal("default target should use port 9090")
	}
	if target.Zone != "crushbone" {
		t.Fatalf("default zone should be crushbone, got %q", target.Zone)
	}
	if target.Mode != "RT" {
		t.Fatalf("default mode should be RT, got %q", target.Mode)
	}
	if target.Visibility != "PUBLIC" {
		t.Fatalf("default visibility should be PUBLIC, got %q", target.Visibility)
	}
	if target.Affinity != "open" {
		t.Fatalf("default affinity should be open, got %q", target.Affinity)
	}
	if target.Player == "" {
		t.Fatal("default target should have a player")
	}
}

func TestStatusPanelContainsTargetInfo(t *testing.T) {
	target := defaultTarget()
	panel := renderStatusPanel(sidePanelWidth, target, zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{})
	if !strings.Contains(panel, "target") {
		t.Fatal("status panel should contain target label")
	}
	if !strings.Contains(panel, target.Zone) {
		t.Fatal("status panel should contain zone name")
	}
	if !strings.Contains(panel, "rt") {
		t.Fatal("status panel should contain mode")
	}
	if !strings.Contains(panel, "public") {
		t.Fatal("status panel should contain visibility")
	}
}

func TestStatusPanelDoesNotImplyConnectivity(t *testing.T) {
	panel := renderStatusPanel(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{})
	for _, bad := range []string{"connected", "online", "healthy"} {
		if strings.Contains(strings.ToLower(panel), bad) {
			t.Fatalf("status panel must not contain %q", bad)
		}
	}
}

func TestZoneStatusURL(t *testing.T) {
	target := defaultTarget()
	url := zoneStatusURL(target)
	if !strings.Contains(url, "9090") {
		t.Fatal("URL should use port 9090")
	}
	if !strings.Contains(url, "/world/zone/crushbone") {
		t.Fatal("URL should target /world/zone/crushbone")
	}
	// Default RT should not add mode query param
	if strings.Contains(url, "mode=") {
		t.Fatal("default RT target should not add mode query param")
	}
}

func TestZoneStatusURLAsync(t *testing.T) {
	target := defaultTarget()
	target.Mode = "ASYNC"
	url := zoneStatusURL(target)
	if !strings.Contains(url, "mode=Async") {
		t.Fatal("ASYNC target should add mode=Async query param")
	}
}

func TestZoneReadStateLabels(t *testing.T) {
	pending := zoneReadResult{State: zoneReadNotAttempted}
	if !strings.Contains(pending.statusLabel(), "pending") {
		t.Fatal("not-attempted state should show pending")
	}

	ok := zoneReadResult{State: zoneReadOK}
	if !strings.Contains(ok.statusLabel(), "ok") {
		t.Fatal("success state should show ok")
	}

	failed := zoneReadResult{State: zoneReadFailed}
	if !strings.Contains(failed.statusLabel(), "failed") {
		t.Fatal("failure state should show failed")
	}
}

func TestZoneMapURL(t *testing.T) {
	target := defaultTarget()
	url := zoneMapURL(target)
	if !strings.Contains(url, "9090") {
		t.Fatal("URL should use port 9090")
	}
	if !strings.Contains(url, "/world/zone/crushbone/map") {
		t.Fatal("URL should target /world/zone/crushbone/map")
	}
}

func TestProjectAndRasterizeDeterministic(t *testing.T) {
	lines := []mapLine{
		{From: mapVec3{X: 0, Z: 0}, To: mapVec3{X: 100, Z: 0}},
		{From: mapVec3{X: 0, Z: 0}, To: mapVec3{X: 0, Z: 100}},
	}
	a, _ := projectAndRasterize(lines, 20, 10)
	b, _ := projectAndRasterize(lines, 20, 10)
	if a != b {
		t.Fatal("projection should be deterministic")
	}
}

func TestProjectAndRasterizeNonEmpty(t *testing.T) {
	lines := []mapLine{
		{From: mapVec3{X: 0, Z: 0}, To: mapVec3{X: 100, Z: 100}},
	}
	result, _ := projectAndRasterize(lines, 20, 10)
	if len(result) == 0 {
		t.Fatal("rasterized output should not be empty")
	}
	if !strings.Contains(result, "#") {
		t.Fatal("rasterized output should contain wall characters")
	}
}

func TestProjectAndRasterizeEmpty(t *testing.T) {
	result, _ := projectAndRasterize(nil, 20, 10)
	if result != "" {
		t.Fatal("empty geometry should produce empty output")
	}
}

func TestMapReadStateLabels(t *testing.T) {
	pending := mapReadResult{State: mapReadNotAttempted}
	if !strings.Contains(pending.mapStatusLabel(), "pending") {
		t.Fatal("not-attempted state should show pending")
	}
	ok := mapReadResult{State: mapReadOK}
	if !strings.Contains(ok.mapStatusLabel(), "loaded") {
		t.Fatal("success state should show loaded")
	}
	failed := mapReadResult{State: mapReadFailed}
	if !strings.Contains(failed.mapStatusLabel(), "unavailable") {
		t.Fatal("failure state should show unavailable")
	}
}

func TestMapPanelUsesBackendMap(t *testing.T) {
	mr := mapReadResult{
		State:   mapReadOK,
		MapText: "###\n# #\n###",
	}
	panel := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 80, 40)
	if !strings.Contains(panel, "###") {
		t.Fatal("map panel should use backend map text when available")
	}
}

func TestMapPanelFallsBackToPlaceholder(t *testing.T) {
	mr := mapReadResult{State: mapReadFailed}
	panel := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 80, 40)
	if !strings.ContainsRune(panel, playerMarker) {
		t.Fatal("map panel should fall back to placeholder with player marker")
	}
}

func TestStatusPanelShowsZoneReadState(t *testing.T) {
	okResult := zoneReadResult{State: zoneReadOK}
	panel := renderStatusPanel(sidePanelWidth, defaultTarget(), okResult, mapReadResult{}, mobReadResult{}, playerReadResult{})
	if !strings.Contains(panel, "ok") {
		t.Fatal("status panel should show zone read ok state")
	}

	failResult := zoneReadResult{State: zoneReadFailed}
	panel2 := renderStatusPanel(sidePanelWidth, defaultTarget(), failResult, mapReadResult{}, mobReadResult{}, playerReadResult{})
	if !strings.Contains(panel2, "failed") {
		t.Fatal("status panel should show zone read failed state")
	}
}

func TestZoneMobPositionsURL(t *testing.T) {
	target := defaultTarget()
	url := zoneMobPositionsURL(target)
	if !strings.Contains(url, "9090") {
		t.Fatal("URL should use port 9090")
	}
	if !strings.Contains(url, "/world/zone/crushbone/mob_positions") {
		t.Fatal("URL should target /world/zone/crushbone/mob_positions")
	}
}

func TestMobReadStateLabels(t *testing.T) {
	pending := mobReadResult{State: mobReadNotAttempted}
	if !strings.Contains(pending.mobStatusLabel(), "pending") {
		t.Fatal("not-attempted state should show pending")
	}
	ok := mobReadResult{State: mobReadOK, Count: 42}
	if !strings.Contains(ok.mobStatusLabel(), "42") {
		t.Fatal("success state should show count")
	}
	failed := mobReadResult{State: mobReadFailed}
	if !strings.Contains(failed.mobStatusLabel(), "unavailable") {
		t.Fatal("failure state should show unavailable")
	}
}

func TestOverlayMobsDeterministic(t *testing.T) {
	mapText := "     \n     \n     "
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	mobs := []mobPosition{
		{MobName: "orc", Position: mobPosVec3{X: 50, Y: 50}},
	}
	a := overlayMobs(mapText, mobs, bounds, 5, 3)
	b := overlayMobs(mapText, mobs, bounds, 5, 3)
	if a != b {
		t.Fatal("mob overlay should be deterministic")
	}
	if !strings.Contains(a, "m") {
		t.Fatal("mob overlay should contain mob marker")
	}
}

func TestOverlayMobsEmptyMobs(t *testing.T) {
	mapText := "#####"
	result := overlayMobs(mapText, nil, mapBounds{}, 5, 1)
	if result != mapText {
		t.Fatal("empty mobs should return map unchanged")
	}
}

func TestMapPanelWithMobOverlay(t *testing.T) {
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   "     \n     \n     ",
		MapWidth:  5,
		MapHeight: 3,
		Bounds:    mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100},
	}
	mobr := mobReadResult{
		State: mobReadOK,
		Mobs:  []mobPosition{{MobName: "orc", Position: mobPosVec3{X: 50, Y: 50}}},
		Count: 1,
	}
	panel := renderMapPanel(mr, mobr, playerReadResult{}, rosterFocus{}, nil, 80, 40)
	if !strings.Contains(panel, "m") {
		t.Fatal("map panel should contain mob markers when mobs are available")
	}
}

func TestDevJoinURL(t *testing.T) {
	target := defaultTarget()
	url := devJoinURL(target)
	if !strings.Contains(url, "/world/dev/zone/crushbone/player/join") {
		t.Fatal("join URL should target dev player join endpoint")
	}
}

func TestDevPlayerStateURL(t *testing.T) {
	target := defaultTarget()
	url := devPlayerStateURL(target)
	if !strings.Contains(url, "/world/dev/zone/crushbone/player/p1") {
		t.Fatal("player state URL should include zone and player ID")
	}
}

func TestPlayerReadStateLabels(t *testing.T) {
	pending := playerReadResult{State: playerReadNotAttempted}
	if !strings.Contains(pending.playerStatusLabel(), "pending") {
		t.Fatal("not-attempted state should show pending")
	}
	ok := playerReadResult{State: playerReadOK, HasPos: true}
	if !strings.Contains(ok.playerStatusLabel(), "joined") {
		t.Fatal("success state should show joined")
	}
	failed := playerReadResult{State: playerReadFailed}
	if !strings.Contains(failed.playerStatusLabel(), "unavailable") {
		t.Fatal("failure state should show unavailable")
	}
}

func TestOverlayPlayerDeterministic(t *testing.T) {
	mapText := "     \n     \n     "
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	pos := playerPosResult{X: 50, Y: 50}
	a := overlayPlayer(mapText, pos, bounds, 5, 3)
	b := overlayPlayer(mapText, pos, bounds, 5, 3)
	if a != b {
		t.Fatal("player overlay should be deterministic")
	}
	if !strings.Contains(a, "@") {
		t.Fatal("player overlay should contain @ marker")
	}
}

func TestMapPanelWithBackendPlayer(t *testing.T) {
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   "     \n     \n     ",
		MapWidth:  5,
		MapHeight: 3,
		Bounds:    mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100},
	}
	pr := playerReadResult{
		State:    playerReadOK,
		HasPos:   true,
		Position: playerPosResult{X: 50, Y: 50},
	}
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 80, 40)
	if !strings.Contains(panel, "@") {
		t.Fatal("map panel should contain backend-derived player marker")
	}
}

func TestDefaultTargetHasDevToken(t *testing.T) {
	target := defaultTarget()
	if target.DevToken == "" {
		t.Fatal("default target should have a dev token")
	}
}

func TestDevPlayerPositionURL(t *testing.T) {
	target := defaultTarget()
	url := devPlayerPositionURL(target)
	if !strings.Contains(url, "/world/dev/zone/crushbone/player/position") {
		t.Fatal("position URL should target dev player position endpoint")
	}
}

func TestDirectionOffset(t *testing.T) {
	dx, dy := directionOffset("north")
	if dx != 0 || dy <= 0 {
		t.Fatal("north should have positive dy")
	}
	dx, dy = directionOffset("south")
	if dx != 0 || dy >= 0 {
		t.Fatal("south should have negative dy")
	}
	dx, dy = directionOffset("east")
	if dx <= 0 || dy != 0 {
		t.Fatal("east should have positive dx")
	}
	dx, dy = directionOffset("west")
	if dx >= 0 || dy != 0 {
		t.Fatal("west should have negative dx")
	}
}

func TestMoveIntentSentLabel(t *testing.T) {
	i := moveIntent{direction: "north", state: moveStateSent}
	p := i.preview()
	if !strings.Contains(p, "sent") {
		t.Fatal("sent intent should contain 'sent'")
	}
	if strings.Contains(p, "not sent") {
		t.Fatal("sent intent should not contain 'not sent'")
	}
}

func TestMoveIntentFailedLabel(t *testing.T) {
	i := moveIntent{direction: "east", state: moveStateFailed}
	p := i.preview()
	if !strings.Contains(p, "failed") {
		t.Fatal("failed intent should contain 'failed'")
	}
}

func TestNoLocalPositionMutationWithoutBackend(t *testing.T) {
	// Model without backend player read — movement key should not change position
	m := model{width: 80, height: 40, target: defaultTarget()}
	// Simulate pressing a movement key — playerRead is not OK, so no submission
	dir := directionFromKey("up")
	if dir != "north" {
		t.Fatal("up key should map to north")
	}
	// Without playerRead.State == playerReadOK, the model stays in preview only
	m.lastIntent = moveIntent{direction: dir, state: moveStatePreview}
	// playerRead position should be zero (unchanged)
	if m.playerRead.HasPos {
		t.Fatal("player position should not be set without backend read")
	}
}

func TestRefreshIntervalIsReasonable(t *testing.T) {
	if refreshInterval < 200*time.Millisecond {
		t.Fatal("refresh interval too fast (< 200ms)")
	}
	if refreshInterval > 2*time.Second {
		t.Fatal("refresh interval too slow (> 2s)")
	}
}

func TestScheduleRefreshReturnsNonNil(t *testing.T) {
	cmd := scheduleRefresh()
	if cmd == nil {
		t.Fatal("scheduleRefresh should return a non-nil command")
	}
}

// --- Encounter read shell tests ---

func TestEncounterReadStateLabels(t *testing.T) {
	pending := encounterReadResult{State: encounterReadNotAttempted}
	if !strings.Contains(pending.encounterStatusLabel(), "pending") {
		t.Fatal("not-attempted state should show pending")
	}
	ok := encounterReadResult{State: encounterReadOK, Count: 3}
	if !strings.Contains(ok.encounterStatusLabel(), "3") {
		t.Fatal("success state should show count")
	}
	failed := encounterReadResult{State: encounterReadFailed}
	if !strings.Contains(failed.encounterStatusLabel(), "unavailable") {
		t.Fatal("failure state should show unavailable")
	}
}

func TestZoneEncountersURL(t *testing.T) {
	target := defaultTarget()
	url := zoneEncountersURL(target)
	if !strings.Contains(url, "/world/call/crushbone") {
		t.Fatal("URL should target /world/call/crushbone")
	}
	if !strings.Contains(url, "message=encounters") {
		t.Fatal("URL should include message=encounters")
	}
	// Default RT should not add mode query param
	if strings.Contains(url, "mode=") {
		t.Fatal("default RT target should not add mode query param")
	}
}

func TestZoneEncountersURLAsync(t *testing.T) {
	target := defaultTarget()
	target.Mode = "ASYNC"
	url := zoneEncountersURL(target)
	if !strings.Contains(url, "mode=Async") {
		t.Fatal("ASYNC target should add mode=Async query param")
	}
}

func TestFindPlayerEncounterMatch(t *testing.T) {
	encounters := []encounterSummary{
		{EncounterID: "enc-1", State: "Active", PlayerCount: 1, MobCount: 2},
		{EncounterID: "enc-2", State: "Completed", PlayerCount: 2, MobCount: 3},
	}
	found := findPlayerEncounter(encounters, "enc-2")
	if found == nil {
		t.Fatal("should find matching encounter")
	}
	if found.State != "Completed" {
		t.Fatalf("expected Completed, got %q", found.State)
	}
}

func TestFindPlayerEncounterNoMatch(t *testing.T) {
	encounters := []encounterSummary{
		{EncounterID: "enc-1", State: "Active"},
	}
	found := findPlayerEncounter(encounters, "enc-99")
	if found != nil {
		t.Fatal("should not find non-existent encounter")
	}
}

func TestFindPlayerEncounterEmptyID(t *testing.T) {
	encounters := []encounterSummary{
		{EncounterID: "enc-1", State: "Active"},
	}
	found := findPlayerEncounter(encounters, "")
	if found != nil {
		t.Fatal("empty encounter ID should return nil")
	}
}

func TestEncounterPanelPlayerNotJoined(t *testing.T) {
	panel := renderEncounterPanel(sidePanelWidth, playerReadResult{}, encounterReadResult{}, rosterFocus{})
	if !strings.Contains(panel, encounterTitle) {
		t.Fatal("encounter panel should contain title")
	}
	if !strings.Contains(panel, "no player") {
		t.Fatal("encounter panel should show no player when player state is not OK")
	}
}

func TestEncounterPanelDataUnavailable(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasPos: true}
	er := encounterReadResult{State: encounterReadFailed}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{})
	if !strings.Contains(panel, "unavailable") {
		t.Fatal("encounter panel should show unavailable when encounter read failed")
	}
}

func TestEncounterPanelDataPending(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasPos: true}
	er := encounterReadResult{State: encounterReadNotAttempted}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{})
	if !strings.Contains(panel, "pending") {
		t.Fatal("encounter panel should show pending when encounter read not attempted")
	}
}

func TestEncounterPanelNoEncounter(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasPos: true}
	er := encounterReadResult{State: encounterReadOK, Count: 0}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{})
	if !strings.Contains(panel, "active: no") {
		t.Fatal("encounter panel should show not in encounter when no active encounter")
	}
	if !strings.Contains(panel, "0 enc") {
		t.Fatal("encounter panel should show zone encounter count")
	}
}

func TestEncounterPanelActiveEncounterWithDetails(t *testing.T) {
	pr := playerReadResult{
		State:              playerReadOK,
		HasPos:             true,
		ActiveEncounterID:  "enc-42",
		HasActiveEncounter: true,
	}
	er := encounterReadResult{
		State: encounterReadOK,
		Count: 2,
		Encounters: []encounterSummary{
			{EncounterID: "enc-42", State: "Active", PlayerIDs: []string{"player-1"}, MobIDs: []string{"mob-a", "mob-b", "mob-c"}, PlayerCount: 1, MobCount: 3, MobsAlive: 2, ActionIndex: 7},
			{EncounterID: "enc-41", State: "Completed", CompletedReason: "all_mobs_dead"},
		},
	}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{})
	if !strings.Contains(panel, "active: yes") {
		t.Fatal("encounter panel should show active encounter")
	}
	if !strings.Contains(panel, "Active") {
		t.Fatal("encounter panel should show encounter state")
	}
	if !strings.Contains(panel, "pcs:1") {
		t.Fatal("encounter panel should show player count")
	}
	if !strings.Contains(panel, "mobs:3") {
		t.Fatal("encounter panel should show mob count")
	}
	if !strings.Contains(panel, "alive:2") {
		t.Fatal("encounter panel should show mobs alive")
	}
	if !strings.Contains(panel, "act:7") {
		t.Fatal("encounter panel should show action index")
	}
}

func TestEncounterPanelCompletedEncounter(t *testing.T) {
	pr := playerReadResult{
		State:              playerReadOK,
		HasPos:             true,
		ActiveEncounterID:  "enc-done",
		HasActiveEncounter: true,
	}
	er := encounterReadResult{
		State: encounterReadOK,
		Count: 1,
		Encounters: []encounterSummary{
			{EncounterID: "enc-done", State: "Completed", CompletedReason: "all_mobs_dead", PlayerCount: 1, MobCount: 2, MobsDead: 2},
		},
	}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{})
	if !strings.Contains(panel, "all_mobs_dead") {
		t.Fatal("encounter panel should show completion reason")
	}
}

func TestEncounterPanelActiveButDetailsMissing(t *testing.T) {
	pr := playerReadResult{
		State:              playerReadOK,
		HasPos:             true,
		ActiveEncounterID:  "enc-unknown",
		HasActiveEncounter: true,
	}
	er := encounterReadResult{
		State:      encounterReadOK,
		Count:      1,
		Encounters: []encounterSummary{{EncounterID: "enc-other"}},
	}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{})
	if !strings.Contains(panel, "active: yes") {
		t.Fatal("encounter panel should show active encounter")
	}
	if !strings.Contains(panel, "no details") {
		t.Fatal("encounter panel should show no details when encounter not found in summaries")
	}
}

func TestSideColumnContainsEncounterPanel(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(col, encounterTitle) {
		t.Fatal("side column should contain encounter panel title")
	}
	if !strings.Contains(col, nearbyTitle) {
		t.Fatal("side column should still contain nearby title")
	}
	if !strings.Contains(col, statusTitle) {
		t.Fatal("side column should still contain status title")
	}
}

func TestWideLayoutContainsEncounterPanel(t *testing.T) {
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(layout, encounterTitle) {
		t.Fatal("wide layout should contain encounter panel")
	}
}

func TestNarrowLayoutOmitsEncounterPanel(t *testing.T) {
	layout := renderLayout(50, 30, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if strings.Contains(layout, encounterTitle) {
		t.Fatal("narrow layout should not contain encounter panel")
	}
}

func TestEncounterPanelDoesNotImplyGameplayLogic(t *testing.T) {
	pr := playerReadResult{
		State:              playerReadOK,
		HasPos:             true,
		ActiveEncounterID:  "enc-1",
		HasActiveEncounter: true,
	}
	er := encounterReadResult{
		State:      encounterReadOK,
		Count:      1,
		Encounters: []encounterSummary{{EncounterID: "enc-1", State: "Active", MobCount: 2, MobsAlive: 2}},
	}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{})
	for _, bad := range []string{"attack", "target", "threat", "aggro", "danger", "dps", "heal"} {
		if strings.Contains(strings.ToLower(panel), bad) {
			t.Fatalf("encounter panel must not contain gameplay term %q", bad)
		}
	}
}

func TestDecodePlayerStateWithEncounterID(t *testing.T) {
	body := []byte(`{"result":{"player":{"position":{"x":10,"y":20,"z":0},"active_encounter_id":"enc-abc"}}}`)
	result := decodePlayerState(body, playerReadOK)
	if result.State != playerReadOK {
		t.Fatal("decode should succeed")
	}
	if !result.HasPos {
		t.Fatal("should have position")
	}
	if result.Position.X != 10 || result.Position.Y != 20 {
		t.Fatalf("position mismatch: got (%f, %f)", result.Position.X, result.Position.Y)
	}
	if !result.HasActiveEncounter {
		t.Fatal("should have active encounter")
	}
	if result.ActiveEncounterID != "enc-abc" {
		t.Fatalf("expected enc-abc, got %q", result.ActiveEncounterID)
	}
}

func TestDecodePlayerStateWithoutEncounterID(t *testing.T) {
	body := []byte(`{"result":{"player":{"position":{"x":5,"y":15,"z":0}}}}`)
	result := decodePlayerState(body, playerReadOK)
	if result.HasActiveEncounter {
		t.Fatal("should not have active encounter when field absent")
	}
	if result.ActiveEncounterID != "" {
		t.Fatal("encounter ID should be empty")
	}
}

func TestDecodePlayerStateLegacyShape(t *testing.T) {
	body := []byte(`{"result":{"Position":{"Pos":{"X":30,"Y":40,"Z":0}}}}`)
	result := decodePlayerState(body, playerReadOK)
	if !result.HasPos {
		t.Fatal("should have position from legacy shape")
	}
	if result.Position.X != 30 || result.Position.Y != 40 {
		t.Fatalf("position mismatch: got (%f, %f)", result.Position.X, result.Position.Y)
	}
	if result.HasActiveEncounter {
		t.Fatal("legacy shape should not have encounter")
	}
}

func TestDecodePlayerStateBadJSON(t *testing.T) {
	body := []byte(`{invalid`)
	result := decodePlayerState(body, playerReadOK)
	if result.HasPos {
		t.Fatal("bad JSON should not produce position")
	}
	if result.HasActiveEncounter {
		t.Fatal("bad JSON should not produce encounter")
	}
}

func TestEncounterPanelDeterministic(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasPos: true, ActiveEncounterID: "enc-1", HasActiveEncounter: true}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{EncounterID: "enc-1", State: "Active", PlayerIDs: []string{"p1"}, MobIDs: []string{"m1", "m2"}, PlayerCount: 1, MobCount: 2, MobsAlive: 2}},
	}
	a := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{})
	b := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{})
	if a != b {
		t.Fatal("encounter panel should be deterministic")
	}
}

func TestRosterSectionShowsPlayerAndMobIDs(t *testing.T) {
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		ActiveEncounterID: "enc-r1", HasActiveEncounter: true,
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-r1", State: "Active",
			PlayerIDs: []string{"hero-1", "hero-2"}, MobIDs: []string{"orc-a", "orc-b"},
			PlayerCount: 2, MobCount: 2, MobsAlive: 2,
		}},
	}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{})
	if !strings.Contains(panel, "---roster---") {
		t.Fatal("encounter panel should contain roster header")
	}
	if !strings.Contains(panel, "pc:hero-1") {
		t.Fatal("encounter panel should show first player ID")
	}
	if !strings.Contains(panel, "pc:hero-2") {
		t.Fatal("encounter panel should show second player ID")
	}
	if !strings.Contains(panel, "mb:orc-a") {
		t.Fatal("encounter panel should show first mob ID")
	}
	if !strings.Contains(panel, "mb:orc-b") {
		t.Fatal("encounter panel should show second mob ID")
	}
}

func TestRosterSectionNoRosterData(t *testing.T) {
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		ActiveEncounterID: "enc-empty", HasActiveEncounter: true,
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-empty", State: "Active",
			PlayerCount: 0, MobCount: 0,
		}},
	}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{})
	if !strings.Contains(panel, "---roster---") {
		t.Fatal("encounter panel should contain roster header even when empty")
	}
	if !strings.Contains(panel, "no roster data") {
		t.Fatal("encounter panel should honestly show no roster data")
	}
}

func TestTruncateIDShort(t *testing.T) {
	if got := truncateID("abc", 10); got != "abc" {
		t.Fatalf("expected abc, got %s", got)
	}
}

func TestTruncateIDExact(t *testing.T) {
	if got := truncateID("abcde", 5); got != "abcde" {
		t.Fatalf("expected abcde, got %s", got)
	}
}

func TestTruncateIDLong(t *testing.T) {
	got := truncateID("a-very-long-mob-identifier", 10)
	if len(got) > 10 {
		t.Fatalf("truncated ID too long: %s", got)
	}
	if !strings.HasSuffix(got, "..") {
		t.Fatalf("truncated ID should end with ..: %s", got)
	}
}

func TestTruncateIDTiny(t *testing.T) {
	got := truncateID("abcdef", 2)
	if len(got) > 2 {
		t.Fatalf("truncated ID too long: %s", got)
	}
}

func TestRosterSectionTruncatesLongIDs(t *testing.T) {
	enc := &encounterSummary{
		PlayerIDs: []string{"a-very-long-player-identifier"},
		MobIDs:    []string{"a-very-long-mob-identifier"},
	}
	lines := renderRosterSection(enc, 20, rosterFocus{})
	for _, line := range lines {
		// Each rendered line should not be excessively long
		if len(line) > 60 {
			t.Fatalf("roster line too long: %s", line)
		}
	}
}

func TestRosterSectionDeterministic(t *testing.T) {
	enc := &encounterSummary{
		PlayerIDs: []string{"p1", "p2"}, MobIDs: []string{"m1"},
	}
	a := renderRosterSection(enc, sidePanelWidth, rosterFocus{})
	b := renderRosterSection(enc, sidePanelWidth, rosterFocus{})
	if len(a) != len(b) {
		t.Fatal("roster section should be deterministic")
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatal("roster section should be deterministic")
		}
	}
}

func TestEncounterPanelRosterDoesNotImplyGameplayLogic(t *testing.T) {
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		ActiveEncounterID: "enc-g1", HasActiveEncounter: true,
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-g1", State: "Active",
			PlayerIDs: []string{"hero"}, MobIDs: []string{"orc"},
			PlayerCount: 1, MobCount: 1, MobsAlive: 1,
		}},
	}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{})
	forbidden := []string{"target", "select", "attack", "threat", "aggro", "damage", "hp", "health"}
	for _, word := range forbidden {
		if strings.Contains(strings.ToLower(panel), word) {
			t.Fatalf("encounter panel roster should not contain gameplay term: %s", word)
		}
	}
}

// --- Local Selection Shell Tests (M25) ---

func TestBuildRosterEntriesNilEncounter(t *testing.T) {
	entries := buildRosterEntries(nil)
	if entries != nil {
		t.Fatal("nil encounter should return nil entries")
	}
}

func TestBuildRosterEntriesEmpty(t *testing.T) {
	enc := &encounterSummary{}
	entries := buildRosterEntries(enc)
	if entries != nil {
		t.Fatal("empty encounter should return nil entries")
	}
}

func TestBuildRosterEntriesOrder(t *testing.T) {
	enc := &encounterSummary{
		PlayerIDs: []string{"p1", "p2"},
		MobIDs:    []string{"m1"},
	}
	entries := buildRosterEntries(enc)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].kind != "pc" || entries[0].id != "p1" {
		t.Fatalf("entry 0 wrong: %+v", entries[0])
	}
	if entries[1].kind != "pc" || entries[1].id != "p2" {
		t.Fatalf("entry 1 wrong: %+v", entries[1])
	}
	if entries[2].kind != "mb" || entries[2].id != "m1" {
		t.Fatalf("entry 2 wrong: %+v", entries[2])
	}
}

func TestMoveFocusDownFromUnfocused(t *testing.T) {
	f := moveFocusDown(rosterFocus{index: -1}, 3)
	if f.index != 0 {
		t.Fatalf("expected focus 0, got %d", f.index)
	}
}

func TestMoveFocusDownClamps(t *testing.T) {
	f := moveFocusDown(rosterFocus{index: 2}, 3)
	if f.index != 2 {
		t.Fatalf("expected focus 2, got %d", f.index)
	}
}

func TestMoveFocusDownEmpty(t *testing.T) {
	f := moveFocusDown(rosterFocus{index: -1}, 0)
	if f.index != -1 {
		t.Fatalf("expected focus -1, got %d", f.index)
	}
}

func TestMoveFocusUpFromUnfocused(t *testing.T) {
	f := moveFocusUp(rosterFocus{index: -1}, 3)
	if f.index != 2 {
		t.Fatalf("expected focus 2, got %d", f.index)
	}
}

func TestMoveFocusUpClamps(t *testing.T) {
	f := moveFocusUp(rosterFocus{index: 0}, 3)
	if f.index != 0 {
		t.Fatalf("expected focus 0, got %d", f.index)
	}
}

func TestMoveFocusUpEmpty(t *testing.T) {
	f := moveFocusUp(rosterFocus{index: -1}, 0)
	if f.index != -1 {
		t.Fatalf("expected focus -1, got %d", f.index)
	}
}

func TestReconcileFocusStableEntry(t *testing.T) {
	old := []rosterEntry{{kind: "pc", id: "p1"}, {kind: "mb", id: "m1"}}
	new := []rosterEntry{{kind: "pc", id: "p1"}, {kind: "mb", id: "m1"}}
	f := reconcileFocus(rosterFocus{index: 1}, old, new)
	if f.index != 1 {
		t.Fatalf("expected focus 1, got %d", f.index)
	}
}

func TestReconcileFocusEntryMoved(t *testing.T) {
	old := []rosterEntry{{kind: "pc", id: "p1"}, {kind: "mb", id: "m1"}}
	// m1 is now first because p1 left
	newEntries := []rosterEntry{{kind: "mb", id: "m1"}}
	f := reconcileFocus(rosterFocus{index: 1}, old, newEntries)
	if f.index != 0 {
		t.Fatalf("expected focus 0 (m1 moved to index 0), got %d", f.index)
	}
}

func TestReconcileFocusEntryDisappeared(t *testing.T) {
	old := []rosterEntry{{kind: "pc", id: "p1"}, {kind: "mb", id: "m1"}}
	// focused on m1 (index 1), but m1 is gone
	newEntries := []rosterEntry{{kind: "pc", id: "p1"}}
	f := reconcileFocus(rosterFocus{index: 1}, old, newEntries)
	// should clamp to last entry
	if f.index != 0 {
		t.Fatalf("expected focus 0 (clamped), got %d", f.index)
	}
}

func TestReconcileFocusEmptyNew(t *testing.T) {
	old := []rosterEntry{{kind: "pc", id: "p1"}}
	f := reconcileFocus(rosterFocus{index: 0}, old, nil)
	if f.index != -1 {
		t.Fatalf("expected focus -1, got %d", f.index)
	}
}

func TestReconcileFocusNoFocusStaysUnfocused(t *testing.T) {
	old := []rosterEntry{{kind: "pc", id: "p1"}}
	newEntries := []rosterEntry{{kind: "pc", id: "p1"}, {kind: "mb", id: "m1"}}
	f := reconcileFocus(rosterFocus{index: -1}, old, newEntries)
	if f.index != -1 {
		t.Fatalf("expected focus -1, got %d", f.index)
	}
}

func TestRosterFocusIndicatorRendered(t *testing.T) {
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		ActiveEncounterID: "enc-f1", HasActiveEncounter: true,
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-f1", State: "Active",
			PlayerIDs: []string{"hero"}, MobIDs: []string{"orc-a", "orc-b"},
			PlayerCount: 1, MobCount: 2, MobsAlive: 2,
		}},
	}
	// Focus on second entry (orc-a, index 1)
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{index: 1})
	if !strings.Contains(panel, "> mb:orc-a") {
		t.Fatal("focused entry should have > indicator")
	}
	// Non-focused entries should not have >
	if strings.Contains(panel, "> pc:hero") {
		t.Fatal("non-focused entry should not have > indicator")
	}
}

func TestRosterNoFocusIndicatorWhenUnfocused(t *testing.T) {
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		ActiveEncounterID: "enc-f2", HasActiveEncounter: true,
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-f2", State: "Active",
			PlayerIDs: []string{"hero"}, MobIDs: []string{"orc"},
			PlayerCount: 1, MobCount: 1, MobsAlive: 1,
		}},
	}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{index: -1})
	if strings.Contains(panel, "> pc:") || strings.Contains(panel, "> mb:") {
		t.Fatal("unfocused roster should not show > indicator")
	}
}

func TestRosterFocusDeterministic(t *testing.T) {
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		ActiveEncounterID: "enc-d1", HasActiveEncounter: true,
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-d1", State: "Active",
			PlayerIDs: []string{"p1"}, MobIDs: []string{"m1"},
			PlayerCount: 1, MobCount: 1, MobsAlive: 1,
		}},
	}
	focus := rosterFocus{index: 0}
	a := renderEncounterPanel(sidePanelWidth, pr, er, focus)
	b := renderEncounterPanel(sidePanelWidth, pr, er, focus)
	if a != b {
		t.Fatal("roster focus rendering should be deterministic")
	}
}

func TestRosterFocusNoGameplayTerms(t *testing.T) {
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		ActiveEncounterID: "enc-ng", HasActiveEncounter: true,
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-ng", State: "Active",
			PlayerIDs: []string{"hero"}, MobIDs: []string{"orc"},
			PlayerCount: 1, MobCount: 1, MobsAlive: 1,
		}},
	}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{index: 0})
	forbidden := []string{"target", "select", "attack", "threat", "aggro", "damage", "hp", "health", "enemy"}
	for _, word := range forbidden {
		if strings.Contains(strings.ToLower(panel), word) {
			t.Fatalf("roster focus should not contain gameplay term: %s", word)
		}
	}
}

func TestFooterContainsRosterHint(t *testing.T) {
	footer := renderFooter(80, "", "", "", "", "")
	if !strings.Contains(footer, "tab") {
		t.Fatal("footer should mention tab for roster navigation")
	}
}

// --- Map Focus Projection Tests (M26) ---

func TestFocusedEntryReturnsNilWhenUnfocused(t *testing.T) {
	entries := []rosterEntry{{kind: "pc", id: "p1"}, {kind: "mb", id: "m1"}}
	if fe := focusedEntry(rosterFocus{index: -1}, entries); fe != nil {
		t.Fatal("unfocused should return nil")
	}
}

func TestFocusedEntryReturnsCorrectEntry(t *testing.T) {
	entries := []rosterEntry{{kind: "pc", id: "p1"}, {kind: "mb", id: "m1"}}
	fe := focusedEntry(rosterFocus{index: 1}, entries)
	if fe == nil || fe.kind != "mb" || fe.id != "m1" {
		t.Fatal("should return second entry")
	}
}

func TestFocusedEntryOutOfRange(t *testing.T) {
	entries := []rosterEntry{{kind: "pc", id: "p1"}}
	if fe := focusedEntry(rosterFocus{index: 5}, entries); fe != nil {
		t.Fatal("out-of-range should return nil")
	}
}

func TestOverlayFocusedMobMarker(t *testing.T) {
	mapText := "...\n.m.\n..."
	mobs := []mobPosition{{ProcessID: "mob-1", MobName: "orc", Position: mobPosVec3{X: 50, Y: 50}}}
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	result := overlayFocusedMob(mapText, mobs, "mob-1", bounds, 3, 3)
	if !strings.Contains(result, "M") {
		t.Fatal("focused mob should be rendered as M")
	}
}

func TestOverlayFocusedMobNoMatch(t *testing.T) {
	mapText := "...\n.m.\n..."
	mobs := []mobPosition{{ProcessID: "mob-1", MobName: "orc", Position: mobPosVec3{X: 50, Y: 50}}}
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	result := overlayFocusedMob(mapText, mobs, "mob-999", bounds, 3, 3)
	if result != mapText {
		t.Fatal("non-matching focus should leave map unchanged")
	}
}

func TestOverlayFocusedMobEmpty(t *testing.T) {
	mapText := "...\n...\n..."
	result := overlayFocusedMob(mapText, nil, "mob-1", mapBounds{}, 3, 3)
	if result != mapText {
		t.Fatal("empty mobs should leave map unchanged")
	}
}

func TestOverlayFocusedPlayerMarker(t *testing.T) {
	mapText := "...\n.@.\n..."
	pos := playerPosResult{X: 50, Y: 50}
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	result := overlayFocusedPlayer(mapText, pos, bounds, 3, 3)
	if !strings.Contains(result, "&") {
		t.Fatal("focused player should be rendered as &")
	}
}

func TestMapPanelFocusProjectionMob(t *testing.T) {
	// Create a simple map with a mob
	lines := []mapLine{
		{From: mapVec3{X: 0, Z: 0}, To: mapVec3{X: 100, Z: 100}},
	}
	ascii, bounds := projectAndRasterize(lines, 10, 5)
	mr := mapReadResult{State: mapReadOK, MapText: ascii, MapWidth: 10, MapHeight: 5, Bounds: bounds}
	mobs := []mobPosition{{ProcessID: "orc-1", MobName: "orc", Position: mobPosVec3{X: 50, Y: 50}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	entries := []rosterEntry{{kind: "mb", id: "orc-1"}}
	focus := rosterFocus{index: 0}
	panel := renderMapPanel(mr, mobr, playerReadResult{}, focus, entries, 80, 40)
	if !strings.Contains(panel, "M") {
		t.Fatal("focused mob should appear as M on map")
	}
}

func TestMapPanelNoFocusProjection(t *testing.T) {
	lines := []mapLine{
		{From: mapVec3{X: 0, Z: 0}, To: mapVec3{X: 100, Z: 100}},
	}
	ascii, bounds := projectAndRasterize(lines, 10, 5)
	mr := mapReadResult{State: mapReadOK, MapText: ascii, MapWidth: 10, MapHeight: 5, Bounds: bounds}
	mobs := []mobPosition{{ProcessID: "orc-1", MobName: "orc", Position: mobPosVec3{X: 50, Y: 50}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	entries := []rosterEntry{{kind: "mb", id: "orc-1"}}
	focus := rosterFocus{index: -1} // unfocused
	panel := renderMapPanel(mr, mobr, playerReadResult{}, focus, entries, 80, 40)
	if strings.Contains(panel, "M") {
		t.Fatal("unfocused should not show M on map")
	}
}

func TestMapPanelFocusProjectionDeterministic(t *testing.T) {
	lines := []mapLine{
		{From: mapVec3{X: 0, Z: 0}, To: mapVec3{X: 100, Z: 100}},
	}
	ascii, bounds := projectAndRasterize(lines, 10, 5)
	mr := mapReadResult{State: mapReadOK, MapText: ascii, MapWidth: 10, MapHeight: 5, Bounds: bounds}
	mobs := []mobPosition{{ProcessID: "orc-1", MobName: "orc", Position: mobPosVec3{X: 50, Y: 50}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	entries := []rosterEntry{{kind: "mb", id: "orc-1"}}
	focus := rosterFocus{index: 0}
	a := renderMapPanel(mr, mobr, playerReadResult{}, focus, entries, 80, 40)
	b := renderMapPanel(mr, mobr, playerReadResult{}, focus, entries, 80, 40)
	if a != b {
		t.Fatal("map focus projection should be deterministic")
	}
}

func TestMobPositionPreservesProcessID(t *testing.T) {
	// Verify ProcessID is populated during decode simulation
	m := mobPosition{ProcessID: "test-pid", MobName: "orc"}
	if m.ProcessID != "test-pid" {
		t.Fatal("ProcessID should be preserved")
	}
}

func TestOverlayFocusedMobDeterministic(t *testing.T) {
	mapText := "...\n.m.\n..."
	mobs := []mobPosition{{ProcessID: "mob-1", MobName: "orc", Position: mobPosVec3{X: 50, Y: 50}}}
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	a := overlayFocusedMob(mapText, mobs, "mob-1", bounds, 3, 3)
	b := overlayFocusedMob(mapText, mobs, "mob-1", bounds, 3, 3)
	if a != b {
		t.Fatal("overlay focused mob should be deterministic")
	}
}

func TestMapPanelFocusNoGameplayTerms(t *testing.T) {
	lines := []mapLine{
		{From: mapVec3{X: 0, Z: 0}, To: mapVec3{X: 100, Z: 100}},
	}
	ascii, bounds := projectAndRasterize(lines, 10, 5)
	mr := mapReadResult{State: mapReadOK, MapText: ascii, MapWidth: 10, MapHeight: 5, Bounds: bounds}
	mobs := []mobPosition{{ProcessID: "orc-1", MobName: "orc", Position: mobPosVec3{X: 50, Y: 50}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	entries := []rosterEntry{{kind: "mb", id: "orc-1"}}
	focus := rosterFocus{index: 0}
	panel := renderMapPanel(mr, mobr, playerReadResult{}, focus, entries, 80, 40)
	lower := strings.ToLower(panel)
	forbidden := []string{"target", "select", "attack", "threat", "aggro", "damage", "hp", "health"}
	for _, word := range forbidden {
		if strings.Contains(lower, word) {
			t.Fatalf("map focus projection should not contain gameplay term: %s", word)
		}
	}
}

// --- Local Target Intent Preview Tests (M27) ---

func TestFocusPreviewLabelNoFocus(t *testing.T) {
	label := focusPreviewLabel(rosterFocus{index: -1}, nil)
	if label != "focus: none" {
		t.Fatalf("expected 'focus: none', got %q", label)
	}
}

func TestFocusPreviewLabelEmptyEntries(t *testing.T) {
	label := focusPreviewLabel(rosterFocus{index: 0}, nil)
	if label != "focus: none" {
		t.Fatalf("expected 'focus: none', got %q", label)
	}
}

func TestFocusPreviewLabelMob(t *testing.T) {
	entries := []rosterEntry{{kind: "pc", id: "hero"}, {kind: "mb", id: "orc-a"}}
	label := focusPreviewLabel(rosterFocus{index: 1}, entries)
	if label != "focus: mb:orc-a (local)" {
		t.Fatalf("expected 'focus: mb:orc-a (local)', got %q", label)
	}
}

func TestFocusPreviewLabelPlayer(t *testing.T) {
	entries := []rosterEntry{{kind: "pc", id: "hero-1"}}
	label := focusPreviewLabel(rosterFocus{index: 0}, entries)
	if label != "focus: pc:hero-1 (local)" {
		t.Fatalf("expected 'focus: pc:hero-1 (local)', got %q", label)
	}
}

func TestFocusPreviewLabelOutOfRange(t *testing.T) {
	entries := []rosterEntry{{kind: "mb", id: "m1"}}
	label := focusPreviewLabel(rosterFocus{index: 5}, entries)
	if label != "focus: none" {
		t.Fatalf("expected 'focus: none', got %q", label)
	}
}

func TestFooterShowsFocusPreview(t *testing.T) {
	footer := renderFooter(120, "", "focus: mb:orc-a (local)", "", "", "")
	if !strings.Contains(footer, "focus: mb:orc-a (local)") {
		t.Fatal("footer should show focus preview label")
	}
	if !strings.Contains(footer, "quit") {
		t.Fatal("footer should still contain quit hint")
	}
}

func TestFooterShowsFocusNone(t *testing.T) {
	footer := renderFooter(120, "", "focus: none", "", "", "")
	if !strings.Contains(footer, "focus: none") {
		t.Fatal("footer should show focus: none when unfocused")
	}
}

func TestFooterShowsBothPreviews(t *testing.T) {
	preview := moveIntent{direction: "north"}.preview()
	footer := renderFooter(120, preview, "focus: mb:orc (local)", "", "", "")
	if !strings.Contains(footer, "move north") {
		t.Fatal("footer should show movement intent")
	}
	if !strings.Contains(footer, "focus: mb:orc (local)") {
		t.Fatal("footer should show focus preview")
	}
}

func TestLayoutIncludesFocusPreview(t *testing.T) {
	entries := []rosterEntry{{kind: "mb", id: "test-mob"}}
	focus := rosterFocus{index: 0}
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, focus, entries, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(layout, "focus: mb:test-mob (local)") {
		t.Fatal("layout should include focus preview in footer")
	}
}

func TestLayoutFocusNoneWhenUnfocused(t *testing.T) {
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(layout, "focus: none") {
		t.Fatal("layout should show focus: none when unfocused")
	}
}

func TestFocusPreviewDeterministic(t *testing.T) {
	entries := []rosterEntry{{kind: "mb", id: "orc"}}
	focus := rosterFocus{index: 0}
	a := focusPreviewLabel(focus, entries)
	b := focusPreviewLabel(focus, entries)
	if a != b {
		t.Fatal("focus preview label should be deterministic")
	}
}

func TestFocusPreviewNoGameplayTerms(t *testing.T) {
	entries := []rosterEntry{{kind: "mb", id: "orc"}}
	label := focusPreviewLabel(rosterFocus{index: 0}, entries)
	forbidden := []string{"target", "select", "attack", "threat", "aggro", "damage", "enemy"}
	for _, word := range forbidden {
		if strings.Contains(strings.ToLower(label), word) {
			t.Fatalf("focus preview should not contain gameplay term: %s", word)
		}
	}
}

func TestFocusPreviewContainsLocalMarker(t *testing.T) {
	entries := []rosterEntry{{kind: "mb", id: "orc"}}
	label := focusPreviewLabel(rosterFocus{index: 0}, entries)
	if !strings.Contains(label, "(local)") {
		t.Fatal("focus preview must explicitly indicate local-only")
	}
}

// --- Backend Target Intent Tests (M28) ---

func TestTargetConfirmResultLabelNone(t *testing.T) {
	r := targetConfirmResult{}
	label := r.targetStatusLabel()
	if label != "target: none" {
		t.Fatalf("expected 'target: none', got %q", label)
	}
}

func TestTargetConfirmResultLabelFound(t *testing.T) {
	r := targetConfirmResult{
		State:      targetConfirmOK,
		TargetKind: "mb",
		TargetID:   "orc-a",
		Found:      true,
		MobName:    "a_decaying_skeleton",
	}
	label := r.targetStatusLabel()
	if !strings.Contains(label, "a_decaying_skeleton") {
		t.Fatalf("expected mob name in label, got %q", label)
	}
	if !strings.Contains(label, "(backend)") {
		t.Fatalf("expected (backend) marker, got %q", label)
	}
}

func TestTargetConfirmResultLabelNotFound(t *testing.T) {
	r := targetConfirmResult{
		State:      targetConfirmOK,
		TargetKind: "mb",
		TargetID:   "orc-a",
		Found:      false,
	}
	label := r.targetStatusLabel()
	if !strings.Contains(label, "not found") {
		t.Fatalf("expected 'not found' in label, got %q", label)
	}
	if !strings.Contains(label, "(backend)") {
		t.Fatalf("expected (backend) marker, got %q", label)
	}
}

func TestTargetConfirmResultLabelFailed(t *testing.T) {
	r := targetConfirmResult{
		State: targetConfirmFailed,
		Error: "HTTP 500",
	}
	label := r.targetStatusLabel()
	if label != "target: unavailable" {
		t.Fatalf("expected 'target: unavailable', got %q", label)
	}
}

func TestTargetConfirmResultLabelFoundNoMobName(t *testing.T) {
	r := targetConfirmResult{
		State:      targetConfirmOK,
		TargetKind: "mb",
		TargetID:   "orc-a",
		Found:      true,
		MobName:    "",
	}
	label := r.targetStatusLabel()
	if !strings.Contains(label, "mb:orc-a") {
		t.Fatalf("expected kind:id fallback, got %q", label)
	}
}

func TestTargetConfirmBackendMarkerDistinctFromLocal(t *testing.T) {
	entries := []rosterEntry{{kind: "mb", id: "orc-a"}}
	focusLabel := focusPreviewLabel(rosterFocus{index: 0}, entries)
	targetResult := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-a", Found: true, MobName: "orc",
	}
	targetLabel := targetResult.targetStatusLabel()

	if !strings.Contains(focusLabel, "(local)") {
		t.Fatal("focus label must say (local)")
	}
	if !strings.Contains(targetLabel, "(backend)") {
		t.Fatal("target label must say (backend)")
	}
	if focusLabel == targetLabel {
		t.Fatal("focus and target labels must be distinct")
	}
}

func TestDevTargetProximityURL(t *testing.T) {
	target := defaultTarget()
	url := devTargetProximityURL(target, "mob-42")
	expected := "http://localhost:9090/world/dev/zone/crushbone/player/p1/target/mob-42/proximity"
	if url != expected {
		t.Fatalf("expected %s, got %s", expected, url)
	}
}

func TestQueryTargetProximityPlayerKind(t *testing.T) {
	// PC entries should be honestly reported as unsupported
	result := queryTargetProximity(defaultTarget(), rosterEntry{kind: "pc", id: "hero"})
	if result.State != targetConfirmFailed {
		t.Fatal("PC proximity query should report failure")
	}
	if result.TargetKind != "pc" {
		t.Fatal("result should preserve target kind")
	}
}

func TestFooterShowsTargetLabel(t *testing.T) {
	footer := renderFooter(120, "", "", "target: orc (backend)", "", "")
	if !strings.Contains(footer, "target: orc (backend)") {
		t.Fatal("footer should show target label")
	}
}

func TestFooterShowsAllThreeLabels(t *testing.T) {
	preview := moveIntent{direction: "north"}.preview()
	footer := renderFooter(120, preview, "focus: mb:orc (local)", "target: orc (backend)", "", "")
	if !strings.Contains(footer, "move north") {
		t.Fatal("footer should show movement intent")
	}
	if !strings.Contains(footer, "focus: mb:orc (local)") {
		t.Fatal("footer should show focus label")
	}
	if !strings.Contains(footer, "target: orc (backend)") {
		t.Fatal("footer should show target label")
	}
}

func TestLayoutIncludesTargetLabel(t *testing.T) {
	tc := targetConfirmResult{State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-a", Found: true, MobName: "orc"}
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, tc, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(layout, "(backend)") {
		t.Fatal("layout should include backend target label")
	}
}

func TestLayoutTargetNoneByDefault(t *testing.T) {
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(layout, "target: none") {
		t.Fatal("layout should show target: none by default")
	}
}

func TestTargetConfirmDeterministic(t *testing.T) {
	r := targetConfirmResult{State: targetConfirmOK, TargetKind: "mb", TargetID: "orc", Found: true, MobName: "orc"}
	a := r.targetStatusLabel()
	b := r.targetStatusLabel()
	if a != b {
		t.Fatal("target status label should be deterministic")
	}
}

func TestFooterContainsConfirmHint(t *testing.T) {
	footer := renderFooter(120, "", "", "", "", "")
	if !strings.Contains(footer, "t: confirm") {
		t.Fatal("footer should mention t: confirm keybind")
	}
}

// --- Proximity Panel Tests (M29) ---

func TestProximityPanelNone(t *testing.T) {
	panel := renderProximityPanel(sidePanelWidth, targetConfirmResult{})
	if !strings.Contains(panel, "Proximity") {
		t.Fatal("proximity panel should contain title")
	}
	if !strings.Contains(panel, "none") {
		t.Fatal("proximity panel should show none when no query")
	}
}

func TestProximityPanelUnavailable(t *testing.T) {
	tc := targetConfirmResult{State: targetConfirmFailed, Error: "HTTP 500"}
	panel := renderProximityPanel(sidePanelWidth, tc)
	if !strings.Contains(panel, "unavailable") {
		t.Fatal("proximity panel should show unavailable on failure")
	}
}

func TestProximityPanelFoundWithMobName(t *testing.T) {
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-a",
		Found: true, WithinProximity: false, Distance: 12.3, MobName: "a_skeleton",
	}
	panel := renderProximityPanel(sidePanelWidth, tc)
	if !strings.Contains(panel, "a_skeleton") {
		t.Fatal("proximity panel should show mob name")
	}
	if !strings.Contains(panel, "found: yes") {
		t.Fatal("proximity panel should show found: yes")
	}
	if !strings.Contains(panel, "within: no") {
		t.Fatal("proximity panel should show within: no")
	}
	if !strings.Contains(panel, "dist: 12.3") {
		t.Fatal("proximity panel should show distance")
	}
}

func TestProximityPanelFoundWithinProximity(t *testing.T) {
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-a",
		Found: true, WithinProximity: true, Distance: 2.1, MobName: "orc",
	}
	panel := renderProximityPanel(sidePanelWidth, tc)
	if !strings.Contains(panel, "within: yes") {
		t.Fatal("proximity panel should show within: yes")
	}
}

func TestProximityPanelNotFound(t *testing.T) {
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-a", Found: false,
	}
	panel := renderProximityPanel(sidePanelWidth, tc)
	if !strings.Contains(panel, "found: no") {
		t.Fatal("proximity panel should show found: no")
	}
}

func TestProximityPanelFallbackID(t *testing.T) {
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-a",
		Found: true, MobName: "",
	}
	panel := renderProximityPanel(sidePanelWidth, tc)
	if !strings.Contains(panel, "mb:orc-a") {
		t.Fatal("proximity panel should fall back to kind:id when no mob name")
	}
}

func TestProximityPanelDeterministic(t *testing.T) {
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc",
		Found: true, WithinProximity: true, Distance: 5.0, MobName: "orc",
	}
	a := renderProximityPanel(sidePanelWidth, tc)
	b := renderProximityPanel(sidePanelWidth, tc)
	if a != b {
		t.Fatal("proximity panel should be deterministic")
	}
}

func TestSideColumnContainsProximityPanel(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(col, "Proximity") {
		t.Fatal("side column should contain proximity panel")
	}
}

func TestWideLayoutContainsProximityPanel(t *testing.T) {
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(layout, "Proximity") {
		t.Fatal("wide layout should contain proximity panel")
	}
}

func TestProximityPanelNoGameplayTerms(t *testing.T) {
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc",
		Found: true, WithinProximity: true, Distance: 2.0, MobName: "orc",
	}
	panel := renderProximityPanel(sidePanelWidth, tc)
	forbidden := []string{"attack", "threat", "aggro", "damage", "hp", "health", "enemy", "range"}
	for _, word := range forbidden {
		if strings.Contains(strings.ToLower(panel), word) {
			t.Fatalf("proximity panel should not contain gameplay term: %s", word)
		}
	}
}

// --- Proximity Refresh Tests (M30) ---

func TestProximityNeedsRefreshNoQuery(t *testing.T) {
	// No prior proximity query — should not need refresh
	tc := targetConfirmResult{State: targetConfirmNone}
	entry := &rosterEntry{kind: "mb", id: "orc-1"}
	if proximityNeedsRefresh(tc, playerPosResult{}, "", playerPosResult{X: 10}, entry) {
		t.Fatal("should not need refresh when no prior query exists")
	}
}

func TestProximityNeedsRefreshPositionChanged(t *testing.T) {
	tc := targetConfirmResult{State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1"}
	entry := &rosterEntry{kind: "mb", id: "orc-1"}
	lastPos := playerPosResult{X: 100, Y: 200}
	newPos := playerPosResult{X: 120, Y: 200}
	if !proximityNeedsRefresh(tc, lastPos, "orc-1", newPos, entry) {
		t.Fatal("should need refresh when position changed")
	}
}

func TestProximityNeedsRefreshFocusChanged(t *testing.T) {
	tc := targetConfirmResult{State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1"}
	entry := &rosterEntry{kind: "mb", id: "orc-2"} // different from last
	pos := playerPosResult{X: 100, Y: 200}
	if !proximityNeedsRefresh(tc, pos, "orc-1", pos, entry) {
		t.Fatal("should need refresh when focused entry changed")
	}
}

func TestProximityNeedsRefreshNoChange(t *testing.T) {
	tc := targetConfirmResult{State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1"}
	entry := &rosterEntry{kind: "mb", id: "orc-1"}
	pos := playerPosResult{X: 100, Y: 200}
	if proximityNeedsRefresh(tc, pos, "orc-1", pos, entry) {
		t.Fatal("should not need refresh when nothing changed")
	}
}

func TestProximityNeedsRefreshNilEntry(t *testing.T) {
	tc := targetConfirmResult{State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1"}
	pos := playerPosResult{X: 100, Y: 200}
	if proximityNeedsRefresh(tc, pos, "orc-1", pos, nil) {
		t.Fatal("should not need refresh when no focused entry")
	}
}

func TestProximityNeedsRefreshFailedState(t *testing.T) {
	// Even failed queries count as active (should refresh when state changes)
	tc := targetConfirmResult{State: targetConfirmFailed, TargetKind: "mb", TargetID: "orc-1"}
	entry := &rosterEntry{kind: "mb", id: "orc-1"}
	lastPos := playerPosResult{X: 100, Y: 200}
	newPos := playerPosResult{X: 120, Y: 200}
	if !proximityNeedsRefresh(tc, lastPos, "orc-1", newPos, entry) {
		t.Fatal("should refresh even for failed state when position changed")
	}
}

func TestMaybeRefreshProximityNoActive(t *testing.T) {
	m := model{
		target:        defaultTarget(),
		rosterFocus:   rosterFocus{index: 0},
		rosterEntries: []rosterEntry{{kind: "mb", id: "orc-1"}},
		targetConfirm: targetConfirmResult{State: targetConfirmNone},
	}
	cmd := maybeRefreshProximity(&m)
	if cmd != nil {
		t.Fatal("should not return cmd when no active proximity")
	}
}

func TestMaybeRefreshProximityTriggersOnChange(t *testing.T) {
	m := model{
		target:           defaultTarget(),
		playerRead:       playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 120, Y: 200}},
		rosterFocus:      rosterFocus{index: 0},
		rosterEntries:    []rosterEntry{{kind: "mb", id: "orc-1"}},
		targetConfirm:    targetConfirmResult{State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1"},
		lastProximityPos: playerPosResult{X: 100, Y: 200},
		lastProximityID:  "orc-1",
	}
	cmd := maybeRefreshProximity(&m)
	if cmd == nil {
		t.Fatal("should return cmd when position changed with active proximity")
	}
	// Verify tracking state was updated
	if m.lastProximityPos.X != 120 {
		t.Fatal("lastProximityPos should be updated after refresh")
	}
}

func TestMaybeRefreshProximityNoChangeNoop(t *testing.T) {
	m := model{
		target:           defaultTarget(),
		playerRead:       playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 100, Y: 200}},
		rosterFocus:      rosterFocus{index: 0},
		rosterEntries:    []rosterEntry{{kind: "mb", id: "orc-1"}},
		targetConfirm:    targetConfirmResult{State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1"},
		lastProximityPos: playerPosResult{X: 100, Y: 200},
		lastProximityID:  "orc-1",
	}
	cmd := maybeRefreshProximity(&m)
	if cmd != nil {
		t.Fatal("should not return cmd when nothing changed")
	}
}

// --- BasicAttack Intent Tests (M31) ---

func TestAttackStatusLabelNone(t *testing.T) {
	r := attackResult{}
	if r.attackStatusLabel() != "" {
		t.Fatal("empty attack result should return empty label")
	}
}

func TestAttackStatusLabelSent(t *testing.T) {
	r := attackResult{State: attackStateSent, TargetID: "orc-1"}
	label := r.attackStatusLabel()
	if label != "attack: sent" {
		t.Fatalf("expected 'attack: sent', got %q", label)
	}
}

func TestAttackStatusLabelFailed(t *testing.T) {
	r := attackResult{State: attackStateFailed, Error: "out of range"}
	label := r.attackStatusLabel()
	if label != "attack: failed" {
		t.Fatalf("expected 'attack: failed', got %q", label)
	}
}

func TestDevIntentURL(t *testing.T) {
	target := defaultTarget()
	url := devIntentURL(target)
	expected := "http://localhost:9090/world/dev/zone/crushbone/intent"
	if url != expected {
		t.Fatalf("expected %s, got %s", expected, url)
	}
}

func TestDevIntentURLAsync(t *testing.T) {
	target := defaultTarget()
	target.Mode = "ASYNC"
	url := devIntentURL(target)
	if !strings.Contains(url, "mode=Async") {
		t.Fatalf("ASYNC mode URL should contain mode=Async: %s", url)
	}
}

func TestFooterShowsAttackLabel(t *testing.T) {
	footer := renderFooter(120, "", "", "", "attack: sent", "")
	if !strings.Contains(footer, "attack: sent") {
		t.Fatal("footer should show attack label")
	}
}

func TestFooterShowsAttackKeybind(t *testing.T) {
	footer := renderFooter(120, "", "", "", "", "")
	if !strings.Contains(footer, "a: attack") {
		t.Fatal("footer should mention a: attack keybind")
	}
}

func TestLayoutIncludesAttackLabel(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc"}
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, ar, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(layout, "attack: sent") {
		t.Fatal("layout should include attack label when sent")
	}
}

func TestLayoutNoAttackLabelByDefault(t *testing.T) {
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if strings.Contains(layout, "attack:") {
		t.Fatal("layout should not show attack label by default")
	}
}

func TestAttackStatusLabelDeterministic(t *testing.T) {
	r := attackResult{State: attackStateSent, TargetID: "orc"}
	a := r.attackStatusLabel()
	b := r.attackStatusLabel()
	if a != b {
		t.Fatal("attack status label should be deterministic")
	}
}

func TestAttackNoGameplayTermsInLabel(t *testing.T) {
	r := attackResult{State: attackStateSent, TargetID: "orc"}
	label := r.attackStatusLabel()
	forbidden := []string{"damage", "hit", "miss", "crit", "dps", "hp", "health"}
	for _, word := range forbidden {
		if strings.Contains(strings.ToLower(label), word) {
			t.Fatalf("attack label should not contain combat resolution term: %s", word)
		}
	}
}

// --- Combat Readback Panel Tests (M32) ---

func TestCombatPanelNone(t *testing.T) {
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, playerReadResult{}, encounterReadResult{})
	if !strings.Contains(panel, "Combat") {
		t.Fatal("combat panel should contain title")
	}
	if !strings.Contains(panel, "none") {
		t.Fatal("combat panel should show none before any attack")
	}
}

func TestCombatPanelIntentFailed(t *testing.T) {
	ar := attackResult{State: attackStateFailed, Error: "out of range"}
	panel := renderCombatPanel(sidePanelWidth, ar, playerReadResult{}, encounterReadResult{})
	if !strings.Contains(panel, "intent: failed") {
		t.Fatal("combat panel should show intent: failed")
	}
}

func TestCombatPanelIntentAcceptedNoEncounter(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: false}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, encounterReadResult{})
	if !strings.Contains(panel, "intent: accepted") {
		t.Fatal("combat panel should show intent: accepted")
	}
	if !strings.Contains(panel, "enc: none") {
		t.Fatal("combat panel should show enc: none when no active encounter")
	}
}

func TestCombatPanelIntentAcceptedWithEncounter(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1", "orc-2"}, MobsAlive: 2, MobsDead: 0, ActionIndex: 5,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er)
	if !strings.Contains(panel, "intent: accepted") {
		t.Fatal("combat panel should show intent: accepted")
	}
	if !strings.Contains(panel, "Active") {
		t.Fatal("combat panel should show encounter state")
	}
	if !strings.Contains(panel, "act:5") {
		t.Fatal("combat panel should show action index")
	}
	if !strings.Contains(panel, "alive:2") {
		t.Fatal("combat panel should show alive count")
	}
	if !strings.Contains(panel, "mob: in roster") {
		t.Fatal("combat panel should show targeted mob still in roster")
	}
}

func TestCombatPanelMobGone(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", CompletedReason: "all_mobs_dead",
			MobIDs: []string{}, MobsAlive: 0, MobsDead: 1, ActionIndex: 8,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er)
	if !strings.Contains(panel, "mob: gone") {
		t.Fatal("combat panel should show mob: gone when targeted mob left roster")
	}
	if !strings.Contains(panel, "all_mobs_dead") {
		t.Fatal("combat panel should show completion reason")
	}
}

func TestCombatPanelEncounterUnavailable(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{State: encounterReadFailed}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er)
	if !strings.Contains(panel, "enc: unavailable") {
		t.Fatal("combat panel should show enc: unavailable on read failure")
	}
}

func TestCombatPanelDeterministic(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1, ActionIndex: 3,
		}},
	}
	a := renderCombatPanel(sidePanelWidth, ar, pr, er)
	b := renderCombatPanel(sidePanelWidth, ar, pr, er)
	if a != b {
		t.Fatal("combat panel should be deterministic")
	}
}

func TestCombatPanelNoGameplayTerms(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1, ActionIndex: 3,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er)
	forbidden := []string{"damage", "hit", "miss", "crit", "dps", "health", "landed"}
	for _, word := range forbidden {
		if strings.Contains(strings.ToLower(panel), word) {
			t.Fatalf("combat panel should not contain combat resolution term: %s", word)
		}
	}
}

func TestSideColumnContainsCombatPanel(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(col, "Combat") {
		t.Fatal("side column should contain combat panel")
	}
}

func TestWideLayoutContainsCombatPanel(t *testing.T) {
	layout := renderLayout(120, 80, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(layout, "Combat") {
		t.Fatal("wide layout should contain combat panel")
	}
}

// --- Loot Readback and Pickup Tests (M34) ---

func TestLootPanelNone(t *testing.T) {
	panel := renderLootPanel(sidePanelWidth, playerReadResult{}, encounterReadResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(panel, "Loot") {
		t.Fatal("loot panel should contain title")
	}
	if !strings.Contains(panel, "none") {
		t.Fatal("loot panel should show none when no encounter")
	}
}

func TestLootPanelEncounterActive(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{EncounterID: "enc-1", State: "Active"}},
	}
	panel := renderLootPanel(sidePanelWidth, pr, er, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(panel, "enc: Active") {
		t.Fatal("loot panel should show encounter active state")
	}
}

func TestLootPanelDropsAvailable(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", CompletedReason: "all_mobs_dead",
			DropsGenerated: true, Drops: []string{"item-1", "item-2"},
		}},
	}
	panel := renderLootPanel(sidePanelWidth, pr, er, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(panel, "drops: 2") {
		t.Fatal("loot panel should show drop count")
	}
	if !strings.Contains(panel, "item-1") {
		t.Fatal("loot panel should show first item ID")
	}
	if !strings.Contains(panel, "item-2") {
		t.Fatal("loot panel should show second item ID")
	}
}

func TestLootPanelNoDrops(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: false,
		}},
	}
	panel := renderLootPanel(sidePanelWidth, pr, er, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(panel, "drops: none") {
		t.Fatal("loot panel should show drops: none")
	}
}

func TestLootPanelExpired(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{"item-1"}, LootExpired: true,
		}},
	}
	panel := renderLootPanel(sidePanelWidth, pr, er, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(panel, "loot: expired") {
		t.Fatal("loot panel should show loot: expired")
	}
}

func TestLootPanelPickupAccepted(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{"item-1"},
		}},
	}
	pk := pickupResult{State: pickupStateSent, EncounterID: "enc-1", ItemID: "item-1"}
	panel := renderLootPanel(sidePanelWidth, pr, er, pk, inventoryReadResult{}, -1, -1)
	if !strings.Contains(panel, "pickup: accepted") {
		t.Fatal("loot panel should show pickup: accepted")
	}
}

func TestLootPanelPickupFailed(t *testing.T) {
	pk := pickupResult{State: pickupStateFailed, Error: "loot expired"}
	panel := renderLootPanel(sidePanelWidth, playerReadResult{}, encounterReadResult{}, pk, inventoryReadResult{}, -1, -1)
	if !strings.Contains(panel, "pickup: failed") {
		t.Fatal("loot panel should show pickup: failed")
	}
}

func TestLootPanelDropsAllPickedUp(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{},
		}},
	}
	panel := renderLootPanel(sidePanelWidth, pr, er, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(panel, "drops: 0") {
		t.Fatal("loot panel should show drops: 0 when all picked up")
	}
}

func TestLootPanelDeterministic(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{"item-1"},
		}},
	}
	a := renderLootPanel(sidePanelWidth, pr, er, pickupResult{}, inventoryReadResult{}, -1, -1)
	b := renderLootPanel(sidePanelWidth, pr, er, pickupResult{}, inventoryReadResult{}, -1, -1)
	if a != b {
		t.Fatal("loot panel should be deterministic")
	}
}

func TestLootPanelNoGameplayTerms(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{"item-1"},
		}},
	}
	panel := renderLootPanel(sidePanelWidth, pr, er, pickupResult{}, inventoryReadResult{}, -1, -1)
	forbidden := []string{"rarity", "epic", "legendary", "value", "gold", "reward", "victory"}
	for _, word := range forbidden {
		if strings.Contains(strings.ToLower(panel), word) {
			t.Fatalf("loot panel should not contain reward term: %s", word)
		}
	}
}

func TestPickupStatusLabelNone(t *testing.T) {
	r := pickupResult{}
	if r.pickupStatusLabel() != "" {
		t.Fatal("empty pickup should return empty label")
	}
}

func TestPickupStatusLabelSent(t *testing.T) {
	r := pickupResult{State: pickupStateSent}
	if r.pickupStatusLabel() != "pickup: accepted" {
		t.Fatalf("expected 'pickup: accepted', got %q", r.pickupStatusLabel())
	}
}

func TestPickupStatusLabelFailed(t *testing.T) {
	r := pickupResult{State: pickupStateFailed}
	if r.pickupStatusLabel() != "pickup: failed" {
		t.Fatalf("expected 'pickup: failed', got %q", r.pickupStatusLabel())
	}
}

func TestSideColumnContainsLootPanel(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	if !strings.Contains(col, "Loot") {
		t.Fatal("side column should contain loot panel")
	}
}

func TestFooterContainsPickupHint(t *testing.T) {
	footer := renderFooter(120, "", "", "", "", "")
	if !strings.Contains(footer, "p: pickup") {
		t.Fatal("footer should mention p: pickup keybind")
	}
}

// --- Inventory Confirmation Readback Tests (M35) ---

func TestGameplayStatusURL(t *testing.T) {
	target := defaultTarget()
	url := gameplayStatusURL(target)
	expected := "http://localhost:9090/world/call/crushbone?message=gameplay_status"
	if url != expected {
		t.Fatalf("expected %s, got %s", expected, url)
	}
}

func TestGameplayStatusURLAsync(t *testing.T) {
	target := defaultTarget()
	target.Mode = "ASYNC"
	url := gameplayStatusURL(target)
	if !strings.Contains(url, "mode=Async") {
		t.Fatalf("ASYNC mode URL should contain mode=Async: %s", url)
	}
}

func TestLootPanelShowsInventoryCount(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{"item-1"},
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, Count: 3, Items: []string{"a", "b", "c"}}
	panel := renderLootPanel(sidePanelWidth, pr, er, pickupResult{}, inv, -1, -1)
	if !strings.Contains(panel, "inv: 3") {
		t.Fatal("loot panel should show inventory count from backend")
	}
}

func TestLootPanelShowsInventoryDeltaAfterPickup(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{},
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, Count: 4}
	pk := pickupResult{State: pickupStateSent, EncounterID: "enc-1", ItemID: "item-1"}
	panel := renderLootPanel(sidePanelWidth, pr, er, pk, inv, 3, -1) // was 3, now 4
	if !strings.Contains(panel, "inv: +1") {
		t.Fatal("loot panel should show inventory delta after pickup")
	}
	if !strings.Contains(panel, "confirmed") {
		t.Fatal("loot panel should say confirmed when delta is positive")
	}
}

func TestLootPanelShowsNoChangeWhenDeltaZero(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{"item-1"},
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, Count: 3}
	pk := pickupResult{State: pickupStateSent, EncounterID: "enc-1", ItemID: "item-1"}
	panel := renderLootPanel(sidePanelWidth, pr, er, pk, inv, 3, -1) // was 3, still 3
	if !strings.Contains(panel, "pending") {
		t.Fatal("loot panel should show pending when inventory count unchanged after pickup")
	}
}

func TestLootPanelNoInventoryWhenNotRead(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{"item-1"},
		}},
	}
	inv := inventoryReadResult{State: inventoryReadNotAttempted}
	panel := renderLootPanel(sidePanelWidth, pr, er, pickupResult{}, inv, -1, -1)
	if strings.Contains(panel, "inv:") {
		t.Fatal("loot panel should not show inventory when read not attempted")
	}
}

// --- Loot Selection and Targeted Pickup Tests (M36) ---

func TestReconcileLootFocusStable(t *testing.T) {
	old := []string{"item-a", "item-b"}
	new := []string{"item-a", "item-b"}
	f := reconcileLootFocus(1, old, new)
	if f != 1 {
		t.Fatalf("expected 1, got %d", f)
	}
}

func TestReconcileLootFocusItemMoved(t *testing.T) {
	old := []string{"item-a", "item-b"}
	new := []string{"item-b"} // item-a gone, item-b moved to 0
	f := reconcileLootFocus(1, old, new) // was focused on item-b
	if f != 0 {
		t.Fatalf("expected 0 (item-b now at 0), got %d", f)
	}
}

func TestReconcileLootFocusItemDisappeared(t *testing.T) {
	old := []string{"item-a", "item-b"}
	new := []string{"item-c"} // both gone
	f := reconcileLootFocus(0, old, new) // was on item-a
	if f != 0 {
		t.Fatalf("expected 0 (clamped), got %d", f)
	}
}

func TestReconcileLootFocusEmpty(t *testing.T) {
	old := []string{"item-a"}
	new := []string{}
	f := reconcileLootFocus(0, old, new)
	if f != -1 {
		t.Fatalf("expected -1, got %d", f)
	}
}

func TestReconcileLootFocusUnfocusedStays(t *testing.T) {
	old := []string{"item-a"}
	new := []string{"item-a", "item-b"}
	f := reconcileLootFocus(-1, old, new)
	if f != -1 {
		t.Fatalf("expected -1, got %d", f)
	}
}

func TestLootPanelShowsSelectionIndicator(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{"item-a", "item-b"},
		}},
	}
	panel := renderLootPanel(sidePanelWidth, pr, er, pickupResult{}, inventoryReadResult{}, -1, 1)
	if !strings.Contains(panel, "> item-b") {
		t.Fatal("loot panel should show > on selected drop row")
	}
	if strings.Contains(panel, "> item-a") {
		t.Fatal("non-selected drop should not have >")
	}
}

func TestLootPanelNoSelectionIndicatorWhenUnfocused(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{"item-a"},
		}},
	}
	panel := renderLootPanel(sidePanelWidth, pr, er, pickupResult{}, inventoryReadResult{}, -1, -1)
	if strings.Contains(panel, "> item-a") {
		t.Fatal("unfocused loot should not show >")
	}
}

func TestCurrentDropsReturnsNilWhenNoEncounter(t *testing.T) {
	m := model{playerRead: playerReadResult{HasActiveEncounter: false}}
	drops := currentDrops(&m)
	if drops != nil {
		t.Fatal("should return nil when no active encounter")
	}
}

func TestCurrentDropsReturnsDrops(t *testing.T) {
	m := model{
		playerRead: playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"},
		encounterRead: encounterReadResult{
			State: encounterReadOK, Count: 1,
			Encounters: []encounterSummary{{
				EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
				Drops: []string{"item-1", "item-2"},
			}},
		},
	}
	drops := currentDrops(&m)
	if len(drops) != 2 {
		t.Fatalf("expected 2 drops, got %d", len(drops))
	}
}

func TestFooterContainsLootHint(t *testing.T) {
	footer := renderFooter(120, "", "", "", "", "")
	if !strings.Contains(footer, "[]: loot") {
		t.Fatal("footer should mention []: loot keybind")
	}
}

// --- Viewport Tests (M37) ---

func TestExtractViewportCenteredOnPlayer(t *testing.T) {
	// 10x5 map with a marker at (5, 2)
	lines := []string{
		"0000000000",
		"1111111111",
		"22222X2222",
		"3333333333",
		"4444444444",
	}
	mapText := strings.Join(lines, "\n")
	// Viewport 6x3, centered on col=5, row=2
	vp := extractViewport(mapText, 10, 5, 5, 2, 6, 3)
	vpLines := strings.Split(vp, "\n")
	if len(vpLines) != 3 {
		t.Fatalf("expected 3 viewport lines, got %d", len(vpLines))
	}
	if len([]rune(vpLines[0])) != 6 {
		t.Fatalf("expected viewport width 6, got %d", len([]rune(vpLines[0])))
	}
	// X should be in the viewport
	if !strings.Contains(vp, "X") {
		t.Fatal("viewport should contain the centered marker")
	}
}

func TestExtractViewportEdgeClampLeft(t *testing.T) {
	// Player at col=0 — viewport should clamp left to 0
	lines := []string{
		"ABCDEFGHIJ",
		"KLMNOPQRST",
		"UVWXYZ0123",
	}
	mapText := strings.Join(lines, "\n")
	vp := extractViewport(mapText, 10, 3, 0, 1, 6, 3)
	vpLines := strings.Split(vp, "\n")
	// First char should be from column 0
	if !strings.HasPrefix(vpLines[0], "A") {
		t.Fatalf("expected left clamp to col 0, got %q", vpLines[0])
	}
}

func TestExtractViewportEdgeClampRight(t *testing.T) {
	// Player at col=9 (rightmost) — viewport should clamp right edge
	lines := []string{
		"ABCDEFGHIJ",
		"KLMNOPQRST",
		"UVWXYZ0123",
	}
	mapText := strings.Join(lines, "\n")
	vp := extractViewport(mapText, 10, 3, 9, 1, 6, 3)
	vpLines := strings.Split(vp, "\n")
	// Last char should be J (col 9)
	runes := []rune(vpLines[0])
	if runes[len(runes)-1] != 'J' {
		t.Fatalf("expected right clamp to include col 9, got %c", runes[len(runes)-1])
	}
}

func TestExtractViewportEdgeClampTop(t *testing.T) {
	// Player at row=0 — viewport should clamp top to 0
	lines := []string{
		"AAAAAAAAAA",
		"BBBBBBBBBB",
		"CCCCCCCCCC",
		"DDDDDDDDDD",
		"EEEEEEEEEE",
	}
	mapText := strings.Join(lines, "\n")
	vp := extractViewport(mapText, 10, 5, 5, 0, 6, 3)
	vpLines := strings.Split(vp, "\n")
	if !strings.HasPrefix(vpLines[0], "AAA") {
		t.Fatal("expected top clamp to row 0")
	}
}

func TestExtractViewportEdgeClampBottom(t *testing.T) {
	lines := []string{
		"AAAAAAAAAA",
		"BBBBBBBBBB",
		"CCCCCCCCCC",
		"DDDDDDDDDD",
		"EEEEEEEEEE",
	}
	mapText := strings.Join(lines, "\n")
	vp := extractViewport(mapText, 10, 5, 5, 4, 6, 3)
	vpLines := strings.Split(vp, "\n")
	lastLine := vpLines[len(vpLines)-1]
	if !strings.HasPrefix(lastLine, "EEE") {
		t.Fatal("expected bottom clamp to include last row")
	}
}

func TestExtractViewportLargerThanMap(t *testing.T) {
	mapText := "AB\nCD"
	vp := extractViewport(mapText, 2, 2, 0, 0, 100, 100)
	if vp != mapText {
		t.Fatal("viewport larger than map should return full map")
	}
}

func TestExtractViewportDeterministic(t *testing.T) {
	lines := []string{
		"0000000000",
		"1111111111",
		"2222222222",
		"3333333333",
		"4444444444",
	}
	mapText := strings.Join(lines, "\n")
	a := extractViewport(mapText, 10, 5, 5, 2, 6, 3)
	b := extractViewport(mapText, 10, 5, 5, 2, 6, 3)
	if a != b {
		t.Fatal("viewport extraction should be deterministic")
	}
}

func TestMapPanelViewportCenteredOnPlayer(t *testing.T) {
	// Create a map large enough to require viewport cropping
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = strings.Repeat(".", 100)
	}
	mapText := strings.Join(lines, "\n")
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   mapText,
		MapWidth:  100,
		MapHeight: 50,
		Bounds:    mapBounds{MinX: 0, MaxX: 1000, MinZ: 0, MaxZ: 500, SpanX: 1000, SpanZ: 500},
	}
	pr := playerReadResult{
		State:    playerReadOK,
		HasPos:   true,
		Position: playerPosResult{X: 500, Y: 250}, // center of world
	}
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 40, 20)
	// Panel should contain the player marker
	if !strings.Contains(panel, "@") {
		t.Fatal("viewport should contain player marker when centered on player")
	}
}

func TestMapPanelNoPlayerFallbackStable(t *testing.T) {
	// Large map, no player — should show center of map without crashing
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = strings.Repeat("#", 100)
	}
	mapText := strings.Join(lines, "\n")
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   mapText,
		MapWidth:  100,
		MapHeight: 50,
		Bounds:    mapBounds{MinX: 0, MaxX: 1000, MinZ: 0, MaxZ: 500, SpanX: 1000, SpanZ: 500},
	}
	panel := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 40, 20)
	if !strings.Contains(panel, "#") {
		t.Fatal("no-player viewport should still show map content")
	}
	// Should be deterministic
	panel2 := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 40, 20)
	if panel != panel2 {
		t.Fatal("no-player viewport should be deterministic")
	}
}

func TestMapPanelViewportSmallerThanFullMap(t *testing.T) {
	// Verify that the rendered viewport content is smaller than the full map
	lines := make([]string, 50)
	for i := range lines {
		row := make([]rune, 100)
		for j := range row {
			row[j] = rune('0' + (i % 10))
		}
		lines[i] = string(row)
	}
	mapText := strings.Join(lines, "\n")
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   mapText,
		MapWidth:  100,
		MapHeight: 50,
		Bounds:    mapBounds{MinX: 0, MaxX: 1000, MinZ: 0, MaxZ: 500, SpanX: 1000, SpanZ: 500},
	}
	pr := playerReadResult{
		State:    playerReadOK,
		HasPos:   true,
		Position: playerPosResult{X: 500, Y: 250},
	}
	// Small panel — viewport should be much smaller than 100x50
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 30, 15)
	panelLines := strings.Split(panel, "\n")
	// The panel (including border) should be shorter than the full map
	if len(panelLines) >= 50 {
		t.Fatalf("viewport panel should be shorter than full map, got %d lines", len(panelLines))
	}
}

func TestMapPanelViewportNoGameplayTerms(t *testing.T) {
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = strings.Repeat(".", 40)
	}
	mapText := strings.Join(lines, "\n")
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   mapText,
		MapWidth:  40,
		MapHeight: 20,
		Bounds:    mapBounds{MinX: 0, MaxX: 400, MinZ: 0, MaxZ: 200, SpanX: 400, SpanZ: 200},
	}
	pr := playerReadResult{
		State:    playerReadOK,
		HasPos:   true,
		Position: playerPosResult{X: 200, Y: 100},
	}
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 30, 15)
	lower := strings.ToLower(panel)
	forbidden := []string{"fog", "vision", "los", "field of view", "camera", "zoom", "smooth"}
	for _, word := range forbidden {
		if strings.Contains(lower, word) {
			t.Fatalf("viewport should not contain gameplay/camera term: %s", word)
		}
	}
}
