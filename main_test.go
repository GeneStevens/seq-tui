package main

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestMain forces a stable lipgloss color profile for deterministic test output.
// Without this, lipgloss detects non-TTY and strips all ANSI color codes.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	os.Exit(m.Run())
}

// ansiPattern matches ANSI escape sequences (SGR and others).
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes all ANSI escape sequences from a string.
// Use in tests that verify semantic content without caring about styling.
func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

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
	panel := renderMapPanel(mapReadResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 80, 40, attackResult{})
	if !strings.ContainsRune(panel, playerMarker) {
		t.Fatal("map panel should contain player marker")
	}
}

func TestRenderLayoutContainsAllSections(t *testing.T) {
	layout := renderLayout(80, 40, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
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
	layout := renderLayout(80, 40, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
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
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
	if !strings.Contains(col, nearbyTitle) {
		t.Fatal("side column should contain nearby title")
	}
	if !strings.Contains(col, statusTitle) {
		t.Fatal("side column should contain status title")
	}
}

func TestWideLayoutContainsPanels(t *testing.T) {
	layout := renderLayout(120, 40, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
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
	layout := renderLayout(50, 30, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
	if strings.Contains(layout, nearbyTitle) {
		t.Fatal("narrow layout should not contain nearby panel")
	}
	if !strings.ContainsRune(layout, playerMarker) {
		t.Fatal("narrow layout should still contain player marker")
	}
}

func TestRenderLayoutSmallTerminal(t *testing.T) {
	// Should not panic with very small dimensions
	layout := renderLayout(20, 5, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
	if len(layout) == 0 {
		t.Fatal("layout should not be empty even for small terminal")
	}
}

func TestRenderLayoutVariousSizes(t *testing.T) {
	sizes := [][2]int{{40, 20}, {80, 40}, {120, 50}, {200, 60}}
	for _, sz := range sizes {
		layout := renderLayout(sz[0], sz[1], "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
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
	panel := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 80, 40, attackResult{})
	// Strip ANSI to check semantic content — wall chars are individually styled
	if !strings.Contains(stripANSI(panel), "###") {
		t.Fatal("map panel should use backend map text when available")
	}
}

func TestMapPanelFallsBackToPlaceholder(t *testing.T) {
	mr := mapReadResult{State: mapReadFailed}
	panel := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 80, 40, attackResult{})
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
	panel := renderMapPanel(mr, mobr, playerReadResult{}, rosterFocus{}, nil, 80, 40, attackResult{})
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
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 80, 40, attackResult{})
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
	panel := renderEncounterPanel(sidePanelWidth, playerReadResult{}, encounterReadResult{}, rosterFocus{}, "p1")
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
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{}, "p1")
	if !strings.Contains(panel, "unavailable") {
		t.Fatal("encounter panel should show unavailable when encounter read failed")
	}
}

func TestEncounterPanelDataPending(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasPos: true}
	er := encounterReadResult{State: encounterReadNotAttempted}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{}, "p1")
	if !strings.Contains(panel, "pending") {
		t.Fatal("encounter panel should show pending when encounter read not attempted")
	}
}

func TestEncounterPanelNoEncounter(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasPos: true}
	er := encounterReadResult{State: encounterReadOK, Count: 0}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{}, "p1")
	if !strings.Contains(panel, "active: no") {
		t.Fatal("encounter panel should show not in encounter when no active encounter")
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
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{}, "p1")
	if !strings.Contains(panel, "Active") {
		t.Fatal("encounter panel should show active encounter state")
	}
	if !strings.Contains(panel, "Active") {
		t.Fatal("encounter panel should show encounter state")
	}
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "1p/3m") {
		t.Fatalf("encounter panel should show compact counts, got: %s", stripped)
	}
	if !strings.Contains(stripped, "act:7") {
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
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{}, "p1")
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
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{}, "p1")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "no enc details") {
		t.Fatalf("encounter panel should show no details when encounter not found, got: %s", stripped)
	}
}

func TestSideColumnContainsEncounterPanel(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
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
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
	if !strings.Contains(layout, encounterTitle) {
		t.Fatal("wide layout should contain encounter panel")
	}
}

func TestNarrowLayoutOmitsEncounterPanel(t *testing.T) {
	layout := renderLayout(50, 30, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
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
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{}, "p1")
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
	a := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{}, "p1")
	b := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{}, "p1")
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
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{}, "p1")
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
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{}, "p1")
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
	lines := renderRosterSection(enc, 20, rosterFocus{}, "p1")
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
	a := renderRosterSection(enc, sidePanelWidth, rosterFocus{}, "p1")
	b := renderRosterSection(enc, sidePanelWidth, rosterFocus{}, "p1")
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
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{}, "p1")
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
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{index: 1}, "p1")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "> mb:orc-a") {
		t.Fatalf("focused entry should have > indicator, got: %s", stripped)
	}
	// Non-focused entries should not have >
	if strings.Contains(stripped, "> pc:hero") {
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
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{index: -1}, "p1")
	stripped := stripANSI(panel)
	if strings.Contains(stripped, "> pc:") || strings.Contains(stripped, "> mb:") {
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
	a := renderEncounterPanel(sidePanelWidth, pr, er, focus, "p1")
	b := renderEncounterPanel(sidePanelWidth, pr, er, focus, "p1")
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
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{}, "p1")
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
	panel := renderMapPanel(mr, mobr, playerReadResult{}, focus, entries, 80, 40, attackResult{})
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
	panel := renderMapPanel(mr, mobr, playerReadResult{}, focus, entries, 80, 40, attackResult{})
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
	a := renderMapPanel(mr, mobr, playerReadResult{}, focus, entries, 80, 40, attackResult{})
	b := renderMapPanel(mr, mobr, playerReadResult{}, focus, entries, 80, 40, attackResult{})
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
	panel := renderMapPanel(mr, mobr, playerReadResult{}, focus, entries, 80, 40, attackResult{})
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
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, focus, entries, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
	if !strings.Contains(layout, "focus: mb:test-mob (local)") {
		t.Fatal("layout should include focus preview in footer")
	}
}

func TestLayoutFocusNoneWhenUnfocused(t *testing.T) {
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
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
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, tc, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
	if !strings.Contains(layout, "(backend)") {
		t.Fatal("layout should include backend target label")
	}
}

func TestLayoutTargetNoneByDefault(t *testing.T) {
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
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
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
	if !strings.Contains(col, "Proximity") {
		t.Fatal("side column should contain proximity panel")
	}
}

func TestWideLayoutContainsProximityPanel(t *testing.T) {
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
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
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, ar, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
	if !strings.Contains(layout, "attack: sent") {
		t.Fatal("layout should include attack label when sent")
	}
}

func TestLayoutNoAttackLabelByDefault(t *testing.T) {
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
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
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, playerReadResult{}, encounterReadResult{}, defaultTarget(), inventoryReadResult{})
	if !strings.Contains(panel, "Combat") {
		t.Fatal("combat panel should contain title")
	}
	if !strings.Contains(panel, "none") {
		t.Fatal("combat panel should show none before any attack")
	}
}

func TestCombatPanelIntentFailed(t *testing.T) {
	ar := attackResult{State: attackStateFailed, Error: "out of range"}
	panel := renderCombatPanel(sidePanelWidth, ar, playerReadResult{}, encounterReadResult{}, defaultTarget(), inventoryReadResult{})
	if !strings.Contains(panel, "intent: failed") {
		t.Fatal("combat panel should show intent: failed")
	}
}

func TestCombatPanelIntentAcceptedNoEncounter(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: false}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, encounterReadResult{}, defaultTarget(), inventoryReadResult{})
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
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{})
	if !strings.Contains(panel, "Active") {
		t.Fatal("combat panel should show encounter state")
	}
	if !strings.Contains(panel, "act:5") {
		t.Fatal("combat panel should show action index")
	}
	if !strings.Contains(stripANSI(panel), "2a/") {
		t.Fatal("combat panel should show alive count")
	}
	if !strings.Contains(panel, "orc-1") {
		t.Fatal("combat panel should show attack target ID")
	}
	// Attack target should be in the mob roster with > prefix
	if !strings.Contains(stripANSI(panel), "> orc-1") {
		t.Fatal("combat panel should show attack target with > prefix")
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
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{})
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "(dead)") {
		t.Fatalf("combat panel should show (dead) when all_mobs_dead, got: %s", stripped)
	}
	if !strings.Contains(stripped, "all_mobs_dead") {
		t.Fatal("combat panel should show completion reason")
	}
}

func TestCombatPanelEncounterUnavailable(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{State: encounterReadFailed}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{})
	// When encounter read fails, falls through to no-encounter path
	if !strings.Contains(panel, "intent: accepted") {
		t.Fatal("combat panel should show intent status when encounter unavailable")
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
	a := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{})
	b := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{})
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
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{})
	forbidden := []string{"damage", "hit", "miss", "crit", "dps", "health", "landed"}
	for _, word := range forbidden {
		if strings.Contains(strings.ToLower(panel), word) {
			t.Fatalf("combat panel should not contain combat resolution term: %s", word)
		}
	}
}

func TestSideColumnContainsCombatPanel(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
	if !strings.Contains(col, "Combat") {
		t.Fatal("side column should contain combat panel")
	}
}

