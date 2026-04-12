package main

import (
	"math"
	"testing"
)

// --- M20260412-16 coordinate correctness tests ---

func TestDirectionEpsilonPreventsTinyDelta(t *testing.T) {
	// Positions that render to the same cell but differ by floating-point noise
	player := playerPosResult{X: 20.0, Y: 15.0}
	mob := mobPosVec3{X: 20.001, Y: 15.002}
	dir := directionTowardMob(player, mob)
	if dir != "" {
		t.Fatalf("tiny delta (%.3f, %.3f) should return empty direction, got: %s",
			mob.X-player.X, mob.Y-player.Y, dir)
	}
}

func TestDirectionEpsilonAllowsRealDelta(t *testing.T) {
	// Positions far enough apart to warrant movement
	player := playerPosResult{X: 20.0, Y: 15.0}
	mob := mobPosVec3{X: 20.0, Y: 16.0}
	dir := directionTowardMob(player, mob)
	if dir == "" {
		t.Fatal("delta of 1.0 should return a direction")
	}
	if dir != "north" {
		t.Fatalf("mob at higher Y should produce 'north', got: %s", dir)
	}
}

func TestDirectionEpsilonBoundary(t *testing.T) {
	player := playerPosResult{X: 0, Y: 0}
	// Exactly at epsilon boundary
	mob := mobPosVec3{X: 0, Y: directionEpsilon - 0.01}
	if dir := directionTowardMob(player, mob); dir != "" {
		t.Fatalf("delta just under epsilon should return empty, got: %s", dir)
	}
	// Just over epsilon
	mob2 := mobPosVec3{X: 0, Y: directionEpsilon + 0.01}
	if dir := directionTowardMob(player, mob2); dir == "" {
		t.Fatal("delta just over epsilon should return a direction")
	}
}

// TestMovementReducesDistance verifies that each cardinal direction,
// when applied as a backend move, reduces the Euclidean distance to the target.
func TestMovementReducesDistance(t *testing.T) {
	// Backend move step (matches backend PlayerMoveStep = 5.0)
	step := 5.0

	cases := []struct {
		name    string
		playerX, playerY float64
		mobX, mobY       float64
	}{
		{"mob north", 0, 0, 0, 20},
		{"mob south", 0, 20, 0, 0},
		{"mob east", 0, 0, 20, 0},
		{"mob west", 20, 0, 0, 0},
		{"mob northeast", 0, 0, 15, 20},
		{"mob southwest", 20, 20, 0, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			player := playerPosResult{X: tc.playerX, Y: tc.playerY}
			mob := mobPosVec3{X: tc.mobX, Y: tc.mobY}

			dir := directionTowardMob(player, mob)
			if dir == "" {
				t.Fatal("should have a direction")
			}

			// Apply the backend move delta
			var dx, dy float64
			backendDir := directionToBackend(dir)
			switch backendDir {
			case "up":
				dy = step
			case "down":
				dy = -step
			case "right":
				dx = step
			case "left":
				dx = -step
			}

			newPlayer := playerPosResult{X: player.X + dx, Y: player.Y + dy}

			distBefore := math.Sqrt(
				(mob.X-player.X)*(mob.X-player.X) +
					(mob.Y-player.Y)*(mob.Y-player.Y))
			distAfter := math.Sqrt(
				(mob.X-newPlayer.X)*(mob.X-newPlayer.X) +
					(mob.Y-newPlayer.Y)*(mob.Y-newPlayer.Y))

			if distAfter >= distBefore {
				t.Fatalf("direction %q (backend %q) did not reduce distance: before=%.1f after=%.1f dx=%.1f dy=%.1f",
					dir, backendDir, distBefore, distAfter, dx, dy)
			}
		})
	}
}

// TestDirectionToBackendMappingComplete verifies all four cardinal directions map correctly.
func TestDirectionToBackendMappingComplete(t *testing.T) {
	cases := map[string]string{
		"north": "up",
		"south": "down",
		"east":  "right",
		"west":  "left",
	}
	for tui, backend := range cases {
		got := directionToBackend(tui)
		if got != backend {
			t.Fatalf("directionToBackend(%q) = %q, want %q", tui, got, backend)
		}
	}
}

// TestDirectionFromKeyMappingComplete verifies all movement keys map correctly.
func TestDirectionFromKeyMappingComplete(t *testing.T) {
	cases := map[string]string{
		"up": "north", "k": "north",
		"down": "south", "j": "south",
		"left": "west", "h": "west",
		"right": "east", "l": "east",
	}
	for key, expected := range cases {
		got := directionFromKey(key)
		if got != expected {
			t.Fatalf("directionFromKey(%q) = %q, want %q", key, got, expected)
		}
	}
}

// TestRenderAndMovementUseConsistentYAxis verifies that the Y axis meaning
// is consistent between projectToCell (rendering) and directionTowardMob (movement).
func TestRenderAndMovementUseConsistentYAxis(t *testing.T) {
	// Two positions: mob at higher Y than player
	bounds := mapBounds{MinX: 0, MaxX: 100, MinZ: 0, MaxZ: 100, SpanX: 100, SpanZ: 100}
	width, height := 20, 20

	playerPos := playerPosResult{X: 50, Y: 30}
	mobPos := mobPosVec3{X: 50, Y: 70}

	// Rendering: higher Y → lower row → higher on screen (north)
	_, playerRow := bounds.projectToCell(playerPos.X, playerPos.Y, width, height)
	_, mobRow := bounds.projectToCell(mobPos.X, mobPos.Y, width, height)

	if mobRow >= playerRow {
		t.Fatalf("mob at higher Y should render at lower row (higher on screen): playerRow=%d mobRow=%d",
			playerRow, mobRow)
	}

	// Movement: mob at higher Y → direction should be "north"
	dir := directionTowardMob(playerPos, mobPos)
	if dir != "north" {
		t.Fatalf("mob at higher Y should produce 'north', got: %s", dir)
	}

	// Backend: "north" → "up" → +Y → player Y increases → player row decreases (moves up on screen)
	backendDir := directionToBackend(dir)
	if backendDir != "up" {
		t.Fatalf("'north' should map to 'up', got: %s", backendDir)
	}
}
