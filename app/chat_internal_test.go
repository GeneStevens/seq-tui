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
		{screen.Line{Text: "hello", Channel: "ooc"}, "/oocsay hello"},
		{screen.Line{Text: "off", Channel: "ooc"}, "/oocsay off"},
		{screen.Line{Text: "/ooc off", Channel: "ooc"}, "/ooc off"},
		{screen.Line{Text: "he spammed", Channel: "report", Target: "Bram"}, "/report Bram he spammed"},
		{screen.Line{Text: "", Channel: "report", Target: "Bram"}, ""},
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
	for _, k := range []string{"up", "down", "pgup"} {
		a.key(k)
	}
	if a.chat.drawer.Scroll != 5 { // the arrows are the character's, not the drawer's
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
	for _, want := range []string{"[party] > ", "[ooc] > ", "[tell Mira] > ", "[say] > ", "[party] > "} {
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

func TestTheCommandWordsOfOOCAndReports(t *testing.T) {
	for cmd, want := range map[string]string{"ooc": "ooc", "oocsay": "oocsay", "report": "report", "whois": "whois", "ignore": "mute", "tell": "tell", "oo": "?"} {
		if got := lineVerb(cmd); got != want {
			t.Errorf("/%s is %q, want %q", cmd, got, want)
		}
	}
}

// structuredChat applies a line with character references, as a current server sends it.
func structuredChat(m *model.Model) {
	ref := func(id, name, kind string) *pb.ChatPart {
		return &pb.ChatPart{Body: &pb.ChatPart_Character{Character: &pb.CharacterRef{Id: id, Name: name, Kind: kind}}}
	}
	txt := func(s string) *pb.ChatPart { return &pb.ChatPart{Body: &pb.ChatPart_Text{Text: s}} }
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Chat{Chat: &pb.ChatMessage{Channel: "say", From: "Bram", Text: "hi Mira and Aria",
		Speaker: &pb.CharacterRef{Id: "bram", Name: "Bram", Kind: "player"}, Parts: []*pb.ChatPart{txt("hi "), ref("mira", "Mira", "player"), txt(" and "), ref("me", "Aria", "player")}}}})
}

func TestNamesStepByKeyAndOpenTheirCardButYourOwnOpensNothing(t *testing.T) {
	a, m, _ := chatApp()
	structuredChat(m)
	a.view.Panel = screen.PanelChat
	a.render(a.w, a.h, 0, screen.Plain())
	var order []string
	for _, n := range a.mouse.hits.Names {
		order = append(order, n.Name)
	}
	if strings.Join(order, ",") != "Bram,Mira" {
		t.Fatalf("names in view: %q (your own is not one)", order)
	}
	if a.key("enter"); a.chat.card != nil {
		t.Fatal("Enter with nothing selected opened a card")
	}
	for i, want := range []string{"Bram", "Mira", "Bram"} { // N selects, Tab steps, round again
		k := "tab"
		if i == 0 {
			k = "N"
		}
		a.key(k)
		if a.chat.sel == nil || *a.chat.sel != []int{0, 1, 0}[i] {
			t.Fatalf("%s %d: selected %v, want %s", k, i, a.chat.sel, want)
		}
	}
	a.key("shift+tab")
	if *a.chat.sel != 1 {
		t.Errorf("Shift+Tab: %d", *a.chat.sel)
	}
	if _, cmd := a.key("enter"); cmd == nil || a.chat.card == nil || a.chat.card.Name != "Mira" || a.chat.card.State != screen.CardAsking || a.chat.sel != nil {
		t.Fatalf("Enter on Mira: card %+v, cmd %v", a.chat.card, cmd)
	}
	a.key("esc")
	if a.chat.card != nil {
		t.Error("Esc left the card open")
	}
	if _, cmd := a.openPerson("aria"); cmd != nil || a.chat.card != nil {
		t.Error("your own name opened a card")
	}
	a.key("N")
	a.key("esc")
	if a.chat.sel != nil {
		t.Error("Esc kept the selection")
	}
}

