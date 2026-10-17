// Package scenario drives the real TUI program with key presses against a real
// SEQ server in the test's own process (seq's testserver, D49, doc 03 layer
// 6). A scenario reads as a short script: keys, clock advances, and checks on
// the model and on golden screens at named points. Nothing in it sleeps for
// game time: the server's clock moves when the scenario says so.
//
// The program under test is app.App run by a real tea.Program; its input is a
// pipe the runner writes key bytes to, so a key is parsed as a terminal's would
// be. The session is client.Session, the transport the program uses.
//
// Two speed gates are enforced here (D49): each scenario under MaxScenario, and
// the whole suite under MaxSuite (see Main).
package scenario

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/app"
	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
	"github.com/GeneStevens/seq/agentkit"
	"github.com/GeneStevens/seq/core"
	"github.com/GeneStevens/seq/server"
	"github.com/GeneStevens/seq/testserver"
)

// The speed gates of D49.
const (
	MaxScenario = 2 * time.Second
	MaxSuite    = 15 * time.Second
)

// Environment variables a run reads.
const (
	// EnvFixtures names a directory: a scenario that declares a fixture writes
	// the log it recorded there (`make fixtures`).
	EnvFixtures = "SEQ_FIXTURES"
	// EnvGolden set to 1 rewrites the golden screens instead of comparing them
	// (`make golden`).
	EnvGolden = "SEQ_GOLDEN"
	// EnvWorld names a directory of packs to load instead of the pinned
	// seq-world's.
	EnvWorld = "SEQ_WORLD_PACKS"
	// EnvSlow is a duration each scenario waits at its end, to show a speed
	// gate failing.
	EnvSlow = "SEQ_SCENARIO_SLOW"
)

// Sizes the scenario goldens are drawn at.
var Sizes = [][2]int{{80, 24}, {140, 45}}

// TooSlow says how a scenario that took took, against limit, fails; "" if it passed.
func TooSlow(what string, took, limit time.Duration) string {
	if took <= limit {
		return ""
	}
	return fmt.Sprintf("%s took %v; the limit is %v", what, took.Round(time.Millisecond), limit)
}

// World is a server and the players in it.
type World struct {
	T       *testing.T
	Srv     *testserver.Server
	began   time.Time
	players []*Player
	rec     *record.Writer
}

// New starts a server for the scenario t and holds the scenario to MaxScenario.
func New(t *testing.T, o testserver.Options) *World {
	t.Helper()
	if len(o.Packs) == 0 && usesWorld(o.Zones) {
		o.Packs = []string{WorldPacks(t)}
	}
	return newOn(t, testserver.Start(t, o))
}

func newOn(t *testing.T, srv *testserver.Server) *World {
	t.Helper()
	w := &World{T: t, began: time.Now(), Srv: srv}
	t.Cleanup(func() {
		if d, err := time.ParseDuration(os.Getenv(EnvSlow)); err == nil {
			time.Sleep(d)
		}
		took := time.Since(w.began)
		t.Logf("scenario %s: %v", t.Name(), took.Round(time.Millisecond))
		note(t.Name(), took)
		if msg := TooSlow("scenario "+t.Name(), took, MaxScenario); msg != "" {
			t.Error(msg)
		}
	})
	return w
}

// The zones of seq-world's Thornwick pack (D54), by id. The scenarios load
// them from seq-world through the test server: this repo holds no copy.
const (
	Thornwick = "thornwick"
	Downs     = "thornwick-downs"
	Warren    = "hollow-warren"
)

// Town is the town and the Downs outside its gate. A World on them starts at
// Thornwick.
var Town = []string{Thornwick, Downs}

func usesWorld(zones []string) bool {
	for _, z := range zones {
		if z != testserver.Testgrounds && z != testserver.Outskirts {
			return true
		}
	}
	return false
}

