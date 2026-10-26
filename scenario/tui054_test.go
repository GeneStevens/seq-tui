package scenario_test

import (
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
	"github.com/GeneStevens/seq/testserver"
)

// D109: a plain click never moves the character; Option-click does. One smoke
// test of the wiring against the real server: the rows of what each click means
// are in app's pickAction table.
func TestAPlainClickNeverWalksAndOptionClickDoes(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria")})
	w.Place("ann", "aria", 84000, 50000) // 16 m from the trophy scuttler
	p := w.Login("ann", "aria")
	p.Resize(140, 45)

	var from *pb.Pos
	p.Model(func(m *model.Model) { from = m.Self.Pos })
	x, y := at(t, p, &pb.Pos{X: from.GetX() - 10000, Y: from.GetY()})

	// A plain click on the ground says how to walk, once, and sends nothing.
	p.Click(x, y)
	notice(t, p, "Option-click to walk here", false)
	for range 5 {
		w.Advance(time.Second)
	}
	p.Model(func(m *model.Model) {
		if m.Self.Pos.GetX() != from.GetX() || m.Self.HasDestination {
			t.Errorf("a plain click moved the character: %v to %v", from, m.Self.Pos)
		}
	})
	p.Click(x, y)
	if got, _ := p.Notice(); strings.Contains(got, "Option-click to walk here") {
		t.Errorf("the hint is shown again: %q", got)
	}

	// A click one cell off the mob selects it; clicking twice more attacks once.
	mob := mobIn(t, p)
	mx, my := at(t, p, mob.GetPos())
	p.Click(mx+1, my)
	p.Model(func(m *model.Model) {
		if m.Self.Target != mob.GetId() {
			t.Errorf("a click one cell off the mob did not select it: target %q", m.Self.Target)
		}
	})
	p.Click(mx, my) // arms the attack
	p.Click(mx, my) // already on: nothing is sent
	p.Click(mx, my)
	s := p.Screen(140, 45)
	if !strings.Contains(lastRow(s), "Too far: 16 m. Option-click the target to close in") {
		t.Errorf("the status line does not say the target is out of reach: %q", lastRow(s))
	}
	if !strings.Contains(hintRow(s), "Too far: 16 m") || !strings.Contains(s, "Attacking · F: stop") {
		t.Errorf("the hint line or the card does not show the attack state:\n%s", s)
	}
	n := 0
	p.Model(func(m *model.Model) {
		for _, e := range m.Log {
			if strings.Contains(e.Text, "start attacking") {
				n++
			}
		}
	})
	if n > 1 {
		t.Errorf("%d lines say the attack started", n)
	}
	p.Press("f", "esc")

	// Option-click on the mob walks up to it and attacks.
	mob = mobIn(t, p)
	mx, my = at(t, p, mob.GetPos())
	p.AltClick(mx, my)
	for range 25 {
		w.Advance(time.Second)
	}
	p.Model(func(m *model.Model) {
		if !m.Self.AutoAttack || m.Self.Target == "" {
			t.Errorf("Option-click on the mob did not close in and attack: target %q, attack %v", m.Self.Target, m.Self.AutoAttack)
		} else if _, far := screen.AttackTooFar(m); far {
			t.Errorf("attacking from out of reach")
		}
	})
}
