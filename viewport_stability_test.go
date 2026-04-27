package main

import (
	"math"
	"strings"
	"testing"
)

// --- Viewport stability tests ---
//
// These tests validate that sub-cell floating-point drift in player position
// does not cause the viewport grid to shift, preventing mob render jitter.

// testGeometry returns a simple box of map lines and its full bounds,
// suitable for viewport stability testing.
func testGeometry() ([]mapLine, mapBounds) {
	lines := []mapLine{
		{From: mapVec3{X: 0, Z: 0}, To: mapVec3{X: 100, Z: 0}},
		{From: mapVec3{X: 100, Z: 0}, To: mapVec3{X: 100, Z: 100}},
		{From: mapVec3{X: 100, Z: 100}, To: mapVec3{X: 0, Z: 100}},
		{From: mapVec3{X: 0, Z: 100}, To: mapVec3{X: 0, Z: 0}},
	}
	bounds := computeBounds(lines)
	return lines, bounds
}

// TestSnapViewportCenter_AlignedUnchanged verifies that already-aligned values
// pass through snapViewportCenter without modification.
func TestSnapViewportCenter_AlignedUnchanged(t *testing.T) {
	spanX := 50.0
	spanZ := 50.0
	vpW, vpH := 40, 20

	// Cell size: spanX/(vpW-1) = 50/39 ~ 1.282
	cellX := spanX / float64(vpW-1)
	// Pick a center that is exactly on a grid point
	alignedX := math.Round(25.0/cellX) * cellX
	cellZ := spanZ / float64(vpH-1)
	alignedZ := math.Round(25.0/cellZ) * cellZ

	gotX, gotZ := snapViewportCenter(alignedX, alignedZ, spanX, spanZ, vpW, vpH)
	if gotX != alignedX {
		t.Errorf("aligned X changed: got %v, want %v", gotX, alignedX)
	}
	if gotZ != alignedZ {
		t.Errorf("aligned Z changed: got %v, want %v", gotZ, alignedZ)
	}
}

// TestSnapViewportCenter_SmallOffsetSnaps verifies that sub-cell offsets
// snap to the nearest grid point.
func TestSnapViewportCenter_SmallOffsetSnaps(t *testing.T) {
	spanX := 50.0
	spanZ := 50.0
	vpW, vpH := 40, 20

	baseX, baseZ := 25.0, 25.0
	driftX, driftZ := 25.0001, 25.0001

	snappedBaseX, snappedBaseZ := snapViewportCenter(baseX, baseZ, spanX, spanZ, vpW, vpH)
	snappedDriftX, snappedDriftZ := snapViewportCenter(driftX, driftZ, spanX, spanZ, vpW, vpH)

	if snappedBaseX != snappedDriftX {
		t.Errorf("X not stable: base snapped to %v, drift snapped to %v", snappedBaseX, snappedDriftX)
	}
	if snappedBaseZ != snappedDriftZ {
		t.Errorf("Z not stable: base snapped to %v, drift snapped to %v", snappedBaseZ, snappedDriftZ)
	}
}

// TestSnapViewportCenter_EdgeValues verifies snapping at boundary values (0 and max).
func TestSnapViewportCenter_EdgeValues(t *testing.T) {
	spanX := 100.0
	spanZ := 100.0
	vpW, vpH := 50, 25

	// Near zero
	gotX, gotZ := snapViewportCenter(0.0001, 0.0001, spanX, spanZ, vpW, vpH)
	if gotX != 0.0 {
		t.Errorf("near-zero X not snapped to 0: got %v", gotX)
	}
	if gotZ != 0.0 {
		t.Errorf("near-zero Z not snapped to 0: got %v", gotZ)
	}

	// Near max
	gotX, gotZ = snapViewportCenter(99.9999, 99.9999, spanX, spanZ, vpW, vpH)
	if gotX != 100.0 {
		t.Errorf("near-max X not snapped to 100: got %v", gotX)
	}
	if gotZ != 100.0 {
		t.Errorf("near-max Z not snapped to 100: got %v", gotZ)
	}
}

// TestSnapViewportCenter_DegenerateViewport verifies that a 1-cell viewport
// does not divide by zero.
func TestSnapViewportCenter_DegenerateViewport(t *testing.T) {
	x, z := snapViewportCenter(50.0, 50.0, 100.0, 100.0, 1, 1)
	if math.IsNaN(x) || math.IsNaN(z) || math.IsInf(x, 0) || math.IsInf(z, 0) {
		t.Errorf("degenerate viewport produced NaN/Inf: x=%v z=%v", x, z)
	}
}

