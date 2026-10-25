package screen_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// D101 §8 and §9: names coloured by kind, and names that are links.

func txt(s string) *pb.ChatPart {
	return &pb.ChatPart{Body: &pb.ChatPart_Text{Text: s}}
}

func who(id, name, kind string) *pb.ChatPart {
	return &pb.ChatPart{Body: &pb.ChatPart_Character{Character: &pb.CharacterRef{Id: id, Name: name, Kind: kind}}}
}

// hear applies a structured line of chat: the speaker, the text and its parts.
func hear(m *model.Model, channel, from, kind, to, text string, parts ...*pb.ChatPart) {
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Chat{Chat: &pb.ChatMessage{Channel: channel, From: from, To: to, Text: text,
		Speaker: &pb.CharacterRef{Id: strings.ToLower(from), Name: from, Kind: kind}, Parts: parts}}})
}

// chatOnly is a world whose chat is just what the test says, shown in the Chat tab.
func chatOnly() (*model.Model, screen.View) {
	m := chatWorld()
	v := view()
	v.Panel = screen.PanelChat
	v.Hits = &screen.Hits{}
	return m, v
}

func TestNamesAreColouredByTheKindTheServerGave(t *testing.T) {
	ansi := screen.ANSI()
	mono := screen.Mono()
	for _, c := range []struct {
		name  string
		st    screen.Styler
		build func(m *model.Model)
		want  []string // each must be in the drawn chat, escapes and all
		not   []string
	}{
		{"a player's name in the text, and the speaker's", ansi, func(m *model.Model) {
			hear(m, "say", "Bram", "player", "", "hello Mira", txt("hello "), who("mira", "Mira", "player"))
		}, []string{"\x1b[94mBram\x1b[0m", "\x1b[94mMira\x1b[0m"}, nil},
		{"an NPC", ansi, func(m *model.Model) {
			hear(m, "say", "Bram", "player", "", "see Gorran", txt("see "), who("gorran", "Gorran", "npc"))
		}, []string{"\x1b[35mGorran\x1b[0m"}, nil},
		{"a game master speaks in the server's style", ansi, func(m *model.Model) {
			hear(m, "say", "Hale", "gm", "", "welcome Mira", txt("welcome "), who("mira", "Mira", "player"))
		}, []string{"\x1b[1;33mHale\x1b[0m", "\x1b[1;33m[say]\x1b[0m", "\x1b[94mMira\x1b[0m"}, nil},
		{"your own name has its own colour, and the line is marked", ansi, func(m *model.Model) {
			hear(m, "say", "Bram", "player", "", "hi Aria", txt("hi "), who("me", "Aria", "player"))
		}, []string{"\x1b[1m\x1b[1;32mAria\x1b[0m\x1b[0m", "»"}, nil},
		{"your own name without colour is underlined and the line bold", mono, func(m *model.Model) {
			hear(m, "say", "Bram", "player", "", "hi Aria", txt("hi "), who("me", "Aria", "player"))
		}, []string{"\x1b[1;4mAria\x1b[0m", "»"}, nil},
		{"an ooc line has its own colour and its tag", ansi, func(m *model.Model) {
			hear(m, "ooc", "Bram", "player", "", "anyone for the Warren?", txt("anyone for the Warren?"))
		}, []string{"\x1b[90m[ooc]\x1b[0m", "\x1b[90manyone\x1b[0m"}, nil},
		{"an old server sends no parts: the TUI-042 rendering, no colour for what is not known", ansi, func(m *model.Model) {
			m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Chat{Chat: &pb.ChatMessage{Channel: "say", From: "Mira", Text: "hi Aria"}}})
		}, []string{"\x1b[94mMira\x1b[0m", "\x1b[1;32mAria\x1b[0m"}, nil},
	} {
		m, v := chatOnly()
		c.build(m)
		out := screen.Render(m, v, 120, 40, c.st)
		for _, w := range c.want {
			if !strings.Contains(out, w) {
				t.Errorf("%s: no %q in\n%s", c.name, w, out)
			}
		}
	}
}

