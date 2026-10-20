package scenario_test

import (
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/testserver"
)

// TUI-033 (D85): a tap is one 3 m step and a hold walks; auto-retaliate is shown
// and toggled. The zone is the walker's (TUI-031): Testgrounds with no spawns,
// the character at (20, 50) m walking east at 4 m/s.

// A tap on a terminal that reports releases, however quick or slow inside 200 ms,
// is one Move 3 m on: no Walk, no Stop, no coasting past the 3 m, and the mark
// shows the step at once and never goes back.
func TestKittyTapIsOneThreeMetreStep(t *testing.T) {
	w, p := walker(t)
	p.Kitty()
	start := p.Pos()

	p.KeyEvent("right", scenario.KeyDown)
	w.Advance(tick) // 100 ms down: still a tap
	p.KeyEvent("right", scenario.KeyRelease)
	prev := p.Drawn0()
	for range 12 {
		w.Advance(33 * time.Millisecond)
		d := p.Drawn0()
		if d[0] < prev[0] || d[1] != start[1] {
			t.Fatalf("the mark went from %v to %v during a step", prev, d)
		}
		prev = d
	}
	if prev[0] <= start[0] {
		t.Fatalf("the mark did not move for a tap: %v from %v", prev, start)
	}
	w.Advance(10 * tick)
	got := p.Pos()
	if got[0] != start[0]+3000 || got[1] != start[1] {
		t.Fatalf("a tap moved the character from %v to %v, not 3 m east", start, got)
	}
	if d := p.Drawn0(); d != got {
		t.Fatalf("at rest the mark is %v and the server says %v", d, got)
	}
	sends := p.WalkSends()
	if len(sends) != 1 || !sends[0].Step || sends[0].Heading != 0 {
		t.Fatalf("a tap sent %+v", sends)
	}

	// Two keys down and up together are one step along the diagonal.
	from := p.Pos()
	p.KeyEvent("right", scenario.KeyDown)
	p.KeyEvent("up", scenario.KeyDown)
	w.Advance(tick)
	p.KeyEvent("up", scenario.KeyRelease)
	p.KeyEvent("right", scenario.KeyRelease)
	w.Advance(15 * tick)
	got = p.Pos()
	if dx, dy := got[0]-from[0], got[1]-from[1]; abs64(dx-2121) > 5 || abs64(dy-2121) > 5 {
		t.Fatalf("a diagonal tap moved the character %d, %d mm", dx, dy)
	}
	if n := len(p.WalkSends()); n != 2 {
		t.Fatalf("two taps sent %d intents: %+v", n, p.WalkSends())
	}
}

// Held past 200 ms the key is a walk, as in TUI-031, and its release stops it: no
// step is added at the end.
func TestKittyHoldPastATapWalksAndNoStepFollows(t *testing.T) {
	w, p := walker(t)
	p.Kitty()
	start := p.Pos()
	p.KeyEvent("right", scenario.KeyDown)
	for range 8 { // 800 ms
		w.Advance(tick)
	}
	p.KeyEvent("right", scenario.KeyRelease)
	w.Advance(3 * tick)
	stopped := p.Pos()
	if stopped[0]-start[0] < 1200 || stopped[0]-start[0] > 4000 {
		t.Fatalf("a hold of 800 ms walked %d mm", stopped[0]-start[0])
	}
	w.Advance(15 * tick)
	if got := p.Pos(); got != stopped {
		t.Fatalf("the character walked on after the release: %v to %v", stopped, got)
	}
	for _, s := range p.WalkSends() {
		if s.Step {
			t.Fatalf("a hold sent a step: %+v", p.WalkSends())
		}
	}
	if _, stops := counts(p); stops != 1 {
		t.Fatalf("a hold sent %d stops", stops)
	}
}

// A terminal that reports no releases: a single press is one 3 m step, and a
// repeat turns it into a walk, which stops about 150 ms after the last repeat.
func TestLegacyTapIsAStepAndRepeatWalks(t *testing.T) {
	w, p := walker(t)
	start := p.Pos()
	p.Press("right")
	w.Advance(15 * tick)
	got := p.Pos()
	if got[0] != start[0]+3000 || got[1] != start[1] {
		t.Fatalf("a press moved the character from %v to %v, not 3 m east", start, got)
	}
	sends := p.WalkSends()
	if len(sends) != 1 || !sends[0].Step {
		t.Fatalf("a press sent %+v", sends)
	}

	// A hold: the press, the terminal's pause, then a key every 40 ms.
	p.Press("right")
	w.Advance(500 * time.Millisecond)
	for range 25 {
		p.Press("right")
		w.Advance(40 * time.Millisecond)
	}
	walks, stops := counts(p)
	if walks < 2 || stops != 0 {
		t.Fatalf("a held key sent %d walks and %d stops while it repeated", walks, stops)
	}
	w.Advance(5 * tick)
	if _, stops := counts(p); stops != 1 {
		t.Fatalf("the walk was stopped %d times after the repeat ended", stops)
	}
	at := p.Pos()
	w.Advance(10 * tick)
	if got := p.Pos(); got != at {
		t.Fatalf("the character walked on after the stop: %v then %v", at, got)
	}
}