// WorldPacks is the directory of seq-world's packs: the pinned module's
// (go.mod), found by asking the go tool where it is. SEQ_WORLD_PACKS names
// another directory (a checkout of seq-world under work), for a run that wants
// one.
func WorldPacks(t testing.TB) string {
	t.Helper()
	if d := os.Getenv(EnvWorld); d != "" {
		return d
	}
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/GeneStevens/seq-world").Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		t.Fatalf("seq-world is not in the module cache (go list: %v)", err)
	}
	return filepath.Join(strings.TrimSpace(string(out)), "packs")
}

// Fixture says the first player of this world is recorded to the named log in
// the fixtures directory, when a run asks for fixtures (EnvFixtures); else it
// does nothing. Call it before Player.
func (w *World) Fixture(name string) {
	dir := os.Getenv(EnvFixtures)
	if dir == "" {
		return
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		w.T.Fatal(err)
	}
	w.T.Cleanup(func() { _ = f.Close() })
	w.rec = record.NewWriter(f)
}

// Advance moves the server's clock by d and returns when every player has
// taken in what the server sent for it.
func (w *World) Advance(d time.Duration) {
	w.T.Helper()
	w.Srv.Advance(d)
	for _, p := range w.players {
		p.Sync()
	}
}

// Settle returns when every player has taken in what the server sent so far,
// without moving the clock: what one player did for the others to see.
func (w *World) Settle() {
	w.T.Helper()
	for _, p := range w.players {
		p.Sync()
	}
}

// Now is the server's clock as a time, the clock the programs read.
func (w *World) Now() time.Time { return time.UnixMilli(server.ManualEpochMs + w.Srv.NowMs()) }

// Place puts the character at a position before it logs in.
func (w *World) Place(account, character string, x, y int) {
	w.Srv.Place(account, character, core.Pos{X: x, Y: y})
}

// Player is one TUI: a program, its session and its key input.
type Player struct {
	w         *World
	Account   string
	Character string
	Sess      *client.Session
	prog      *tea.Program
	in        *io.PipeWriter
	done      chan struct{}
	keys      int
	rec       *record.Writer
	creating  bool // the program opened on the creation screen
	second    string
	duo       *client.Duo // the account's second character on a stance, if the player has one
	duoAgent  bool        // the second character logs in with an agent credential (its turn-based plans are stand-ins)
}

// Login starts a TUI for the account's character: it logs in with a session
// token, as `seq-tui` does, and runs the program. The first player of a world
// with a fixture is recorded.
func (w *World) Login(account, character string) *Player {
	w.T.Helper()
	p := &Player{w: w, Account: account, Character: character}
	if len(w.players) == 0 {
		p.rec = w.rec
	}
	w.players = append(w.players, p)
	p.start()
	return p
}

// LoginWithDuo is Login for a player who also has the account's second
// character on a stance (D42): it is out of play until the player presses P, and
// the scenario drives it once each time the player syncs, not by a clock of its
// own.
func (w *World) LoginWithDuo(account, character, second string) *Player {
	w.T.Helper()
	p := &Player{w: w, Account: account, Character: character, second: second}
	if len(w.players) == 0 {
		p.rec = w.rec
	}
	w.players = append(w.players, p)
	p.start()
	return p
}

// LoginWithDuoAgent is LoginWithDuo with the second character on an agent
// credential, as SEQ_DUO_AGENT_SECRET gives the program: in a turn-based
// instance its stance sends a stand-in plan each round (D55).
func (w *World) LoginWithDuoAgent(account, character, second string) *Player {
	w.T.Helper()
	p := &Player{w: w, Account: account, Character: character, second: second, duoAgent: true}
	if len(w.players) == 0 {
		p.rec = w.rec
	}
	w.players = append(w.players, p)
	p.start()
	return p
}

// NewCharacter starts a TUI for the account with no character chosen: it
// connects and opens on the creation screen, as `seq-tui -new` does. Press the
// keys that make the character; Created says which it is. The first player of a
// world with a fixture is recorded.
func (w *World) NewCharacter(account string) *Player {
	w.T.Helper()
	p := &Player{w: w, Account: account, creating: true}
	if len(w.players) == 0 {
		p.rec = w.rec
	}
	w.players = append(w.players, p)
	p.start()
	return p
}