// names are the plain text under each name the screen noted as a link.
func names(t *testing.T, m *model.Model, v screen.View, w, h int) (got []string, hits []screen.NameHit) {
	t.Helper()
	rows := rowsOf(screen.Render(m, v, w, h, screen.Plain()))
	for _, n := range v.Hits.Names {
		cells := []rune(rows[n.Rect.Y])
		got = append(got, string(cells[n.Rect.X:n.Rect.X+n.Rect.W]))
	}
	return got, v.Hits.Names
}

func TestEveryNameIsALinkWhereItWasDrawnExceptYourOwnAndTheServer(t *testing.T) {
	for _, c := range []struct {
		name  string
		build func(m *model.Model)
		want  []string
	}{
		{"speaker and a name in the text", func(m *model.Model) {
			hear(m, "say", "Bram", "player", "", "hello Mira, and Gorran", txt("hello "), who("mira", "Mira", "player"), txt(", and "), who("gorran", "Gorran", "npc"))
		}, []string{"Bram", "Mira", "Gorran"}},
		{"your own name opens nothing", func(m *model.Model) {
			hear(m, "say", "Bram", "player", "", "hi Aria", txt("hi "), who("me", "Aria", "player"))
		}, []string{"Bram"}},
		{"your own lines have no speaker link; the one told does", func(m *model.Model) {
			hear(m, "tell", "Aria", "player", "Bram", "psst", txt("psst"))
		}, []string{"Bram"}},
		{"a game master is a link, the server is not", func(m *model.Model) {
			hear(m, "say", "Hale", "gm", "", "x", txt("x"))
			hear(m, "say", "Server", "server", "", "y", txt("y"))
		}, []string{"Hale"}},
		{"an old server has no links", func(m *model.Model) {
			m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Chat{Chat: &pb.ChatMessage{Channel: "say", From: "Mira", Text: "hi"}}})
		}, nil},
		{"a name cut across rows is a link on each row", func(m *model.Model) {
			hear(m, "say", "Bram", "player", "", strings.Repeat("long ", 12)+"Mira", txt(strings.Repeat("long ", 12)), who("mira", "Mira", "player"))
		}, []string{"Bram", "Mira"}},
	} {
		m, v := chatOnly()
		c.build(m)
		got, hits := names(t, m, v, 80, 24)
		if strings.Join(got, ",") != strings.Join(c.want, ",") {
			t.Errorf("%s: linked %q, want %q", c.name, got, c.want)
		}
		for i, h := range hits {
			if h.Name != c.want[i] {
				t.Errorf("%s: link %d opens %q", c.name, i, h.Name)
			}
		}
	}
}

func TestNamesInTheDrawerAreLinksAndTheTabUnderItIsNot(t *testing.T) {
	m, v := chatOnly()
	hear(m, "say", "Bram", "player", "", "hello Mira", txt("hello "), who("mira", "Mira", "player"))
	v.Drawer = &screen.DrawerUI{}
	for _, size := range [][2]int{{80, 24}, {120, 40}} {
		got, hits := names(t, m, v, size[0], size[1])
		if strings.Join(got, ",") != "Bram,Mira" {
			t.Errorf("%v: drawer linked %q (the tab under it must not add its own)", size, got)
		}
		if !v.Hits.Drawer.In(hits[0].Rect.X, hits[0].Rect.Y) {
			t.Errorf("%v: %+v is not inside the drawer %+v", size, hits[0], v.Hits.Drawer)
		}
	}
}

func TestTheSelectedNameIsInReverseAndTheOthersAreNot(t *testing.T) {
	m, v := chatOnly()
	hear(m, "say", "Bram", "player", "", "hello Mira", txt("hello "), who("mira", "Mira", "player"))
	for sel, want := range map[int]string{0: "\x1b[94m\x1b[7mBram\x1b[0m\x1b[0m", 1: "\x1b[94m\x1b[7mMira\x1b[0m\x1b[0m"} {
		v.NameSel = &sel
		out := screen.Render(m, v, 120, 40, screen.ANSI())
		if strings.Count(out, "\x1b[7m") < 1 || !strings.Contains(out, want) {
			t.Errorf("name %d selected: %q not drawn", sel, want)
		}
	}
	v.NameSel = nil
	if out := screen.Render(m, v, 120, 40, screen.ANSI()); strings.Contains(out, "\x1b[94m\x1b[7m") {
		t.Errorf("a name is selected with no selection")
	}
}

