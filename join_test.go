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
	footer := renderFooter(120, "", "", "", "", "", false, false, inventoryReadResult{})
	stripped := stripANSI(footer)
	if !strings.Contains(stripped, "(joining...)") {
		t.Fatalf("expected footer to contain '(joining...)' for pending state, got: %s", stripped)
	}
}

// TestFooterShowsRetryingWhenFailed verifies the footer displays a retry message
// when player join has failed.
func TestFooterShowsRetryingWhenFailed(t *testing.T) {
	footer := renderFooter(120, "", "", "", "", "", false, true, inventoryReadResult{})
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
	footer := renderFooter(120, "", "", "", "", "", true, false, inventoryReadResult{})
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
	failedFooter := renderFooter(120, "", "", "", "", "", false, true, inventoryReadResult{})
	failedStripped := stripANSI(failedFooter)
	if !strings.Contains(failedStripped, "not joined") {
		t.Fatalf("pre-condition: expected 'not joined' in failed footer, got: %s", failedStripped)
	}

	// Simulate successful rejoin: now playerJoined=true, playerFailed=false
	okFooter := renderFooter(120, "", "", "", "", "", true, false, inventoryReadResult{})
	okStripped := stripANSI(okFooter)
	if strings.Contains(okStripped, "not joined") || strings.Contains(okStripped, "joining") {
		t.Fatalf("after rejoin, footer should not show join status text, got: %s", okStripped)
	}
	if !strings.Contains(okStripped, "hjkl:mv") {
		t.Fatalf("after rejoin, footer should show normal help, got: %s", okStripped)
	}
}
