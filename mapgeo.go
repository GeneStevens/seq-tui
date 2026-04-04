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

// mapReadResult holds the outcome of a map geometry read.
type mapReadResult struct {
	State    mapReadState
	Error    string
	MapText  string // projected ASCII map (populated on success)
	MapWidth int
	MapHeight int
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

	// Project and rasterize
	width, height := 60, 30
	ascii := projectAndRasterize(mapResp.Result.Lines, width, height)

	return mapReadResult{
		State:     mapReadOK,
		MapText:   ascii,
		MapWidth:  width,
		MapHeight: height,
	}
}

// --- Projection and rasterization ---

// projectAndRasterize converts 3D line segments into a 2D ASCII map.
// Uses top-down projection (X, Z plane; Y/elevation ignored).
// Normalizes coordinates into a fixed-size ASCII canvas.
func projectAndRasterize(lines []mapLine, width, height int) string {
	if len(lines) == 0 || width < 1 || height < 1 {
		return ""
	}

	// Find bounding box in X,Z plane
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
		// Project 3D to 2D canvas coordinates
		// X maps to column, Z maps to row (inverted so north is up)
		c0 := int(math.Round(float64(width-1) * (l.From.X - minX) / spanX))
		r0 := int(math.Round(float64(height-1) * (1.0 - (l.From.Z-minZ)/spanZ)))
		c1 := int(math.Round(float64(width-1) * (l.To.X - minX) / spanX))
		r1 := int(math.Round(float64(height-1) * (1.0 - (l.To.Z-minZ)/spanZ)))

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
	return sb.String()
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