func TestTheCardInEachState(t *testing.T) {
	for _, c := range []struct {
		name string
		card screen.CharCard
		want []string
		acts bool
	}{
		{"online", screen.CharCard{Name: "Bram", State: screen.CardFound, Kind: "player", Class: "warrior", Level: 4, Zone: "Thornwick", Online: true}, []string{"Bram", "Level 4 warrior", "online, in Thornwick"}, true},
		{"online with a status", screen.CharCard{Name: "Bram", State: screen.CardFound, Class: "mage", Level: 9, Zone: "Downs", Online: true, Status: "away"}, []string{"Level 9 mage", "status: away"}, true},
		{"offline", screen.CharCard{Name: "Bram", State: screen.CardFound, Class: "warrior", Level: 4}, []string{"Level 4 warrior", "offline"}, true},
		{"anonymous", screen.CharCard{Name: "Bram", State: screen.CardFound, Class: "warrior", Level: 4, Online: true}, []string{"online, anonymous: the zone is hidden"}, true},
		{"level not known", screen.CharCard{Name: "Bram", State: screen.CardFound, Class: "warrior"}, []string{"Level unknown warrior"}, true},
		{"a game master", screen.CharCard{Name: "Hale", State: screen.CardFound, Kind: "gm", Class: "warrior", Level: 1, Online: true, Zone: "Thornwick"}, []string{"Hale  (game master)"}, true},
		{"unknown", screen.CharCard{Name: "Nobody", State: screen.CardUnknown}, []string{"no character by that name"}, false},
		{"asking", screen.CharCard{Name: "Bram", State: screen.CardAsking}, []string{"asking the server…"}, true},
		{"not answered", screen.CharCard{Name: "Bram", State: screen.CardFailed, Err: "timed out"}, []string{"could not ask: timed out"}, false},
	} {
		m, v := chatOnly()
		v.Person = &c.card
		out := screen.Render(m, v, 120, 40, screen.Plain())
		if !strings.Contains(out, "┌─ Character: "+c.card.Name) {
			t.Errorf("%s: no card:\n%s", c.name, out)
		}
		for _, w := range c.want {
			if !strings.Contains(out, w) {
				t.Errorf("%s: no %q:\n%s", c.name, w, out)
			}
		}
		if got := strings.Contains(out, "T Tell   I Invite   N Ignore   R Report"); got != c.acts {
			t.Errorf("%s: actions drawn %v, want %v", c.name, got, c.acts)
		}
		if acts := len(v.Hits.CardActs); (acts == screen.CardActions) != c.acts || (acts != 0 && acts != screen.CardActions) {
			t.Errorf("%s: %d action hits", c.name, acts)
		}
		if v.Hits.Card.W == 0 {
			t.Errorf("%s: the card's block is not noted", c.name)
		}
	}
}

func TestTheCardsActionsAreWhereTheyWereDrawn(t *testing.T) {
	m, v := chatOnly()
	v.Person = &screen.CharCard{Name: "Bram", State: screen.CardFound, Level: 4, Class: "warrior", Zone: "Thornwick", Online: true}
	rows := rowsOf(screen.Render(m, v, 80, 24, screen.Plain()))
	for _, a := range v.Hits.CardActs {
		cells := []rune(rows[a.Rect.Y])
		got := string(cells[a.Rect.X : a.Rect.X+a.Rect.W])
		want := map[int]string{screen.ActTell: "T Tell", screen.ActInvite: "I Invite", screen.ActIgnore: "N Ignore", screen.ActReport: "R Report"}[a.Tab]
		if got != want {
			t.Errorf("action %d is at %q, want %q", a.Tab, got, want)
		}
	}
}
