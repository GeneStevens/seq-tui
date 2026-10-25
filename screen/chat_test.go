package screen_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// D101, chat on a small screen: the tabbed panel, the bubbles, the toast and the
// drawer. chatWorld is Aria with Mira beside her, a wolf, a guard and a party.
func chatWorld() *model.Model {
	m := model.New()
	pos := func(x, y int64) *pb.Pos { return &pb.Pos{X: 10000 + x, Y: 10000 + y} }
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "Z", Seq: 1, GameTimeMs: 1000,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Aria", Level: 3, Pos: pos(0, 0), Hp: 30, MaxHp: 30}},
		Entities: []*pb.EntityState{
			{Id: "mira", Name: "Mira", Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, Pos: pos(4000, 2000), Hp: 20, MaxHp: 20},
			{Id: "wolf", Name: "Gray Wolf", Kind: pb.EntityKind_ENTITY_KIND_MOB, Level: 2, Pos: pos(-6000, -2000), Hp: 20, MaxHp: 20},
		},
		Npcs: []*pb.NpcInfo{{Id: "gorran", Name: "Gorran Stoutarm", Title: "Warrior Trainer", Role: "trainer", Pos: pos(-3000, 4000), ReachMm: 3000}},
	}}})
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: "Z", Name: "Z", ViewRadiusMm: 20000,
		Bounds: &pb.Rect{MinX: -20000, MinY: -20000, MaxX: 40000, MaxY: 40000}}}})
	return m
}

func say(m *model.Model, channel, from, to, text string) {
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Chat{Chat: &pb.ChatMessage{Channel: channel, From: from, To: to, Text: text}}})
}

var evSeq = map[*model.Model]uint64{}

func combat(m *model.Model, on bool) {
	evSeq[m]++
	seq := 10 + evSeq[m]
	ev := &pb.CombatState{Actor: "me", InCombat: on}
	if on {
		ev.Attackers = []string{"wolf"}
	}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: seq, GameTimeMs: int64(seq) * 100, Body: &pb.Event_CombatState{CombatState: ev}}}})
}

// talk is the scenario of the mission: say, a tell, a mention and a party line.
func talk(m *model.Model) {
	say(m, "say", "Mira", "", "well met, traveller, the road is long")
	say(m, "party", "Mira", "", "ready when you are")
	say(m, "tell", "Mira", "Aria", "group for the Warren?")
	say(m, "say", "Gorran Stoutarm", "", "Aria, come back when you are stronger")
	say(m, "say", "Aria", "", "on my way")
}

func rowsOf(s string) []string { return strings.Split(s, "\n") }

func view() screen.View {
	return screen.View{Conn: screen.ConnOnline, HasTick: true, Since: 400 * time.Millisecond, Read: &screen.Marks{}}
}

