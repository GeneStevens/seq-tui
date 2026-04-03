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
	directionalPhrases := []string{"north", "east", "south"}
	for _, phrase := range directionalPhrases {
		if !strings.Contains(panel, phrase) {
			t.Fatalf("nearby panel should contain directional cue %q", phrase)
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
	footer := renderFooter(80)
	if !strings.Contains(footer, "quit") {
		t.Fatal("footer should contain quit hint")
	}
}

func TestRenderMapPanelContainsPlayerMarker(t *testing.T) {
	panel := renderMapPanel()
	if !strings.ContainsRune(panel, playerMarker) {
		t.Fatal("map panel should contain player marker")
	}
}

func TestRenderLayoutContainsAllSections(t *testing.T) {
	layout := renderLayout(80, 40)
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
	layout := renderLayout(80, 40)
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
	panel := renderStatusPanel(sidePanelWidth)
	if !strings.Contains(panel, statusTitle) {
		t.Fatal("status panel should contain title")
	}
}

func TestRenderSideColumnContainsBothSections(t *testing.T) {
	col := renderSideColumn(sidePanelWidth)
	if !strings.Contains(col, nearbyTitle) {
		t.Fatal("side column should contain nearby title")
	}
	if !strings.Contains(col, statusTitle) {
		t.Fatal("side column should contain status title")
	}
}

func TestWideLayoutContainsPanels(t *testing.T) {
	layout := renderLayout(120, 40)
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
	layout := renderLayout(50, 30)
	if strings.Contains(layout, nearbyTitle) {
		t.Fatal("narrow layout should not contain nearby panel")
	}
	if !strings.ContainsRune(layout, playerMarker) {
		t.Fatal("narrow layout should still contain player marker")
	}
}

func TestRenderLayoutSmallTerminal(t *testing.T) {
	// Should not panic with very small dimensions
	layout := renderLayout(20, 5)
	if len(layout) == 0 {
		t.Fatal("layout should not be empty even for small terminal")
	}
}

func TestRenderLayoutVariousSizes(t *testing.T) {
	sizes := [][2]int{{40, 20}, {80, 40}, {120, 50}, {200, 60}}
	for _, sz := range sizes {
		layout := renderLayout(sz[0], sz[1])
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

func TestViewEmptyBeforeResize(t *testing.T) {
	m := model{}
	if m.View() != "" {
		t.Fatal("View() should be empty before receiving window size")
	}
}

func TestViewNonEmptyAfterResize(t *testing.T) {
	m := model{width: 80, height: 40}
	view := m.View()
	if len(view) == 0 {
		t.Fatal("View() should not be empty after resize")
	}
}
