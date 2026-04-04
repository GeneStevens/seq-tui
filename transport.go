package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// zoneReadState represents the outcome of a zone status read.
type zoneReadState int

const (
	zoneReadNotAttempted zoneReadState = iota
	zoneReadOK
	zoneReadFailed
)

// zoneReadResult holds the outcome of a single zone status read.
type zoneReadResult struct {
	State   zoneReadState
	Error   string         // populated on failure
	Summary zoneStatusSummary // populated on success
}

// statusLabel returns a calm, honest label for the read state.
func (r zoneReadResult) statusLabel() string {
	switch r.State {
	case zoneReadOK:
		return "zone read: ok"
	case zoneReadFailed:
		return "zone read: failed"
	default:
		return "zone read: pending"
	}
}

// zoneStatusSummary is a conservative partial decode of the zone status response.
// Only fields that are confirmed safe to display are included.
type zoneStatusSummary struct {
	ProcessName string `json:"process_name"`
	Message     string `json:"message"`
}

// zoneStatusURL builds the zone status endpoint URL from a backend target.
func zoneStatusURL(target backendTarget) string {
	base := strings.TrimRight(target.BaseURL, "/")
	url := fmt.Sprintf("%s/world/zone/%s", base, target.Zone)
	// Add mode query parameter if not default RT
	if strings.EqualFold(target.Mode, "ASYNC") {
		url += "?mode=Async"
	}
	return url
}

