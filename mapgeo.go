package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

// mapReadState represents the outcome of a map geometry read.
type mapReadState int

const (
	mapReadNotAttempted mapReadState = iota
	mapReadOK
	mapReadFailed
)

// mapBounds holds the bounding box used for projection.
// Shared between map rasterization and mob position projection.
type mapBounds struct {
	MinX, MaxX float64
	MinZ, MaxZ float64
	SpanX, SpanZ float64
}

// projectToCell projects a world coordinate (x maps to col, z maps to row)
// into canvas cell coordinates using the shared bounding box.
func (b mapBounds) projectToCell(worldX, worldZ float64, width, height int) (col, row int) {
	col = int(math.Round(float64(width-1) * (worldX - b.MinX) / b.SpanX))
	row = int(math.Round(float64(height-1) * (1.0 - (worldZ-b.MinZ)/b.SpanZ)))
	return col, row
}

// mapReadResult holds the outcome of a map geometry read.
type mapReadResult struct {
	State     mapReadState
	Error     string
	MapText   string // projected ASCII map (populated on success)
	MapWidth  int
	MapHeight int
	Bounds    mapBounds // shared projection basis
	Lines     []mapLine // raw geometry for adaptive re-rasterization
}

// mapStatusLabel returns a calm, honest label for the map read state.
func (r mapReadResult) mapStatusLabel() string {
	switch r.State {
	case mapReadOK:
		return "map: loaded"
	case mapReadFailed:
		return "map: unavailable"
	default:
		return "map: pending"
	}
}

// --- Geometry decode types (conservative partial decode) ---

type mapResponse struct {
	Result mapResult `json:"result"`
}

type mapResult struct {
	Lines []mapLine `json:"Lines"`
}

type mapLine struct {
	From mapVec3 `json:"From"`
	To   mapVec3 `json:"To"`
}

type mapVec3 struct {
	X float64 `json:"X"`
	Y float64 `json:"Y"`
	Z float64 `json:"Z"`
}

// --- Map endpoint URL ---

// zoneMapURL builds the zone map endpoint URL from a backend target.
func zoneMapURL(target backendTarget) string {
	base := strings.TrimRight(target.BaseURL, "/")
	url := fmt.Sprintf("%s/world/zone/%s/map", base, target.Zone)
	if strings.EqualFold(target.Mode, "ASYNC") {
		url += "?mode=Async"
	}
	return url
}

// --- Map fetch ---

