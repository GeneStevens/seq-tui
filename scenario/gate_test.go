package scenario_test

import (
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/scenario"
)

// The gates say no to a run over its limit and yes at it.
func TestSpeedGates(t *testing.T) {
	for _, tc := range []struct {
		took, limit time.Duration
		slow        bool
	}{
		{100 * time.Millisecond, scenario.MaxScenario, false},
		{scenario.MaxScenario, scenario.MaxScenario, false},
		{scenario.MaxScenario + time.Millisecond, scenario.MaxScenario, true},
		{scenario.MaxSuite + time.Second, scenario.MaxSuite, true},
		{scenario.MaxSuite - time.Second, scenario.MaxSuite, false},
	} {
		if msg := scenario.TooSlow("x", tc.took, tc.limit); (msg != "") != tc.slow {
			t.Errorf("%v against %v: %q", tc.took, tc.limit, msg)
		}
	}
	if scenario.MaxScenario != 2*time.Second || scenario.MaxSuite != 15*time.Second {
		t.Error("the gates are D49's: 2 s a scenario, 15 s the suite")
	}
}

// A contended host warns instead of failing, up to a ceiling; a quiet host
// fails at the plain limit, so a real slowdown is caught (TUI-032).
func TestAContendedHostWarnsAndAQuietOneFails(t *testing.T) {
	const lim = scenario.MaxSuite
	for _, tc := range []struct {
		name       string
		took       time.Duration
		load       float64
		fail, warn bool
	}{
		{"within the limit", lim, 0.2, false, false},
		{"within the limit, contended", lim, 9, false, false},
		{"quiet host, 1.5x", lim * 3 / 2, 0.2, true, false},
		{"quiet host, an injected sleep", lim + 10*time.Second, 0.3, true, false},
		{"lightly shared host", lim * 2, 0.4, true, false},
		{"contended host, 22 s", 22 * time.Second, 0.9, false, true},
		{"contended host, 5x", lim * 5, 0.9, false, true},
		{"contended host, past the ceiling", lim*5 + time.Second, 0.9, true, false},
		{"heavy load widens the ceiling", lim * 20, 8, false, true},
		{"heavy load, past 3x the load", lim * 25, 8, true, false},
	} {
		fail, warn := scenario.Judge(tc.took, lim, tc.load)
		if fail != tc.fail || warn != tc.warn {
			t.Errorf("%s: fail=%v warn=%v, want %v %v", tc.name, fail, warn, tc.fail, tc.warn)
		}
	}
	if f, w := scenario.Verdict("x", 22*time.Second, scenario.MaxSuite, 0.9); f != "" || w == "" {
		t.Errorf("a contended 22 s: failure %q warning %q", f, w)
	}
	if f, w := scenario.Verdict("x", 22*time.Second, scenario.MaxSuite, 0.3); f == "" || w != "" {
		t.Errorf("a quiet 22 s: failure %q warning %q", f, w)
	}
}
