package main

import (
	"strings"
	"testing"
	"time"
)

// TestStartupSmokeJoinedFooter verifies that a joined player model renders
// "player: joined" and does not show "not joined" in the layout output.
func TestStartupSmokeJoinedFooter(t *testing.T) {
	raw := renderLayout(
		120, 40, "",
		defaultTarget(),
		zoneReadResult{},
		mapReadResult{},
		mobReadResult{},
		playerReadResult{
			State:  playerReadOK,
			HasPos: true,
			Position: playerPosResult{X: 10, Y: 20},
		},
		encounterReadResult{},
		rosterFocus{},
		nil,
		targetConfirmResult{},
		attackResult{},
		pickupResult{},
		inventoryReadResult{},
		0, -1,
		respawnResult{},
		eventLog{},
		true, // showDiag: diagnostics panel includes player status label
	)
	out := stripANSI(raw)
	if strings.Contains(out, "not joined") {
		t.Fatal("joined player layout should not contain 'not joined'")
	}
	if !strings.Contains(out, "player: joined") {
		t.Fatal("joined player layout should contain 'player: joined'")
	}
}

// TestStartupSmokeFailedJoinFooter verifies that a failed-join model shows
// "not joined" and the error reason in the layout output.
func TestStartupSmokeFailedJoinFooter(t *testing.T) {
	raw := renderLayout(
		120, 40, "",
		defaultTarget(),
		zoneReadResult{},
		mapReadResult{},
		mobReadResult{},
		playerReadResult{
			State: playerReadFailed,
			Error: "connection refused (is backend running?)",
		},
		encounterReadResult{},
		rosterFocus{},
		nil,
		targetConfirmResult{},
		attackResult{},
		pickupResult{},
		inventoryReadResult{},
		0, -1,
		respawnResult{},
		eventLog{},
		false,
	)
	out := stripANSI(raw)
	if !strings.Contains(out, "not joined") {
		t.Fatal("failed-join layout should contain 'not joined'")
	}
	if !strings.Contains(out, "connection refused") {
		t.Fatal("failed-join layout should contain the error reason 'connection refused'")
	}
}

// TestStartupSmokeRetryCmd verifies that a refreshTickMsg on a failed-join model
// returns a non-nil Cmd, indicating a retry will fire.
func TestStartupSmokeRetryCmd(t *testing.T) {
	m := model{
		width:  120,
		height: 40,
		target: defaultTarget(),
		playerRead: playerReadResult{
			State: playerReadFailed,
			Error: "connection refused",
		},
	}
	_, cmd := m.Update(refreshTickMsg(time.Now()))
	if cmd == nil {
		t.Fatal("expected non-nil Cmd from refreshTickMsg when player state is failed (retry should fire)")
	}
}
