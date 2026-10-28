// Package scripts_test holds the tests of the play scripts (TUI-036): that
// `make play` configures seqd to serve Thornwick and start new characters there,
// that Testgrounds is behind its flag, and (with docker, SEQ_PLAY_DOCKER_TEST=1)
// that the world persists across two runs and play-reset clears it.
package scripts_test

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
)

// dryRun runs play.sh with SEQ_PLAY_DRY=1 and the given environment, and returns
// its report as a map of the line's first word to the rest.
func dryRun(t *testing.T, env ...string) map[string]string {
	t.Helper()
	cmd := exec.Command("sh", "scripts/play.sh")
	cmd.Dir = ".."
	cmd.Env = append(os.Environ(), append([]string{"SEQ_PLAY_DRY=1", "SEQ_TESTGROUNDS="}, env...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("play.sh (dry run): %v\n%s", err, out)
	}
	rep := map[string]string{}
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		k, v, _ := strings.Cut(l, " ")
		rep[strings.TrimSuffix(k, ":")] = v
	}
	return rep
}

var buildOnce sync.Once
var seqdPath string

// seqd builds the pinned server once for the tests.
func seqd(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "seqd-play-test")
		if err != nil {
			t.Fatal(err)
		}
		seqdPath = filepath.Join(dir, "seqd")
		if out, err := exec.Command("go", "build", "-o", seqdPath, "github.com/GeneStevens/seq/cmd/seqd").CombinedOutput(); err != nil {
			seqdPath = ""
			t.Fatalf("building seqd: %v\n%s", err, out)
		}
	})
	if seqdPath == "" {
		t.Skip("seqd did not build")
	}
	return seqdPath
}

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

// server is a running seqd.
type server struct {
	cmd   *exec.Cmd
	addr  string
	token string
	log   *lockedBuf
}

type lockedBuf struct {
	mu sync.Mutex
	b  strings.Builder
}

func (l *lockedBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

// waitFor polls until the log holds want, for up to 10 s: seqd's stdout and stderr
// are copied by two goroutines, so a line on one can land before an earlier line
// on the other.
func (l *lockedBuf) waitFor(want string) bool {
	for end := time.Now().Add(10 * time.Second); time.Now().Before(end); time.Sleep(10 * time.Millisecond) {
		if strings.Contains(l.String(), want) {
			return true
		}
	}
	return strings.Contains(l.String(), want)
}

func (l *lockedBuf) String() string { l.mu.Lock(); defer l.mu.Unlock(); return l.b.String() }

var tokenLine = regexp.MustCompile(`development token for \w+: (\S+)`)

// start runs seqd with the flags the dry run printed (the listen address
// replaced), on the database url if it is not empty, and waits for the token.
func start(t *testing.T, flags []string, dbURL string) *server {
	t.Helper()
	s := &server{addr: freeAddr(t), log: &lockedBuf{}}
	args := append([]string(nil), flags...)
	for i, a := range args {
		if a == "-listen" {
			args[i+1] = s.addr
		}
	}
	args = append(args, "-log-dir", t.TempDir())
	s.cmd = exec.Command(seqd(t), args...)
	s.cmd.Env = append(os.Environ(), "SEQ_DATABASE_URL="+dbURL)
	if dbURL == "" {
		s.cmd.Env = append(s.cmd.Env, "SEQ_DATABASE_URL=")
	}
	r, w := io.Pipe()
	s.cmd.Stdout, s.cmd.Stderr = io.MultiWriter(w, s.log), io.MultiWriter(w, s.log)
	if err := s.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.stop)
	got := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			if m := tokenLine.FindStringSubmatch(sc.Text()); m != nil {
				got <- m[1]
				break
			}
		}
		for sc.Scan() { // keep draining
		}
	}()
	select {
	case s.token = <-got:
	case <-time.After(60 * time.Second):
		t.Fatalf("seqd gave no token:\n%s", s.log)
	}
	return s
}

func (s *server) stop() {
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_, _ = s.cmd.Process.Wait()
	}
}

func (s *server) session(t *testing.T) *client.Session {
	t.Helper()
	sess := client.NewSession(client.Config{URL: "wss://" + s.addr + "/seq/v1", TLS: &tls.Config{InsecureSkipVerify: true}, Credential: s.token, Session: client.NewSessionID()})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := sess.Connect(ctx); err != nil {
		t.Fatalf("connect: %v\n%s", err, s.log)
	}
	t.Cleanup(sess.Close)
	return sess
}