func TestWideLayoutContainsCombatPanel(t *testing.T) {
	layout := renderLayout(120, 80, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
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
	if !strings.Contains(panel, "loot: none") {
		t.Fatal("loot panel should show loot: none when no drops generated")
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
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "pk:item-1") {
		t.Fatalf("loot panel should show pk:item-id, got: %s", stripped)
	}
}

func TestLootPanelPickupFailed(t *testing.T) {
	pk := pickupResult{State: pickupStateFailed, Error: "loot expired"}
	panel := renderLootPanel(sidePanelWidth, playerReadResult{}, encounterReadResult{}, pk, inventoryReadResult{}, -1, -1)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "pk:fail") {
		t.Fatalf("loot panel should show pk:fail, got: %s", stripped)
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
	if !strings.Contains(panel, "loot: collected") {
		t.Fatal("loot panel should show loot: collected when all picked up")
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
	r := pickupResult{State: pickupStateSent, ItemID: "sword-1"}
	if r.pickupStatusLabel() != "pk:sword-1" {
		t.Fatalf("expected 'pk:sword-1', got %q", r.pickupStatusLabel())
	}
}

func TestPickupStatusLabelFailed(t *testing.T) {
	r := pickupResult{State: pickupStateFailed}
	if r.pickupStatusLabel() != "pk:fail" {
		t.Fatalf("expected 'pk:fail', got %q", r.pickupStatusLabel())
	}
}

func TestSideColumnContainsLootPanel(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
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
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "inv:+1") {
		t.Fatalf("loot panel should show compact inventory delta, got: %s", stripped)
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
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 40, 20, attackResult{})
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
	panel := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 40, 20, attackResult{})
	if !strings.Contains(panel, "#") {
		t.Fatal("no-player viewport should still show map content")
	}
	// Should be deterministic
	panel2 := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 40, 20, attackResult{})
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
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 30, 15, attackResult{})
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
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 30, 15, attackResult{})
	lower := strings.ToLower(panel)
	forbidden := []string{"fog", "vision", "los", "field of view", "camera", "zoom", "smooth"}
	for _, word := range forbidden {
		if strings.Contains(lower, word) {
			t.Fatalf("viewport should not contain gameplay/camera term: %s", word)
		}
	}
}

// --- Colorization Tests (M38) ---

func TestColorizePlayerGlyph(t *testing.T) {
	result := colorizeMapContent("@")
	// Should contain ANSI escape (color code) and the @ character
	if !strings.Contains(result, "\033[") {
		t.Fatal("player glyph should be styled with ANSI escape")
	}
	if !strings.Contains(result, "@") {
		t.Fatal("player glyph character should be preserved")
	}
}

func TestColorizeMobGlyph(t *testing.T) {
	result := colorizeMapContent("m")
	if !strings.Contains(result, "\033[") {
		t.Fatal("mob glyph should be styled with ANSI escape")
	}
	if !strings.Contains(result, "m") {
		t.Fatal("mob glyph character should be preserved")
	}
}

func TestColorizeFocusedMobGlyph(t *testing.T) {
	result := colorizeMapContent("M")
	if !strings.Contains(result, "\033[") {
		t.Fatal("focused mob glyph should be styled with ANSI escape")
	}
	if !strings.Contains(result, "M") {
		t.Fatal("focused mob glyph character should be preserved")
	}
}

func TestColorizeFocusedPlayerGlyph(t *testing.T) {
	result := colorizeMapContent("&")
	if !strings.Contains(result, "\033[") {
		t.Fatal("focused player glyph should be styled with ANSI escape")
	}
	if !strings.Contains(result, "&") {
		t.Fatal("focused player glyph character should be preserved")
	}
}

func TestColorizeWallGlyph(t *testing.T) {
	result := colorizeMapContent("#")
	if !strings.Contains(result, "\033[") {
		t.Fatal("wall glyph should be styled with ANSI escape")
	}
	if !strings.Contains(result, "#") {
		t.Fatal("wall glyph character should be preserved")
	}
}

func TestColorizeEmptySpaceUnstyled(t *testing.T) {
	result := colorizeMapContent(" ")
	if strings.Contains(result, "\033[") {
		t.Fatal("empty space should not be styled")
	}
	if result != " " {
		t.Fatalf("empty space should pass through unchanged, got %q", result)
	}
}

func TestColorizeDotUnstyled(t *testing.T) {
	result := colorizeMapContent(".")
	if strings.Contains(result, "\033[") {
		t.Fatal("dot should not be styled")
	}
	if result != "." {
		t.Fatalf("dot should pass through unchanged, got %q", result)
	}
}

func TestColorizeNewlinePreserved(t *testing.T) {
	result := colorizeMapContent("@\nm")
	if !strings.Contains(result, "\n") {
		t.Fatal("newlines should be preserved in colorized output")
	}
}

func TestColorizeDeterministic(t *testing.T) {
	input := "#.@.m\n#.M.&"
	a := colorizeMapContent(input)
	b := colorizeMapContent(input)
	if a != b {
		t.Fatal("colorization should be deterministic")
	}
}

func TestColorizePlayerDistinctFromMob(t *testing.T) {
	playerStyled := colorizeMapContent("@")
	mobStyled := colorizeMapContent("m")
	// Both should have ANSI but with different escape sequences
	if playerStyled == mobStyled {
		t.Fatal("player and mob should have distinct styling")
	}
}

func TestColorizePreservesLineCount(t *testing.T) {
	input := "###\n.@.\n.m."
	result := colorizeMapContent(input)
	inputLines := strings.Count(input, "\n")
	resultLines := strings.Count(result, "\n")
	if inputLines != resultLines {
		t.Fatalf("colorization should preserve line count: input %d, result %d", inputLines, resultLines)
	}
}

func TestMapPanelColorizedPlayer(t *testing.T) {
	// Backend map with player — panel should contain ANSI escape for colorized player
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = strings.Repeat(".", 20)
	}
	mapText := strings.Join(lines, "\n")
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   mapText,
		MapWidth:  20,
		MapHeight: 10,
		Bounds:    mapBounds{MinX: 0, MaxX: 200, MinZ: 0, MaxZ: 100, SpanX: 200, SpanZ: 100},
	}
	pr := playerReadResult{
		State:    playerReadOK,
		HasPos:   true,
		Position: playerPosResult{X: 100, Y: 50},
	}
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 80, 40, attackResult{})
	if !strings.Contains(panel, "\033[") {
		t.Fatal("map panel with player should contain ANSI color escapes")
	}
	if !strings.Contains(panel, "@") {
		t.Fatal("map panel should contain player marker")
	}
}

func TestMapPanelColorizedMob(t *testing.T) {
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = strings.Repeat(".", 20)
	}
	mapText := strings.Join(lines, "\n")
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   mapText,
		MapWidth:  20,
		MapHeight: 10,
		Bounds:    mapBounds{MinX: 0, MaxX: 200, MinZ: 0, MaxZ: 100, SpanX: 200, SpanZ: 100},
	}
	mobr := mobReadResult{
		State: mobReadOK,
		Mobs:  []mobPosition{{ProcessID: "orc-1", MobName: "orc", Position: mobPosVec3{X: 100, Y: 50}}},
		Count: 1,
	}
	panel := renderMapPanel(mr, mobr, playerReadResult{}, rosterFocus{}, nil, 80, 40, attackResult{})
	if !strings.Contains(panel, "m") {
		t.Fatal("map panel should contain mob marker")
	}
}

func TestColorizeNoGameplayTerms(t *testing.T) {
	input := "#.@.m.M.&"
	result := colorizeMapContent(input)
	lower := strings.ToLower(result)
	forbidden := []string{"threat", "aggro", "health", "hp", "status", "damage", "safe", "danger"}
	for _, word := range forbidden {
		if strings.Contains(lower, word) {
			t.Fatalf("colorized output should not contain gameplay term: %s", word)
		}
	}
}

func TestMapPanelFallbackNoColorize(t *testing.T) {
	// When backend map is not available, static fallback is used.
	// The fallback has its own styling via renderStyledMap — colorize should not be applied.
	mr := mapReadResult{State: mapReadFailed}
	panel := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 80, 40, attackResult{})
	// Should still render (fallback) without crashing
	if panel == "" {
		t.Fatal("fallback panel should not be empty")
	}
	// Fallback should contain player marker from static map
	if !strings.ContainsRune(panel, playerMarker) {
		t.Fatal("fallback panel should contain static player marker")
	}
}

// --- Styling Abstraction Tests (M39) ---

func TestStripANSIRemovesEscapes(t *testing.T) {
	styled := colorizeMapContent("@")
	if !strings.Contains(styled, "\033[") {
		t.Fatal("styled output should contain ANSI escapes")
	}
	stripped := stripANSI(styled)
	if stripped != "@" {
		t.Fatalf("stripANSI should leave plain glyph, got %q", stripped)
	}
}

func TestStripANSIPreservesPlainText(t *testing.T) {
	plain := "hello world"
	if stripANSI(plain) != plain {
		t.Fatal("stripANSI should not alter plain text")
	}
}

func TestStripANSIMixedContent(t *testing.T) {
	styled := colorizeMapContent("#.@.m")
	stripped := stripANSI(styled)
	if stripped != "#.@.m" {
		t.Fatalf("stripANSI should recover original glyphs, got %q", stripped)
	}
}

func TestLipglossStyleProducesANSI(t *testing.T) {
	// With forced ANSI256 profile, lipgloss styles should produce ANSI escapes
	result := playerGlyphStyle.Render("@")
	if !strings.Contains(result, "\033[") {
		t.Fatal("lipgloss style should produce ANSI with forced color profile")
	}
	if !strings.Contains(result, "@") {
		t.Fatal("lipgloss style should preserve the character")
	}
}

func TestLipglossStyleDeterministic(t *testing.T) {
	a := playerGlyphStyle.Render("@")
	b := playerGlyphStyle.Render("@")
	if a != b {
		t.Fatal("lipgloss style rendering should be deterministic")
	}
}

func TestLipglossPlayerDistinctFromMob(t *testing.T) {
	player := playerGlyphStyle.Render("@")
	mob := mobGlyphStyle.Render("m")
	// Extract just the ANSI codes (not the glyph) to compare styling
	playerCodes := ansiPattern.FindAllString(player, -1)
	mobCodes := ansiPattern.FindAllString(mob, -1)
	if len(playerCodes) == 0 || len(mobCodes) == 0 {
		t.Fatal("both styles should produce ANSI codes")
	}
	if playerCodes[0] == mobCodes[0] {
		t.Fatal("player and mob should have distinct ANSI color codes")
	}
}

func TestColorizeRoundtripWithStripANSI(t *testing.T) {
	input := "#.@.m\n.M.&."
	styled := colorizeMapContent(input)
	stripped := stripANSI(styled)
	if stripped != input {
		t.Fatalf("stripANSI(colorize(x)) should recover x, got %q", stripped)
	}
}

// --- Adaptive Spatial Zoom Tests (M40) ---