// TestRenderIdempotent verifies that identical inputs to rasterizeAdaptiveViewport
// plus overlayMobs produce identical ASCII output across multiple calls.
func TestRenderIdempotent(t *testing.T) {
	lines, bounds := testGeometry()
	vpW, vpH := 40, 20
	centerX, centerZ := 50.0, 50.0

	mobs := []mobPosition{
		{MobID: "mob-1", Position: mobPosVec3{X: 30, Y: 40}},
		{MobID: "mob-2", Position: mobPosVec3{X: 70, Y: 60}},
	}

	var outputs [3]string
	for i := range outputs {
		ascii, vpBounds := rasterizeAdaptiveViewport(lines, bounds, centerX, centerZ, vpW, vpH)
		outputs[i] = overlayMobs(ascii, mobs, vpBounds, vpW, vpH)
	}

	for i := 1; i < len(outputs); i++ {
		if outputs[i] != outputs[0] {
			t.Errorf("render call %d differs from call 0", i)
		}
	}
}

// TestJitterImmunity verifies that tiny floating-point drift in player position
// does not change where a mob projects on the viewport.
func TestJitterImmunity(t *testing.T) {
	lines, bounds := testGeometry()
	vpW, vpH := 40, 20

	mobPos := mobPosVec3{X: 60, Y: 70}
	mobs := []mobPosition{{MobID: "mob-1", Position: mobPos}}

	// Two player positions with sub-cell drift
	centerA_X, centerA_Z := 50.0, 50.0
	centerB_X, centerB_Z := 50.0001, 50.0001

	asciiA, boundsA := rasterizeAdaptiveViewport(lines, bounds, centerA_X, centerA_Z, vpW, vpH)
	asciiB, boundsB := rasterizeAdaptiveViewport(lines, bounds, centerB_X, centerB_Z, vpW, vpH)

	overlayA := overlayMobs(asciiA, mobs, boundsA, vpW, vpH)
	overlayB := overlayMobs(asciiB, mobs, boundsB, vpW, vpH)

	// Find mob cell in each output
	colA, rowA := findGlyph(overlayA, 'm')
	colB, rowB := findGlyph(overlayB, 'm')

	if colA != colB || rowA != rowB {
		t.Errorf("mob position shifted: (%d,%d) vs (%d,%d)", colA, rowA, colB, rowB)
	}
}

// TestMobVisibility verifies that mobs within the viewport bounds always render
// and do not disappear due to clipping edge effects.
func TestMobVisibility(t *testing.T) {
	lines, bounds := testGeometry()
	vpW, vpH := 40, 20
	centerX, centerZ := 50.0, 50.0

	// Place mobs at various positions within the viewport region
	mobs := []mobPosition{
		{MobID: "center", Position: mobPosVec3{X: 50, Y: 50}},
		{MobID: "near-edge", Position: mobPosVec3{X: 45, Y: 45}},
	}

	ascii, vpBounds := rasterizeAdaptiveViewport(lines, bounds, centerX, centerZ, vpW, vpH)
	overlay := overlayMobs(ascii, mobs, vpBounds, vpW, vpH)

	// Both mobs should project within viewport and be visible
	mobCount := 0
	for _, ch := range overlay {
		if ch == 'm' {
			mobCount++
		}
	}

	// Two mobs at distinct positions should produce 2 'm' markers (or a '2' if same cell)
	hasTwoMs := mobCount == 2
	hasDigit := strings.ContainsRune(overlay, '2')

	if !hasTwoMs && !hasDigit {
		t.Errorf("expected 2 visible mobs: got %d 'm' markers and no '2' digit", mobCount)
	}
}

// TestJitterImmunity_LargerDrift verifies stability with slightly larger drift values
// that are still sub-cell.
func TestJitterImmunity_LargerDrift(t *testing.T) {
	lines, bounds := testGeometry()
	vpW, vpH := 40, 20

	mobs := []mobPosition{
		{MobID: "mob-1", Position: mobPosVec3{X: 30, Y: 40}},
		{MobID: "mob-2", Position: mobPosVec3{X: 70, Y: 60}},
	}

	drifts := [][2]float64{
		{50.0, 50.0},
		{50.001, 50.001},
		{49.999, 49.999},
		{50.0005, 49.9995},
	}

	var baseline string
	for i, d := range drifts {
		ascii, vpBounds := rasterizeAdaptiveViewport(lines, bounds, d[0], d[1], vpW, vpH)
		overlay := overlayMobs(ascii, mobs, vpBounds, vpW, vpH)
		if i == 0 {
			baseline = overlay
		} else if overlay != baseline {
			t.Errorf("drift case %d (center %.4f,%.4f) produced different output", i, d[0], d[1])
		}
	}
}

// findGlyph returns the (col, row) of the first occurrence of the given glyph
// in an ASCII map string. Returns (-1, -1) if not found.
func findGlyph(ascii string, glyph rune) (col, row int) {
	for r, line := range strings.Split(ascii, "\n") {
		for c, ch := range line {
			if ch == glyph {
				return c, r
			}
		}
	}
	return -1, -1
}
