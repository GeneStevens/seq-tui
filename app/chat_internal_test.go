package app

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// D101: the program's half of chat: bubbles and toasts raised by new lines, the
// drawer that closes on an attack, the keys, and what the command line sends.

func chatApp() (*App, *model.Model, *time.Time) {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "Z", Seq: 1, GameTimeMs: 1000,
		Self:     &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Aria", Pos: &pb.Pos{X: 10000, Y: 10000}, Hp: 30, MaxHp: 30}},
		Entities: []*pb.EntityState{{Id: "mira", Name: "Mira", Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, Pos: &pb.Pos{X: 12000, Y: 10000}, Hp: 20, MaxHp: 20}},
	}}})
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: "Z", Name: "Z", ViewRadiusMm: 20000,
		Bounds: &pb.Rect{MinX: 0, MinY: 0, MaxX: 40000, MaxY: 40000}}}})
	now := time.Unix(1000, 0)
	a := New(client.NewSession(client.Config{Model: m}))
	a.Clock = func() time.Time { return now }
	a.sess.View(func(m *model.Model) { a.chat.seen, a.chat.init = m.Social.ChatN, true })
	return a, m, &now
}

func hear(a *App, m *model.Model, channel, from, to, text string) {
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Chat{Chat: &pb.ChatMessage{Channel: channel, From: from, To: to, Text: text}}})
	a.sess.View(a.watchChat)
}

func attack(m *model.Model, seq uint64, on bool) {
	cs := &pb.CombatState{Actor: "me", InCombat: on}
	if on {
		cs.Attackers = []string{"wolf"}
	}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: seq, GameTimeMs: int64(seq) * 100, Body: &pb.Event_CombatState{CombatState: cs}}}})
}

func TestLineWireSendsPlainTextOnTheChannelAndCommandsAsTyped(t *testing.T) {
	for _, c := range []struct {
		l    screen.Line
		want string
	}{
		{screen.Line{Text: "hello"}, "hello"},
		{screen.Line{Text: "  hello  "}, "hello"},
		{screen.Line{Text: "hello", Channel: "party"}, "/p hello"},
		{screen.Line{Text: "hello", Channel: "tell", Target: "Mira"}, "/tell Mira hello"},
		{screen.Line{Text: "hello", Channel: "tell"}, "hello"},
		{screen.Line{Text: "/who", Channel: "party"}, "/who"},
		{screen.Line{Text: "/tell Bram hi", Channel: "tell", Target: "Mira"}, "/tell Bram hi"},
		{screen.Line{Text: "/"}, "/"},
		{screen.Line{}, ""},
	} {
		if got := lineWire(&c.l); got != c.want {
			t.Errorf("%+v sends %q, want %q", c.l, got, c.want)
		}
	}
}

func TestTheDrawerClosesOnANewAttackAndOnlyThen(t *testing.T) {
	for _, c := range []struct {
		name        string
		wasUnder    bool
		alerts      int
		under       bool
		nowAlerts   int
		wantClosing bool
	}{
		{"quiet", false, 0, false, 0, false},
		{"an attack begins", false, 0, true, 1, true},
		{"under attack from the start, still the same", true, 3, true, 3, false},
		{"under attack, a second mob joins", true, 3, true, 4, true},
		{"the fight ends", true, 3, false, 3, false},
		{"under attack seen without an alert", false, 3, true, 3, true},
	} {
		if got := DrawerShouldClose(c.wasUnder, c.alerts, c.under, c.nowAlerts); got != c.wantClosing {
			t.Errorf("%s: %v", c.name, got)
		}
	}
}

func TestSayRaisesABubbleATellAToastAndAFightHoldsTheToast(t *testing.T) {
	a, m, now := chatApp()
	hear(a, m, "party", "Mira", "", "ready")
	var v screen.View
	a.chatView(&v)
	if len(v.Bubbles) != 0 || v.Toast != nil {
		t.Fatalf("a party line raised %+v %+v", v.Bubbles, v.Toast)
	}
	hear(a, m, "say", "Mira", "", "well met")
	hear(a, m, "tell", "Mira", "Aria", "group for the Warren?")
	a.chatView(&v)
	if len(v.Bubbles) != 1 || v.Bubbles[0].Name != "Mira" || v.Toast == nil || v.Toast.Hint != "r reply" {
		t.Fatalf("bubbles %+v toast %+v", v.Bubbles, v.Toast)
	}
	*now = now.Add(screen.BubbleLasts + time.Second)
	v = screen.View{}
	a.chatView(&v)
	if len(v.Bubbles) != 0 || v.Toast == nil {
		t.Errorf("after the bubble's time: %+v, toast %+v", v.Bubbles, v.Toast)
	}
	*now = now.Add(ToastLasts)
	v = screen.View{}
	a.chatView(&v)
	if v.Toast != nil {
		t.Error("the toast stays past its time")
	}
	// In a fight chat waits: a mention raises no toast.
	a.sess.View(func(m *model.Model) { attack(m, 5, true) })
	hear(a, m, "say", "Mira", "", "Aria, behind you")
	v = screen.View{}
	a.chatView(&v)
	if v.Toast != nil {
		t.Errorf("a toast in a fight: %+v", v.Toast)
	}
}

