package scenario_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/testserver"
)

// determinismRuns is how many times the determinism scenario is played (SEQ-059's STOP,
// TUI-021): the outcome of each must be the one of the first.
const determinismRuns = 50

// twoAccountsPlay plays two accounts' TUIs against a scuttler camp and returns
// everything the outcome is: both players' event logs, and the XP and HP of
// both characters on the server. Each sends its target and attack without
// waiting for the answer, ann's and bob's interleaved, as two clients do; which
// tick they land in must not depend on how fast the sockets ran.
func twoAccountsPlay(t *testing.T) string {
	t.Helper()
	w := scenario.New(t, testserver.Options{Seed: 7, Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{
		{Name: "ann", Characters: []string{"ann1"}}, {Name: "bob", Characters: []string{"bob1"}}}})
	w.Place("ann", "ann1", 23000, 50000)
	w.Place("bob", "bob1", 23000, 50800)
	ann := w.Login("ann", "ann1")
	bob := w.Login("bob", "bob1")
	var mob string
	ann.Model(func(m *model.Model) {
		for _, id := range m.EntityIDs() {
			if strings.HasPrefix(id, "mob-near") {
				mob = id
			}
		}
	})
	if mob == "" {
		t.Fatal("no mob of the near camp in view")
	}
	ctx := context.Background()
	for _, step := range []struct {
		p      *scenario.Player
		attack bool
	}{{ann, false}, {bob, false}, {ann, true}, {bob, true}} {
		var err error
		if step.attack {
			_, err = step.p.Sess.Attack(ctx, true)
		} else {
			_, err = step.p.Sess.Target(ctx, mob)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	w.Advance(12 * time.Second)
	w.Settle()
	if err := w.Srv.Replay(testserver.Testgrounds); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, p := range []*scenario.Player{ann, bob} {
		p.Model(func(m *model.Model) {
			for _, e := range m.Log {
				fmt.Fprintf(&b, "%s %d %d %s %s\n", p.Character, e.Seq, e.GameTimeMs, e.Kind, e.Text)
			}
		})
	}
	for _, id := range []string{"ann1", "bob1"} {
		c, _ := w.Srv.Character(id)
		fmt.Fprintf(&b, "%s xp=%d hp=%d\n", id, c.Entity.XP, c.Entity.HP)
	}
	return b.String()
}

// Two accounts fighting one mob end the same way every time: the same events in
// the same ticks for both, the same XP and HP, and a journal that rebuilds the
// space. (Run once with the test server's clock on its pump, the logs differ
// from run to run.)
func TestTwoAccountsPlayTheSameEveryRun(t *testing.T) {
	var first string
	for i := range determinismRuns {
		t.Run(fmt.Sprint(i), func(t *testing.T) { // each run is a scenario, held to the limit
			got := twoAccountsPlay(t)
			switch {
			case i == 0:
				first = got
				if !strings.Contains(first, "xp=") || !strings.Contains(first, "combat_") {
					t.Fatalf("nothing happened:\n%s", first)
				}
			case got != first:
				t.Fatalf("run %d differs from run 0 at %s", i, firstDifference(first, got))
			}
		})
	}
}

func firstDifference(a, b string) string {
	la, lb := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; i < len(la) || i < len(lb); i++ {
		var x, y string
		if i < len(la) {
			x = la[i]
		}
		if i < len(lb) {
			y = lb[i]
		}
		if x != y {
			return fmt.Sprintf("line %d: %q, then %q", i+1, x, y)
		}
	}
	return "no line"
}
