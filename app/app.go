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
	menu     *screen.Menu // the instance menu, nil when closed
	draft    Draft        // the plan being built in a turn-based instance
	Err      error        // why the program ended, if it ended badly
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
		// planSent: the note is the receipt for the plan of this round and edit.
		planSent  bool
		planRound uint64
		planVer   int
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
		if msg.planSent && msg.planRound == a.draft.Round && msg.planVer == a.draft.Ver {
			a.draft.Sent = true
		}
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

// sync drops a plan that belongs to a round that is over: a new round has
// opened, or the character is no longer in a round.
func (a *App) sync(m *model.Model) {
	switch {
	case m.Round == nil:
		if a.draft.Round != 0 || len(a.draft.Steps) > 0 {
			a.draft = Draft{Ver: a.draft.Ver + 1}
		}
	case m.Round.Open && m.Round.Number != a.draft.Round:
		a.draft = Draft{Round: m.Round.Number, Ver: a.draft.Ver + 1}
	}
}

func (a *App) key(k string) (tea.Model, tea.Cmd) {
	var act Action
	a.sess.View(func(m *model.Model) {
		a.sync(m)
		switch {
		case a.menu != nil:
			act = a.menuKey(k, m)
		case m.TurnBased():
			act = TurnKey(k, m, &a.draft)
		default:
			act = Plan(k, m)
		}
	})
	a.view.Notice, a.view.NoticeBad = "", false
	switch act.Kind {
	case Quit:
		return a, tea.Quit
	case ToggleHelp:
		a.view.Help = !a.view.Help
	case ToggleVerbose:
		a.view.Verbose = !a.view.Verbose
	case OpenMenu:
		a.menu = &screen.Menu{}
	case PlanAdd:
		if !a.draft.Add(*act.Step) {
			a.view.Notice, a.view.NoticeBad = fmt.Sprintf("the plan is full (%d steps): Enter submits it, Backspace removes a step", screen.MaxPlanSteps), true
		}
	case PlanUndo:
		if !a.draft.Undo() {
			a.view.Notice = "the plan is empty"
		}
	case PlanClear:
		if !a.draft.Clear() {
			a.view.Notice = "the plan is empty"
		}
	case None:
		a.view.Notice = act.Note
	default:
		if act.Kind == PlanSubmit {
			act.Round, act.Ver, act.Steps = a.draft.Round, a.draft.Ver, a.draft.Agent()
		}
		return a, a.send(act)
	}
	return a, nil
}

// menuKey handles a key while the instance menu is open.
func (a *App) menuKey(k string, m *model.Model) Action {
	n := len(m.Presets)
	switch k {
	case "up", "w", "k":
		a.menu.Sel = max(a.menu.Sel-1, 0)
	case "down", "s", "j":
		a.menu.Sel = min(a.menu.Sel+1, max(n-1, 0))
	case "esc", "i", "q":
		a.menu = nil
	case "ctrl+c":
		return Action{Kind: Quit}
	case "enter", "space", " ":
		if a.menu.Sel < n {
			id := m.Presets[a.menu.Sel].GetId()
			a.menu = nil
			return Action{Kind: OpenInstance, ID: id}
		}
		return Action{Note: "this zone offers no instances"}
	default:
		if len(k) == 1 && k[0] >= '1' && k[0] <= '9' {
			if i := int(k[0] - '1'); i < n {
				a.menu.Sel = i
			}
		}
	}
	return Action{}
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
		case Stop:
			what = "stop"
			id, err = a.sess.Stop(ctx)
		case OpenInstance:
			what = "open instance"
			id, err = a.sess.OpenInstance(ctx, act.ID)
		case PlanSubmit:
			what = "plan"
			id, err = a.sess.SubmitPlan(ctx, act.Steps...)
		case Logout:
			if err := a.sess.Logout(ctx); err != nil {
				return resultMsg{note: "logout failed: " + err.Error(), bad: true}
			}
			return resultMsg{note: "logging out…"}
		}
		if err != nil {
			return resultMsg{note: what + " not sent: " + err.Error(), bad: true}
		}
		st, err := a.sess.Result(ctx, id)
		switch {
		case err != nil:
			return resultMsg{note: what + ": " + err.Error(), bad: true}
		case st.State == model.IntentDenied:
			return resultMsg{note: fmt.Sprintf("%s denied: %s", what, model.Words(st.Reason)), bad: true}
		case st.Unknown && act.Kind == PlanSubmit:
			return resultMsg{note: "plan: already applied (result unknown)", planSent: true, planRound: act.Round, planVer: act.Ver}
		case st.Unknown:
			return resultMsg{note: what + ": already applied (result unknown)"}
		case act.Kind == PlanSubmit:
			return resultMsg{note: "plan accepted, counts even if you disconnect", planSent: true, planRound: act.Round, planVer: act.Ver}
		case act.Kind == OpenInstance:
			return resultMsg{note: "instance requested; the log says when it is open"}
		}
		return resultMsg{note: what + " ok"}
	}
}

func (a *App) View() tea.View {
	v := a.view
	v.Since = time.Since(a.lastTick)
	var out string
	v.Menu, v.Now = a.menu, time.Now()
	a.sess.View(func(m *model.Model) {
		a.sync(m)
		if m.TurnBased() {
			v.Plan = screen.Plan{Steps: a.draft.Lines(m), Sent: a.draft.Sent, Dest: a.draft.Dest()}
		}
		out = screen.Render(m, v, a.w, a.h, screen.ANSI())
	})
	tv := tea.NewView(out)
	tv.AltScreen = true
	tv.WindowTitle = "SEQ"
	return tv
}