// testLines returns simple map geometry for testing: a box from (0,0) to (1000,1000)
// with cross-lines through the center so any viewport region contains visible walls.
func testLines() []mapLine {
	return []mapLine{
		// Outer box
		{From: mapVec3{X: 0, Z: 0}, To: mapVec3{X: 1000, Z: 0}},
		{From: mapVec3{X: 1000, Z: 0}, To: mapVec3{X: 1000, Z: 1000}},
		{From: mapVec3{X: 1000, Z: 1000}, To: mapVec3{X: 0, Z: 1000}},
		{From: mapVec3{X: 0, Z: 1000}, To: mapVec3{X: 0, Z: 0}},
		// Cross-lines through center
		{From: mapVec3{X: 0, Z: 500}, To: mapVec3{X: 1000, Z: 500}},
		{From: mapVec3{X: 500, Z: 0}, To: mapVec3{X: 500, Z: 1000}},
	}
}

func testFullBounds() mapBounds {
	return computeBounds(testLines())
}

func TestComputeAdaptiveWorldWindowCentered(t *testing.T) {
	full := testFullBounds()
	win := computeAdaptiveWorldWindow(full, 500, 500, 60, 30)
	// Window should be centered on (500, 500)
	midX := (win.MinX + win.MaxX) / 2
	midZ := (win.MinZ + win.MaxZ) / 2
	if math.Abs(midX-500) > 1 || math.Abs(midZ-500) > 1 {
		t.Fatalf("window should be centered on (500,500), got mid=(%f,%f)", midX, midZ)
	}
	// Window should be smaller than full zone
	if win.SpanX >= full.SpanX {
		t.Fatal("60-cell viewport should show less than full zone width")
	}
	if win.SpanZ >= full.SpanZ {
		t.Fatal("30-cell viewport should show less than full zone height")
	}
}

func TestComputeAdaptiveWorldWindowSmallViewportTighter(t *testing.T) {
	full := testFullBounds()
	winSmall := computeAdaptiveWorldWindow(full, 500, 500, 30, 15)
	winLarge := computeAdaptiveWorldWindow(full, 500, 500, 90, 45)
	// Smaller viewport should show less world
	if winSmall.SpanX >= winLarge.SpanX {
		t.Fatalf("smaller viewport should show tighter world: small=%f large=%f", winSmall.SpanX, winLarge.SpanX)
	}
	if winSmall.SpanZ >= winLarge.SpanZ {
		t.Fatalf("smaller viewport should show tighter world: small=%f large=%f", winSmall.SpanZ, winLarge.SpanZ)
	}
}

func TestComputeAdaptiveWorldWindowEdgeClampLeft(t *testing.T) {
	full := testFullBounds()
	win := computeAdaptiveWorldWindow(full, 0, 500, 60, 30)
	if win.MinX < full.MinX {
		t.Fatal("window should not extend below zone MinX")
	}
}

func TestComputeAdaptiveWorldWindowEdgeClampRight(t *testing.T) {
	full := testFullBounds()
	win := computeAdaptiveWorldWindow(full, 1000, 500, 60, 30)
	if win.MaxX > full.MaxX+0.01 {
		t.Fatalf("window should not extend above zone MaxX: %f > %f", win.MaxX, full.MaxX)
	}
}

func TestComputeAdaptiveWorldWindowDeterministic(t *testing.T) {
	full := testFullBounds()
	a := computeAdaptiveWorldWindow(full, 500, 500, 60, 30)
	b := computeAdaptiveWorldWindow(full, 500, 500, 60, 30)
	if a != b {
		t.Fatal("adaptive world window should be deterministic")
	}
}

func TestComputeAdaptiveWorldWindowFullZoneAtRefSize(t *testing.T) {
	full := testFullBounds()
	// At reference dimensions (240x120), should show full zone
	win := computeAdaptiveWorldWindow(full, 500, 500, 240, 120)
	if math.Abs(win.SpanX-full.SpanX) > 0.01 {
		t.Fatalf("at reference size, should show full zone width: got %f want %f", win.SpanX, full.SpanX)
	}
}

func TestRasterizeAdaptiveViewportProducesCorrectDimensions(t *testing.T) {
	lines := testLines()
	full := testFullBounds()
	ascii, _ := rasterizeAdaptiveViewport(lines, full, 500, 500, 40, 20)
	rows := strings.Split(ascii, "\n")
	if len(rows) != 20 {
		t.Fatalf("expected 20 rows, got %d", len(rows))
	}
	if len([]rune(rows[0])) != 40 {
		t.Fatalf("expected 40 columns, got %d", len([]rune(rows[0])))
	}
}

func TestRasterizeAdaptiveViewportContainsWalls(t *testing.T) {
	lines := testLines()
	full := testFullBounds()
	ascii, _ := rasterizeAdaptiveViewport(lines, full, 500, 500, 60, 30)
	if !strings.Contains(ascii, "#") {
		t.Fatal("adaptive viewport should contain wall characters from geometry")
	}
}

func TestRasterizeAdaptiveViewportDeterministic(t *testing.T) {
	lines := testLines()
	full := testFullBounds()
	a, ba := rasterizeAdaptiveViewport(lines, full, 500, 500, 40, 20)
	b, bb := rasterizeAdaptiveViewport(lines, full, 500, 500, 40, 20)
	if a != b {
		t.Fatal("adaptive rasterization should be deterministic")
	}
	if ba != bb {
		t.Fatal("adaptive bounds should be deterministic")
	}
}

func TestRasterizeAdaptiveViewportEmpty(t *testing.T) {
	full := testFullBounds()
	ascii, _ := rasterizeAdaptiveViewport(nil, full, 500, 500, 40, 20)
	if ascii != "" {
		t.Fatal("empty lines should produce empty output")
	}
}

func TestMapPanelAdaptivePathUsedWithLines(t *testing.T) {
	lines := testLines()
	full := computeBounds(lines)
	// Pre-rasterize at 200x100 as fetchZoneMap does
	ascii, _ := projectAndRasterize(lines, 200, 100)
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   ascii,
		MapWidth:  200,
		MapHeight: 100,
		Bounds:    full,
		Lines:     lines,
	}
	pr := playerReadResult{
		State:    playerReadOK,
		HasPos:   true,
		Position: playerPosResult{X: 500, Y: 500},
	}
	// Small panel to test adaptive tighter view
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 30, 15, attackResult{})
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "@") {
		t.Fatal("adaptive panel should contain player marker")
	}
	if !strings.Contains(stripped, "#") {
		t.Fatal("adaptive panel should contain wall characters")
	}
}

func TestMapPanelAdaptiveNoPlayer(t *testing.T) {
	lines := testLines()
	full := computeBounds(lines)
	ascii, _ := projectAndRasterize(lines, 200, 100)
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   ascii,
		MapWidth:  200,
		MapHeight: 100,
		Bounds:    full,
		Lines:     lines,
	}
	// No player — should center on map midpoint
	panel := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 40, 20, attackResult{})
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "#") {
		t.Fatal("no-player adaptive panel should contain wall characters")
	}
	// Should be deterministic
	panel2 := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 40, 20, attackResult{})
	if panel != panel2 {
		t.Fatal("no-player adaptive panel should be deterministic")
	}
}

func TestMapPanelAdaptiveMobOverlay(t *testing.T) {
	lines := testLines()
	full := computeBounds(lines)
	ascii, _ := projectAndRasterize(lines, 200, 100)
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   ascii,
		MapWidth:  200,
		MapHeight: 100,
		Bounds:    full,
		Lines:     lines,
	}
	mobr := mobReadResult{
		State: mobReadOK,
		Mobs:  []mobPosition{{ProcessID: "orc-1", MobName: "orc", Position: mobPosVec3{X: 500, Y: 500}}},
		Count: 1,
	}
	pr := playerReadResult{
		State:    playerReadOK,
		HasPos:   true,
		Position: playerPosResult{X: 500, Y: 500},
	}
	panel := renderMapPanel(mr, mobr, pr, rosterFocus{}, nil, 80, 40, attackResult{})
	stripped := stripANSI(panel)
	// Mob might overlap player; check at least one marker is visible
	if !strings.Contains(stripped, "@") && !strings.Contains(stripped, "m") {
		t.Fatal("adaptive panel should show player or mob marker")
	}
}

func TestMapPanelAdaptiveSmallVsLargeViewport(t *testing.T) {
	lines := testLines()
	full := computeBounds(lines)
	ascii, _ := projectAndRasterize(lines, 200, 100)
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   ascii,
		MapWidth:  200,
		MapHeight: 100,
		Bounds:    full,
		Lines:     lines,
	}
	pr := playerReadResult{
		State:    playerReadOK,
		HasPos:   true,
		Position: playerPosResult{X: 500, Y: 500},
	}
	// Both should produce valid output
	smallPanel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 20, 10, attackResult{})
	largePanel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 120, 60, attackResult{})
	if smallPanel == "" || largePanel == "" {
		t.Fatal("both small and large panels should produce output")
	}
	// Large panel should have more content lines
	smallLines := strings.Count(smallPanel, "\n")
	largeLines := strings.Count(largePanel, "\n")
	if largeLines <= smallLines {
		t.Fatalf("large panel should have more lines than small: large=%d small=%d", largeLines, smallLines)
	}
}

func TestMapPanelAdaptiveColorization(t *testing.T) {
	lines := testLines()
	full := computeBounds(lines)
	ascii, _ := projectAndRasterize(lines, 200, 100)
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   ascii,
		MapWidth:  200,
		MapHeight: 100,
		Bounds:    full,
		Lines:     lines,
	}
	pr := playerReadResult{
		State:    playerReadOK,
		HasPos:   true,
		Position: playerPosResult{X: 500, Y: 500},
	}
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 60, 30, attackResult{})
	// Should contain ANSI escapes from colorization
	if !strings.Contains(panel, "\033[") {
		t.Fatal("adaptive panel should have colorized output")
	}
}

// --- Combat Target and Engagement Readback Tests (M41) ---

func TestCombatPanelShowsLatestResult(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID:         "enc-1",
			State:               "Active",
			MobIDs:              []string{"orc-1"},
			MobsAlive:           1,
			ActionIndex:         5,
			LatestResultKind:    "damage_applied",
			LatestResultValue:   25,
			LatestResultTarget:  "orc-1",
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{})
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "damage_applied") {
		t.Fatal("combat panel should show latest result kind with res: prefix")
	}
	if !strings.Contains(stripped, "25") {
		t.Fatal("combat panel should show latest result value")
	}
	if !strings.Contains(stripped, "orc-1") {
		t.Fatal("combat panel should show target in mob roster")
	}
}

