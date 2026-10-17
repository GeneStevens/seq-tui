package scenario_test

import (
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/testserver"
)

// matesHP is the health the player's Party panel shows for a member.
func matesHP(p *scenario.Player, name string) (hp, max int64) {
	p.Model(func(m *model.Model) {
		for _, mt := range m.Mates() {
			if mt.Name == name {
				hp, max = mt.HP, mt.MaxHP
			}
		}
	})
	return
}

func selfHP(p *scenario.Player) (hp int64) {
	p.Model(func(m *model.Model) { hp = m.Self.HP })
	return
}

// Real-time play (D60): a party member's health reaches the Party panel in the
// tick it changes, as a changed-field message, not at the space's 6 s rules tick.
// A scuttler comes at the two as they stand by its camp; whoever it strikes, the
// other's panel shows the new health in the same tick or the next, and the
// clock never moves as far as the pulse.
func TestAMatesHealthShowsInTheTickItChanges(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{
		{Name: "ann", Characters: []string{"ann1"}}, {Name: "bob", Characters: []string{"bob1"}}}})
	w.Place("ann", "ann1", 20000, 50000)
	w.Place("bob", "bob1", 20000, 50500)
	ann := w.Login("ann", "ann1")
	bob := w.Login("bob", "bob1")
	command(ann, "/invite bob1")
	w.Settle()
	bob.Press("y")
	w.Settle()
	for _, p := range []*scenario.Player{ann, bob} {
		p.Eventually("the party of two", func(m *model.Model) bool { return len(m.PartyMembers()) == 2 })
	}
	w.Advance(100 * time.Millisecond) // the first sighting: the whole party state
	bob.Eventually("ann's vitals", func(m *model.Model) bool { return mate(m, "ann1") })
	ann.Eventually("bob's vitals", func(m *model.Model) bool { return mate(m, "bob1") })

	// Step in ticks until the scuttler lands a blow, then one more tick. The mob
	// strikes every 3 s, so it is well inside the 6 s rules tick.
	var hurt *scenario.Player
	var other *scenario.Player
	start := w.Srv.NowMs()
	for hurt == nil {
		if w.Srv.NowMs()-start > 5000 {
			t.Fatal("the scuttler struck no one in 5 s")
		}
		w.Advance(100 * time.Millisecond)
		switch {
		case selfHP(ann) < 100:
			hurt, other = ann, bob
		case selfHP(bob) < 100:
			hurt, other = bob, ann
		}
	}
	w.Advance(100 * time.Millisecond)
	name := hurt.Character
	hp, max := matesHP(other, name)
	if want := selfHP(hurt); hp != want || max != 100 || hp >= 100 {
		t.Fatalf("%s's panel shows %s at %d/%d, %s has %d", other.Character, name, hp, max, name, want)
	}
	if took := w.Srv.NowMs() - start; took >= 6000 {
		t.Fatalf("the clock moved %d ms: the vitals waited for the pulse", took)
	}
	if s := other.Screen(140, 45); !strings.Contains(s, name) {
		t.Fatalf("the panel does not list %s:\n%s", name, s)
	}
	other.Golden("party_vitals_hurt")
}

// Turn-based play (D60): a mate's health is told once per round, after the round
// result. While the round waits for plans nothing about the mates changes, and
// the pulse in between moves nothing; when the round has run, the panel shows
// each member's health as the round left it.
func TestAMatesHealthChangesAtTheRoundsEnd(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{
		{Name: "ann", Characters: []string{"ann1"}}, {Name: "bob", Characters: []string{"bob1"}}}})
	w.Place("ann", "ann1", gateX, gateY)
	w.Place("bob", "bob1", gateX+1000, gateY)
	ann := w.Login("ann", "ann1")
	bob := w.Login("bob", "bob1")
	command(ann, "/invite bob1")
	w.Settle()
	bob.Press("y")
	w.Settle()
	for _, p := range []*scenario.Player{ann, bob} {
		p.Eventually("the party of two", func(m *model.Model) bool { return len(m.PartyMembers()) == 2 })
	}
	w.Advance(100 * time.Millisecond)
	openPreset(t, bob, "duo-turns", true)
	w.Advance(100 * time.Millisecond)
	bob.Press("e")
	ann.Press("e")
	for _, p := range []*scenario.Player{ann, bob} {
		p.Eventually("round 1", openRound)
		p.Eventually("both in the round", func(m *model.Model) bool { return m.Round != nil && m.Round.Participants == 2 })
	}
	players := map[string]*scenario.Player{"ann1": ann, "bob1": bob}
	otherOf := map[string]string{"ann1": "bob1", "bob1": "ann1"}
	panel := func() map[string]int64 {
		hp := map[string]int64{}
		for _, p := range []*scenario.Player{ann, bob} {
			hp[otherOf[p.Character]], _ = matesHP(p, otherOf[p.Character])
		}
		return hp
	}
	changed := false
	for round := 1; round <= 8 && !changed; round++ {
		for _, p := range []*scenario.Player{ann, bob} {
			p.Eventually("the round open", func(m *model.Model) bool { return openRound(m) && m.Round.Resolved == uint64(round-1) })
		}
		w.Advance(100 * time.Millisecond)
		before := panel()
		for name, hp := range before {
			if hp == 0 {
				t.Fatalf("round %d: %s is at nothing on the panel", round, name)
			}
		}

		// The round waits for the plans: a second of ticks changes nothing.
		for range 10 {
			w.Advance(100 * time.Millisecond)
		}
		if got := panel(); got["ann1"] != before["ann1"] || got["bob1"] != before["bob1"] {
			t.Fatalf("round %d: the panels moved while the round waits: %v, was %v", round, got, before)
		}

		// Both send; the round runs; the panels show the round's values.
		bob.Press("down", "down", "down", "tab", "f", "enter")
		ann.Press("down", "down", "down", "tab", "f", "enter")
		w.Advance(100 * time.Millisecond)
		for _, p := range []*scenario.Player{ann, bob} {
			p.Eventually("the round resolved", func(m *model.Model) bool { return m.Round != nil && m.Round.Resolved >= uint64(round) })
		}
		w.Settle()
		after := panel()
		for name, hp := range after {
			if want := selfHP(players[name]); hp != want {
				t.Fatalf("round %d: the panel shows %s at %d, who has %d", round, name, hp, want)
			}
			if hp != before[name] {
				changed = true
			}
		}
	}
	if !changed {
		t.Fatal("no round changed anybody's health")
	}
	ann.Golden("party_round_vitals")
}
