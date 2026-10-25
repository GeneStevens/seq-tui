package app

import (
	"context"
	"fmt"
	"os"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	"github.com/GeneStevens/seq-tui/sessionlog"
	"github.com/GeneStevens/seq/agentkit"
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
	// paint is what the program keeps so that it draws once for what changed,
	// not once for each message it is sent (D89, T4): see Update and View.
	paint    paintState
	retrying bool
	pings    int
	menu     *screen.Menu   // the instance menu, nil when closed
	slots    *screen.Slots  // the slot screen, nil when closed
	create   *screen.Create // character creation, nil once a character is chosen
	picker   *screen.Picker // the character picker, nil once a character is chosen or made
	back     *screen.Picker // the picker the creation screen came from (Esc goes back)
	char     bool           // the character view is open (C)
	traders  *screen.Traders
	trade    *screen.TradeUI
	talk     *screen.Talk // the conversation, nil when closed
	board    *screen.Board
	cons     *screen.ContractsUI
	role     *screen.Role       // the choice of the recruit's role, nil when closed
	duo      *client.Duo        // the second character on a stance, nil when there is none
	toDuo    bool               // the helper keys talk to it, not to the house companion
	focus    bool               // directing the second character: the keys and the screen are its (D56)
	focusCmd bool               // a command went to it since the focus changed
	draftAlt Draft              // the other character's plan while its focus is away
	overview bool               // the whole-zone map
	line     *screen.Line       // the command line, nil when closed
	soc      *screen.SocialUI   // the Social screen, nil when closed
	ses      *screen.SessionsUI // the sessions view, nil when closed
	quest    *screen.QuestUI    // the quest chain screen, nil when closed
	replay   *replay            // a recording drawn in place of the screen, nil when none
	mouse    mouseState         // the pointer, the zoom, the card and the sidebar (D77, D78)
	log      *sessionlog.Log    // this session's own log, nil when recording is off
	stopTold bool               // the recording's stop has been put in the event log
	logRoot  string             // the directory of the player's session logs
	bannerN  int                // the announcement the banner is for, and when it came
	bannerAt time.Time
	inviteN  int // the invitation the prompt is for, and when it came
	inviteAt time.Time
	gameAt   time.Time // when the model's game time last moved
	gameMs   int64
	draft    Draft // the plan being built in a turn-based instance
	Err      error // why the program ended, if it ended badly

	// Clock tells the time the screen's countdowns and "heard ago" read; nil is
	// the wall clock. A scenario sets it to its server's clock.
	Clock func() time.Time

	// Bell rings the terminal's bell when a mob begins attacking the character
	// (D75). It is off by default.
	Bell   bool
	alerts int // the alerts the bell has rung for

	// NoMouse stops the program asking the terminal for mouse reports (D77).
	NoMouse bool

	// LogLabel, if set, is what the sessions view says of where the logs live,
	// in place of the directory (a scenario's is a temporary one).
	LogLabel string

	chat     chatState  // chat on the screen: the drawer, the toast, the bubbles (D101)
	walk     *walkState // hold-to-walk: the keys down, the intents, the predicted mark
	keys     int        // key presses handled, and commands sent and answered: what a scenario waits on
	inflight int
}

// New returns the program's model for a session that has logged in.
func New(sess *client.Session) *App {
	return &App{sess: sess, w: screen.MinWidth, h: screen.MinHeight, view: screen.View{Conn: screen.ConnOnline}, walk: newWalkState()}
}

// NewCreator returns the program's model for a session that is connected and
// has not chosen a character: it opens on the creation screen, and plays the
// character once it is made.
func NewCreator(sess *client.Session) *App {
	a := New(sess)
	a.create = &screen.Create{}
	return a
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
		// talk: the talk was taken; open the conversation with this NPC.
		talk string
		// quiet: the answer to a request that shows nothing. after runs on the
		// program; clearInvite drops the invitation the prompt shows.
		quiet       bool
		after       func(*App)
		clearInvite bool
	}
	reconnected struct{ err error }
	// created answers a character creation: err is nil when the character was
	// made and chosen.
	created struct{ err error }
)