func TestCombatPanelShowsAttackMiss(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID:      "enc-1",
			State:            "Active",
			MobIDs:           []string{"orc-1"},
			MobsAlive:        1,
			LatestResultKind: "attack_miss",
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{})
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "attack_miss") {
		t.Fatal("combat panel should show attack miss result with res: prefix")
	}
}

func TestCombatPanelShowsEngagedMobs(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1",
			State:       "Active",
			MobIDs:      []string{"orc-1", "orc-2"},
			MobsAlive:   2,
			MobThreat: []mobThreatEntry{
				{MobID: "orc-1", SelectedTargetPlayerID: "p1"},
				{MobID: "orc-2", SelectedTargetPlayerID: "p1"},
			},
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{})
	stripped := stripANSI(panel)
	// Both mobs should be visible with <- suffix indicating engagement
	if !strings.Contains(stripped, "orc-1") {
		t.Fatal("combat panel should show first engaged mob ID")
	}
	if !strings.Contains(stripped, "orc-2") {
		t.Fatal("combat panel should show second engaged mob ID")
	}
	if !strings.Contains(stripped, "<-") {
		t.Fatal("combat panel should show engagement indicator")
	}
}

func TestCombatPanelNoEngagedWhenOtherTarget(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1",
			State:       "Active",
			MobIDs:      []string{"orc-1"},
			MobsAlive:   1,
			MobThreat: []mobThreatEntry{
				{MobID: "orc-1", SelectedTargetPlayerID: "other-player"},
			},
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{})
	if strings.Contains(panel, "engaged") {
		t.Fatal("combat panel should not show engaged mobs when they target another player")
	}
}

func TestCombatPanelShowsTextSummary(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID:       "enc-1",
			State:             "Active",
			MobIDs:            []string{"orc-1"},
			MobsAlive:         1,
			TextSummaryLatest: "p1 attacks orc-1",
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{})
	if !strings.Contains(panel, "p1 attacks") {
		t.Fatal("combat panel should show backend text summary")
	}
}

func TestCombatPanelWithoutEncounterShowsNone(t *testing.T) {
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, playerReadResult{}, encounterReadResult{}, defaultTarget(), inventoryReadResult{})
	if !strings.Contains(panel, "none") {
		t.Fatal("combat panel without encounter should show 'none'")
	}
}

func TestCombatPanelDeterministicWithCombatData(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID:        "enc-1",
			State:              "Active",
			MobIDs:             []string{"orc-1"},
			MobsAlive:          1,
			LatestResultKind:   "damage_applied",
			LatestResultValue:  15,
			LatestResultTarget: "orc-1",
			MobThreat: []mobThreatEntry{
				{MobID: "orc-1", SelectedTargetPlayerID: "p1"},
			},
		}},
	}
	a := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{})
	b := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{})
	if a != b {
		t.Fatal("combat panel with combat data should be deterministic")
	}
}

func TestMobsEngagingPlayerBasic(t *testing.T) {
	enc := &encounterSummary{
		MobThreat: []mobThreatEntry{
			{MobID: "orc-1", SelectedTargetPlayerID: "p1"},
			{MobID: "orc-2", SelectedTargetPlayerID: "p2"},
			{MobID: "orc-3", SelectedTargetPlayerID: "p1"},
		},
	}
	engaged := mobsEngagingPlayer(enc, "p1")
	if len(engaged) != 2 {
		t.Fatalf("expected 2 mobs engaging p1, got %d", len(engaged))
	}
	if engaged[0] != "orc-1" || engaged[1] != "orc-3" {
		t.Fatalf("unexpected engaged mob IDs: %v", engaged)
	}
}

func TestMobsEngagingPlayerNone(t *testing.T) {
	enc := &encounterSummary{
		MobThreat: []mobThreatEntry{
			{MobID: "orc-1", SelectedTargetPlayerID: "other"},
		},
	}
	engaged := mobsEngagingPlayer(enc, "p1")
	if len(engaged) != 0 {
		t.Fatalf("expected 0 mobs engaging p1, got %d", len(engaged))
	}
}

func TestMobsEngagingPlayerNilEncounter(t *testing.T) {
	engaged := mobsEngagingPlayer(nil, "p1")
	if engaged != nil {
		t.Fatal("nil encounter should return nil engaged list")
	}
}

func TestMobsEngagingPlayerEmptyPlayerID(t *testing.T) {
	enc := &encounterSummary{
		MobThreat: []mobThreatEntry{
			{MobID: "orc-1", SelectedTargetPlayerID: "p1"},
		},
	}
	engaged := mobsEngagingPlayer(enc, "")
	if engaged != nil {
		t.Fatal("empty player ID should return nil engaged list")
	}
}

// --- Player Death and Recovery Readback Tests (M42) ---

func TestPlayerPanelNotJoined(t *testing.T) {
	panel := renderPlayerPanel(sidePanelWidth, playerReadResult{}, inventoryReadResult{}, respawnResult{})
	if !strings.Contains(panel, "Player") {
		t.Fatal("player panel should contain title")
	}
	if !strings.Contains(panel, "not joined") {
		t.Fatal("player panel should show 'not joined' when player not joined")
	}
}

func TestPlayerPanelEncounterActive(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	if !strings.Contains(panel, "enc: active") {
		t.Fatal("player panel should show enc: active when in encounter")
	}
}

func TestPlayerPanelEncounterNone(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	if !strings.Contains(panel, "enc: none") {
		t.Fatal("player panel should show enc: none when not in encounter")
	}
}

func TestPlayerPanelShowsHP(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 75, HPMax: 100}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	if !strings.Contains(panel, "hp: 75/100") {
		t.Fatal("player panel should show HP from backend")
	}
}

func TestPlayerPanelShowsDead(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, BlockedReason: "player_dead", HPCurrent: 0, HPMax: 100}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	if !strings.Contains(panel, "hp: 0/100") {
		t.Fatal("player panel should show HP 0")
	}
	if !strings.Contains(panel, "state: dead") {
		t.Fatal("player panel should show dead state when HP is 0")
	}
	if !strings.Contains(panel, "can act: no") {
		t.Fatal("player panel should show cannot act")
	}
	if !strings.Contains(panel, "player_dead") {
		t.Fatal("player panel should show blocked reason from backend")
	}
}

func TestPlayerPanelCanAct(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	if !strings.Contains(panel, "can act: yes") {
		t.Fatal("player panel should show can act: yes")
	}
}

func TestPlayerPanelBlockedWithReason(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, BlockedReason: "cooldown", HPCurrent: 80, HPMax: 100}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	if !strings.Contains(panel, "can act: no") {
		t.Fatal("player panel should show cannot act")
	}
	if !strings.Contains(panel, "cooldown") {
		t.Fatal("player panel should show blocked reason")
	}
}

func TestPlayerPanelGracefulDegradationNoLifecycle(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: false}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	// Should not crash, should show pending status
	if !strings.Contains(panel, "Player") {
		t.Fatal("player panel should render with title even without lifecycle data")
	}
	// Should not contain HP or can-act when lifecycle not available
	if strings.Contains(panel, "hp:") {
		t.Fatal("player panel should not show HP when lifecycle not decoded")
	}
}

func TestPlayerPanelStatusUnavailable(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadFailed}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	if !strings.Contains(panel, "unavailable") {
		t.Fatal("player panel should show unavailable when gameplay status fails")
	}
}

func TestPlayerPanelDeterministic(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, BlockedReason: "player_dead", HPCurrent: 0, HPMax: 100}
	a := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	b := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	if a != b {
		t.Fatal("player panel should be deterministic")
	}
}

func TestPlayerPanelNoGameplayTerms(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	lower := strings.ToLower(panel)
	// Panel should not contain terms that imply client-side combat logic
	forbidden := []string{"respawn", "timer", "countdown", "revive", "resurrect"}
	for _, word := range forbidden {
		if strings.Contains(lower, word) {
			t.Fatalf("player panel should not contain gameplay term: %s", word)
		}
	}
}

func TestSideColumnContainsPlayerPanel(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{})
	if !strings.Contains(col, "Player") {
		t.Fatal("side column should contain player panel")
	}
}

func TestPlayerPanelZeroMaxHP(t *testing.T) {
	// When HPMax is 0, HP line should not appear (field not populated)
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 0, HPMax: 0}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	if strings.Contains(panel, "hp:") {
		t.Fatal("player panel should not show HP when HPMax is 0 (field not populated)")
	}
}

// --- Respawn Action and Confirmation Readback Tests (M43) ---

func TestRespawnStatusLabelNone(t *testing.T) {
	rs := respawnResult{State: respawnStateNone}
	if rs.respawnStatusLabel() != "" {
		t.Fatal("respawnStateNone should produce empty label")
	}
}

func TestRespawnStatusLabelSent(t *testing.T) {
	rs := respawnResult{State: respawnStateSent}
	if rs.respawnStatusLabel() != "respawn: sent" {
		t.Fatalf("expected 'respawn: sent', got %q", rs.respawnStatusLabel())
	}
}

func TestRespawnStatusLabelFailed(t *testing.T) {
	rs := respawnResult{State: respawnStateFailed, Error: "not dead"}
	if rs.respawnStatusLabel() != "respawn: failed" {
		t.Fatalf("expected 'respawn: failed', got %q", rs.respawnStatusLabel())
	}
}

func TestPlayerPanelRespawnSent(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, BlockedReason: "player_dead", HPCurrent: 0, HPMax: 100}
	rs := respawnResult{State: respawnStateSent}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, rs)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "respawn: sent") {
		t.Fatal("player panel should show respawn: sent when submitted but not yet restored")
	}
	if strings.Contains(stripped, "restored") {
		t.Fatal("player panel should not show restored while still dead")
	}
}

func TestPlayerPanelRespawnRestored(t *testing.T) {
	// After respawn: backend shows HP restored and can_act=true
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	rs := respawnResult{State: respawnStateSent}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, rs)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "restored") {
		t.Fatalf("player panel should show restored when backend confirms recovery, got: %q", stripped)
	}
}

func TestPlayerPanelRespawnFailed(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, BlockedReason: "player_dead", HPCurrent: 0, HPMax: 100}
	rs := respawnResult{State: respawnStateFailed, Error: "not dead"}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, rs)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "respawn: failed") {
		t.Fatal("player panel should show respawn: failed")
	}
}

