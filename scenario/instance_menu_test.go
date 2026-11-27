package scenario_test

import (
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/testserver"
)

// The instance menu lets a player leave an instance or close it, and says what
// opening another would do to the one they hold. A player is in one instance at
// a time: the server closes the old one, or takes the player out of it.

func soloAnn(t *testing.T) (*scenario.World, *scenario.Player) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria")})
	w.Fixture("solo_trial.jsonl")
	w.Place("ann", "aria", gateX, gateY)
	return w, w.Login("ann", "aria")
}

func hasText(p *scenario.Player, want string) bool { return strings.Contains(p.Screen(140, 45), want) }

func TestLeaveFromTheMenuGivesUpTheInstance(t *testing.T) {
	w, p := soloAnn(t)
	p.Press("i", "l")
	notice(t, p, "you hold no place in an instance", false) // nothing to leave: nothing is sent
	p.Press("esc")
	openPreset(t, p, "solo-trial", true)
	w.Advance(time.Second)
	p.Press("e")
	p.Eventually("the instance", inside)
	p.Press("i")
	if !hasText(p, "L leaves your instance; C closes it") {
		t.Fatalf("the menu does not offer Leave and Close to the opener:\n%s", p.Screen(100, 30))
	}
	p.Press("l")
	notice(t, p, "you gave up your place in the instance", false)
	p.Eventually("handed back, the empty instance ended", back)
	p.Model(func(m *model.Model) {
		if m.Ended.Reason != "abandoned" || m.InInstance() {
			t.Fatalf("ended %+v, inside %v", m.Ended, m.InInstance())
		}
	})
}

func TestCloseFromTheMenuRunsTheCountdownAndHandsBack(t *testing.T) {
	w, p := soloAnn(t)
	openPreset(t, p, "solo-trial", true)
	w.Advance(time.Second)
	p.Press("e")
	p.Eventually("the instance", inside)
	p.Press("i", "c")
	notice(t, p, "closing the instance", false)
	p.Eventually("the countdown", func(m *model.Model) bool { return m.Closing != nil && m.Closing.Seconds == 60 })
	w.Advance(20 * time.Second)
	p.Sync()
	p.Model(func(m *model.Model) {
		if m.Closing == nil || !m.InInstance() {
			t.Fatalf("closing %+v, inside %v: it ends after the countdown", m.Closing, m.InInstance())
		}
	})
	if scr := p.Screen(140, 45); !strings.Contains(scr, "Closes in 40 s") {
		t.Fatalf("the Instance panel does not follow the countdown:\n%s", scr)
	}
	p.Golden("instance_closing")
	w.Advance(41 * time.Second)
	p.Eventually("the hand-back", back)
	p.Model(func(m *model.Model) {
		if m.Ended.Reason != "closed" || m.Closing != nil {
			t.Fatalf("ended %+v, closing %+v", m.Ended, m.Closing)
		}
	})
	if hasText(p, "Closes in") {
		t.Fatal("the countdown stays on the screen after the instance ended")
	}
}

func TestTheReplaceLineAsksBeforeOpeningAnotherInstance(t *testing.T) {
	w, p := soloAnn(t)
	openPreset(t, p, "solo-trial", true) // none held: no line, it opens
	w.Advance(time.Second)
	var first string
	p.Model(func(m *model.Model) {
		if m.Opened == nil || m.Replacing == nil || m.ReplacingN() != 1 {
			t.Fatalf("opened %+v after %d dry runs", m.Opened, m.ReplacingN())
		}
		if m.Replacing.Instance != "" {
			t.Fatalf("the dry run named %q with none held", m.Replacing.Instance)
		}
		first = m.Opened.Space
	})
	notice(t, p, "instance requested", false)

	// Another open: the line names what happens to the first, and Esc keeps it.
	p.Press("i", "1", "enter")
	p.Eventually("the line", func(m *model.Model) bool { return m.Replacing != nil && m.Replacing.Instance == first })
	p.Press() // let the program read the answer
	w.Advance(100 * time.Millisecond)
	scr := p.Screen(140, 45)
	if !strings.Contains(scr, "Opening this closes Solo Trial") || !strings.Contains(scr, "Enter to go on") {
		t.Fatalf("no replace line:\n%s", scr)
	}
	p.Golden("instance_replace_line")
	p.Press("esc")
	p.Model(func(m *model.Model) {
		if m.Opened.Space != first {
			t.Fatalf("Esc opened another instance: %+v", m.Opened)
		}
	})
	if hasText(p, "Opening this closes") {
		t.Fatal("the line stays after Esc")
	}

	// Asked again and confirmed: the first is closed and the second opens.
	p.Press("enter")
	p.Eventually("the line again", func(m *model.Model) bool { return m.ReplacingN() == 3 })
	w.Advance(100 * time.Millisecond)
	p.Press("enter")
	w.Advance(time.Second)
	notice(t, p, "instance requested", false)
	p.Eventually("the second instance", func(m *model.Model) bool { return m.Opened != nil && m.Opened.Space != first })
	w.Advance(70 * time.Second)
	p.Model(func(m *model.Model) {
		if m.Ended == nil || m.Ended.Space != first || m.Ended.Reason != "closed" {
			t.Fatalf("the first instance: ended %+v, want closed", m.Ended)
		}
	})
}

func TestAMemberIsOfferedLeaveButNotClose(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{
		{Name: "ann", Characters: []string{"aria"}}, {Name: "bob", Characters: []string{"bram"}}}})
	w.Place("ann", "aria", gateX, gateY)
	w.Place("bob", "bram", gateX+1000, gateY)
	ann := w.Login("ann", "aria")
	bob := w.Login("bob", "bram")
	command(ann, "/invite bram")
	w.Settle()
	bob.Press("y")
	for _, p := range []*scenario.Player{ann, bob} {
		p.Eventually("the party of two", func(m *model.Model) bool { return len(m.PartyMembers()) == 2 })
	}
	w.Advance(time.Second)
	openPreset(t, ann, "skirmish", true)
	bob.Eventually("the group's instance", func(m *model.Model) bool { return m.Opened != nil })
	w.Advance(time.Second)

	ann.Press("i")
	if !hasText(ann, "C closes it for everyone") {
		t.Fatalf("the leader is not offered Close:\n%s", ann.Screen(100, 30))
	}
	ann.Press("esc")
	bob.Press("i")
	if scr := bob.Screen(140, 45); !strings.Contains(scr, "L leaves your instance") || strings.Contains(scr, "C closes") {
		t.Fatalf("a member's menu:\n%s", scr)
	}
	bob.Press("c")
	notice(t, bob, "only your group's leader can close the instance", false)
	bob.Model(func(m *model.Model) {
		if m.Closing != nil {
			t.Fatal("a member's key closed the instance")
		}
	})
}
