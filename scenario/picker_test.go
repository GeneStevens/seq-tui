package scenario_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/roster"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/testserver"
)

// TUI-036: the character picker, on the Thornwick server.

func TestPickerEmptyAccountMakesACharacterInThornwickAndListsIt(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})
	path := filepath.Join(t.TempDir(), "roster.json")

	p := w.Pick("ann", path)
	if s := p.Screen(80, 24); !strings.Contains(s, "Make a character") {
		t.Fatalf("an empty account does not open on creation:\n%s", s)
	}
	makeWarrior(p, "Hale")
	p.Model(func(m *model.Model) {
		if m.Map.GetName() != "Thornwick" {
			t.Errorf("a new character starts in %q, not Thornwick", m.Map.GetName())
		}
	})
	r, err := roster.Load(path)
	if err != nil || len(r.Entries) != 1 {
		t.Fatalf("roster after creating: %+v, %v", r, err)
	}
	if e := r.Entries[0]; e.Name != "Hale" || e.Class != "Warrior" || e.Level != 1 || e.Zone != "Thornwick" || e.ID == "" {
		t.Errorf("entry = %+v", e)
	}
}

func TestPickerListsPreselectsAndPlays(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})
	path := filepath.Join(t.TempDir(), "roster.json")
	for i, name := range []string{"Hale", "Brin"} {
		p := w.Pick("ann", path)
		if i > 0 {
			p.Press("n") // the picker lists Hale; N makes another
		}
		makeWarrior(p, name)
		p.LogOut()
	}
	// Brin was played last; Hale first. The roster says when, in its own file.
	r, err := roster.Load(path)
	if err != nil || len(r.Entries) != 2 {
		t.Fatalf("roster: %+v, %v", r, err)
	}
	ids := map[string]string{}
	for i, e := range r.Entries {
		ids[e.Name] = e.ID
		r.Entries[i].Played = time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
		if e.Name == "Hale" {
			r.Entries[i].Played = time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
			r.Entries[i].Level = 3
		}
	}
	if err := r.Save(path); err != nil {
		t.Fatal(err)
	}

	p := w.Pick("ann", path)
	s := p.Screen(80, 24)
	for _, want := range []string{"Choose a character", "Name", "Class", "Level", "Zone", "Last played", "Hale", "Brin", "Warrior", "Thornwick", "Create a new character", "Up, Down", "Enter", "Ctrl+C"} {
		if !strings.Contains(s, want) {
			t.Errorf("the picker lacks %q:\n%s", want, s)
		}
	}
	if !strings.Contains(s, "> Brin") || strings.Contains(s, "> Hale") {
		t.Errorf("the last played (Brin) is not the one chosen:\n%s", s)
	}
	if strings.Index(s, "Brin") > strings.Index(s, "Hale") {
		t.Errorf("not sorted last played first:\n%s", s)
	}
	p.Press("down")
	if s := p.Screen(80, 24); !strings.Contains(s, "> Hale") {
		t.Errorf("Down does not move to Hale:\n%s", s)
	}
	p.Press("enter")
	p.Created()
	p.Model(func(m *model.Model) {
		if m.Self.Name != "Hale" || m.Self.ID != ids["Hale"] {
			t.Errorf("playing %q (%s), want Hale", m.Self.Name, m.Self.ID)
		}
	})
	if s := p.Screen(80, 24); strings.Contains(s, "Choose a character") {
		t.Errorf("the picker is still open:\n%s", s)
	}
}

func TestPickerCreateRowAndBack(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})
	path := filepath.Join(t.TempDir(), "roster.json")
	first := w.Pick("ann", path)
	makeWarrior(first, "Hale")
	first.LogOut()

	p := w.Pick("ann", path)
	p.Press("down") // the create row
	if s := p.Screen(80, 24); !strings.Contains(s, "> + Create a new character") {
		t.Fatalf("the cursor is not on the create row:\n%s", s)
	}
	p.Press("enter")
	if s := p.Screen(80, 24); !strings.Contains(s, "Make a character") {
		t.Fatalf("Enter on the create row does not open creation:\n%s", s)
	}
	p.Press("esc")
	if s := p.Screen(80, 24); !strings.Contains(s, "Choose a character") {
		t.Fatalf("Esc does not go back to the picker:\n%s", s)
	}
	p.Press("n") // N is the shortcut
	p.Type("Mira")
	p.Press("tab", "1", "enter")
	p.Created()
	p.Model(func(m *model.Model) {
		if m.Self.Name != "Mira" || m.Map.GetName() != "Thornwick" {
			t.Errorf("made %q in %q", m.Self.Name, m.Map.GetName())
		}
	})
	if r, _ := roster.Load(path); len(r.Entries) != 2 {
		t.Errorf("roster after the second: %+v", r)
	}
}

func TestPickerDropsACharacterTheServerForgot(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})
	path := filepath.Join(t.TempDir(), "roster.json")
	r := &roster.Roster{}
	r.Note(roster.Entry{ID: "ann-ghost", Name: "Ghost", Class: "Warrior", Level: 9, Zone: "Thornwick", Played: time.Now()})
	if err := r.Save(path); err != nil {
		t.Fatal(err)
	}
	p := w.Pick("ann", path)
	if s := p.Screen(80, 24); !strings.Contains(s, "Ghost") {
		t.Fatalf("no Ghost:\n%s", s)
	}
	p.Press("enter")
	var s string
	for end := time.Now().Add(5 * time.Second); time.Now().Before(end); time.Sleep(20 * time.Millisecond) {
		if s = p.Screen(80, 24); !strings.Contains(s, "Logging in") {
			break
		}
	}
	if strings.Contains(s, "> Ghost") || !strings.Contains(s, "! ") {
		t.Errorf("the missing character is not dropped with a reason:\n%s", s)
	}
	b, _ := os.ReadFile(path)
	if strings.Contains(string(b), "Ghost") {
		t.Errorf("Ghost is still in the roster file: %s", b)
	}
}
