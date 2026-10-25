package app

import (
	"strings"
	"time"

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
}

func (a *App) drawer() bool { return a.chat.drawer != nil }

// chatView puts chat in the view the screen draws: the tab, the marks, the drawer,
// and the toast and the bubbles still up.
func (a *App) chatView(v *screen.View) {
	c := &a.chat
	now := a.now()
	v.Read, v.Drawer = &c.marks, c.drawer
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
func (a *App) chatDo(act Action) {
	c := &a.chat
	switch act.Kind {
	case ToggleDrawer:
		if c.drawer != nil {
			c.drawer = nil
			return
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
	}
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
	case l.Channel == screen.ChanTell && l.Target != "":
		return "/tell " + l.Target + " " + text
	}
	return text
}

// lineTab is Tab on the command line: the lone slash that opened it gives way to
// chat, and the channel moves on (say, party, the character told last).
func (a *App) lineTab() {
	l := a.line
	if strings.HasPrefix(l.Text, "/") && l.Text != "/" {
		return // a command has no channel
	}
	l.Text = strings.TrimPrefix(l.Text, "/")
	a.sess.View(func(m *model.Model) {
		l.Channel = screen.NextChannel(l.Channel, m.InParty(), m.Social.TellPeer)
		l.Target = ""
		if l.Channel == screen.ChanTell {
			l.Target = m.Social.TellPeer
		}
	})
}
