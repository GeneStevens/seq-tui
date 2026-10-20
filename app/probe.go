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
	return a.keys >= keys && a.inflight == 0 && a.walk.queued.Load() == 0 && seen
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

// Frame runs what a frame tick does, once, at the program's clock: the walker
// says what to send, and the mark moves. A scenario's clock is the server's and
// moves only when the scenario says, so it asks for the frame itself.
//
// It reports whether it sent a movement intent, and whether a key is down or a walk
// is not yet stopped.
func (a *App) Frame() (sent, active bool) {
	if !a.walk.w.Active() && a.walk.queued.Load() == 0 {
		return false, false
	}
	a.walk.framing = false
	_ = a.walkPoll()
	a.walk.framing = false
	a.sess.View(func(m *model.Model) { a.drawAhead(m)() }) // the draw the renderer would make
	return a.walk.queued.Load() > 0, a.walk.w.Active()
}

// KeyboardEvents tells the program the terminal reports key releases and
// repeats, as a terminal with the kitty keyboard protocol answers when asked.
func (a *App) KeyboardEvents(on bool) { a.walk.w.Releases = on }

// Walking says the walker has told the server to walk and not yet to stop.
func (a *App) Walking() bool { return a.walk.w.Walking() }

// WalkSends lists every movement intent (Walk or Stop) the program has sent for
// held keys, with the time of each.
func (a *App) WalkSends() []WalkSend { return append([]WalkSend(nil), a.walk.sends...) }

// Drawn is where the screen draws the character now: the predicted mark, which
// runs ahead of the server's position while a key is held.
func (a *App) Drawn() (x, y int64) {
	a.sess.View(func(m *model.Model) {
		restore := a.drawAhead(m)
		x, y = m.Self.Pos.GetX(), m.Self.Pos.GetY()
		restore()
	})
	return
}
