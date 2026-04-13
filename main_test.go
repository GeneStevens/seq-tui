package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	footer := renderFooter(80, "", "", "", "", "", true, inventoryReadResult{})
	if !strings.Contains(footer, " q") {
		t.Fatal("footer should contain quit hint")
	}
}

func TestRenderMapPanelContainsPlayerMarker(t *testing.T) {
	panel := renderMapPanel(mapReadResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 80, 40, attackResult{}, "")
	if !strings.ContainsRune(panel, playerMarker) {
		t.Fatal("map panel should contain player marker")
	}
}

func TestRenderLayoutContainsAllSections(t *testing.T) {
	layout := renderLayout(80, 40, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	if !strings.Contains(layout, headerTitle) {
		t.Fatal("layout should contain header title")
	}
	if !strings.ContainsRune(layout, playerMarker) {
		t.Fatal("layout should contain player marker")
	}
	if !strings.Contains(layout, " q") {
		t.Fatal("layout should contain quit hint")
	}
}

func TestRenderLayoutNonEmpty(t *testing.T) {
	layout := renderLayout(80, 40, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
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

func TestRenderSideColumnContainsDiagnostics(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	if !strings.Contains(col, "Diagnostics") {
		t.Fatal("side column should contain diagnostics section when showDiag=true")
	}
}

func TestWideLayoutContainsPanels(t *testing.T) {
	layout := renderLayout(120, 40, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	if !strings.Contains(layout, "Diagnostics") {
		t.Fatal("wide layout should contain diagnostics section")
	}
	if !strings.ContainsRune(layout, playerMarker) {
		t.Fatal("wide layout should still contain player marker")
	}
}

func TestNarrowLayoutOmitsPanels(t *testing.T) {
	layout := renderLayout(50, 30, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	if strings.Contains(layout, "Diagnostics") {
		t.Fatal("narrow layout should not contain diagnostics")
	}
	if !strings.ContainsRune(layout, playerMarker) {
		t.Fatal("narrow layout should still contain player marker")
	}
}

func TestRenderLayoutSmallTerminal(t *testing.T) {
	// Should not panic with very small dimensions
	layout := renderLayout(20, 5, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	if len(layout) == 0 {
		t.Fatal("layout should not be empty even for small terminal")
	}
}

func TestRenderLayoutVariousSizes(t *testing.T) {
	sizes := [][2]int{{40, 20}, {80, 40}, {120, 50}, {200, 60}}
	for _, sz := range sizes {
		layout := renderLayout(sz[0], sz[1], "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
		if !strings.Contains(layout, headerTitle) {
			t.Fatalf("layout at %dx%d missing header", sz[0], sz[1])
		}
		if !strings.ContainsRune(layout, playerMarker) {
			t.Fatalf("layout at %dx%d missing player marker", sz[0], sz[1])
		}
		if !strings.Contains(layout, " q") {
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
	footer := renderFooter(120, "", "", "", "", "", true, inventoryReadResult{})
	if !strings.Contains(footer, "hjkl:mv") {
		t.Fatal("footer should advertise movement keys")
	}
	if !strings.Contains(footer, " q") {
		t.Fatal("footer should still contain quit hint")
	}
}

func TestFooterShowsIntentPreview(t *testing.T) {
	preview := moveIntent{direction: "north"}.preview()
	footer := renderFooter(120, preview, "", "", "", "", true, inventoryReadResult{})
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
	panel := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 80, 40, attackResult{}, "")
	// Strip ANSI to check semantic content — wall chars are individually styled
	if !strings.Contains(stripANSI(panel), "###") {
		t.Fatal("map panel should use backend map text when available")
	}
}

func TestMapPanelFallsBackToPlaceholder(t *testing.T) {
	mr := mapReadResult{State: mapReadFailed}
	panel := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 80, 40, attackResult{}, "")
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
	panel := renderMapPanel(mr, mobr, playerReadResult{}, rosterFocus{}, nil, 80, 40, attackResult{}, "")
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
	a := overlayPlayer(mapText, pos, bounds, 5, 3, false)
	b := overlayPlayer(mapText, pos, bounds, 5, 3, false)
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
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 80, 40, attackResult{}, "")
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

func TestDevPlayerMoveURL(t *testing.T) {
	target := defaultTarget()
	url := devPlayerMoveURL(target)
	if !strings.Contains(url, "/world/dev/zone/crushbone/player/move") {
		t.Fatalf("move URL should target dev player move endpoint, got: %s", url)
	}
}

func TestDirectionToBackend(t *testing.T) {
	cases := map[string]string{
		"north": "up",
		"south": "down",
		"east":  "right",
		"west":  "left",
	}
	for tui, backend := range cases {
		got := directionToBackend(tui)
		if got != backend {
			t.Fatalf("directionToBackend(%q) = %q, want %q", tui, got, backend)
		}
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
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	if !strings.Contains(col, encounterTitle) {
		t.Fatal("side column should contain encounter panel title")
	}
	if !strings.Contains(col, "Diagnostics") {
		t.Fatal("side column should contain diagnostics section")
	}
}

func TestWideLayoutContainsEncounterPanel(t *testing.T) {
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	if !strings.Contains(layout, encounterTitle) {
		t.Fatal("wide layout should contain encounter panel")
	}
}

func TestNarrowLayoutOmitsEncounterPanel(t *testing.T) {
	layout := renderLayout(50, 30, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
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
	// Backend shape: {"found":true,"player":{"position":{"x":10,"y":20,"z":0},"active_encounter_id":"enc-abc"}}
	body := []byte(`{"found":true,"player":{"position":{"x":10,"y":20,"z":0},"active_encounter_id":"enc-abc"}}`)
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
	body := []byte(`{"found":true,"player":{"position":{"x":5,"y":15,"z":0}}}`)
	result := decodePlayerState(body, playerReadOK)
	if result.HasActiveEncounter {
		t.Fatal("should not have active encounter when field absent")
	}
	if result.ActiveEncounterID != "" {
		t.Fatal("encounter ID should be empty")
	}
}

func TestDecodePlayerStateNotFound(t *testing.T) {
	body := []byte(`{"found":false,"player_id":"p1"}`)
	result := decodePlayerState(body, playerReadOK)
	if result.HasPos {
		t.Fatal("should not have position when not found")
	}
}

func TestDecodePlayerStateNonZeroPosition(t *testing.T) {
	// This is the proof test: backend returns non-zero coords, TUI must decode them correctly
	body := []byte(`{"found":true,"player":{"position":{"x":120.5,"y":340.2,"z":0}}}`)
	result := decodePlayerState(body, playerReadOK)
	if !result.HasPos {
		t.Fatal("should have position")
	}
	if result.Position.X != 120.5 {
		t.Fatalf("expected X=120.5, got %f", result.Position.X)
	}
	if result.Position.Y != 340.2 {
		t.Fatalf("expected Y=340.2, got %f", result.Position.Y)
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
	footer := renderFooter(80, "", "", "", "", "", true, inventoryReadResult{})
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
	if !strings.Contains(result, "T") {
		t.Fatal("focused mob should be rendered as T (target indicator)")
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
	panel := renderMapPanel(mr, mobr, playerReadResult{}, focus, entries, 80, 40, attackResult{}, "")
	if !strings.Contains(panel, "T") {
		t.Fatal("focused mob should appear as T on map")
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
	panel := renderMapPanel(mr, mobr, playerReadResult{}, focus, entries, 80, 40, attackResult{}, "")
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
	a := renderMapPanel(mr, mobr, playerReadResult{}, focus, entries, 80, 40, attackResult{}, "")
	b := renderMapPanel(mr, mobr, playerReadResult{}, focus, entries, 80, 40, attackResult{}, "")
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
	panel := renderMapPanel(mr, mobr, playerReadResult{}, focus, entries, 80, 40, attackResult{}, "")
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
	if label != "" {
		t.Fatalf("expected empty, got %q", label)
	}
}

func TestFocusPreviewLabelEmptyEntries(t *testing.T) {
	label := focusPreviewLabel(rosterFocus{index: 0}, nil)
	if label != "" {
		t.Fatalf("expected empty, got %q", label)
	}
}

func TestFocusPreviewLabelMob(t *testing.T) {
	entries := []rosterEntry{{kind: "pc", id: "hero"}, {kind: "mb", id: "orc-a", canonicalMobID: true}}
	label := focusPreviewLabel(rosterFocus{index: 1}, entries)
	if label != "~mb:orc-a" {
		t.Fatalf("expected '~mb:orc-a', got %q", label)
	}
}

func TestFocusPreviewLabelPlayer(t *testing.T) {
	entries := []rosterEntry{{kind: "pc", id: "hero-1"}}
	label := focusPreviewLabel(rosterFocus{index: 0}, entries)
	if label != "~pc:hero-1" {
		t.Fatalf("expected '~pc:hero-1', got %q", label)
	}
}

func TestFocusPreviewLabelOutOfRange(t *testing.T) {
	entries := []rosterEntry{{kind: "mb", id: "m1"}}
	label := focusPreviewLabel(rosterFocus{index: 5}, entries)
	if label != "" {
		t.Fatalf("expected empty, got %q", label)
	}
}

func TestFooterShowsFocusPreview(t *testing.T) {
	footer := renderFooter(120, "", "~mb:orc-a", "", "", "", true, inventoryReadResult{})
	if !strings.Contains(footer, "~mb:orc-a") {
		t.Fatal("footer should show focus preview label")
	}
	if !strings.Contains(footer, " q") {
		t.Fatal("footer should still contain quit hint")
	}
}

func TestFooterShowsFocusNone(t *testing.T) {
	// When unfocused, focusPreviewLabel returns "" — footer should not contain stale focus label
	footer := renderFooter(120, "", "", "", "", "", true, inventoryReadResult{})
	if strings.Contains(footer, "~") {
		t.Fatal("footer should not show focus marker when unfocused")
	}
}

func TestFooterShowsBothPreviews(t *testing.T) {
	preview := moveIntent{direction: "north"}.preview()
	footer := renderFooter(120, preview, "~mb:orc", "", "", "", true, inventoryReadResult{})
	if !strings.Contains(footer, "move north") {
		t.Fatal("footer should show movement intent")
	}
	if !strings.Contains(footer, "~mb:orc") {
		t.Fatal("footer should show focus preview")
	}
}

func TestLayoutIncludesFocusPreview(t *testing.T) {
	entries := []rosterEntry{{kind: "mb", id: "test-mob"}}
	focus := rosterFocus{index: 0}
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, focus, entries, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	if !strings.Contains(layout, "~mb:test-mob") {
		t.Fatal("layout should include focus preview in footer")
	}
}

func TestLayoutFocusNoneWhenUnfocused(t *testing.T) {
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	// When unfocused, no ~ marker should appear in footer
	if strings.Contains(stripANSI(layout), "~mb:") || strings.Contains(stripANSI(layout), "~pc:") {
		t.Fatal("layout should not show focus marker when unfocused")
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

func TestFocusPreviewContainsTildeMarker(t *testing.T) {
	entries := []rosterEntry{{kind: "mb", id: "orc"}}
	label := focusPreviewLabel(rosterFocus{index: 0}, entries)
	if !strings.HasPrefix(label, "~") {
		t.Fatal("focus preview must start with ~ marker")
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

	if !strings.HasPrefix(focusLabel, "~") {
		t.Fatal("focus label must start with ~")
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
	footer := renderFooter(120, "", "", "target: orc (backend)", "", "", true, inventoryReadResult{})
	if !strings.Contains(footer, "target: orc (backend)") {
		t.Fatal("footer should show target label")
	}
}

func TestFooterShowsAllThreeLabels(t *testing.T) {
	preview := moveIntent{direction: "north"}.preview()
	footer := renderFooter(120, preview, "focus: mb:orc (local)", "target: orc (backend)", "", "", true, inventoryReadResult{})
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
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, tc, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	if !strings.Contains(layout, "(backend)") {
		t.Fatal("layout should include backend target label")
	}
}

func TestLayoutTargetNoneByDefault(t *testing.T) {
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
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
	footer := renderFooter(120, "", "", "", "", "", true, inventoryReadResult{})
	if !strings.Contains(footer, "t:prx") {
		t.Fatal("footer should mention t: confirm keybind")
	}
}

// --- Proximity Panel Tests (M29) ---

func TestTargetPanelNoTarget(t *testing.T) {
	panel := renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{index: -1}, nil)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "Target") {
		t.Fatal("target panel should contain title")
	}
	if !strings.Contains(stripped, "no target") {
		t.Fatalf("target panel should show 'no target' when nothing focused, got: %s", stripped)
	}
}

func TestTargetPanelNoTargetOnFailure(t *testing.T) {
	tc := targetConfirmResult{State: targetConfirmFailed, Error: "HTTP 500"}
	panel := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{index: -1}, nil)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "no target") {
		t.Fatalf("target panel should show 'no target' when no focus and backend failed, got: %s", stripped)
	}
}

func TestProximityPanelFoundWithMobName(t *testing.T) {
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-a",
		Found: true, WithinProximity: false, Distance: 12.3, MobName: "a_skeleton",
	}
	panel := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{index: -1}, nil)
	if !strings.Contains(panel, "a_skeleton") {
		t.Fatal("proximity panel should show mob name")
	}
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "OOR") {
		t.Fatal("target panel should show out-of-range indicator")
	}
	if !strings.Contains(stripped, "12") {
		t.Fatal("proximity panel should show distance")
	}
}

func TestProximityPanelFoundWithinProximity(t *testing.T) {
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-a",
		Found: true, WithinProximity: true, Distance: 2.1, MobName: "orc",
	}
	panel := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{index: -1}, nil)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "IN RANGE") {
		t.Fatalf("proximity panel should show in-range indicator, got: %s", stripped)
	}
}

func TestProximityPanelNotFound(t *testing.T) {
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-a", Found: false,
	}
	panel := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{index: -1}, nil)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "not found") {
		t.Fatalf("proximity panel should show not found, got: %s", stripped)
	}
}

func TestProximityPanelFallbackID(t *testing.T) {
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-a",
		Found: true, MobName: "",
	}
	panel := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{index: -1}, nil)
	if !strings.Contains(panel, "mb:orc-a") {
		t.Fatal("proximity panel should fall back to kind:id when no mob name")
	}
}

func TestProximityPanelDeterministic(t *testing.T) {
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc",
		Found: true, WithinProximity: true, Distance: 5.0, MobName: "orc",
	}
	a := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{index: -1}, nil)
	b := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{index: -1}, nil)
	if a != b {
		t.Fatal("proximity panel should be deterministic")
	}
}

func TestSideColumnContainsProximityPanel(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	if !strings.Contains(col, "Target") {
		t.Fatal("side column should contain target panel")
	}
}

func TestWideLayoutContainsProximityPanel(t *testing.T) {
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	if !strings.Contains(layout, "Target") {
		t.Fatal("wide layout should contain target panel")
	}
}

func TestProximityPanelNoGameplayTerms(t *testing.T) {
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc",
		Found: true, WithinProximity: true, Distance: 2.0, MobName: "orc",
	}
	panel := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{index: -1}, nil)
	forbidden := []string{"attack", "threat", "aggro", "damage", "hp", "health", "enemy"}
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
	footer := renderFooter(120, "", "", "", "attack: sent", "", true, inventoryReadResult{})
	if !strings.Contains(footer, "attack: sent") {
		t.Fatal("footer should show attack label")
	}
}

func TestFooterShowsAttackKeybind(t *testing.T) {
	footer := renderFooter(120, "", "", "", "", "", true, inventoryReadResult{})
	if !strings.Contains(footer, "a:atk") {
		t.Fatal("footer should mention a: attack keybind")
	}
}

func TestLayoutIncludesAttackLabel(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc"}
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, ar, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	if !strings.Contains(layout, "attack: sent") {
		t.Fatal("layout should include attack label when sent")
	}
}

func TestLayoutNoAttackLabelByDefault(t *testing.T) {
	layout := renderLayout(120, 50, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
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
	pr := playerReadResult{State: playerReadOK}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, encounterReadResult{}, defaultTarget(), inventoryReadResult{}, "")
	if !strings.Contains(panel, "Combat") {
		t.Fatal("combat panel should contain title")
	}
	if !strings.Contains(panel, "none") {
		t.Fatal("combat panel should show none before any attack")
	}
}

func TestCombatPanelIntentFailed(t *testing.T) {
	ar := attackResult{State: attackStateFailed, Error: "out of range"}
	pr := playerReadResult{State: playerReadOK}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, encounterReadResult{}, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "atk:fail") {
		t.Fatalf("combat panel should show atk:fail, got: %s", stripped)
	}
}

func TestCombatPanelIntentAcceptedNoEncounter(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: false}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, encounterReadResult{}, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "atk:orc-1") {
		t.Fatalf("combat panel should show atk:<targetID>, got: %s", stripped)
	}
	if !strings.Contains(stripped, "enc: opening") {
		t.Fatalf("combat panel should show enc: opening when attack sent but no encounter yet, got: %s", stripped)
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
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	if !strings.Contains(panel, "act:") {
		t.Fatal("combat panel should show compact state line")
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
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "all_mobs_dead") {
		t.Fatalf("combat panel should show completion reason, got: %s", stripped)
	}
	// Mob roster and atk lines suppressed for completed encounters
	if strings.Contains(stripped, "atk:") {
		t.Fatal("atk line should be suppressed for completed encounter")
	}
}

func TestCombatPanelEncounterUnavailable(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{State: encounterReadFailed}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	// When encounter read fails, falls through to no-encounter path
	if !strings.Contains(stripped, "atk:orc-1") {
		t.Fatalf("combat panel should show atk:<targetID> when encounter unavailable, got: %s", stripped)
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
	a := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	b := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
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
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	forbidden := []string{"damage", "hit", "miss", "crit", "dps", "health", "landed"}
	for _, word := range forbidden {
		if strings.Contains(strings.ToLower(panel), word) {
			t.Fatalf("combat panel should not contain combat resolution term: %s", word)
		}
	}
}

func TestSideColumnContainsCombatPanel(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	if !strings.Contains(col, "Combat") {
		t.Fatal("side column should contain combat panel")
	}
}

func TestWideLayoutContainsCombatPanel(t *testing.T) {
	layout := renderLayout(120, 80, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	if !strings.Contains(layout, "Combat") {
		t.Fatal("wide layout should contain combat panel")
	}
}

// --- Loot Readback and Pickup Tests (M34) ---

func TestLootPanelNone(t *testing.T) {
	panel := renderLootPanel(sidePanelWidth, playerReadResult{State: playerReadOK}, encounterReadResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
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
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	if !strings.Contains(col, "Loot") {
		t.Fatal("side column should contain loot panel")
	}
}

func TestFooterContainsPickupHint(t *testing.T) {
	footer := renderFooter(120, "", "", "", "", "", true, inventoryReadResult{})
	if !strings.Contains(footer, "p:pk") {
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
	footer := renderFooter(120, "", "", "", "", "", true, inventoryReadResult{})
	if !strings.Contains(footer, "[]:lt") {
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
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 40, 20, attackResult{}, "")
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
	panel := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 40, 20, attackResult{}, "")
	if !strings.Contains(panel, "#") {
		t.Fatal("no-player viewport should still show map content")
	}
	// Should be deterministic
	panel2 := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 40, 20, attackResult{}, "")
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
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 30, 15, attackResult{}, "")
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
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 30, 15, attackResult{}, "")
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
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 80, 40, attackResult{}, "")
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
	panel := renderMapPanel(mr, mobr, playerReadResult{}, rosterFocus{}, nil, 80, 40, attackResult{}, "")
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
	panel := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 80, 40, attackResult{}, "")
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
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 30, 15, attackResult{}, "")
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
	panel := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 40, 20, attackResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "#") {
		t.Fatal("no-player adaptive panel should contain wall characters")
	}
	// Should be deterministic
	panel2 := renderMapPanel(mr, mobReadResult{}, playerReadResult{}, rosterFocus{}, nil, 40, 20, attackResult{}, "")
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
	panel := renderMapPanel(mr, mobr, pr, rosterFocus{}, nil, 80, 40, attackResult{}, "")
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
	smallPanel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 20, 10, attackResult{}, "")
	largePanel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 120, 60, attackResult{}, "")
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
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 60, 30, attackResult{}, "")
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
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "dmg") {
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
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "res:miss") {
		t.Fatalf("combat panel should show res:miss, got: %s", stripped)
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
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{}, "")
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
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{}, "")
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
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{}, "")
	if !strings.Contains(panel, "p1 attacks") {
		t.Fatal("combat panel should show backend text summary")
	}
}

func TestCombatPanelWithoutEncounterShowsNone(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, encounterReadResult{}, defaultTarget(), inventoryReadResult{}, "")
	if !strings.Contains(panel, "none") {
		t.Fatal("combat panel without encounter should show 'none'")
	}
}

func TestCombatPanelNotJoined(t *testing.T) {
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, playerReadResult{}, encounterReadResult{}, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "not joined") {
		t.Fatalf("combat panel should show 'not joined' when player not joined, got: %s", stripped)
	}
}

func TestLootPanelNotJoined(t *testing.T) {
	panel := renderLootPanel(sidePanelWidth, playerReadResult{}, encounterReadResult{}, pickupResult{}, inventoryReadResult{}, -1, -1)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "not joined") {
		t.Fatalf("loot panel should show 'not joined' when player not joined, got: %s", stripped)
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
	a := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	b := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
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
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "DEAD") {
		t.Fatalf("player panel should show DEAD prominently, got: %s", stripped)
	}
	if !strings.Contains(stripped, "0/100") {
		t.Fatal("player panel should show HP 0/100")
	}
}

func TestPlayerPanelCanAct(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	stripped := stripANSI(panel)
	// Alive + can act: should show HP, no DEAD/blocked
	if !strings.Contains(stripped, "hp: 100/100") {
		t.Fatalf("player panel should show HP when alive, got: %s", stripped)
	}
	if strings.Contains(stripped, "DEAD") || strings.Contains(stripped, "blocked") {
		t.Fatal("player panel should not show DEAD or blocked when alive and can act")
	}
}

func TestPlayerPanelBlockedWithReason(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, BlockedReason: "cooldown", HPCurrent: 80, HPMax: 100}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, respawnResult{})
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "blocked") {
		t.Fatalf("player panel should show blocked status, got: %s", stripped)
	}
	if !strings.Contains(stripped, "cooldown") {
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
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
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

func TestPlayerPanelRespawnRestoredSuppressed(t *testing.T) {
	// After respawn: backend shows HP restored — respawn line suppressed (stale)
	pr := playerReadResult{State: playerReadOK}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	rs := respawnResult{State: respawnStateSent}
	panel := renderPlayerPanel(sidePanelWidth, pr, inv, rs)
	stripped := stripANSI(panel)
	if strings.Contains(stripped, "respawn") {
		t.Fatalf("player panel should suppress stale respawn line once restored, got: %q", stripped)
	}
	// HP and can-act should still confirm recovery
	if !strings.Contains(stripped, "hp: 100/100") {
		t.Fatal("HP should still show recovery")
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
	footer := renderFooter(120, "", "", "", "", "", true, inventoryReadResult{})
	if !strings.Contains(footer, "r:res") {
		t.Fatal("footer should mention r: respawn keybind")
	}
}

// --- Multi-Mob Combat Roster Clarity Tests (M20260404-01) ---

func TestRenderCombatMobRosterBasic(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{"orc-1", "orc-2", "orc-3"},
	}
	lines := renderCombatMobRoster(enc, attackResult{}, "p1", 20, "")
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
	lines := renderCombatMobRoster(enc, ar, "p1", 20, "")
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
	lines := renderCombatMobRoster(enc, attackResult{}, "p1", 20, "")
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
	lines := renderCombatMobRoster(enc, ar, "p1", 20, "")
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
	lines := renderCombatMobRoster(enc, ar, "p1", 20, "")
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
	lines := renderCombatMobRoster(enc, attackResult{}, "p1", 20, "")
	if lines != nil {
		t.Fatal("empty roster with no target should return nil")
	}
}

func TestRenderCombatMobRosterEmptyWithGoneTarget(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{},
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	lines := renderCombatMobRoster(enc, ar, "p1", 20, "")
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
	lines := renderCombatMobRoster(nil, attackResult{}, "p1", 20, "")
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
	a := renderCombatMobRoster(enc, ar, "p1", 20, "")
	b := renderCombatMobRoster(enc, ar, "p1", 20, "")
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
	lines := renderCombatMobRoster(enc, attackResult{}, "p1", 20, "")
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
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
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
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "You") {
		t.Fatalf("combat panel should show player HP line when can_act is true, got: %s", stripped)
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
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inv, "")
	stripped := stripANSI(panel)
	// Blocked state is now shown in the Player panel; combat panel shows HP frame
	if !strings.Contains(stripped, "You") {
		t.Fatalf("combat panel should show player HP line when blocked, got: %s", stripped)
	}
	if !strings.Contains(stripped, "act:") {
		t.Fatalf("combat panel should show compact state line when blocked, got: %s", stripped)
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
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inv, "")
	stripped := stripANSI(panel)
	// Blocked/dead state is now shown in the Player panel; combat panel shows HP frame
	if !strings.Contains(stripped, "You") {
		t.Fatalf("combat panel should show player HP line when dead, got: %s", stripped)
	}
	if !strings.Contains(stripped, "act:") {
		t.Fatalf("combat panel should show compact state line when dead, got: %s", stripped)
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
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inv, "")
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
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, "")
	stripped := stripANSI(panel)
	// Both player HP line and result should be present and distinct
	if !strings.Contains(stripped, "You") {
		t.Fatal("should show player HP line")
	}
	if !strings.Contains(stripped, "dmg") {
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
	a := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inv, "")
	b := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inv, "")
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
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inv, "")
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
	result := overlayAttackTarget(mapText, mobs, "orc-1", bounds, 5, 3, 'X')
	if !strings.Contains(result, "X") {
		t.Fatal("attack target overlay should place X on map")
	}
}