func TestPlayerPanelNoRespawnStatusWhenNone(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	rs := respawnResult{State: respawnStateNone}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, rs)
	if strings.Contains(panel, "respawn") {
		t.Fatal("player panel should not show respawn status when no respawn submitted")
	}
}

func TestPlayerPanelRespawnDeterministic(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, BlockedReason: "player_dead", HPCurrent: 0, HPMax: 100}
	rs := respawnResult{State: respawnStateSent}
	a := renderPlayerPanel(sidePanelWidth, pr, inv, rs)
	b := renderPlayerPanel(sidePanelWidth, pr, inv, rs)
	if a != b {
		t.Fatal("player panel with respawn should be deterministic")
	}
}

func TestPlayerPanelRespawnNoGameplayTerms(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, HPCurrent: 0, HPMax: 100}
	rs := respawnResult{State: respawnStateSent}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, rs)
	lower := strings.ToLower(panel)
	forbidden := []string{"timer", "countdown", "auto", "resurrect", "graveyard"}
	for _, word := range forbidden {
		if strings.Contains(lower, word) {
			t.Fatalf("player panel should not contain gameplay term: %s", word)
		}
	}
}

func TestFooterContainsRespawnHint(t *testing.T) {
	footer := renderFooter(120, "", "", "", "", "")
	if !strings.Contains(footer, "r: respawn") {
		t.Fatal("footer should mention r: respawn keybind")
	}
}

// --- Multi-Mob Combat Roster Clarity Tests (M20260404-01) ---

func TestRenderCombatMobRosterBasic(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{"orc-1", "orc-2", "orc-3"},
	}
	lines := renderCombatMobRoster(enc, attackResult{}, "p1", 20)
	if len(lines) == 0 {
		t.Fatal("roster should produce output with mobs")
	}
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	if !strings.Contains(joined, "orc-1") {
		t.Fatal("roster should show first mob")
	}
	if !strings.Contains(joined, "orc-2") {
		t.Fatal("roster should show second mob")
	}
	if !strings.Contains(joined, "orc-3") {
		t.Fatal("roster should show third mob")
	}
}

func TestRenderCombatMobRosterAttackTarget(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{"orc-1", "orc-2"},
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-2"}
	lines := renderCombatMobRoster(enc, ar, "p1", 20)
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	if !strings.Contains(joined, "> orc-2") {
		t.Fatal("roster should mark attack target with > prefix")
	}
	// orc-1 should NOT have > prefix
	if strings.Contains(joined, "> orc-1") {
		t.Fatal("non-target mob should not have > prefix")
	}
}

func TestRenderCombatMobRosterEngagement(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{"orc-1", "orc-2", "orc-3"},
		MobThreat: []mobThreatEntry{
			{MobID: "orc-1", SelectedTargetPlayerID: "p1"},
			{MobID: "orc-3", SelectedTargetPlayerID: "p1"},
		},
	}
	lines := renderCombatMobRoster(enc, attackResult{}, "p1", 20)
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	// orc-1 and orc-3 should have <- suffix
	if !strings.Contains(joined, "orc-1 <-") {
		t.Fatalf("engaging mob should have <- suffix, got: %s", joined)
	}
	if !strings.Contains(joined, "orc-3 <-") {
		t.Fatal("engaging mob should have <- suffix")
	}
	// orc-2 should NOT have <- suffix
	for _, l := range lines {
		s := stripANSI(l)
		if strings.Contains(s, "orc-2") && strings.Contains(s, "<-") {
			t.Fatal("non-engaging mob should not have <- suffix")
		}
	}
}

func TestRenderCombatMobRosterTargetAndEngaged(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{"orc-1"},
		MobThreat: []mobThreatEntry{
			{MobID: "orc-1", SelectedTargetPlayerID: "p1"},
		},
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	lines := renderCombatMobRoster(enc, ar, "p1", 20)
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	// Should have both > prefix and <- suffix
	if !strings.Contains(joined, "> orc-1") {
		t.Fatal("target mob should have > prefix")
	}
	if !strings.Contains(joined, "<-") {
		t.Fatal("engaging mob should have <- suffix")
	}
}

func TestRenderCombatMobRosterTargetGone(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{"orc-2"}, // orc-1 no longer in roster
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	lines := renderCombatMobRoster(enc, ar, "p1", 20)
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	if !strings.Contains(joined, "orc-1") {
		t.Fatal("gone target should still appear in roster")
	}
	if !strings.Contains(joined, "(gone)") {
		t.Fatal("gone target should be marked (gone)")
	}
}

func TestRenderCombatMobRosterEmptyNoTarget(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{},
	}
	lines := renderCombatMobRoster(enc, attackResult{}, "p1", 20)
	if lines != nil {
		t.Fatal("empty roster with no target should return nil")
	}
}

func TestRenderCombatMobRosterEmptyWithGoneTarget(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{},
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	lines := renderCombatMobRoster(enc, ar, "p1", 20)
	if lines == nil {
		t.Fatal("empty roster with gone target should still show the gone target")
	}
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	if !strings.Contains(joined, "(gone)") {
		t.Fatal("gone target should be marked")
	}
}

func TestRenderCombatMobRosterNilEncounter(t *testing.T) {
	lines := renderCombatMobRoster(nil, attackResult{}, "p1", 20)
	if lines != nil {
		t.Fatal("nil encounter should return nil")
	}
}

func TestRenderCombatMobRosterDeterministic(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{"orc-1", "orc-2", "orc-3"},
		MobThreat: []mobThreatEntry{
			{MobID: "orc-1", SelectedTargetPlayerID: "p1"},
		},
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-2"}
	a := renderCombatMobRoster(enc, ar, "p1", 20)
	b := renderCombatMobRoster(enc, ar, "p1", 20)
	if len(a) != len(b) {
		t.Fatal("mob roster should be deterministic")
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatal("mob roster should be deterministic")
		}
	}
}

func TestRenderCombatMobRosterBackendOrder(t *testing.T) {
	// Verify mobs appear in backend MobIDs order, not sorted
	enc := &encounterSummary{
		MobIDs: []string{"zebra-mob", "alpha-mob", "mid-mob"},
	}
	lines := renderCombatMobRoster(enc, attackResult{}, "p1", 20)
	var mobOrder []string
	for _, l := range lines {
		s := stripANSI(l)
		if strings.Contains(s, "mob") && !strings.Contains(s, "---") {
			mobOrder = append(mobOrder, s)
		}
	}
	if len(mobOrder) != 3 {
		t.Fatalf("expected 3 mob lines, got %d", len(mobOrder))
	}
	if !strings.Contains(mobOrder[0], "zebra") {
		t.Fatal("first mob should be zebra (backend order)")
	}
	if !strings.Contains(mobOrder[1], "alpha") {
		t.Fatal("second mob should be alpha (backend order)")
	}
	if !strings.Contains(mobOrder[2], "mid") {
		t.Fatal("third mob should be mid (backend order)")
	}
}

func TestIsMobEngagingPlayerTrue(t *testing.T) {
	enc := &encounterSummary{
		MobThreat: []mobThreatEntry{
			{MobID: "orc-1", SelectedTargetPlayerID: "p1"},
		},
	}
	if !isMobEngagingPlayer(enc, "orc-1", "p1") {
		t.Fatal("orc-1 should be engaging p1")
	}
}

func TestIsMobEngagingPlayerFalse(t *testing.T) {
	enc := &encounterSummary{
		MobThreat: []mobThreatEntry{
			{MobID: "orc-1", SelectedTargetPlayerID: "p2"},
		},
	}
	if isMobEngagingPlayer(enc, "orc-1", "p1") {
		t.Fatal("orc-1 should not be engaging p1")
	}
}

func TestIsMobEngagingPlayerNil(t *testing.T) {
	if isMobEngagingPlayer(nil, "orc-1", "p1") {
		t.Fatal("nil encounter should return false")
	}
}

func TestCombatPanelFullRosterIntegration(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-2"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1",
			State:       "Active",
			MobIDs:      []string{"orc-1", "orc-2", "orc-3", "orc-4"},
			MobsAlive:   4,
			MobThreat: []mobThreatEntry{
				{MobID: "orc-1", SelectedTargetPlayerID: "p1"},
				{MobID: "orc-2", SelectedTargetPlayerID: "p1"},
				{MobID: "orc-4", SelectedTargetPlayerID: "p1"},
			},
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{})
	stripped := stripANSI(panel)
	// All 4 mobs should be visible
	if !strings.Contains(stripped, "orc-1") {
		t.Fatal("all mobs should be visible")
	}
	if !strings.Contains(stripped, "orc-4") {
		t.Fatal("all mobs should be visible (no +N more cutoff)")
	}
	// Attack target should have > prefix
	if !strings.Contains(stripped, "> orc-2") {
		t.Fatal("attack target should have > prefix")
	}
	// Engaged mobs should have <- suffix
	if !strings.Contains(stripped, "<-") {
		t.Fatal("engaged mobs should have <- suffix")
	}
}

// --- Combat Cadence and Readiness Readback Tests (M20260404-02) ---

func TestCombatPanelShowsReadyYes(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "rdy:yes") {
		t.Fatalf("combat panel should show ready: yes when can_act is true, got: %s", stripped)
	}
}

func TestCombatPanelShowsReadyNo(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, BlockedReason: "cooldown", HPCurrent: 80, HPMax: 100}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inv)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "rdy:no") {
		t.Fatalf("combat panel should show ready: no when blocked, got: %s", stripped)
	}
	if !strings.Contains(stripped, "cooldown") {
		t.Fatal("combat panel should show blocked reason")
	}
}

func TestCombatPanelShowsReadyNoDead(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, BlockedReason: "dead", HPCurrent: 0, HPMax: 100}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inv)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "rdy:no") {
		t.Fatal("combat panel should show ready: no when dead")
	}
	if !strings.Contains(stripped, "dead") {
		t.Fatal("combat panel should show dead as blocked reason")
	}
}

func TestCombatPanelNoReadyWithoutLifecycle(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}
	// No lifecycle data
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: false}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inv)
	stripped := stripANSI(panel)
	if strings.Contains(stripped, "rdy:") {
		t.Fatal("combat panel should not show ready when lifecycle data absent")
	}
}