// now is the time the screen reads.
func (a *App) now() time.Time {
	if a.Clock != nil {
		return a.Clock()
	}
	return time.Now()
}

func (a *App) Init() tea.Cmd {
	a.watch = a.sess.Changed()
	a.sess.View(func(m *model.Model) { a.chat.seen, a.chat.init = m.Social.ChatN, true })
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
			a.lastTick, a.lastN, a.view.HasTick = a.now(), m.Heard, true
		}
		a.noteLog(m)
		if m.GameTimeMs != a.gameMs {
			a.gameMs, a.gameAt = m.GameTimeMs, a.now()
		}
	})
}

// paintState says whether the screen must be drawn again, and what was last
// drawn. A message changes the screen unless it is the pointer coming to rest
// on the cell it was already on, so only that kind is let through without a
// repaint (a motion is sent for every cell the pointer crosses, and a terminal
// sends many for each).
type paintState struct {
	dirty     bool // something changed since the last View
	shown     tea.View
	haveShown bool
	// hitsStale: what the map's last drawing recorded of where things are is
	// older than the last message, or was made for another size.
	hitsStale bool
	hitsW     int
	hitsH     int
	draws     int // how many times the screen has been drawn, by View or for a click
}

// Update handles a message. Whatever it is, the next View draws the screen
// again, unless the message cannot have changed it.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if a.restsOnTheSameCell(msg) {
		return a.update(msg)
	}
	a.paint.dirty = true
	if _, ok := msg.(tea.MouseMsg); ok {
		// A click is read against the screen the player sees, which is the last
		// one drawn: its geometry goes stale once the click is handled.
		defer func() { a.paint.hitsStale = true }()
		return a.update(msg)
	}
	a.paint.hitsStale = true
	return a.update(msg)
}

// restsOnTheSameCell says msg is a pointer motion that leaves the hovered cell
// and the Alt key as they were: nothing on the screen depends on it (T4).
func (a *App) restsOnTheSameCell(msg tea.Msg) bool {
	mv, ok := msg.(tea.MouseMotionMsg)
	if !ok || a.replay != nil {
		return ok
	}
	h := a.mouse.hover
	mo := mv.Mouse()
	return h != nil && h.X == mo.X && h.Y == mo.Y && a.mouse.alt == mo.Mod.Contains(tea.ModAlt)
}

