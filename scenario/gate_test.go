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
