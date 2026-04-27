package main

import (
	"math"
	"strings"
	"testing"
)

// --- A) Movement direction arrows in event text ---

func TestMoveEventArrowNorth(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 10, Y: 20}}
	msg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 10, Y: 25}},
		direction: "north",
	}
	updated, _ := m.Update(msg)
	um := updated.(model)
	last := um.events.recent()[len(um.events.recent())-1]
	if last.text != "↑ north" {
		t.Fatalf("expected '↑ north', got: %s", last.text)
	}
}

func TestMoveEventArrowSouth(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 10, Y: 20}}
	msg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 10, Y: 15}},
		direction: "south",
	}
	updated, _ := m.Update(msg)
	um := updated.(model)
	last := um.events.recent()[len(um.events.recent())-1]
	if last.text != "↓ south" {
		t.Fatalf("expected '↓ south', got: %s", last.text)
	}
}

func TestMoveEventArrowEast(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 10, Y: 20}}
	msg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 15, Y: 20}},
		direction: "east",
	}
	updated, _ := m.Update(msg)
	um := updated.(model)
	last := um.events.recent()[len(um.events.recent())-1]
	if last.text != "→ east" {
		t.Fatalf("expected '→ east', got: %s", last.text)
	}
}

func TestMoveEventArrowWest(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 10, Y: 20}}
	msg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 5, Y: 20}},
		direction: "west",
	}
	updated, _ := m.Update(msg)
	um := updated.(model)
	last := um.events.recent()[len(um.events.recent())-1]
	if last.text != "← west" {
		t.Fatalf("expected '← west', got: %s", last.text)
	}
}

func TestStepEventIncludesArrow(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 10, Y: 20}}
	msg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 15, Y: 20}},
		direction: "east",
		source:    "step",
	}
	updated, _ := m.Update(msg)
	um := updated.(model)
	last := um.events.recent()[len(um.events.recent())-1]
	if last.text != "→ step east" {
		t.Fatalf("expected '→ step east', got: %s", last.text)
	}
}

func TestBlockedMoveEventIncludesArrow(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 10, Y: 20}}
	msg := moveResultMsg{
		result:    moveResult{OK: false, Error: "boundary"},
		direction: "north",
	}
	updated, _ := m.Update(msg)
	um := updated.(model)
	last := um.events.recent()[len(um.events.recent())-1]
	if !strings.HasPrefix(last.text, "↑") {
		t.Fatalf("blocked move event should start with direction arrow, got: %s", last.text)
	}
	if !strings.Contains(last.text, "blocked") {
		t.Fatalf("blocked move event should contain 'blocked', got: %s", last.text)
	}
}

// --- B) Distance delta computation ---

func TestDistDeltaCloserWhenMovingToward(t *testing.T) {
	mobs := []mobPosition{{MobID: "orc-1", LiveMobID: "orc-1", Position: mobPosVec3{X: 50, Y: 20}}}
	m := model{slog: &sessionLogger{}}
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 10, Y: 20}}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1"}}
	m.rosterFocus = rosterFocus{index: 0}

	msg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 15, Y: 20}},
		direction: "east",
	}
	updated, _ := m.Update(msg)
	um := updated.(model)

	if !um.hasDistDelta {
		t.Fatal("distDelta should be set after moving toward focused target")
	}
	if um.distDelta >= 0 {
		t.Fatalf("distDelta should be negative (closer), got: %.1f", um.distDelta)
	}
}

func TestDistDeltaFartherWhenMovingAway(t *testing.T) {
	mobs := []mobPosition{{MobID: "orc-1", LiveMobID: "orc-1", Position: mobPosVec3{X: 50, Y: 20}}}
	m := model{slog: &sessionLogger{}}
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 10, Y: 20}}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1"}}
	m.rosterFocus = rosterFocus{index: 0}

	msg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 5, Y: 20}},
		direction: "west",
	}
	updated, _ := m.Update(msg)
	um := updated.(model)

	if !um.hasDistDelta {
		t.Fatal("distDelta should be set after moving away from focused target")
	}
	if um.distDelta <= 0 {
		t.Fatalf("distDelta should be positive (farther), got: %.1f", um.distDelta)
	}
}

func TestDistDeltaClearedWithoutFocus(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 10, Y: 20}}
	m.rosterFocus = rosterFocus{index: -1}

	msg := moveResultMsg{
		result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: 15, Y: 20}},
		direction: "east",
	}
	updated, _ := m.Update(msg)
	um := updated.(model)

	if um.hasDistDelta {
		t.Fatal("distDelta should not be set when no target is focused")
	}
}

// --- C) Distance delta suffix rendering ---

