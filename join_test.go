package main

import (
	"strings"
	"testing"
	"time"
)

// --- Join reliability tests ---
//
// Validates that the TUI retries join when playerRead.State is failed,
// and that the footer reflects the correct join state.

// TestRefreshRetriesToJoinWhenFailed verifies that the refreshTickMsg handler
// returns a non-nil Cmd when player state is failed, triggering a rejoin.
func TestRefreshRetriesToJoinWhenFailed(t *testing.T) {
	m := model{
		width:  80,
		height: 40,
		target: defaultTarget(),
		playerRead: playerReadResult{
			State: playerReadFailed,
			Error: "connection refused",
		},
	}
	msg := refreshTickMsg(time.Now())
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("expected non-nil Cmd from refreshTickMsg when player state is failed")
	}
}

// TestRefreshReturnsCommandWhenPlayerOK verifies the normal refresh path
// also returns a non-nil Cmd (for readPlayerState).
func TestRefreshReturnsCommandWhenPlayerOK(t *testing.T) {
	m := model{
		width:  80,
		height: 40,
		target: defaultTarget(),
		playerRead: playerReadResult{
			State: playerReadOK,
		},
	}
	msg := refreshTickMsg(time.Now())
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("expected non-nil Cmd from refreshTickMsg when player state is OK")
	}
}

// TestFooterShowsJoiningWhenPending verifies the footer displays "(joining...)"
// when player has not joined and has not failed.
func TestFooterShowsJoiningWhenPending(t *testing.T) {
	footer := renderFooter(120, "", "", "", "", "", false, false, "", inventoryReadResult{})
	stripped := stripANSI(footer)
	if !strings.Contains(stripped, "(joining...)") {
		t.Fatalf("expected footer to contain '(joining...)' for pending state, got: %s", stripped)
	}
}

// TestFooterShowsRetryingWhenFailed verifies the footer displays a retry message
// when player join has failed.
func TestFooterShowsRetryingWhenFailed(t *testing.T) {
	footer := renderFooter(120, "", "", "", "", "", false, true, "", inventoryReadResult{})
	stripped := stripANSI(footer)
	if !strings.Contains(stripped, "not joined") {
		t.Fatalf("expected footer to contain 'not joined' for failed state, got: %s", stripped)
	}
	if !strings.Contains(stripped, "retrying") {
		t.Fatalf("expected footer to contain 'retrying' for failed state, got: %s", stripped)
	}
}

// TestFooterShowsNormalHelpWhenJoined verifies the footer displays the full
// keybinding help when player is successfully joined.
func TestFooterShowsNormalHelpWhenJoined(t *testing.T) {
	footer := renderFooter(120, "", "", "", "", "", true, false, "", inventoryReadResult{})
	stripped := stripANSI(footer)
	if !strings.Contains(stripped, "hjkl:mv") {
		t.Fatalf("expected footer to contain normal keybinding help, got: %s", stripped)
	}
	if strings.Contains(stripped, "joining") || strings.Contains(stripped, "not joined") {
		t.Fatalf("footer should not show join status text when joined, got: %s", stripped)
	}
}

// TestFooterTransitionsFromFailedToJoined verifies that when player state
// transitions from failed to OK, the footer changes from retry text to normal help.
func TestFooterTransitionsFromFailedToJoined(t *testing.T) {
	// Failed state footer
	failedFooter := renderFooter(120, "", "", "", "", "", false, true, "", inventoryReadResult{})
	failedStripped := stripANSI(failedFooter)
	if !strings.Contains(failedStripped, "not joined") {
		t.Fatalf("pre-condition: expected 'not joined' in failed footer, got: %s", failedStripped)
	}

	// Simulate successful rejoin: now playerJoined=true, playerFailed=false
	okFooter := renderFooter(120, "", "", "", "", "", true, false, "", inventoryReadResult{})
	okStripped := stripANSI(okFooter)
	if strings.Contains(okStripped, "not joined") || strings.Contains(okStripped, "joining") {
		t.Fatalf("after rejoin, footer should not show join status text, got: %s", okStripped)
	}
	if !strings.Contains(okStripped, "hjkl:mv") {
		t.Fatalf("after rejoin, footer should show normal help, got: %s", okStripped)
	}
}