// makeAndPlay makes a character of the first class on the menu and chooses it,
// and returns its id and the space it started in.
func makeAndPlay(t *testing.T, sess *client.Session, name string) (id, space string) {
	t.Helper()
	var class string
	sess.View(func(m *model.Model) {
		if c := m.Classes(); len(c) > 0 {
			class = c[0].GetId()
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	id, err := sess.CreateCharacter(ctx, name, class)
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	if err := sess.Select(ctx, id); err != nil {
		t.Fatalf("select %s: %v", id, err)
	}
	sess.View(func(m *model.Model) { space = m.Space })
	return id, space
}

func TestPlayServesThornwickAndNewCharactersStartThere(t *testing.T) {
	rep := dryRun(t)
	if rep["mode"] != "world" || !strings.HasPrefix(rep["docker"], "project: seq-tui-world (volume ") {
		t.Fatalf("make play is not the kept world: %v", rep)
	}
	flags := strings.Fields(rep["seqd"])
	if strings.Contains(rep["seqd"], "-start-space") {
		t.Errorf("make play overrides the packs' start zone: %s", rep["seqd"])
	}
	s := start(t, flags, "")
	if !s.log.waitFor(`"start_space":"thornwick"`) {
		t.Errorf("the server does not start new characters in Thornwick:\n%s", s.log)
	}
	sess := s.session(t)
	if _, space := makeAndPlay(t, sess, "Hale"); !strings.EqualFold(space, "thornwick") {
		t.Errorf("a new character started in %q, want thornwick", space)
	}
}

// The kept world's account lists no characters before one is made (no
// Testgrounds testers), and then exactly the one made; the Testgrounds setup
// keeps its testers (me-1, me-2), which its script plays.
func TestPlayWorldHoldsNoDevTesters(t *testing.T) {
	list := func(sess *client.Session) []string {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		chars, err := sess.ListCharacters(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var ids []string
		for _, c := range chars {
			ids = append(ids, c.GetCharacter())
		}
		return ids
	}
	rep := dryRun(t)
	if !strings.Contains(rep["seqd"], "-dev-testers=false") {
		t.Fatalf("make play keeps the dev testers: %s", rep["seqd"])
	}
	sess := start(t, strings.Fields(rep["seqd"]), "").session(t)
	if got := list(sess); len(got) != 0 {
		t.Errorf("the kept world's account starts with %v", got)
	}
	id, _ := makeAndPlay(t, sess, "Hale")
	if got := list(sess); len(got) != 1 || got[0] != id {
		t.Errorf("after making %s the account lists %v", id, got)
	}
	rep = dryRun(t, "SEQ_TESTGROUNDS=1")
	if strings.Contains(rep["seqd"], "-dev-testers") {
		t.Errorf("Testgrounds drops its testers: %s", rep["seqd"])
	}
	if got := list(start(t, strings.Fields(rep["seqd"]), "").session(t)); len(got) != 2 {
		t.Errorf("the Testgrounds account lists %v, want me-1 and me-2", got)
	}
}

func TestPlayTestgroundsIsBehindItsFlag(t *testing.T) {
	for _, env := range [][]string{{"SEQ_TESTGROUNDS=1"}} {
		rep := dryRun(t, env...)
		if rep["mode"] != "testgrounds" || !strings.HasPrefix(rep["docker"], "project: seq-tui-play (tmpfs ") {
			t.Fatalf("%v: not the throwaway setup: %v", env, rep)
		}
		if !strings.Contains(rep["seq-tui"], "-character me-1") || !strings.Contains(rep["seqd"], "-start-space testgrounds") {
			t.Fatalf("%v: not me-1 in Testgrounds: %v", env, rep)
		}
		s := start(t, strings.Fields(rep["seqd"]), "")
		sess := s.session(t)
		if _, space := makeAndPlay(t, sess, "Hale"); !strings.EqualFold(space, "testgrounds") {
			t.Errorf("a new character started in %q, want testgrounds", space)
		}
	}
	if rep := dryRun(t); strings.Contains(rep["seq-tui"], "me-1") {
		t.Errorf("make play without the flag plays me-1: %v", rep)
	}
}

func TestPlayPicksNoCharacterUnlessAskedTo(t *testing.T) {
	if rep := dryRun(t); strings.Contains(rep["seq-tui"], "-character") || strings.Contains(rep["seq-tui"], "-new") {
		t.Errorf("the picker is skipped by default: %s", rep["seq-tui"])
	}
	if rep := dryRun(t, "SEQ_NEW=1"); !strings.Contains(rep["seq-tui"], "-new") {
		t.Errorf("SEQ_NEW=1 does not skip the picker: %s", rep["seq-tui"])
	}
	if rep := dryRun(t, "SEQ_CHARACTER=player-9"); !strings.Contains(rep["seq-tui"], "-character player-9") {
		t.Errorf("SEQ_CHARACTER does not skip the picker: %s", rep["seq-tui"])
	}
}

// TestPlayWorldPersistsAndResets runs the real thing against docker (so it is
// opt-in: SEQ_PLAY_DOCKER_TEST=1): a character made on the kept database is
// there after the server and the container are stopped and started again, a
// play-reset that is not confirmed wipes nothing, and a confirmed one wipes the
// volume, characters and all. It uses its own docker project and port, never
// the ones make play does.
func TestPlayWorldPersistsAndResets(t *testing.T) {
	if os.Getenv("SEQ_PLAY_DOCKER_TEST") != "1" {
		t.Skip("set SEQ_PLAY_DOCKER_TEST=1 to run against docker")
	}
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/GeneStevens/seq").Output()
	if err != nil {
		t.Fatal(err)
	}
	compose := filepath.Join(strings.TrimSpace(string(out)), "deploy", "docker-compose.yml")
	project := fmt.Sprintf("tui036-test-%d", os.Getpid())
	_, port, _ := net.SplitHostPort(freeAddr(t))
	env := append(os.Environ(), "SEQ_PG_MOUNT=volume", "SEQ_PG_PORT="+port, "SEQ_PLAY_PROJECT="+project)
	run := func(name string, stdin string, args ...string) (string, error) {
		cmd := exec.Command(name, args...)
		cmd.Env, cmd.Dir = env, ".."
		cmd.Stdin = strings.NewReader(stdin)
		o, err := cmd.CombinedOutput()
		return string(o), err
	}
	dc := func(args ...string) {
		t.Helper()
		if o, err := run("docker", "", append([]string{"compose", "-p", project, "-f", compose}, args...)...); err != nil {
			t.Fatalf("docker compose %v: %v\n%s", args, err, o)
		}
	}
	reset := func(answer string) (string, error) {
		return run("sh", answer+"\n", "scripts/play-reset.sh")
	}
	t.Cleanup(func() { _, _ = reset("yes") })

	flags := strings.Fields(dryRun(t)["seqd"])
	db := "postgres://seq:seq@127.0.0.1:" + port + "/seq?sslmode=disable"

	dc("up", "-d", "--wait")
	s := start(t, flags, db)
	id, _ := makeAndPlay(t, s.session(t), "Hale")
	s.stop()
	dc("down") // what quitting make play does: the volume stays

	dc("up", "-d", "--wait")
	s = start(t, flags, db)
	sess := s.session(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := sess.Select(ctx, id); err != nil {
		t.Fatalf("the character is gone after a restart: %v\n%s", err, s.log)
	}
	sess.View(func(m *model.Model) {
		if m.Self.Name != "Hale" {
			t.Errorf("after the restart the character is %q", m.Self.Name)
		}
	})
	s.stop()

	if o, err := reset("no"); err == nil {
		t.Errorf("play-reset went ahead without a yes:\n%s", o)
	}
	if o, err := run("docker", "", "volume", "inspect", project+"_pgdata"); err != nil {
		t.Errorf("an unconfirmed play-reset removed the volume:\n%s", o)
	}

	if o, err := reset("yes"); err != nil {
		t.Fatalf("play-reset: %v\n%s", err, o)
	}
	if o, err := run("docker", "", "volume", "inspect", project+"_pgdata"); err == nil {
		t.Errorf("play-reset left the volume:\n%s", o)
	}
	dc("up", "-d", "--wait")
	s = start(t, flags, db)
	if err := s.session(t).Select(ctx, id); err == nil {
		t.Errorf("the character survived play-reset")
	}
}