// Created waits for the character the creation screen made to be playing, and
// names the player after it: the character's id.
func (p *Player) Created() {
	t := p.w.T
	t.Helper()
	p.Eventually("the new character", func(m *model.Model) bool { return m.Ready() && m.Map != nil && m.Map.GetSpace() == m.Space })
	p.Model(func(m *model.Model) { p.Character = m.Self.ID })
	p.creating = false
}

func (p *Player) start() {
	t := p.w.T
	t.Helper()
	cfg := client.Config{URL: p.w.Srv.URL(), TLS: p.w.Srv.TLS(), Credential: p.w.Srv.SessionToken(p.Account), Character: p.Character, Record: p.rec}
	p.Sess = client.NewSession(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	login := p.Sess.Login
	if p.creating {
		login = p.Sess.Connect
	}
	if err := login(ctx); err != nil {
		t.Fatalf("%s: login: %v", p.Character, err)
	}
	a := app.New(p.Sess)
	if p.creating {
		a = app.NewCreator(p.Sess)
	}
	if p.second != "" {
		dc := cfg
		dc.Character = p.second
		if p.duoAgent {
			dc.Agent = p.w.Srv.AgentSecret(p.Account, p.second)
		}
		p.duo = client.NewDuo(dc, p.Character)
		p.duo.Clock = p.w.Now
		a.SetDuo(p.duo)
	}
	a.Clock = p.w.Now
	pr, pw := io.Pipe()
	p.in, p.keys, p.done = pw, 0, make(chan struct{})
	p.prog = tea.NewProgram(a, tea.WithInput(pr), tea.WithOutput(io.Discard), tea.WithoutRenderer(), tea.WithoutSignalHandler(),
		tea.WithWindowSize(80, 24), tea.WithEnvironment([]string{"TERM=xterm-256color"}))
	go func() {
		defer close(p.done)
		if _, err := p.prog.Run(); err != nil && !errors.Is(err, tea.ErrProgramKilled) {
			t.Errorf("%s: the program ended with %v", p.Character, err)
		}
	}()
	t.Cleanup(p.stop)
	p.Sync()
}

// stop ends the program and the session.
func (p *Player) stop() {
	if p.prog == nil {
		return
	}
	p.prog.Kill()
	<-p.done
	_ = p.in.Close()
	p.Sess.Close()
	if p.duo != nil {
		p.duo.Close()
	}
	p.prog = nil
}

// Fresh closes this player's program and session, as quitting does, and logs
// in again: a new session and a new model, with no memory of the old.
func (p *Player) Fresh() {
	p.w.T.Helper()
	p.stop()
	p.start()
}

var keyBytes = map[string]string{
	"enter": "\r", "backspace": "\x7f", "esc": "\x1b", "tab": "\t", "space": " ",
	"up": "\x1b[A", "down": "\x1b[B", "right": "\x1b[C", "left": "\x1b[D",
}

// Press presses the keys, one after another, waiting after each until the
// program has handled it and the server has answered what it sent. A key is a
// character ("e", "?"), or "enter", "backspace", "esc", "tab", "space" or an
// arrow's name.
func (p *Player) Press(keys ...string) {
	p.w.T.Helper()
	for _, k := range keys {
		b, ok := keyBytes[k]
		if !ok {
			b = k
		}
		if _, err := io.WriteString(p.in, b); err != nil {
			p.w.T.Fatalf("%s: pressing %q: %v", p.Character, k, err)
		}
		p.keys++
		p.Sync()
	}
}

// probe runs f inside the program's loop.
func (p *Player) probe(f func(*app.App)) {
	d := make(chan struct{})
	p.prog.Send(app.Probe{Fn: f, Done: d})
	select {
	case <-d:
	case <-p.done:
		p.w.T.Fatalf("%s: the program ended", p.Character)
	}
}

// Sync returns when the program has handled every key pressed, the server has
// answered every command they sent, and every message the server sent before
// now has reached the model (a ping's pong comes after them).
func (p *Player) Sync() {
	t := p.w.T
	t.Helper()
	p.idle()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.Sess.Ping(ctx); err != nil {
		t.Fatalf("%s: waiting for the server: %v", p.Character, err)
	}
	p.idle()
	p.driveDuo(ctx)
}

// driveDuo lets the second character's stance see what the server has sent and
// send its answer: a ping first, so the character's replica holds everything the
// server sent before now, and one after, so the answers are taken in.
func (p *Player) driveDuo(ctx context.Context) {
	t := p.w.T
	t.Helper()
	if p.duo == nil {
		return
	}
	s := p.duo.Session()
	if s == nil {
		return
	}
	if err := s.Ping(ctx); err != nil {
		t.Fatalf("%s: waiting for the server: %v", p.second, err)
	}
	p.duo.Step(ctx)
	if err := s.Ping(ctx); err != nil {
		t.Fatalf("%s: waiting for the server: %v", p.second, err)
	}
}

// Duo runs f on the model of the second character; it fails the test when that
// character is not in play.
func (p *Player) Duo(f func(*model.Model)) {
	p.w.T.Helper()
	if p.duo == nil || p.duo.Session() == nil {
		p.w.T.Fatalf("%s: the second character is not in play", p.Character)
	}
	p.duo.Session().View(f)
}

// DuoPlan is what the second character's stance sent for the last round it
// played, in words, and the round.
func (p *Player) DuoPlan() ([]string, uint64) { return p.duo.Plan() }

func (p *Player) idle() {
	t := p.w.T
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		ok := false
		p.probe(func(a *app.App) { ok = a.Idle(p.keys) })
		if ok {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: the program did not finish handling %d key presses", p.Character, p.keys)
		}
		time.Sleep(200 * time.Microsecond)
	}
}