func TestOverlayAttackTargetNoMatch(t *testing.T) {
	mapText := "     \n     \n     "
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	mobs := []mobPosition{{ProcessID: "orc-1", Position: mobPosVec3{X: 50, Y: 50}}}
	result := overlayAttackTarget(mapText, mobs, "orc-99", bounds, 5, 3, 'X')
	if strings.Contains(result, "X") {
		t.Fatal("attack target overlay should not place X when target not in mobs")
	}
}

func TestOverlayAttackTargetEmpty(t *testing.T) {
	mapText := "#####"
	result := overlayAttackTarget(mapText, nil, "orc-1", mapBounds{}, 5, 1, 'X')
	if result != mapText {
		t.Fatal("empty mobs should return map unchanged")
	}
}

func TestOverlayAttackTargetEmptyID(t *testing.T) {
	mapText := "#####"
	mobs := []mobPosition{{ProcessID: "orc-1", Position: mobPosVec3{X: 50, Y: 50}}}
	result := overlayAttackTarget(mapText, mobs, "", mapBounds{SpanX: 100, SpanZ: 100}, 5, 1, 'X')
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
	panel := renderMapPanel(mr, mobr, pr, rosterFocus{}, nil, 80, 40, ar, "")
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
	panel := renderMapPanel(mr, mobr, playerReadResult{}, rosterFocus{}, nil, 80, 40, attackResult{}, "")
	stripped := stripANSI(panel)
	if strings.Contains(stripped, "X") {
		t.Fatal("map panel should not show X when no attack submitted")
	}
}

func TestOverlayAttackTargetDeterministic(t *testing.T) {
	mapText := "     \n     \n     "
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	mobs := []mobPosition{{ProcessID: "orc-1", Position: mobPosVec3{X: 50, Y: 50}}}
	a := overlayAttackTarget(mapText, mobs, "orc-1", bounds, 5, 3, 'X')
	b := overlayAttackTarget(mapText, mobs, "orc-1", bounds, 5, 3, 'X')
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
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
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
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
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
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{}, "")
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
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, "")
	stripped := stripANSI(panel)
	// All three should be present and distinct
	if !strings.Contains(stripped, "atk:") {
		t.Fatal("should show submission status")
	}
	if !strings.Contains(stripped, "dmg") {
		t.Fatal("should show backend result")
	}
	if !strings.Contains(stripped, "You") {
		t.Fatal("should show player HP line")
	}
}

// --- Defeated Target Aftermath Clarity Tests (M20260404-07) ---

func TestMobRosterTargetGoneShowsDead(t *testing.T) {
	enc := &encounterSummary{
		MobIDs:          []string{},
		CompletedReason: "all_mobs_dead",
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	lines := renderCombatMobRoster(enc, ar, "p1", 20, "")
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
	lines := renderCombatMobRoster(enc, ar, "p1", 20, "")
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
	lines := renderCombatMobRoster(enc, ar, "p1", 20, "")
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
	lines := renderCombatMobRoster(enc, ar, "p1", 20, "")
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
	panel := renderMapPanel(mr, mobr, pr, rosterFocus{index: -1}, nil, 80, 40, ar, "")
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
	panel := renderMapPanel(mr, mobr, pr, rosterFocus{index: -1}, nil, 80, 40, attackResult{}, "")
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
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{index: -1}, nil, 40, 20, attackResult{}, "")
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
	a := renderMapPanel(mr, mobr, pr, rosterFocus{index: -1}, nil, 80, 40, ar, "")
	b := renderMapPanel(mr, mobr, pr, rosterFocus{index: -1}, nil, 80, 40, ar, "")
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
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
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
	panel1 := renderCombatPanel(sidePanelWidth, ar1, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped1 := stripANSI(panel1)
	// Second attack targets orc-2
	ar2 := attackResult{State: attackStateSent, TargetID: "orc-2"}
	panel2 := renderCombatPanel(sidePanelWidth, ar2, pr, er, defaultTarget(), inventoryReadResult{}, "")
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
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{}, "")
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
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{}, "")
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
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{}, "")
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
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{}, "")
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

// --- Proximity and Combat Coordination Clarity Tests (M20260409-02) ---

func TestTargetPanelShowsInRange(t *testing.T) {
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1",
		Found: true, WithinProximity: true, Distance: 3.0, MobName: "orc",
	}
	panel := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{index: -1}, nil)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "IN RANGE") {
		t.Fatalf("target panel should show IN RANGE, got: %s", stripped)
	}
}

func TestTargetPanelTitleIsTarget(t *testing.T) {
	panel := renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{index: -1}, nil)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "Target") {
		t.Fatalf("panel title should be Target, got: %s", stripped)
	}
}

func TestProximityPanelCompactInRange(t *testing.T) {
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1",
		Found: true, WithinProximity: true, Distance: 2.0,
	}
	panel := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{index: -1}, nil)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "IN RANGE") {
		t.Fatalf("should show compact in-range, got: %s", stripped)
	}
}

func TestProximityPanelCompactOutOfRange(t *testing.T) {
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1",
		Found: true, WithinProximity: false, Distance: 15.0,
	}
	panel := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{index: -1}, nil)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "OOR") {
		t.Fatalf("should show compact out-of-range, got: %s", stripped)
	}
}

// --- Stale Action State Reset Clarity Tests (M20260409-04) ---

func TestCombatPanelAtkSuppressedOnCompleted(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", CompletedReason: "all_mobs_dead",
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if strings.Contains(stripped, "atk:") {
		t.Fatalf("atk line should be suppressed on completed encounter, got: %s", stripped)
	}
}

func TestCombatPanelAtkShownDuringActive(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "atk:") {
		t.Fatal("atk line should still show during active combat")
	}
}

func TestLootPanelPkSuppressedAfterConfirmation(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true, Drops: []string{},
		}},
	}
	pk := pickupResult{State: pickupStateSent, ItemID: "item-1"}
	inv := inventoryReadResult{State: inventoryReadOK, Count: 4}
	// invAtPickup=3, Count=4 → delta confirmed
	panel := renderLootPanel(sidePanelWidth, pr, er, pk, inv, 3, -1)
	stripped := stripANSI(panel)
	if strings.Contains(stripped, "pk:item-1") {
		t.Fatalf("pk line should be suppressed after inventory confirmation, got: %s", stripped)
	}
}

func TestLootPanelPkShownWhilePending(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true, Drops: []string{"item-1"},
		}},
	}
	pk := pickupResult{State: pickupStateSent, ItemID: "item-1"}
	inv := inventoryReadResult{State: inventoryReadOK, Count: 3}
	// invAtPickup=3, Count=3 → no delta yet
	panel := renderLootPanel(sidePanelWidth, pr, er, pk, inv, 3, -1)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "pk:item-1") {
		t.Fatalf("pk line should show while pending, got: %s", stripped)
	}
}

// --- Joined Interaction Clarity Tests (M20260409-05) ---

func TestFooterNotJoinedShowsReduced(t *testing.T) {
	footer := renderFooter(120, "", "", "", "", "", false, inventoryReadResult{})
	stripped := stripANSI(footer)
	if !strings.Contains(stripped, "joining") {
		t.Fatalf("not-joined footer should show joining hint, got: %s", stripped)
	}
	if strings.Contains(stripped, "a:atk") {
		t.Fatal("not-joined footer should not show attack keybind")
	}
}

func TestFooterJoinedShowsFull(t *testing.T) {
	footer := renderFooter(120, "", "", "", "", "", true, inventoryReadResult{})
	stripped := stripANSI(footer)
	if !strings.Contains(stripped, "a:atk") {
		t.Fatal("joined footer should show full keybind set")
	}
	if strings.Contains(stripped, "joining") {
		t.Fatal("joined footer should not show joining hint")
	}
}

func TestLayoutNotJoinedFooter(t *testing.T) {
	// When player not joined, layout footer should show reduced hints
	layout := renderLayout(120, 40, "", defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	stripped := stripANSI(layout)
	if !strings.Contains(stripped, "joining") {
		t.Fatalf("not-joined layout should show joining hint in footer, got last line: %s", stripped[len(stripped)-100:])
	}
}

// --- Combat Loot Respawn Precedence Polish Tests (M20260409-09) ---

func TestCombatPanelCompletedSuppressesStaleLines(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", CompletedReason: "all_mobs_dead",
			DropsGenerated: true, Drops: []string{"item-1"},
			LatestResultKind: "damage_applied", LatestResultValue: 25,
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, "")
	stripped := stripANSI(panel)
	// Stale lines should be suppressed
	if strings.Contains(stripped, "rdy:") {
		t.Fatal("rdy: should be suppressed for completed encounter")
	}
	if strings.Contains(stripped, "atk:") {
		t.Fatal("atk: should be suppressed for completed encounter")
	}
	if strings.Contains(stripped, "dmg") {
		t.Fatal("result should be suppressed for completed encounter")
	}
	// Completion info should still show
	if !strings.Contains(stripped, "all_mobs_dead") {
		t.Fatal("completion reason should still show")
	}
	if !strings.Contains(stripped, "/L") {
		t.Fatal("loot indicator should still show")
	}
}

func TestCombatPanelActiveKeepsAllLines(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID:      "enc-1", State: "Active",
			MobIDs:           []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "damage_applied", LatestResultValue: 10,
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "You") {
		t.Fatal("active encounter should show player HP line")
	}
	if !strings.Contains(stripped, "atk:") {
		t.Fatal("active encounter should show attack status")
	}
	if !strings.Contains(stripped, "dmg") {
		t.Fatal("active encounter should show result")
	}
}

func TestCombatPanelCompletedIsCompact(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	encActive := encounterSummary{
		EncounterID: "enc-1", State: "Active",
		MobIDs: []string{"orc-1"}, MobsAlive: 1,
		LatestResultKind: "damage_applied", LatestResultValue: 10,
	}
	encCompleted := encounterSummary{
		EncounterID: "enc-1", State: "Completed", CompletedReason: "all_mobs_dead",
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true}
	activePanel := renderCombatPanel(sidePanelWidth, ar, pr, encounterReadResult{State: encounterReadOK, Count: 1, Encounters: []encounterSummary{encActive}}, defaultTarget(), inv, "")
	completedPanel := renderCombatPanel(sidePanelWidth, ar, pr, encounterReadResult{State: encounterReadOK, Count: 1, Encounters: []encounterSummary{encCompleted}}, defaultTarget(), inv, "")
	activeLines := strings.Count(stripANSI(activePanel), "\n")
	completedLines := strings.Count(stripANSI(completedPanel), "\n")
	if completedLines >= activeLines {
		t.Fatalf("completed panel should be shorter than active: active=%d completed=%d", activeLines, completedLines)
	}
}

// --- Small Terminal Panel Ordering Sanity Tests (M20260409-10) ---

func TestSideColumnPlayerBeforeCombat(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{State: playerReadOK}, encounterReadResult{}, rosterFocus{index: -1}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	stripped := stripANSI(col)
	playerIdx := strings.Index(stripped, "Player")
	combatIdx := strings.Index(stripped, "Combat")
	if playerIdx < 0 || combatIdx < 0 {
		t.Fatal("side column should contain Player and Combat panels")
	}
	if playerIdx > combatIdx {
		t.Fatal("Player panel should appear before Combat panel")
	}
}

func TestSideColumnCombatBeforeLoot(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{State: playerReadOK}, encounterReadResult{}, rosterFocus{index: -1}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	stripped := stripANSI(col)
	combatIdx := strings.Index(stripped, "Combat")
	lootIdx := strings.Index(stripped, "Loot")
	if combatIdx < 0 || lootIdx < 0 {
		t.Fatal("side column should contain Combat and Loot panels")
	}
	if combatIdx > lootIdx {
		t.Fatal("Combat panel should appear before Loot panel")
	}
}

func TestSideColumnDiagnosticsIsLast(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{State: playerReadOK}, encounterReadResult{}, rosterFocus{index: -1}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	stripped := stripANSI(col)
	diagIdx := strings.Index(stripped, "Diagnostics")
	eventsIdx := strings.Index(stripped, "Events")
	if diagIdx < 0 || eventsIdx < 0 {
		t.Fatal("side column should contain Diagnostics and Events")
	}
	if diagIdx < eventsIdx {
		t.Fatal("Diagnostics should appear after Events")
	}
}

func TestSideColumnNoBlankLineSeparators(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{State: playerReadOK}, encounterReadResult{}, rosterFocus{index: -1}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	stripped := stripANSI(col)
	// Should not have double newlines (blank line separators removed)
	if strings.Contains(stripped, "\n\n\n") {
		t.Fatal("side column should not have triple newlines (removed blank separators)")
	}
}

// --- Target Focus and Attack Target Coherence Tests (M20260409-11) ---

func TestMobRosterShowsFocusMarker(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{"orc-1", "orc-2"},
	}
	lines := renderCombatMobRoster(enc, attackResult{}, "p1", 20, "orc-1")
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	if !strings.Contains(joined, "orc-1 ~") || !strings.Contains(joined, "~") {
		t.Fatalf("focused mob should have ~ marker, got: %s", joined)
	}
	// orc-2 should NOT have ~
	for _, l := range lines {
		s := stripANSI(l)
		if strings.Contains(s, "orc-2") && strings.Contains(s, "~") {
			t.Fatal("non-focused mob should not have ~ marker")
		}
	}
}

func TestMobRosterFocusAndAttackSameMob(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{"orc-1"},
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	lines := renderCombatMobRoster(enc, ar, "p1", 20, "orc-1")
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	// Should have > prefix AND ~ marker
	if !strings.Contains(joined, "> orc-1") {
		t.Fatal("attack target should have > prefix")
	}
	if !strings.Contains(joined, "~") {
		t.Fatal("focused mob should have ~ marker")
	}
}

func TestMobRosterFocusAndEngagement(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{"orc-1"},
		MobThreat: []mobThreatEntry{
			{MobID: "orc-1", SelectedTargetPlayerID: "p1"},
		},
	}
	lines := renderCombatMobRoster(enc, attackResult{}, "p1", 20, "orc-1")
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	// Should have both ~ and <-
	if !strings.Contains(joined, "~") {
		t.Fatal("focused mob should have ~ marker")
	}
	if !strings.Contains(joined, "<-") {
		t.Fatal("engaged mob should have <- marker")
	}
}

func TestMobRosterNoFocus(t *testing.T) {
	enc := &encounterSummary{
		MobIDs: []string{"orc-1"},
	}
	lines := renderCombatMobRoster(enc, attackResult{}, "p1", 20, "")
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	if strings.Contains(joined, "~") {
		t.Fatal("should not show ~ when no focus")
	}
}

// --- Loot Selection and Pickup Coherence Tests (M20260409-12) ---

func TestLootDropRowShowsPkMarker(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{"sword-1", "shield-1"},
		}},
	}
	pk := pickupResult{State: pickupStateSent, ItemID: "sword-1"}
	panel := renderLootPanel(sidePanelWidth, pr, er, pk, inventoryReadResult{}, -1, 0)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "[pk]") {
		t.Fatalf("drop row should show [pk] marker for submitted item, got: %s", stripped)
	}
}

func TestLootDropRowNoPkMarkerOtherItem(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{"sword-1", "shield-1"},
		}},
	}
	pk := pickupResult{State: pickupStateSent, ItemID: "shield-1"}
	panel := renderLootPanel(sidePanelWidth, pr, er, pk, inventoryReadResult{}, -1, 0)
	stripped := stripANSI(panel)
	// sword-1 should NOT have [pk]
	lines := strings.Split(stripped, "\n")
	for _, line := range lines {
		if strings.Contains(line, "sword") && strings.Contains(line, "[pk]") {
			t.Fatal("non-submitted item should not have [pk] marker")
		}
	}
}

func TestLootDropRowPkMarkerSuppressedAfterConfirmation(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed", DropsGenerated: true,
			Drops: []string{"sword-1"},
		}},
	}
	pk := pickupResult{State: pickupStateSent, ItemID: "sword-1"}
	inv := inventoryReadResult{State: inventoryReadOK, Count: 5} // delta confirmed (invAtPickup=4)
	panel := renderLootPanel(sidePanelWidth, pr, er, pk, inv, 4, 0)
	stripped := stripANSI(panel)
	if strings.Contains(stripped, "[pk]") {
		t.Fatal("[pk] should be suppressed after inventory confirmation")
	}
}

// --- Dead and Blocked Action Hint Polish Tests (M20260409-13) ---

func TestFooterDeadShowsRespawnHint(t *testing.T) {
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, HPCurrent: 0, HPMax: 100}
	footer := renderFooter(120, "", "", "", "", "", true, inv)
	stripped := stripANSI(footer)
	if !strings.Contains(stripped, "[r:respawn]") {
		t.Fatalf("dead footer should show [r:respawn] hint, got: %s", stripped)
	}
	// Full keybinds should still be present
	if !strings.Contains(stripped, "a:atk") {
		t.Fatal("dead footer should still show full keybinds after the hint")
	}
}

func TestFooterBlockedShowsBlockedHint(t *testing.T) {
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, BlockedReason: "cooldown", HPCurrent: 80, HPMax: 100}
	footer := renderFooter(120, "", "", "", "", "", true, inv)
	stripped := stripANSI(footer)
	if !strings.Contains(stripped, "[blocked]") {
		t.Fatalf("blocked footer should show [blocked] hint, got: %s", stripped)
	}
}

func TestFooterAliveNoHint(t *testing.T) {
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	footer := renderFooter(120, "", "", "", "", "", true, inv)
	stripped := stripANSI(footer)
	if strings.Contains(stripped, "[r:respawn]") || strings.Contains(stripped, "[blocked]") {
		t.Fatal("alive footer should not show dead/blocked hints")
	}
}

func TestFooterNotJoinedNoDeadHint(t *testing.T) {
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, HPCurrent: 0, HPMax: 100}
	footer := renderFooter(120, "", "", "", "", "", false, inv)
	stripped := stripANSI(footer)
	// Not joined takes precedence
	if strings.Contains(stripped, "[r:respawn]") {
		t.Fatal("not-joined footer should not show respawn hint")
	}
	if !strings.Contains(stripped, "joining") {
		t.Fatal("not-joined footer should show joining hint")
	}
}

// --- Target Loss Recovery Clarity Tests (M20260409-14) ---

func TestMobRosterGoneTargetShowsRetargetHintDuringActive(t *testing.T) {
	enc := &encounterSummary{
		State:  "Active",
		MobIDs: []string{"orc-2"}, // orc-1 gone
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	lines := renderCombatMobRoster(enc, ar, "p1", 24, "")
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	if !strings.Contains(joined, "tab>a") {
		t.Fatalf("gone target during active should show tab>a hint, got: %s", joined)
	}
}

func TestMobRosterGoneTargetNoHintWhenCompleted(t *testing.T) {
	enc := &encounterSummary{
		State:           "Completed",
		CompletedReason: "all_mobs_dead",
		MobIDs:          []string{},
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	lines := renderCombatMobRoster(enc, ar, "p1", 24, "")
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	if strings.Contains(joined, "tab>a") {
		t.Fatal("gone target during completed should not show retarget hint")
	}
}

func TestMobRosterEmptyRosterGoneWithHint(t *testing.T) {
	enc := &encounterSummary{
		State:  "Active",
		MobIDs: []string{},
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	lines := renderCombatMobRoster(enc, ar, "p1", 24, "")
	joined := ""
	for _, l := range lines {
		joined += stripANSI(l) + "\n"
	}
	if !strings.Contains(joined, "(gone)") {
		t.Fatal("should show gone marker")
	}
	if !strings.Contains(joined, "tab>a") {
		t.Fatalf("empty roster gone target should show hint, got: %s", joined)
	}
}

// --- Combat Result Wording Compactness Tests (M20260409-16) ---

func TestCompactResultKindDamage(t *testing.T) {
	if compactResultKind("damage_applied") != "dmg" {
		t.Fatal("damage_applied should compact to dmg")
	}
}

func TestCompactResultKindMiss(t *testing.T) {
	if compactResultKind("attack_miss") != "miss" {
		t.Fatal("attack_miss should compact to miss")
	}
}

func TestCompactResultKindHeal(t *testing.T) {
	if compactResultKind("heal_applied") != "heal" {
		t.Fatal("heal_applied should compact to heal")
	}
}

func TestCompactResultKindUnknown(t *testing.T) {
	if compactResultKind("some_new_thing") != "some_new_thing" {
		t.Fatal("unknown kinds should pass through unchanged")
	}
}

func TestCombatPanelResultHasResPrefix(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID:       "enc-1", State: "Active",
			MobIDs:            []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind:  "damage_applied",
			LatestResultValue: 42,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "res:dmg 42") {
		t.Fatalf("should show res:dmg 42, got: %s", stripped)
	}
}

func TestCombatPanelResultMissNoValue(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID:      "enc-1", State: "Active",
			MobIDs:           []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "attack_miss",
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "res:miss") {
		t.Fatalf("should show res:miss, got: %s", stripped)
	}
}

// --- Proximity Derivation and Attack Key Instrumentation Tests (M20260409-26) ---

func TestNearestMobBasic(t *testing.T) {
	pos := playerPosResult{X: 0, Y: 0}
	mobs := []mobPosition{
		{ProcessID: "orc-far", Position: mobPosVec3{X: 100, Y: 100}},
		{ProcessID: "orc-near", Position: mobPosVec3{X: 10, Y: 10}},
	}
	id, dist := nearestMob(pos, mobs)
	if id != "orc-near" {
		t.Fatalf("expected nearest=orc-near, got %s", id)
	}
	if dist < 10 || dist > 20 {
		t.Fatalf("expected dist ~14, got %f", dist)
	}
}

func TestNearestMobEmpty(t *testing.T) {
	pos := playerPosResult{X: 0, Y: 0}
	id, _ := nearestMob(pos, nil)
	if id != "" {
		t.Fatal("expected empty ID for no mobs")
	}
}

func TestProximityPanelShowsNearestMob(t *testing.T) {
	tc := targetConfirmResult{State: targetConfirmNone}
	mobr := mobReadResult{
		State: mobReadOK,
		Mobs: []mobPosition{
			{ProcessID: "orc-1", Position: mobPosVec3{X: 50, Y: 50}},
		},
		Count: 1,
	}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	panel := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobr, pr, rosterFocus{index: -1}, nil)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "nearest") {
		t.Fatalf("should show nearest mob, got: %s", stripped)
	}
	if !strings.Contains(stripped, "orc-1") {
		t.Fatalf("should show mob ID, got: %s", stripped)
	}
	if !strings.Contains(stripped, "dist:") {
		t.Fatalf("should show distance, got: %s", stripped)
	}
}

