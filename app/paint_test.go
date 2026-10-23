package app_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// The screen is drawn once for what changed, not once for each message: a motion
// of the pointer within the cell it is in changes nothing (T4).
func TestHoverDrawsOnlyWhenTheCellChanges(t *testing.T) {
	a, _ := benchProgram()
	for _, tc := range []struct {
		name  string
		msg   tea.Msg
		draws int
	}{
		{"onto a cell", tea.MouseMotionMsg{X: 30, Y: 12}, 1},
		{"again, on the same cell", tea.MouseMotionMsg{X: 30, Y: 12}, 0},
		{"and again", tea.MouseMotionMsg{X: 30, Y: 12}, 0},
		{"to the next cell", tea.MouseMotionMsg{X: 31, Y: 12}, 1},
		{"with Alt held, labels show", tea.MouseMotionMsg{X: 31, Y: 12, Mod: tea.ModAlt}, 1},
		{"a key", tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}), 1},
		{"a resize", tea.WindowSizeMsg{Width: 120, Height: 40}, 1},
	} {
		before := a.Draws()
		a.Update(tc.msg)
		a.View()
		a.View() // a second look at the same screen draws nothing
		if got := a.Draws() - before; got != tc.draws {
			t.Errorf("%s: %d draws, want %d", tc.name, got, tc.draws)
		}
	}
}

// A click is read against the geometry of the screen last drawn: it does not draw
// the screen to find it, and the one draw after it is the screen's own (T4).
func TestAClickReusesTheGeometryOfTheLastDraw(t *testing.T) {
	a, _ := benchProgram()
	before := a.Draws()
	a.Update(tea.MouseClickMsg{X: 60, Y: 12, Button: tea.MouseLeft})
	a.View()
	if got := a.Draws() - before; got != 1 {
		t.Errorf("a click took %d draws, want the one after it", got)
	}
	// What changed since the last draw makes the geometry old: it is drawn again.
	a.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	before = a.Draws()
	a.Update(tea.MouseClickMsg{X: 40, Y: 10, Button: tea.MouseLeft})
	if got := a.Draws() - before; got != 1 {
		t.Errorf("a click on a screen not yet drawn took %d draws, want 1 to find it", got)
	}
}
