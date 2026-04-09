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
	start := time.Now()
	globalSessionLog.LogRequest("zone_status", "GET", url)

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

	globalSessionLog.LogResponse("zone_status", resp.StatusCode, true, time.Since(start).Milliseconds())
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
	ProcessID string     // backend process key, preserved for identity matching
	MobName   string     `json:"mob_name"`
	Position  mobPosVec3 `json:"position"`
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
	for pid, m := range raw.Result {
		m.ProcessID = pid
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
	start := time.Now()

	// Submit position change
	newX := currentPos.X + dx
	newY := currentPos.Y + dy

	globalSessionLog.LogRequestWith("submit_move", "POST", devPlayerPositionURL(target), map[string]any{
		"action": "move",
		"from":   []float64{currentPos.X, currentPos.Y},
		"to":     []float64{newX, newY},
	})

	payload := fmt.Sprintf(`{"player_id":"%s","x":%f,"y":%f,"z":0}`, target.Player, newX, newY)

	req, err := http.NewRequest("POST", devPlayerPositionURL(target), strings.NewReader(payload))
	if err != nil {
		globalSessionLog.LogResponseWith("submit_move", 0, false, time.Since(start).Milliseconds(), map[string]any{"error": "request_create"})
		return moveResult{OK: false, Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Seq-Dev-Token", target.DevToken)

	resp, err := client.Do(req)
	if err != nil {
		globalSessionLog.LogResponseWith("submit_move", 0, false, time.Since(start).Milliseconds(), map[string]any{"error": "network"})
		return moveResult{OK: false, Error: err.Error()}
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		globalSessionLog.LogResponse("submit_move", resp.StatusCode, false, time.Since(start).Milliseconds())
		return moveResult{OK: false, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	globalSessionLog.LogResponse("submit_move", resp.StatusCode, true, time.Since(start).Milliseconds())

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

	result := moveResult{
		OK: true,
		Position: playerPosResult{
			X: raw.Result.Position.Pos.X,
			Y: raw.Result.Position.Pos.Y,
		},
		HasPos: true,
	}
	globalSessionLog.LogPlayerSnapshot("move_readback", true, true, result.Position.X, result.Position.Y, false)
	return result
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
	DropsGenerated  bool     // backend says drops have been generated
	Drops           []string // backend-owned available drop item IDs
	LootExpired     bool     // backend says loot window has expired

	// Combat readback — backend-owned attack resolution and engagement data.
	// Read-only display, no client inference.
	LatestResultKind    string // "damage_applied", "attack_miss", "heal_applied", etc.
	LatestResultActor   string // attacker/caster ID
	LatestResultTarget  string // defender/target ID
	LatestResultValue   int    // damage or heal amount
	LatestResultSummary string // human-readable one-line summary from backend
	TextSummaryLatest   string // backend-composed latest event description

	// Mob engagement — which mobs are targeting which players.
	// Populated from backend mob_threat array. Read-only, no threat logic.
	MobThreat []mobThreatEntry
}

// mobThreatEntry holds one mob's backend-owned targeting and threat data.
// Read-only — no client-side threat calculation or inference.
type mobThreatEntry struct {
	MobID                  string
	SelectedTargetPlayerID string
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
			DropsGenerated  bool     `json:"drops_generated"`
			Drops           []string `json:"drops"`
			LootExpired     bool     `json:"loot_expired"`
			// Combat readback fields
			LatestResultKind    string `json:"latest_result_kind"`
			LatestResultActor   string `json:"latest_result_actor"`
			LatestResultTarget  string `json:"latest_result_target"`
			LatestResultValue   int    `json:"latest_result_value"`
			LatestResultSummary string `json:"latest_result_summary"`
			TextSummaryLatest   string `json:"text_summary_latest"`
			// Mob engagement
			MobThreat []struct {
				MobID                  string `json:"mob_id"`
				SelectedTargetPlayerID string `json:"selected_target_player_id"`
			} `json:"mob_threat"`
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
		var threat []mobThreatEntry
		for _, mt := range e.MobThreat {
			threat = append(threat, mobThreatEntry{
				MobID:                  mt.MobID,
				SelectedTargetPlayerID: mt.SelectedTargetPlayerID,
			})
		}
		summaries = append(summaries, encounterSummary{
			EncounterID:         e.EncounterID,
			State:               e.State,
			CompletedReason:     e.CompletedReason,
			PlayerIDs:           e.PlayerIDs,
			MobIDs:              e.MobIDs,
			PlayerCount:         len(e.PlayerIDs),
			MobCount:            len(e.MobIDs),
			MobsAlive:           e.MobsAliveCount,
			MobsDead:            e.MobsDeadCount,
			ActionIndex:         e.ActionIndex,
			TimelineLength:      e.TimelineLength,
			DropsGenerated:      e.DropsGenerated,
			Drops:               e.Drops,
			LootExpired:         e.LootExpired,
			LatestResultKind:    e.LatestResultKind,
			LatestResultActor:   e.LatestResultActor,
			LatestResultTarget:  e.LatestResultTarget,
			LatestResultValue:   e.LatestResultValue,
			LatestResultSummary: e.LatestResultSummary,
			TextSummaryLatest:   e.TextSummaryLatest,
			MobThreat:           threat,
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

// --- Target proximity ---

// targetConfirmState represents the outcome of a target confirmation query.
type targetConfirmState int

const (
	targetConfirmNone targetConfirmState = iota
	targetConfirmOK
	targetConfirmFailed
)

// targetConfirmResult holds the outcome of a backend target proximity query.
// This is purely a backend-owned read result, not local state.
type targetConfirmResult struct {
	State           targetConfirmState
	Error           string
	TargetKind      string  // "mb" or "pc" — mirrors roster entry kind
	TargetID        string  // backend-owned ID
	Found           bool    // backend says target exists
	WithinProximity bool    // backend says target is within action proximity
	Distance        float64 // backend-owned 2D distance
	MobName         string  // backend-owned mob display name
}

// targetStatusLabel returns a compact display label for the target confirmation state.
// Explicitly distinguishes from local focus by using "(backend)" marker.
func (r targetConfirmResult) targetStatusLabel() string {
	switch r.State {
	case targetConfirmOK:
		if !r.Found {
			return "target: not found (backend)"
		}
		label := "target: " + r.TargetKind + ":" + r.TargetID
		if r.MobName != "" {
			label = "target: " + r.MobName
		}
		return label + " (backend)"
	case targetConfirmFailed:
		return "target: unavailable"
	default:
		return "target: none"
	}
}

// devTargetProximityURL builds the dev target proximity endpoint URL.
func devTargetProximityURL(target backendTarget, actorPid string) string {
	base := strings.TrimRight(target.BaseURL, "/")
	return fmt.Sprintf("%s/world/dev/zone/%s/player/%s/target/%s/proximity",
		base, target.Zone, target.Player, actorPid)
}

// queryTargetProximity performs a single GET to the backend target proximity endpoint.
// Returns backend-owned truth about the target relationship. No writes.
func queryTargetProximity(target backendTarget, entry rosterEntry) targetConfirmResult {
	if entry.kind != "mb" {
		// Proximity endpoint is designed for mob targets; for PCs, report honestly
		return targetConfirmResult{
			State:      targetConfirmFailed,
			Error:      "proximity query supports mob targets only",
			TargetKind: entry.kind,
			TargetID:   entry.id,
		}
	}

	url := devTargetProximityURL(target, entry.id)
	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return targetConfirmResult{State: targetConfirmFailed, Error: err.Error(), TargetKind: entry.kind, TargetID: entry.id}
	}
	req.Header.Set("X-Seq-Dev-Token", target.DevToken)

	resp, err := client.Do(req)
	if err != nil {
		return targetConfirmResult{State: targetConfirmFailed, Error: err.Error(), TargetKind: entry.kind, TargetID: entry.id}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return targetConfirmResult{
			State:      targetConfirmFailed,
			Error:      fmt.Sprintf("HTTP %d", resp.StatusCode),
			TargetKind: entry.kind,
			TargetID:   entry.id,
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return targetConfirmResult{State: targetConfirmFailed, Error: "failed to read body", TargetKind: entry.kind, TargetID: entry.id}
	}

	var raw struct {
		Found           bool    `json:"found"`
		WithinProximity bool    `json:"within_proximity"`
		Distance2D      float64 `json:"distance_2d"`
		TargetMobName   string  `json:"target_mob_name"`
		TargetMobID     string  `json:"target_mob_id"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return targetConfirmResult{State: targetConfirmFailed, Error: "failed to decode proximity", TargetKind: entry.kind, TargetID: entry.id}
	}

	return targetConfirmResult{
		State:           targetConfirmOK,
		TargetKind:      entry.kind,
		TargetID:        entry.id,
		Found:           raw.Found,
		WithinProximity: raw.WithinProximity,
		Distance:        raw.Distance2D,
		MobName:         raw.TargetMobName,
	}
}

// --- BasicAttack intent submission ---

// attackState represents the outcome of a BasicAttack intent submission.
type attackState int

const (
	attackStateNone attackState = iota
	attackStateSent
	attackStateFailed
)

// attackResult holds the outcome of a BasicAttack intent submission.
// Purely a submission receipt — no combat logic, no damage tracking.
type attackResult struct {
	State    attackState
	Error    string
	TargetID string // mob ID that was targeted
}

// attackStatusLabel returns a compact display label for the attack submission state.
func (r attackResult) attackStatusLabel() string {
	switch r.State {
	case attackStateSent:
		return "attack: sent"
	case attackStateFailed:
		return "attack: failed"
	default:
		return ""
	}
}

// devIntentURL builds the dev intent submission endpoint URL.
func devIntentURL(target backendTarget) string {
	base := strings.TrimRight(target.BaseURL, "/")
	url := fmt.Sprintf("%s/world/dev/zone/%s/intent", base, target.Zone)
	if strings.EqualFold(target.Mode, "ASYNC") {
		url += "?mode=Async"
	}
	return url
}

// submitBasicAttack submits a BasicAttack intent against the specified mob.
// Returns a submission receipt only — no combat logic or damage tracking.
func submitBasicAttack(target backendTarget, mobID string) attackResult {
	url := devIntentURL(target)
	start := time.Now()
	globalSessionLog.LogRequest("basic_attack", "POST", url)
	payload := fmt.Sprintf(`{"player_id":"%s","intent_kind":"BasicAttack","target":{"kind":"Mob","id":"%s"}}`,
		target.Player, mobID)

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return attackResult{State: attackStateFailed, Error: err.Error(), TargetID: mobID}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Seq-Dev-Token", target.DevToken)

	resp, err := client.Do(req)
	if err != nil {
		return attackResult{State: attackStateFailed, Error: err.Error(), TargetID: mobID}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return attackResult{State: attackStateFailed, Error: "failed to read body", TargetID: mobID}
	}

	// Check for backend success/failure
	var raw struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return attackResult{State: attackStateFailed, Error: "failed to decode response", TargetID: mobID}
	}

	if !raw.OK {
		errMsg := raw.Error
		if errMsg == "" {
			errMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return attackResult{State: attackStateFailed, Error: errMsg, TargetID: mobID}
	}

	globalSessionLog.LogResponse("basic_attack", resp.StatusCode, true, time.Since(start).Milliseconds())
	return attackResult{State: attackStateSent, TargetID: mobID}
}

// --- Player inventory readback ---

// inventoryReadState represents the outcome of an inventory read.
type inventoryReadState int

const (
	inventoryReadNotAttempted inventoryReadState = iota
	inventoryReadOK
	inventoryReadFailed
)

// inventoryReadResult holds the outcome of a backend inventory and lifecycle read.
// Decoded from the gameplay_status endpoint. Includes both inventory and player
// lifecycle fields. All values are backend-owned truth — no client inference.
type inventoryReadResult struct {
	State inventoryReadState
	Error string
	Items []string // backend-owned item IDs
	Count int

	// Player lifecycle fields — backend-owned, read-only.
	CanAct        bool   // backend says player can currently act
	BlockedReason string // backend reason for being unable to act (empty if can act)
	HPCurrent     int    // backend-owned current HP
	HPMax         int    // backend-owned max HP
	HasLifecycle  bool   // true if lifecycle fields were decoded from response
}

// gameplayStatusURL builds the gameplay status call URL.
func gameplayStatusURL(target backendTarget) string {
	base := strings.TrimRight(target.BaseURL, "/")
	url := fmt.Sprintf("%s/world/call/%s?message=gameplay_status", base, target.Zone)
	if strings.EqualFold(target.Mode, "ASYNC") {
		url += "&mode=Async"
	}
	return url
}

// fetchPlayerInventory reads the player's inventory from the gameplay_status call surface.
// Returns backend-owned inventory truth only — no simulation.
func fetchPlayerInventory(target backendTarget) inventoryReadResult {
	url := gameplayStatusURL(target)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return inventoryReadResult{State: inventoryReadFailed, Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return inventoryReadResult{State: inventoryReadFailed, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return inventoryReadResult{State: inventoryReadFailed, Error: "failed to read body"}
	}

	var envelope struct {
		Result struct {
			Players []struct {
				PlayerID      string   `json:"player_id"`
				Inventory     []string `json:"inventory"`
				CanAct        bool     `json:"can_act"`
				BlockedReason string   `json:"blocked_reason"`
				HPCurrent     int      `json:"hp_current"`
				HPMax         int      `json:"hp_max"`
			} `json:"players"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return inventoryReadResult{State: inventoryReadFailed, Error: "failed to decode"}
	}

	// Find our player in the players array
	for _, p := range envelope.Result.Players {
		if p.PlayerID == target.Player {
			return inventoryReadResult{
				State:         inventoryReadOK,
				Items:         p.Inventory,
				Count:         len(p.Inventory),
				CanAct:        p.CanAct,
				BlockedReason: p.BlockedReason,
				HPCurrent:     p.HPCurrent,
				HPMax:         p.HPMax,
				HasLifecycle:  true,
			}
		}
	}

	// Player not found in gameplay status — return OK with empty inventory
	return inventoryReadResult{State: inventoryReadOK, Count: 0}
}

// --- Pickup intent submission ---

// pickupState represents the outcome of a pickup_item intent submission.
type pickupState int

const (
	pickupStateNone pickupState = iota
	pickupStateSent
	pickupStateFailed
)

// pickupResult holds the outcome of a pickup_item intent submission.
// Purely a submission receipt — no inventory simulation.
type pickupResult struct {
	State       pickupState
	Error       string
	EncounterID string
	ItemID      string
}

// pickupStatusLabel returns a compact display label for the pickup submission state.
func (r pickupResult) pickupStatusLabel() string {
	switch r.State {
	case pickupStateSent:
		return "pk:" + r.ItemID
	case pickupStateFailed:
		return "pk:fail"
	default:
		return ""
	}
}

// submitPickupItem submits a pickup_item intent for the specified encounter and item.
// Returns a submission receipt only — no inventory simulation.
func submitPickupItem(target backendTarget, encounterID, itemID string) pickupResult {
	url := devIntentURL(target)
	payload := fmt.Sprintf(`{"player_id":"%s","intent_kind":"pickup_item","encounter_id":"%s","item_id":"%s"}`,
		target.Player, encounterID, itemID)

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return pickupResult{State: pickupStateFailed, Error: err.Error(), EncounterID: encounterID, ItemID: itemID}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Seq-Dev-Token", target.DevToken)

	resp, err := client.Do(req)
	if err != nil {
		return pickupResult{State: pickupStateFailed, Error: err.Error(), EncounterID: encounterID, ItemID: itemID}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return pickupResult{State: pickupStateFailed, Error: "failed to read body", EncounterID: encounterID, ItemID: itemID}
	}

	var raw struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return pickupResult{State: pickupStateFailed, Error: "failed to decode response", EncounterID: encounterID, ItemID: itemID}
	}

	if !raw.OK {
		errMsg := raw.Error
		if errMsg == "" {
			errMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return pickupResult{State: pickupStateFailed, Error: errMsg, EncounterID: encounterID, ItemID: itemID}
	}

	return pickupResult{State: pickupStateSent, EncounterID: encounterID, ItemID: itemID}
}

// --- Respawn intent submission ---

// respawnState represents the outcome of a Respawn intent submission.
type respawnState int

const (
	respawnStateNone respawnState = iota
	respawnStateSent
	respawnStateFailed
)

// respawnResult holds the outcome of a Respawn intent submission.
// Purely a submission receipt — backend-confirmed restoration is shown
// through lifecycle readback (HP, can-act), not through this result.
type respawnResult struct {
	State respawnState
	Error string
}

// respawnStatusLabel returns a compact display label for the respawn submission state.
func (r respawnResult) respawnStatusLabel() string {
	switch r.State {
	case respawnStateSent:
		return "respawn: sent"
	case respawnStateFailed:
		return "respawn: failed"
	default:
		return ""
	}
}

// submitRespawn submits a Respawn intent via the dev intent endpoint.
// Returns a submission receipt only — recovery confirmation comes from lifecycle readback.
func submitRespawn(target backendTarget) respawnResult {
	url := devIntentURL(target)
	payload := fmt.Sprintf(`{"player_id":"%s","intent_kind":"Respawn","target":{"kind":"none"}}`,
		target.Player)

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return respawnResult{State: respawnStateFailed, Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Seq-Dev-Token", target.DevToken)

	resp, err := client.Do(req)
	if err != nil {
		return respawnResult{State: respawnStateFailed, Error: err.Error()}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return respawnResult{State: respawnStateFailed, Error: "failed to read body"}
	}

	var raw struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return respawnResult{State: respawnStateFailed, Error: "failed to decode response"}
	}

	if !raw.OK {
		errMsg := raw.Error
		if errMsg == "" {
			errMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return respawnResult{State: respawnStateFailed, Error: errMsg}
	}

	return respawnResult{State: respawnStateSent}
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