func (a *App) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.w, a.h = msg.Width, msg.Height
	case redrawMsg:
		cmds := []tea.Cmd{tea.Tick(redraw, func(time.Time) tea.Msg { return redrawMsg{} })}
		if a.pings++; a.pings%pingEvery == 0 {
			cmds = append(cmds, a.ping)
		}
		cmds = append(cmds, a.walkTick())
		return a, tea.Batch(cmds...)
	case frameMsg:
		return a, a.walkFrame()
	case tea.KeyboardEnhancementsMsg:
		a.walk.w.Releases = msg.SupportsEventTypes()
	case tea.BlurMsg:
		return a, a.stopWalking()
	case tea.KeyReleaseMsg:
		a.keys++
		if MoveKey(msg.String()) {
			return a, a.walkRelease(msg.String())
		}
	case changedMsg:
		a.watch = a.sess.Changed() // before reading, so no change is missed
		a.observe()
		cmds := []tea.Cmd{a.waitChange()}
		cmds = append(cmds, a.walkTick())
		a.sess.View(a.watchInvite)
		a.sess.View(a.watchBanner)
		a.sess.View(a.watchChat)
		a.sess.View(func(m *model.Model) {
			a.walkedAway(m)
			if a.ring(m) {
				cmds = append(cmds, tea.Raw("\a"))
			}
		})
		if closed, _ := a.sess.Closed(); closed {
			if a.sess.LoggedOut() {
				a.view.Conn = screen.ConnLoggedOut
				a.closeWalk()
				return a, tea.Quit
			}
			a.view.Conn = screen.ConnReconnecting
			a.sess.View(func(m *model.Model) { a.view.Cut = m.Cut })
			if !a.retrying {
				a.retrying = true
				cmds = append(cmds, a.reconnect(0))
			}
		}
		return a, tea.Batch(cmds...)
	case reconnected:
		if msg.err != nil {
			a.view.Conn, a.view.Notice, a.view.NoticeBad = screen.ConnReconnecting, "reconnect failed: "+msg.err.Error(), true
			a.sess.View(func(m *model.Model) {
				if a.view.Cut = m.Cut; m.Cut != "" {
					a.view.Notice = m.Cut // the server's words, not the toolkit's
				}
			})
			return a, a.reconnect(retryEvery)
		}
		a.retrying, a.view.Conn, a.view.Notice, a.view.NoticeBad = false, screen.ConnOnline, "reconnected", false
		a.view.Cut = ""
		a.sess.View(func(m *model.Model) { m.ClearCut() })
	case sessionsLoaded, logLoaded:
		a.sessionsMsg(msg)
	case Probe:
		msg.Fn(a)
		close(msg.Done)
	case picked:
		a.pickedMsg(msg)
	case created:
		a.inflight--
		switch {
		case msg.err == nil:
			a.create, a.back = nil, nil
			a.view.Notice, a.view.NoticeBad = "", false
		default:
			a.create.Busy, a.create.Err = false, createError(msg.err)
		}
	case resultMsg:
		a.inflight--
		if msg.quiet {
			return a, nil
		}
		a.view.Notice, a.view.NoticeBad = msg.note, msg.bad
		if msg.clearInvite {
			a.sess.View(func(m *model.Model) { m.ClearInvite() })
		}
		if msg.after != nil {
			msg.after(a)
		}
		if msg.talk != "" {
			a.talk = &screen.Talk{NPC: msg.talk}
		}
		if msg.planSent && msg.planRound == a.draft.Round && msg.planVer == a.draft.Ver {
			a.draft.Sent = true
		}
	case tea.MouseMsg:
		a.keys++
		return a.onMouse(msg)
	case tea.KeyPressMsg:
		a.keys++
		a.mouse.alt = false // a key press: Alt held for the mouse is not known any more
		if a.line != nil && !a.view.Help {
			return a.lineKey(msg)
		}
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
	a.checkFocus()
	a.sess.View(func(m *model.Model) { m.ClearBanner() }) // a key dismisses a server announcement
	a.focusSess().View(func(m *model.Model) {
		a.sync(m)
		switch {
		case a.view.Help:
			act = a.helpKey(k, m)
		case k == "?" || k == "f1":
			act = Action{Kind: ToggleHelp} // the help opens over every view
		case a.replay != nil:
			act = a.replayKey(k)
		case a.chat.card != nil:
			act = a.personKey(k)
		case a.ses != nil:
			act = a.sessionsKey(k)
		case a.picker != nil:
			act = a.pickerKey(k)
		case a.create != nil:
			act = a.createKey(k, m)
		case a.soc != nil:
			act = a.socialKey(k, m)
		case a.sideKey(k, &act):
		case a.talk != nil:
			act = a.talkKey(k, m)
		case a.board != nil:
			act = a.boardKey(k, m)
		case a.role != nil:
			act = a.roleKey(k, m)
		case a.quest != nil:
			act = a.questKey(k, m)
		case a.cons != nil:
			act = a.contractsKey(k, m)
		case a.overview:
			act = a.overviewKey(k, m)
		case a.slots != nil:
			act = a.slotKey(k, m)
			if act.Kind == SetSlot && m.TurnBased() {
				// Between rounds a slot changes by the plan (D50).
				step := act
				act = Action{Kind: PlanAdd, Step: &step}
			}
		case a.char:
			act = a.charKey(k, m)
		case a.traders != nil:
			act = a.tradersKey(k, m)
		case a.trade != nil:
			act = a.tradeKey(k, m)
		case a.menu != nil:
			act = a.menuKey(k, m)
		case a.nameKey(k, &act):
		case a.drawer() && a.drawerKey(k, &act):
		case k == "esc" && a.mouse.card != "":
			act = Action{Kind: CloseCard}

		case m.Social.Invite != nil && (k == "y" || k == "n"):
			act = Action{Kind: SocialDo, ID: map[bool]string{true: "accept", false: "decline"}[k == "y"], Note: m.Social.Invite.From}
		case k == "x":
			act = a.focusAction()
		case helperKey(k):
			act = a.helperAction(k, m)
		case m.TurnBased():
			act = TurnKey(k, m, &a.draft)
		case MoveKey(k):
			act = Action{Kind: WalkKey, Note: k}
		default:
			act = Plan(k, m)
		}
	})
	return a.do(act)
}

