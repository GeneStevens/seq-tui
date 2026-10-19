package scenario_test

import (
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/testserver"
)

// Combat reflexes (D66, D75): an attack with nothing targeted and nothing
// attacking is refused; once something attacks, the server offers it as the
// target (D75), so one key fights back. A fight that stops a walk is in the log.
func TestAnUntargetedAttackGoesToWhatAttacksYou(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria")})
	p := w.Login("ann", "aria")

	p.Press("f")
	notice(t, p, "You have no target", true)

	p.Press("right")
	w.Advance(time.Second)
	p.Press("right")
	w.Advance(2 * time.Second) // the scuttler wakes, attacks, and closes in
	p.Model(func(m *model.Model) {
		if m.Self.Target == "" || m.Label(m.Self.Target) != "Plain Scuttler" || !hasLog(m, "walk_stopped", "you are fighting Plain Scuttler") {
			t.Fatalf("target %q once attacked; log %v", m.Self.Target, m.Log)
		}
	})
	p.Press("f")
	notice(t, p, "attack ok", false)
	p.Model(func(m *model.Model) {
		if m.Self.Target == "" || !m.Self.AutoAttack {
			t.Fatalf("target %q, auto-attack %v", m.Self.Target, m.Self.AutoAttack)
		}
	})
}