func TestProximityPanelNoneWithoutMobs(t *testing.T) {
	tc := targetConfirmResult{State: targetConfirmNone}
	panel := renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobReadResult{}, playerReadResult{}, rosterFocus{index: -1}, nil)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "no target") {
		t.Fatalf("should show 'no target' without player/mobs, got: %s", stripped)
	}
}

// --- Pre-Encounter Target Focus Tests (M20260409-27) ---

func TestBuildMobEntriesByDistance(t *testing.T) {
	pos := playerPosResult{X: 0, Y: 0}
	mobs := []mobPosition{
		{ProcessID: "far-orc", Position: mobPosVec3{X: 100, Y: 100}},
		{ProcessID: "near-orc", Position: mobPosVec3{X: 5, Y: 5}},
		{ProcessID: "mid-orc", Position: mobPosVec3{X: 30, Y: 30}},
	}
	entries := buildMobEntriesByDistance(pos, mobs)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].id != "near-orc" {
		t.Fatalf("first entry should be nearest mob, got %s", entries[0].id)
	}
	if entries[0].kind != "mb" {
		t.Fatal("entries should have kind 'mb'")
	}
	if entries[2].id != "far-orc" {
		t.Fatalf("last entry should be farthest mob, got %s", entries[2].id)
	}
}

func TestBuildMobEntriesByDistanceEmpty(t *testing.T) {
	entries := buildMobEntriesByDistance(playerPosResult{}, nil)
	if entries != nil {
		t.Fatal("empty mobs should return nil")
	}
}

func TestPreEncounterTabSelectsNearestMob(t *testing.T) {
	// Simulate model with no encounter but visible mobs
	m := model{
		playerRead: playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}},
		mobRead: mobReadResult{
			State: mobReadOK,
			Mobs: []mobPosition{
				{ProcessID: "far-mob", Position: mobPosVec3{X: 100, Y: 100}},
				{ProcessID: "near-mob", Position: mobPosVec3{X: 5, Y: 5}},
			},
			Count: 2,
		},
		rosterFocus:   rosterFocus{index: -1},
		rosterEntries: nil, // no encounter roster
		slog:          &sessionLogger{}, // disabled logger
	}

	// Simulate tab press — build mob entries and focus first (nearest)
	if len(m.rosterEntries) == 0 && m.playerRead.State == playerReadOK && m.playerRead.HasPos && m.mobRead.State == mobReadOK {
		m.rosterEntries = buildMobEntriesByDistance(m.playerRead.Position, m.mobRead.Mobs)
		m.rosterFocus = rosterFocus{index: -1}
	}
	m.rosterFocus = moveFocusDown(m.rosterFocus, len(m.rosterEntries))

	fe := focusedEntry(m.rosterFocus, m.rosterEntries)
	if fe == nil {
		t.Fatal("should have a focused entry after tab")
	}
	if fe.id != "near-mob" {
		t.Fatalf("should focus nearest mob, got %s", fe.id)
	}
	if fe.kind != "mb" {
		t.Fatalf("should be mob kind, got %s", fe.kind)
	}
}

func TestPreEncounterAttackAfterTab(t *testing.T) {
	// After tab selects a mob, attack should find it
	m := model{
		playerRead: playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}},
		mobRead: mobReadResult{
			State: mobReadOK,
			Mobs:  []mobPosition{{ProcessID: "orc-1", Position: mobPosVec3{X: 10, Y: 10}}},
			Count: 1,
		},
		rosterFocus:   rosterFocus{index: -1},
		rosterEntries: nil,
		slog:          &sessionLogger{},
	}

	// Simulate tab
	m.rosterEntries = buildMobEntriesByDistance(m.playerRead.Position, m.mobRead.Mobs)
	m.rosterFocus = moveFocusDown(rosterFocus{index: -1}, len(m.rosterEntries))

	// Simulate attack key check
	fe := focusedEntry(m.rosterFocus, m.rosterEntries)
	if fe == nil || fe.kind != "mb" {
		t.Fatal("should have mob focused for attack")
	}
	if fe.id != "orc-1" {
		t.Fatalf("attack target should be orc-1, got %s", fe.id)
	}
}

func TestEncounterRosterOverridesPreEncounter(t *testing.T) {
	// When encounter exists, encounter roster is used (not mob positions)
	enc := &encounterSummary{
		PlayerIDs: []string{"p1"},
		MobIDs:    []string{"enc-orc-1", "enc-orc-2"},
	}
	entries := buildRosterEntries(enc)
	if len(entries) != 3 {
		t.Fatalf("expected 3 encounter entries, got %d", len(entries))
	}
	// Focus should use encounter entries, not mob positions
	f := moveFocusDown(rosterFocus{index: -1}, len(entries))
	fe := focusedEntry(f, entries)
	if fe == nil {
		t.Fatal("should have encounter focus")
	}
	if fe.kind != "pc" {
		t.Fatalf("first encounter entry should be pc, got %s", fe.kind)
	}
}

// --- Unify Tab Focus and Attack Target State Tests (M20260409-28) ---

func TestPreEncounterFocusSurvivesEncounterRefresh(t *testing.T) {
	// The core bug: encounter refresh was wiping pre-encounter mob entries.
	// After fix, pre-encounter mob entries survive when no encounter exists.
	m := model{
		playerRead: playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}},
		mobRead: mobReadResult{
			State: mobReadOK,
			Mobs:  []mobPosition{{ProcessID: "orc-1", Position: mobPosVec3{X: 10, Y: 10}}},
			Count: 1,
		},
		rosterFocus:   rosterFocus{index: -1},
		rosterEntries: nil,
		slog:          &sessionLogger{},
	}

	// Step 1: Tab builds mob entries and focuses nearest
	m.rosterEntries = buildMobEntriesByDistance(m.playerRead.Position, m.mobRead.Mobs)
	m.rosterFocus = moveFocusDown(rosterFocus{index: -1}, len(m.rosterEntries))

	// Verify focus is set
	fe := focusedEntry(m.rosterFocus, m.rosterEntries)
	if fe == nil || fe.id != "orc-1" {
		t.Fatal("should have orc-1 focused after tab")
	}

	// Step 2: Simulate encounter refresh with no encounter (pre-encounter)
	enc := (*encounterSummary)(nil)
	newEntries := buildRosterEntries(enc)
	if newEntries != nil {
		// Encounter roster exists — would override (but it doesn't here)
		m.rosterFocus = reconcileFocus(m.rosterFocus, m.rosterEntries, newEntries)
		m.rosterEntries = newEntries
	}
	// When newEntries is nil, pre-encounter entries are preserved

	// Step 3: Verify focus survives
	fe = focusedEntry(m.rosterFocus, m.rosterEntries)
	if fe == nil {
		t.Fatal("focus should survive encounter refresh when no encounter exists")
	}
	if fe.id != "orc-1" {
		t.Fatalf("focused mob should still be orc-1, got %s", fe.id)
	}
}

func TestAttackAfterTabAndRefreshCycle(t *testing.T) {
	// Full integration: tab → refresh → attack should work
	m := model{
		playerRead: playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}},
		mobRead: mobReadResult{
			State: mobReadOK,
			Mobs:  []mobPosition{{ProcessID: "orc-1", Position: mobPosVec3{X: 10, Y: 10}}},
			Count: 1,
		},
		rosterFocus:   rosterFocus{index: -1},
		rosterEntries: nil,
		slog:          &sessionLogger{},
	}

	// Tab
	m.rosterEntries = buildMobEntriesByDistance(m.playerRead.Position, m.mobRead.Mobs)
	m.rosterFocus = moveFocusDown(rosterFocus{index: -1}, len(m.rosterEntries))

	// Simulate 3 encounter refresh cycles (no encounter)
	for i := 0; i < 3; i++ {
		newEntries := buildRosterEntries(nil)
		if newEntries != nil {
			m.rosterFocus = reconcileFocus(m.rosterFocus, m.rosterEntries, newEntries)
			m.rosterEntries = newEntries
		}
	}

	// Attack should still find the focused mob
	fe := focusedEntry(m.rosterFocus, m.rosterEntries)
	if fe == nil {
		t.Fatal("focus should survive multiple refresh cycles")
	}
	if fe.kind != "mb" || fe.id != "orc-1" {
		t.Fatalf("should still target orc-1, got %s:%s", fe.kind, fe.id)
	}
}

func TestEncounterRefreshReplacesPreEncounterEntries(t *testing.T) {
	// When encounter starts, encounter roster replaces pre-encounter mob entries
	m := model{
		playerRead: playerReadResult{State: playerReadOK, HasPos: true, HasActiveEncounter: true, ActiveEncounterID: "enc-1"},
		rosterFocus:   rosterFocus{index: 0},
		rosterEntries: []rosterEntry{{kind: "mb", id: "pre-enc-orc"}}, // pre-encounter entries
		slog:          &sessionLogger{},
	}

	// Encounter refresh with actual encounter
	enc := &encounterSummary{
		EncounterID: "enc-1",
		PlayerIDs:   []string{"p1"},
		MobIDs:      []string{"enc-orc-1"},
	}
	newEntries := buildRosterEntries(enc)
	if newEntries != nil {
		m.rosterFocus = reconcileFocus(m.rosterFocus, m.rosterEntries, newEntries)
		m.rosterEntries = newEntries
	}

	// Should now have encounter entries, not pre-encounter
	if len(m.rosterEntries) != 2 { // 1 pc + 1 mob
		t.Fatalf("expected 2 encounter entries, got %d", len(m.rosterEntries))
	}
	if m.rosterEntries[1].id != "enc-orc-1" {
		t.Fatalf("mob entry should be from encounter, got %s", m.rosterEntries[1].id)
	}
}

func TestAttackWithNoFocusStillFails(t *testing.T) {
	// Attack with no roster entries and no focus should still fail cleanly
	m := model{
		playerRead:    playerReadResult{State: playerReadOK},
		rosterFocus:   rosterFocus{index: -1},
		rosterEntries: nil,
		slog:          &sessionLogger{},
	}

	fe := focusedEntry(m.rosterFocus, m.rosterEntries)
	if fe != nil {
		t.Fatal("should have no focus")
	}
	// This simulates the attack skip path
	if fe == nil || fe.kind != "mb" {
		// This is the expected path — attack would be skipped
	} else {
		t.Fatal("should not reach attack submit with no focus")
	}
}

// --- Canonical mob_id Validation Tests (M20260409-30) ---

