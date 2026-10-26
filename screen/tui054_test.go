package screen

import "testing"

// A click within one cell of a mob or person counts as a click on it; the
// nearest when there are several; never a drop, a door or a corpse (D109).
func TestNearMiss(t *testing.T) {
	ts := []thing{
		{c: 10, r: 5, pick: Pick{Kind: PickMob, ID: "a"}},
		{c: 12, r: 5, pick: Pick{Kind: PickMob, ID: "b"}},
		{c: 20, r: 5, pick: Pick{Kind: PickNPC, ID: "n"}},
		{c: 30, r: 5, pick: Pick{Kind: PickDrop, ID: "d"}},
		{c: 40, r: 5, pick: Pick{Kind: PickMob, ID: "dead"}, dead: true},
	}
	for _, tc := range []struct {
		name string
		c, r int64
		want string // "" for none
	}{
		{"one cell left of a mob", 9, 5, "a"},
		{"diagonal", 11, 6, "a"},
		{"between two: the nearer, a tie going to the first", 11, 5, "a"},
		{"nearer to the second", 13, 4, "b"},
		{"two cells away", 7, 5, ""},
		{"a person", 21, 5, "n"},
		{"beside a drop", 31, 5, ""},
		{"beside a corpse", 41, 5, ""},
		{"empty ground", 0, 0, ""},
	} {
		got, ok := nearMiss(ts, tc.c, tc.r)
		if (tc.want == "") == ok || got.pick.ID != tc.want {
			t.Errorf("%s: got %q (%v), want %q", tc.name, got.pick.ID, ok, tc.want)
		}
	}
}