func TestDistDeltaSuffixCloser(t *testing.T) {
	s := distDeltaSuffix(-5.0, true)
	if !strings.Contains(s, "↓") {
		t.Fatalf("closer delta should contain ↓, got: %s", s)
	}
	if !strings.Contains(s, "5") {
		t.Fatalf("closer delta should contain magnitude, got: %s", s)
	}
}

func TestDistDeltaSuffixFarther(t *testing.T) {
	s := distDeltaSuffix(5.0, true)
	if !strings.Contains(s, "↑") {
		t.Fatalf("farther delta should contain ↑, got: %s", s)
	}
}

func TestDistDeltaSuffixNegligible(t *testing.T) {
	s := distDeltaSuffix(0.3, true)
	if s != "" {
		t.Fatalf("negligible delta should produce empty suffix, got: %s", s)
	}
}

func TestDistDeltaSuffixNoData(t *testing.T) {
	s := distDeltaSuffix(5.0, false)
	if s != "" {
		t.Fatalf("no delta data should produce empty suffix, got: %s", s)
	}
}

func TestProximityPanelShowsDistDelta(t *testing.T) {
	mobs := []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 30, Y: 0}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	entries := []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}

	panel := stripANSI(renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobr, pr, focus, entries, -5.0, true))
	if !strings.Contains(panel, "↓5") {
		t.Fatalf("proximity panel should show ↓5 delta, got:\n%s", panel)
	}
}

func TestProximityPanelNoDistDeltaWhenNone(t *testing.T) {
	mobs := []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 30, Y: 0}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	entries := []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}

	panel := stripANSI(renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobr, pr, focus, entries, 0, false))
	if strings.Contains(panel, "↓") || strings.Contains(panel, "↑") {
		t.Fatalf("proximity panel without delta should not show ↓ or ↑, got:\n%s", panel)
	}
}

// --- D) Range transition events ---

func TestRangeTransitionOORToInRange(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.hasPrevRange = true
	m.prevRangeInRange = false

	msg := targetConfirmMsg{result: targetConfirmResult{
		State:           targetConfirmOK,
		Found:           true,
		WithinProximity: true,
		Distance:        5.0,
		TargetID:        "orc-1",
	}}
	updated, _ := m.Update(msg)
	um := updated.(model)

	found := false
	for _, e := range um.events.recent() {
		if e.text == "IN RANGE" {
			found = true
		}
	}
	if !found {
		t.Fatal("OOR → IN RANGE transition should emit 'IN RANGE' event")
	}
}

func TestRangeTransitionInRangeToOOR(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.hasPrevRange = true
	m.prevRangeInRange = true

	msg := targetConfirmMsg{result: targetConfirmResult{
		State:           targetConfirmOK,
		Found:           true,
		WithinProximity: false,
		Distance:        50.0,
		TargetID:        "orc-1",
	}}
	updated, _ := m.Update(msg)
	um := updated.(model)

	found := false
	for _, e := range um.events.recent() {
		if e.text == "out of range" {
			found = true
		}
	}
	if !found {
		t.Fatal("IN RANGE → OOR transition should emit 'out of range' event")
	}
}

func TestNoRangeEventWhenStateUnchanged(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.hasPrevRange = true
	m.prevRangeInRange = true

	msg := targetConfirmMsg{result: targetConfirmResult{
		State:           targetConfirmOK,
		Found:           true,
		WithinProximity: true,
		Distance:        5.0,
		TargetID:        "orc-1",
	}}
	updated, _ := m.Update(msg)
	um := updated.(model)

	for _, e := range um.events.recent() {
		if e.text == "IN RANGE" || e.text == "out of range" {
			t.Fatal("no range event should be emitted when range state is unchanged")
		}
	}
}

func TestNoRangeEventOnFirstProximityCheck(t *testing.T) {
	m := model{slog: &sessionLogger{}}
	m.hasPrevRange = false

	msg := targetConfirmMsg{result: targetConfirmResult{
		State:           targetConfirmOK,
		Found:           true,
		WithinProximity: true,
		Distance:        5.0,
		TargetID:        "orc-1",
	}}
	updated, _ := m.Update(msg)
	um := updated.(model)

	for _, e := range um.events.recent() {
		if e.text == "IN RANGE" || e.text == "out of range" {
			t.Fatal("first proximity check should not emit range event")
		}
	}
	if !um.hasPrevRange {
		t.Fatal("hasPrevRange should be set after first proximity check")
	}
}

func TestRangeTransitionEventEmphasis(t *testing.T) {
	if eventEmphasis("IN RANGE") != "strong" {
		t.Fatal("IN RANGE event should have strong emphasis")
	}
	if eventEmphasis("out of range") != "strong" {
		t.Fatal("out of range event should have strong emphasis")
	}
}

// --- E) Spatial agreement: proximity panel and map data agree ---

