package scenario_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/testserver"
)

// Legibility (D76): the marks, the pointer and the hints along First Steps. The
// server says which person, board or door matters (objective hints); the screen
// marks them by shape, points to the objective when it is off the map, names it
// on the status line, and says which key does what next to a door or a person.

// mapPart is the map box of a screen of that width: its rows, cut before the
// right-hand panels.
func mapPart(screen string, w int) string {
	rw := min(max(w*3/10, 28), 44)
	var out []string
	for _, l := range strings.Split(screen, "\n") {
		if strings.HasPrefix(l, "┌─ Abilities") || strings.HasPrefix(l, "┌─ Events") {
			break
		}
		if r := []rune(l); len(r) >= w {
			out = append(out, string(r[:w-rw]))
		}
	}
	return strings.Join(out, "\n")
}

func lastRow(screen string) string {
	rows := strings.Split(screen, "\n")
	return rows[len(rows)-1]
}

func hintRow(screen string) string {
	rows := strings.Split(screen, "\n")
	return rows[len(rows)-2]
}

var pointer = func(name string) *regexp.Regexp {
	return regexp.MustCompile(`[◄►▲▼◤◥◣◢] ` + name + ` \d+m|` + name + ` \d+m [◄►▲▼◤◥◣◢]`)
}

func TestMarksPointerAndHintsAlongFirstSteps(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})
	p := w.NewCharacter("ann")
	makeWarrior(p, "Hale")

	w.Step("Tamsin has something new", func() {
		for _, sz := range [][2]int{{140, 45}, {80, 24}} {
			s := p.Screen(sz[0], sz[1])
			if sz[0] == 140 && !strings.Contains(s, "! Tamsin Hale") {
				t.Errorf("%v: Nearby does not mark Tamsin Hale with the gold !:\n%s", sz, s)
			}
			if !strings.Contains(mapPart(s, sz[0]), "!") {
				t.Errorf("%v: no ! on the map:\n%s", sz, s)
			}
			if st := lastRow(s); !strings.Contains(st, "→ Talk to Tamsin Hale") {
				t.Errorf("%v: the status line does not name the objective: %q", sz, st)
			}
		}
		s := p.Screen(140, 45)
		for _, want := range []string{"Tamsin Hale <", "[step 1]", "Hawthorn Square"} {
			if !strings.Contains(s, want) {
				t.Errorf("lacks %q:\n%s", want, s)
			}
		}
		if h := hintRow(s); !strings.Contains(h, "G: talk to Tamsin Hale <") {
			t.Errorf("hint line: %q", h)
		}
		p.Golden("marks_tamsin")
	})

	w.Step("then the class trainer, off the map", func() {
		talkTo(t, w, p, "greeter")
		p.Press("esc")
		w.Advance(time.Second)
		for _, sz := range [][2]int{{140, 45}, {80, 24}} {
			s := p.Screen(sz[0], sz[1])
			if !pointer("Gorran").MatchString(mapPart(s, sz[0])) {
				t.Errorf("%v: no pointer to Gorran at the edge of the map:\n%s", sz, s)
			}
			if st := lastRow(s); !strings.Contains(st, "→ Talk to Gorran Stoutarm") {
				t.Errorf("%v: the status line: %q", sz, st)
			}
			if !strings.Contains(s, "<Warrior Trainer>") && sz[0] == 140 {
				t.Errorf("%v: Nearby does not show the title:\n%s", sz, s)
			}
		}
		p.Golden("marks_pointer_trainer")
		walkVia(t, w, p, [2]int64{60000, 22000}, [2]int64{60000, 17000}, [2]int64{16000, 17000})
		if h := hintRow(p.Screen(140, 45)); !strings.Contains(h, "G: talk to Gorran Stoutarm <Warrior Trainer>") {
			t.Errorf("at the trainer, the hint line: %q", h)
		}
		talkTo(t, w, p, "trainer-warrior")
		p.Press("esc")
		w.Advance(time.Second)
	})

	w.Step("then the board", func() {
		walkVia(t, w, p, [2]int64{60000, 17000}, [2]int64{60000, 45000}, [2]int64{57000, 45000}, [2]int64{57000, 51000})
		s := p.Screen(140, 45)
		if !strings.Contains(s, "→ Take a contract") || !strings.Contains(s, "[step 3]") || !strings.Contains(mapPart(s, 140), "&") {
			t.Errorf("at the board:\n%s", s)
		}
		p.Press("g", "1", "enter")
		notice(t, p, "ok", false)
		p.Press("esc", "esc")
		w.Advance(time.Second)
	})

	w.Step("then the East Gate", func() {
		for _, sz := range [][2]int{{140, 45}, {80, 24}} {
			s := p.Screen(sz[0], sz[1])
			if st := lastRow(s); !strings.Contains(st, "→ East Gate") {
				t.Errorf("%v: the status line does not name the gate: %q", sz, st)
			}
		}
		walkVia(t, w, p, [2]int64{108000, 45000})
		s := p.Screen(140, 45)
		if !strings.Contains(s, "[next door]") || !strings.Contains(mapPart(s, 140), "D") {
			t.Errorf("the next door is not marked on the way to it:\n%s", s)
		}
		walkVia(t, w, p, [2]int64{114000, 45000})
		for _, sz := range [][2]int{{140, 45}, {80, 24}} {
			s := p.Screen(sz[0], sz[1])
			if h := hintRow(s); !strings.Contains(h, "E: go through the East Gate to the Thornwick Downs") {
				t.Errorf("%v: the door hint: %q\n%s", sz, h, s)
			}
		}
		p.Golden("marks_door_hint")
		p.Press("e")
		p.Eventually("the Downs", inSpace("Thornwick Downs"))
	})
}
