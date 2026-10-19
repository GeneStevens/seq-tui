package app

import (
	"testing"

	"github.com/GeneStevens/seq-tui/screen"
)

// Every mouse action has a key (D77): the game is fully playable without a mouse.
// Each row names what the mouse does, and the key whose Action is the same.
func TestEveryMouseActionHasAKey(t *testing.T) {
	m := bindingModel(t)
	for _, tc := range []struct {
		mouse, key string
		want       Kind
	}{
		{"click the ground: walk", "up", Move},
		{"click a mob: select it", "tab", Target},
		{"click it again: attack", "f", Attack},
		{"click a person again: talk", "g", Talk},
		{"click loot: loot all", "l", Loot},
		{"scroll up: zoom in", "+", ZoomMap},
		{"scroll down: zoom out", "-", ZoomMap},
		{"hold Alt: label everything", "ctrl+a", ToggleLabels},
		{"click the large map: walk (arrows and Enter)", "m", OpenOverview},
	} {
		a := Plan(tc.key, m)
		switch {
		case zero(a):
			t.Errorf("%s: key %q does nothing", tc.mouse, tc.key)
		case a.Kind != None && a.Kind != tc.want:
			t.Errorf("%s: key %q is %+v, want kind %d", tc.mouse, tc.key, a, tc.want)
		}
		// A key that needs something in view (a target, a person, a drop) says so in a
		// note when there is nothing: it is bound all the same.
	}
	// The keys are in the help.
	listed := screen.HelpBindingKeys()
	for _, k := range []string{"+", "-", "ctrl+a", "tab", "f", "g", "l", "m", "esc"} {
		found := false
		for _, l := range listed {
			found = found || l == k
		}
		if !found {
			t.Errorf("key %q is not in the help's table", k)
		}
	}
}
