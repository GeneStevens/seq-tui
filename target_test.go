package main

import "testing"

// TestDefaultTargetExactDefaults verifies the exact hardcoded defaults when no env vars are set.
func TestDefaultTargetExactDefaults(t *testing.T) {
	tgt := defaultTarget()
	if tgt.BaseURL != "http://localhost:9090" {
		t.Fatalf("expected BaseURL 'http://localhost:9090', got: %s", tgt.BaseURL)
	}
	if tgt.Zone != "crushbone" {
		t.Fatalf("expected Zone 'crushbone', got: %s", tgt.Zone)
	}
	if tgt.Player != "p1" {
		t.Fatalf("expected Player 'p1', got: %s", tgt.Player)
	}
	if tgt.DevToken != "abc" {
		t.Fatalf("expected DevToken 'abc', got: %s", tgt.DevToken)
	}
}

// TestDefaultTargetEnvOverride verifies that env vars override the defaults.
func TestDefaultTargetEnvOverride(t *testing.T) {
	t.Setenv("SEQ_BACKEND_URL", "http://example:8080")
	t.Setenv("SEQ_ZONE", "testzone")
	t.Setenv("SEQ_PLAYER", "testplayer")
	t.Setenv("SEQ_DEV_TOKEN", "testtoken")

	tgt := defaultTarget()
	if tgt.BaseURL != "http://example:8080" {
		t.Fatalf("expected BaseURL 'http://example:8080', got: %s", tgt.BaseURL)
	}
	if tgt.Zone != "testzone" {
		t.Fatalf("expected Zone 'testzone', got: %s", tgt.Zone)
	}
	if tgt.Player != "testplayer" {
		t.Fatalf("expected Player 'testplayer', got: %s", tgt.Player)
	}
	if tgt.DevToken != "testtoken" {
		t.Fatalf("expected DevToken 'testtoken', got: %s", tgt.DevToken)
	}
}