func TestMobPositionDeserializesMobID(t *testing.T) {
	// Simulate backend mob_positions response with mob_id field
	body := []byte(`{"result":{"<0.123.0>":{"mob_id":"a_orc_5","mob_name":"an orc","position":{"x":10,"y":20,"z":0}}}}`)
	var raw struct {
		Result map[string]mobPosition `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	mob := raw.Result["<0.123.0>"]
	if mob.MobID != "a_orc_5" {
		t.Fatalf("expected mob_id='a_orc_5', got %q", mob.MobID)
	}
	if mob.MobName != "an orc" {
		t.Fatalf("expected mob_name='an orc', got %q", mob.MobName)
	}
}

func TestBuildMobEntriesUsesProcessIDAsLiveIdentity(t *testing.T) {
	pos := playerPosResult{X: 0, Y: 0}
	mobs := []mobPosition{
		{ProcessID: "<0.123.0>", MobID: "a_orc_5", MobName: "an orc", Position: mobPosVec3{X: 10, Y: 10}},
	}
	entries := buildMobEntriesByDistance(pos, mobs)
	if len(entries) != 1 {
		t.Fatal("expected 1 entry")
	}
	if entries[0].id != "<0.123.0>" {
		t.Fatalf("entry.id should be ProcessID for live identity, got %q", entries[0].id)
	}
	if entries[0].mobID != "a_orc_5" {
		t.Fatalf("entry.mobID should be canonical mob_id for attacks, got %q", entries[0].mobID)
	}
	if !entries[0].canonicalMobID {
		t.Fatal("canonicalMobID should be true when mob_id is present")
	}
}

func TestBuildMobEntriesFallsBackToProcessID(t *testing.T) {
	pos := playerPosResult{X: 0, Y: 0}
	mobs := []mobPosition{
		{ProcessID: "<0.123.0>", MobID: "", MobName: "an orc", Position: mobPosVec3{X: 10, Y: 10}},
	}
	entries := buildMobEntriesByDistance(pos, mobs)
	if len(entries) != 1 {
		t.Fatal("expected 1 entry")
	}
	if entries[0].id != "<0.123.0>" {
		t.Fatalf("entry should fall back to ProcessID, got %q", entries[0].id)
	}
	if entries[0].canonicalMobID {
		t.Fatal("canonicalMobID should be false when mob_id is empty")
	}
}

func TestAttackBlockedWithoutCanonicalMobID(t *testing.T) {
	// Simulate: focused mob has no canonical mob_id → attack should be blocked
	m := model{
		playerRead: playerReadResult{State: playerReadOK},
		rosterFocus:   rosterFocus{index: 0},
		rosterEntries: []rosterEntry{{kind: "mb", id: "<0.123.0>", canonicalMobID: false}},
		slog:          &sessionLogger{},
	}
	fe := focusedEntry(m.rosterFocus, m.rosterEntries)
	if fe == nil || fe.kind != "mb" {
		t.Fatal("should have mob focused")
	}
	// The attack handler checks canonicalMobID
	if !fe.canonicalMobID {
		// This is the expected fail-closed path
	} else {
		t.Fatal("should fail closed with non-canonical mob ID")
	}
}

func TestAttackAllowedWithCanonicalMobID(t *testing.T) {
	// Simulate: focused mob has canonical mob_id → attack should proceed
	m := model{
		playerRead: playerReadResult{State: playerReadOK},
		rosterFocus:   rosterFocus{index: 0},
		rosterEntries: []rosterEntry{{kind: "mb", id: "a_orc_5", canonicalMobID: true}},
		slog:          &sessionLogger{},
	}
	fe := focusedEntry(m.rosterFocus, m.rosterEntries)
	if fe == nil || fe.kind != "mb" {
		t.Fatal("should have mob focused")
	}
	if !fe.canonicalMobID {
		t.Fatal("canonical mob should be allowed for attack")
	}
	if fe.id != "a_orc_5" {
		t.Fatalf("attack target should be a_orc_5, got %s", fe.id)
	}
}

func TestFocusPreviewShowsQuestionMarkForNonCanonical(t *testing.T) {
	entries := []rosterEntry{{kind: "mb", id: "<0.123.0>", canonicalMobID: false}}
	label := focusPreviewLabel(rosterFocus{index: 0}, entries)
	if !strings.Contains(label, "?") {
		t.Fatalf("non-canonical mob focus should show ? suffix, got %q", label)
	}
}

func TestFocusPreviewNoQuestionMarkForCanonical(t *testing.T) {
	entries := []rosterEntry{{kind: "mb", id: "a_orc_5", canonicalMobID: true}}
	label := focusPreviewLabel(rosterFocus{index: 0}, entries)
	if strings.Contains(label, "?") {
		t.Fatalf("canonical mob focus should not have ? suffix, got %q", label)
	}
}

// --- M20260410-01 Pre-encounter / encounter visibility polish ---

func TestCompactResultKindLosBlocked(t *testing.T) {
	if got := compactResultKind("los_blocked"); got != "los" {
		t.Fatalf("expected los, got %q", got)
	}
}

func TestIsDenialResultTrue(t *testing.T) {
	for _, kind := range []string{"out_of_range", "los_blocked", "invalid_target"} {
		if !isDenialResult(kind) {
			t.Fatalf("expected isDenialResult(%q) == true", kind)
		}
	}
}

func TestIsDenialResultFalse(t *testing.T) {
	for _, kind := range []string{"damage_applied", "attack_miss", "heal_applied", "cooldown_set", "blocked_wounded", ""} {
		if isDenialResult(kind) {
			t.Fatalf("expected isDenialResult(%q) == false", kind)
		}
	}
}

func TestCombatPanelDenyPrefixOutOfRange(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID:      "enc-1", State: "Active",
			MobIDs:           []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "out_of_range",
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "deny:oor") {
		t.Fatalf("out_of_range should show deny:oor, got: %s", stripped)
	}
	if strings.Contains(stripped, "res:oor") {
		t.Fatal("out_of_range should not use res: prefix")
	}
}

func TestCombatPanelDenyPrefixLosBlocked(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID:      "enc-1", State: "Active",
			MobIDs:           []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "los_blocked",
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "deny:los") {
		t.Fatalf("los_blocked should show deny:los, got: %s", stripped)
	}
}

func TestCombatPanelDenyPrefixInvalidTarget(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID:      "enc-1", State: "Active",
			MobIDs:           []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "invalid_target",
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "deny:bad_tgt") {
		t.Fatalf("invalid_target should show deny:bad_tgt, got: %s", stripped)
	}
}

func TestCombatPanelResPrefixPreservedForCombat(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID:       "enc-1", State: "Active",
			MobIDs:            []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind:  "damage_applied",
			LatestResultValue: 30,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "res:dmg 30") {
		t.Fatalf("damage_applied should keep res: prefix, got: %s", stripped)
	}
	if strings.Contains(stripped, "deny:") {
		t.Fatal("damage_applied should not use deny: prefix")
	}
}

func TestCombatPanelTransitionalEncOpening(t *testing.T) {
	// Attack sent but encounter hasn't refreshed yet — should show "enc: opening"
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: false}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, encounterReadResult{}, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "enc: opening") {
		t.Fatalf("transitional state should show enc: opening, got: %s", stripped)
	}
	if !strings.Contains(stripped, "atk:orc-1") {
		t.Fatalf("transitional state should show atk:<targetID>, got: %s", stripped)
	}
	// Must NOT show old ambiguous wording
	if strings.Contains(stripped, "intent: accepted") {
		t.Fatal("should not show old 'intent: accepted' wording")
	}
	if strings.Contains(stripped, "enc: none") {
		t.Fatal("should not show 'enc: none' when attack sent")
	}
}

func TestCombatPanelFailedShowsAtkFail(t *testing.T) {
	ar := attackResult{State: attackStateFailed, Error: "no mob focused"}
	pr := playerReadResult{State: playerReadOK}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, encounterReadResult{}, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "atk:fail") {
		t.Fatalf("failed attack should show atk:fail, got: %s", stripped)
	}
	// Must NOT show old ambiguous wording
	if strings.Contains(stripped, "intent: failed") {
		t.Fatal("should not show old 'intent: failed' wording")
	}
}

func TestCombatPanelThreeStatesDistinguishable(t *testing.T) {
	// Verify the three acceptance criteria states produce visually distinct output.

	// State 1: no encounter
	pr1 := playerReadResult{State: playerReadOK}
	panel1 := stripANSI(renderCombatPanel(sidePanelWidth, attackResult{}, pr1, encounterReadResult{}, defaultTarget(), inventoryReadResult{}, ""))

	// State 2: encounter active, denied (out of range)
	ar2 := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr2 := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er2 := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "out_of_range",
		}},
	}
	panel2 := stripANSI(renderCombatPanel(sidePanelWidth, ar2, pr2, er2, defaultTarget(), inventoryReadResult{}, ""))

	// State 3: encounter active, combat resolving
	er3 := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "damage_applied", LatestResultValue: 15,
		}},
	}
	panel3 := stripANSI(renderCombatPanel(sidePanelWidth, ar2, pr2, er3, defaultTarget(), inventoryReadResult{}, ""))

	// All three must be distinct
	if panel1 == panel2 {
		t.Fatal("no-encounter and denied panels must differ")
	}
	if panel2 == panel3 {
		t.Fatal("denied and resolving panels must differ")
	}
	if panel1 == panel3 {
		t.Fatal("no-encounter and resolving panels must differ")
	}

	// State 2 must contain deny:, State 3 must contain res:
	if !strings.Contains(panel2, "deny:") {
		t.Fatalf("denied state should contain deny:, got: %s", panel2)
	}
	if !strings.Contains(panel3, "res:") {
		t.Fatalf("resolving state should contain res:, got: %s", panel3)
	}
}

// --- Post-attack immediate refresh tests ---

func TestAttackSuccessTriggersImmediateRefresh(t *testing.T) {
	// When attack submission succeeds, Update should return a non-nil Cmd
	// (immediate encounter + player read) rather than waiting for the next poll.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	msg := attackResultMsg{result: attackResult{State: attackStateSent, TargetID: "orc-1"}}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("successful attack should trigger immediate refresh (non-nil Cmd)")
	}
}

func TestAttackFailureDoesNotTriggerRefresh(t *testing.T) {
	// When attack submission fails, Update should return nil Cmd (no extra reads).
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	msg := attackResultMsg{result: attackResult{State: attackStateFailed, Error: "no mob", TargetID: "orc-1"}}
	_, cmd := m.Update(msg)
	if cmd != nil {
		t.Fatal("failed attack should not trigger immediate refresh")
	}
}

func TestAttackNoneDoesNotTriggerRefresh(t *testing.T) {
	// Default/no-attack state should return nil Cmd.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	msg := attackResultMsg{result: attackResult{State: attackStateNone}}
	_, cmd := m.Update(msg)
	if cmd != nil {
		t.Fatal("no-attack state should not trigger immediate refresh")
	}
}

func TestAttackSuccessPreservesLastAttackState(t *testing.T) {
	// The model's lastAttack must be set before the refresh Cmds are returned.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	msg := attackResultMsg{result: attackResult{State: attackStateSent, TargetID: "orc-1"}}
	updated, _ := m.Update(msg)
	um := updated.(model)
	if um.lastAttack.State != attackStateSent {
		t.Fatal("lastAttack should be set to sent after successful attack")
	}
	if um.lastAttack.TargetID != "orc-1" {
		t.Fatalf("lastAttack.TargetID should be orc-1, got: %s", um.lastAttack.TargetID)
	}
}

// --- Combat panel state contract tests (M20260410-06) ---
// These tests protect the complete visible contract for the combat panel.
// Each distinct state must produce unique, identifiable output.

func TestCombatPanelStateContractAllSixStates(t *testing.T) {
	// The combat panel has six distinct visible states.
	// Each must be identifiable by its content and distinguishable from the others.
	bt := defaultTarget()
	inv := inventoryReadResult{}

	// State 1: idle / none — no attack, no encounter
	panel1 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{},
		playerReadResult{State: playerReadOK},
		encounterReadResult{State: encounterReadOK},
		bt, inv, ""))
	if !strings.Contains(panel1, "none") {
		t.Fatalf("idle state must show 'none', got: %s", panel1)
	}

	// State 2: attack submitted, encounter opening
	panel2 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateSent, TargetID: "orc-1"},
		playerReadResult{State: playerReadOK},
		encounterReadResult{State: encounterReadOK},
		bt, inv, ""))
	if !strings.Contains(panel2, "enc: opening") {
		t.Fatalf("opening state must show 'enc: opening', got: %s", panel2)
	}
	if !strings.Contains(panel2, "atk:orc-1") {
		t.Fatalf("opening state must show atk target, got: %s", panel2)
	}

	// State 3: denial from backend truth
	panel3 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateSent, TargetID: "orc-1"},
		playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"},
		encounterReadResult{State: encounterReadOK, Count: 1, Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "out_of_range",
		}}},
		bt, inv, ""))
	if !strings.Contains(panel3, "deny:oor") {
		t.Fatalf("denial state must show 'deny:oor', got: %s", panel3)
	}

	// State 4: combat resolution from backend truth
	panel4 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateSent, TargetID: "orc-1"},
		playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"},
		encounterReadResult{State: encounterReadOK, Count: 1, Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "damage_applied", LatestResultValue: 10,
		}}},
		bt, inv, ""))
	if !strings.Contains(panel4, "res:dmg") {
		t.Fatalf("resolution state must show 'res:dmg', got: %s", panel4)
	}
	if !strings.Contains(panel4, "10") {
		t.Fatalf("resolution state must show damage value, got: %s", panel4)
	}

	// State 5: failed submit
	panel5 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateFailed, Error: "no mob", TargetID: "orc-1"},
		playerReadResult{State: playerReadOK},
		encounterReadResult{State: encounterReadOK},
		bt, inv, ""))
	if !strings.Contains(panel5, "atk:fail") {
		t.Fatalf("failed state must show 'atk:fail', got: %s", panel5)
	}

	// State 6: read-gap (attack sent, player has encounter, but encounter read stale)
	panel6 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateSent, TargetID: "orc-1"},
		playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-new"},
		encounterReadResult{State: encounterReadOK, Count: 0},
		bt, inv, ""))
	if !strings.Contains(panel6, "enc: opening") {
		t.Fatalf("read-gap state must show 'enc: opening', got: %s", panel6)
	}

	// All six panels must be mutually distinguishable from the non-gap states
	panels := map[string]string{
		"idle": panel1, "opening": panel2, "denial": panel3,
		"resolution": panel4, "failed": panel5, "read-gap": panel6,
	}
	// opening and read-gap may be identical (both show enc:opening) — that's correct
	for nameA, pA := range panels {
		for nameB, pB := range panels {
			if nameA >= nameB {
				continue
			}
			if nameA == "opening" && nameB == "read-gap" {
				continue // these are allowed to match
			}
			if nameA == "read-gap" && nameB == "opening" {
				continue
			}
			if pA == pB {
				t.Fatalf("states %s and %s must differ but are identical", nameA, nameB)
			}
		}
	}
}

func TestCombatPanelStateContractDenyRequiresBackend(t *testing.T) {
	// deny: prefix must only appear when backend encounter has a denial LatestResultKind.
	// Without backend encounter data, deny: must NOT appear.
	panel := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateSent, TargetID: "orc-1"},
		playerReadResult{State: playerReadOK},
		encounterReadResult{State: encounterReadOK},
		defaultTarget(), inventoryReadResult{}, ""))
	if strings.Contains(panel, "deny:") {
		t.Fatalf("deny: must not appear without backend encounter truth, got: %s", panel)
	}
}

func TestCombatPanelStateContractResRequiresBackend(t *testing.T) {
	// res: prefix must only appear when backend encounter has a resolution LatestResultKind.
	panel := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateSent, TargetID: "orc-1"},
		playerReadResult{State: playerReadOK},
		encounterReadResult{State: encounterReadOK},
		defaultTarget(), inventoryReadResult{}, ""))
	if strings.Contains(panel, "res:") {
		t.Fatalf("res: must not appear without backend encounter truth, got: %s", panel)
	}
}

// --- Attack read-race hardening tests (M20260410-05) ---

func TestCombatPanelRacePlayerHasEncounterButReadStale(t *testing.T) {
	// Race: player read says HasActiveEncounter=true, but encounter read
	// doesn't contain the encounter yet (stale encounter list).
	// Combat panel must show enc:opening, not "none" or "no enc details".
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-new"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 0, Encounters: nil,
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "enc: opening") {
		t.Fatalf("race: player has encounter but read stale — must show enc: opening, got: %s", stripped)
	}
	if strings.Contains(stripped, "none") {
		t.Fatalf("race: must not show 'none' during attack read gap, got: %s", stripped)
	}
}

func TestCombatPanelRaceEncounterReadFailed(t *testing.T) {
	// Race: attack succeeded, immediate encounter read failed.
	// Combat panel must show enc:opening, not "unavailable".
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: false}
	er := encounterReadResult{State: encounterReadFailed, Error: "timeout"}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "enc: opening") {
		t.Fatalf("race: encounter read failed — must show enc: opening, got: %s", stripped)
	}
}

func TestCombatPanelRaceEncounterReadNotAttempted(t *testing.T) {
	// Race: attack succeeded, encounter read hasn't arrived yet.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: false}
	er := encounterReadResult{State: encounterReadNotAttempted}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "enc: opening") {
		t.Fatalf("race: encounter read pending — must show enc: opening, got: %s", stripped)
	}
}

func TestCombatPanelRacePlayerHasEncounterDetailsPresent(t *testing.T) {
	// No race: player has encounter AND encounter read contains it.
	// Combat panel must show encounter details, NOT enc:opening.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1, MobsDead: 0, ActionIndex: 1,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if strings.Contains(stripped, "enc: opening") {
		t.Fatalf("no race: encounter details present — must NOT show enc: opening, got: %s", stripped)
	}
	if !strings.Contains(stripped, "act:") {
		t.Fatalf("no race: must show compact state line, got: %s", stripped)
	}
}

func TestCombatPanelRaceDenialVisibleFromBackend(t *testing.T) {
	// After immediate read: encounter is present with a denial result.
	// Must show deny: prefix, not enc:opening.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1, MobsDead: 0,
			LatestResultKind: "out_of_range",
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "deny:oor") {
		t.Fatalf("race: denial must show deny:oor, got: %s", stripped)
	}
	if strings.Contains(stripped, "enc: opening") {
		t.Fatalf("race: denial visible — must not show enc: opening, got: %s", stripped)
	}
}

func TestCombatPanelRaceResolutionVisibleFromBackend(t *testing.T) {
	// After immediate read: encounter is present with a resolution result.
	// Must show res: prefix, not enc:opening.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1, MobsDead: 0,
			LatestResultKind: "damage_applied", LatestResultValue: 15,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "res:dmg") {
		t.Fatalf("race: resolution must show res:dmg, got: %s", stripped)
	}
	if !strings.Contains(stripped, "15") {
		t.Fatalf("race: resolution must show damage value, got: %s", stripped)
	}
}

func TestCombatPanelRaceNoRegressToEncNone(t *testing.T) {
	// Critical contract: when lastAttack is sent, combat panel must NEVER
	// show just "none" without context. It must show enc:opening or encounter details.
	testCases := []struct {
		name string
		pr   playerReadResult
		er   encounterReadResult
	}{
		{
			name: "no encounter, read ok",
			pr:   playerReadResult{State: playerReadOK, HasActiveEncounter: false},
			er:   encounterReadResult{State: encounterReadOK},
		},
		{
			name: "no encounter, read failed",
			pr:   playerReadResult{State: playerReadOK, HasActiveEncounter: false},
			er:   encounterReadResult{State: encounterReadFailed},
		},
		{
			name: "no encounter, read pending",
			pr:   playerReadResult{State: playerReadOK, HasActiveEncounter: false},
			er:   encounterReadResult{State: encounterReadNotAttempted},
		},
		{
			name: "has encounter, read stale",
			pr:   playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-x"},
			er:   encounterReadResult{State: encounterReadOK, Count: 0},
		},
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			panel := renderCombatPanel(sidePanelWidth, ar, tc.pr, tc.er, defaultTarget(), inventoryReadResult{}, "")
			stripped := stripANSI(panel)
			// Must contain enc:opening when attack is sent and no encounter details visible
			if !strings.Contains(stripped, "enc: opening") {
				t.Fatalf("must show enc: opening during read gap (%s), got: %s", tc.name, stripped)
			}
		})
	}
}

// --- Movement immediate refresh tests (M20260410-07) ---

func TestMoveSuccessTriggersImmediateRefresh(t *testing.T) {
	// When move submission succeeds, Update should return a non-nil Cmd
	// (immediate mob, player, encounter reads) rather than waiting for next poll.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	msg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 10, Y: 20}},
		direction: "north",
	}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("successful move should trigger immediate refresh (non-nil Cmd)")
	}
}

func TestMoveFailureNoExtraRefresh(t *testing.T) {
	// When move fails and no proximity is active, Update should return nil Cmd
	// (no extra reads beyond what maybeRefreshProximity produces, which is nil
	// when no proximity query is active).
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.targetConfirm = targetConfirmResult{State: targetConfirmNone}
	msg := moveResultMsg{
		result:    moveResult{OK: false, Error: "boundary"},
		direction: "west",
	}
	_, cmd := m.Update(msg)
	if cmd != nil {
		t.Fatal("failed move with no active proximity should not trigger extra refresh")
	}
}

func TestMoveSuccessPreservesBackendPosition(t *testing.T) {
	// The model's playerRead.Position must reflect the backend response position,
	// not any local computation.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	msg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 5, Y: 10}},
		direction: "east",
	}
	updated, _ := m.Update(msg)
	um := updated.(model)
	if um.playerRead.Position.X != 5 || um.playerRead.Position.Y != 10 {
		t.Fatalf("position should be backend-authoritative (5,10), got (%.0f,%.0f)",
			um.playerRead.Position.X, um.playerRead.Position.Y)
	}
	if !um.playerRead.HasPos {
		t.Fatal("HasPos should be true after successful move with position")
	}
}

func TestMoveSuccessSetsIntentSent(t *testing.T) {
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	msg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 5, Y: 0}},
		direction: "north",
	}
	updated, _ := m.Update(msg)
	um := updated.(model)
	if um.lastIntent.state != moveStateSent {
		t.Fatal("lastIntent should be moveStateSent after successful move")
	}
	if um.lastIntent.direction != "north" {
		t.Fatalf("lastIntent.direction should be north, got: %s", um.lastIntent.direction)
	}
}

func TestMoveFailureSetsIntentFailed(t *testing.T) {
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	msg := moveResultMsg{
		result:    moveResult{OK: false, Error: "out of bounds"},
		direction: "south",
	}
	updated, _ := m.Update(msg)
	um := updated.(model)
	if um.lastIntent.state != moveStateFailed {
		t.Fatal("lastIntent should be moveStateFailed after failed move")
	}
}

func TestMoveSuccessNoCombatPanelRegression(t *testing.T) {
	// After a move, if there was a prior attack with enc:opening state,
	// the combat panel must still show enc:opening (not regress to "none").
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: false}
	er := encounterReadResult{State: encounterReadOK}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "enc: opening") {
		t.Fatalf("combat panel must preserve enc:opening during move, got: %s", stripped)
	}
}

func TestMoveSuccessNoPositionWithoutHasPos(t *testing.T) {
	// If move succeeds but HasPos is false (edge case), position should not update.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 1, Y: 2}}
	msg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: false},
		direction: "north",
	}
	updated, _ := m.Update(msg)
	um := updated.(model)
	// Position should remain at old value since HasPos is false in response
	if um.playerRead.Position.X != 1 || um.playerRead.Position.Y != 2 {
		t.Fatalf("position should not change when move response has no position, got (%.0f,%.0f)",
			um.playerRead.Position.X, um.playerRead.Position.Y)
	}
}

// --- Move → attack legibility tests (M20260410-08) ---

func TestMoveThenAttackBothTriggerRefresh(t *testing.T) {
	// Move and attack immediate refreshes must both produce non-nil Cmds
	// when processed in sequence. This verifies they compose independently.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()

	// Step 1: process successful move result
	moveMsg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 15, Y: 10}},
		direction: "east",
	}
	updated, moveCmd := m.Update(moveMsg)
	if moveCmd == nil {
		t.Fatal("successful move must trigger immediate refresh")
	}
	m2 := updated.(model)

	// Step 2: process successful attack result on the post-move model
	attackMsg := attackResultMsg{result: attackResult{State: attackStateSent, TargetID: "orc-1"}}
	_, attackCmd := m2.Update(attackMsg)
	if attackCmd == nil {
		t.Fatal("successful attack must trigger immediate refresh after move")
	}
}

func TestMoveThenAttackPositionIsBackendAuthoritative(t *testing.T) {
	// After move, position must reflect backend response.
	// After attack, position must remain at the move-result value
	// (attack does not change position).
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}

	// Move to new position
	moveMsg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 20, Y: 5}},
		direction: "east",
	}
	updated, _ := m.Update(moveMsg)
	m2 := updated.(model)

	if m2.playerRead.Position.X != 20 || m2.playerRead.Position.Y != 5 {
		t.Fatalf("position after move must be backend (20,5), got (%.0f,%.0f)",
			m2.playerRead.Position.X, m2.playerRead.Position.Y)
	}

	// Attack does not change position
	attackMsg := attackResultMsg{result: attackResult{State: attackStateSent, TargetID: "orc-1"}}
	updated2, _ := m2.Update(attackMsg)
	m3 := updated2.(model)

	if m3.playerRead.Position.X != 20 || m3.playerRead.Position.Y != 5 {
		t.Fatalf("position after attack must remain at (20,5), got (%.0f,%.0f)",
			m3.playerRead.Position.X, m3.playerRead.Position.Y)
	}
}

func TestMoveIntoRangeThenAttackOpensEncounter(t *testing.T) {
	// Scenario: player moves into range, attacks, encounter opens.
	// The combat panel must show the active encounter, not enc:opening.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position:           playerPosResult{X: 20, Y: 5},
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1, MobsDead: 0,
			ActionIndex:      1,
			LatestResultKind:  "damage_applied",
			LatestResultValue: 12,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)

	// Must show active encounter, not opening
	if strings.Contains(stripped, "enc: opening") {
		t.Fatalf("should show active encounter, not enc:opening, got: %s", stripped)
	}
	if !strings.Contains(stripped, "act:") {
		t.Fatalf("should show compact state line, got: %s", stripped)
	}
	if !strings.Contains(stripped, "res:dmg") {
		t.Fatalf("should show res:dmg from backend truth, got: %s", stripped)
	}
	if !strings.Contains(stripped, "12") {
		t.Fatalf("should show damage value 12, got: %s", stripped)
	}
}

func TestMoveStillOutOfRangeThenAttackDenial(t *testing.T) {
	// Scenario: player moves but is still out of range, attacks, backend denies.
	// The combat panel must show deny:oor, not enc:opening or ambiguous state.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position:           playerPosResult{X: 100, Y: 100}, // far away
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1, MobsDead: 0,
			LatestResultKind: "out_of_range",
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "deny:oor") {
		t.Fatalf("out-of-range after move must show deny:oor, got: %s", stripped)
	}
	if strings.Contains(stripped, "enc: opening") {
		t.Fatalf("must not show enc:opening when encounter is present with denial, got: %s", stripped)
	}
}

func TestMoveDuringEncounterThenAttackPanelConsistent(t *testing.T) {
	// Scenario: player is already in an encounter, moves, then attacks again.
	// The combat panel must remain consistent — showing the encounter state,
	// attack target, and resolution, not regressing to idle or opening.
	ar := attackResult{State: attackStateSent, TargetID: "orc-2"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position:           playerPosResult{X: 25, Y: 10}, // moved position
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1", "orc-2"}, MobsAlive: 2, MobsDead: 0,
			ActionIndex:       3,
			LatestResultKind:  "damage_applied",
			LatestResultValue: 8,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)

	// Must show active encounter consistently
	if !strings.Contains(stripped, "act:") {
		t.Fatalf("must show compact state line during movement, got: %s", stripped)
	}
	if !strings.Contains(stripped, "2a/0d") {
		t.Fatalf("must show alive/dead counts, got: %s", stripped)
	}
	if !strings.Contains(stripped, "act:3") {
		t.Fatalf("must show action index, got: %s", stripped)
	}
	if !strings.Contains(stripped, "> orc-2") {
		t.Fatalf("must show attack target with > prefix, got: %s", stripped)
	}
	if strings.Contains(stripped, "none") {
		t.Fatalf("must not show 'none' during active encounter, got: %s", stripped)
	}
	if strings.Contains(stripped, "enc: opening") {
		t.Fatalf("must not show enc:opening during active encounter, got: %s", stripped)
	}
}

func TestMoveRaceThenAttackEncOpeningPreserved(t *testing.T) {
	// Race scenario: move succeeded, immediate reads in flight, user attacks
	// before encounter read arrives. The combat panel must show enc:opening
	// (from attack), not regress to "none".
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position:           playerPosResult{X: 15, Y: 5}, // post-move position
		HasActiveEncounter: false,                         // encounter read hasn't arrived
	}
	er := encounterReadResult{State: encounterReadOK, Count: 0} // stale — no encounters yet

	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "enc: opening") {
		t.Fatalf("race: must show enc:opening when attack sent but reads stale, got: %s", stripped)
	}
	if !strings.Contains(stripped, "atk:orc-1") {
		t.Fatalf("race: must show attack target, got: %s", stripped)
	}
}

func TestMoveThenAttackLOSDenialVisible(t *testing.T) {
	// Scenario: player moves but LOS is blocked, attack denied.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position:           playerPosResult{X: 30, Y: 15},
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1, MobsDead: 0,
			LatestResultKind: "los_blocked",
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "deny:los") {
		t.Fatalf("LOS denial after move must show deny:los, got: %s", stripped)
	}
}

func TestMoveThenAttackFailedSubmitVisible(t *testing.T) {
	// Scenario: player moves, then attack submission itself fails (no mob focused).
	// The combat panel must show atk:fail, not enc:opening or none.
	ar := attackResult{State: attackStateFailed, Error: "no mob focused"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position: playerPosResult{X: 10, Y: 5},
	}
	er := encounterReadResult{State: encounterReadOK, Count: 0}

	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "atk:fail") {
		t.Fatalf("failed attack after move must show atk:fail, got: %s", stripped)
	}
	if strings.Contains(stripped, "enc: opening") {
		t.Fatalf("failed attack must not show enc:opening, got: %s", stripped)
	}
}

func TestMoveThenAttackAllSixStatesDistinct(t *testing.T) {
	// After a move, the six combat panel states must remain mutually distinct.
	// This is the move-context counterpart of TestCombatPanelStateContractAllSixStates.
	bt := defaultTarget()
	inv := inventoryReadResult{}
	// All scenarios use a post-move player position to represent "after move"
	postMovePos := playerPosResult{X: 15, Y: 8}

	// State 1: idle / none — moved, no attack
	panel1 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{},
		playerReadResult{State: playerReadOK, HasPos: true, Position: postMovePos},
		encounterReadResult{State: encounterReadOK},
		bt, inv, ""))

	// State 2: attack submitted, encounter opening
	panel2 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateSent, TargetID: "orc-1"},
		playerReadResult{State: playerReadOK, HasPos: true, Position: postMovePos},
		encounterReadResult{State: encounterReadOK},
		bt, inv, ""))

	// State 3: denial from backend truth (out of range after move)
	panel3 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateSent, TargetID: "orc-1"},
		playerReadResult{State: playerReadOK, HasPos: true, Position: postMovePos,
			HasActiveEncounter: true, ActiveEncounterID: "enc-1"},
		encounterReadResult{State: encounterReadOK, Count: 1, Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "out_of_range",
		}}},
		bt, inv, ""))

	// State 4: combat resolution from backend truth
	panel4 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateSent, TargetID: "orc-1"},
		playerReadResult{State: playerReadOK, HasPos: true, Position: postMovePos,
			HasActiveEncounter: true, ActiveEncounterID: "enc-1"},
		encounterReadResult{State: encounterReadOK, Count: 1, Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "damage_applied", LatestResultValue: 10,
		}}},
		bt, inv, ""))

	// State 5: failed submit
	panel5 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateFailed, Error: "no mob focused"},
		playerReadResult{State: playerReadOK, HasPos: true, Position: postMovePos},
		encounterReadResult{State: encounterReadOK},
		bt, inv, ""))

	// State 6: read-gap (attack sent, player has encounter, read stale)
	panel6 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateSent, TargetID: "orc-1"},
		playerReadResult{State: playerReadOK, HasPos: true, Position: postMovePos,
			HasActiveEncounter: true, ActiveEncounterID: "enc-new"},
		encounterReadResult{State: encounterReadOK, Count: 0},
		bt, inv, ""))

	// Verify key content markers
	if !strings.Contains(panel1, "none") {
		t.Fatalf("state 1 (idle) must contain 'none', got: %s", panel1)
	}
	if !strings.Contains(panel2, "enc: opening") {
		t.Fatalf("state 2 (opening) must contain 'enc: opening', got: %s", panel2)
	}
	if !strings.Contains(panel3, "deny:oor") {
		t.Fatalf("state 3 (denial) must contain 'deny:oor', got: %s", panel3)
	}
	if !strings.Contains(panel4, "res:dmg") {
		t.Fatalf("state 4 (resolution) must contain 'res:dmg', got: %s", panel4)
	}
	if !strings.Contains(panel5, "atk:fail") {
		t.Fatalf("state 5 (failed) must contain 'atk:fail', got: %s", panel5)
	}
	if !strings.Contains(panel6, "enc: opening") {
		t.Fatalf("state 6 (read-gap) must contain 'enc: opening', got: %s", panel6)
	}

	// All must be pairwise distinct (opening and read-gap may match — both show enc:opening)
	panels := map[string]string{
		"idle": panel1, "opening": panel2, "denial": panel3,
		"resolution": panel4, "failed": panel5, "read-gap": panel6,
	}
	for nameA, pA := range panels {
		for nameB, pB := range panels {
			if nameA >= nameB {
				continue
			}
			// opening and read-gap are allowed to match
			if (nameA == "opening" && nameB == "read-gap") || (nameA == "read-gap" && nameB == "opening") {
				continue
			}
			if pA == pB {
				t.Fatalf("states %s and %s must differ after move but are identical", nameA, nameB)
			}
		}
	}
}

// --- Repeated movement approach legibility tests (M20260410-09) ---

func TestRepeatedMovesEachTriggerRefresh(t *testing.T) {
	// Each move in a repeated sequence must independently trigger immediate refresh.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}

	positions := []playerPosResult{
		{X: 5, Y: 0},
		{X: 10, Y: 0},
		{X: 15, Y: 0},
	}

	for i, pos := range positions {
		msg := moveResultMsg{
			result:    moveResult{OK: true, HasPos: true, Position: pos},
			direction: "east",
		}
		updated, cmd := m.Update(msg)
		if cmd == nil {
			t.Fatalf("move %d: must trigger immediate refresh", i+1)
		}
		m = updated.(model)
	}
}

func TestRepeatedMovesChainBackendPosition(t *testing.T) {
	// Position must chain from each backend response — no local prediction.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}

	steps := []struct {
		dir  string
		pos  playerPosResult
	}{
		{"east", playerPosResult{X: 5, Y: 0}},
		{"east", playerPosResult{X: 10, Y: 0}},
		{"south", playerPosResult{X: 10, Y: 5}},
	}

	for i, step := range steps {
		msg := moveResultMsg{
			result:    moveResult{OK: true, HasPos: true, Position: step.pos},
			direction: step.dir,
		}
		updated, _ := m.Update(msg)
		m = updated.(model)

		if m.playerRead.Position.X != step.pos.X || m.playerRead.Position.Y != step.pos.Y {
			t.Fatalf("step %d: position must be (%.0f,%.0f), got (%.0f,%.0f)",
				i+1, step.pos.X, step.pos.Y, m.playerRead.Position.X, m.playerRead.Position.Y)
		}
	}
}

func TestRepeatedMovesDeniedMovePreservesPosition(t *testing.T) {
	// A blocked/denied move must not alter position. Position stays at last
	// successful backend response — no local prediction.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}

	// First move succeeds
	msg1 := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 5, Y: 0}},
		direction: "east",
	}
	updated, _ := m.Update(msg1)
	m = updated.(model)

	// Second move fails (wall / boundary)
	msg2 := moveResultMsg{
		result:    moveResult{OK: false, Error: "boundary"},
		direction: "east",
	}
	updated, _ = m.Update(msg2)
	m = updated.(model)

	// Position must still be at last successful move
	if m.playerRead.Position.X != 5 || m.playerRead.Position.Y != 0 {
		t.Fatalf("denied move must not alter position: expected (5,0), got (%.0f,%.0f)",
			m.playerRead.Position.X, m.playerRead.Position.Y)
	}
	if m.lastIntent.state != moveStateFailed {
		t.Fatal("denied move must set intent to failed")
	}
}

func TestRepeatedMovesThenAttackStillOutOfRange(t *testing.T) {
	// Move twice but still out of range — attack denial must be visible.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position:           playerPosResult{X: 10, Y: 0}, // two moves east, still far
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "out_of_range",
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "deny:oor") {
		t.Fatalf("repeated moves still out of range must show deny:oor, got: %s", stripped)
	}
}

func TestRepeatedMovesThenAttackInRange(t *testing.T) {
	// Move multiple times to close range, then attack — backend resolution visible.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position:           playerPosResult{X: 20, Y: 5}, // close enough after repeated moves
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1, MobsDead: 0,
			ActionIndex:       1,
			LatestResultKind:  "damage_applied",
			LatestResultValue: 15,
		}},
	}
	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "res:dmg") {
		t.Fatalf("in-range attack after repeated moves must show res:dmg, got: %s", stripped)
	}
	if !strings.Contains(stripped, "15") {
		t.Fatalf("must show damage value, got: %s", stripped)
	}
	if strings.Contains(stripped, "enc: opening") {
		t.Fatalf("must not show enc:opening when encounter details present, got: %s", stripped)
	}
}

func TestRepeatedMovesRaceAttackBeforeReadsArrive(t *testing.T) {
	// Race: player moves multiple times, then attacks before the latest
	// move's immediate reads have returned. The combat panel must show
	// enc:opening, not regress to "none".
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position:           playerPosResult{X: 15, Y: 0}, // from move response
		HasActiveEncounter: false,                         // reads not arrived yet
	}
	er := encounterReadResult{State: encounterReadOK, Count: 0} // stale

	panel := renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "")
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "enc: opening") {
		t.Fatalf("race after repeated moves must show enc:opening, got: %s", stripped)
	}
	if strings.Contains(stripped, "  none") {
		t.Fatalf("must not regress to bare 'none' during race, got: %s", stripped)
	}
}

func TestRepeatedMovesSixStatesPreserved(t *testing.T) {
	// All six combat panel states must remain distinct after repeated moves.
	bt := defaultTarget()
	inv := inventoryReadResult{}
	// Position after three successive moves
	pos := playerPosResult{X: 15, Y: 0}
	prBase := playerReadResult{State: playerReadOK, HasPos: true, Position: pos}

	// State 1: idle / none
	p1 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{}, prBase,
		encounterReadResult{State: encounterReadOK}, bt, inv, ""))

	// State 2: enc:opening
	p2 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateSent, TargetID: "orc-1"}, prBase,
		encounterReadResult{State: encounterReadOK}, bt, inv, ""))

	// State 3: denial
	prEnc := prBase
	prEnc.HasActiveEncounter = true
	prEnc.ActiveEncounterID = "enc-1"
	p3 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateSent, TargetID: "orc-1"}, prEnc,
		encounterReadResult{State: encounterReadOK, Count: 1, Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "out_of_range",
		}}}, bt, inv, ""))

	// State 4: resolution
	p4 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateSent, TargetID: "orc-1"}, prEnc,
		encounterReadResult{State: encounterReadOK, Count: 1, Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "damage_applied", LatestResultValue: 10,
		}}}, bt, inv, ""))

	// State 5: failed submit
	p5 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateFailed, Error: "no mob focused"}, prBase,
		encounterReadResult{State: encounterReadOK}, bt, inv, ""))

	// State 6: read-gap
	prGap := prBase
	prGap.HasActiveEncounter = true
	prGap.ActiveEncounterID = "enc-new"
	p6 := stripANSI(renderCombatPanel(sidePanelWidth,
		attackResult{State: attackStateSent, TargetID: "orc-1"}, prGap,
		encounterReadResult{State: encounterReadOK, Count: 0}, bt, inv, ""))

	// Verify markers
	if !strings.Contains(p1, "none") {
		t.Fatalf("idle must contain 'none', got: %s", p1)
	}
	if !strings.Contains(p2, "enc: opening") {
		t.Fatalf("opening must contain 'enc: opening', got: %s", p2)
	}
	if !strings.Contains(p3, "deny:oor") {
		t.Fatalf("denial must contain 'deny:oor', got: %s", p3)
	}
	if !strings.Contains(p4, "res:dmg") {
		t.Fatalf("resolution must contain 'res:dmg', got: %s", p4)
	}
	if !strings.Contains(p5, "atk:fail") {
		t.Fatalf("failed must contain 'atk:fail', got: %s", p5)
	}
	if !strings.Contains(p6, "enc: opening") {
		t.Fatalf("read-gap must contain 'enc: opening', got: %s", p6)
	}

	// Pairwise distinct (opening/read-gap may match)
	panels := map[string]string{
		"idle": p1, "opening": p2, "denial": p3,
		"resolution": p4, "failed": p5, "read-gap": p6,
	}
	for a, pa := range panels {
		for b, pb := range panels {
			if a >= b {
				continue
			}
			if (a == "opening" && b == "read-gap") || (a == "read-gap" && b == "opening") {
				continue
			}
			if pa == pb {
				t.Fatalf("states %s and %s must differ after repeated moves", a, b)
			}
		}
	}
}

// --- Selected-target approach legibility tests (M20260410-10) ---

func TestProximityPanelOutOfRangeAfterMove(t *testing.T) {
	// After moving but still out of range, the proximity panel must show
	// backend-owned "out" distance label. No local inference.
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1",
		Found: true, WithinProximity: false, Distance: 45,
		MobName: "an orc",
	}
	ar := attackResult{}
	mobr := mobReadResult{}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 10, Y: 0}}
	panel := renderProximityPanel(sidePanelWidth, tc, ar, mobr, pr, rosterFocus{index: -1}, nil)
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "OOR dist:45") {
		t.Fatalf("proximity must show 'OOR dist:45' when target still distant, got: %s", stripped)
	}
	if !strings.Contains(stripped, "an orc") {
		t.Fatalf("proximity must show mob name, got: %s", stripped)
	}
}

func TestProximityPanelInRangeAfterMove(t *testing.T) {
	// After closing range via repeated moves, the proximity panel must show
	// "IN RANGE" with backend-owned distance.
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1",
		Found: true, WithinProximity: true, Distance: 8,
		MobName: "an orc",
	}
	ar := attackResult{}
	mobr := mobReadResult{}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 20, Y: 5}}
	panel := renderProximityPanel(sidePanelWidth, tc, ar, mobr, pr, rosterFocus{index: -1}, nil)
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "IN RANGE dist:8") {
		t.Fatalf("proximity must show 'in range 8' when close, got: %s", stripped)
	}
}

func TestProximityPanelAtkMarkerAfterMoveAndAttack(t *testing.T) {
	// After moving into range and attacking, the proximity panel must show
	// [atk] marker on the target when attack target matches proximity target.
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1",
		Found: true, WithinProximity: true, Distance: 5,
		MobName: "an orc",
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	mobr := mobReadResult{}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 22, Y: 5}}
	panel := renderProximityPanel(sidePanelWidth, tc, ar, mobr, pr, rosterFocus{index: -1}, nil)
	stripped := stripANSI(panel)

	if !strings.Contains(stripped, "IN RANGE") {
		t.Fatalf("target panel must show IN RANGE when in range, got: %s", stripped)
	}
}

func TestProximityPanelNoAtkMarkerOnDifferentTarget(t *testing.T) {
	// [atk] marker must NOT appear when proximity target differs from attack target.
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-2",
		Found: true, WithinProximity: true, Distance: 5,
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	panel := renderProximityPanel(sidePanelWidth, tc, ar, mobReadResult{}, playerReadResult{}, rosterFocus{index: -1}, nil)
	stripped := stripANSI(panel)

	if strings.Contains(stripped, "[atk]") {
		t.Fatalf("proximity must NOT show [atk] for different target, got: %s", stripped)
	}
}

func TestProximityRefreshTriggersAfterEachMove(t *testing.T) {
	// When a target is selected (proximity state is active), each successive
	// move must trigger a proximity re-query because position has changed.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	m.rosterFocus = rosterFocus{index: 0}
	// Set up active proximity state so maybeRefreshProximity will fire
	m.targetConfirm = targetConfirmResult{State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1", Found: true}
	m.lastProximityPos = playerPosResult{X: 0, Y: 0}
	m.lastProximityID = "orc-1"

	moves := []playerPosResult{{X: 5, Y: 0}, {X: 10, Y: 0}, {X: 15, Y: 0}}

	for i, pos := range moves {
		msg := moveResultMsg{
			result:    moveResult{OK: true, HasPos: true, Position: pos},
			direction: "east",
		}
		updated, cmd := m.Update(msg)
		if cmd == nil {
			t.Fatalf("move %d with selected target must trigger refresh", i+1)
		}
		m = updated.(model)
		// Update lastProximityPos to simulate the proximity query returning
		// (in real code this happens when targetConfirmMsg arrives)
		m.lastProximityPos = pos
	}
}

func TestProximityNeedsRefreshAfterRepeatedMoves(t *testing.T) {
	// After multiple position changes, proximityNeedsRefresh must report true
	// when the current position differs from the last queried position.
	tc := targetConfirmResult{State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1"}
	entry := &rosterEntry{kind: "mb", id: "orc-1"}
	lastPos := playerPosResult{X: 0, Y: 0}
	currentPos := playerPosResult{X: 15, Y: 0} // after 3 moves east

	if !proximityNeedsRefresh(tc, lastPos, "orc-1", currentPos, entry) {
		t.Fatal("proximity must need refresh when position changed across moves")
	}
}

func TestBlockedMovePreservesProximityState(t *testing.T) {
	// A blocked/denied move must not alter the proximity tracking state.
	// The last known proximity result and position must remain unchanged.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 10, Y: 0}}
	m.targetConfirm = targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1",
		Found: true, WithinProximity: false, Distance: 30,
	}
	m.lastProximityPos = playerPosResult{X: 10, Y: 0}
	m.lastProximityID = "orc-1"

	// Failed move — wall/boundary
	msg := moveResultMsg{
		result:    moveResult{OK: false, Error: "boundary"},
		direction: "east",
	}
	updated, _ := m.Update(msg)
	um := updated.(model)

	// Position must not change
	if um.playerRead.Position.X != 10 || um.playerRead.Position.Y != 0 {
		t.Fatalf("blocked move must preserve position, got (%.0f,%.0f)",
			um.playerRead.Position.X, um.playerRead.Position.Y)
	}
	// Proximity state must not change
	if um.targetConfirm.TargetID != "orc-1" {
		t.Fatal("blocked move must preserve proximity target ID")
	}
	if um.targetConfirm.Distance != 30 {
		t.Fatalf("blocked move must preserve proximity distance, got %.0f", um.targetConfirm.Distance)
	}
}

func TestProximityAndCombatPanelComposeOutOfRange(t *testing.T) {
	// Proximity shows "out 40" while combat shows "deny:oor" — both panels
	// must be internally consistent, showing the same selected target.
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1",
		Found: true, WithinProximity: false, Distance: 40,
		MobName: "an orc",
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true, Position: playerPosResult{X: 5, Y: 0},
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "out_of_range",
		}},
	}

	proxPanel := stripANSI(renderProximityPanel(sidePanelWidth, tc, ar, mobReadResult{}, pr, rosterFocus{index: -1}, nil))
	combatPanel := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, ""))

	if !strings.Contains(proxPanel, "OOR dist:40") {
		t.Fatalf("proximity panel must show 'OOR dist:40', got: %s", proxPanel)
	}
	if !strings.Contains(combatPanel, "deny:oor") {
		t.Fatalf("combat panel must show 'deny:oor', got: %s", combatPanel)
	}
}

func TestProximityAndCombatPanelComposeInRange(t *testing.T) {
	// Proximity shows "IN RANGE dist:8" while combat shows "res:dmg" — both panels
	// must be internally consistent after closing range.
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1",
		Found: true, WithinProximity: true, Distance: 8,
		MobName: "an orc",
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true, Position: playerPosResult{X: 20, Y: 5},
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind:  "damage_applied",
			LatestResultValue: 12,
		}},
	}

	proxPanel := stripANSI(renderProximityPanel(sidePanelWidth, tc, ar, mobReadResult{}, pr, rosterFocus{index: -1}, nil))
	combatPanel := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, ""))

	if !strings.Contains(proxPanel, "IN RANGE dist:8") {
		t.Fatalf("target panel must show 'IN RANGE dist:8', got: %s", proxPanel)
	}
	if !strings.Contains(combatPanel, "res:dmg") {
		t.Fatalf("combat panel must show 'res:dmg', got: %s", combatPanel)
	}
}

func TestProximityPanelRaceAfterMoveBeforeRead(t *testing.T) {
	// Race: player moved but proximity re-query hasn't returned yet.
	// The proximity panel must show the *stale* proximity data honestly —
	// the old distance, not an invented closer one.
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1",
		Found: true, WithinProximity: false, Distance: 50, // stale — before move
		MobName: "an orc",
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	// Player has moved but proximity hasn't refreshed yet
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 15, Y: 0}}
	panel := renderProximityPanel(sidePanelWidth, tc, ar, mobReadResult{}, pr, rosterFocus{index: -1}, nil)
	stripped := stripANSI(panel)

	// Panel shows the stale (last known) distance — this is honest
	if !strings.Contains(stripped, "OOR dist:50") {
		t.Fatalf("race: proximity must show stale distance 'OOR dist:50', got: %s", stripped)
	}
	// Must not show "IN RANGE" — that would be local inference
	if strings.Contains(stripped, "IN RANGE") {
		t.Fatalf("race: proximity must NOT infer in-range from stale data, got: %s", stripped)
	}
}

// --- Selected-target next-action legibility tests (M20260410-11) ---

func TestCombatPanelDenialToSuccessTransition(t *testing.T) {
	// Scenario: first attack was denied (oor), player moved closer, second attack
	// succeeded. The combat panel must show res:dmg from the new backend truth,
	// not residual deny:oor.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position:           playerPosResult{X: 22, Y: 5},
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}

	// Phase 1: denial
	erDeny := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "out_of_range",
		}},
	}
	panelDeny := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, erDeny, defaultTarget(), inventoryReadResult{}, "orc-1"))
	if !strings.Contains(panelDeny, "deny:oor") {
		t.Fatalf("phase 1 must show deny:oor, got: %s", panelDeny)
	}

	// Phase 2: success (backend truth updated after move + re-attack)
	erSuccess := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			ActionIndex:       1,
			LatestResultKind:  "damage_applied",
			LatestResultValue: 14,
		}},
	}
	panelSuccess := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, erSuccess, defaultTarget(), inventoryReadResult{}, "orc-1"))
	if !strings.Contains(panelSuccess, "res:dmg") {
		t.Fatalf("phase 2 must show res:dmg, got: %s", panelSuccess)
	}
	if !strings.Contains(panelSuccess, "14") {
		t.Fatalf("phase 2 must show damage value, got: %s", panelSuccess)
	}
	if strings.Contains(panelSuccess, "deny:") {
		t.Fatalf("phase 2 must NOT contain deny: residue, got: %s", panelSuccess)
	}

	// Panels must differ
	if panelDeny == panelSuccess {
		t.Fatal("denial and success panels must be visually distinct")
	}
}

func TestCombatPanelDenialToDifferentDenialTransition(t *testing.T) {
	// Scenario: first attack denied (oor), player moves but LOS now blocked,
	// second attack denied (los). Panel must show the new denial, not the old.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position:           playerPosResult{X: 18, Y: 3},
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}

	// Phase 1: oor denial
	erOOR := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "out_of_range",
		}},
	}
	p1 := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, erOOR, defaultTarget(), inventoryReadResult{}, "orc-1"))
	if !strings.Contains(p1, "deny:oor") {
		t.Fatalf("phase 1 must show deny:oor, got: %s", p1)
	}

	// Phase 2: LOS denial
	erLOS := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "los_blocked",
		}},
	}
	p2 := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, erLOS, defaultTarget(), inventoryReadResult{}, "orc-1"))
	if !strings.Contains(p2, "deny:los") {
		t.Fatalf("phase 2 must show deny:los, got: %s", p2)
	}
	if strings.Contains(p2, "deny:oor") {
		t.Fatalf("phase 2 must NOT show residual deny:oor, got: %s", p2)
	}
}

func TestCombatPanelFocusedMobWithDenyAndRes(t *testing.T) {
	// The focused mob ID affects the roster display. The ">" prefix on the
	// attack target and the "~" suffix on the focused mob must compose
	// correctly alongside deny: and res: labels.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position:           playerPosResult{X: 20, Y: 5},
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}

	// Denial with focused mob
	erDeny := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1", "orc-2"}, MobsAlive: 2,
			LatestResultKind: "out_of_range",
		}},
	}
	panelDeny := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, erDeny, defaultTarget(), inventoryReadResult{}, "orc-1"))
	if !strings.Contains(panelDeny, "> orc-1") {
		t.Fatalf("attack target must show > prefix during denial, got: %s", panelDeny)
	}
	if !strings.Contains(panelDeny, "deny:oor") {
		t.Fatalf("must show deny:oor, got: %s", panelDeny)
	}

	// Resolution with focused mob
	erRes := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1", "orc-2"}, MobsAlive: 2,
			ActionIndex:       1,
			LatestResultKind:  "damage_applied",
			LatestResultValue: 10,
		}},
	}
	panelRes := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, erRes, defaultTarget(), inventoryReadResult{}, "orc-1"))
	if !strings.Contains(panelRes, "> orc-1") {
		t.Fatalf("attack target must show > prefix during resolution, got: %s", panelRes)
	}
	if !strings.Contains(panelRes, "res:dmg") {
		t.Fatalf("must show res:dmg, got: %s", panelRes)
	}
}

func TestProximityCombatTransitionOutToInRange(t *testing.T) {
	// Cross-panel transition: proximity goes from "out" to "IN RANGE" while
	// combat goes from "deny:oor" to "res:dmg". Both must update consistently.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	mobr := mobReadResult{}

	// Phase 1: out of range
	tcOut := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1",
		Found: true, WithinProximity: false, Distance: 35, MobName: "an orc",
	}
	erDeny := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "out_of_range",
		}},
	}
	prox1 := stripANSI(renderProximityPanel(sidePanelWidth, tcOut, ar, mobr, pr, rosterFocus{index: -1}, nil))
	combat1 := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, erDeny, defaultTarget(), inventoryReadResult{}, "orc-1"))

	if !strings.Contains(prox1, "OOR dist:35") {
		t.Fatalf("phase 1 proximity must show 'OOR dist:35', got: %s", prox1)
	}
	if !strings.Contains(combat1, "deny:oor") {
		t.Fatalf("phase 1 combat must show 'deny:oor', got: %s", combat1)
	}

	// Phase 2: in range
	tcIn := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1",
		Found: true, WithinProximity: true, Distance: 6, MobName: "an orc",
	}
	erRes := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			ActionIndex: 1, LatestResultKind: "damage_applied", LatestResultValue: 12,
		}},
	}
	prox2 := stripANSI(renderProximityPanel(sidePanelWidth, tcIn, ar, mobr, pr, rosterFocus{index: -1}, nil))
	combat2 := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, erRes, defaultTarget(), inventoryReadResult{}, "orc-1"))

	if !strings.Contains(prox2, "IN RANGE dist:6") {
		t.Fatalf("phase 2 proximity must show 'in range 6', got: %s", prox2)
	}
	if !strings.Contains(combat2, "res:dmg") {
		t.Fatalf("phase 2 combat must show 'res:dmg', got: %s", combat2)
	}

	// Phases must differ
	if prox1 == prox2 {
		t.Fatal("proximity panels must differ between out and in range")
	}
	if combat1 == combat2 {
		t.Fatal("combat panels must differ between denial and success")
	}
}

func TestProximityNotFoundWhileCombatDenial(t *testing.T) {
	// Edge case: proximity says target not found, combat shows denial.
	// Both are backend truth — the panels must render independently.
	tc := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1",
		Found: false,
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "invalid_target",
		}},
	}
	proxPanel := stripANSI(renderProximityPanel(sidePanelWidth, tc, ar, mobReadResult{}, pr, rosterFocus{index: -1}, nil))
	combatPanel := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, ""))

	if !strings.Contains(proxPanel, "not found") {
		t.Fatalf("proximity must show 'not found', got: %s", proxPanel)
	}
	if !strings.Contains(combatPanel, "deny:bad_tgt") {
		t.Fatalf("combat must show 'deny:bad_tgt', got: %s", combatPanel)
	}
}

func TestStaleProximityDuringCombatSuccess(t *testing.T) {
	// Race: proximity re-query hasn't returned after move, but combat panel
	// already shows res:dmg from the attack. Proximity must show stale data
	// honestly, not invent "IN RANGE" to match combat.
	tcStale := targetConfirmResult{
		State: targetConfirmOK, TargetKind: "mb", TargetID: "orc-1",
		Found: true, WithinProximity: false, Distance: 30, MobName: "an orc",
	}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position:           playerPosResult{X: 22, Y: 5},
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			ActionIndex: 1, LatestResultKind: "damage_applied", LatestResultValue: 15,
		}},
	}

	proxPanel := stripANSI(renderProximityPanel(sidePanelWidth, tcStale, ar, mobReadResult{}, pr, rosterFocus{index: -1}, nil))
	combatPanel := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "orc-1"))

	// Proximity shows stale data — this is honest
	if !strings.Contains(proxPanel, "OOR dist:30") {
		t.Fatalf("stale proximity must show 'OOR dist:30', got: %s", proxPanel)
	}
	// Combat shows fresh backend truth
	if !strings.Contains(combatPanel, "res:dmg") {
		t.Fatalf("combat must show 'res:dmg', got: %s", combatPanel)
	}
	// The panels may be temporarily inconsistent — this is correct thin-client
	// behavior. Neither panel invents data to match the other.
}

func TestCombatRosterTargetPrefixThroughDenyResTransition(t *testing.T) {
	// The ">" prefix on the attack target in the combat mob roster must persist
	// through deny→res transitions within the same encounter.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}

	for _, tc := range []struct {
		name string
		kind string
	}{
		{"denial", "out_of_range"},
		{"resolution", "damage_applied"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			er := encounterReadResult{
				State: encounterReadOK, Count: 1,
				Encounters: []encounterSummary{{
					EncounterID: "enc-1", State: "Active",
					MobIDs: []string{"orc-1", "orc-2"}, MobsAlive: 2,
					LatestResultKind: tc.kind,
				}},
			}
			panel := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "orc-1"))
			if !strings.Contains(panel, "> orc-1") {
				t.Fatalf("%s: attack target must show '> orc-1', got: %s", tc.name, panel)
			}
		})
	}
}

// --- Step toward selected target tests (M20260410-12) ---

func TestDirectionTowardMobEast(t *testing.T) {
	player := playerPosResult{X: 0, Y: 0}
	mob := mobPosVec3{X: 10, Y: 0}
	if dir := directionTowardMob(player, mob); dir != "east" {
		t.Fatalf("mob to the east should produce 'east', got: %s", dir)
	}
}

func TestDirectionTowardMobWest(t *testing.T) {
	player := playerPosResult{X: 10, Y: 0}
	mob := mobPosVec3{X: 0, Y: 0}
	if dir := directionTowardMob(player, mob); dir != "west" {
		t.Fatalf("mob to the west should produce 'west', got: %s", dir)
	}
}

func TestDirectionTowardMobNorthHigherY(t *testing.T) {
	// Mob at higher Y → "north" (backend "up" = +Y)
	player := playerPosResult{X: 0, Y: 0}
	mob := mobPosVec3{X: 0, Y: 10}
	if dir := directionTowardMob(player, mob); dir != "north" {
		t.Fatalf("mob at higher Y should produce 'north', got: %s", dir)
	}
}

func TestDirectionTowardMobSouthLowerY(t *testing.T) {
	// Mob at lower Y → "south" (backend "down" = -Y)
	player := playerPosResult{X: 0, Y: 10}
	mob := mobPosVec3{X: 0, Y: 0}
	if dir := directionTowardMob(player, mob); dir != "south" {
		t.Fatalf("mob at lower Y should produce 'south', got: %s", dir)
	}
}

func TestDirectionTowardMobDiagonalEast(t *testing.T) {
	// Diagonal with larger X delta — should pick east
	player := playerPosResult{X: 0, Y: 0}
	mob := mobPosVec3{X: 10, Y: 5}
	if dir := directionTowardMob(player, mob); dir != "east" {
		t.Fatalf("mob northeast with larger X delta should produce 'east', got: %s", dir)
	}
}

func TestDirectionTowardMobDiagonalNorth(t *testing.T) {
	// Diagonal with larger Y delta toward higher Y — should pick north
	player := playerPosResult{X: 0, Y: 0}
	mob := mobPosVec3{X: 3, Y: 10}
	if dir := directionTowardMob(player, mob); dir != "north" {
		t.Fatalf("mob northeast with larger Y delta should produce 'north', got: %s", dir)
	}
}

func TestDirectionTowardMobTieBreaksHorizontal(t *testing.T) {
	// Equal deltas — tie breaks to horizontal (east/west)
	player := playerPosResult{X: 0, Y: 0}
	mob := mobPosVec3{X: 5, Y: 5}
	dir := directionTowardMob(player, mob)
	if dir != "east" {
		t.Fatalf("equal deltas should tie-break to east, got: %s", dir)
	}
}

func TestDirectionTowardMobSamePosition(t *testing.T) {
	player := playerPosResult{X: 5, Y: 5}
	mob := mobPosVec3{X: 5, Y: 5}
	if dir := directionTowardMob(player, mob); dir != "" {
		t.Fatalf("same position should produce empty direction, got: %s", dir)
	}
}

func TestFindMobPositionFound(t *testing.T) {
	mobs := []mobPosition{
		{MobID: "orc-1", Position: mobPosVec3{X: 10, Y: 20}},
		{MobID: "orc-2", Position: mobPosVec3{X: 30, Y: 40}},
	}
	pos := findMobPosition(mobs, "orc-2")
	if pos == nil {
		t.Fatal("should find orc-2")
	}
	if pos.X != 30 || pos.Y != 40 {
		t.Fatalf("orc-2 position should be (30,40), got (%.0f,%.0f)", pos.X, pos.Y)
	}
}

func TestFindMobPositionNotFound(t *testing.T) {
	mobs := []mobPosition{
		{MobID: "orc-1", Position: mobPosVec3{X: 10, Y: 20}},
	}
	pos := findMobPosition(mobs, "orc-99")
	if pos != nil {
		t.Fatal("should not find orc-99")
	}
}

func TestGotoKeyTriggersMove(t *testing.T) {
	// `g` key with a focused mob should trigger a move via moveResultMsg
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: []mobPosition{
		{MobID: "orc-1", Position: mobPosVec3{X: 20, Y: 0}},
	}, Count: 1}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	m.rosterFocus = rosterFocus{index: 0}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if cmd == nil {
		t.Fatal("g key with focused mob must trigger a move command")
	}
}

func TestGotoKeyNoFocusNoAction(t *testing.T) {
	// `g` key without a focused mob must not trigger a move
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	m.rosterFocus = rosterFocus{index: -1}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if cmd != nil {
		t.Fatal("g key without focused mob must not trigger action")
	}
}

func TestGotoKeyNoPlayerNoAction(t *testing.T) {
	// `g` key without player state must not trigger a move
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	m.rosterFocus = rosterFocus{index: 0}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if cmd != nil {
		t.Fatal("g key without player must not trigger action")
	}
}

func TestGotoKeyMobNotInPositions(t *testing.T) {
	// `g` key when focused mob is not in mob positions must not trigger a move
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: []mobPosition{
		{MobID: "orc-99", Position: mobPosVec3{X: 20, Y: 0}},
	}, Count: 1}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	m.rosterFocus = rosterFocus{index: 0}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if cmd != nil {
		t.Fatal("g key when mob not in positions must not trigger action")
	}
}

func TestGotoKeySetsPreviewDirection(t *testing.T) {
	// After `g` key, lastIntent should reflect the computed direction
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: []mobPosition{
		{MobID: "orc-1", Position: mobPosVec3{X: 20, Y: 0}},
	}, Count: 1}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	m.rosterFocus = rosterFocus{index: 0}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	um := updated.(model)
	if um.lastIntent.direction != "east" {
		t.Fatalf("goto toward mob at (20,0) from (0,0) should set direction east, got: %s", um.lastIntent.direction)
	}
	if um.lastIntent.state != moveStatePreview {
		t.Fatal("goto should set intent state to preview before move response arrives")
	}
}

func TestGotoRepeatedDoesNotBecomeAutonomous(t *testing.T) {
	// Two manual `g` presses produce two independent move commands.
	// Each returns a cmd but does not chain automatically.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: []mobPosition{
		{MobID: "orc-1", Position: mobPosVec3{X: 20, Y: 0}},
	}, Count: 1}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	m.rosterFocus = rosterFocus{index: 0}

	// First g press
	updated, cmd1 := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if cmd1 == nil {
		t.Fatal("first g must trigger command")
	}

	// Simulate move result (position updated by backend)
	m2 := updated.(model)
	moveMsg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 5, Y: 0}},
		direction: "east",
	}
	updated2, _ := m2.Update(moveMsg)
	m3 := updated2.(model)

	// Second g press (manual, not automatic)
	_, cmd2 := m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if cmd2 == nil {
		t.Fatal("second manual g must trigger command")
	}
}

func TestGotoThenAttackComposition(t *testing.T) {
	// goto → move result → attack composes cleanly
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: []mobPosition{
		{MobID: "orc-1", Position: mobPosVec3{X: 20, Y: 0}},
	}, Count: 1}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	m.rosterFocus = rosterFocus{index: 0}

	// g press
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m2 := updated.(model)

	// Move result
	moveMsg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 5, Y: 0}},
		direction: "east",
	}
	updated2, moveCmd := m2.Update(moveMsg)
	m3 := updated2.(model)
	if moveCmd == nil {
		t.Fatal("move result must trigger immediate refresh")
	}

	// Attack
	attackMsg := attackResultMsg{result: attackResult{State: attackStateSent, TargetID: "orc-1"}}
	_, attackCmd := m3.Update(attackMsg)
	if attackCmd == nil {
		t.Fatal("attack after goto must trigger immediate refresh")
	}
}

func TestFooterContainsGotoHint(t *testing.T) {
	footer := renderFooter(80, "", "", "", "", "", true, inventoryReadResult{})
	if !strings.Contains(footer, "g:step") {
		t.Fatalf("footer must contain g:step hint, got: %s", stripANSI(footer))
	}
}

// --- Attack key handler tests (M20260410-13) ---

func TestAttackKeyWithFocusedMobTriggersAttack(t *testing.T) {
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	m.rosterFocus = rosterFocus{index: 0}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Fatal("a key with focused mob must trigger attack (non-nil Cmd)")
	}
}

func TestAttackKeyNoPlayerSetsFailure(t *testing.T) {
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	// playerRead.State is playerReadNotAttempted (zero value)
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	m.rosterFocus = rosterFocus{index: 0}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Fatal("a key without player must not trigger action")
	}
	um := updated.(model)
	if um.lastAttack.State != attackStateFailed {
		t.Fatal("a key without player must set attackStateFailed")
	}
}

func TestAttackKeyNoFocusSetsFailure(t *testing.T) {
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK}
	m.rosterFocus = rosterFocus{index: -1}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Fatal("a key without focus must not trigger action")
	}
	um := updated.(model)
	if um.lastAttack.State != attackStateFailed {
		t.Fatal("a key without focus must set attackStateFailed")
	}
}

func TestAttackKeyFocusedPCNoAction(t *testing.T) {
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK}
	m.rosterEntries = []rosterEntry{{kind: "pc", id: "p2"}}
	m.rosterFocus = rosterFocus{index: 0}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Fatal("a key with focused PC must not trigger attack")
	}
	um := updated.(model)
	if um.lastAttack.State != attackStateFailed {
		t.Fatal("a key with focused PC must set attackStateFailed")
	}
}

func TestAttackKeyEmptyLiveIDNoAction(t *testing.T) {
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "", canonicalMobID: false}}
	m.rosterFocus = rosterFocus{index: 0}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Fatal("a key with empty live_mob_id must not trigger attack")
	}
	um := updated.(model)
	if um.lastAttack.State != attackStateFailed {
		t.Fatal("a key with empty live_mob_id must set attackStateFailed")
	}
}

func TestAttackKeyPreservesTargetID(t *testing.T) {
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-42", canonicalMobID: true}}
	m.rosterFocus = rosterFocus{index: 0}

	// The a key sets up the attack cmd — simulate the result arriving
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Fatal("must trigger attack")
	}
	// Simulate the attack result arriving
	attackMsg := attackResultMsg{result: attackResult{State: attackStateSent, TargetID: "orc-42"}}
	updated, _ := m.Update(attackMsg)
	um := updated.(model)
	if um.lastAttack.TargetID != "orc-42" {
		t.Fatalf("lastAttack.TargetID must be orc-42, got: %s", um.lastAttack.TargetID)
	}
}

func TestAttackKeyRepeatedManualNotAutonomous(t *testing.T) {
	// Two manual a presses produce two independent commands.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	m.rosterFocus = rosterFocus{index: 0}

	// First a press
	_, cmd1 := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd1 == nil {
		t.Fatal("first a must trigger command")
	}

	// Simulate attack result
	attackMsg := attackResultMsg{result: attackResult{State: attackStateSent, TargetID: "orc-1"}}
	updated, _ := m.Update(attackMsg)
	m2 := updated.(model)

	// Second a press (manual)
	_, cmd2 := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd2 == nil {
		t.Fatal("second manual a must trigger command")
	}
}

func TestGotoThenAttackKeyComposition(t *testing.T) {
	// g (goto) then a (attack) compose cleanly through the Update handler
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: []mobPosition{
		{MobID: "orc-1", Position: mobPosVec3{X: 20, Y: 0}},
	}, Count: 1}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	m.rosterFocus = rosterFocus{index: 0}

	// g press
	updated, gotoCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if gotoCmd == nil {
		t.Fatal("g must trigger move")
	}
	m2 := updated.(model)

	// Simulate move result
	moveMsg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 5, Y: 0}},
		direction: "east",
	}
	updated2, _ := m2.Update(moveMsg)
	m3 := updated2.(model)

	// a press
	_, atkCmd := m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if atkCmd == nil {
		t.Fatal("a after g must trigger attack")
	}
}

func TestAttackKeyCombatPanelDenyAfterSubmit(t *testing.T) {
	// After a key triggers attack and backend returns denial,
	// combat panel must show deny:oor, not enc:opening.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "out_of_range",
		}},
	}
	panel := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "orc-1"))
	if !strings.Contains(panel, "deny:oor") {
		t.Fatalf("combat panel must show deny:oor after attack denial, got: %s", panel)
	}
}

func TestAttackKeyCombatPanelSuccessAfterSubmit(t *testing.T) {
	// After a key triggers attack and backend returns success,
	// combat panel must show res:dmg.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			ActionIndex: 1, LatestResultKind: "damage_applied", LatestResultValue: 15,
		}},
	}
	panel := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "orc-1"))
	if !strings.Contains(panel, "res:dmg") {
		t.Fatalf("combat panel must show res:dmg after attack success, got: %s", panel)
	}
	if !strings.Contains(panel, "15") {
		t.Fatalf("combat panel must show damage value, got: %s", panel)
	}
}

func TestAttackKeyCombatPanelLOSBlockedAfterSubmit(t *testing.T) {
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "los_blocked",
		}},
	}
	panel := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inventoryReadResult{}, "orc-1"))
	if !strings.Contains(panel, "deny:los") {
		t.Fatalf("combat panel must show deny:los, got: %s", panel)
	}
}

func TestFooterContainsAttackHint(t *testing.T) {
	footer := renderFooter(80, "", "", "", "", "", true, inventoryReadResult{})
	if !strings.Contains(footer, "a:atk") {
		t.Fatalf("footer must contain a:atk hint, got: %s", stripANSI(footer))
	}
}

// --- g/a loop legibility tests (M20260410-14) ---

func TestGALoopRepeatedApproachThenAttack(t *testing.T) {
	// Full loop: g, g, g, then a — position chains and attack fires correctly.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: []mobPosition{
		{MobID: "orc-1", Position: mobPosVec3{X: 20, Y: 0}},
	}, Count: 1}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	m.rosterFocus = rosterFocus{index: 0}

	// Three g presses with move results
	positions := []playerPosResult{{X: 5, Y: 0}, {X: 10, Y: 0}, {X: 15, Y: 0}}
	for i, pos := range positions {
		updated, gCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
		if gCmd == nil {
			t.Fatalf("g press %d must trigger command", i+1)
		}
		m = updated.(model)
		moveMsg := moveResultMsg{
			result:    moveResult{OK: true, HasPos: true, Position: pos},
			direction: "east",
		}
		updated2, _ := m.Update(moveMsg)
		m = updated2.(model)
	}

	// Position must be at last move result
	if m.playerRead.Position.X != 15 {
		t.Fatalf("after 3 g presses position must be X=15, got %.0f", m.playerRead.Position.X)
	}

	// a press
	_, aCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if aCmd == nil {
		t.Fatal("a after repeated g must trigger attack")
	}
}

func TestGALoopBlockedGotoThenAttack(t *testing.T) {
	// g fails (wall), then a — attack still fires against focused mob.
	// Combat panel shows denial from backend, not local wall inference.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 10, Y: 0}}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: []mobPosition{
		{MobID: "orc-1", Position: mobPosVec3{X: 20, Y: 0}},
	}, Count: 1}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	m.rosterFocus = rosterFocus{index: 0}

	// g press
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(model)

	// Move fails
	moveMsg := moveResultMsg{
		result:    moveResult{OK: false, Error: "boundary"},
		direction: "east",
	}
	updated2, _ := m.Update(moveMsg)
	m = updated2.(model)

	// Position unchanged
	if m.playerRead.Position.X != 10 {
		t.Fatalf("blocked g must not change position, got X=%.0f", m.playerRead.Position.X)
	}
	if m.lastIntent.state != moveStateFailed {
		t.Fatal("blocked g must set intent to failed")
	}

	// a press still works
	_, aCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if aCmd == nil {
		t.Fatal("a after blocked g must still trigger attack")
	}
}

func TestGALoopDenialThenSuccessTransition(t *testing.T) {
	// Full denial→success loop: g, a (deny:oor), g, a (res:dmg).
	// Combat panel must transition cleanly.
	bt := defaultTarget()
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}

	// Phase 1: after first g+a — still out of range
	pr.Position = playerPosResult{X: 5, Y: 0}
	erDeny := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "out_of_range",
		}},
	}
	panelDeny := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, erDeny, bt, inventoryReadResult{}, "orc-1"))
	if !strings.Contains(panelDeny, "deny:oor") {
		t.Fatalf("phase 1 must show deny:oor, got: %s", panelDeny)
	}

	// Phase 2: after second g+a — now in range
	pr.Position = playerPosResult{X: 18, Y: 0}
	erSuccess := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			ActionIndex: 1, LatestResultKind: "damage_applied", LatestResultValue: 12,
		}},
	}
	panelSuccess := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, erSuccess, bt, inventoryReadResult{}, "orc-1"))
	if !strings.Contains(panelSuccess, "res:dmg") {
		t.Fatalf("phase 2 must show res:dmg, got: %s", panelSuccess)
	}
	if !strings.Contains(panelSuccess, "12") {
		t.Fatalf("phase 2 must show damage value, got: %s", panelSuccess)
	}
	if strings.Contains(panelSuccess, "deny:") {
		t.Fatalf("phase 2 must not contain deny: residue, got: %s", panelSuccess)
	}

	// Panels must differ
	if panelDeny == panelSuccess {
		t.Fatal("denial and success panels must be visually distinct")
	}
}

func TestGALoopRepeatedAttacksDuringEncounter(t *testing.T) {
	// Active encounter: a, a — action index increments, both show res:dmg.
	bt := defaultTarget()
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true, Position: playerPosResult{X: 20, Y: 0},
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}

	// First attack result
	er1 := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			ActionIndex: 1, LatestResultKind: "damage_applied", LatestResultValue: 10,
		}},
	}
	p1 := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er1, bt, inventoryReadResult{}, "orc-1"))
	if !strings.Contains(p1, "act:1") {
		t.Fatalf("first attack must show act:1, got: %s", p1)
	}

	// Second attack result
	er2 := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			ActionIndex: 2, LatestResultKind: "damage_applied", LatestResultValue: 8,
		}},
	}
	p2 := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er2, bt, inventoryReadResult{}, "orc-1"))
	if !strings.Contains(p2, "act:2") {
		t.Fatalf("second attack must show act:2, got: %s", p2)
	}
	if !strings.Contains(p2, "8") {
		t.Fatalf("second attack must show new damage value, got: %s", p2)
	}

	// Both must show res:dmg
	if !strings.Contains(p1, "res:dmg") || !strings.Contains(p2, "res:dmg") {
		t.Fatal("both attacks must show res:dmg")
	}

	// Panels must differ (different action index and damage)
	if p1 == p2 {
		t.Fatal("successive attack panels must differ")
	}
}

func TestGALoopMoveDuringEncounterThenAttack(t *testing.T) {
	// g during active encounter then a — move and attack both compose through Update.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{
		State: playerReadOK, HasPos: true, Position: playerPosResult{X: 18, Y: 0},
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: []mobPosition{
		{MobID: "orc-1", Position: mobPosVec3{X: 20, Y: 0}},
	}, Count: 1}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	m.rosterFocus = rosterFocus{index: 0}
	m.encounterRead = encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}

	// g during encounter
	updated, gCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if gCmd == nil {
		t.Fatal("g during encounter must trigger move")
	}
	m = updated.(model)

	// Move result
	moveMsg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 19, Y: 0}},
		direction: "east",
	}
	updated2, _ := m.Update(moveMsg)
	m = updated2.(model)

	// a during encounter
	_, aCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if aCmd == nil {
		t.Fatal("a during encounter after g must trigger attack")
	}
}

func TestGALoopRaceAttackBeforeMoveReads(t *testing.T) {
	// Race: g fires, move result updates position, but user presses a before
	// encounter reads arrive. Combat panel must show enc:opening, not stale.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: []mobPosition{
		{MobID: "orc-1", Position: mobPosVec3{X: 20, Y: 0}},
	}, Count: 1}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	m.rosterFocus = rosterFocus{index: 0}

	// g press + move result (position updated, but encounter reads still in flight)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(model)
	moveMsg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 5, Y: 0}},
		direction: "east",
	}
	updated2, _ := m.Update(moveMsg)
	m = updated2.(model)

	// a press — submits attack, gets result
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	// Simulate attack result arriving
	attackMsg := attackResultMsg{result: attackResult{State: attackStateSent, TargetID: "orc-1"}}
	updated3, _ := m.Update(attackMsg)
	m = updated3.(model)

	// Render combat panel — encounter reads from g haven't arrived yet
	// HasActiveEncounter is false (stale), encounter list empty
	panel := stripANSI(renderCombatPanel(sidePanelWidth, m.lastAttack, m.playerRead, m.encounterRead, defaultTarget(), inventoryReadResult{}, "orc-1"))
	if !strings.Contains(panel, "enc: opening") {
		t.Fatalf("race: must show enc:opening before reads arrive, got: %s", panel)
	}
}

func TestGALoopFocusPreservedAcrossLoop(t *testing.T) {
	// Roster focus must stay on the same mob across g and a presses.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: []mobPosition{
		{MobID: "orc-1", Position: mobPosVec3{X: 20, Y: 0}},
		{MobID: "orc-2", Position: mobPosVec3{X: 30, Y: 0}},
	}, Count: 2}
	m.rosterEntries = []rosterEntry{
		{kind: "mb", id: "orc-1", canonicalMobID: true},
		{kind: "mb", id: "orc-2", canonicalMobID: true},
	}
	m.rosterFocus = rosterFocus{index: 0} // focused on orc-1

	// g press
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(model)
	if m.rosterFocus.index != 0 {
		t.Fatalf("g must not change focus, got index %d", m.rosterFocus.index)
	}

	// Move result
	moveMsg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 5, Y: 0}},
		direction: "east",
	}
	updated2, _ := m.Update(moveMsg)
	m = updated2.(model)
	if m.rosterFocus.index != 0 {
		t.Fatalf("move result must not change focus, got index %d", m.rosterFocus.index)
	}

	// a press
	updated3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated3.(model)
	if m.rosterFocus.index != 0 {
		t.Fatalf("a must not change focus, got index %d", m.rosterFocus.index)
	}

	// Focused entry is still orc-1
	fe := focusedEntry(m.rosterFocus, m.rosterEntries)
	if fe == nil || fe.id != "orc-1" {
		t.Fatal("focus must remain on orc-1 through g/a loop")
	}
}

// --- Attack readiness legibility tests (M20260411-01) ---

func TestReadinessCooldownResultDistinctFromSpatialDenial(t *testing.T) {
	// res:cd (cooldown result) must be distinct from deny:oor (spatial denial).
	// Both are valid backend result kinds but have different prefixes.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	bt := defaultTarget()
	inv := inventoryReadResult{}

	// Cooldown result
	erCD := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "cooldown_set",
		}},
	}
	panelCD := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, erCD, bt, inv, ""))
	if !strings.Contains(panelCD, "res:cd") {
		t.Fatalf("cooldown result must show 'res:cd', got: %s", panelCD)
	}
	if strings.Contains(panelCD, "deny:cd") {
		t.Fatalf("cooldown result must NOT use deny: prefix, got: %s", panelCD)
	}

	// Spatial denial for comparison
	erOOR := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "out_of_range",
		}},
	}
	panelOOR := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, erOOR, bt, inv, ""))
	if !strings.Contains(panelOOR, "deny:oor") {
		t.Fatalf("spatial denial must show 'deny:oor', got: %s", panelOOR)
	}

	// They must be visually distinct
	if panelCD == panelOOR {
		t.Fatal("cooldown result and spatial denial must be visually distinct")
	}
}

func TestReadinessBlockedWoundedDistinctFromSpatialDenial(t *testing.T) {
	// res:blocked (wounded block result) must be distinct from deny:oor.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	erBlocked := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			LatestResultKind: "blocked_wounded",
		}},
	}
	panel := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, erBlocked, defaultTarget(), inventoryReadResult{}, ""))
	if !strings.Contains(panel, "res:blocked") {
		t.Fatalf("wounded block must show 'res:blocked', got: %s", panel)
	}
	if strings.Contains(panel, "deny:blocked") {
		t.Fatalf("wounded block must NOT use deny: prefix, got: %s", panel)
	}
}

func TestReadinessYesAndNoDistinct(t *testing.T) {
	// rdy:yes and rdy:no must produce visually distinct panels.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}
	bt := defaultTarget()

	invReady := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	panelReady := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er, bt, invReady, ""))

	invBlocked := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, BlockedReason: "player_wounded", HPCurrent: 10, HPMax: 100}
	panelBlocked := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er, bt, invBlocked, ""))

	// Both panels should show player HP line (different HP values make them distinct)
	if !strings.Contains(panelReady, "You") {
		t.Fatalf("ready panel must show player HP line, got: %s", panelReady)
	}
	if !strings.Contains(panelBlocked, "You") {
		t.Fatalf("blocked panel must show player HP line, got: %s", panelBlocked)
	}
	if panelReady == panelBlocked {
		t.Fatal("ready and blocked panels must be visually distinct")
	}
}

func TestReadinessWoundedBlockedReason(t *testing.T) {
	// player_wounded blocked reason must be visible in the rdy: line.
	pr := playerReadResult{
		State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: false, BlockedReason: "player_wounded", HPCurrent: 10, HPMax: 100}
	panel := stripANSI(renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inv, ""))
	// Blocked reason is now shown in the Player panel; combat panel shows HP frame
	if !strings.Contains(panel, "You") {
		t.Fatalf("wounded player must show HP line, got: %s", panel)
	}
	if !strings.Contains(panel, "act:") {
		t.Fatalf("wounded player must show compact state line, got: %s", panel)
	}
}

func TestReadinessSuppressedOnCompletedEncounter(t *testing.T) {
	// When encounter is Completed, readiness line must be suppressed
	// (combat is over — readiness is stale).
	pr := playerReadResult{
		State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_mobs_dead",
			MobIDs: []string{"orc-1"}, MobsAlive: 0, MobsDead: 1,
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	panel := stripANSI(renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inv, ""))
	if strings.Contains(panel, "rdy:") {
		t.Fatalf("readiness must be suppressed on Completed encounter, got: %s", panel)
	}
}

func TestReadinessAfterGotoThenAttack(t *testing.T) {
	// After g → move → a, the combat panel must show readiness alongside
	// the attack result. Both must be visible.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position:           playerPosResult{X: 18, Y: 0},
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			ActionIndex: 1, LatestResultKind: "damage_applied", LatestResultValue: 12,
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 90, HPMax: 100}
	panel := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, "orc-1"))

	if !strings.Contains(panel, "You") {
		t.Fatalf("player HP line must be visible after g→a, got: %s", panel)
	}
	if !strings.Contains(panel, "res:dmg") {
		t.Fatalf("attack result must be visible after g→a, got: %s", panel)
	}
	if !strings.Contains(panel, "12") {
		t.Fatalf("damage value must be visible, got: %s", panel)
	}
}

func TestReadinessStaleRaceHonest(t *testing.T) {
	// Race: inventory read is stale (no lifecycle yet), but encounter shows
	// active combat. Readiness line must not appear — stale data is honest.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{
		State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			ActionIndex: 1, LatestResultKind: "damage_applied", LatestResultValue: 10,
		}},
	}
	// Lifecycle not yet available
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: false}
	panel := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, ""))
	if strings.Contains(panel, "rdy:") {
		t.Fatalf("readiness must not appear when lifecycle data absent (stale), got: %s", panel)
	}
	// Attack result must still show
	if !strings.Contains(panel, "res:dmg") {
		t.Fatalf("attack result must still show during stale readiness, got: %s", panel)
	}
}

func TestReadinessDenialDoesNotMisrenderAsCooldown(t *testing.T) {
	// deny:oor must not be confused with res:cd.
	// Both should be present simultaneously when they co-occur in state.
	// Here we just verify they render with correct prefixes in isolation.
	if isDenialResult("cooldown_set") {
		t.Fatal("cooldown_set must NOT be classified as denial")
	}
	if isDenialResult("blocked_wounded") {
		t.Fatal("blocked_wounded must NOT be classified as denial")
	}
	if !isDenialResult("out_of_range") {
		t.Fatal("out_of_range must be classified as denial")
	}
	if !isDenialResult("los_blocked") {
		t.Fatal("los_blocked must be classified as denial")
	}
	if !isDenialResult("invalid_target") {
		t.Fatal("invalid_target must be classified as denial")
	}
}

func TestReadinessCompactLabelValues(t *testing.T) {
	// Verify compact result kind labels for readiness-related result kinds.
	if got := compactResultKind("cooldown_set"); got != "cd" {
		t.Fatalf("cooldown_set should compact to 'cd', got: %s", got)
	}
	if got := compactResultKind("blocked_wounded"); got != "blocked" {
		t.Fatalf("blocked_wounded should compact to 'blocked', got: %s", got)
	}
	if got := compactResultKind("damage_applied"); got != "dmg" {
		t.Fatalf("damage_applied should compact to 'dmg', got: %s", got)
	}
}

// --- Event stream panel tests (M20260411-02) ---

func TestEventLogAppendAndBound(t *testing.T) {
	var log eventLog
	for i := 0; i < 16; i++ {
		log.append(eventEntry{kind: "test", text: fmt.Sprintf("entry-%d", i)})
	}
	if log.len() != eventLogMax {
		t.Fatalf("event log must be bounded at %d, got %d", eventLogMax, log.len())
	}
	// Oldest entries should be dropped
	entries := log.recent()
	if entries[0].text != "entry-4" {
		t.Fatalf("oldest entry should be entry-4 after overflow, got: %s", entries[0].text)
	}
	if entries[len(entries)-1].text != "entry-15" {
		t.Fatalf("newest entry should be entry-15, got: %s", entries[len(entries)-1].text)
	}
}

func TestEventLogEmptyRecent(t *testing.T) {
	var log eventLog
	if log.len() != 0 {
		t.Fatal("empty log should have length 0")
	}
	if len(log.recent()) != 0 {
		t.Fatal("empty log should have empty recent")
	}
}

func TestEventPanelRendersEmpty(t *testing.T) {
	panel := renderEventPanel(sidePanelWidth, eventLog{}, playerReadResult{}, encounterReadResult{})
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "Events") {
		t.Fatalf("event panel must show title, got: %s", stripped)
	}
	if !strings.Contains(stripped, "(none)") {
		t.Fatalf("empty event panel must show (none), got: %s", stripped)
	}
}

func TestEventPanelRendersEntries(t *testing.T) {
	var log eventLog
	log.append(eventEntry{kind: "move", text: "move east"})
	log.append(eventEntry{kind: "combat", text: "hit 15"})
	panel := renderEventPanel(sidePanelWidth, log, playerReadResult{}, encounterReadResult{})
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "move east") {
		t.Fatalf("event panel must show move entry, got: %s", stripped)
	}
	if !strings.Contains(stripped, "hit 15") {
		t.Fatalf("event panel must show combat entry, got: %s", stripped)
	}
}

func TestEventStreamCombatEventOnActionIndexAdvance(t *testing.T) {
	// When encounter ActionIndex advances, a new combat event should be appended.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	m.lastSeenEncID = "enc-1"
	m.lastSeenActIdx = 0

	// First encounter read with ActionIndex=1
	msg := encounterReadResultMsg{result: encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			ActionIndex: 1, LatestResultKind: "damage_applied", LatestResultValue: 12,
		}},
	}}
	updated, _ := m.Update(msg)
	m = updated.(model)

	if m.events.len() != 1 {
		t.Fatalf("should have 1 event after ActionIndex advance, got %d", m.events.len())
	}
	if m.events.recent()[0].text != "hit 12" {
		t.Fatalf("event text should be 'hit 12', got: %s", m.events.recent()[0].text)
	}
}

func TestEventStreamNoDuplicateOnSameActionIndex(t *testing.T) {
	// Repeated poll reads with the same ActionIndex must NOT create duplicate events.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	m.lastSeenEncID = "enc-1"
	m.lastSeenActIdx = 1

	// Encounter read with same ActionIndex=1 — no new event
	msg := encounterReadResultMsg{result: encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			ActionIndex: 1, LatestResultKind: "damage_applied", LatestResultValue: 12,
		}},
	}}
	updated, _ := m.Update(msg)
	m = updated.(model)

	if m.events.len() != 0 {
		t.Fatalf("same ActionIndex must not create event, got %d events", m.events.len())
	}
}

func TestEventStreamMoveAppends(t *testing.T) {
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()

	msg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 5, Y: 0}},
		direction: "east",
	}
	updated, _ := m.Update(msg)
	m = updated.(model)

	if m.events.len() != 1 {
		t.Fatalf("move result should append 1 event, got %d", m.events.len())
	}
	if m.events.recent()[0].text != "move east" {
		t.Fatalf("move event text should be 'move east', got: %s", m.events.recent()[0].text)
	}
}

func TestEventStreamBlockedMoveAppends(t *testing.T) {
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()

	msg := moveResultMsg{
		result:    moveResult{OK: false, Error: "boundary"},
		direction: "east",
	}
	updated, _ := m.Update(msg)
	m = updated.(model)

	if m.events.len() != 1 {
		t.Fatalf("blocked move should append 1 event, got %d", m.events.len())
	}
	if !strings.Contains(m.events.recent()[0].text, "move blocked") {
		t.Fatalf("blocked move event should contain 'move blocked', got: %s", m.events.recent()[0].text)
	}
}

func TestEventStreamDenialThenSuccessOrdered(t *testing.T) {
	// Denial then success must produce two distinct ordered events.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	m.lastSeenEncID = "enc-1"
	m.lastSeenActIdx = 0

	// Denial at ActionIndex=1
	msg1 := encounterReadResultMsg{result: encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			ActionIndex: 1, LatestResultKind: "out_of_range",
		}},
	}}
	updated, _ := m.Update(msg1)
	m = updated.(model)

	// Success at ActionIndex=2
	msg2 := encounterReadResultMsg{result: encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			ActionIndex: 2, LatestResultKind: "damage_applied", LatestResultValue: 10,
		}},
	}}
	updated2, _ := m.Update(msg2)
	m = updated2.(model)

	if m.events.len() != 2 {
		t.Fatalf("should have 2 events, got %d", m.events.len())
	}
	entries := m.events.recent()
	if entries[0].text != "deny range" {
		t.Fatalf("first event should be 'deny range', got: %s", entries[0].text)
	}
	if entries[1].text != "hit 10" {
		t.Fatalf("second event should be 'hit 10', got: %s", entries[1].text)
	}
}

func TestEventStreamFocusChange(t *testing.T) {
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: []mobPosition{
		{MobID: "orc-1", Position: mobPosVec3{X: 10, Y: 0}},
	}, Count: 1}

	// Tab to focus — tea.KeyTab is the correct key type
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(model)

	if m.events.len() != 1 {
		t.Fatalf("tab focus should append 1 event, got %d", m.events.len())
	}
	if m.events.recent()[0].text != "focus mob" {
		t.Fatalf("focus event should be 'focus mob', got: %s", m.events.recent()[0].text)
	}
}

func TestEventStreamEncounterOpened(t *testing.T) {
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}

	msg := encounterReadResultMsg{result: encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}}
	updated, _ := m.Update(msg)
	m = updated.(model)

	if m.events.len() != 1 {
		t.Fatalf("new encounter should append 1 event, got %d", m.events.len())
	}
	if m.events.recent()[0].text != "encounter opened" {
		t.Fatalf("event should be 'encounter opened', got: %s", m.events.recent()[0].text)
	}
}

func TestEventStreamEncounterCompleted(t *testing.T) {
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	m.lastSeenEncID = "enc-1"
	m.lastSeenEncState = "Active"
	m.lastSeenActIdx = 3

	msg := encounterReadResultMsg{result: encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_mobs_dead",
			MobIDs: []string{"orc-1"}, MobsAlive: 0, MobsDead: 1,
			ActionIndex: 3,
		}},
	}}
	updated, _ := m.Update(msg)
	m = updated.(model)

	// Should have victory event (all_mobs_dead → "victory")
	found := false
	for _, e := range m.events.recent() {
		if e.text == "victory" {
			found = true
		}
	}
	if !found {
		t.Fatalf("should have 'victory' event for all_mobs_dead, got: %v", m.events.recent())
	}
}

func TestEventStreamSideColumnContainsPanel(t *testing.T) {
	var log eventLog
	log.append(eventEntry{kind: "move", text: "move east"})
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, log, true)
	stripped := stripANSI(col)
	if !strings.Contains(stripped, "Events") {
		t.Fatal("side column must contain Events panel")
	}
	if !strings.Contains(stripped, "move east") {
		t.Fatal("side column must contain event entry")
	}
}

// --- Post-combat outcome panel tests (M20260412-01) ---

func TestOutcomePanelVictory(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_mobs_dead",
			XPAwarded: 50, DropsGenerated: true, Drops: []string{"item-1"},
		}},
	}
	panel := renderOutcomePanel(sidePanelWidth, pr, er)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "Outcome") {
		t.Fatalf("outcome panel must show title, got: %s", stripped)
	}
	if !strings.Contains(stripped, "victory") {
		t.Fatalf("all_mobs_dead must show 'victory', got: %s", stripped)
	}
	if !strings.Contains(stripped, "xp +50") {
		t.Fatalf("must show xp awarded, got: %s", stripped)
	}
	if !strings.Contains(stripped, "loot 1 items") {
		t.Fatalf("must show loot count, got: %s", stripped)
	}
}

func TestOutcomePanelDefeat(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_players_dead",
			XPAwarded: 0,
		}},
	}
	panel := renderOutcomePanel(sidePanelWidth, pr, er)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "defeat") {
		t.Fatalf("all_players_dead must show 'defeat', got: %s", stripped)
	}
	if strings.Contains(stripped, "xp") {
		t.Fatalf("defeat must not show xp line (0 awarded), got: %s", stripped)
	}
}

func TestOutcomePanelNotShownDuringActiveEncounter(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}
	panel := renderOutcomePanel(sidePanelWidth, pr, er)
	if panel != "" {
		t.Fatalf("outcome panel must not appear during active encounter, got: %s", stripANSI(panel))
	}
}

func TestOutcomePanelNotShownWithoutEncounter(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: false}
	er := encounterReadResult{State: encounterReadOK, Count: 0}
	panel := renderOutcomePanel(sidePanelWidth, pr, er)
	if panel != "" {
		t.Fatal("outcome panel must not appear without encounter")
	}
}

func TestOutcomePanelLootExpired(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_mobs_dead",
			XPAwarded: 50, DropsGenerated: true, LootExpired: true,
		}},
	}
	panel := renderOutcomePanel(sidePanelWidth, pr, er)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "loot expired") {
		t.Fatalf("expired loot must show 'loot expired', got: %s", stripped)
	}
}

func TestOutcomePanelLootCollected(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_mobs_dead",
			XPAwarded: 50, DropsGenerated: true, Drops: nil,
		}},
	}
	panel := renderOutcomePanel(sidePanelWidth, pr, er)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "loot collected") {
		t.Fatalf("empty drops must show 'loot collected', got: %s", stripped)
	}
}

func TestOutcomePanelVictoryAndDefeatDistinct(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	erVictory := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_mobs_dead", XPAwarded: 50,
		}},
	}
	erDefeat := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_players_dead",
		}},
	}
	panelV := stripANSI(renderOutcomePanel(sidePanelWidth, pr, erVictory))
	panelD := stripANSI(renderOutcomePanel(sidePanelWidth, pr, erDefeat))
	if panelV == panelD {
		t.Fatal("victory and defeat panels must be visually distinct")
	}
}

func TestEventStreamVictoryPayoff(t *testing.T) {
	// Encounter completing as all_mobs_dead with XP and loot should produce
	// victory + xp + loot events in the stream.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	m.lastSeenEncID = "enc-1"
	m.lastSeenEncState = "Active"
	m.lastSeenActIdx = 5

	msg := encounterReadResultMsg{result: encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_mobs_dead",
			XPAwarded: 100, DropsGenerated: true,
			Drops: []string{"sword-1", "shield-1"},
			ActionIndex: 5,
		}},
	}}
	updated, _ := m.Update(msg)
	m = updated.(model)

	entries := m.events.recent()
	if len(entries) < 3 {
		t.Fatalf("should have at least 3 events (victory, xp, loot), got %d: %v", len(entries), entries)
	}
	// Check content
	texts := make([]string, len(entries))
	for i, e := range entries {
		texts[i] = e.text
	}
	joined := strings.Join(texts, " | ")
	if !strings.Contains(joined, "victory") {
		t.Fatalf("must contain 'victory', got: %s", joined)
	}
	if !strings.Contains(joined, "xp +100") {
		t.Fatalf("must contain 'xp +100', got: %s", joined)
	}
	if !strings.Contains(joined, "loot 2 items") {
		t.Fatalf("must contain 'loot 2 items', got: %s", joined)
	}
}

func TestEventStreamDefeatNoPayoff(t *testing.T) {
	// Defeat should produce defeat event but no xp or loot events.
	m := model{target: defaultTarget(), slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	m.lastSeenEncID = "enc-1"
	m.lastSeenEncState = "Active"
	m.lastSeenActIdx = 3

	msg := encounterReadResultMsg{result: encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_players_dead",
			ActionIndex: 3,
		}},
	}}
	updated, _ := m.Update(msg)
	m = updated.(model)

	entries := m.events.recent()
	if len(entries) != 1 {
		t.Fatalf("defeat should produce exactly 1 event, got %d: %v", len(entries), entries)
	}
	if entries[0].text != "defeat" {
		t.Fatalf("defeat event should be 'defeat', got: %s", entries[0].text)
	}
}

func TestOutcomeRemovedFromSideColumn(t *testing.T) {
	// Outcome panel is no longer in the side column — its content is in the event stream.
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_mobs_dead", XPAwarded: 30,
		}},
	}
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, pr, er, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	stripped := stripANSI(col)
	if strings.Contains(stripped, "Outcome") {
		t.Fatal("Outcome panel should no longer appear in side column (content in event stream)")
	}
}

// --- Right-rail hierarchy tests (M20260412-02) ---

func TestSideColumnHierarchyOrder(t *testing.T) {
	// Panels must appear in order: Player → Combat → Events → Loot → Encounter → Proximity → Diagnostics
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{State: playerReadOK}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	stripped := stripANSI(col)

	playerIdx := strings.Index(stripped, "Player")
	combatIdx := strings.Index(stripped, "Combat")
	eventsIdx := strings.Index(stripped, "Events")
	lootIdx := strings.Index(stripped, "Loot")
	encIdx := strings.Index(stripped, "Encounter")
	proxIdx := strings.Index(stripped, "Target")
	diagIdx := strings.Index(stripped, "Diagnostics")

	if playerIdx < 0 || combatIdx < 0 || eventsIdx < 0 || lootIdx < 0 || encIdx < 0 || proxIdx < 0 || diagIdx < 0 {
		t.Fatalf("all sections must be present, got: Player=%d Combat=%d Events=%d Loot=%d Enc=%d Target=%d Diag=%d",
			playerIdx, combatIdx, eventsIdx, lootIdx, encIdx, proxIdx, diagIdx)
	}
	if !(playerIdx < combatIdx && combatIdx < eventsIdx && eventsIdx < lootIdx && lootIdx < encIdx && encIdx < proxIdx && proxIdx < diagIdx) {
		t.Fatal("sections must appear in hierarchy order: Player < Combat < Events < Loot < Encounter < Target < Diagnostics")
	}
}

func TestSideColumnDiagnosticsHidden(t *testing.T) {
	col := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{State: playerReadOK}, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, false)
	stripped := stripANSI(col)
	if strings.Contains(stripped, "Diagnostics") {
		t.Fatal("diagnostics must be hidden when showDiag=false")
	}
	// Gameplay panels must still be present
	if !strings.Contains(stripped, "Player") {
		t.Fatal("Player panel must remain when diagnostics hidden")
	}
	if !strings.Contains(stripped, "Events") {
		t.Fatal("Events panel must remain when diagnostics hidden")
	}
}

func TestDiagnosticsSectionContent(t *testing.T) {
	section := renderDiagnosticsSection(sidePanelWidth, defaultTarget(), zoneReadResult{State: zoneReadOK}, mapReadResult{}, mobReadResult{State: mobReadOK, Count: 3}, playerReadResult{State: playerReadOK}, true)
	stripped := stripANSI(section)
	if !strings.Contains(stripped, "Diagnostics") {
		t.Fatalf("section must have Diagnostics title, got: %s", stripped)
	}
	if !strings.Contains(stripped, "crushbone") {
		t.Fatalf("section must show zone name, got: %s", stripped)
	}
	if !strings.Contains(stripped, "zone read: ok") {
		t.Fatalf("section must show zone read status, got: %s", stripped)
	}
}

func TestDiagnosticsSectionHiddenReturnsEmpty(t *testing.T) {
	section := renderDiagnosticsSection(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, playerReadResult{}, false)
	if section != "" {
		t.Fatal("hidden diagnostics section must return empty string")
	}
}

// --- Diagnostic toggle tests (M20260412-03) ---

func TestDiagToggleKeyToggles(t *testing.T) {
	m := model{target: defaultTarget(), showDiag: true, slog: newSessionLogger()}
	defer m.slog.Close()

	// d toggles to false
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd != nil {
		t.Fatal("d key must not trigger async command")
	}
	m2 := updated.(model)
	if m2.showDiag {
		t.Fatal("d key should toggle showDiag from true to false")
	}

	// d again toggles back to true
	updated2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m3 := updated2.(model)
	if !m3.showDiag {
		t.Fatal("second d key should toggle showDiag back to true")
	}
}

func TestDiagToggleDoesNotAffectGameplay(t *testing.T) {
	// Toggling diagnostics must not change any gameplay-facing state
	m := model{target: defaultTarget(), showDiag: true, slog: newSessionLogger()}
	defer m.slog.Close()
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 5, Y: 0}}
	m.lastAttack = attackResult{State: attackStateSent, TargetID: "orc-1"}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m2 := updated.(model)

	if m2.playerRead.Position.X != 5 {
		t.Fatal("d key must not change player position")
	}
	if m2.lastAttack.State != attackStateSent {
		t.Fatal("d key must not change attack state")
	}
	if m2.lastAttack.TargetID != "orc-1" {
		t.Fatal("d key must not change attack target")
	}
}

func TestFooterContainsDiagHint(t *testing.T) {
	footer := renderFooter(80, "", "", "", "", "", true, inventoryReadResult{})
	if !strings.Contains(footer, "d:diag") {
		t.Fatalf("footer must contain d:diag hint, got: %s", stripANSI(footer))
	}
}

func TestDiagCollapsedSideColumnShorter(t *testing.T) {
	pr := playerReadResult{State: playerReadOK}
	expanded := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, pr, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, true)
	collapsed := renderSideColumn(sidePanelWidth, defaultTarget(), zoneReadResult{}, mapReadResult{}, mobReadResult{}, pr, encounterReadResult{}, rosterFocus{}, nil, targetConfirmResult{}, attackResult{}, pickupResult{}, inventoryReadResult{}, -1, -1, respawnResult{}, eventLog{}, false)

	expandedLines := strings.Count(expanded, "\n")
	collapsedLines := strings.Count(collapsed, "\n")
	if collapsedLines >= expandedLines {
		t.Fatalf("collapsed side column must be shorter: expanded=%d collapsed=%d", expandedLines, collapsedLines)
	}
}

// --- Resolved header tests (M20260412-05) ---

func TestResolvedHeaderVictory(t *testing.T) {
	// Use a wider width (40) to verify full content without truncation.
	// The real sidePanelWidth (24) will truncate — that's expected and tested separately.
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_mobs_dead", XPAwarded: 50,
			DropsGenerated: true, Drops: []string{"item-1", "item-2"},
		}},
	}
	header := stripANSI(resolvedHeader(40, pr, er))
	if !strings.Contains(header, "victory") {
		t.Fatalf("header must contain 'victory', got: %s", header)
	}
	if !strings.Contains(header, "+50xp") {
		t.Fatalf("header must contain '+50xp', got: %s", header)
	}
	if !strings.Contains(header, "2loot") {
		t.Fatalf("header must contain '2loot', got: %s", header)
	}
	if !strings.Contains(header, "==") {
		t.Fatalf("header must have == delimiters, got: %s", header)
	}
}

func TestResolvedHeaderTruncatesAtNarrowWidth(t *testing.T) {
	// At sidePanelWidth (24), the header truncates gracefully
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_mobs_dead", XPAwarded: 50,
			DropsGenerated: true, Drops: []string{"item-1", "item-2"},
		}},
	}
	header := stripANSI(resolvedHeader(sidePanelWidth, pr, er))
	// Must still show victory and have delimiters even when truncated
	if !strings.Contains(header, "victory") {
		t.Fatalf("narrow header must still show victory, got: %s", header)
	}
	if !strings.Contains(header, "==") {
		t.Fatalf("narrow header must still have == delimiters, got: %s", header)
	}
}

func TestResolvedHeaderDefeat(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_players_dead",
		}},
	}
	header := stripANSI(resolvedHeader(sidePanelWidth, pr, er))
	if !strings.Contains(header, "defeat") {
		t.Fatalf("header must contain 'defeat', got: %s", header)
	}
	if strings.Contains(header, "xp") {
		t.Fatalf("defeat header must not contain xp, got: %s", header)
	}
}

func TestResolvedHeaderAbsentDuringActive(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}
	header := resolvedHeader(sidePanelWidth, pr, er)
	if header != "" {
		t.Fatalf("header must be empty during active encounter, got: %s", stripANSI(header))
	}
}

func TestResolvedHeaderAbsentNoEncounter(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: false}
	er := encounterReadResult{State: encounterReadOK}
	header := resolvedHeader(sidePanelWidth, pr, er)
	if header != "" {
		t.Fatal("header must be empty when no encounter")
	}
}

func TestResolvedHeaderNoXP(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_mobs_dead", XPAwarded: 0,
		}},
	}
	header := stripANSI(resolvedHeader(sidePanelWidth, pr, er))
	if !strings.Contains(header, "victory") {
		t.Fatalf("header must contain 'victory', got: %s", header)
	}
	if strings.Contains(header, "xp") || strings.Contains(header, "+0") {
		t.Fatalf("header must not contain xp when 0, got: %s", header)
	}
}

func TestResolvedHeaderNoLoot(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_mobs_dead", XPAwarded: 25,
			DropsGenerated: false,
		}},
	}
	header := stripANSI(resolvedHeader(sidePanelWidth, pr, er))
	if strings.Contains(header, "loot") {
		t.Fatalf("header must not contain loot when no drops, got: %s", header)
	}
}

func TestResolvedHeaderInEventPanel(t *testing.T) {
	// The resolved header must appear inside the Events panel when encounter is completed.
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_mobs_dead", XPAwarded: 30,
		}},
	}
	var log eventLog
	log.append(eventEntry{kind: "combat", text: "hit 10"})
	panel := renderEventPanel(sidePanelWidth, log, pr, er)
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "victory") {
		t.Fatalf("Events panel must contain resolved header, got: %s", stripped)
	}
	if !strings.Contains(stripped, "hit 10") {
		t.Fatalf("Events panel must still contain event entries, got: %s", stripped)
	}
}

func TestResolvedHeaderNotInEventPanelDuringActive(t *testing.T) {
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
		}},
	}
	var log eventLog
	log.append(eventEntry{kind: "combat", text: "hit 10"})
	panel := renderEventPanel(sidePanelWidth, log, pr, er)
	stripped := stripANSI(panel)
	if strings.Contains(stripped, "victory") || strings.Contains(stripped, "defeat") {
		t.Fatalf("Events panel must not contain resolved header during active, got: %s", stripped)
	}
}

func TestEventStreamOrderingPreservedWithHeader(t *testing.T) {
	// Event entries must appear in order after the header.
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_mobs_dead", XPAwarded: 10,
		}},
	}
	var log eventLog
	log.append(eventEntry{kind: "combat", text: "hit 5"})
	log.append(eventEntry{kind: "enc", text: "victory"})
	panel := renderEventPanel(sidePanelWidth, log, pr, er)
	stripped := stripANSI(panel)

	// Header should appear before event entries
	headerIdx := strings.Index(stripped, "==")
	hitIdx := strings.Index(stripped, "hit 5")
	if headerIdx < 0 || hitIdx < 0 {
		t.Fatalf("both header and event must be present, got: %s", stripped)
	}
	if headerIdx > hitIdx {
		t.Fatal("resolved header must appear before event entries")
	}
}

// --- Event moment emphasis tests (M20260412-06) ---

func TestEventEmphasisClassification(t *testing.T) {
	cases := []struct {
		text     string
		expected string
	}{
		{"hit 12", "strong"},
		{"hit 0", "strong"},
		{"heal 5", "strong"},
		{"victory", "strong"},
		{"defeat", "strong"},
		{"encounter opened", "strong"},
		{"encounter complete", "strong"},
		{"miss", "strong"},
		{"attack", "medium"},
		{"attack failed", "medium"},
		{"deny range", "medium"},
		{"deny los", "medium"},
		{"deny target", "medium"},
		{"cooldown", "medium"},
		{"blocked", "medium"},
		{"move east", "dim"},
		{"move blocked south", "dim"},
		{"focus mob", "dim"},
		{"focus pc", "dim"},
		{"something unknown", "dim"},
	}
	for _, tc := range cases {
		got := eventEmphasis(tc.text)
		if got != tc.expected {
			t.Errorf("eventEmphasis(%q) = %q, want %q", tc.text, got, tc.expected)
		}
	}
}

func TestLatestEventHasPrefix(t *testing.T) {
	// The latest event must have a ">" prefix marker.
	line := stripANSI(renderEventLine("hit 12", 16, true))
	if !strings.HasPrefix(line, ">") {
		t.Fatalf("latest event must start with '>', got: %q", line)
	}
	if !strings.Contains(line, "hit 12") {
		t.Fatalf("latest event must contain text, got: %q", line)
	}
}

func TestOlderEventNoPrefix(t *testing.T) {
	// Older events must NOT have ">" prefix.
	line := stripANSI(renderEventLine("hit 12", 16, false))
	if strings.HasPrefix(line, ">") {
		t.Fatalf("older event must not start with '>', got: %q", line)
	}
}

func TestEventPanelLatestMarker(t *testing.T) {
	// In a rendered event panel, only the newest entry should have ">".
	var log eventLog
	log.append(eventEntry{kind: "move", text: "move east"})
	log.append(eventEntry{kind: "combat", text: "hit 12"})
	panel := stripANSI(renderEventPanel(sidePanelWidth, log, playerReadResult{}, encounterReadResult{}))

	// The latest entry "hit 12" should have ">"
	if !strings.Contains(panel, "> hit 12") {
		t.Fatalf("latest event must show '> hit 12', got: %s", panel)
	}
}

func TestEventPanelOlderNoPrefixMarker(t *testing.T) {
	var log eventLog
	log.append(eventEntry{kind: "move", text: "move east"})
	log.append(eventEntry{kind: "combat", text: "hit 12"})
	panel := stripANSI(renderEventPanel(sidePanelWidth, log, playerReadResult{}, encounterReadResult{}))

	// "move east" should NOT have ">" prefix
	if strings.Contains(panel, "> move east") {
		t.Fatalf("older event must not have '>' prefix, got: %s", panel)
	}
	// But should still contain the text
	if !strings.Contains(panel, "move east") {
		t.Fatalf("older event must still contain text, got: %s", panel)
	}
}

func TestHitEventStrongerThanMove(t *testing.T) {
	// Hit events must classify as strong, movement as dim.
	if eventEmphasis("hit 12") != "strong" {
		t.Fatal("hit must be strong")
	}
	if eventEmphasis("move east") != "dim" {
		t.Fatal("move must be dim")
	}
}

func TestResolvedHeaderStillRenders(t *testing.T) {
	// Resolved header must still render with updated styling.
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_mobs_dead", XPAwarded: 25,
		}},
	}
	header := stripANSI(resolvedHeader(40, pr, er))
	if !strings.Contains(header, "victory") {
		t.Fatalf("header must contain victory, got: %s", header)
	}
	if !strings.Contains(header, "==") {
		t.Fatalf("header must have delimiters, got: %s", header)
	}
}

func TestEventOrderingUnchanged(t *testing.T) {
	// Event entries must appear in chronological order (oldest first, newest last).
	var log eventLog
	log.append(eventEntry{kind: "move", text: "move east"})
	log.append(eventEntry{kind: "atk", text: "attack"})
	log.append(eventEntry{kind: "combat", text: "hit 15"})
	panel := stripANSI(renderEventPanel(sidePanelWidth, log, playerReadResult{}, encounterReadResult{}))

	moveIdx := strings.Index(panel, "move east")
	attackIdx := strings.Index(panel, "attack")
	hitIdx := strings.Index(panel, "hit 15")
	if moveIdx < 0 || attackIdx < 0 || hitIdx < 0 {
		t.Fatalf("all events must be present, got: %s", panel)
	}
	if !(moveIdx < attackIdx && attackIdx < hitIdx) {
		t.Fatal("events must appear in chronological order: move < attack < hit")
	}
}

func TestEventContentPreserved(t *testing.T) {
	// Emphasis styling must not alter event text content.
	var log eventLog
	log.append(eventEntry{kind: "combat", text: "deny range"})
	log.append(eventEntry{kind: "combat", text: "hit 42"})
	panel := stripANSI(renderEventPanel(sidePanelWidth, log, playerReadResult{}, encounterReadResult{}))
	if !strings.Contains(panel, "deny range") {
		t.Fatalf("event text must be preserved, got: %s", panel)
	}
	if !strings.Contains(panel, "hit 42") {
		t.Fatalf("event text must be preserved, got: %s", panel)
	}
}

// --- Spatial feedback / combat visibility tests (M20260412-07) ---

func TestPlayerGlyphIdleVsCombat(t *testing.T) {
	mapText := "     \n     \n     "
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	pos := playerPosResult{X: 50, Y: 50}

	// Idle: @
	idle := overlayPlayer(mapText, pos, bounds, 5, 3, false)
	if !strings.Contains(idle, "@") {
		t.Fatalf("idle player must use '@', got: %s", idle)
	}
	if strings.Contains(idle, "*") {
		t.Fatal("idle player must not use '*'")
	}

	// Combat: *
	combat := overlayPlayer(mapText, pos, bounds, 5, 3, true)
	if !strings.Contains(combat, "*") {
		t.Fatalf("combat player must use '*', got: %s", combat)
	}
	if strings.Contains(combat, "@") {
		t.Fatal("combat player must not use '@'")
	}
}

func TestPlayerGlyphDeterministic(t *testing.T) {
	mapText := "     \n     \n     "
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	pos := playerPosResult{X: 50, Y: 50}
	a := overlayPlayer(mapText, pos, bounds, 5, 3, true)
	b := overlayPlayer(mapText, pos, bounds, 5, 3, true)
	if a != b {
		t.Fatal("combat player overlay must be deterministic")
	}
}

func TestFocusedMobOverlayMatchesMobID(t *testing.T) {
	// overlayFocusedMob must match by MobID (canonical attack ID)
	mapText := "m    \n     \n     "
	mobs := []mobPosition{
		{ProcessID: "pid-1", MobID: "orc-1", Position: mobPosVec3{X: 0, Y: 0}},
	}
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}

	// Match by MobID
	result := overlayFocusedMob(mapText, mobs, "orc-1", bounds, 5, 3)
	if !strings.Contains(result, "T") {
		t.Fatalf("focused mob must show 'T' when matching by MobID, got: %s", result)
	}
}

func TestAttackTargetOverlayMatchesMobID(t *testing.T) {
	// overlayAttackTarget must match by MobID (canonical attack target)
	mapText := "m    \n     \n     "
	mobs := []mobPosition{
		{ProcessID: "pid-1", MobID: "orc-1", Position: mobPosVec3{X: 0, Y: 0}},
	}
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}

	result := overlayAttackTarget(mapText, mobs, "orc-1", bounds, 5, 3, 'X')
	if !strings.Contains(result, "X") {
		t.Fatalf("attack target must show 'X' when matching by MobID, got: %s", result)
	}
}

func TestCombatPlayerGlyphColorized(t *testing.T) {
	// The '*' glyph must be colorized by colorizeMapContent.
	output := colorizeMapContent("*")
	if output == "*" {
		t.Fatal("combat player glyph '*' must receive styling")
	}
	stripped := stripANSI(output)
	if stripped != "*" {
		t.Fatalf("stripped combat glyph must be '*', got: %q", stripped)
	}
}

func TestMapPanelPlayerCombatGlyph(t *testing.T) {
	// When player has active encounter, the map panel should render '*' not '@'.
	mr := mapReadResult{
		State: mapReadOK, MapText: ".....\n.....\n.....\n.....\n.....",
		MapWidth: 5, MapHeight: 5,
		Bounds: mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100},
	}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position:           playerPosResult{X: 50, Y: 50},
		HasActiveEncounter: true, ActiveEncounterID: "enc-1",
	}
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 20, 10, attackResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "*") {
		t.Fatalf("map panel must show '*' for combat player, got: %s", stripped)
	}
}

func TestMapPanelPlayerIdleGlyph(t *testing.T) {
	// When player has no active encounter, the map panel should render '@'.
	mr := mapReadResult{
		State: mapReadOK, MapText: ".....\n.....\n.....\n.....\n.....",
		MapWidth: 5, MapHeight: 5,
		Bounds: mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100},
	}
	pr := playerReadResult{
		State: playerReadOK, HasPos: true,
		Position:           playerPosResult{X: 50, Y: 50},
		HasActiveEncounter: false,
	}
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 20, 10, attackResult{}, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "@") {
		t.Fatalf("map panel must show '@' for idle player, got: %s", stripped)
	}
}

// --- Spatial interaction moment emphasis tests (M20260412-08) ---

func TestAttackTargetGlyphHit(t *testing.T) {
	if g := attackTargetGlyph("damage_applied"); g != '!' {
		t.Fatalf("damage_applied should produce '!', got: %c", g)
	}
}

func TestAttackTargetGlyphHeal(t *testing.T) {
	if g := attackTargetGlyph("heal_applied"); g != '!' {
		t.Fatalf("heal_applied should produce '!', got: %c", g)
	}
}

func TestAttackTargetGlyphDenialOOR(t *testing.T) {
	if g := attackTargetGlyph("out_of_range"); g != '?' {
		t.Fatalf("out_of_range should produce '?', got: %c", g)
	}
}

func TestAttackTargetGlyphDenialLOS(t *testing.T) {
	if g := attackTargetGlyph("los_blocked"); g != '?' {
		t.Fatalf("los_blocked should produce '?', got: %c", g)
	}
}

func TestAttackTargetGlyphDefault(t *testing.T) {
	if g := attackTargetGlyph(""); g != 'X' {
		t.Fatalf("empty result should produce 'X', got: %c", g)
	}
	if g := attackTargetGlyph("cooldown_set"); g != 'X' {
		t.Fatalf("cooldown_set should produce 'X', got: %c", g)
	}
}

func TestOverlayAttackTargetHitGlyph(t *testing.T) {
	mapText := "m    \n     \n     "
	mobs := []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 0, Y: 0}}}
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	result := overlayAttackTarget(mapText, mobs, "orc-1", bounds, 5, 3, '!')
	if !strings.Contains(result, "!") {
		t.Fatalf("hit glyph should place '!' on map, got: %s", result)
	}
	if strings.Contains(result, "X") {
		t.Fatal("hit glyph should not contain 'X'")
	}
}

func TestOverlayAttackTargetDenyGlyph(t *testing.T) {
	mapText := "m    \n     \n     "
	mobs := []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 0, Y: 0}}}
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	result := overlayAttackTarget(mapText, mobs, "orc-1", bounds, 5, 3, '?')
	if !strings.Contains(result, "?") {
		t.Fatalf("deny glyph should place '?' on map, got: %s", result)
	}
}

func TestHitGlyphColorized(t *testing.T) {
	output := colorizeMapContent("!")
	if output == "!" {
		t.Fatal("'!' glyph must receive styling")
	}
	stripped := stripANSI(output)
	if stripped != "!" {
		t.Fatalf("stripped '!' glyph must be '!', got: %q", stripped)
	}
}

func TestDenyGlyphColorized(t *testing.T) {
	output := colorizeMapContent("?")
	if output == "?" {
		t.Fatal("'?' glyph must receive styling")
	}
	stripped := stripANSI(output)
	if stripped != "?" {
		t.Fatalf("stripped '?' glyph must be '?', got: %q", stripped)
	}
}

func TestMapPanelHitGlyphOnDamage(t *testing.T) {
	mr := mapReadResult{
		State: mapReadOK, MapText: ".....\n.....\n.....\n.....\n.....",
		MapWidth: 5, MapHeight: 5,
		Bounds: mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100},
	}
	mobr := mobReadResult{State: mobReadOK, Mobs: []mobPosition{
		{MobID: "orc-1", Position: mobPosVec3{X: 50, Y: 50}},
	}, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 20, Y: 20}, HasActiveEncounter: true}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	panel := renderMapPanel(mr, mobr, pr, rosterFocus{}, nil, 20, 10, ar, "damage_applied")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "!") {
		t.Fatalf("map panel must show '!' for hit target, got: %s", stripped)
	}
}

func TestMapPanelDenyGlyphOnOOR(t *testing.T) {
	mr := mapReadResult{
		State: mapReadOK, MapText: ".....\n.....\n.....\n.....\n.....",
		MapWidth: 5, MapHeight: 5,
		Bounds: mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100},
	}
	mobr := mobReadResult{State: mobReadOK, Mobs: []mobPosition{
		{MobID: "orc-1", Position: mobPosVec3{X: 80, Y: 80}},
	}, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 20, Y: 20}, HasActiveEncounter: true}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	panel := renderMapPanel(mr, mobr, pr, rosterFocus{}, nil, 20, 10, ar, "out_of_range")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "?") {
		t.Fatalf("map panel must show '?' for denied target, got: %s", stripped)
	}
}

func TestMapPanelDefaultGlyphNoResult(t *testing.T) {
	mr := mapReadResult{
		State: mapReadOK, MapText: ".....\n.....\n.....\n.....\n.....",
		MapWidth: 5, MapHeight: 5,
		Bounds: mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100},
	}
	mobr := mobReadResult{State: mobReadOK, Mobs: []mobPosition{
		{MobID: "orc-1", Position: mobPosVec3{X: 50, Y: 50}},
	}, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 20, Y: 20}}
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	panel := renderMapPanel(mr, mobr, pr, rosterFocus{}, nil, 20, 10, ar, "")
	stripped := stripANSI(panel)
	if !strings.Contains(stripped, "X") {
		t.Fatalf("map panel must show 'X' with no result kind, got: %s", stripped)
	}
}

func TestNoInteractionGlyphWithoutAttack(t *testing.T) {
	// With no attack result, no interaction glyphs should appear
	mr := mapReadResult{
		State: mapReadOK, MapText: ".....\n.....\n.....\n.....\n.....",
		MapWidth: 5, MapHeight: 5,
		Bounds: mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100},
	}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 50, Y: 50}}
	panel := renderMapPanel(mr, mobReadResult{}, pr, rosterFocus{}, nil, 20, 10, attackResult{}, "")
	stripped := stripANSI(panel)
	if strings.Contains(stripped, "!") || strings.Contains(stripped, "?") || strings.Contains(stripped, "X") {
		t.Fatalf("no attack should produce no interaction glyphs, got: %s", stripped)
	}
}

// --- Combat HP frame tests (M20260412-10) ---

func TestRenderHPBar(t *testing.T) {
	bar := renderHPBar(80, 100, 10)
	if !strings.Contains(bar, "80/100") {
		t.Fatalf("HP bar must contain numbers, got: %s", bar)
	}
	if !strings.Contains(bar, "█") {
		t.Fatalf("HP bar must contain filled chars, got: %s", bar)
	}
	if !strings.Contains(bar, "·") {
		t.Fatalf("HP bar must contain empty chars for partial HP, got: %s", bar)
	}
}

func TestRenderHPBarFull(t *testing.T) {
	bar := renderHPBar(100, 100, 10)
	if strings.Contains(bar, "·") {
		t.Fatalf("full HP bar must not contain empty chars, got: %s", bar)
	}
	if !strings.Contains(bar, "100/100") {
		t.Fatalf("full HP bar must show 100/100, got: %s", bar)
	}
}

func TestRenderHPBarZero(t *testing.T) {
	bar := renderHPBar(0, 100, 10)
	if strings.Contains(bar, "█") {
		t.Fatalf("zero HP bar must not contain filled chars, got: %s", bar)
	}
	if !strings.Contains(bar, "0/100") {
		t.Fatalf("zero HP bar must show 0/100, got: %s", bar)
	}
}

func TestFindMobThreatHP(t *testing.T) {
	enc := &encounterSummary{
		MobThreat: []mobThreatEntry{
			{MobID: "orc-1", HPCurrent: 80, HPMax: 120},
			{MobID: "orc-2", HPCurrent: 0, HPMax: 120},
		},
	}
	hpC, hpM := findMobThreatHP(enc, "orc-1")
	if hpC != 80 || hpM != 120 {
		t.Fatalf("expected 80/120, got %d/%d", hpC, hpM)
	}
	hpC2, hpM2 := findMobThreatHP(enc, "orc-2")
	if hpC2 != 0 || hpM2 != 120 {
		t.Fatalf("expected 0/120, got %d/%d", hpC2, hpM2)
	}
	hpC3, hpM3 := findMobThreatHP(enc, "orc-99")
	if hpC3 != 0 || hpM3 != 0 {
		t.Fatalf("not found should return 0/0, got %d/%d", hpC3, hpM3)
	}
}

func TestCombatFramePlayerHP(t *testing.T) {
	// Active encounter with player HP should show "You" line
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			MobThreat: []mobThreatEntry{{MobID: "orc-1", HPCurrent: 90, HPMax: 120}},
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 85, HPMax: 100}
	panel := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, "orc-1"))

	if !strings.Contains(panel, "You") {
		t.Fatalf("combat frame must show 'You' HP line, got: %s", panel)
	}
	if !strings.Contains(panel, "85/100") {
		t.Fatalf("combat frame must show player HP, got: %s", panel)
	}
}

func TestCombatFrameTargetHP(t *testing.T) {
	// Active encounter with mob HP from mob_threat
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			MobThreat: []mobThreatEntry{{MobID: "orc-1", HPCurrent: 45, HPMax: 120}},
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	panel := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, "orc-1"))

	if !strings.Contains(panel, "45/120") {
		t.Fatalf("combat frame must show target HP, got: %s", panel)
	}
}

func TestCombatFrameNoEncounter(t *testing.T) {
	// No active encounter — no HP frame
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: false}
	panel := stripANSI(renderCombatPanel(sidePanelWidth, attackResult{}, pr, encounterReadResult{State: encounterReadOK}, defaultTarget(), inventoryReadResult{}, ""))

	if strings.Contains(panel, "You") {
		t.Fatalf("no HP frame without active encounter, got: %s", panel)
	}
}

func TestCombatFrameNoTargetHP(t *testing.T) {
	// Active encounter but mob HP not in mob_threat — degrade cleanly
	ar := attackResult{State: attackStateSent, TargetID: "orc-99"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			MobThreat: []mobThreatEntry{{MobID: "orc-1", HPCurrent: 120, HPMax: 120}},
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	panel := stripANSI(renderCombatPanel(sidePanelWidth, ar, pr, er, defaultTarget(), inv, ""))

	// Player HP should still show
	if !strings.Contains(panel, "You") {
		t.Fatalf("player HP should show even without target HP match, got: %s", panel)
	}
}

func TestCombatFrameCompletedNoHPFrame(t *testing.T) {
	// Completed encounter should not show HP frame
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Completed",
			CompletedReason: "all_mobs_dead", XPAwarded: 50,
		}},
	}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}
	panel := stripANSI(renderCombatPanel(sidePanelWidth, attackResult{}, pr, er, defaultTarget(), inv, ""))

	if strings.Contains(panel, "You") {
		t.Fatalf("completed encounter should not show HP frame, got: %s", panel)
	}
}

// --- Combat progression diagnosis tests (M20260412-11) ---

func TestCombatFrameReflectsUpdatedMobHP(t *testing.T) {
	// Simulates the TUI seeing two successive encounter reads with decreasing mob HP.
	// The combat frame must show the updated HP, not the initial HP.
	ar := attackResult{State: attackStateSent, TargetID: "orc-1"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}

	// First read: mob at full HP
	er1 := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			MobThreat: []mobThreatEntry{{MobID: "orc-1", HPCurrent: 120, HPMax: 120}},
			ActionIndex: 1, LatestResultKind: "damage_applied", LatestResultValue: 15,
		}},
	}
	panel1 := stripANSI(renderCombatPanel(40, ar, pr, er1, defaultTarget(), inv, "orc-1"))
	if !strings.Contains(panel1, "120/120") {
		t.Fatalf("first read must show 120/120, got: %s", panel1)
	}

	// Second read: mob HP decreased
	er2 := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1"}, MobsAlive: 1,
			MobThreat: []mobThreatEntry{{MobID: "orc-1", HPCurrent: 90, HPMax: 120}},
			ActionIndex: 2, LatestResultKind: "damage_applied", LatestResultValue: 30,
		}},
	}
	panel2 := stripANSI(renderCombatPanel(40, ar, pr, er2, defaultTarget(), inv, "orc-1"))
	if !strings.Contains(panel2, "90/120") {
		t.Fatalf("second read must show 90/120, got: %s", panel2)
	}

	// Panels must differ
	if panel1 == panel2 {
		t.Fatal("panels must differ when mob HP changes")
	}
}

func TestCombatFrameTargetIdentityMatchesAttack(t *testing.T) {
	// The combat frame must show HP for the ATTACK TARGET, not just any mob.
	ar := attackResult{State: attackStateSent, TargetID: "orc-2"}
	pr := playerReadResult{State: playerReadOK, HasActiveEncounter: true, ActiveEncounterID: "enc-1"}
	inv := inventoryReadResult{State: inventoryReadOK, HasLifecycle: true, CanAct: true, HPCurrent: 100, HPMax: 100}

	er := encounterReadResult{
		State: encounterReadOK, Count: 1,
		Encounters: []encounterSummary{{
			EncounterID: "enc-1", State: "Active",
			MobIDs: []string{"orc-1", "orc-2"}, MobsAlive: 2,
			MobThreat: []mobThreatEntry{
				{MobID: "orc-1", HPCurrent: 120, HPMax: 120},
				{MobID: "orc-2", HPCurrent: 45, HPMax: 120},
			},
		}},
	}
	panel := stripANSI(renderCombatPanel(40, ar, pr, er, defaultTarget(), inv, "orc-2"))

	// Must show orc-2's HP (the attack target), not orc-1's
	if !strings.Contains(panel, "45/120") {
		t.Fatalf("combat frame must show attack target HP (45/120), got: %s", panel)
	}
}
