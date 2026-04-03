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