// do carries out what a key or a click meant: it changes the screen's state, or
// sends the intent, and the server decides whether it is allowed.
func (a *App) do(act Action) (tea.Model, tea.Cmd) {
	a.view.Notice, a.view.NoticeBad = "", false
	switch act.Kind {
	case Quit:
		a.closeWalk()
		return a, tea.Quit
	case WalkKey:
		return a, a.walkKey(act.Note)
	case ZoomMap:
		a.zoomMap(act.Count)
	case ToggleLabels:
		a.mouse.labels = !a.mouse.labels
		a.view.Notice = map[bool]string{true: "labels on: everything on the map is named (Ctrl+A turns them off)", false: "labels off"}[a.mouse.labels]
	case CloseCard:
		a.mouse.card = ""
	case MoveCursor:
		a.moveCursor(act.Note)
	case SelectSide:
		a.activeSide().Tab = min(max(act.Count, 0), screen.Tabs-1)
	case ScrollSide:
		sb := a.activeSide()
		sb.Scroll[sb.Tab] = max(sb.Scroll[sb.Tab]+act.Count, 0)
	case ToggleHelp:
		a.view.Help, a.view.HelpPage, a.view.HelpScroll = true, 0, 0
	case ToggleVerbose:
		a.view.Verbose = !a.view.Verbose
	case OpenMenu:
		a.menu = &screen.Menu{}
	case OpenSlots:
		a.slots = &screen.Slots{}
	case OpenInventory:
		a.char = true
		a.sess.View(func(m *model.Model) {
			// It opens on the bags when they hold anything: the way to wear a find.
			sb := &a.mouse.charTab
			*sb = screen.Sidebar{}
			if len(m.Self.Inventory) > 0 {
				sb.Tab = screen.TabBags
			}
		})
	case OpenTraders:
		a.char, a.traders = false, &screen.Traders{}
	case OpenTradersAt:
		a.talk = nil
		a.sess.View(func(m *model.Model) {
			a.traders = &screen.Traders{Sel: max(tradersAt(m, act.NPC), 0), Open: true, NPC: act.NPC}
		})
	case OpenBoard:
		a.talk, a.board = nil, &screen.Board{NPC: act.NPC, Pane: act.Count}
	case OpenContracts:
		a.cons = &screen.ContractsUI{}
	case OpenOverview:
		a.overview, a.mouse.card = true, ""
	case OpenQuest:
		a.quest = &screen.QuestUI{}
	case OpenLine:
		a.line = &screen.Line{Text: "/"}
	case ToggleDrawer, PanelTab, SelectPanel, ReplyLine, NameStep, OpenPerson, PersonDo, PersonClose:
		return a.chatDo(act)
	case OpenSessions:
		return a, a.openSessions()
	case SessionsDo:
		return a, a.sessionsDo(act)
	case SocialDo:
		a.inflight++
		return a, a.send(act)
	case OpenRole:
		a.talk, a.role = nil, &screen.Role{NPC: act.NPC}
		a.sess.View(func(m *model.Model) {
			a.role.Sel = max(slices.Index(agentkit.CompanionRoles, m.Companion().GetRole()), 0)
		})
	case Focus:
		cmd := a.stopWalking() // the Stop goes to the character that was walking
		a.setFocus(!a.focus)
		if a.focus {
			a.view.Notice = "directing " + a.duoName() + ": X back, Z its stance"
		} else {
			a.view.Notice = "you are back on your own character"
		}
		return a, cmd
	case DuoSummon, DuoDismiss, DuoStance, DuoOrder:
		cmd, note := a.duoDo(act)
		a.view.Notice = note
		return a, cmd
	case OpenTrade:
		a.char, a.trade = false, &screen.TradeUI{}
		a.sess.View(func(m *model.Model) { a.trade.Who = askerIndex(m) })
	case PlanAdd:
		if why := a.draft.Add(*act.Step); why != "" {
			a.view.Notice, a.view.NoticeBad = why, true
		} else if act.Step.Kind == SetSlot {
			a.view.Notice = "slot change added to the plan: Esc closes this, Enter submits the plan"
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
	case PickCharacter:
		a.picker.Busy, a.picker.Err = true, ""
		a.inflight++
		return a, a.pickCmd(act.ID)
	case CreateCharacter:
		a.create.Busy, a.create.Err = true, ""
		a.inflight++
		return a, a.createCmd(act.Note, act.ID)
	case Logout:
		if a.focus {
			a.view.Notice = "X gives you back to yourself first: O logs yourself out"
			return a, nil
		}
		a.inflight++
		return a, a.send(act)
	default:
		if a.focus {
			act.For, a.focusCmd = a.duo.Character(), true
			a.duo.SetDirected(true)
		}
		if act.Kind == PlanSubmit {
			act.Round, act.Ver, act.Steps = a.draft.Round, a.draft.Ver, a.draft.Agent()
		}
		a.inflight++
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
		if act.For != "" {
			ctx = client.Directing(ctx, act.For)
		}
		hadTarget := true
		if act.For == "" {
			a.sess.View(func(m *model.Model) { hadTarget = m.Self.Target != "" })
		}
		var (
			id    string
			err   error
			what  string
			extra string // what to add to an "ok": the second character's refusal of a door
		)
		switch act.Kind {
		case SocialDo:
			return a.socialSend(ctx, act)
		case Move:
			what = "move"
			id, err = a.sess.Move(ctx, act.X, act.Y)
		case Target:
			what = "target"
			id, err = a.sess.Target(ctx, act.ID)
		case Attack:
			what = "attack"
			id, err = a.sess.Attack(ctx, act.On)
		case AutoRetaliate:
			what = "auto-retaliate"
			id, err = a.sess.AutoRetaliate(ctx, act.On)
		case Loot:
			what = "loot"
			id, err = a.sess.Loot(ctx, act.ID)
		case UseDoor:
			what = "use door"
			id, err = a.sess.UseDoor(ctx, act.ID)
			if err == nil && a.duo != nil && act.For == "" {
				// The second character takes the same door; the server decides.
				space := ""
				a.sess.View(func(m *model.Model) { space = m.Space })
				if why := a.duo.UseDoor(ctx, act.ID, space); why != "" {
					extra = "; second character: " + why
				}
			}
		case Release:
			what = "release"
			id, err = a.sess.Release(ctx)
		case UseAbility:
			what = act.Note
			if what == "" {
				what = "ability"
			}
			id, err = a.sess.UseAbility(ctx, act.Slot, act.ID)
		case SetSlot:
			what = fmt.Sprintf("slot %d", act.Slot)
			id, err = a.sess.SetSlot(ctx, act.Slot, act.ID)
		case QuestDismiss:
			what = "dismiss quest"
			id, err = a.sess.QuestDismiss(ctx, "")
		case Stop:
			what = "stop"
			id, err = a.sess.Stop(ctx)
		case OpenInstance:
			what = "open instance"
			id, err = a.sess.OpenInstance(ctx, act.ID)
		case PlanSubmit:
			what = "plan"
			id, err = a.sess.SubmitPlan(ctx, act.Steps...)
		case Talk, ContractAccept, ContractTurnIn, ContractAbandon:
			what, id, err = a.people(ctx, act)
		case CompanionSummon, CompanionDismiss, CompanionStance, CompanionRole, CompanionOrder:
			what, id, err = a.companion(ctx, act)
		case Equip, Unequip, Buy, Sell, SellJunk, Train, Bind, BankDeposit, BankWithdraw, TradeOpen, TradeAdd, TradeCrowns, TradeRemove, TradeConfirm, TradeCancel:
			what, id, err = a.economy(ctx, act)
		case Logout:
			if a.duo != nil && a.duo.Active() {
				_ = a.duo.Dismiss(ctx)
			}
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
			return resultMsg{note: deniedWords(st), bad: true} // the server's line, with no internal prefix
		case st.Unknown && act.Kind == PlanSubmit:
			return resultMsg{note: "plan: already applied (result unknown)", planSent: true, planRound: act.Round, planVer: act.Ver}
		case st.Unknown:
			return resultMsg{note: what + ": already applied (result unknown)"}
		case act.Kind == PlanSubmit:
			return resultMsg{note: "plan accepted, counts even if you disconnect", planSent: true, planRound: act.Round, planVer: act.Ver}
		case act.Kind == OpenInstance:
			return resultMsg{note: "instance requested; the log says when it is open"}
		case act.Kind == Talk:
			return resultMsg{note: what + " ok", talk: act.NPC}
		case act.Kind == QuestDismiss:
			return resultMsg{note: "quest chain dismissed for good; it will not be offered again"}
		case !hadTarget && act.For == "" && act.ID == "" && (act.Kind == Attack && act.On || act.Kind == UseAbility):
			// With no target the server picks what is attacking the character (D66).
			if name := a.sess.Picked(ctx); name != "" {
				return resultMsg{note: what + " ok: the server picked " + name + " as your target"}
			}
		}
		return resultMsg{note: what + " ok" + extra}
	}
}

// render draws the screen at w x h; since is how long ago the server was last
// heard.
func (a *App) render(w, h int, since time.Duration, st screen.Styler) string {
	if a.replay != nil {
		return a.replayView(w, h, st)
	}
	v := a.view
	v.Since = since
	var out string
	v.Menu, v.Slots, v.Create, v.Picker, v.Now = a.menu, a.slots, a.create, a.picker, a.now()
	v.Char, v.Traders, v.Trade = a.char, a.traders, a.trade
	v.Talk, v.Board, v.Contracts, v.Overview, v.Role = a.talk, a.board, a.cons, a.overview, a.role
	v.Line, v.Social, v.Sessions, v.Quest = a.line, a.soc, a.ses, a.quest
	a.chatView(&v)
	a.mouseView(&v)
	a.checkFocus()
	if a.focus {
		v.Directing = a.duoName()
	}
	if a.duo != nil && !a.focus {
		v.Duo = &screen.DuoView{ID: a.duo.Character(), Active: a.duo.Active(), Stance: a.duo.Stance(), Order: a.duo.Order().Kind, Directed: a.duo.Directed()}
		v.Duo.Plan, v.Duo.PlanRound = a.duo.Plan()
		if s := a.duo.Session(); s != nil {
			s.View(func(dm *model.Model) {
				v.Duo.Target = dm.Self.Target
				v.Duo.Mana.Cur, v.Duo.Mana.Max = dm.Self.Mana, dm.Self.MaxMana
			})
		}
	}
	a.focusSess().View(func(m *model.Model) {
		a.sync(m)
		defer a.drawAhead(m)()
		v.KeysToDuo = a.keysToDuo(m)
		if !a.gameAt.IsZero() && a.gameMs == m.GameTimeMs && a.Clock == nil && !m.TurnBased() { // between events the game time runs on
			v.GameNow = m.GameTimeMs + a.now().Sub(a.gameAt).Milliseconds()
		}
		if m.TurnBased() {
			v.Plan = screen.Plan{Steps: a.draft.Lines(m), Sent: a.draft.Sent, Dest: a.draft.Dest()}
			v.PlanSlots = a.draft.Slots()
		}
		out = screen.Render(m, v, w, h, st)
	})
	a.paint.hitsStale, a.paint.hitsW, a.paint.hitsH = false, w, h
	a.paint.draws++
	return out
}

func (a *App) View() tea.View {
	if !a.paint.dirty && a.paint.haveShown {
		return a.paint.shown // nothing has changed since it was drawn
	}
	a.paint.dirty = false
	// A scenario's clock is the server's and moves between messages; the program's
	// own draws (this one is made after every message) leave the walking mark
	// alone, and the scenario draws when it looks.
	a.walk.still = a.Clock != nil
	defer func() { a.walk.still = false }()
	tv := tea.NewView(a.render(a.w, a.h, a.now().Sub(a.lastTick), screen.Terminal(os.Getenv("NO_COLOR") != "")))
	tv.AltScreen = true
	if !a.NoMouse {
		tv.MouseMode = tea.MouseModeAllMotion // clicks, the wheel and hovering (D77)
	}
	tv.WindowTitle = "SEQ"
	// Key releases (the kitty keyboard protocol) end a walk the moment the key
	// comes up, where the terminal reports them; the focus tells a walk that its
	// keys are gone when the window loses it.
	tv.KeyboardEnhancements = tea.KeyboardEnhancements{ReportEventTypes: true}
	tv.ReportFocus = true
	a.paint.shown, a.paint.haveShown = tv, true
	return tv
}

// helpKey handles a key while the help is open: ? turns the page (and closes
// after the last), Left and Right turn it either way, Up and Down scroll.
func (a *App) helpKey(k string, m *model.Model) Action {
	v := &a.view
	last := screen.HelpPages() - 1
	maxScroll := screen.HelpMaxScroll(m, a.helpView(), a.w, a.h)
	switch k {
	case "?", "right", "pgdown", "n", "space", " ":
		if v.HelpPage >= last && k == "?" {
			v.Help = false
			break
		}
		v.HelpPage, v.HelpScroll = min(v.HelpPage+1, last), 0
	case "left", "pgup", "p", "backspace":
		v.HelpPage, v.HelpScroll = max(v.HelpPage-1, 0), 0
	case "up", "k":
		v.HelpScroll = max(v.HelpScroll-1, 0)
	case "down", "j":
		v.HelpScroll = min(v.HelpScroll+1, maxScroll)
	case "home":
		v.HelpScroll = 0
	case "end":
		v.HelpScroll = maxScroll
	case "esc", "q", "f1":
		v.Help = false
	case "ctrl+c":
		return Action{Kind: Quit}
	}
	return Action{}
}

// helpView is the view the help is drawn with, for the lines it will have.
func (a *App) helpView() screen.View {
	v := a.view
	if a.duo != nil {
		v.Duo = &screen.DuoView{}
	}
	if a.focus {
		v.Directing = a.duoName()
	}
	return v
}

// questKey handles a key on the quest chain screen. X asks the one question that
// dismisses the chain; Y answers it, anything else withdraws it.
func (a *App) questKey(k string, m *model.Model) Action {
	q := a.quest
	if q.Confirm {
		q.Confirm = false
		switch k {
		case "y":
			a.quest = nil
			return Action{Kind: QuestDismiss}
		case "ctrl+c":
			return Action{Kind: Quit}
		}
		return Action{}
	}
	switch k {
	case "esc", "q", "ctrl+k":
		a.quest = nil
	case "ctrl+c":
		return Action{Kind: Quit}
	case "x":
		if m.Quest() == nil {
			return Action{Note: "you are on no quest chain"}
		}
		q.Confirm = true
	}
	return Action{}
}

// ring says the terminal's bell is to ring (D75): the bell is on, and a mob has
// begun attacking the character since the last time it was asked.
func (a *App) ring(m *model.Model) bool {
	if m.Alerts == a.alerts {
		return false
	}
	a.alerts = m.Alerts
	return a.Bell
}

// deniedWords is why an intent was refused, in the server's own words (D76), else
// from the reason code.
func deniedWords(st model.IntentStatus) string {
	if st.Text != "" {
		return st.Text
	}
	return model.Words(st.Reason)
}

// walkedAway closes the screen of a person the character has walked away from,
// and says so (D78): a trade, talk or board screen stays with the NPC it was
// opened on, and closes only when the character leaves its reach.
func (a *App) walkedAway(m *model.Model) {
	far := func(id string) bool {
		n := m.NPC(id)
		return n == nil || (n.GetReachMm() > 0 && n.GetPos() != nil && !m.InReach(n))
	}
	what, npc := "", ""
	switch {
	case a.traders != nil && a.traders.Open && far(a.traders.NPC):
		what, npc, a.traders = "trade", a.traders.NPC, nil
	case a.talk != nil && far(a.talk.NPC):
		what, npc, a.talk = "conversation", a.talk.NPC, nil
	case a.board != nil && far(a.board.NPC):
		what, npc, a.board = "board", a.board.NPC, nil
	default:
		return
	}
	a.view.Notice, a.view.NoticeBad = fmt.Sprintf("You walked away from %s, so the %s closed", m.NPCName(npc), what), false
}
