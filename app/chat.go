package app

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
)

// Chat for a small screen (D101): the drawer, the tabs of the bottom panel, the
// toast and the speech bubbles. The screen draws them from the state kept here;
// the clock is the program's, so a bubble and a toast end on a redraw.

// ToastLasts is how long a tell or a mention stays on the status line.
const ToastLasts = 6 * time.Second

// chatState is what the program keeps for chat besides the model's scrollback.
type chatState struct {
	init bool // seen is set (Init, or the first change)
	seen int  // chat lines up to this number have been looked at for a bubble or a toast

	drawer *screen.DrawerUI
	// drawerAlerts and drawerUnder are the attacks begun, and whether one was on,
	// when the drawer opened: a new attack closes it.
	drawerAlerts int
	drawerUnder  bool

	toast      *screen.Toast
	toastUntil time.Time
	bubbles    []screen.Bubble
	marks      screen.Marks

	// card is the open card of a character (D101 §9); sel the name the keyboard
	// has selected among the names drawn, nil for none.
	card *screen.CharCard
	sel  *int
}

func (a *App) drawer() bool { return a.chat.drawer != nil }

// chatView puts chat in the view the screen draws: the tab, the marks, the drawer,
// and the toast and the bubbles still up.
func (a *App) chatView(v *screen.View) {
	c := &a.chat
	now := a.now()
	v.Read, v.Drawer, v.Person, v.NameSel = &c.marks, c.drawer, c.card, c.sel
	if c.toast != nil && now.Before(c.toastUntil) {
		v.Toast = c.toast
	}
	v.Bubbles = screen.LiveBubbles(c.bubbles, now)
}

// newChat returns the lines of chat that came after Seq n, oldest first.
func newChat(m *model.Model, n int) []model.ChatLine {
	chat := m.Social.Chat
	i := len(chat)
	for i > 0 && chat[i-1].Seq > n {
		i--
	}
	return chat[i:]
}

// watchChat looks at the chat that has come since it last looked: a line said
// raises a bubble, a tell or a mention a toast (not in a fight, when chat waits
// for its end). It also closes the drawer when an attack begins. The model is
// locked by the caller.
func (a *App) watchChat(m *model.Model) {
	c := &a.chat
	now := a.now()
	if !c.init {
		c.seen, c.init = m.Social.ChatN, true // what was said before the program began is not new
	}
	under := m.UnderAttack()
	for _, l := range newChat(m, c.seen) {
		if l.Channel == "say" {
			c.bubbles = append(c.bubbles, screen.Bubble{Name: l.From, Text: l.Text, Until: now.Add(screen.BubbleLasts)})
		}
		if t := screen.ToastFor(l); t != nil && !under {
			c.toast, c.toastUntil = t, now.Add(ToastLasts)
		}
	}
	c.seen = m.Social.ChatN
	c.bubbles = screen.LiveBubbles(c.bubbles, now)
	if under {
		c.sel = nil // the keys are the fight's
	}
	if c.drawer != nil {
		if DrawerShouldClose(c.drawerUnder, c.drawerAlerts, under, m.Alerts) {
			c.drawer = nil
			a.view.Notice, a.view.NoticeBad = "chat closed: you are under attack", false
		}
		c.drawerUnder = under
	}
}

// DrawerShouldClose says the drawer closes by itself (D101 §4; D75, danger is
// never hidden): an attack began since it opened, or one begins now.
func DrawerShouldClose(wasUnder bool, alerts int, under bool, nowAlerts int) bool {
	return nowAlerts != alerts || (under && !wasUnder)
}

// chatDo carries out a chat action.
func (a *App) chatDo(act Action) (tea.Model, tea.Cmd) {
	c := &a.chat
	switch act.Kind {
	case ToggleDrawer:
		if c.drawer != nil {
			c.drawer = nil
			return a, nil
		}
		c.drawer = &screen.DrawerUI{}
		a.sess.View(func(m *model.Model) { c.drawerAlerts, c.drawerUnder = m.Alerts, m.UnderAttack() })
	case PanelTab, SelectPanel:
		tab := act.Count
		if act.Kind == PanelTab {
			tab = (a.view.Panel + act.Count + screen.PanelTabs) % screen.PanelTabs
		}
		a.view.Panel = min(max(tab, 0), screen.PanelTabs-1)
		a.sess.View(func(m *model.Model) {
			if m.UnderAttack() && a.view.Panel != screen.PanelEvents {
				a.view.Notice = "in a fight the panel shows Events; " + screen.PanelName(a.view.Panel) + " comes back when it ends"
			}
		})
	case ReplyLine:
		a.line = &screen.Line{Channel: screen.ChanTell, Target: act.ID}
	case NameStep:
		a.stepName(act.Count)
	case OpenPerson:
		name := act.Note
		if act.Count < 0 { // the keyboard's selection
			a.refresh()
			if names := a.mouse.hits.Names; c.sel != nil && *c.sel < len(names) {
				name = names[*c.sel].Name
			}
		}
		return a.openPerson(name)
	case PersonClose:
		c.card = nil
	case PersonDo:
		return a.personDo(act.Count)
	}
	return a, nil
}