// Ten taps a second for ten seconds, with and without releases reported, are
// never rate limited, denied or protocol errors.
func TestTenTapsASecondForTenSecondsIsNeverRateLimited(t *testing.T) {
	for _, kitty := range []bool{true, false} {
		w := scenario.NewWithWalkTimeout(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria"), Zone: noSpawns}, 0)
		w.Place("ann", "aria", 10000, 80000)
		p := w.Login("ann", "aria")
		if kitty {
			p.Kitty()
		}
		start := p.Pos()
		for range 100 {
			key := "right"
			if kitty {
				p.KeyEvent(key, scenario.KeyDown)
				p.KeyEvent(key, scenario.KeyRelease)
			} else {
				p.Press(key)
			}
			w.Advance(100 * time.Millisecond)
		}
		sends := p.WalkSends()
		for i := 1; i < len(sends); i++ {
			if gap := sends[i].At.Sub(sends[i-1].At); gap < 100*time.Millisecond {
				t.Fatalf("kitty=%v: intents %d and %d were %v apart", kitty, i-1, i, gap)
			}
		}
		if len(sends) < 40 {
			t.Fatalf("kitty=%v: 100 taps sent only %d intents", kitty, len(sends))
		}
		if closed, why := p.Sess.Closed(); closed {
			t.Fatalf("kitty=%v: the server ended the session: %v", kitty, why)
		}
		p.Model(func(m *model.Model) {
			for _, in := range m.Intents {
				if in.State == model.IntentDenied {
					t.Errorf("kitty=%v: an intent was denied: %+v", kitty, in)
				}
			}
			for _, e := range m.Log {
				if e.Reason == "rate_limited" || e.Kind == "rate_limited" {
					t.Errorf("kitty=%v: the server limited the session: %+v", kitty, e)
				}
			}
			if len(m.Errors) > 0 {
				t.Errorf("kitty=%v: protocol errors: %v", kitty, m.Errors)
			}
		})
		if got := p.Pos(); got == start {
			t.Fatalf("kitty=%v: ten seconds of tapping went nowhere", kitty)
		}
	}
}

// Auto-retaliate (D85) is on by default and shown in the You panel; Ctrl+R sends
// the intent, the server's word is in Events, and the panel follows it.
func TestAutoRetaliateIsShownAndToggled(t *testing.T) {
	w, p := walker(t)
	p.Resize(140, 45)
	p.Model(func(m *model.Model) {
		if !m.Self.AutoRetaliate {
			t.Fatal("auto-retaliate is not on for a new character")
		}
	})
	if s := p.Screen(140, 45); !strings.Contains(s, "Auto-retaliate: ON") {
		t.Fatalf("the You panel does not show auto-retaliate on:\n%s", s)
	}
	p.Press("ctrl+r")
	notice(t, p, "auto-retaliate ok", false)
	w.Advance(tick)
	p.Model(func(m *model.Model) {
		if m.Self.AutoRetaliate || !hasLog(m, "auto_retaliate_changed", "off") {
			t.Fatalf("auto-retaliate %v; log %v", m.Self.AutoRetaliate, m.Log)
		}
	})
	s := p.Screen(140, 45)
	if !strings.Contains(s, "Auto-retaliate: off") || !strings.Contains(s, "Auto-retaliate is off") {
		t.Fatalf("the screen does not show it off, in the You panel and in Events:\n%s", s)
	}
	p.Press("ctrl+r")
	w.Advance(tick)
	p.Model(func(m *model.Model) {
		if !m.Self.AutoRetaliate || !hasLog(m, "auto_retaliate_changed", "on") {
			t.Fatalf("auto-retaliate %v; log %v", m.Self.AutoRetaliate, m.Log)
		}
	})
	// The short screen folds it into the attack row; the help names the key.
	if s := p.Screen(80, 24); !strings.Contains(s, "Retaliate ON") && !strings.Contains(s, "Ret ON") {
		t.Fatalf("the 80 x 24 screen does not show auto-retaliate:\n%s", s)
	}
	p.Press("?")
	found := false
	for range 8 { // the keys page, and the page on fighting, name it
		if sc := p.Screen(140, 45); strings.Contains(sc, "Ctrl+R") {
			found = true
			break
		}
		p.Press("right")
	}
	if !found {
		t.Fatalf("no page of the help lists Ctrl+R")
	}
}
