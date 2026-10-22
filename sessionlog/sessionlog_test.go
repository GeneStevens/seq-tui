package sessionlog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/sessionlog"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func open(t *testing.T, root, session string, o sessionlog.Options) *sessionlog.Log {
	t.Helper()
	o.Meta.Session, o.Meta.Server, o.Meta.Client = session, "wss://user:pw@example.test:8443/seq/v1?token=abc", "seq-tui"
	l, err := sessionlog.Open(root, o)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func msg(text string) *pb.ServerMessage {
	return &pb.ServerMessage{Body: &pb.ServerMessage_Error{Error: &pb.Error{Message: text}}}
}

func TestASessionIsWrittenUnderItsIDWithNoUserInfo(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sessions")
	start := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	l := open(t, root, "tui-abc", sessionlog.Options{Credential: "seq_st_secret", Now: func() time.Time { return start }})
	l.SetCharacter("aria")
	l.SetZone("testgrounds")
	if err := l.Rec.Write(msg("hello")); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	es := sessionlog.List(root)
	if len(es) != 1 || filepath.Base(es[0].Dir) != "20261017-120000-tui-abc" {
		t.Fatalf("sessions: %+v", es)
	}
	m := es[0].Meta
	if m.Session != "tui-abc" || m.Character != "aria" || m.Zone != "testgrounds" || m.Server != "wss://example.test:8443/seq/v1" || m.End.IsZero() {
		t.Fatalf("session.json: %+v", m)
	}
	b, err := os.ReadFile(es[0].RecordPath())
	if err != nil || !strings.HasPrefix(string(b), "# seq session tui-abc\n") || !strings.Contains(string(b), "hello") {
		t.Fatalf("record.jsonl: %q, %v", b, err)
	}
}

func TestNoRecordWritesNothing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sessions")
	l, err := sessionlog.Start(root, true, sessionlog.Options{})
	if err != nil || l != nil {
		t.Fatalf("Start with off: %v, %v", l, err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("-no-record made %s", root)
	}
}

func TestTheCredentialIsScrubbedFromWhatIsWritten(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sessions")
	l := open(t, root, "s", sessionlog.Options{Credential: "seq_st_secret"})
	_ = l.Rec.Write(msg("the server said seq_st_secret by mistake"))
	_ = l.Close()
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if b, _ := os.ReadFile(p); strings.Contains(string(b), "seq_st_secret") {
				t.Errorf("%s holds the credential", p)
			}
		}
		return nil
	})
	es := sessionlog.List(root)
	if b, _ := os.ReadFile(es[0].RecordPath()); !strings.Contains(string(b), "[scrubbed]") {
		t.Fatalf("record.jsonl: %s", b)
	}
}

func TestASessionStopsAtItsSizeCap(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sessions")
	l := open(t, root, "s", sessionlog.Options{SessionBytes: 400})
	if l.Stopped() {
		t.Fatal("stopped before anything was written")
	}
	for range 50 {
		_ = l.Rec.Write(msg("a line that takes some room"))
	}
	if !l.Stopped() {
		t.Error("the log does not say the cap stopped it") // the UI tells the player from this
	}
	_ = l.Close()
	b, _ := os.ReadFile(sessionlog.List(root)[0].RecordPath())
	if len(b) > 600 || !strings.Contains(string(b), "# stopped:") {
		t.Fatalf("%d bytes: %.200q", len(b), b)
	}
}

func TestTheDirectoryCapPrunesTheOldestSessions(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sessions")
	at := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	for i, id := range []string{"old", "mid", "new"} {
		l := open(t, root, id, sessionlog.Options{Now: func() time.Time { return at.Add(time.Duration(i) * time.Hour) }})
		for range 20 {
			_ = l.Rec.Write(msg("padding padding padding padding"))
		}
		_ = l.Close()
	}
	// Something that is not a session is never removed.
	keep := filepath.Join(root, "notes", "mine.txt")
	_ = os.MkdirAll(filepath.Dir(keep), 0o755)
	_ = os.WriteFile(keep, []byte("x"), 0o644)

	es := sessionlog.List(root)
	if len(es) != 3 || es[0].Meta.Session != "new" {
		t.Fatalf("sessions: %+v", es)
	}
	sessionlog.Prune(root, 2*(es[0].Size+450))
	var left []string
	for _, e := range sessionlog.List(root) {
		left = append(left, e.Meta.Session)
	}
	if strings.Join(left, ",") != "new,mid" {
		t.Fatalf("left %v; the oldest should go first", left)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatal("a directory with no session.json was removed")
	}
}

func TestFindTakesAFileADirectoryOrAnID(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sessions")
	l := open(t, root, "tui-x", sessionlog.Options{})
	_ = l.Close()
	want := sessionlog.List(root)[0].RecordPath()
	for _, arg := range []string{"tui-x", filepath.Dir(want), want} {
		if got, err := sessionlog.Find(root, arg); err != nil || got != want {
			t.Errorf("Find(%q) = %q, %v; want %q", arg, got, err, want)
		}
	}
	if _, err := sessionlog.Find(root, "nope"); err == nil {
		t.Error("Find found a session that is not there")
	}
}

func TestTheRecordingIsInTheFileOnceFlushed(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sessions")
	l := open(t, root, "s", sessionlog.Options{})
	_ = l.Rec.Write(msg("hello"))
	if err := l.Rec.Flush(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(sessionlog.List(root)[0].RecordPath()); !strings.Contains(string(b), "hello") {
		t.Errorf("flushed, and the file lacks the line: %q", b)
	}
	_ = l.Close()
}
