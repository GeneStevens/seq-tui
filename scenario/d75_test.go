package scenario_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/core"
	"github.com/GeneStevens/seq/testserver"
)

// Danger is never hidden (D75). A scuttler is stood a few metres from the
// vendor, and the character is in a menu when it comes: the menu is not closed,
// its border says so, the status line names the attacker and the character's
// health, and every screen keeps the vitals.

// scuttlerBy puts the zone's scuttler at (x, y).
func scuttlerBy(x, y int) func(string, *core.Zone) {
	return func(_ string, z *core.Zone) {
		for i := range z.Spawns {
			if z.Spawns[i].Template == z.MobTemplates[0].ID {
				z.Spawns[i].Pos = core.Pos{X: x, Y: y}
				return
			}
		}
	}
}

// attackedIn opens a screen with keys, waits for the scuttler to hit the
// character, and checks what the screen says, at both sizes.
// waitForAttack gives game time until something has attacked the character.
func waitForAttack(t *testing.T, w *scenario.World, p *scenario.Player) {
	t.Helper()
	for range 60 {
		attacked := false
		p.Model(func(m *model.Model) { attacked = m.UnderAttack() })
		if attacked {
			return
		}
		w.Advance(250 * time.Millisecond)
	}
	t.Fatal("nothing attacked the character")
}

func attackedIn(t *testing.T, name string, keys []string, shows string, leave int, early bool) {
	t.Helper()
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria"), Zone: scuttlerBy(23500, 57000)})
	w.Place("ann", "aria", 11500, 57500)
	p := w.Login("ann", "aria")
	p.Press(keys...)
	if s := p.Screen(140, 45); !strings.Contains(s, shows) {
		t.Fatalf("%s: the screen is not open:\n%s", name, s)
	}
	p.Model(func(m *model.Model) {
		if m.UnderAttack() && !early { // a talk waits on the server's answer, and the clock runs meanwhile
			t.Fatalf("%s: attacked before the mob came", name)
		}
	})
	waitForAttack(t, w, p)
	var mob string
	p.Model(func(m *model.Model) {
		if len(m.Self.Attackers) != 1 || m.Alert == nil || m.Alert.Name != "Plain Scuttler" || !m.Self.InCombat {
			t.Fatalf("%s: attackers %v, alert %+v, in combat %v", name, m.Self.Attackers, m.Alert, m.Self.InCombat)
		}
		mob = m.Self.Attackers[0]
	})

	for _, sz := range [][2]int{{140, 45}, {80, 24}} {
		s := p.Screen(sz[0], sz[1])
		rows := strings.Split(s, "\n")
		status := rows[len(rows)-1]
		for _, want := range []string{"⚠ Plain Scuttler", "is attacking you", "HP "} {
			if !strings.Contains(status, want) {
				t.Errorf("%s at %v: the status line lacks %q:\n%s", name, sz, want, s)
			}
		}
		if !strings.Contains(s, shows) {
			t.Errorf("%s at %v: the screen was closed by the attack:\n%s", name, sz, s)
		}
		if !strings.Contains(rows[0]+s, "UNDER ATTACK") {
			t.Errorf("%s at %v: the border does not say it is under attack:\n%s", name, sz, s)
		}
		if lower := strings.ToLower(s); !strings.Contains(lower, "attacked by plain scuttler") && !strings.Contains(lower, "plain scuttler 40/40, attacking you") {
			t.Errorf("%s at %v: the vitals do not name the attacker:\n%s", name, sz, s)
		}
	}
	p.Golden(name + "_attacked")

	// Leaving the menu leaves the attacker targeted: one key fights back.
	for range leave {
		p.Press("esc")
	}
	p.Model(func(m *model.Model) {
		if m.Self.Target != mob {
			t.Fatalf("%s: target %q after leaving the menu; the attacker is %q", name, m.Self.Target, mob)
		}
	})
	p.Press("f")
	notice(t, p, "attack ok", false)
}

func TestAttackedInTheVendorWindow(t *testing.T) {
	attackedIn(t, "vendor", []string{"n", "enter"}, "For sale", 2, false)
}

func TestAttackedInTheGearScreen(t *testing.T) {
	attackedIn(t, "gear", []string{"c"}, "Gear worn", 1, false)
}

func TestAttackedInAConversation(t *testing.T) {
	attackedIn(t, "talk", []string{"g"}, "What they can do for you", 1, true)
}

// Trading is refused while fighting (D75), in words, in the panel being looked
// at as well as on the status line.
func TestARefusalIsShownInThePanel(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria"), Zone: scuttlerBy(23500, 57000)})
	w.Place("ann", "aria", 11500, 57500)
	p := w.Login("ann", "aria")
	p.Press("n", "enter")
	waitForAttack(t, w, p)
	p.Press("enter") // buy
	notice(t, p, "can't do that while you're fighting", true)
	for _, sz := range [][2]int{{140, 45}, {80, 24}} {
		s := p.Screen(sz[0], sz[1])
		rows := strings.Split(s, "\n")
		if !strings.Contains(strings.Join(rows[:len(rows)-1], "\n"), "! You can't do that while you're fighting") {
			t.Errorf("at %v the refusal is not in the panel:\n%s", sz, s)
		}
		// On the status line the alarm stays and the answer follows it, cut if it must be.
		if st := rows[len(rows)-1]; !strings.Contains(st, "⚠ Plain Scuttler") || !strings.Contains(st, "! You can't") {
			t.Errorf("at %v the status line: %q", sz, st)
		}
	}
}

// ids and coordinates are for the program, not the player.
var idLike = regexp.MustCompile(`\b(mob|drop|enc|inst|comp|npc)-[a-z0-9.]+|\(-?\d+, -?\d+\)|\bdrop-`)

// noIDs fails when a line of the log that the short view shows holds an id or a
// coordinate (D76).
func noIDs(t *testing.T, m *model.Model) {
	t.Helper()
	for _, e := range m.Log {
		if e.Quiet {
			continue
		}
		if idLike.MatchString(e.Text) {
			t.Errorf("the log shows an id or a coordinate: %q (%s)", e.Text, e.Kind)
		}
	}
}