func TestProximityDistanceMatchesLocalComputation(t *testing.T) {
	playerPos := playerPosResult{X: 10, Y: 20}
	mobPos := mobPosVec3{X: 40, Y: 20}
	expectedDist := math.Sqrt(float64((40-10)*(40-10) + (20-20)*(20-20)))

	mobs := []mobPosition{{MobID: "orc-1", Position: mobPos}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPos}
	entries := []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}

	panel := stripANSI(renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobr, pr, focus, entries, 0, false))
	distStr := strings.TrimSpace(panel)
	if !strings.Contains(distStr, "dist:30") {
		t.Fatalf("proximity panel should show dist:%.0f, got:\n%s", expectedDist, panel)
	}
}

func TestBackendDistanceShownWhenAvailable(t *testing.T) {
	mobs := []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 40, Y: 20}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 10, Y: 20}}
	entries := []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}
	tc := targetConfirmResult{
		State:           targetConfirmOK,
		Found:           true,
		WithinProximity: false,
		Distance:        30.0,
		TargetID:        "orc-1",
	}

	panel := stripANSI(renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobr, pr, focus, entries, 0, false))
	if !strings.Contains(panel, "OOR") {
		t.Fatalf("proximity panel should show OOR when out of range, got:\n%s", panel)
	}
	if !strings.Contains(panel, "dist:30") {
		t.Fatalf("proximity panel should show backend distance, got:\n%s", panel)
	}
}

func TestInRangeShownWithBackendTruth(t *testing.T) {
	mobs := []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 15, Y: 20}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 10, Y: 20}}
	entries := []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}
	tc := targetConfirmResult{
		State:           targetConfirmOK,
		Found:           true,
		WithinProximity: true,
		Distance:        5.0,
		TargetID:        "orc-1",
	}

	panel := stripANSI(renderProximityPanel(sidePanelWidth, tc, attackResult{}, mobr, pr, focus, entries, 0, false))
	if !strings.Contains(panel, "IN RANGE") {
		t.Fatalf("proximity panel should show IN RANGE, got:\n%s", panel)
	}
	if !strings.Contains(panel, "dist:5") {
		t.Fatalf("proximity panel should show backend distance, got:\n%s", panel)
	}
}

// --- F) Repeated frame correctness ---

func TestProximityPanelWithDeltaIdempotent(t *testing.T) {
	mobs := []mobPosition{{MobID: "orc-1", Position: mobPosVec3{X: 30, Y: 0}}}
	mobr := mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	pr := playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 0, Y: 0}}
	entries := []rosterEntry{{kind: "mb", id: "orc-1", canonicalMobID: true}}
	focus := rosterFocus{index: 0}

	var outputs [3]string
	for i := range outputs {
		outputs[i] = renderProximityPanel(sidePanelWidth, targetConfirmResult{}, attackResult{}, mobr, pr, focus, entries, -5.0, true)
	}
	for i := 1; i < len(outputs); i++ {
		if outputs[i] != outputs[0] {
			t.Errorf("renderProximityPanel with delta: call %d differs from call 0", i)
		}
	}
}

func TestDistDeltaConsistentAcrossMoves(t *testing.T) {
	mobs := []mobPosition{{MobID: "orc-1", LiveMobID: "orc-1", Position: mobPosVec3{X: 50, Y: 20}}}
	m := model{slog: &sessionLogger{}}
	m.playerRead = playerReadResult{State: playerReadOK, HasPos: true, Position: playerPosResult{X: 10, Y: 20}}
	m.mobRead = mobReadResult{State: mobReadOK, Mobs: mobs, Count: 1}
	m.rosterEntries = []rosterEntry{{kind: "mb", id: "orc-1"}}
	m.rosterFocus = rosterFocus{index: 0}

	// Two successive moves toward target should both produce negative delta
	for step := 0; step < 2; step++ {
		oldX := m.playerRead.Position.X
		msg := moveResultMsg{
			result:    moveResult{OK: true, HasPos: true, Position: playerPosResult{X: oldX + 5, Y: 20}},
			direction: "east",
		}
		updated, _ := m.Update(msg)
		m = updated.(model)

		if !m.hasDistDelta {
			t.Fatalf("step %d: distDelta should be set", step)
		}
		if m.distDelta >= 0 {
			t.Fatalf("step %d: distDelta should be negative (closer), got: %.1f", step, m.distDelta)
		}
	}
}

func TestDirectionArrowFunction(t *testing.T) {
	cases := []struct {
		dir  string
		want string
	}{
		{"north", "↑"},
		{"south", "↓"},
		{"east", "→"},
		{"west", "←"},
		{"", ""},
		{"northeast", ""},
	}
	for _, tc := range cases {
		got := directionArrow(tc.dir)
		if got != tc.want {
			t.Errorf("directionArrow(%q) = %q, want %q", tc.dir, got, tc.want)
		}
	}
}