// Type presses the keys of the text, one letter at a time, all written before
// the player syncs once (so a line of text costs the connection one round trip,
// not one for each letter). "\r" is Enter and "\x7f" Backspace.
func (p *Player) Type(text string) {
	p.w.T.Helper()
	if _, err := io.WriteString(p.in, text); err != nil {
		p.w.T.Fatalf("%s: typing %q: %v", p.Character, text, err)
	}
	p.keys += len([]rune(text))
	p.Sync()
}

// Model runs f on the model.
func (p *Player) Model(f func(*model.Model)) { p.Sess.View(f) }

// Notice is the status line's note: the last command's answer.
func (p *Player) Notice() (text string, bad bool) {
	p.probe(func(a *app.App) { text, bad = a.Notice() })
	return
}

// Eventually waits, in real time and without moving the server's clock, until
// pred holds on the model and the program shows the connection as up: for what
// the program does by itself, such as dialling again after a drop.
func (p *Player) Eventually(what string, pred func(*model.Model) bool) {
	t := p.w.T
	t.Helper()
	began := time.Now()
	defer func() {
		if d := time.Since(began); d > 300*time.Millisecond {
			t.Logf("SLOW eventually %s: %v", what, d)
		}
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		ok := false
		p.Model(func(m *model.Model) { ok = pred(m) })
		online := false
		p.probe(func(a *app.App) { online = a.Online() })
		if ok && online {
			p.Sync()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: gave up waiting for %s", p.Character, what)
		}
		time.Sleep(time.Millisecond)
	}
}

// DuoEventually waits for pred to hold of the second character's model, for what
// the server does outside the clock (its own handoff).
func (p *Player) DuoEventually(what string, pred func(*model.Model) bool) {
	t := p.w.T
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		ok := false
		p.Duo(func(m *model.Model) { ok = pred(m) })
		if ok {
			p.Sync()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: gave up waiting for %s", p.second, what)
		}
		time.Sleep(time.Millisecond)
	}
}

