package scenario_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/testserver"
)

// The ? help opens over every view and closes back to it (D62): the page keys
// turn it, Esc closes it, and the view underneath is as it was.
func TestTheHelpOpensAndClosesFromEveryView(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})
	p := w.NewCharacter("ann")
	view := func(want string) {
		t.Helper()
		if s := p.Screen(80, 24); !strings.Contains(s, want) {
			t.Fatalf("the view is not %q:\n%s", want, s)
		}
	}
	helpOpen := func(open bool) {
		t.Helper()
		if s := p.Screen(80, 24); strings.Contains(s, "┌─ Help: ") != open {
			t.Fatalf("help open %v wanted:\n%s", open, s)
		}
	}
	// The creation screen: the help opens there too.
	view("New character")
	p.Press("?")
	helpOpen(true)
	p.Press("esc")
	helpOpen(false)
	view("New character")
	makeWarrior(p, "Hale")

	views := []struct {
		name, title string
		open        []string
	}{
		{"the map", "Thornwick — Hawthorn Square, at", nil},
		{"inventory", "Inventory and gear", []string{"c"}},
		{"traders", "Traders", []string{"n"}},
		{"slots", "Ability slots", []string{"b"}},
		{"contracts", "Contracts", []string{"k"}},
		{"overview", "Zone overview", []string{"m"}},
		{"quest chain", "Quest chain", []string{"ctrl+k"}},
		{"instances", "Instances", []string{"i"}},
		{"sessions", "Sessions: your logs", []string{"ctrl+l"}},
		{"talking", "Talking", []string{"g"}},
	}
	for _, v := range views {
		w.Step(v.name, func() {
			p.Press(v.open...)
			view(v.title)
			key := "?"
			if v.name == "talking" || v.name == "sessions" {
				key = "f1" // the other way in
			}
			p.Press(key)
			helpOpen(true)
			p.Press("esc") // Esc closes the help, not the view under it
			helpOpen(false)
			view(v.title)
			p.Press("esc")
			if len(v.open) > 0 {
				if s := p.Screen(80, 24); strings.Contains(s, "┌─ "+v.title+" ─") {
					t.Fatalf("%s: Esc did not close the view:\n%s", v.name, s)
				}
			}
		})
	}

	// The command line takes ? as a letter: F1 is its key to the help.
	p.Press("/")
	p.Press("f1")
	helpOpen(true)
	p.Press("esc")
	helpOpen(false)
	p.Press("esc")

	// ? turns the pages and closes after the last; Left and Right turn them;
	// Down scrolls the page of keys.
	p.Press("?")
	for page := 1; page <= scenarioHelpPages; page++ {
		view("Page " + string(rune('0'+page)) + " of 8")
		p.Press("?")
	}
	helpOpen(false)
	p.Press("?", "right", "right", "right", "right", "right", "right")
	view("Help: Every key")
	p.Press("left")
	view("Help: Logs and replay")
	p.Press("esc")
}

const scenarioHelpPages = 8

// The help covers D62, D66, D68 and D69 in words a player can use, and the
// page of keys is the table's.
func TestTheHelpSaysWhatAPlayerNeeds(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})
	p := w.NewCharacter("ann")
	makeWarrior(p, "Hale")
	p.Press("?")
	p.Golden("help_page_one")
	var all strings.Builder
	for range scenarioHelpPages {
		all.WriteString(p.Screen(140, 45))
		p.Press("?")
	}
	text := strings.Join(strings.Fields(strings.ReplaceAll(all.String(), "│", " ")), " ")
	for _, want := range []string{
		"First Steps", "optional", "Ctrl+K", "/dismiss", "never offered again", // D62
		"any contract board in town",                                 // D66
		"whatever is attacking you", "Your walk stopped", "back off", // D66
		"fixed people even before you have seen them",                                                          // D69
		"trivial, easy, even, hard or deadly", "breaks off the hexer's bolt", "heals the mob it fights beside", // D70
		"runs for home once", "joins the fight",
		"Logs and replay", "Ctrl+L", // D64
		"Ctrl+K", "? or F1", "Arrows, W A S D",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the help does not say %q", want)
		}
	}
}