func TestTheKeysOpenTheDrawerSwitchTabsAndReply(t *testing.T) {
	a, m, _ := chatApp()
	a.key("C")
	if !a.drawer() {
		t.Fatal("Shift+C did not open the drawer")
	}
	for _, k := range []string{"up", "pgup"} {
		a.key(k)
	}
	if a.chat.drawer.Scroll != 6 {
		t.Errorf("scroll %d", a.chat.drawer.Scroll)
	}
	a.key("tab")
	a.key("end")
	if d := a.chat.drawer; d.Filter != screen.FilterSay || d.Scroll != 0 {
		t.Errorf("filter %d scroll %d", d.Filter, d.Scroll)
	}
	a.key("esc")
	if a.drawer() {
		t.Error("Esc did not drop the drawer")
	}
	a.key("]")
	a.key("]")
	if a.view.Panel != screen.PanelParty {
		t.Errorf("] twice: tab %d", a.view.Panel)
	}
	a.key("]")
	a.key("[")
	if a.view.Panel != screen.PanelParty {
		t.Errorf("wraps round: %d", a.view.Panel)
	}
	a.key("r")
	if a.line != nil || a.view.Notice == "" {
		t.Errorf("r with nobody to answer opened %+v, notice %q", a.line, a.view.Notice)
	}
	hear(a, m, "tell", "Mira", "Aria", "group for the Warren?")
	a.key("r")
	if a.line == nil || a.line.Channel != "tell" || a.line.Target != "Mira" || a.line.Prompt() != "[tell Mira] > " {
		t.Fatalf("r opened %+v", a.line)
	}
}

func TestTabOnTheLineCyclesTheChannelAndTheLoneSlashGivesWay(t *testing.T) {
	a, m, _ := chatApp()
	hear(a, m, "tell", "Mira", "Aria", "hi")
	a.sess.View(func(m *model.Model) {
		m.Social.Party = &pb.PartyState{Id: "p1", Leader: "Aria", Members: []*pb.PartyMemberInfo{{Name: "Aria", Online: true}}}
	})
	a.key("/")
	if a.line == nil || a.line.Prompt() != "[cmd] > " {
		t.Fatalf("/ opened %+v", a.line)
	}
	for _, want := range []string{"[party] > ", "[tell Mira] > ", "[say] > ", "[party] > "} {
		a.lineKey(keyPress("tab"))
		if got := a.line.Prompt(); got != want || a.line.Text != "" {
			t.Fatalf("Tab: %q %q, want %q", got, a.line.Text, want)
		}
	}
	for _, r := range "see you" {
		a.lineKey(keyPress(string(r)))
	}
	if got := lineWire(a.line); got != "/p see you" {
		t.Errorf("sends %q", got)
	}
	// A command has no channel: Tab leaves it be.
	a.key("esc")
	a.key("/")
	a.lineKey(keyPress("w"))
	a.lineKey(keyPress("tab"))
	if a.line.Text != "/w" || a.line.Prompt() != "[cmd] > " {
		t.Errorf("Tab changed a command: %+v", a.line)
	}
	// Backspace over the slash is a chat line.
	a.lineKey(keyPress("backspace"))
	a.lineKey(keyPress("backspace"))
	if a.line.Prompt() != "[say] > " {
		t.Errorf("after the slash is gone: %q", a.line.Prompt())
	}
}

func TestAnAttackClosesTheDrawerAndSaysSo(t *testing.T) {
	a, m, _ := chatApp()
	a.key("C")
	attack(m, 2, true)
	a.sess.View(a.watchChat)
	if a.drawer() || !strings.Contains(a.view.Notice, "under attack") {
		t.Errorf("drawer %v notice %q", a.drawer(), a.view.Notice)
	}
	// Opened in the middle of a fight, it stays until another begins.
	a.key("C")
	a.sess.View(a.watchChat)
	if !a.drawer() {
		t.Fatal("opening it in a fight was refused")
	}
	if s := screenNow(a); !strings.Contains(s, "UNDER ATTACK: F fights back") {
		t.Errorf("no warning in the drawer:\n%s", s)
	}
}

func TestClickingATabSelectsItAndTheWheelScrollsTheDrawer(t *testing.T) {
	a, m, _ := chatApp()
	hear(a, m, "say", "Mira", "", "well met")
	a.render(a.w, a.h, 0, screen.Plain()) // the draw notes where the tabs are
	var chat screen.Tab
	for _, tb := range a.mouse.hits.Panel {
		if tb.Tab == screen.PanelChat {
			chat = tb
		}
	}
	if chat.Rect.W == 0 {
		t.Fatalf("no Chat tab recorded: %+v", a.mouse.hits.Panel)
	}
	a.click(chat.Rect.X+1, chat.Rect.Y)
	if a.view.Panel != screen.PanelChat {
		t.Errorf("a click on Chat: tab %d", a.view.Panel)
	}
	a.key("C")
	a.paint.hitsStale = true
	a.render(a.w, a.h, 0, screen.Plain())
	d := a.mouse.hits.Drawer
	if d.W == 0 {
		t.Fatal("no drawer recorded")
	}
	if _, cmd := a.click(d.X+2, d.Y+2); cmd != nil || a.mouse.cursor != nil {
		t.Error("a click in the drawer did something on the map")
	}
}

// keyPress is a key the command line gets: a letter, or one of the named keys.
func keyPress(k string) tea.KeyPressMsg {
	switch k {
	case "tab":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyTab})
	case "backspace":
		return tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace})
	}
	return tea.KeyPressMsg(tea.Key{Code: rune(k[0]), Text: k})
}
