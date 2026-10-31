package scenario_test

import (
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/testserver"
)

// TUI-036, TUI-037: the character picker, on the Thornwick server. It lists
// the account's characters as the server has them.

// outsideAccount is an account with two characters that were made some other
// way than through this program (the server has them already): Hale in the
// Downs, Brin in Thornwick.
func outsideAccount() []testserver.Account {
	return []testserver.Account{{Name: "ann", Home: scenario.Thornwick,
		Classed: map[string]string{"hale": "warrior", "brin": "warrior"},
		Names:   map[string]string{"hale": "Hale", "brin": "Brin"}}}
}

func TestPickerEmptyAccountMakesACharacterInThornwickAndListsIt(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})

	p := w.Pick("ann")
	if s := p.Screen(80, 24); !strings.Contains(s, "Make a character") {
		t.Fatalf("an empty account does not open on creation:\n%s", s)
	}
	makeWarrior(p, "Hale")
	p.Model(func(m *model.Model) {
		if m.Map.GetName() != "Thornwick" {
			t.Errorf("a new character starts in %q, not Thornwick", m.Map.GetName())
		}
	})
	p.LogOut()

	q := w.Pick("ann") // the server now lists it
	s := q.Screen(80, 24)
	for _, want := range []string{"Choose a character", "> Hale", "Warrior", "Thornwick"} {
		if !strings.Contains(s, want) {
			t.Errorf("the picker lacks %q:\n%s", want, s)
		}
	}
}

// Characters the server has but this program never made are listed, and the
// one played last is chosen.
func TestPickerListsTheAccountsCharactersAndPreselectsTheNewest(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: outsideAccount()})

	p := w.Pick("ann")
	s := p.Screen(80, 24)
	for _, want := range []string{"Choose a character", "Name", "Class", "Level", "Zone", "Last played", "Hale", "Brin", "Warrior", "Thornwick", "Create a new character", "Up, Down", "Enter", "Ctrl+C"} {
		if !strings.Contains(s, want) {
			t.Errorf("the picker lacks %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "ann-") || strings.Contains(s, "roster") {
		t.Errorf("the picker shows something that is not the account's list:\n%s", s)
	}
	// Play Hale; the server notes when, so Hale is the newest played next time.
	for _, name := range []string{"Brin", "Hale"} {
		choose(t, p, name)
		p.Press("enter")
		p.Created()
		p.Model(func(m *model.Model) {
			if m.Self.Name != name {
				t.Errorf("playing %q, want %s", m.Self.Name, name)
			}
		})
		p.LogOut()
		time.Sleep(5 * time.Millisecond)
		p = w.Pick("ann")
	}
	s = p.Screen(80, 24)
	if !strings.Contains(s, "> Hale") || strings.Contains(s, "> Brin") {
		t.Errorf("the last played (Hale) is not the one chosen:\n%s", s)
	}
	if strings.Index(s, "Hale") > strings.Index(s, "Brin") {
		t.Errorf("not sorted last played first:\n%s", s)
	}
	p.Press("enter")
	p.Created()
	p.Model(func(m *model.Model) {
		if m.Self.Name != "Hale" {
			t.Errorf("playing %q, want Hale", m.Self.Name)
		}
	})
	if s := p.Screen(80, 24); strings.Contains(s, "Choose a character") {
		t.Errorf("the picker is still open:\n%s", s)
	}
}

func TestPickerCreateRowAndBack(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: outsideAccount()})

	p := w.Pick("ann")
	p.Press("down", "down") // past both characters: the create row
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
	p.LogOut()
	if s := w.Pick("ann").Screen(80, 24); !strings.Contains(s, "Mira") || !strings.Contains(s, "Hale") {
		t.Errorf("the server's list lacks the new character:\n%s", s)
	}
}

// A character a connection is playing is listed as online, and picking it is
// refused by the server with a reason; the picker stays open and the row stays.
func TestPickerShowsOnlineCharacterAndRefusesIt(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: outsideAccount()})
	w.Login("ann", "hale")

	p := w.Pick("ann")
	s := p.Screen(80, 24)
	var hale string
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, "Hale") {
			hale = l
		}
	}
	if !strings.Contains(hale, "online") {
		t.Fatalf("Hale is not shown online (%q):\n%s", hale, s)
	}
	if strings.Contains(s, "Brin") && strings.Contains(lineWith(s, "Brin"), "online") {
		t.Errorf("Brin is shown online:\n%s", s)
	}
	choose(t, p, "Hale")
	p.Press("enter")
	var s2 string
	for end := time.Now().Add(5 * time.Second); time.Now().Before(end); time.Sleep(20 * time.Millisecond) {
		if s2 = p.Screen(80, 24); !strings.Contains(s2, "Logging in") {
			break
		}
	}
	if !strings.Contains(s2, "! ") || !strings.Contains(s2, "character_in_use") || !strings.Contains(s2, "> Hale") {
		t.Errorf("picking an online character is not refused with a reason, or the row left the list:\n%s", s2)
	}
}

func lineWith(s, sub string) string {
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, sub) {
			return l
		}
	}
	return ""
}

// choose moves the cursor onto the named character.
func choose(t *testing.T, p *scenario.Player, name string) {
	t.Helper()
	for i := 0; i < 5; i++ {
		if strings.Contains(p.Screen(80, 24), "> "+name) {
			return
		}
		p.Press("down")
	}
	t.Fatalf("no row for %s:\n%s", name, p.Screen(80, 24))
}

// TUI-070: an open picker asks the server for the list on its own. Hale is in
// play on another connection; when that connection leaves, Hale's line stops
// saying "online (in use)" with no key pressed.
func TestPickerLineFreesItselfWhenTheOtherSessionLeaves(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: outsideAccount()})
	other := w.Login("ann", "hale")
	p := w.Pick("ann")
	if l := lineWith(p.Screen(80, 24), "Hale"); !strings.Contains(l, "online (in use)") {
		t.Fatalf("Hale is not shown in use:\n%s", p.Screen(80, 24))
	}
	other.LogOut()
	w.Advance(7 * time.Second) // past the picker's refresh interval; the real redraw tick does the asking
	var s string
	for end := time.Now().Add(5 * time.Second); time.Now().Before(end); time.Sleep(20 * time.Millisecond) {
		if s = p.Screen(80, 24); !strings.Contains(lineWith(s, "Hale"), "online") {
			break
		}
	}
	if l := lineWith(s, "Hale"); l == "" || strings.Contains(l, "online") {
		t.Errorf("Hale's line did not change to free by itself:\n%s", s)
	}
}
