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
	if scenario.MaxScenario != 2*time.Second || scenario.MaxSuite != 25*time.Second {
		t.Error("the gates are D49's 2 s a scenario, and 25 s the suite: about twice its time on an idle host (T8)")
	}
}

// The limit grows with the load, a contended host warns instead of failing up
// to a ceiling, and a quiet host fails at the plain limit, so a real slowdown is
// caught (TUI-032, T8 in TUI-045).
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
		{"idle host, twice the idle time", lim, 0, false, false},
		{"idle host, 1.5x the limit", lim * 3 / 2, 0.05, true, false},
		{"quiet host, an injected sleep", lim + 10*time.Second, 0.1, true, false},
		{"T8: 18 s at a load of 0.37 per CPU, old limit 15 s", 18 * time.Second, 0.37, false, false},
		{"moderately busy host, stretched limit", lim * 13 / 10, 0.37, false, false},
		{"moderately busy host, twice the limit", lim * 2, 0.4, true, false},
		{"contended host, 1.8x is within the stretched limit", lim * 18 / 10, 0.9, false, false},
		{"contended host, 2.5x", lim * 5 / 2, 0.9, false, true},
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
	if f, w := scenario.Verdict("x", 2*lim, lim, 0.9); f != "" || w == "" {
		t.Errorf("a contended double: failure %q warning %q", f, w)
	}
	if f, w := scenario.Verdict("x", 2*lim, lim, 0.3); f == "" || w != "" {
		t.Errorf("a quiet double: failure %q warning %q", f, w)
	}
	if f, w := scenario.Verdict("x", lim+time.Second, lim, 0.3); f != "" || w != "" {
		t.Errorf("a slightly slow run on a busy host: failure %q warning %q", f, w)
	}
}