func TestBubbleTextCutsAtTwentyFourCharacters(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"hi", "hi"},
		{"  spaced   out  ", "spaced out"},
		{"exactly twenty-four chars", "exactly twenty-four chars"[:24] + "…"},
		{"0123456789012345678901234", "012345678901234567890123…"},
		{"012345678901234567890123", "012345678901234567890123"},
		{"tab\there\x1b[31m", "tab?here?[31m"},
	} {
		if got := screen.BubbleText(c.in); got != c.want {
			t.Errorf("BubbleText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLiveBubblesDropTheExpiredAndKeepTheNewestPerSpeaker(t *testing.T) {
	now := time.Unix(100, 0)
	at := func(s float64) time.Time { return now.Add(time.Duration(s * float64(time.Second))) }
	for _, c := range []struct {
		name string
		in   []screen.Bubble
		want string
	}{
		{"none", nil, ""},
		{"expired", []screen.Bubble{{Name: "A", Text: "1", Until: at(-1)}, {Name: "B", Text: "2", Until: at(0)}}, ""},
		{"up", []screen.Bubble{{Name: "A", Text: "1", Until: at(0.1)}}, "1"},
		{"newest per speaker", []screen.Bubble{{Name: "mira", Text: "1", Until: at(2)}, {Name: "Mira", Text: "2", Until: at(3)}}, "2"},
		{"an older one outlives a newer: still the newest", []screen.Bubble{{Name: "A", Text: "1", Until: at(9)}, {Name: "A", Text: "2", Until: at(1)}}, "2"},
		{"the newest four, oldest first", []screen.Bubble{{Name: "A", Text: "1", Until: at(9)}, {Name: "B", Text: "2", Until: at(9)}, {Name: "C", Text: "3", Until: at(9)}, {Name: "D", Text: "4", Until: at(9)}, {Name: "E", Text: "5", Until: at(9)}}, "2345"},
	} {
		var got string
		for _, b := range screen.LiveBubbles(c.in, now) {
			got += b.Text
		}
		if got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	if screen.BubbleLasts != 4*time.Second {
		t.Errorf("a bubble lasts %v, not about 4 s", screen.BubbleLasts)
	}
}

func TestNextChannelCyclesSayPartyOOCTellAndSkipsWhatIsNotThere(t *testing.T) {
	for _, c := range []struct {
		cur   string
		party bool
		ooc   bool
		peer  string
		want  string
	}{
		{"say", true, true, "Mira", "party"},
		{"party", true, true, "Mira", "ooc"},
		{"ooc", true, true, "Mira", "tell"},
		{"tell", true, true, "Mira", "say"},
		{"say", false, true, "Mira", "ooc"},
		{"say", false, false, "Mira", "tell"},
		{"party", true, false, "Mira", "tell"},
		{"say", true, true, "", "party"},
		{"party", true, true, "", "ooc"},
		{"ooc", true, true, "", "say"},
		{"ooc", false, true, "", "say"},
		{"say", false, false, "", "say"},
		{"", true, true, "", "party"},
		{"tell", false, true, "Mira", "say"},
		{"report", true, true, "Mira", "party"},
	} {
		if got := screen.NextChannel(c.cur, c.party, c.ooc, c.peer); got != c.want {
			t.Errorf("NextChannel(%q, party %v, ooc %v, peer %q) = %q, want %q", c.cur, c.party, c.ooc, c.peer, got, c.want)
		}
	}
}

func TestThePromptNamesTheChannel(t *testing.T) {
	for _, c := range []struct {
		l    screen.Line
		want string
	}{
		{screen.Line{}, "[say] > "},
		{screen.Line{Text: "hello"}, "[say] > "},
		{screen.Line{Channel: "party"}, "[party] > "},
		{screen.Line{Channel: "ooc"}, "[ooc] > "},
		{screen.Line{Channel: "report", Target: "Mira"}, "[report Mira: why] > "},
		{screen.Line{Channel: "tell", Target: "Mira"}, "[tell Mira] > "},
		{screen.Line{Channel: "tell"}, "[say] > "},
		{screen.Line{Text: "/", Channel: "party"}, "[cmd] > "},
		{screen.Line{Text: "/who", Channel: "tell", Target: "Mira"}, "[cmd] > "},
	} {
		if got := c.l.Prompt(); got != c.want {
			t.Errorf("%+v: %q, want %q", c.l, got, c.want)
		}
	}
}

func TestUnreadCountsPerTabAndWhatViewingClears(t *testing.T) {
	m := chatWorld()
	talk(m) // 4 lines from others (one of them party), 1 of Aria's own
	var mk screen.Marks
	_, chat, party := mk.Unread(m)
	if chat != 4 || party != 1 {
		t.Fatalf("unread chat %d party %d, want 4 and 1 (own lines do not count)", chat, party)
	}
	if mail := mk.Mail(m); mail != 2 {
		t.Errorf("mail %d: a tell and a mention make two", mail)
	}
	mk.Viewed(m, screen.PanelParty)
	if _, chat, party := mk.Unread(m); chat != 4 || party != 0 {
		t.Errorf("after the Party tab: chat %d party %d", chat, party)
	}
	say(m, "party", "Mira", "", "go")
	say(m, "say", "Mira", "", "go")
	if _, chat, party := mk.Unread(m); chat != 6 || party != 1 {
		t.Errorf("two more: chat %d party %d", chat, party)
	}
	mk.Viewed(m, screen.PanelChat)
	if ev, chat, party := mk.Unread(m); chat != 0 || party != 0 || mk.Mail(m) != 0 || ev != m.LogN-mk.Events {
		t.Errorf("after the Chat tab: chat %d party %d mail %d", chat, party, mk.Mail(m))
	}
	var nilMarks *screen.Marks
	if _, chat, _ := nilMarks.Unread(m); chat != 6 {
		t.Errorf("a nil mark has read nothing: %d", chat)
	}
}

func TestTheBadgesAreOnTheTabsAndClearWhenTheTabIsShown(t *testing.T) {
	m := chatWorld()
	talk(m)
	v := view()
	out := screen.Render(m, v, 80, 24, screen.Plain())
	top := panelTop(t, out)
	if !strings.Contains(top, "[Events: short · v: all]") || !strings.Contains(top, "Chat •4") || !strings.Contains(top, "Party •1") {
		t.Errorf("the tabs: %q", top)
	}
	if !strings.Contains(top, "[say] Aria: on my way") {
		t.Errorf("the newest chat line does not run in the border while Events is shown: %q", top)
	}
	v.Panel = screen.PanelChat
	top = panelTop(t, screen.Render(m, v, 80, 24, screen.Plain()))
	if !strings.Contains(top, "[Chat]") || strings.Contains(top, "•") || strings.Contains(top, "on my way") {
		t.Errorf("Chat shown: %q (no badge, no ticker: the lines are below)", top)
	}
	v.Panel = screen.PanelParty
	out = screen.Render(m, v, 80, 24, screen.Plain())
	if top = panelTop(t, out); !strings.Contains(top, "[Party]") || !strings.Contains(out, "ready when you are") || strings.Contains(out, "well met") {
		t.Errorf("Party shows only the party's chat: %q\n%s", top, out)
	}
}

// panelTop is the top border of the bottom panel: the row with the tabs.
func panelTop(t *testing.T, out string) string {
	t.Helper()
	for _, r := range rowsOf(out) {
		if strings.HasPrefix(r, "┌─ ") && (strings.Contains(r, "Events") || strings.Contains(r, "[Chat]") || strings.Contains(r, "[Party]")) {
			return r
		}
	}
	t.Fatalf("no tabs on the screen:\n%s", out)
	return ""
}

func TestInAFightThePanelShowsEventsAndChatIsABadgeAndATicker(t *testing.T) {
	m := chatWorld()
	talk(m)
	combat(m, true)
	v := view()
	v.Panel = screen.PanelChat // the player's choice; the fight overrides it
	out := screen.Render(m, v, 80, 24, screen.Plain())
	top := panelTop(t, out)
	if !strings.Contains(top, "[Events:") || !strings.Contains(top, "Chat •4") || !strings.Contains(top, "on my way") {
		t.Errorf("in a fight: %q", top)
	}
	if strings.Contains(out, "well met") {
		t.Errorf("the chat is on the screen in a fight:\n%s", out)
	}
	say(m, "tell", "Mira", "Aria", "behind you")
	if _, chat, _ := v.Read.Unread(m); chat != 5 {
		t.Errorf("a line in the fight is unread: %d", chat)
	}
	combat(m, false)
	out = screen.Render(m, v, 80, 24, screen.Plain())
	if !strings.Contains(out, "[Chat]") || !strings.Contains(out, "behind you") {
		t.Errorf("after the fight the chat tab is back:\n%s", out)
	}
}

func TestACatchUpLineSaysWhatCameInTheFight(t *testing.T) {
	m := chatWorld()
	talk(m)
	combat(m, true)
	say(m, "say", "Mira", "", "watch out")
	say(m, "party", "Mira", "", "heal?")
	say(m, "party", "Mira", "", "heal?")
	say(m, "tell", "Mira", "Aria", "behind you")
	say(m, "say", "Aria", "", "busy")
	combat(m, false)
	last := m.Log[len(m.Log)-1]
	if last.Kind != "chat_catchup" || last.Text != "While fighting: 4 chat (2 party, 1 tells)" {
		t.Errorf("the catch-up line: %+v", last)
	}
	count := func() (n int) {
		for _, e := range m.Log {
			if e.Kind == "chat_catchup" {
				n++
			}
		}
		return n
	}
	before := count()
	combat(m, true)
	combat(m, false)
	if count() != before {
		t.Error("a fight with no chat added a catch-up line")
	}
}

func TestATellOrAMentionIsAToastAndThenACount(t *testing.T) {
	m := chatWorld()
	talk(m)
	v := view()
	v.Toast = screen.ToastFor(m.Social.Chat[2])
	out := screen.Render(m, v, 80, 24, screen.Plain())
	last := rowsOf(out)[23]
	if !strings.Contains(last, "✉ Mira: group for the Warren?") || !strings.HasSuffix(strings.TrimRight(last, " "), "r reply") {
		t.Errorf("the toast: %q", last)
	}
	m2 := screen.ToastFor(m.Social.Chat[3])
	if m2 == nil || !strings.Contains(m2.Body, "Gorran Stoutarm: Aria, come back") || m2.Hint != "C chat" {
		t.Errorf("a mention's toast: %+v", m2)
	}
	for i, want := range []bool{false, false, true, true, false} {
		if got := screen.ToastFor(m.Social.Chat[i]) != nil; got != want {
			t.Errorf("line %d raises a toast: %v, want %v", i, got, want)
		}
	}
	v.Toast = nil
	last = rowsOf(screen.Render(m, v, 80, 24, screen.Plain()))[23]
	if !strings.Contains(last, "✉ 2") {
		t.Errorf("after the toast the count stays on the status line: %q", last)
	}
	v.Panel = screen.PanelChat
	last = rowsOf(screen.Render(m, v, 80, 24, screen.Plain()))[23]
	if strings.Contains(last, "✉") {
		t.Errorf("the count stays after the Chat tab was shown: %q", last)
	}
	// An attack takes the line; the toast waits.
	combat(m, true)
	v.Toast = screen.ToastFor(m.Social.Chat[2])
	if last := rowsOf(screen.Render(m, v, 80, 24, screen.Plain()))[23]; strings.Contains(last, "✉") {
		t.Errorf("a toast hides the line in a fight: %q", last)
	}
}

func TestBubblesSitByTheirSpeakersInOpenGroundAndNeverOverAGlyph(t *testing.T) {
	m := chatWorld()
	talk(m)
	v := view()
	until := time.Now().Add(time.Hour)
	v.Bubbles = []screen.Bubble{
		{Name: "Aria", Text: "on my way", Until: until},
		{Name: "Nobody", Text: "I am not here", Until: until},
	}
	plain := screen.Render(m, v, 120, 40, screen.Plain())
	row := ""
	for _, r := range rowsOf(plain) {
		if strings.Contains(r, "“on my way”") {
			row = r
		}
	}
	if row == "" {
		t.Fatalf("no bubble over the map:\n%s", plain)
	}
	if strings.Contains(plain, "not here") {
		t.Error("a speaker out of view got a bubble")
	}
	// The glyphs of the map are all still there with the bubble up.
	v.Bubbles = nil
	bare := screen.Render(m, v, 120, 40, screen.Plain())
	for _, glyph := range []string{"@", "p", "m", "T"} {
		if strings.Count(plain, glyph) < strings.Count(bare, glyph) {
			t.Errorf("a bubble covered a %q", glyph)
		}
	}
	// A line longer than 24 characters is cut in the bubble and whole in the log.
	v.Bubbles = []screen.Bubble{{Name: "Mira", Text: m.Social.Chat[0].Text, Until: until}}
	out := screen.Render(m, v, 120, 40, screen.Plain())
	if !strings.Contains(out, "“well met, traveller, the…”") {
		t.Errorf("the long line is not cut at 24:\n%s", out)
	}
}

func TestEveryChatPartDrawsAtItsSizeAndStylingOnlyAddsEscapes(t *testing.T) {
	m := chatWorld()
	talk(m)
	until := time.Now().Add(time.Hour)
	for _, sz := range [][2]int{{80, 24}, {81, 25}, {120, 40}, {200, 60}} {
		for _, tc := range []struct {
			name string
			set  func(*screen.View)
		}{
			{"events", func(*screen.View) {}},
			{"chat", func(v *screen.View) { v.Panel = screen.PanelChat }},
			{"party", func(v *screen.View) { v.Panel = screen.PanelParty }},
			{"drawer", func(v *screen.View) { v.Drawer = &screen.DrawerUI{Filter: screen.FilterTell, Scroll: 99} }},
			{"toast", func(v *screen.View) { v.Toast = screen.ToastFor(m.Social.Chat[2]) }},
			{"bubbles", func(v *screen.View) { v.Bubbles = []screen.Bubble{{Name: "Aria", Text: "hi", Until: until}} }},
			{"line", func(v *screen.View) { v.Line = &screen.Line{Channel: "tell", Target: "Mira", Text: "hello"} }},
		} {
			v := view()
			tc.set(&v)
			plain := screen.Render(m, v, sz[0], sz[1], screen.Plain())
			for i, l := range rowsOf(plain) {
				if n := len([]rune(l)); n != sz[0] {
					t.Fatalf("%s at %v: line %d is %d cells: %q", tc.name, sz, i, n, l)
				}
			}
			if len(rowsOf(plain)) != sz[1] {
				t.Fatalf("%s at %v: %d lines", tc.name, sz, len(rowsOf(plain)))
			}
			v2 := view()
			tc.set(&v2)
			for _, st := range []screen.Styler{screen.ANSI(), screen.Mono()} {
				if got := sgr.ReplaceAllString(screen.Render(m, v2, sz[0], sz[1], st), ""); got != plain {
					t.Errorf("%s at %v: the styled screen differs from the plain one", tc.name, sz)
				}
			}
		}
	}
}

func TestColourCarriesMeaningAndNoColourStaysReadable(t *testing.T) {
	m := chatWorld()
	talk(m)
	v := view()
	v.Panel = screen.PanelChat
	ansi := screen.Render(m, v, 120, 40, screen.ANSI())
	mono := screen.Render(m, v, 120, 40, screen.Mono())
	for _, want := range []struct{ what, code string }{
		{"party cyan", "\x1b[36m[party]"},
		{"tell yellow", "\x1b[33m[tell]"},
		{"a visible player", "\x1b[94mMira"},
		{"an NPC", "\x1b[35mGorran Stoutarm"},
		{"your own name in the text", "\x1b[1;32mAria"},
	} {
		if !strings.Contains(ansi, want.code) {
			t.Errorf("%s: no %q in the chat", want.what, want.code)
		}
	}
	if strings.Contains(mono, "\x1b[3") || strings.Contains(mono, "\x1b[9") {
		t.Error("NO_COLOR draws a colour")
	}
	if !strings.Contains(mono, "\x1b[1;4mAria") {
		t.Error("NO_COLOR: your own name is not underlined")
	}
	plain := sgr.ReplaceAllString(mono, "")
	for _, tag := range []string{"[say]", "[party]", "[tell]", "»[say] Gorran Stoutarm: Aria,"} {
		if !strings.Contains(plain, tag) {
			t.Errorf("the screen without colour lacks %q", tag)
		}
	}
	// The palette is one place: the OOC channel and the items extend it there.
	st := screen.ANSI()
	if st.Chat.Say == nil || st.Chat.Server == nil || st.Chat.Mention == nil {
		t.Error("the chat palette has holes")
	}
}

func TestKindOfNameTellsPlayersNPCsAndOwnName(t *testing.T) {
	m := chatWorld()
	for name, want := range map[string]model.NameKind{
		"Aria": model.NameSelf, "aria": model.NameSelf, "Mira": model.NamePlayer, "Gorran Stoutarm": model.NameNPC,
		"Gray Wolf": model.NameOther, "Nobody": model.NameOther, "": model.NameOther,
	} {
		if got := m.KindOfName(name); got != want {
			t.Errorf("%q is %v, want %v", name, got, want)
		}
	}
}

func TestMentionsFindsTheWholeWordInAnyCase(t *testing.T) {
	for _, c := range []struct {
		text, name string
		want       bool
	}{
		{"hello Aria", "Aria", true},
		{"aria, come here", "Aria", true},
		{"ARIA!", "Aria", true},
		{"(aria)", "Aria", true},
		{"Ariana", "Aria", false},
		{"Maria", "Aria", false},
		{"aria's sword", "Aria", false},
		{"no one", "Aria", false},
		{"Aria", "Aria", true},
		{"hi", "", false},
		{"Aria Stone is here", "Aria Stone", true},
	} {
		if got := model.Mentions(c.text, c.name); got != c.want {
			t.Errorf("Mentions(%q, %q) = %v", c.text, c.name, got)
		}
	}
}

// The goldens of the panel, the drawer, the toast and the bubbles, at 80x24 and
// 120x40. `make golden` rewrites them; the test only compares.
func TestChatGoldens(t *testing.T) {
	until := time.Now().Add(time.Hour)
	cases := []struct {
		name string
		set  func(m *model.Model, v *screen.View)
	}{
		{"chat_panel_events", func(m *model.Model, v *screen.View) {}},
		{"chat_panel_chat", func(m *model.Model, v *screen.View) { v.Panel = screen.PanelChat }},
		{"chat_panel_party", func(m *model.Model, v *screen.View) { v.Panel = screen.PanelParty }},
		{"chat_drawer", func(m *model.Model, v *screen.View) { v.Drawer = &screen.DrawerUI{} }},
		{"chat_drawer_under_attack", func(m *model.Model, v *screen.View) {
			combat(m, true)
			v.Drawer = &screen.DrawerUI{Filter: screen.FilterSay}
		}},
		{"chat_toast", func(m *model.Model, v *screen.View) { v.Toast = screen.ToastFor(m.Social.Chat[2]) }},
		{"chat_bubbles", func(m *model.Model, v *screen.View) {
			v.Bubbles = []screen.Bubble{{Name: "Mira", Text: m.Social.Chat[0].Text, Until: until}, {Name: "Aria", Text: "on my way", Until: until}}
		}},
		{"chat_fight", func(m *model.Model, v *screen.View) { combat(m, true); v.Panel = screen.PanelChat }},
		{"chat_names", func(m *model.Model, v *screen.View) {
			structured(m)
			v.Panel = screen.PanelChat
		}},
		{"chat_names_drawer", func(m *model.Model, v *screen.View) {
			structured(m)
			v.Drawer = &screen.DrawerUI{}
		}},
		{"chat_card", func(m *model.Model, v *screen.View) {
			structured(m)
			v.Panel = screen.PanelChat
			v.Person = &screen.CharCard{Name: "Bram", State: screen.CardFound, Kind: "player", Class: "warrior", Level: 4, Zone: "Thornwick", Online: true}
		}},
		{"chat_card_unknown", func(m *model.Model, v *screen.View) {
			structured(m)
			v.Person = &screen.CharCard{Name: "Nobody", State: screen.CardUnknown}
		}},
		{"chat_prompt_ooc", func(m *model.Model, v *screen.View) {
			v.Line = &screen.Line{Channel: "ooc", Text: "anyone for the Warren?"}
		}},
		{"chat_prompt", func(m *model.Model, v *screen.View) {
			v.Line = &screen.Line{Channel: "tell", Target: "Mira", Text: "see you there"}
		}},
	}
	for _, c := range cases {
		for _, sz := range [][2]int{{80, 24}, {120, 40}} {
			m := chatWorld()
			talk(m)
			v := view()
			c.set(m, &v)
			got := screen.Render(m, v, sz[0], sz[1], screen.Plain()) + "\n"
			file := filepath.Join("../testdata/chat", c.name+"_"+strconv.Itoa(sz[0])+"x"+strconv.Itoa(sz[1])+".txt")
			if os.Getenv("SEQ_GOLDEN") == "1" {
				if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			want, err := os.ReadFile(file)
			if err != nil {
				t.Errorf("%s: %v (run make golden)", file, err)
				continue
			}
			if got != string(want) {
				t.Errorf("%s differs from its golden (make golden rewrites it after review)\n--- got\n%s--- want\n%s", file, got, want)
			}
		}
	}
}

// structured adds what a current server sends: an OOC line, a game master's line
// and a line with character references in it.
func structured(m *model.Model) {
	hear(m, "ooc", "Bram", "player", "", "anyone for the Warren, Aria?", txt("anyone for the Warren, "), who("me", "Aria", "player"), txt("?"))
	hear(m, "say", "Hale", "gm", "", "welcome Mira and Gorran", txt("welcome "), who("mira", "Mira", "player"), txt(" and "), who("gorran", "Gorran", "npc"))
	hear(m, "say", "Aria", "player", "", "thanks Hale", txt("thanks "), who("hale", "Hale", "gm"))
}