func TestCombatPanelReadyDistinctFromResult(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID:      "enc-1", State: "Active",
			MobIDs:           []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "damage_applied", LatestResultValue: 25,
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv)
	stripped := stripANSI(panel)
	// Both should be present and distinct
	if !strings.Contains(stripped, "rdy:yes") {
		t.Fatal("should show readiness")
	}
	if !strings.Contains(stripped, "damage_applied") {
		t.Fatal("should show attack result with res: prefix")
	}
}

func TestCombatPanelReadyDeterministic(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, BlockedReason: "cooldown"}
	a := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inv)
	b := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inv)
	if a != b {
		t.Fatal("combat panel with readiness should be deterministic")
	}
}

func TestCombatPanelReadyNoGameplayTerms(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inv)
	lower := strings.ToLower(stripANSI(panel))
	forbidden := []string{"timer", "countdown", "cooldown_remaining", "next attack", "speed"}
	for _, word := range forbidden {
		if strings.Contains(lower, word) {
			t.Fatalf("combat panel should not contain timing term: %s", word)
		}
	}
}

// --- Active Target Spatial Highlight Tests (M20260404-04) ---

func TestOverlayAttackTargetPlacesX(t *testing.T) {
	mapText := "     \n     \n     "
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	mobs := []mobPosition{{ProcessID: "orc-1", Position: mobPosVec3{X: 50, Y: 50}}}
	result := overlayAttackTarget(mapText, mobs, "orc-1", bounds, 5, 3)
	if !strings.Contains(result, "X") {
		t.Fatal("attack target overlay should place X on map")
	}
}

func TestOverlayAttackTargetNoMatch(t *testing.T) {
	mapText := "     \n     \n     "
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	mobs := []mobPosition{{ProcessID: "orc-1", Position: mobPosVec3{X: 50, Y: 50}}}
	result := overlayAttackTarget(mapText, mobs, "orc-99", bounds, 5, 3)
	if strings.Contains(result, "X") {
		t.Fatal("attack target overlay should not place X when target not in mobs")
	}
}

func TestOverlayAttackTargetEmpty(t *testing.T) {
	mapText := "#####"
	result := overlayAttackTarget(mapText, nil, "orc-1", mapBounds{}, 5, 1)
	if result != mapText {
		t.Fatal("empty mobs should return map unchanged")
	}
}

func TestOverlayAttackTargetEmptyID(t *testing.T) {
	mapText := "#####"
	mobs := []mobPosition{{ProcessID: "orc-1", Position: mobPosVec3{X: 50, Y: 50}}}
	result := overlayAttackTarget(mapText, mobs, "", mapBounds{SpanX: 100, SpanZ: 100}, 5, 1)
	if result != mapText {
		t.Fatal("empty target ID should return map unchanged")
	}
}

func TestColorizeAttackTargetGlyph(t *testing.T) {
	result := colorizeMapContent("X")
	if !strings.Contains(result, "\033[") {
		t.Fatal("attack target glyph should be styled with ANSI")
	}
	if !strings.Contains(result, "X") {
		t.Fatal("attack target glyph character should be preserved")
	}
}

func TestColorizeAttackTargetDistinctFromMob(t *testing.T) {
	targetStyled := colorizeMapContent("X")
	mobStyled := colorizeMapContent("m")
	if targetStyled == mobStyled {
		t.Fatal("attack target should have distinct styling from regular mob")
	}
}

func TestMapPanelWithAttackTarget(t *testing.T) {
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = strings.Repeat(".", 20)
	}
	mapText := strings.Join(lines, "\n")
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   mapText,
		MapWidth:  20,
		MapHeight: 10,
		Bounds:    mapBounds{MinX: 0, MaxX: 200, MinZ: 0, MaxZ: 100, SpanX: 200, SpanZ: 100},
	}
	mobr := mobReadResult{
		State: mobReadOK,
		Mobs:  []mobPosition{{ProcessID: "orc-1", MobName: "orc", Position: mobPosVec3{X: 100, Y: 50}}},
		Count: 1,
	}
	pr := playerReadResult{
		State:    playerReadOK,
		HasPos:   true,
		Position: playerPosResult{X: 80, Y: 50},
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	panel := renderMapPanel(mr, mobr, pr, rosterFocus{}, nil, 80, 40, ar)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "X") {
		t.Fatal("map panel should show X for attack target")
	}
}

func TestMapPanelNoAttackTargetWithoutAttack(t *testing.T) {
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = strings.Repeat(".", 20)
	}
	mapText := strings.Join(lines, "\n")
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   mapText,
		MapWidth:  20,
		MapHeight: 10,
		Bounds:    mapBounds{MinX: 0, MaxX: 200, MinZ: 0, MaxZ: 100, SpanX: 200, SpanZ: 100},
	}
	mobr := mobReadResult{
		State: mobReadOK,
		Mobs:  []mobPosition{{ProcessID: "orc-1", MobName: "orc", Position: mobPosVec3{X: 100, Y: 50}}},
		Count: 1,
	}
	panel := renderMapPanel(mr, mobr, playerReadResult{}, rosterFocus{}, nil, 80, 40, attackResult{})
	stripped := stripANSI(panel)
	if strings.Contains(stripped, "X") {
		t.Fatal("map panel should not show X when no attack submitted")
	}
}

func TestOverlayAttackTargetDeterministic(t *testing.T) {
	mapText := "     \n     \n     "
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	mobs := []mobPosition{{ProcessID: "orc-1", Position: mobPosVec3{X: 50, Y: 50}}}
	a := overlayAttackTarget(mapText, mobs, "orc-1", bounds, 5, 3)
	b := overlayAttackTarget(mapText, mobs, "orc-1", bounds, 5, 3)
	if a != b {
		t.Fatal("attack target overlay should be deterministic")
	}
}

// --- Encounter Participant Clarity Tests (M20260404-05) ---

func TestRosterSelfMarker(t *testing.T) {
	enc := &encounterSummary{
		PlayerIDs:   []string{"p1", "p2"},
		MobIDs:      []string{"orc-1"},
		PlayerCount: 2,
		MobCount:    1,
	}
	lines := renderRosterSection(enc, sidePanelWidth, rosterFocus{index: -1}, "p1")
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	if !strings.Contains(joined, "pc:p1*") {
		t.Fatalf("self should be marked with * suffix, got: %s", joined)
	}
	if strings.Contains(joined, "pc:p2*") {
		t.Fatal("other player should not have * suffix")
	}
}

func TestRosterSelfMarkerNotOnMobs(t *testing.T) {
	enc := &encounterSummary{
		PlayerIDs:   []string{"p1"},
		MobIDs:      []string{"p1-mob"}, // mob ID happens to match player — should NOT get *
		PlayerCount: 1,
		MobCount:    1,
	}
	lines := renderRosterSection(enc, sidePanelWidth, rosterFocus{index: -1}, "p1")
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	// Only the pc: entry should have *
	if !strings.Contains(joined, "pc:p1*") {
		t.Fatal("self player should be marked")
	}
	for _, l := range lines {
		s := stripANSI(l)
		if strings.Contains(s, "mb:") && strings.Contains(s, "*") {
			t.Fatal("mob entries should not have * marker")
		}
	}
}

func TestEncounterPanelCompactCounts(t *testing.T) {
	pr := playerReadResult{
		State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			PlayerIDs: []string{"p1"}, MobIDs: []string{"orc-1", "orc-2"},
			PlayerCount: 1, MobCount: 2, MobsAlive: 2, ActionIndex: 3,
		}},
	}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{index: -1}, "p1")
	stripped := stripANSI(panel)
	// State and counts on one line
	if !strings.Contains(stripped, "Active 1p/2m") {
		t.Fatalf("encounter panel should show compact state+counts, got: %s", stripped)
	}
	// Alive/dead + act on one line
	if !strings.Contains(stripped, "2a/0d act:3") {
		t.Fatalf("encounter panel should show compact alive/dead/act, got: %s", stripped)
	}
}

func TestEncounterPanelSelfMarkerIntegration(t *testing.T) {
	pr := playerReadResult{
		State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			PlayerIDs: []string{"p1", "p2"}, MobIDs: []string{"orc-1"},
			PlayerCount: 2, MobCount: 1, MobsAlive: 1,
		}},
	}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{index: -1}, "p1")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "pc:p1*") {
		t.Fatalf("encounter panel should mark self with *, got: %s", stripped)
	}
}

func TestEncounterPanelNoRosterHeader(t *testing.T) {
	pr := playerReadResult{
		State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			PlayerIDs: []string{"p1"}, MobIDs: []string{"orc-1"},
			PlayerCount: 1, MobCount: 1, MobsAlive: 1,
		}},
	}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{index: -1}, "p1")
	stripped := stripANSI(panel)
	if strings.Contains(stripped, "---roster---") {
		t.Fatal("encounter panel should not have ---roster--- header")
	}
}

func TestEncounterPanelCompactDeterministic(t *testing.T) {
	pr := playerReadResult{
		State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			PlayerIDs: []string{"p1"}, MobIDs: []string{"orc-1"},
			PlayerCount: 1, MobCount: 1, MobsAlive: 1,
		}},
	}
	a := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{index: -1}, "p1")
	b := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{index: -1}, "p1")
	if a != b {
		t.Fatal("encounter panel should be deterministic")
	}
}

// --- Attack Attempt Feedback Tightening Tests (M20260404-06) ---

func TestCombatPanelShowsAtkSent(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{})
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "atk:") {
		t.Fatalf("combat panel should show atk:sent when attack submitted, got: %s", stripped)
	}
}

func TestCombatPanelShowsAtkFail(t *testing.T) {
	ar := attackResult{State: attackStateFailed, Error: "no mob"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{})
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "atk:fail") {
		t.Fatalf("combat panel should show atk:fail in encounter, got: %s", stripped)
	}
	if !strings.Contains(stripped, "no mob") {
		t.Fatal("combat panel should show failure reason")
	}
}

func TestCombatPanelNoAtkWithoutAttack(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{})
	stripped := stripANSI(panel)
	if strings.Contains(stripped, "atk:") {
		t.Fatal("combat panel should not show atk: when no attack submitted")
	}
}

