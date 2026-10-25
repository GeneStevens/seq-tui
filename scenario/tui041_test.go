package scenario_test

import (
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
	"github.com/GeneStevens/seq/testserver"
)

// TUI-041 (D100): a server announcement arriving in the middle of play shows at
// the top of the screen, stays in Events, and goes at the next key; a second one
// goes by itself after a few seconds. The test server cannot send one, so the
// session is told it as the server would.
func TestAnAnnouncementArrivesMidPlay(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria")})
	p := w.Login("ann", "aria")
	w.Advance(time.Second)
	say := func(text string) {
		p.Sess.Hear(&pb.ServerMessage{Body: &pb.ServerMessage_Announcement{Announcement: &pb.Announcement{Text: text}}})
		w.Advance(300 * time.Millisecond)
	}
	say("Restart in 5 minutes.")
	s := p.Screen(140, 45)
	if !strings.Contains(s, "SERVER Restart in 5 minutes.") || !strings.Contains(s, "Server: Restart in 5 minutes.") {
		t.Fatalf("the announcement is not at the top and in Events:\n%s", s)
	}
	p.Press("tab")
	s = p.Screen(140, 45)
	if strings.Contains(s, "SERVER Restart") || !strings.Contains(s, "Server: Restart in 5 minutes.") {
		t.Errorf("a key should clear the banner and keep the log line:\n%s", s)
	}

	say("Back to normal.")
	if !strings.Contains(p.Screen(140, 45), "SERVER Back to normal.") {
		t.Fatal("the second announcement is not shown")
	}
	w.Advance(12 * time.Second)
	p.Model(func(m *model.Model) {
		if m.Banner != nil {
			t.Errorf("the banner stays up: %+v", m.Banner)
		}
	})
}