// fetchZoneMap performs a single GET request to the zone map endpoint.
func fetchZoneMap(target backendTarget) mapReadResult {
	url := zoneMapURL(target)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return mapReadResult{State: mapReadFailed, Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mapReadResult{State: mapReadFailed, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mapReadResult{State: mapReadFailed, Error: "failed to read response body"}
	}

	var mapResp mapResponse
	if err := json.Unmarshal(body, &mapResp); err != nil {
		return mapReadResult{State: mapReadFailed, Error: "failed to decode map geometry"}
	}

	if len(mapResp.Result.Lines) == 0 {
		return mapReadResult{State: mapReadFailed, Error: "map contains no geometry"}
	}

	// Project and rasterize at high internal resolution for viewport cropping.
	// The full zone is rasterized once; a viewport is extracted at render time.
	width, height := 200, 100
	ascii, bounds := projectAndRasterize(mapResp.Result.Lines, width, height)

	return mapReadResult{
		State:     mapReadOK,
		MapText:   ascii,
		MapWidth:  width,
		MapHeight: height,
		Bounds:    bounds,
		Lines:     mapResp.Result.Lines,
	}
}

// --- Projection and rasterization ---

// computeBounds calculates the bounding box for map line geometry.
func computeBounds(lines []mapLine) mapBounds {
	minX, maxX := lines[0].From.X, lines[0].From.X
	minZ, maxZ := lines[0].From.Z, lines[0].From.Z
	for _, l := range lines {
		for _, p := range [2]mapVec3{l.From, l.To} {
			if p.X < minX { minX = p.X }
			if p.X > maxX { maxX = p.X }
			if p.Z < minZ { minZ = p.Z }
			if p.Z > maxZ { maxZ = p.Z }
		}
	}
	spanX := maxX - minX
	spanZ := maxZ - minZ
	if spanX == 0 { spanX = 1 }
	if spanZ == 0 { spanZ = 1 }
	return mapBounds{MinX: minX, MaxX: maxX, MinZ: minZ, MaxZ: maxZ, SpanX: spanX, SpanZ: spanZ}
}

// projectAndRasterize converts 3D line segments into a 2D ASCII map.
// Uses top-down projection (X, Z plane; Y/elevation ignored).
// Returns the ASCII string and the bounding box used for projection.
func projectAndRasterize(lines []mapLine, width, height int) (string, mapBounds) {
	if len(lines) == 0 || width < 1 || height < 1 {
		return "", mapBounds{}
	}

	bounds := computeBounds(lines)

	// Initialize canvas with empty space
	canvas := make([][]rune, height)
	for r := range canvas {
		canvas[r] = make([]rune, width)
		for c := range canvas[r] {
			canvas[r][c] = ' '
		}
	}

	// Rasterize each line segment using Bresenham's algorithm
	for _, l := range lines {
		c0, r0 := bounds.projectToCell(l.From.X, l.From.Z, width, height)
		c1, r1 := bounds.projectToCell(l.To.X, l.To.Z, width, height)
		rasterizeLine(canvas, r0, c0, r1, c1, width, height)
	}

	// Convert canvas to string
	var sb strings.Builder
	for r, row := range canvas {
		if r > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(string(row))
	}
	return sb.String(), bounds
}

// rasterizeLine draws a line on the canvas using Bresenham's algorithm.
func rasterizeLine(canvas [][]rune, r0, c0, r1, c1, width, height int) {
	dr := abs(r1 - r0)
	dc := abs(c1 - c0)
	sr := 1
	if r0 > r1 { sr = -1 }
	sc := 1
	if c0 > c1 { sc = -1 }
	err := dc - dr

	for {
		if r0 >= 0 && r0 < height && c0 >= 0 && c0 < width {
			canvas[r0][c0] = '#'
		}
		if r0 == r1 && c0 == c1 {
			break
		}
		e2 := 2 * err
		if e2 > -dr {
			err -= dr
			c0 += sc
		}
		if e2 < dc {
			err += dc
			r0 += sr
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// --- Mob overlay ---

// overlayPlayer places the player marker onto an ASCII map string using shared projection bounds.
// Player position uses x,y as ground plane (same as mobs).
func overlayPlayer(mapText string, pos playerPosResult, bounds mapBounds, width, height int) string {
	col, row := bounds.projectToCell(pos.X, pos.Y, width, height)
	lines := strings.Split(mapText, "\n")
	if row >= 0 && row < len(lines) {
		runes := []rune(lines[row])
		for len(runes) < width {
			runes = append(runes, ' ')
		}
		if col >= 0 && col < len(runes) {
			runes[col] = '@'
			lines[row] = string(runes)
		}
	}
	return strings.Join(lines, "\n")
}

// overlayFocusedMob replaces the mob marker at the focused mob's position with 'M'.
// Purely visual, non-authoritative. If focusedMobID is empty or not found among mobs,
// the map is returned unchanged.
func overlayFocusedMob(mapText string, mobs []mobPosition, focusedMobID string, bounds mapBounds, width, height int) string {
	if focusedMobID == "" || len(mobs) == 0 || width < 1 || height < 1 {
		return mapText
	}
	for _, mob := range mobs {
		if mob.ProcessID == focusedMobID {
			col, row := bounds.projectToCell(mob.Position.X, mob.Position.Y, width, height)
			lines := strings.Split(mapText, "\n")
			if row >= 0 && row < len(lines) {
				runes := []rune(lines[row])
				for len(runes) < width {
					runes = append(runes, ' ')
				}
				if col >= 0 && col < len(runes) {
					runes[col] = 'M'
					lines[row] = string(runes)
				}
			}
			return strings.Join(lines, "\n")
		}
	}
	return mapText
}

// overlayAttackTarget replaces the mob marker at the attack target's position with 'X'.
// Purely visual, non-authoritative. Shows which mob the player last attacked.
// If targetID is empty or not found among mobs, the map is returned unchanged.
func overlayAttackTarget(mapText string, mobs []mobPosition, targetID string, bounds mapBounds, width, height int) string {
	if targetID == "" || len(mobs) == 0 || width < 1 || height < 1 {
		return mapText
	}
	for _, mob := range mobs {
		if mob.ProcessID == targetID {
			col, row := bounds.projectToCell(mob.Position.X, mob.Position.Y, width, height)
			lines := strings.Split(mapText, "\n")
			if row >= 0 && row < len(lines) {
				runes := []rune(lines[row])
				for len(runes) < width {
					runes = append(runes, ' ')
				}
				if col >= 0 && col < len(runes) {
					runes[col] = 'X'
					lines[row] = string(runes)
				}
			}
			return strings.Join(lines, "\n")
		}
	}
	return mapText
}

// overlayFocusedPlayer replaces the player marker '@' with '&' at the player's position.
// Purely visual, non-authoritative.
func overlayFocusedPlayer(mapText string, pos playerPosResult, bounds mapBounds, width, height int) string {
	col, row := bounds.projectToCell(pos.X, pos.Y, width, height)
	lines := strings.Split(mapText, "\n")
	if row >= 0 && row < len(lines) {
		runes := []rune(lines[row])
		for len(runes) < width {
			runes = append(runes, ' ')
		}
		if col >= 0 && col < len(runes) {
			runes[col] = '&'
			lines[row] = string(runes)
		}
	}
	return strings.Join(lines, "\n")
}

// extractViewport extracts a viewport-sized sub-grid from a rasterized map,
// centered on the given cell position with deterministic edge clamping.
// Returns the full map unchanged if viewport dimensions exceed the map.
func extractViewport(mapText string, mapWidth, mapHeight, centerCol, centerRow, vpWidth, vpHeight int) string {
	if vpWidth >= mapWidth && vpHeight >= mapHeight {
		return mapText
	}

	// Compute viewport origin (top-left), centered on target cell
	left := centerCol - vpWidth/2
	top := centerRow - vpHeight/2

	// Clamp to map edges
	if left < 0 {
		left = 0
	}
	if top < 0 {
		top = 0
	}
	if left+vpWidth > mapWidth {
		left = mapWidth - vpWidth
	}
	if top+vpHeight > mapHeight {
		top = mapHeight - vpHeight
	}
	if left < 0 {
		left = 0
	}
	if top < 0 {
		top = 0
	}

	// Actual extraction dimensions (when viewport > map)
	ew := vpWidth
	if ew > mapWidth {
		ew = mapWidth
	}
	eh := vpHeight
	if eh > mapHeight {
		eh = mapHeight
	}

	lines := strings.Split(mapText, "\n")
	var sb strings.Builder
	for r := top; r < top+eh && r < len(lines); r++ {
		if r > top {
			sb.WriteByte('\n')
		}
		runes := []rune(lines[r])
		end := left + ew
		if end > len(runes) {
			end = len(runes)
		}
		start := left
		if start > len(runes) {
			start = len(runes)
		}
		sb.WriteString(string(runes[start:end]))
	}
	return sb.String()
}

// --- Adaptive viewport rasterization ---

// Adaptive scaling reference dimensions. At these viewport dimensions, the full zone
// is visible. Smaller viewports show a proportionally tighter local region.
// Using values larger than the old 200x100 internal canvas means smaller viewports
// get a tighter view than the old fixed-crop approach.
const (
	adaptiveRefWidth  = 240.0
	adaptiveRefHeight = 120.0
)

// computeAdaptiveWorldWindow computes a world-space bounding box for adaptive viewport
// rendering. The world span scales linearly with viewport dimensions relative to
// reference dimensions, producing tighter local views for smaller viewports.
// Centered on (centerX, centerZ) with deterministic edge clamping.
func computeAdaptiveWorldWindow(fullBounds mapBounds, centerX, centerZ float64, vpWidth, vpHeight int) mapBounds {
	fractionX := float64(vpWidth) / adaptiveRefWidth
	fractionZ := float64(vpHeight) / adaptiveRefHeight
	if fractionX > 1.0 {
		fractionX = 1.0
	}
	if fractionZ > 1.0 {
		fractionZ = 1.0
	}
	if fractionX < 0.05 {
		fractionX = 0.05
	}
	if fractionZ < 0.05 {
		fractionZ = 0.05
	}

	spanX := fullBounds.SpanX * fractionX
	spanZ := fullBounds.SpanZ * fractionZ

	minX := centerX - spanX/2
	maxX := centerX + spanX/2
	minZ := centerZ - spanZ/2
	maxZ := centerZ + spanZ/2

	// Clamp to zone bounds
	if minX < fullBounds.MinX {
		shift := fullBounds.MinX - minX
		minX = fullBounds.MinX
		maxX += shift
	}
	if maxX > fullBounds.MaxX {
		shift := maxX - fullBounds.MaxX
		maxX = fullBounds.MaxX
		minX -= shift
	}
	if minZ < fullBounds.MinZ {
		shift := fullBounds.MinZ - minZ
		minZ = fullBounds.MinZ
		maxZ += shift
	}
	if maxZ > fullBounds.MaxZ {
		shift := maxZ - fullBounds.MaxZ
		maxZ = fullBounds.MaxZ
		minZ -= shift
	}
	// Final clamp for edge cases where span exceeds zone
	if minX < fullBounds.MinX {
		minX = fullBounds.MinX
	}
	if minZ < fullBounds.MinZ {
		minZ = fullBounds.MinZ
	}

	return mapBounds{
		MinX:  minX,
		MaxX:  maxX,
		MinZ:  minZ,
		MaxZ:  maxZ,
		SpanX: maxX - minX,
		SpanZ: maxZ - minZ,
	}
}

// rasterizeAdaptiveViewport rasterizes zone geometry into a viewport-sized canvas
// using adaptive world bounds. Returns the ASCII map and the viewport-local bounds
// for use by overlay functions. The viewport is the final render size — no further
// cropping is needed.
func rasterizeAdaptiveViewport(lines []mapLine, fullBounds mapBounds, centerX, centerZ float64, vpWidth, vpHeight int) (string, mapBounds) {
	if len(lines) == 0 || vpWidth < 1 || vpHeight < 1 {
		return "", mapBounds{}
	}

	vpBounds := computeAdaptiveWorldWindow(fullBounds, centerX, centerZ, vpWidth, vpHeight)

	// Initialize canvas
	canvas := make([][]rune, vpHeight)
	for r := range canvas {
		canvas[r] = make([]rune, vpWidth)
		for c := range canvas[r] {
			canvas[r][c] = ' '
		}
	}

	// Rasterize each line segment using the viewport-local bounds
	for _, l := range lines {
		c0, r0 := vpBounds.projectToCell(l.From.X, l.From.Z, vpWidth, vpHeight)
		c1, r1 := vpBounds.projectToCell(l.To.X, l.To.Z, vpWidth, vpHeight)
		rasterizeLine(canvas, r0, c0, r1, c1, vpWidth, vpHeight)
	}

	// Convert canvas to string
	var sb strings.Builder
	for r, row := range canvas {
		if r > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(string(row))
	}
	return sb.String(), vpBounds
}

// overlayMobs places mob markers onto an ASCII map string using shared projection bounds.
// Mob positions use x,y as ground plane (mob.x → map.X, mob.y → map.Z).
func overlayMobs(mapText string, mobs []mobPosition, bounds mapBounds, width, height int) string {
	if len(mobs) == 0 || width < 1 || height < 1 {
		return mapText
	}

	lines := strings.Split(mapText, "\n")
	// Ensure we have enough lines
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	for _, mob := range mobs {
		// mob.Position.X → map X (column), mob.Position.Y → map Z (row)
		col, row := bounds.projectToCell(mob.Position.X, mob.Position.Y, width, height)
		if row >= 0 && row < len(lines) {
			runes := []rune(lines[row])
			// Pad if needed
			for len(runes) < width {
				runes = append(runes, ' ')
			}
			if col >= 0 && col < len(runes) {
				runes[col] = 'm'
				lines[row] = string(runes)
			}
		}
	}

	return strings.Join(lines, "\n")
}
