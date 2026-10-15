package app

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
)

const (
	commandTimeout = 5 * time.Second
	redraw         = 250 * time.Millisecond
	retryEvery     = 2 * time.Second
	pingEvery      = 8 // redraws between pings: every two seconds
)

// App is the Bubble Tea model. The game state lives in the client's model; App
// holds only what the screen needs besides it.
type App struct {
	sess *client.Session

	w, h     int
	view     screen.View
	watch    <-chan struct{}
	lastTick time.Time
	lastN    uint64
	retrying bool
	pings    int
	Err      error // why the program ended, if it ended badly
}

// New returns the program's model for a session that has logged in.
func New(sess *client.Session) *App {
	return &App{sess: sess, w: screen.MinWidth, h: screen.MinHeight, view: screen.View{Conn: screen.ConnOnline}}
}

type (
	changedMsg struct{}
	redrawMsg  struct{}
	resultMsg  struct {
		note string
		bad  bool
	}
	reconnected struct{ err error }
)

func (a *App) Init() tea.Cmd {
	a.watch = a.sess.Changed()
	a.observe()
	return tea.Batch(a.waitChange(), tea.Tick(redraw, func(time.Time) tea.Msg { return redrawMsg{} }))
}

func (a *App) waitChange() tea.Cmd {
	ch := a.watch
	return func() tea.Msg { <-ch; return changedMsg{} }
}

// observe notes when the server's clock last moved: the "last tick" on the
// status line.
func (a *App) observe() {
	a.sess.View(func(m *model.Model) {
		if !a.view.HasTick || m.Heard != a.lastN {
			a.lastTick, a.lastN, a.view.HasTick = time.Now(), m.Heard, true
		}
	})
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.w, a.h = msg.Width, msg.Height
	case redrawMsg:
		cmds := []tea.Cmd{tea.Tick(redraw, func(time.Time) tea.Msg { return redrawMsg{} })}
		if a.pings++; a.pings%pingEvery == 0 {
			cmds = append(cmds, a.ping)
		}
		return a, tea.Batch(cmds...)
	case changedMsg:
		a.watch = a.sess.Changed() // before reading, so no change is missed
		a.observe()
		cmds := []tea.Cmd{a.waitChange()}
		if closed, _ := a.sess.Closed(); closed {
			if a.sess.LoggedOut() {
				a.view.Conn = screen.ConnLoggedOut
				return a, tea.Quit
			}
			a.view.Conn = screen.ConnReconnecting
			if !a.retrying {
				a.retrying = true
				cmds = append(cmds, a.reconnect(0))
			}
		}
		return a, tea.Batch(cmds...)
	case reconnected:
		if msg.err != nil {
			a.view.Conn, a.view.Notice, a.view.NoticeBad = screen.ConnReconnecting, "reconnect failed: "+msg.err.Error(), true
			return a, a.reconnect(retryEvery)
		}
		a.retrying, a.view.Conn, a.view.Notice, a.view.NoticeBad = false, screen.ConnOnline, "reconnected", false
	case resultMsg:
		a.view.Notice, a.view.NoticeBad = msg.note, msg.bad
	case tea.KeyPressMsg:
		return a.key(msg.String())
	}
	return a, nil
}

// ping asks the server for a pong; a failure shows as a closed connection.
func (a *App) ping() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	_ = a.sess.Ping(ctx)
	return nil
}

func (a *App) reconnect(after time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(after)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return reconnected{a.sess.Reconnect(ctx)}
	}
}

func (a *App) key(k string) (tea.Model, tea.Cmd) {
	var act Action
	a.sess.View(func(m *model.Model) { act = Plan(k, m) })
	a.view.Notice, a.view.NoticeBad = "", false
	switch act.Kind {
	case Quit:
		return a, tea.Quit
	case ToggleHelp:
		a.view.Help = !a.view.Help
	case ToggleVerbose:
		a.view.Verbose = !a.view.Verbose
	case None:
		a.view.Notice = act.Note
	default:
		return a, a.send(act)
	}
	return a, nil
}

// send sends the intent for act and waits for the server's answer, which
// becomes the notice on the status line.
func (a *App) send(act Action) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		defer cancel()
		var (
			id   string
			err  error
			what string
		)
		switch act.Kind {
		case Move:
			what = "move"
			id, err = a.sess.Move(ctx, act.X, act.Y)
		case Target:
			what = "target"
			id, err = a.sess.Target(ctx, act.ID)
		case Attack:
			what = "attack"
			id, err = a.sess.Attack(ctx, act.On)
		case Loot:
			what = "loot"
			id, err = a.sess.Loot(ctx, act.ID)
		case UseDoor:
			what = "use door"
			id, err = a.sess.UseDoor(ctx, act.ID)
		case Release:
			what = "release"
			id, err = a.sess.Release(ctx)
		case Logout:
			if err := a.sess.Logout(ctx); err != nil {
				return resultMsg{"logout failed: " + err.Error(), true}
			}
			return resultMsg{"logging out…", false}
		}
		if err != nil {
			return resultMsg{what + " not sent: " + err.Error(), true}
		}
		st, err := a.sess.Result(ctx, id)
		switch {
		case err != nil:
			return resultMsg{what + ": " + err.Error(), true}
		case st.State == model.IntentDenied:
			return resultMsg{fmt.Sprintf("%s denied: %s", what, st.Reason), true}
		case st.Unknown:
			return resultMsg{what + ": already applied (result unknown)", false}
		}
		return resultMsg{what + " ok", false}
	}
}

func (a *App) View() tea.View {
	v := a.view
	v.Since = time.Since(a.lastTick)
	var out string
	a.sess.View(func(m *model.Model) { out = screen.Render(m, v, a.w, a.h, screen.ANSI()) })
	tv := tea.NewView(out)
	tv.AltScreen = true
	tv.WindowTitle = "SEQ"
	return tv
}