func TestAClickOnANameOpensItsCardAndTheCardHoldsTheClicks(t *testing.T) {
	a, m, _ := chatApp()
	structuredChat(m)
	a.view.Panel = screen.PanelChat
	a.render(a.w, a.h, 0, screen.Plain())
	n := a.mouse.hits.Names[1]
	if _, cmd := a.click(n.Rect.X+1, n.Rect.Y); cmd == nil || a.chat.card == nil || a.chat.card.Name != "Mira" {
		t.Fatalf("click on %+v: card %+v", n, a.chat.card)
	}
	a.card("Mira", screen.CharCard{Name: "Mira", State: screen.CardFound, Level: 3, Class: "mage", Zone: "Z", Online: true})
	a.paint.hitsStale = true
	a.render(a.w, a.h, 0, screen.Plain())
	h := a.mouse.hits
	if h.Card.W == 0 || len(h.CardActs) != screen.CardActions {
		t.Fatalf("card %+v acts %+v", h.Card, h.CardActs)
	}
	// A click on Tell opens the line to Mira.
	tell := h.CardActs[screen.ActTell]
	a.click(tell.Rect.X, tell.Rect.Y)
	if a.chat.card != nil || a.line == nil || a.line.Channel != screen.ChanTell || a.line.Target != "Mira" {
		t.Fatalf("Tell: card %+v line %+v", a.chat.card, a.line)
	}
}

func TestTheCardsKeys(t *testing.T) {
	for _, c := range []struct {
		key     string
		state   int
		want    Kind
		count   int
		pressed bool
	}{
		{"t", screen.CardFound, PersonDo, screen.ActTell, true},
		{"i", screen.CardFound, PersonDo, screen.ActInvite, true},
		{"n", screen.CardFound, PersonDo, screen.ActIgnore, true},
		{"R", screen.CardFound, PersonDo, screen.ActReport, true},
		{"r", screen.CardAsking, PersonDo, screen.ActReport, true},
		{"t", screen.CardUnknown, None, 0, false},
		{"i", screen.CardFailed, None, 0, false},
		{"esc", screen.CardUnknown, PersonClose, 0, true},
		{"q", screen.CardFound, PersonClose, 0, true},
		{"w", screen.CardFound, None, 0, false}, // the card holds the movement keys
	} {
		a, _, _ := chatApp()
		a.chat.card = &screen.CharCard{Name: "Mira", State: c.state}
		got := a.personKey(c.key)
		if got.Kind != c.want || (c.pressed && got.Count != c.count) {
			t.Errorf("%q in state %d: %+v, want kind %d count %d", c.key, c.state, got, c.want, c.count)
		}
	}
}

func TestCardActionsOpenTheLinesReportAsksWhyAndIgnoreMutes(t *testing.T) {
	for _, c := range []struct {
		act           int
		channel, tgt  string
		verb, argNote string
	}{
		{screen.ActTell, screen.ChanTell, "Mira", "", ""},
		{screen.ActReport, screen.ChanReport, "Mira", "", ""},
		{screen.ActInvite, "", "", "invite", "Mira"},
		{screen.ActIgnore, "", "", "mute", "Mira"},
	} {
		a, _, _ := chatApp()
		a.chat.card = &screen.CharCard{Name: "Mira", State: screen.CardFound}
		_, cmd := a.personDo(c.act)
		if a.chat.card != nil {
			t.Errorf("action %d left the card open", c.act)
		}
		switch {
		case c.channel != "":
			if a.line == nil || a.line.Channel != c.channel || a.line.Target != c.tgt || cmd != nil {
				t.Errorf("action %d: line %+v cmd %v", c.act, a.line, cmd)
			}
		case cmd == nil || a.line != nil:
			t.Errorf("action %d asked nothing of the server (line %+v)", c.act, a.line)
		}
	}
}

