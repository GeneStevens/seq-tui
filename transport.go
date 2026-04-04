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
	State               playerReadState
	Error               string
	Position            playerPosResult
	HasPos              bool   // true if position was successfully decoded
	ActiveEncounterID   string // backend-owned encounter ID, empty if none
	HasActiveEncounter  bool   // true if player is in an encounter
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

// devPlayerPositionURL builds the dev player-position endpoint URL.
func devPlayerPositionURL(target backendTarget) string {
	base := strings.TrimRight(target.BaseURL, "/")
	return fmt.Sprintf("%s/world/dev/zone/%s/player/position", base, target.Zone)
}

// moveResult holds the outcome of a movement submission + readback.
type moveResult struct {
	OK       bool
	Error    string
	Position playerPosResult
	HasPos   bool
}

// submitMoveAndReadback submits a position change and reads back the result.
// Uses the dev player/position endpoint (direct position set).
func submitMoveAndReadback(target backendTarget, currentPos playerPosResult, dx, dy float64) moveResult {
	client := &http.Client{Timeout: 5 * time.Second}

	// Submit position change
	newX := currentPos.X + dx
	newY := currentPos.Y + dy
	payload := fmt.Sprintf(`{"player_id":"%s","x":%f,"y":%f,"z":0}`, target.Player, newX, newY)

	req, err := http.NewRequest("POST", devPlayerPositionURL(target), strings.NewReader(payload))
	if err != nil {
		return moveResult{OK: false, Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Seq-Dev-Token", target.DevToken)

	resp, err := client.Do(req)
	if err != nil {
		return moveResult{OK: false, Error: err.Error()}
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return moveResult{OK: false, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	// Readback player state
	stateReq, err := http.NewRequest("GET", devPlayerStateURL(target), nil)
	if err != nil {
		return moveResult{OK: true, HasPos: false}
	}
	stateReq.Header.Set("X-Seq-Dev-Token", target.DevToken)

	stateResp, err := client.Do(stateReq)
	if err != nil {
		return moveResult{OK: true, HasPos: false}
	}
	defer stateResp.Body.Close()

	if stateResp.StatusCode != http.StatusOK {
		return moveResult{OK: true, HasPos: false}
	}

	body, err := io.ReadAll(stateResp.Body)
	if err != nil {
		return moveResult{OK: true, HasPos: false}
	}

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
		return moveResult{OK: true, HasPos: false}
	}

	return moveResult{
		OK: true,
		Position: playerPosResult{
			X: raw.Result.Position.Pos.X,
			Y: raw.Result.Position.Pos.Y,
		},
		HasPos: true,
	}
}

// decodePlayerState is a shared helper for conservative partial decode of
// the dev player state response. It extracts position and active encounter ID.
func decodePlayerState(body []byte, baseState playerReadState) playerReadResult {
	var raw struct {
		Result struct {
			Player struct {
				Position struct {
					X float64 `json:"x"`
					Y float64 `json:"y"`
					Z float64 `json:"z"`
				} `json:"position"`
				ActiveEncounterID string `json:"active_encounter_id"`
			} `json:"player"`
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
		return playerReadResult{State: baseState, HasPos: false}
	}

	// Use player.position (new shape) if available, fall back to Position.Pos (legacy shape)
	posX := raw.Result.Player.Position.X
	posY := raw.Result.Player.Position.Y
	if posX == 0 && posY == 0 {
		posX = raw.Result.Position.Pos.X
		posY = raw.Result.Position.Pos.Y
	}

	encID := raw.Result.Player.ActiveEncounterID

	return playerReadResult{
		State: baseState,
		Position: playerPosResult{
			X: posX,
			Y: posY,
		},
		HasPos:             true,
		ActiveEncounterID:  encID,
		HasActiveEncounter: encID != "",
	}
}

// readPlayerState reads player state without joining (for refresh cycles).
func readPlayerState(target backendTarget) playerReadResult {
	client := &http.Client{Timeout: 3 * time.Second}

	req, err := http.NewRequest("GET", devPlayerStateURL(target), nil)
	if err != nil {
		return playerReadResult{State: playerReadFailed, Error: err.Error()}
	}
	req.Header.Set("X-Seq-Dev-Token", target.DevToken)

	resp, err := client.Do(req)
	if err != nil {
		return playerReadResult{State: playerReadFailed, Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return playerReadResult{State: playerReadFailed, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return playerReadResult{State: playerReadOK, HasPos: false}
	}

	return decodePlayerState(body, playerReadOK)
}

// --- Encounter read ---

// encounterReadState represents the outcome of an encounter read.
type encounterReadState int

const (
	encounterReadNotAttempted encounterReadState = iota
	encounterReadOK
	encounterReadFailed
)

// encounterSummary is a conservative partial decode of one encounter summary.
// Only backend-owned facts needed for display are decoded.
type encounterSummary struct {
	EncounterID     string   `json:"encounter_id"`
	State           string   `json:"state"`
	CompletedReason string   `json:"completed_reason"`
	PlayerIDs       []string // backend-owned participant IDs for roster display
	MobIDs          []string // backend-owned mob IDs for roster display
	PlayerCount     int      // derived from len(PlayerIDs)
	MobCount        int      // derived from len(MobIDs)
	MobsAlive       int      `json:"mobs_alive_count"`
	MobsDead        int      `json:"mobs_dead_count"`
	ActionIndex     uint64   `json:"action_index"`
	TimelineLength  int      `json:"timeline_length"`
}

// encounterReadResult holds the outcome of a zone encounter read.
type encounterReadResult struct {
	State      encounterReadState
	Error      string
	Encounters []encounterSummary
	Count      int
}

// encounterStatusLabel returns an honest label for the encounter read state.
func (r encounterReadResult) encounterStatusLabel() string {
	switch r.State {
	case encounterReadOK:
		return fmt.Sprintf("encounters: %d", r.Count)
	case encounterReadFailed:
		return "encounters: unavailable"
	default:
		return "encounters: pending"
	}
}

// zoneEncountersURL builds the zone encounters call URL.
func zoneEncountersURL(target backendTarget) string {
	base := strings.TrimRight(target.BaseURL, "/")
	url := fmt.Sprintf("%s/world/call/%s?message=encounters", base, target.Zone)
	if strings.EqualFold(target.Mode, "ASYNC") {
		url += "&mode=Async"
	}
	return url
}

// fetchZoneEncounters performs a single GET to read encounter summaries.
func fetchZoneEncounters(target backendTarget) encounterReadResult {
	url := zoneEncountersURL(target)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return encounterReadResult{State: encounterReadFailed, Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return encounterReadResult{State: encounterReadFailed, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return encounterReadResult{State: encounterReadFailed, Error: "failed to read response body"}
	}

	// Response shape: {"result": [...encounter summaries...], "process_name": "...", "message": "encounters"}
	var envelope struct {
		Result []struct {
			EncounterID     string   `json:"encounter_id"`
			State           string   `json:"state"`
			CompletedReason string   `json:"completed_reason"`
			PlayerIDs       []string `json:"player_ids"`
			MobIDs          []string `json:"mob_ids"`
			MobsAliveCount  int      `json:"mobs_alive_count"`
			MobsDeadCount   int      `json:"mobs_dead_count"`
			ActionIndex     uint64   `json:"action_index"`
			TimelineLength  int      `json:"timeline_length"`
		} `json:"result"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		// Payload received but couldn't decode
		return encounterReadResult{
			State: encounterReadOK,
			Count: 0,
		}
	}

	if envelope.Error != "" {
		return encounterReadResult{State: encounterReadFailed, Error: envelope.Error}
	}

	summaries := make([]encounterSummary, 0, len(envelope.Result))
	for _, e := range envelope.Result {
		summaries = append(summaries, encounterSummary{
			EncounterID:     e.EncounterID,
			State:           e.State,
			CompletedReason: e.CompletedReason,
			PlayerIDs:       e.PlayerIDs,
			MobIDs:          e.MobIDs,
			PlayerCount:     len(e.PlayerIDs),
			MobCount:        len(e.MobIDs),
			MobsAlive:       e.MobsAliveCount,
			MobsDead:        e.MobsDeadCount,
			ActionIndex:     e.ActionIndex,
			TimelineLength:  e.TimelineLength,
		})
	}

	return encounterReadResult{
		State:      encounterReadOK,
		Encounters: summaries,
		Count:      len(summaries),
	}
}

// findPlayerEncounter returns the encounter matching the player's active encounter ID.
// Returns nil if not found or no active encounter.
func findPlayerEncounter(encounters []encounterSummary, activeEncounterID string) *encounterSummary {
	if activeEncounterID == "" {
		return nil
	}
	for i := range encounters {
		if encounters[i].EncounterID == activeEncounterID {
			return &encounters[i]
		}
	}
	return nil
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
	return decodePlayerState(body, playerReadOK)
}
