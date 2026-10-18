package app

import (
	"slices"
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
	"github.com/GeneStevens/seq-tui/screen"
)

// The help's list of keys is drawn from screen.Bindings. These tests hold the
// key handler to that table, both ways, so the help cannot drift from the keys.

func candidateKeys() []string {
	var keys []string
	for c := 'a'; c <= 'z'; c++ {
		keys = append(keys, string(c), "ctrl+"+string(c))
	}
	for c := '0'; c <= '9'; c++ {
		keys = append(keys, string(c))
	}
	keys = append(keys, strings.Split("/ ? . , ; ' - = [ ] \\ ` ! @ # $ % ^ & * ( ) _ + : \" < > |", " ")...)
	keys = append(keys, "up", "down", "left", "right", "tab", "esc", "enter", "space", " ", "backspace", "delete", "home", "end", "pgup", "pgdown", "shift+tab")
	for n := 1; n <= 12; n++ {
		keys = append(keys, "f"+string(rune('0'+n%10)), "f1"+string(rune('0'+n%10)))
	}
	return keys
}

func zero(a Action) bool { return a.Kind == None && a.Note == "" }

func bindingModel(t *testing.T) *model.Model {
	t.Helper()
	msgs, err := record.Load("../testdata/logs", "play.jsonl@play_login")
	if err != nil {
		t.Fatal(err)
	}
	m := model.New()
	for _, msg := range msgs {
		m.Apply(msg)
	}
	m.Self.Target = "somebody" // Esc clears a target
	return m
}

func TestEveryKeyTheMainViewHandlesIsInTheBindingTable(t *testing.T) {
	m := bindingModel(t)
	listed := screen.HelpBindingKeys()
	for _, k := range candidateKeys() {
		if zero(Plan(k, m)) {
			continue
		}
		if !slices.Contains(listed, k) && !(k == " " && slices.Contains(listed, "space")) {
			t.Errorf("key %q does something (%+v) and is not in the help's binding table", k, Plan(k, m))
		}
	}
}

func TestEveryKeyInTheBindingTableDoesSomething(t *testing.T) {
	m := bindingModel(t)
	for _, b := range screen.Bindings {
		for _, k := range b.Keys {
			switch b.When {
			case "", "turn":
				if b.When == "turn" {
					continue // TurnKey is held to the same table below
				}
				if zero(Plan(k, m)) {
					t.Errorf("%s: key %q is in the binding table and does nothing", b.Show, k)
				}
			case "companion", "duo":
				if !helperKey(k) && k != "x" {
					t.Errorf("%s: key %q is listed as a helper's and is not one", b.Show, k)
				}
			}
		}
	}
}

func TestTheHelpersKeysAreTheTablesAndTheTurnKeysToo(t *testing.T) {
	helpers := map[string]bool{}
	for _, b := range screen.Bindings {
		if b.When == "companion" || b.When == "duo" {
			for _, k := range b.Keys {
				helpers[k] = true
			}
		}
	}
	for _, k := range candidateKeys() {
		if helperKey(k) && !helpers[k] {
			t.Errorf("helper key %q is not in the binding table", k)
		}
	}
	// The plan keys of a turn-based round that the table names do something there.
	m := bindingModel(t)
	d := &Draft{}
	for _, b := range screen.Bindings {
		if b.When != "turn" {
			continue
		}
		for _, k := range b.Keys {
			if zero(TurnKey(k, m, d)) {
				t.Errorf("%s: plan key %q does nothing", b.Show, k)
			}
		}
	}
}

func TestTheKeysPageListsEveryBindingTheCharacterCanUse(t *testing.T) {
	m := bindingModel(t)
	page := strings.Join(screen.KeyLines(m, screen.View{}, 200), "\n")
	for _, b := range screen.Bindings {
		if b.When != "" && b.When != "turn" {
			continue
		}
		if !strings.Contains(page, b.Show) || !strings.Contains(page, b.What) {
			t.Errorf("the keys page lacks %q: %s", b.Show, b.What)
		}
	}
	// A companion's keys are listed for a character that has one, and a second
	// character's for a client that has one.
	if strings.Contains(page, "call your companion") || strings.Contains(page, "second character") {
		t.Errorf("helper keys listed for a character with none:\n%s", page)
	}
	with := strings.Join(screen.KeyLines(m, screen.View{Duo: &screen.DuoView{}}, 200), "\n")
	if !strings.Contains(with, "bring your second character in or out") {
		t.Errorf("a client with a second character is not told its keys:\n%s", with)
	}
}