func TestAReportLinePromptsForTheReasonAndTabLeavesItBe(t *testing.T) {
	a, _, _ := chatApp()
	a.line = &screen.Line{Channel: screen.ChanReport, Target: "Mira"}
	a.lineTab()
	if a.line.Channel != screen.ChanReport {
		t.Errorf("Tab moved a report to %q", a.line.Channel)
	}
	if got := a.line.Prompt(); got != "[report Mira: why] > " {
		t.Errorf("prompt %q", got)
	}
}

func TestOOCLeavesTheTabCycleWhileItIsOff(t *testing.T) {
	a, m, _ := chatApp()
	hear(a, m, "tell", "Mira", "Aria", "hi")
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_ChatSettings{ChatSettings: &pb.ChatSettings{Character: "Aria", OocOn: false}}})
	a.line = &screen.Line{Channel: screen.ChanSay}
	a.lineTab()
	if a.line.Channel != screen.ChanTell {
		t.Errorf("with ooc off Tab went to %q", a.line.Channel)
	}
}

// card puts the server's answer on the open card, as the program does when it comes.
func (a *App) card(name string, c screen.CharCard) { a.setCard(name, &c) }

func TestTheDrawerTakesOnlyItsOwnKeys(t *testing.T) {
	a, _, _ := chatApp()
	a.key("C")
	for k, taken := range map[string]bool{"up": false, "down": false, "left": false, "right": false, "w": false, "f": false,
		"pgup": true, "pgdown": true, "end": true, "tab": true, "esc": true} {
		var act Action
		if got := a.drawerKey(k, &act); got != taken {
			t.Errorf("drawer took %q: %v, want %v", k, got, taken)
		}
		if !a.drawer() { // Esc dropped it
			a.key("C")
		}
	}
}

func TestChatIsReadWhenAnUpdateFollowsADrawingThatShowedIt(t *testing.T) {
	a, m, _ := chatApp()
	a.w, a.h = 80, 24
	hear(a, m, "say", "Mira", "", "hello")
	hear(a, m, "tell", "Mira", "Aria", "psst")
	unread := func() int {
		var n int
		a.sess.View(func(m *model.Model) { _, n, _ = a.chat.marks.Unread(m) })
		return n
	}
	a.view.Panel = screen.PanelEvents
	a.render(80, 24, 0, screen.Plain())
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if unread() != 2 {
		t.Fatalf("a drawing of the Events tab read chat: %d unread", unread())
	}
	a.view.Panel = screen.PanelChat
	a.render(80, 24, 0, screen.Plain())
	if unread() != 2 {
		t.Fatalf("a drawing alone read chat: %d unread", unread())
	}
	hear(a, m, "say", "Mira", "", "and later") // arrives after the drawing: not seen
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if unread() != 1 {
		t.Errorf("the update after the drawing: %d unread, want 1 (the line after the drawing)", unread())
	}
}

func TestOOCIsKnownFromTheSettingsAtLoginAndTheyReplaceWhatWasKnown(t *testing.T) {
	a, m, _ := chatApp()
	settings := func(on bool) {
		m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_ChatSettings{ChatSettings: &pb.ChatSettings{Character: "Aria", OocOn: on}}})
	}
	cycle := func() string {
		a.line = &screen.Line{Channel: screen.ChanSay}
		hear(a, m, "tell", "Mira", "Aria", "hi")
		a.lineTab()
		return a.line.Channel
	}
	settings(false)
	if got := cycle(); got != screen.ChanTell {
		t.Errorf("ooc off at login: Tab went to %q", got)
	}
	settings(true)
	a.line = &screen.Line{Channel: screen.ChanParty}
	a.lineTab()
	if a.line.Channel != screen.ChanOOC {
		t.Errorf("ooc on: Tab went to %q", a.line.Channel)
	}
}
