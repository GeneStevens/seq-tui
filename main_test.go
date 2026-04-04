package main

import (
	"strings"
	"testing"
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
	footer := renderFooter(80, "")
	if !strings.Contains(footer, "quit") {
		t.Fatal("footer should contain quit hint")
	}
}

func TestRenderMapPanelContainsPlayerMarker(t *testing.T) {
	panel := renderMapPanel(mapReadResult{}, mobReadResult{}, playerReadResult{})
	if !strings.ContainsRune(panel, playerMarker) {
		t.Fatal("map panel should contain player marker")
	}
}

func TestRenderLayoutContainsAllSections(t *testing.T) {
	layout := renderLayout(80, 40, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{})
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
	layout := renderLayout(80, 40, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{})
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
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{})
	if !strings.Contains(col, nearbyTitle) {
		t.Fatal("side column should contain nearby title")
	}
	if !strings.Contains(col, statusTitle) {
		t.Fatal("side column should contain status title")
	}
}

func TestWideLayoutContainsPanels(t *testing.T) {
	layout := renderLayout(120, 40, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{})
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
	layout := renderLayout(50, 30, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{})
	if strings.Contains(layout, nearbyTitle) {
		t.Fatal("narrow layout should not contain nearby panel")
	}
	if !strings.ContainsRune(layout, playerMarker) {
		t.Fatal("narrow layout should still contain player marker")
	}
}

func TestRenderLayoutSmallTerminal(t *testing.T) {
	// Should not panic with very small dimensions
	layout := renderLayout(20, 5, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{})
	if len(layout) == 0 {
		t.Fatal("layout should not be empty even for small terminal")
	}
}

func TestRenderLayoutVariousSizes(t *testing.T) {
	sizes := [][2]int{{40, 20}, {80, 40}, {120, 50}, {200, 60}}
	for _, sz := range sizes {
		layout := renderLayout(sz[0], sz[1], "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{})
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
	footer := renderFooter(120, "")
	if !strings.Contains(footer, "move") {
		t.Fatal("footer should advertise movement keys")
	}
	if !strings.Contains(footer, "quit") {
		t.Fatal("footer should still contain quit hint")
	}
}

func TestFooterShowsIntentPreview(t *testing.T) {
	preview := moveIntent{direction: "north"}.preview()
	footer := renderFooter(120, preview)
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
	panel := renderMapPanel(mr, mobReadResult{}, playerReadResult{})
	if !strings.Contains(panel, "###") {
		t.Fatal("map panel should use backend map text when available")
	}
}

func TestMapPanelFallsBackToPlaceholder(t *testing.T) {
	mr := mapReadResult{State: mapReadFailed}
	panel := renderMapPanel(mr, mobReadResult{}, playerReadResult{})
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
	panel := renderMapPanel(mr, mobr, playerReadResult{})
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
	panel := renderMapPanel(mr, mobReadResult{}, pr)
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