// stepName selects the next or previous name among those drawn, round again; 0
// leaves the selection. The drawing before the key is what says which are drawn.
func (a *App) stepName(by int) {
	c := &a.chat
	if by == 0 {
		c.sel = nil
		return
	}
	a.refresh()
	n := len(a.mouse.hits.Names)
	if n == 0 {
		c.sel = nil
		a.view.Notice = "no names to select: open the Chat tab ([ ]) or the drawer (Shift+C)"
		return
	}
	at := -1
	if c.sel != nil {
		at = *c.sel
	}
	next := (at + by + n) % n
	if c.sel == nil && by < 0 {
		next = n - 1
	}
	c.sel = &next
	a.paint.hitsStale = true
}

// nameKey handles a key that selects a name in the chat (D101 §9): N selects the
// first (or next), then Tab and Shift+Tab step, Enter opens its card and Esc lets
// go. Every other key is the game's. It says whether it took the key.
func (a *App) nameKey(k string, act *Action) bool {
	c := &a.chat
	switch {
	case c.sel == nil:
		return false
	case k == "tab":
		*act = Action{Kind: NameStep, Count: 1}
	case k == "shift+tab":
		*act = Action{Kind: NameStep, Count: -1}
	case k == "esc":
		*act = Action{Kind: NameStep}
	case k == "enter":
		*act = Action{Kind: OpenPerson, Count: -1} // the selected one, found when the action is done
	default:
		return false
	}
	return true
}

// personKey handles a key while a character's card is open: its four actions, and
// Esc. The card holds every other key until it is closed.
func (a *App) personKey(k string) Action {
	switch k {
	case "esc", "q":
		return Action{Kind: PersonClose}
	case "ctrl+c":
		return Action{Kind: Quit}
	}
	for i, key := range screen.CardKeys {
		if k == key || k == strings.ToUpper(key) {
			if a.chat.card.Acts() {
				return Action{Kind: PersonDo, Count: i}
			}
		}
	}
	return Action{}
}

// openPerson opens the card of the character called name and asks the server who
// it is. The character's own name opens nothing.
func (a *App) openPerson(name string) (tea.Model, tea.Cmd) {
	own := false
	a.sess.View(func(m *model.Model) { own = model.Key(name) == model.Key(m.Self.Name) })
	if own || strings.TrimSpace(name) == "" {
		return a, nil
	}
	a.chat.sel = nil
	a.chat.card = &screen.CharCard{Name: name}
	a.inflight++
	return a, a.send(Action{Kind: SocialDo, ID: "whoisask", Note: name})
}

// personDo carries out an action of the open card: Tell and Report open the command
// line for it, Invite and Ignore are asked of the server at once.
func (a *App) personDo(act int) (tea.Model, tea.Cmd) {
	c := a.chat.card
	a.chat.card = nil
	switch act {
	case screen.ActTell:
		a.line = &screen.Line{Channel: screen.ChanTell, Target: c.Name}
	case screen.ActReport:
		a.line = &screen.Line{Channel: screen.ChanReport, Target: c.Name}
	case screen.ActInvite:
		a.inflight++
		return a, a.send(Action{Kind: SocialDo, ID: "invite", Note: c.Name})
	case screen.ActIgnore:
		a.inflight++
		return a, a.send(Action{Kind: SocialDo, ID: "mute", Note: c.Name})
	}
	return a, nil
}

// drawerKey handles a key while the drawer is open and no screen is over it: it
// scrolls, filters and closes, and says whether it took the key. Movement and the
// fight keys go on to play, so the drawer never stops the character's hands.
func (a *App) drawerKey(k string, act *Action) bool {
	d := a.chat.drawer
	switch k {
	case "esc", "C":
		a.chat.drawer = nil
	case "up":
		d.Scroll++
	case "down":
		d.Scroll = max(d.Scroll-1, 0)
	case "pgup":
		d.Scroll += 5
	case "pgdown":
		d.Scroll = max(d.Scroll-5, 0)
	case "end":
		d.Scroll = 0
	case "tab":
		d.Filter, d.Scroll = (d.Filter+1)%screen.Filters, 0
	default:
		return false
	}
	return true
}

// lineWire is the text a command line sends: a command as typed, and plain text on
// the channel the line shows.
func lineWire(l *screen.Line) string {
	text := strings.TrimSpace(l.Text)
	switch {
	case text == "" || strings.HasPrefix(text, "/"):
		return text
	case l.Channel == screen.ChanParty:
		return "/p " + text
	case l.Channel == screen.ChanOOC:
		return "/oocsay " + text // a line that is only "off" is still a line, not the switch
	case l.Channel == screen.ChanReport && l.Target != "":
		return "/report " + l.Target + " " + text
	case l.Channel == screen.ChanTell && l.Target != "":
		return "/tell " + l.Target + " " + text
	}
	return text
}

// lineTab is Tab on the command line: the lone slash that opened it gives way to
// chat, and the channel moves on (say, party, the character told last).
func (a *App) lineTab() {
	l := a.line
	if (strings.HasPrefix(l.Text, "/") && l.Text != "/") || l.Channel == screen.ChanReport {
		return // a command has no channel, and a report has its reason to give
	}
	l.Text = strings.TrimPrefix(l.Text, "/")
	a.sess.View(func(m *model.Model) {
		l.Channel = screen.NextChannel(l.Channel, m.InParty(), !m.Social.OOCOff, m.Social.TellPeer)
		l.Target = ""
		if l.Channel == screen.ChanTell {
			l.Target = m.Social.TellPeer
		}
	})
}