// Drop ends the connection as a network failure would, and waits for the
// program to dial again and resume, in real time: the clock does not move.
func (p *Player) Drop() {
	t := p.w.T
	t.Helper()
	var n int
	p.Model(func(m *model.Model) { n = m.Connections })
	p.Sess.Drop()
	began := time.Now()
	defer func() {
		if d := time.Since(began); d > 300*time.Millisecond {
			t.Logf("SLOW drop: %v", d)
		}
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		ok, online := false, false
		p.Model(func(m *model.Model) { ok = m.Connections > n })
		p.probe(func(a *app.App) { online = a.Online() })
		if ok && online {
			p.Sync()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: the program did not reconnect", p.Character)
		}
		time.Sleep(time.Millisecond)
	}
}

// Mark names the present point of the player's recording, if it has one.
func (p *Player) Mark(name string) {
	p.w.T.Helper()
	p.Sync()
	if p.rec != nil {
		if err := p.rec.Mark(name); err != nil {
			p.w.T.Fatal(err)
		}
	}
}

// Screen draws what the player sees at w x h, plain.
func (p *Player) Screen(w, h int) string {
	var s string
	p.probe(func(a *app.App) { s = a.Plain(w, h) })
	return s
}

var age = regexp.MustCompile(`\d+\.\ds`)

// maskAges replaces the digits of every age in the screen ("24.1s", the event
// log's) with N, keeping its width. An age is a difference of game times, and
// the server's pump may add a tick of 100 ms to one between two runs, so it is
// not what a golden screen is about.
func maskAges(s string) string {
	return age.ReplaceAllStringFunc(s, func(a string) string {
		return strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return 'N'
			}
			return r
		}, a)
	})
}

// Golden compares the screen with the golden files testdata/scenarios/NAME_WxH.txt
// (one for each size), and with EnvGolden=1 writes them. It also marks the
// recording, so the render tests can replay the log to the same point.
func (p *Player) Golden(name string) {
	t := p.w.T
	t.Helper()
	p.Mark(name)
	for _, sz := range Sizes {
		got := maskAges(p.Screen(sz[0], sz[1])) + "\n"
		file := filepath.Join(root(), "testdata", "scenarios", fmt.Sprintf("%s_%dx%d.txt", name, sz[0], sz[1]))
		if os.Getenv(EnvGolden) == "1" {
			if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(file)
		if err != nil {
			t.Errorf("%s: %v (make golden writes it)", filepath.Base(file), err)
			continue
		}
		if got != string(want) {
			t.Errorf("%s differs from its golden screen (make golden rewrites it after review)\n--- got\n%s--- want\n%s", filepath.Base(file), got, want)
		}
	}
}

// root is the repository root, found from this file's directory.
func root() string {
	wd, _ := os.Getwd()
	for d := wd; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		if d == filepath.Dir(d) {
			return wd
		}
	}
}

var (
	timesMu sync.Mutex
	times   []string
)

func note(name string, took time.Duration) {
	timesMu.Lock()
	defer timesMu.Unlock()
	times = append(times, fmt.Sprintf("  %-50s %v", name, took.Round(time.Millisecond)))
}

// Main is a TestMain for a package of scenarios: it runs them, prints each one's
// time and holds the whole suite to MaxSuite.
func Main(m *testing.M) {
	began := time.Now()
	code := m.Run()
	total := time.Since(began)
	if len(times) > 0 {
		fmt.Printf("scenario times:\n%s\nsuite: %v\n", strings.Join(times, "\n"), total.Round(time.Millisecond))
	}
	if msg := TooSlow("the scenario suite", total, MaxSuite); msg != "" {
		fmt.Println("FAIL:", msg)
		code = 1
	}
	os.Exit(code)
}

// Plan is the plan the player is building, a line a step.
func (p *Player) Plan() []string {
	var lines []string
	p.probe(func(a *app.App) { lines = a.PlanLines() })
	return lines
}

// DuoReplica runs f on the toolkit's replica of the second character.
func (p *Player) DuoReplica(f func(*agentkit.Replica)) { p.duo.Session().Replica(f) }
