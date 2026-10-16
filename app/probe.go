package app

import (
	"github.com/GeneStevens/seq-tui/model"
	"time"

	"github.com/GeneStevens/seq-tui/screen"
)

// Probe is a message that runs Fn inside the program's own loop, between two
// other messages, and then closes Done. A scenario sends one with
// Program.Send to read or wait on the App without a race.
type Probe struct {
	Fn   func(*App)
	Done chan struct{}
}

// Idle says every key press written so far has been handled and every command
// they sent has been answered. keys is how many presses were written.
func (a *App) Idle(keys int) bool { return a.keys >= keys && a.inflight == 0 }

// Notice is the answer to the last command, as the status line shows it.
func (a *App) Notice() (text string, bad bool) { return a.view.Notice, a.view.NoticeBad }

// Plain draws the screen at w x h without colour. The "heard ago" clock stands
// at 400 ms, as in the golden screens of recorded logs, so a screen does not
// depend on how fast the test ran.
func (a *App) Plain(w, h int) string { return a.render(w, h, 400*time.Millisecond, screen.Plain()) }

// Online says the screen shows the connection as up.
func (a *App) Online() bool { return a.view.Conn == screen.ConnOnline && !a.retrying }

// PlanLines is the plan being built, one line a step, as the Round panel lists it.
func (a *App) PlanLines() []string {
	var lines []string
	a.sess.View(func(m *model.Model) { lines = a.draft.Lines(m) })
	return lines
}
