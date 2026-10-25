package scenario_test

import (
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/testserver"
)

// cellOf is where the plain screen shows needle: the column and row of its first
// cell, in cells (runes), not bytes.
func cellOf(t *testing.T, s, needle string) (x, y int) {
	t.Helper()
	for y, row := range strings.Split(s, "\n") {
		if i := strings.Index(row, needle); i >= 0 {
			return len([]rune(row[:i])), y
		}
	}
	t.Fatalf("%q is not on the screen:\n%s", needle, s)
	return 0, 0
}

// D101 part 2, in one session: ann1 and bob1 talk out of character, ann switches OOC
// off and on, clicks bob's name to his card, tells him from it, steps to the name by
// key and reports him, and the server acknowledges. Bob's speech is structured, so his
// name is a link.
func TestOOCNameLinksTheCardTellAndReport(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{
		{Name: "ann", Characters: []string{"ann1"}}, {Name: "bob", Characters: []string{"bob1"}}}})
	w.Srv.GM() // starts the report store: without it the server refuses a report
	w.Place("ann", "ann1", 11500, 57500)
	w.Place("bob", "bob1", 11500, 58500)
	ann := w.Login("ann", "ann1")
	bob := w.Login("bob", "bob1")
	ann.Resize(140, 45)
	ann.Press("]") // the Chat tab

	// OOC reaches ann in the Chat tab with its tag, and is never a bubble.
	command(bob, "ooc anyone for the Warren?")
	w.Settle()
	mustHear(t, ann, "[ooc] bob1: anyone for the Warren?")
	s := ann.Screen(140, 45)
	if !strings.Contains(s, "[ooc] bob1: anyone for the Warren?") || strings.Contains(s, "“anyone") {
		t.Errorf("the ooc line is not in the Chat tab as a tagged line, or is a bubble:\n%s", s)
	}

	// The off switch: ann neither hears nor speaks until she turns it on. The client
	// knows the setting from the server's chat settings, at login (on) and on each change.
	oocOff := func() (off bool) {
		ann.Model(func(m *model.Model) { off = m.Social.OOCOff })
		return off
	}
	if oocOff() {
		t.Error("ooc is known off straight after login")
	}
	command(ann, "ooc off")
	w.Settle()
	notice(t, ann, "ooc is off", false)
	if !oocOff() {
		t.Error("the client does not know ooc is off after the switch")
	}
	command(bob, "ooc second line")
	w.Settle()
	if heard(ann, "[ooc] bob1: second line") {
		t.Error("ann heard ooc with it switched off")
	}
	command(ann, "ooc can anyone hear me")
	w.Settle()
	notice(t, ann, "ooc is switched off", true)
	command(ann, "ooc on")
	w.Settle()
	notice(t, ann, "ooc is on", false)
	if oocOff() {
		t.Error("the client still thinks ooc is off after /ooc on")
	}
	w.Advance(12 * time.Second) // ooc has its own small allowance: one line back each 10 seconds
	command(bob, "ooc third line")
	w.Settle()
	mustHear(t, ann, "[ooc] bob1: third line")

	// Bob speaks with ann's name in it: a character reference, so his name is a link.
	saysLine(bob, "hello ann1, well met")
	w.Settle()
	s = ann.Screen(140, 45)
	x, y := cellOf(t, s, "[say] bob1: hello")
	ann.Click(x+len("[say] "), y) // on "bob1"
	if got := ann.Screen(140, 45); !strings.Contains(got, "┌─ Character: bob1") || !strings.Contains(got, "online") || !strings.Contains(got, "T Tell   I Invite   N Ignore   R Report") {
		t.Fatalf("no card for bob1 after the click:\n%s", got)
	}
	ann.Golden("tui043_card")

	// Tell from the card opens the line to him.
	ann.Press("t")
	if last := lastRowOf(ann.Screen(140, 45)); !strings.Contains(last, "[tell bob1] > ") {
		t.Fatalf("Tell did not open the line to bob1: %q", last)
	}
	ann.Type("on my way\r")
	w.Settle()
	mustHear(t, bob, "[tell] ann1: on my way")

	// Report by key: N selects the first name in view, Enter opens its card, R asks why.
	ann.Press("N", "enter")
	w.Settle()
	if got := ann.Screen(140, 45); !strings.Contains(got, "┌─ Character: bob1") {
		t.Fatalf("N and Enter did not open the card:\n%s", got)
	}
	ann.Press("r")
	if last := lastRowOf(ann.Screen(140, 45)); !strings.Contains(last, "[report bob1: why] > ") {
		t.Fatalf("Report did not ask why: %q", last)
	}
	ann.Type("spamming the square\r")
	w.Settle()
	notice(t, ann, "report", false)
	ann.Model(func(m *model.Model) {
		found := false
		for _, e := range m.Log {
			found = found || e.Kind == "notice_report_filed"
		}
		if !found {
			t.Error("the server's acknowledgement of the report is not in the log")
		}
	})
	ann.Golden("tui043_reported")
}
