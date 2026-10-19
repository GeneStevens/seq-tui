package scenario_test

import (
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/testserver"
)

// The quest chain of the first session (D62, D68): First Steps on the pinned
// Thornwick packs. The scenarios press the keys of a player.

// talkTo presses G until the person talked to is id (a patrolling guard can be
// the nearest for a moment) and closes the conversation.
func talkTo(t *testing.T, w *scenario.World, p *scenario.Player, id string) {
	t.Helper()
	for range 8 {
		p.Press("g")
		said := false
		p.Model(func(m *model.Model) { said = m.Said(id) != nil })
		if said {
			return
		}
		p.Press("esc")
		w.Advance(3 * time.Second)
	}
	t.Fatalf("could not talk to %s", id)
}

func quest(p *scenario.Player) (name string, step, steps int64, text string) {
	p.Model(func(m *model.Model) {
		if q := m.Quest(); q != nil {
			name, step, steps, text = q.GetName(), q.GetStep(), q.GetSteps(), q.GetText()
		}
	})
	return
}

func wantStep(t *testing.T, p *scenario.Player, step int64, textHas string) {
	t.Helper()
	name, got, steps, text := quest(p)
	if name != "First Steps" || got != step || steps != 4 || !strings.Contains(text, textHas) {
		t.Fatalf("the chain is %q step %d of %d %q; want First Steps step %d of 4 about %q", name, got, steps, text, step, textHas)
	}
	// The step is on the screen: in the status line and in the Quest chain panel.
	if s := p.Screen(80, 24); !strings.Contains(s, "First Steps "+string(rune('0'+step))+"/4") {
		t.Fatalf("the status line does not name the step:\n%s", s)
	}
	s := p.Screen(140, 45)
	for _, want := range []string{"Quest chain", "First Steps", "Step " + string(rune('0'+step)) + " of 4"} {
		if !strings.Contains(s, want) {
			t.Fatalf("the Quest chain panel lacks %q:\n%s", want, s)
		}
	}
}

// A new character is offered First Steps, sees the step on the status line, and
// walks greeter, class trainer, board and east gate; the 15 crowns come at the
// trainer and the 20 xp after crossing the gate.
func TestFirstStepsFromTheGreeterToTheEastGate(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})
	w.Fixture("first_steps.jsonl")
	p := w.NewCharacter("ann")
	makeWarrior(p, "Hale")

	w.Step("offered", func() {
		wantStep(t, p, 1, "Tamsin Hale")
		p.Model(func(m *model.Model) {
			if !hasLog(m, "quest", "Quest chain offered: First Steps") || m.Self.Crowns != 0 || m.Self.XP != 0 {
				t.Fatalf("on entering: %d crowns, %d xp, log %v", m.Self.Crowns, m.Self.XP, m.Log)
			}
		})
		p.Golden("first_steps_offered")
	})

	w.Step("greeter", func() {
		talkTo(t, w, p, "greeter")
		p.Press("esc")
		w.Advance(time.Second)
		wantStep(t, p, 2, "class trainer")
		p.Model(func(m *model.Model) {
			if !hasLog(m, "quest", "First Steps: step 1 done") {
				t.Fatalf("the log does not say step 1 is done: %v", m.Log)
			}
		})
		p.Golden("first_steps_trainer")
	})

	w.Step("trainer", func() {
		// Down the lane to the square's south side, west along Candlemakers' Lane.
		walkVia(t, w, p, [2]int64{60000, 22000}, [2]int64{60000, 17000}, [2]int64{16000, 17000})
		talkTo(t, w, p, "trainer-warrior")
		p.Press("esc")
		w.Advance(time.Second)
		wantStep(t, p, 3, "contract")
		p.Model(func(m *model.Model) {
			if m.Self.Crowns != 15 || !hasLog(m, "quest", "step 2 done: 15 crowns") {
				t.Fatalf("at the trainer: %d crowns; log %v", m.Self.Crowns, m.Log)
			}
		})
	})

	w.Step("board", func() {
		walkVia(t, w, p, [2]int64{60000, 17000}, [2]int64{60000, 45000}, [2]int64{57000, 45000}, [2]int64{57000, 51000})
		p.Press("g", "1", "enter")
		notice(t, p, "ok", false)
		p.Press("esc", "esc")
		w.Advance(time.Second)
		wantStep(t, p, 4, "east gate")
		p.Golden("first_steps_gate")
	})

	w.Step("gate", func() {
		walkVia(t, w, p, [2]int64{114000, 45000})
		p.Model(func(m *model.Model) {
			if m.Self.XP != 0 {
				t.Fatalf("%d xp before the gate is crossed", m.Self.XP)
			}
		})
		p.Press("e")
		p.Eventually("the Downs", inSpace("Thornwick Downs"))
		p.Eventually("the chain done", func(m *model.Model) bool { return m.Quest() == nil && m.Self.XP == 20 })
		p.Model(func(m *model.Model) {
			if !hasLog(m, "quest", "Quest complete: First Steps: 20 experience") {
				t.Fatalf("the log does not say the chain is finished: %v", m.Log)
			}
		})
		p.Golden("first_steps_done")
	})
}

// Dismissing the chain asks one question first, and the chain is never offered
// again: not on the next login either.
func TestDismissingFirstStepsIsForGood(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})
	p := w.NewCharacter("ann")
	makeWarrior(p, "Hale")
	wantStep(t, p, 1, "Tamsin Hale")

	p.Press("ctrl+k")
	p.Golden("quest_screen")
	p.Press("x")
	p.Golden("quest_dismiss_question")
	p.Press("n") // any other key withdraws the question
	p.Model(func(m *model.Model) {
		if m.Quest() == nil {
			t.Fatal("the chain was dismissed without an answer")
		}
	})
	p.Press("x", "y")
	notice(t, p, "dismissed for good", false)
	p.Eventually("no chain", func(m *model.Model) bool { return m.Quest() == nil })
	p.Model(func(m *model.Model) {
		if !hasLog(m, "quest", "You gave up the quest First Steps") {
			t.Fatalf("the log does not say it: %v", m.Log)
		}
	})
	// Talking to the greeter no longer starts anything, and /quest says so.
	talkTo(t, w, p, "greeter")
	p.Press("esc", "ctrl+k")
	p.Model(func(m *model.Model) {
		if m.Quest() != nil {
			t.Fatalf("the greeter offered the dismissed chain again: %v", m.Quest())
		}
	})
	p.Golden("quest_none")
	p.Press("esc")

	char := p.Character
	p.LogOut()
	p2 := w.Login("ann", char)
	p2.Model(func(m *model.Model) {
		if m.Quest() != nil || hasLog(m, "quest", "offered") {
			t.Fatalf("offered again at the next login: %v", m.Quest())
		}
	})
}

// /dismiss opens the same question.
func TestTheDismissCommandAsksFirst(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})
	p := w.NewCharacter("ann")
	makeWarrior(p, "Hale")
	command(p, "/dismiss")
	if s := p.Screen(80, 24); !strings.Contains(s, "for good? It is never offered again") {
		t.Fatalf("no question:\n%s", s)
	}
	p.Press("y")
	p.Eventually("no chain", func(m *model.Model) bool { return m.Quest() == nil })
}