func TestCombatPanelAtkAndResultDistinct(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID:      "enc-1", State: "Active",
			MobIDs:           []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "damage_applied", LatestResultValue: 30,
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv)
	stripped := stripANSI(panel)
	// All three should be present and distinct
	if !strings.Contains(stripped, "atk:") {
		t.Fatal("should show submission status")
	}
	if !strings.Contains(stripped, "damage_applied") {
		t.Fatal("should show backend result")
	}
	if !strings.Contains(stripped, "rdy:yes") {
		t.Fatal("should show readiness")
	}
}

// --- Defeated Target Aftermath Clarity Tests (M20260404-07) ---

func TestMobRosterTargetGoneShowsDead(t *testing.T) {
	enc := &encounterSummary{
		MobIDs:          []string{},
		CompletedReason: "all_mobs_dead",
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	lines := renderCombatMobRoster(enc, ar, "p1", 20)
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	if !strings.Contains(joined, "(dead)") {
		t.Fatalf("target gone + all_mobs_dead should show (dead), got: %s", joined)
	}
}

func TestMobRosterTargetGoneShowsGoneWhenActive(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{"orc-2"}, // orc-1 gone but encounter still active
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	lines := renderCombatMobRoster(enc, ar, "p1", 20)
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	if !strings.Contains(joined, "(gone)") {
		t.Fatalf("target gone + active encounter should show (gone), got: %s", joined)
	}
	if strings.Contains(joined, "(dead)") {
		t.Fatal("should not show (dead) when encounter is still active")
	}
}

func TestMobRosterTargetGoneShowsDeadWithPartialRoster(t *testing.T) {
	enc := &encounterSummary{
		MobIDs:          []string{"orc-2"}, // orc-1 gone, orc-2 still alive
		CompletedReason: "all_mobs_dead",
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	lines := renderCombatMobRoster(enc, ar, "p1", 20)
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	if !strings.Contains(joined, "(dead)") {
		t.Fatalf("target gone + all_mobs_dead should show (dead), got: %s", joined)
	}
}

func TestMobRosterTargetPresentNoGoneLabel(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{"orc-1"},
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	lines := renderCombatMobRoster(enc, ar, "p1", 20)
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	if strings.Contains(joined, "(gone)") || strings.Contains(joined, "(dead)") {
		t.Fatal("target still in roster should not have gone/dead label")
	}
}

// --- Joined-Player Spatial Centering Sanity Tests (M20260404-08) ---

func TestPlayerMarkerWinsOverAttackTarget(t *testing.T) {
	// When player and attack target mob are at the same position,
	// the player marker @ should be visible (not overwritten by X)
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = strings.Repeat(".", 20)
	}
	mapText := strings.Join(lines, "\n")
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   mapText,
		MapWidth:  20,
		MapHeight: 10,
		Bounds:    mapBounds{MinX: 0, MaxX: 200, MinZ: 0, MaxZ: 100, SpanX: 200, SpanZ: 100},
	}
	// Mob and player at same position
	mobr := mobReadResult{
		State: mobReadOK,
		Mobs:  []mobPosition{{ProcessID: "orc-1", Position: mobPosVec3{X: 100, Y: 50}}},
		Count: 1,
	}
	pr := playerReadResult{
		State:    playerReadOK,
		HasPos:   true,
		Position: playerPosResult{X: 100, Y: 50}, // same as mob
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	panel := renderMapPanel(mr, mobr, pr, rosterFocus{index: -1}, nil, 80, 40, ar)
	stripped := stripANSI(panel)
	// Player @ should be visible (not hidden by X)
	if !strings.Contains(stripped, "@") {
		t.Fatalf("player marker should be visible when overlapping with attack target, got: %s", stripped)
	}
}

func TestPlayerMarkerWinsOverRegularMob(t *testing.T) {
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = strings.Repeat(".", 20)
	}
	mapText := strings.Join(lines, "\n")
	mr := mapReadResult{
		State:     mapReadOK,
		MapText:   mapText,
		MapWidth:  20,
		MapHeight: 10,
		Bounds:    mapBounds{MinX: 0, MaxX: 200, MinZ: 0, MaxZ: 100, SpanX: 200, SpanZ: 100},
	}
	mobr := mobReadResult{
		State: mobReadOK,
		Mobs:  []mobPosition{{ProcessID: "orc-1", Position: mobPosVec3{X: 100, Y: 50}}},
		Count: 1,
	}
	pr := playerReadResult{
		State:    playerReadOK,
		HasPos:   true,
		Position: playerPosResult{X: 100, Y: 50}, // same as mob
	}
	panel := renderMapPanel(mr, mobr, pr, rosterFocus{index: -1}, nil, 80, 40, attackResult{})
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "@") {
		t.Fatal("player marker should be visible when overlapping with mob")
	}
}

func TestPlayerCenteredInViewport(t *testing.T) {
	// Player should be near the center of the viewport when not edge-clamped
	tl := testLines()
	full := computeBounds(tl)
	ascii, _ := projectAndRasterize(tl, 200, 100)
	mr := mapReadResult{
		State: mapReadOK, MapText: ascii,
		MapWidth: 200, MapHeight: 100,
		Bounds: full, Lines: tl,
	}
	pr := playerReadResult{
		State:    playerReadOK,
		HasPos:   true,
		Position: playerPosResult{X: 500, Y: 500}, // center of 0-1000 zone
	}
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{index: -1}, nil, 40, 20, attackResult{})
	stripped := stripANSI(panel)
	// Find the @ position — it should be roughly in the middle rows
	panelLines := strings.Split(stripped, "\n")
	playerRow := -1
	for i, line := range panelLines {
		if strings.Contains(line, "@") {
			playerRow = i
			break
		}
	}
	if playerRow < 0 {
		t.Fatal("player should be visible in viewport")
	}
	// Player should be in the middle third of the panel (roughly centered)
	middleStart := len(panelLines) / 3
	middleEnd := len(panelLines) * 2 / 3
	if playerRow < middleStart || playerRow > middleEnd {
		t.Fatalf("player should be roughly centered, found at row %d of %d", playerRow, len(panelLines))
	}
}

func TestSpatialOverlayOrderDeterministic(t *testing.T) {
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = strings.Repeat(".", 20)
	}
	mapText := strings.Join(lines, "\n")
	mr := mapReadResult{
		State: mapReadOK, MapText: mapText,
		MapWidth: 20, MapHeight: 10,
		Bounds: mapBounds{MinX: 0, MaxX: 200, MinZ: 0, MaxZ: 100, SpanX: 200, SpanZ: 100},
	}
	mobr := mobReadResult{
		State: mobReadOK,
		Mobs:  []mobPosition{{ProcessID: "orc-1", Position: mobPosVec3{X: 100, Y: 50}}},
		Count: 1,
	}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position: playerPosResult{X: 100, Y: 50},
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	a := renderMapPanel(mr, mobr, pr, rosterFocus{index: -1}, nil, 80, 40, ar)
	b := renderMapPanel(mr, mobr, pr, rosterFocus{index: -1}, nil, 80, 40, ar)
	if a != b {
		t.Fatal("spatial overlay ordering should be deterministic")
	}
}

// --- Target Switching Readback Clarity Tests (M20260404-09) ---

func TestCombatPanelAtkShowsTargetID(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1", "orc-2"}, MobsAlive: 2,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{})
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "atk:orc-1") {
		t.Fatalf("atk line should show target ID, got: %s", stripped)
	}
}

func TestCombatPanelAtkTargetSwitchVisible(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1", "orc-2"}, MobsAlive: 2,
		}},
	}
	// First attack targets orc-1
	ar1 := attackResult{State: attackStateSent, TargetID: "orc-1"}
	panel1 := renderCombatPanel(sidePanelWidth, ar1, pr, er, defaultTarget(), inventoryReadResult{})
	stripped1 := stripANSI(panel1)
	// Second attack targets orc-2
	ar2 := attackResult{State: attackStateSent, TargetID: "orc-2"}
	panel2 := renderCombatPanel(sidePanelWidth, ar2, pr, er, defaultTarget(), inventoryReadResult{})
	stripped2 := stripANSI(panel2)
	// The panels should differ — showing different target IDs
	if stripped1 == stripped2 {
		t.Fatal("panels should differ when attack target changes")
	}
	if !strings.Contains(stripped1, "atk:orc-1") {
		t.Fatal("first panel should show orc-1")
	}
	if !strings.Contains(stripped2, "atk:orc-2") {
		t.Fatal("second panel should show orc-2")
	}
}

// --- Loot Readiness Visibility Tests (M20260404-10) ---

func TestLootPanelShowsLootReady(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{"sword-1", "shield-1"},
		}},
	}
	panel := renderLootPanel(sidePanelWidth, pr, er, pickupResult{}, inventoryReadResult{}, -1, -1)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "loot: ready") {
		t.Fatalf("loot panel should show loot: ready when drops available, got: %s", stripped)
	}
}

func TestLootPanelShowsLootCollected(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{},
		}},
	}
	panel := renderLootPanel(sidePanelWidth, pr, er, pickupResult{}, inventoryReadResult{}, -1, -1)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "loot: collected") {
		t.Fatalf("loot panel should show loot: collected when all drops taken, got: %s", stripped)
	}
}

func TestLootPanelShowsLootNone(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: false,
		}},
	}
	panel := renderLootPanel(sidePanelWidth, pr, er, pickupResult{}, inventoryReadResult{}, -1, -1)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "loot: none") {
		t.Fatalf("loot panel should show loot: none when no drops generated, got: %s", stripped)
	}
}

func TestLootPanelLootReadyHasDropCount(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{"item-1", "item-2", "item-3"},
		}},
	}
	panel := renderLootPanel(sidePanelWidth, pr, er, pickupResult{}, inventoryReadResult{}, -1, -1)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "loot: ready") {
		t.Fatal("should show loot: ready")
	}
	if !strings.Contains(stripped, "drops: 3") {
		t.Fatalf("should show drop count after ready line, got: %s", stripped)
	}
}

// --- Multi-Mob Spatial Disambiguation Tests (M20260404-11) ---

func TestOverlayMobsSingleMob(t *testing.T) {
	mapText := "     \n     \n     "
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	mobs := []mobPosition{{ProcessID: "orc-1", Position: mobPosVec3{X: 50, Y: 50}}}
	result := overlayMobs(mapText, mobs, bounds, 5, 3)
	if !strings.Contains(result, "m") {
		t.Fatal("single mob should show m")
	}
}