// --- Error display and classification tests ---

// TestFooterShowsErrorReasonWhenFailed verifies that a failed join with a specific
// error string surfaces that error in the footer.
func TestFooterShowsErrorReasonWhenFailed(t *testing.T) {
	footer := renderFooter(120, "", "", "", "", "", false, true, "connection refused (is backend running?)", inventoryReadResult{})
	stripped := stripANSI(footer)
	if !strings.Contains(stripped, "connection refused") {
		t.Fatalf("expected footer to contain error reason 'connection refused', got: %s", stripped)
	}
}

// TestFooterTruncatesLongError verifies that an excessively long error string
// is truncated to 60 characters in the footer display.
func TestFooterTruncatesLongError(t *testing.T) {
	longError := strings.Repeat("x", 100)
	footer := renderFooter(120, "", "", "", "", "", false, true, longError, inventoryReadResult{})
	stripped := stripANSI(footer)
	if strings.Contains(stripped, longError) {
		t.Fatalf("footer should not contain the full 100-char error, got: %s", stripped)
	}
	// The error should be truncated to 60 chars inside the "(not joined: ...)" wrapper
	truncated := longError[:60]
	if !strings.Contains(stripped, truncated) {
		t.Fatalf("footer should contain truncated error (first 60 chars), got: %s", stripped)
	}
}

// TestClassifyJoinErrorConnectionRefused verifies connection-refused errors
// are classified with a helpful backend-running hint.
func TestClassifyJoinErrorConnectionRefused(t *testing.T) {
	result := classifyJoinError("dial tcp: connection refused")
	if !strings.Contains(result, "connection refused") {
		t.Fatalf("expected 'connection refused' in result, got: %s", result)
	}
	if !strings.Contains(result, "backend running") {
		t.Fatalf("expected 'backend running' hint in result, got: %s", result)
	}
}

// TestClassifyJoinErrorTimeout verifies timeout errors are classified as request timeout.
func TestClassifyJoinErrorTimeout(t *testing.T) {
	result := classifyJoinError("context deadline exceeded: timeout")
	if result != "request timeout" {
		t.Fatalf("expected 'request timeout', got: %s", result)
	}
}

// TestClassifyJoinErrorHTTP403 verifies HTTP 403 is classified as forbidden.
func TestClassifyJoinErrorHTTP403(t *testing.T) {
	result := classifyJoinError("HTTP 403")
	if !strings.Contains(result, "forbidden") {
		t.Fatalf("expected 'forbidden' in result, got: %s", result)
	}
}

// TestClassifyJoinErrorHTTP404 verifies HTTP 404 is classified as not found.
func TestClassifyJoinErrorHTTP404(t *testing.T) {
	result := classifyJoinError("HTTP 404")
	if !strings.Contains(result, "not found") {
		t.Fatalf("expected 'not found' in result, got: %s", result)
	}
}

// TestClassifyJoinErrorGenericHTTP verifies an unrecognized HTTP status code
// is returned as-is.
func TestClassifyJoinErrorGenericHTTP(t *testing.T) {
	result := classifyJoinError("HTTP 500")
	if result != "HTTP 500" {
		t.Fatalf("expected 'HTTP 500' returned as-is, got: %s", result)
	}
}

// TestClassifyJoinErrorLongMessage verifies that a long raw error message
// is truncated to 80 characters.
func TestClassifyJoinErrorLongMessage(t *testing.T) {
	longMsg := strings.Repeat("a", 100)
	result := classifyJoinError(longMsg)
	if len(result) != 80 {
		t.Fatalf("expected result truncated to 80 chars, got %d chars: %s", len(result), result)
	}
}

// TestPlayerStatusLabelIncludesError verifies that playerStatusLabel surfaces
// the error string when the player state is failed.
func TestPlayerStatusLabelIncludesError(t *testing.T) {
	pr := playerReadResult{
		State: playerReadFailed,
		Error: "forbidden (check dev token)",
	}
	label := pr.playerStatusLabel()
	if !strings.Contains(label, "forbidden") {
		t.Fatalf("expected 'forbidden' in player status label, got: %s", label)
	}
}
