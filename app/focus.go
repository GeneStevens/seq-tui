package app

import (
	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
)

// Directing the account's second character (D56, Pillar 10): one TUI session
// plays both. X switches the focus to the other character; while it has the
// focus the screen is its screen (its hit points, abilities, round and plan)
// and every key is its key, sent by this session as a command for that
// character. The stance that drives it waits meanwhile: Z hands it back.
//
//	X  switch to the other character, and back
//	Z  (directing) hand it back to its stance; H the next stance

// focusSess is the session the screen shows and the keys read: the player's
// own, or the second character's while it is directed.
func (a *App) focusSess() *client.Session {
	if a.focus {
		if s := a.duo.Session(); s != nil {
			return s
		}
	}
	return a.sess
}

// checkFocus gives the focus back when the second character is no longer in
// play.
func (a *App) checkFocus() {
	if a.focus && (a.duo == nil || !a.duo.Active()) {
		a.setFocus(false)
	}
}

// setFocus moves the focus; each character keeps its own plan for the round.
func (a *App) setFocus(on bool) {
	if a.focus == on {
		return
	}
	a.focus = on
	a.draft, a.draftAlt = a.draftAlt, a.draft
	if on {
		a.focusCmd = false
		a.duo.SetDirected(true)
		return
	}
	if !a.focusCmd { // nothing was ordered: its stance has not been interrupted
		a.duo.SetDirected(false)
	}
}

// duoName is what the second character is called.
func (a *App) duoName() string {
	name := a.duo.Character()
	if s := a.duo.Session(); s != nil {
		s.View(func(m *model.Model) {
			if m.Self.Name != "" {
				name = m.Self.Name
			}
		})
	}
	return name
}

// focusAction is the meaning of X.
func (a *App) focusAction() Action {
	switch {
	case a.duo == nil:
		return Action{Note: "there is no second character to direct"}
	case !a.focus && !a.duo.Active():
		return Action{Note: "the second character is not in play: P brings it in"}
	}
	return Action{Kind: Focus}
}