func TestOverlayMobsTwoSameCell(t *testing.T) {
	mapText := "     \n     \n     "
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	mobs := []mobPosition{
		{ProcessID: "orc-1", Position: mobPosVec3{X: 50, Y: 50}},
		{ProcessID: "orc-2", Position: mobPosVec3{X: 50, Y: 50}}, // same position
	}
	result := overlayMobs(mapText, mobs, bounds, 5, 3)
	if !strings.Contains(result, "2") {
		t.Fatalf("two mobs at same cell should show 2, got: %q", result)
	}
}

func TestOverlayMobsThreeSameCell(t *testing.T) {
	mapText := "     \n     \n     "
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	mobs := []mobPosition{
		{ProcessID: "orc-1", Position: mobPosVec3{X: 50, Y: 50}},
		{ProcessID: "orc-2", Position: mobPosVec3{X: 50, Y: 50}},
		{ProcessID: "orc-3", Position: mobPosVec3{X: 50, Y: 50}},
	}
	result := overlayMobs(mapText, mobs, bounds, 5, 3)
	if !strings.Contains(result, "3") {
		t.Fatalf("three mobs at same cell should show 3, got: %q", result)
	}
}

func TestOverlayMobsMixedCells(t *testing.T) {
	mapText := "     \n     \n     "
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	mobs := []mobPosition{
		{ProcessID: "orc-1", Position: mobPosVec3{X: 25, Y: 50}}, // one cell
		{ProcessID: "orc-2", Position: mobPosVec3{X: 75, Y: 50}}, // different cell
		{ProcessID: "orc-3", Position: mobPosVec3{X: 75, Y: 50}}, // same cell as orc-2
	}
	result := overlayMobs(mapText, mobs, bounds, 5, 3)
	if !strings.Contains(result, "m") {
		t.Fatal("single mob cell should show m")
	}
	if !strings.Contains(result, "2") {
		t.Fatal("double mob cell should show 2")
	}
}

func TestOverlayMobsTenPlusShowsPlus(t *testing.T) {
	mapText := "     \n     \n     "
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	var mobs []mobPosition
	for i := 0; i < 12; i++ {
		mobs = append(mobs, mobPosition{ProcessID: fmt.Sprintf("orc-%d", i), Position: mobPosVec3{X: 50, Y: 50}})
	}
	result := overlayMobs(mapText, mobs, bounds, 5, 3)
	if !strings.Contains(result, "+") {
		t.Fatalf("10+ mobs at same cell should show +, got: %q", result)
	}
}

func TestColorizeCountDigit(t *testing.T) {
	result := colorizeMapContent("3")
	if !strings.Contains(result, "\033[") {
		t.Fatal("count digit should be styled")
	}
	if !strings.Contains(result, "3") {
		t.Fatal("digit character should be preserved")
	}
}

func TestColorizeCountDigitSameStyleAsMob(t *testing.T) {
	mobStyled := colorizeMapContent("m")
	digitStyled := colorizeMapContent("3")
	// Extract ANSI codes — both should use mob color
	mobCodes := ansiPattern.FindAllString(mobStyled, -1)
	digitCodes := ansiPattern.FindAllString(digitStyled, -1)
	if len(mobCodes) == 0 || len(digitCodes) == 0 {
		t.Fatal("both should have ANSI codes")
	}
	if mobCodes[0] != digitCodes[0] {
		t.Fatal("count digit should use same color as regular mob")
	}
}

func TestOverlayMobsDeterministicWithClusters(t *testing.T) {
	mapText := "     \n     \n     "
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	mobs := []mobPosition{
		{ProcessID: "orc-1", Position: mobPosVec3{X: 50, Y: 50}},
		{ProcessID: "orc-2", Position: mobPosVec3{X: 50, Y: 50}},
	}
	a := overlayMobs(mapText, mobs, bounds, 5, 3)
	b := overlayMobs(mapText, mobs, bounds, 5, 3)
	if a != b {
		t.Fatal("mob overlay with clusters should be deterministic")
	}
}

// --- Combat Loot Phase Cross-Link Clarity Tests (M20260404-12) ---

func TestCombatPanelShowsPhaseLoot(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", CompletedReason: "all_mobs_dead",
			MobIDs: []string{}, DropsGenerated: true, Drops: []string{"item-1"},
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{})
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "/L") {
		t.Fatalf("combat panel should show /L suffix when drops available, got: %s", stripped)
	}
}

func TestCombatPanelCompletedNoDrop(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", CompletedReason: "all_mobs_dead",
			MobIDs: []string{}, DropsGenerated: false,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{})
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "all_mobs_dead") {
		t.Fatalf("combat panel should show completion reason, got: %s", stripped)
	}
	if strings.Contains(stripped, "/L") {
		t.Fatal("should not show /L when no drops")
	}
}

func TestCombatPanelCompletedExpired(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			DropsGenerated: true, Drops: []string{"item-1"}, LootExpired: true,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{})
	stripped := stripANSI(panel)
	if strings.Contains(stripped, "/L") {
		t.Fatal("should not show /L when loot expired")
	}
}

func TestCombatPanelNoPhaseWhenActive(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{})
	stripped := stripANSI(panel)
	if strings.Contains(stripped, "/L") {
		t.Fatal("combat panel should not show phase indicator during active combat")
	}
}

// --- Pickup Feedback Tightening Tests (M20260404-13) ---

func TestLootPanelCompactPickupSent(t *testing.T) {
	pk := pickupResult{State: pickupStateSent, ItemID: "sword-1"}
	panel := renderLootPanel(sidePanelWidth, playerReadResult{}, encounterReadResult{}, pk, inventoryReadResult{}, -1, -1)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "pk:sword-1") {
		t.Fatalf("should show pk:item-id, got: %s", stripped)
	}
}

func TestLootPanelCompactPickupFail(t *testing.T) {
	pk := pickupResult{State: pickupStateFailed, Error: "expired"}
	panel := renderLootPanel(sidePanelWidth, playerReadResult{}, encounterReadResult{}, pk, inventoryReadResult{}, -1, -1)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "pk:fail") {
		t.Fatalf("should show pk:fail, got: %s", stripped)
	}
	if !strings.Contains(stripped, "expired") {
		t.Fatal("should show failure reason")
	}
}

func TestLootPanelCompactInventoryDelta(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true, Drops: []string{},
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, Count: 5}
	pk := pickupResult{State: pickupStateSent, ItemID: "item-1"}
	panel := renderLootPanel(sidePanelWidth, pr, er, pk, inv, 3, -1) // delta = 5 - 3 = +2
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "inv:+2") {
		t.Fatalf("should show compact inv:+N, got: %s", stripped)
	}
}

func TestLootPanelCompactInventoryPending(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true, Drops: []string{},
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, Count: 3}
	pk := pickupResult{State: pickupStateSent, ItemID: "item-1"}
	panel := renderLootPanel(sidePanelWidth, pr, er, pk, inv, 3, -1) // delta = 0, still pending
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "inv:3 pending") {
		t.Fatalf("should show inv:N pending, got: %s", stripped)
	}
}

// --- Encounter Completion Summary Compactness Tests (M20260404-14) ---

func TestEncounterPanelCompletedSummary(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", CompletedReason: "all_mobs_dead",
			PlayerIDs: []string{"p1"}, MobIDs: []string{},
			PlayerCount: 1, MobCount: 2, MobsAlive: 0, MobsDead: 2,
		}},
	}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{index: -1}, "p1")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "done:all_mobs_dead") {
		t.Fatalf("completed encounter should show done:reason, got: %s", stripped)
	}
	if !strings.Contains(stripped, "1p/2m") {
		t.Fatalf("should show compact counts, got: %s", stripped)
	}
}

func TestEncounterPanelCompletedNoReason(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			PlayerCount: 1, MobCount: 1,
		}},
	}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{index: -1}, "p1")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "done:completed") {
		t.Fatalf("completed encounter without reason should show done:completed, got: %s", stripped)
	}
}

func TestEncounterPanelActiveNotCompacted(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			PlayerIDs: []string{"p1"}, MobIDs: []string{"orc-1"},
			PlayerCount: 1, MobCount: 1, MobsAlive: 1, ActionIndex: 3,
		}},
	}
	panel := renderEncounterPanel(sidePanelWidth, pr, er, rosterFocus{index: -1}, "p1")
	stripped := stripANSI(panel)
	if strings.Contains(stripped, "done:") {
		t.Fatal("active encounter should not show done: prefix")
	}
	if !strings.Contains(stripped, "Active") {
		t.Fatal("active encounter should show Active state")
	}
	if !strings.Contains(stripped, "act:3") {
		t.Fatal("active encounter should show action index")
	}
}

func TestEncounterPanelCompletedFewerLines(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	encActive := encounterSummary{
		EncounterID: "enc-1", State: "Active",
		PlayerIDs: []string{"p1"}, MobIDs: []string{"orc-1"},
		PlayerCount: 1, MobCount: 1, MobsAlive: 1, ActionIndex: 5,
	}
	encCompleted := encounterSummary{
		EncounterID: "enc-1", State: "Completed", CompletedReason: "all_mobs_dead",
		PlayerIDs: []string{"p1"}, MobIDs: []string{},
		PlayerCount: 1, MobCount: 1, MobsAlive: 0, MobsDead: 1,
	}
	activePanel := renderEncounterPanel(sidePanelWidth, pr, encounterReadResult{State: encounterReadOK, Count: 1, Encounters: []encounterSummary{encActive}}, rosterFocus{index: -1}, "p1")
	completedPanel := renderEncounterPanel(sidePanelWidth, pr, encounterReadResult{State: encounterReadOK, Count: 1, Encounters: []encounterSummary{encCompleted}}, rosterFocus{index: -1}, "p1")
	activeLines := strings.Count(stripANSI(activePanel), "\n")
	completedLines := strings.Count(stripANSI(completedPanel), "\n")
	// Completed should use same or fewer lines (no separate CompletedReason line)
	if completedLines > activeLines+1 {
		t.Fatalf("completed panel should not be significantly larger: active=%d completed=%d", activeLines, completedLines)
	}
}
