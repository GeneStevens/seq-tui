package scenario_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq-tui/sessionlog"
	"github.com/GeneStevens/seq/testserver"
)

// D64: every session is recorded under its id; Ctrl+L lists the player's own
// sessions, here and on the server; a local one is replayed in place, either is
// exported to a file, and the server's log is read as lines.

func screenHas(p *scenario.Player, want ...string) bool {
	s := p.Screen(140, 45)
	for _, w := range want {
		if !strings.Contains(s, w) {
			return false
		}
	}
	return true
}

func TestSessionsViewListsReplaysAndExports(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria")})
	p := w.Login("ann", "aria")
	p.Press("f") // commands for the logs to hold: the walk's keep-alives are timed by the clock, and a log shows when the server answered
	w.Advance(time.Second)
	p.Press("f")
	w.Advance(time.Second)

	// A second session of the same player: the first is over and has its end.
	p.Fresh()
	p.Press("f")
	w.Advance(time.Second)

	p.Press("ctrl+l")
	if !screenHas(p, "On the server", "tui-ann-1", "Testgrounds") {
		t.Fatalf("the server's list is not on the screen:\n%s", p.Screen(140, 45))
	}
	p.Golden("sessions_list")
	if !screenHas(p, "On this machine", "tui-ann-2", "tui-ann-1", "aria", "Logs live in") {
		t.Fatalf("the list lacks a session:\n%s", p.Screen(140, 45))
	}

	// The list's first row is the session being played, the second the first one:
	// Enter replays its recording in place, up to its last message.
	p.Press("down", "enter")
	if !screenHas(p, "REPLAY", "tui-ann-1: message") {
		t.Fatalf("no replay on the screen:\n%s", p.Screen(140, 45))
	}
	p.Golden("sessions_replay")
	p.Press("home")
	if !screenHas(p, "message 0 of") {
		t.Fatalf("Home did not go to the start:\n%s", p.Screen(140, 45))
	}
	p.Press("esc")
	if !screenHas(p, "On this machine") {
		t.Fatalf("Esc did not return to the list:\n%s", p.Screen(140, 45))
	}

	// Export a local recording.
	p.Press("e")
	local := filepath.Join(sessionlog.ExportDir(w.LogRoot()), "*-tui-ann-1.jsonl")
	if got, _ := filepath.Glob(local); len(got) != 1 {
		t.Fatalf("export wrote %v; want one file matching %s", got, local)
	}
	if n, _ := p.Notice(); !strings.Contains(n, "exported to") {
		t.Fatalf("the status line says %q", n)
	}

	// The server's own log of a session, as readable lines; then exported.
	p.Press("down", "down", "enter")
	if !screenHas(p, "the server's log", "connect: aria in testgrounds", "intent") {
		t.Fatalf("no server log on the screen:\n%s", p.Screen(140, 45))
	}
	p.Golden("sessions_server_log")
	p.Press("e")
	b, err := os.ReadFile(filepath.Join(sessionlog.ExportDir(w.LogRoot()), "server-tui-ann-1.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "connect: aria in testgrounds") || !strings.Contains(string(b), "result ") {
		t.Fatalf("the exported server log:\n%s", b)
	}
	p.Press("esc", "esc")
	if screenHas(p, "On this machine") || !screenHas(p, "Testgrounds") {
		t.Fatalf("Esc did not return to play:\n%s", p.Screen(140, 45))
	}
	p.Press("right")
	if sends := p.WalkSends(); len(sends) == 0 || sends[0].Stop {
		t.Fatalf("the arrow key did not walk the character: %v", sends)
	}
}

// The sessions view opens from the character creation screen, where a player
// who has not chosen a character yet can still read their own logs.
func TestSessionsViewOpensFromTheCreationScreen(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria")})
	p := w.NewCharacter("ann")
	p.Press("ctrl+l")
	if !screenHas(p, "On this machine", "tui-ann-1") {
		t.Fatalf("the list is not on the screen:\n%s", p.Screen(140, 45))
	}
	p.Press("esc")
	if !screenHas(p, "New character") {
		t.Fatalf("Esc did not return to the creation screen:\n%s", p.Screen(140, 45))
	}
}

// No recording or log holds the session's credential, or any session token.
func TestNoRecordingHoldsACredential(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria")})
	p := w.Login("ann", "aria")
	p.Press("right")
	w.Advance(time.Second)
	p.Press("ctrl+l")
	if !screenHas(p, "On the server") {
		t.Fatalf("the list is not on the screen:\n%s", p.Screen(140, 45))
	}
	p.Press("e")
	p.Press("down", "enter")
	p.Press("e", "esc", "esc")
	p.Fresh() // closes the first session: its session.json is complete

	token := w.Srv.SessionToken("ann")
	n := 0
	err := filepath.WalkDir(filepath.Dir(w.LogRoot()), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		n++
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(b), token) || strings.Contains(strings.ToLower(string(b)), "seq_st_") {
			t.Errorf("%s holds a credential", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n < 4 {
		t.Fatalf("only %d files were written; the scan proves nothing", n)
	}
}
