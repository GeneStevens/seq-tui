package scenario_test

import (
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/testserver"
)

// D101, chat on a small screen, in one session: ann1 and bob1 stand side by side
// with a scuttler some way off. Bob speaks (a bubble), mentions ann (a toast), tells
// her (a toast and a reply), and, once the scuttler comes for her, speaks again;
// the fight keeps Events on the panel and, when it is over, one line catches ann up.
func TestChatBubblesToastsReplyAndTheFight(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{
		{Name: "ann", Characters: []string{"ann1"}}, {Name: "bob", Characters: []string{"bob1"}}},
		Zone: scuttlerBy(11500+22000, 57000)})
	w.Place("ann", "ann1", 11500, 57500)
	w.Place("bob", "bob1", 11500, 58500)
	ann := w.Login("ann", "ann1")
	bob := w.Login("bob", "bob1")

	// Bob says a line that has ann's name in it: a bubble by him, a toast, and the
	// line in the log; nothing is read until the Chat tab is.
	saysLine(bob, "hey ann1 over here, come see the road")
	w.Settle()
	s := ann.Screen(120, 40)
	rows := strings.Split(s, "\n")
	if !strings.Contains(s, "“hey ann1 over here, come…”") {
		t.Errorf("no bubble by bob:\n%s", s)
	}
	if last := rows[len(rows)-1]; !strings.Contains(last, "✉ bob1: hey ann1 over here, come see the road") || !strings.Contains(last, "C chat") {
		t.Errorf("no toast for the mention: %q", last)
	}
	if top := panelTopOf(rows); !strings.Contains(top, "Chat •1") || !strings.Contains(top, "[say] bob1: hey ann1") {
		t.Errorf("no badge and ticker: %q", top)
	}
	mustHear(t, ann, "[say] bob1: hey ann1 over here, come see the road")

	// A tell: a toast that says how to reply, and R opens the reply to it. A tell
	// is never a bubble.
	command(bob, "/tell ann1 group for the Warren?")
	w.Settle()
	s = ann.Screen(80, 24)
	rows = strings.Split(s, "\n")
	if last := rows[len(rows)-1]; !strings.Contains(last, "✉ bob1: group for the Warren?") || !strings.HasSuffix(strings.TrimSpace(last), "r reply") {
		t.Errorf("no toast for the tell: %q", last)
	}
	if strings.Contains(s, "“group") {
		t.Errorf("a tell is a bubble:\n%s", s)
	}
	ann.Press("r")
	if last := lastRowOf(ann.Screen(80, 24)); !strings.Contains(last, "[tell bob1] > ") {
		t.Errorf("R did not open a reply: %q", last)
	}
	ann.Type("on my way\r")
	w.Settle()
	mustHear(t, bob, "[tell] ann1: on my way")
	ann.Golden("chat_reply")

	// Looking at Chat clears the badge and the count.
	ann.Press("]")
	s = ann.Screen(80, 24)
	if top := panelTopOf(strings.Split(s, "\n")); !strings.Contains(top, "[Chat]") || strings.Contains(top, "•") || strings.Contains(lastRowOf(s), "✉") {
		t.Errorf("Chat shown but unread remains: %q / %q", top, lastRowOf(s))
	}
	ann.Press("[")

	// The scuttler comes. In the fight bob speaks and tells: chat is a badge and the
	// ticker, Events is the panel, and the drawer would close.
	ann.Press("C")
	for range 8 { // walk toward the scuttler until it takes notice
		attacked := false
		ann.Model(func(m *model.Model) { attacked = m.UnderAttack() })
		if attacked {
			break
		}
		ann.Step("right")
	}
	waitForAttack(t, w, ann)
	if s := ann.Screen(80, 24); strings.Contains(s, "┌─ Chat: all") {
		t.Errorf("the drawer is still open in the fight:\n%s", s)
	}
	saysLine(bob, "watch out ann1")
	command(bob, "/tell ann1 behind you")
	w.Settle()
	ann.Press("]")
	s = ann.Screen(80, 24)
	if top := panelTopOf(strings.Split(s, "\n")); !strings.Contains(top, "[Events:") || !strings.Contains(top, "Chat •2") || !strings.Contains(top, "behind you") {
		t.Errorf("in the fight: %q", top)
	}
	if strings.Contains(lastRowOf(s), "✉ bob1") {
		t.Errorf("a toast on the alarm line: %q", lastRowOf(s))
	}
	ann.Golden("chat_in_fight")

	// When it is over, Events says what came.
	for range 120 {
		over := false
		ann.Model(func(m *model.Model) { over = !m.UnderAttack() })
		if over {
			break
		}
		w.Advance(250 * time.Millisecond)
	}
	ann.Model(func(m *model.Model) {
		if m.UnderAttack() {
			t.Fatal("the fight did not end")
		}
		last := ""
		for _, e := range m.Log {
			if e.Kind == "chat_catchup" {
				last = e.Text
			}
		}
		if last != "While fighting: 2 chat (1 tells)" {
			t.Errorf("the catch-up line is %q", last)
		}
	})
}

func lastRowOf(s string) string {
	rows := strings.Split(s, "\n")
	return rows[len(rows)-1]
}

// panelTopOf is the border row that holds the tabs of the bottom panel.
func panelTopOf(rows []string) string {
	for _, r := range rows {
		if strings.HasPrefix(r, "┌─ ") && (strings.Contains(r, "[Events") || strings.Contains(r, "[Chat]") || strings.Contains(r, "[Party]")) {
			return r
		}
	}
	return ""
}
