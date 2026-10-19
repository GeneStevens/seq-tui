package app

import (
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
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
// It also says the program has seen the last thing the server sent (what it does
// by itself at the sight of a message, such as asking after a party's members,
// has begun).
func (a *App) Idle(keys int) bool {
	seen := false
	a.sess.View(func(m *model.Model) { seen = m.Heard == a.lastN })
	return a.keys >= keys && a.inflight == 0 && seen
}

// Commanding says a command the program sent has not been answered: it is
// waiting for the server's result.
func (a *App) Commanding() bool { return a.inflight > 0 }

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
	a.focusSess().View(func(m *model.Model) { lines = a.draft.Lines(m) })
	return lines
}

// CellOf says where the screen draws the world place p (millimetres), after
// drawing it now: a scenario clicks there.
func (a *App) CellOf(p *pb.Pos) (x, y int, ok bool) {
	a.refresh()
	pt, ok := a.mouse.hits.CellOf(p)
	return pt.X, pt.Y, ok
}

// MouseState is what the mouse has set: the zoom of the map and of the large map,
// the id whose card is open, whether everything is labelled, the sidebar's tab and
// whether its drawer is open.
func (a *App) MouseState() (zoom, bigZoom int, card string, labels bool, tab int, drawer bool) {
	ms := &a.mouse
	return ms.zoom, ms.bigZoom, ms.card, ms.alt || ms.labels, ms.side.Tab, ms.side.Drawer
}

// Resize tells the program the window is w x h, as the terminal does.
func (a *App) Resize(w, h int) { a.w, a.h = w, h }