// fetchZoneStatus performs a single GET request to the zone status endpoint.
// Returns a zoneReadResult — never panics.
func fetchZoneStatus(target backendTarget) zoneReadResult {
	url := zoneStatusURL(target)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return zoneReadResult{
			State: zoneReadFailed,
			Error: err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return zoneReadResult{
			State: zoneReadFailed,
			Error: fmt.Sprintf("HTTP %d", resp.StatusCode),
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return zoneReadResult{
			State: zoneReadFailed,
			Error: "failed to read response body",
		}
	}

	var summary zoneStatusSummary
	if err := json.Unmarshal(body, &summary); err != nil {
		// Payload received but couldn't fully decode — still treat as OK
		return zoneReadResult{
			State:   zoneReadOK,
			Summary: zoneStatusSummary{Message: "received (partial decode)"},
		}
	}

	return zoneReadResult{
		State:   zoneReadOK,
		Summary: summary,
	}
}

// --- Mob positions ---

// mobReadState represents the outcome of a mob-position read.
type mobReadState int

const (
	mobReadNotAttempted mobReadState = iota
	mobReadOK
	mobReadFailed
)

// mobPosition is a conservative partial decode of one mob's position data.
type mobPosition struct {
	MobName  string     `json:"mob_name"`
	Position mobPosVec3 `json:"position"`
}

type mobPosVec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// mobReadResult holds the outcome of a mob-position read.
type mobReadResult struct {
	State mobReadState
	Error string
	Mobs  []mobPosition
	Count int
}

// mobStatusLabel returns a calm, honest label for the mob read state.
func (r mobReadResult) mobStatusLabel() string {
	switch r.State {
	case mobReadOK:
		return fmt.Sprintf("mobs: %d loaded", r.Count)
	case mobReadFailed:
		return "mobs: unavailable"
	default:
		return "mobs: pending"
	}
}

// zoneMobPositionsURL builds the mob-positions endpoint URL.
func zoneMobPositionsURL(target backendTarget) string {
	base := strings.TrimRight(target.BaseURL, "/")
	url := fmt.Sprintf("%s/world/zone/%s/mob_positions", base, target.Zone)
	if strings.EqualFold(target.Mode, "ASYNC") {
		url += "?mode=Async"
	}
	return url
}

// fetchMobPositions performs a single GET for mob positions.
func fetchMobPositions(target backendTarget) mobReadResult {
	url := zoneMobPositionsURL(target)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return mobReadResult{State: mobReadFailed, Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mobReadResult{State: mobReadFailed, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mobReadResult{State: mobReadFailed, Error: "failed to read response body"}
	}

	// Response shape: {"result": {"<pid>": {...mob data...}, ...}, "process_name": "...", "message": "..."}
	var raw struct {
		Result map[string]mobPosition `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return mobReadResult{State: mobReadFailed, Error: "failed to decode mob positions"}
	}

	mobs := make([]mobPosition, 0, len(raw.Result))
	for _, m := range raw.Result {
		mobs = append(mobs, m)
	}

	return mobReadResult{
		State: mobReadOK,
		Mobs:  mobs,
		Count: len(mobs),
	}
}

// --- Player join and state ---

// playerReadState represents the outcome of a player join/read flow.
type playerReadState int

const (
	playerReadNotAttempted playerReadState = iota
	playerReadOK
	playerReadFailed
)

// playerPosResult holds the player's backend-owned position.
type playerPosResult struct {
	X float64
	Y float64
}

// playerReadResult holds the outcome of a player join + state read.
type playerReadResult struct {
	State    playerReadState
	Error    string
	Position playerPosResult
	HasPos   bool // true if position was successfully decoded
}

// playerStatusLabel returns a calm, honest label.
func (r playerReadResult) playerStatusLabel() string {
	switch r.State {
	case playerReadOK:
		if r.HasPos {
			return "player: joined"
		}
		return "player: joined (no pos)"
	case playerReadFailed:
		return "player: unavailable"
	default:
		return "player: pending"
	}
}

// devJoinURL builds the dev player-join endpoint URL.
func devJoinURL(target backendTarget) string {
	base := strings.TrimRight(target.BaseURL, "/")
	return fmt.Sprintf("%s/world/dev/zone/%s/player/join", base, target.Zone)
}

// devPlayerStateURL builds the dev player-state endpoint URL.
func devPlayerStateURL(target backendTarget) string {
	base := strings.TrimRight(target.BaseURL, "/")
	return fmt.Sprintf("%s/world/dev/zone/%s/player/%s", base, target.Zone, target.Player)
}

// joinAndReadPlayer performs the dev join + player state read sequence.
func joinAndReadPlayer(target backendTarget) playerReadResult {
	client := &http.Client{Timeout: 5 * time.Second}

	// Step 1: Join player into zone
	joinBody := fmt.Sprintf(`{"player_id":"%s"}`, target.Player)
	req, err := http.NewRequest("POST", devJoinURL(target), strings.NewReader(joinBody))
	if err != nil {
		return playerReadResult{State: playerReadFailed, Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Seq-Dev-Token", target.DevToken)

	resp, err := client.Do(req)
	if err != nil {
		return playerReadResult{State: playerReadFailed, Error: err.Error()}
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return playerReadResult{State: playerReadFailed, Error: fmt.Sprintf("join HTTP %d", resp.StatusCode)}
	}

	// Step 2: Read player state
	stateReq, err := http.NewRequest("GET", devPlayerStateURL(target), nil)
	if err != nil {
		return playerReadResult{State: playerReadFailed, Error: err.Error()}
	}
	stateReq.Header.Set("X-Seq-Dev-Token", target.DevToken)

	stateResp, err := client.Do(stateReq)
	if err != nil {
		return playerReadResult{State: playerReadFailed, Error: err.Error()}
	}
	defer stateResp.Body.Close()

	if stateResp.StatusCode != http.StatusOK {
		// Join succeeded but state read failed — still report join as OK
		return playerReadResult{State: playerReadOK, HasPos: false}
	}

	body, err := io.ReadAll(stateResp.Body)
	if err != nil {
		return playerReadResult{State: playerReadOK, HasPos: false}
	}

	// Conservative partial decode of player state
	var raw struct {
		Result struct {
			Position struct {
				Pos struct {
					X float64 `json:"X"`
					Y float64 `json:"Y"`
					Z float64 `json:"Z"`
				} `json:"Pos"`
			} `json:"Position"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return playerReadResult{State: playerReadOK, HasPos: false}
	}

	return playerReadResult{
		State: playerReadOK,
		Position: playerPosResult{
			X: raw.Result.Position.Pos.X,
			Y: raw.Result.Position.Pos.Y,
		},
		HasPos: true,
	}
}
