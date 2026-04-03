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
