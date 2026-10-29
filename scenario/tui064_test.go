package scenario_test

import (
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq/testserver"

	"github.com/GeneStevens/seq-tui/scenario"
)

// /bug TEXT files a bug with the game masters and the status line carries the
// server's acknowledgement with the bug's id; a bare /bug says how to use it.
func TestBugCommandFilesAndShowsTheId(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "ann1")})
	w.Srv.GM() // starts the report store: without it the server refuses a bug
	w.Place("ann", "ann1", 11500, 57500)
	ann := w.Login("ann", "ann1")

	command(ann, "bug")
	notice(t, ann, "usage: /bug", true)

	command(ann, "bug the ferry sign has a typo")
	w.Settle()
	notice(t, ann, "bug 1", false)
	ann.Model(func(m *model.Model) {
		if m.Social.BugAck != 1 {
			t.Errorf("BugAck = %d, want 1", m.Social.BugAck)
		}
	})
}
