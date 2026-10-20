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
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/app"
	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
	"github.com/GeneStevens/seq-tui/sessionlog"
	"github.com/GeneStevens/seq/agentkit"
	"github.com/GeneStevens/seq/core"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
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

// Tick is the server's tick: how far the clock moves for a wait on one intent.
const Tick = 100 * time.Millisecond

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
	logRoot string // where the players' own session logs are kept (D64)
	nlogs   int    // sessions started, for their ids
	// stepped: the scenario is held to MaxScenario a step (see Step), not as a
	// whole.
	stepped bool
	stepAt  time.Time
}

// New starts a server for the scenario t and holds the scenario to MaxScenario.
func New(t *testing.T, o testserver.Options) *World {
	t.Helper()
	if len(o.Packs) == 0 && usesWorld(o.Zones) {
		o.Packs = []string{WorldPacks(t)}
	}
	o.Manual = true
	return newOn(t, testserver.Start(t, o))
}

func newOn(t *testing.T, srv *testserver.Server) *World {
	t.Helper()
	w := &World{T: t, began: time.Now(), Srv: srv, logRoot: filepath.Join(t.TempDir(), "sessions")}
	t.Cleanup(func() {
		if d, err := time.ParseDuration(os.Getenv(EnvSlow)); err == nil {
			time.Sleep(d)
		}
		took := time.Since(w.began)
		t.Logf("scenario %s: %v", t.Name(), took.Round(time.Millisecond))
		if w.stepped {
			return // every step was held to the limit as it ended
		}
		note(t.Name(), took)
		if msg := TooSlow("scenario "+t.Name(), took, MaxScenario); msg != "" {
			t.Error(msg)
		}
	})
	return w
}

// Step runs one step of a scenario that is too long for MaxScenario as a whole
// (a delve cleared from end to end): the steps are held to the limit one by one,
// the first from the start of the scenario, each of the others from the end of the
// one before, and the suite is still held to MaxSuite. A scenario of one piece
// stays under MaxScenario in all.
func (w *World) Step(name string, f func()) {
	w.T.Helper()
	if !w.stepped {
		w.stepped, w.stepAt = true, w.began
	}
	f()
	took := time.Since(w.stepAt)
	w.stepAt = time.Now()
	w.T.Logf("scenario %s, step %s: %v", w.T.Name(), name, took.Round(time.Millisecond))
	note(w.T.Name()+" "+name, took)
	if msg := TooSlow("scenario "+w.T.Name()+", step "+name, took, MaxScenario); msg != "" {
		w.T.Error(msg)
	}
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
	for d > 0 {
		step := d
		if w.walking() {
			// A held key is kept alive, or let go, as the program's frame clock
			// does in real time; the server's walk lasts 500 ms.
			step = min(d, 3*Tick)
		}
		w.barrier() // every player's intents are in before the clock moves
		w.Srv.Advance(step)
		for _, p := range w.players {
			p.Sync()
		}
		d -= step
	}
}

// walking says some player's program has a movement key down, or a walk it has
// not yet stopped.
func (w *World) walking() bool {
	any := false
	for _, p := range w.players {
		if p.prog != nil {
			p.probe(func(a *app.App) { any = any || a.WalkActive() })
		}
	}
	return any
}

// Settle returns when every player has taken in what the server sent so far,
// without moving the clock: what one player did for the others to see.
func (w *World) Settle() {
	w.T.Helper()
	for _, p := range w.players {
		p.Sync()
	}
}

// LogRoot is the directory the players' session logs are kept in.
func (w *World) LogRoot() string { return w.logRoot }

// Now is the server's clock as a time, the clock the programs read.
func (w *World) Now() time.Time { return time.UnixMilli(server.ManualEpochMs + w.Srv.NowMs()) }

// Place puts the character at a position before it logs in.
func (w *World) Place(account, character string, x, y int) {
	w.Srv.Place(account, character, core.Pos{X: x, Y: y})
}

// Player is one TUI: a program, its session and its key input.
type Player struct {
	w         *World
	stepCarry int64 // how far the last Step overshot 3 m
	stepCoast int   // the ticks the character walked on after the key came up
	Account   string
	Character string
	Sess      *client.Session
	prog      *tea.Program
	in        *io.PipeWriter
	done      chan struct{}
	keys      int
	rec       *record.Writer
	log       *sessionlog.Log
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
	// Like the program, a player records its session under its id (D64); the id
	// is counted, not random, so a screen that shows it is stable.
	p.w.nlogs++
	cfg := client.Config{URL: p.w.Srv.URL(), TLS: p.w.Srv.TLS(), Credential: p.w.Srv.SessionToken(p.Account), Character: p.Character, Record: p.rec,
		Session: fmt.Sprintf("tui-%s-%d", p.Account, p.w.nlogs)}
	who := p.Character
	if p.creating {
		who = ""
	}
	slog, err := sessionlog.Open(p.w.logRoot, sessionlog.Options{Credential: cfg.Credential, Now: p.w.Now, Meta: sessionlog.Meta{
		Session: cfg.Session, Server: p.w.Srv.URL(), Character: who, Client: "seq-tui", Versions: sessionlog.Versions(client.ProtocolVersion)}})
	if err != nil {
		t.Fatalf("%s: the session log: %v", p.Account, err)
	}
	p.log, cfg.Log = slog, slog.Rec
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
		dc.Log = nil // the second character is not a session of its own
		dc.Character = p.second
		if p.duoAgent {
			dc.Agent = p.w.Srv.AgentSecret(p.Account, p.second)
		}
		p.duo = client.NewDuo(dc, p.Character)
		p.duo.Clock = p.w.Now
		a.SetDuo(p.duo)
	}
	a.Clock = p.w.Now
	a.SetLogs(p.w.logRoot, slog)
	a.LogLabel = "<the scenario's log directory>"
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
	if p.log != nil {
		_ = p.log.Close()
		p.log = nil
	}
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
	"ctrl+l": "\x0c", "ctrl+k": "\x0b", "ctrl+a": "\x01", "f1": "\x1bOP", "home": "\x1b[H", "end": "\x1b[F", "up": "\x1b[A", "down": "\x1b[B", "right": "\x1b[C", "left": "\x1b[D",
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

// Kitty makes this player's terminal one that reports key repeats and releases
// (the kitty keyboard protocol's event types), as the program learns when the
// terminal answers its request for them.
func (p *Player) Kitty() {
	p.w.T.Helper()
	p.prog.Send(tea.KeyboardEnhancementsMsg{Flags: 1 | 2})
	p.probe(func(*app.App) {})
}

// Legacy makes the terminal one that reports no releases again.
func (p *Player) Legacy() {
	p.w.T.Helper()
	p.prog.Send(tea.KeyboardEnhancementsMsg{Flags: 0})
	p.probe(func(*app.App) {})
}

// Key events of the kitty protocol.
const (
	KeyDown    = 1
	KeyRepeat  = 2
	KeyRelease = 3
)

// kittyKey is the bytes a terminal that reports event types sends for a key's
// press, repeat or release: CSI 1 ; 1 : event A for an arrow, CSI code ; 1 : event
// u for a letter.
func kittyKey(key string, event int) string {
	arrows := map[string]string{"up": "A", "down": "B", "right": "C", "left": "D"}
	if f, ok := arrows[key]; ok {
		return fmt.Sprintf("\x1b[1;1:%d%s", event, f)
	}
	return fmt.Sprintf("\x1b[%d;1:%du", []rune(key)[0], event)
}

// KeyEvent writes one key event as a terminal with the kitty protocol sends it:
// event is KeyDown, KeyRepeat or KeyRelease. The clock does not move.
func (p *Player) KeyEvent(key string, event int) {
	p.w.T.Helper()
	if _, err := io.WriteString(p.in, kittyKey(key, event)); err != nil {
		p.w.T.Fatalf("%s: key %q event %d: %v", p.Character, key, event, err)
	}
	p.keys++
	p.Sync()
}

// Step walks one step of 3 m the way a player taps a key on a terminal that
// reports releases: the key goes down, the clock runs until the character has
// moved 3 m, the key comes up and the walk has stopped. It fails if the
// character does not move.
func (p *Player) Step(key string) {
	p.w.T.Helper()
	if !p.TryStep(key) {
		p.w.T.Fatalf("%s: a step %s moved the character nowhere", p.Character, key)
	}
}

// TryStep is Step that says whether the character moved, for a walk a fight may
// stop (D66). The steps average 3 m: a tick of walking is a few hundred
// millimetres, so each step makes up for the last one's overshoot, and for the
// tick the character coasts after the key comes up.
func (p *Player) TryStep(key string) bool { return p.TryStepBy(key, 3000) }

// TryStepBy is TryStep for a walk of mm millimetres.
func (p *Player) TryStepBy(key string, mm int64) bool {
	p.w.T.Helper()
	p.Kitty()
	from := p.pos()
	var perTick int64 = 400
	p.Model(func(m *model.Model) {
		if v := m.Self.EffectiveSpeedMmS; v > 0 {
			perTick = v / 10
		}
	})
	target := mm - p.stepCarry
	p.KeyEvent(key, KeyDown)
	for range 80 {
		rem := target - p.dist(from) - int64(p.stepCoast)*perTick
		if rem < perTick/2 {
			break
		}
		p.w.Advance(time.Duration(min((rem+perTick/2)/perTick, 3)) * Tick)
	}
	atRelease := p.dist(from)
	p.KeyEvent(key, KeyRelease)
	for range 5 {
		p.w.Advance(Tick)
		walking := false
		p.probe(func(a *app.App) { walking = a.Walking() })
		if !walking {
			break
		}
	}
	p.Legacy()
	moved := p.dist(from)
	p.stepCarry = max(min(moved-mm, 1500), -1500)
	p.stepCoast = int((moved - atRelease + perTick/2) / perTick)
	return moved > 0
}

// Pos is the character's position in millimetres, as the server last said.
func (p *Player) Pos() [2]int64 { return p.pos() }

func (p *Player) pos() (xy [2]int64) {
	p.Model(func(m *model.Model) { xy = [2]int64{m.Self.Pos.GetX(), m.Self.Pos.GetY()} })
	return
}

func (p *Player) dist(from [2]int64) int64 {
	to := p.pos()
	return int64(math.Hypot(float64(to[0]-from[0]), float64(to[1]-from[1])))
}

// WalkSends lists the movement intents the program has sent for held keys.
func (p *Player) WalkSends() (out []app.WalkSend) {
	p.probe(func(a *app.App) { out = a.WalkSends() })
	return
}

// Drawn0 is Drawn as a pair.
func (p *Player) Drawn0() [2]int64 {
	x, y := p.Drawn()
	return [2]int64{x, y}
}

// Drawn is where the screen draws the character: its predicted mark.
func (p *Player) Drawn() (x, y int64) {
	p.probe(func(a *app.App) { x, y = a.Drawn() })
	return
}

// LogOut presses O, the logout key, and waits for the program to end, as it
// does once the server has let the character go. The player is then done and
// leaves the world: the scenario goes on with the others.
func (p *Player) LogOut() {
	p.w.T.Helper()
	if _, err := io.WriteString(p.in, "o"); err != nil {
		p.w.T.Fatalf("%s: pressing the logout key: %v", p.Character, err)
	}
	select {
	case <-p.done:
	case <-time.After(5 * time.Second):
		p.w.T.Fatalf("%s: the program did not end after the logout key", p.Character)
	}
	loggedOut := p.Sess.LoggedOut()
	p.stop()
	p.w.players = slices.DeleteFunc(p.w.players, func(q *Player) bool { return q == p })
	if !loggedOut {
		p.w.T.Fatalf("%s: the session does not say it is logged out", p.Character)
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
	if p.prog != nil {
		// A frame tick at the new time: a held key is kept alive or let go.
		busy := false
		p.probe(func(a *app.App) { busy = a.Frame() })
		if busy {
			p.idle()
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// A pong comes from the session's actor and can overtake an event still on its
	// way from a space; the server is made idle first, so that what it sent is on
	// the socket ahead of the pong, and the player has all of it when this returns
	// (and what a scenario decides on what it sees does not depend on luck).
	p.w.Srv.Idle()
	p.ping(ctx)
	if p.duo != nil && p.duo.Session() != nil {
		p.driveDuo(ctx)
		p.w.Srv.Idle()
		p.ping(ctx)
	}
}

// ping returns when the server has answered this player's ping and the player
// has taken in everything the server sent before the answer.
func (p *Player) ping(ctx context.Context) {
	t := p.w.T
	t.Helper()
	if err := p.Sess.Ping(ctx); err != nil {
		t.Fatalf("%s: waiting for the server: %v", p.Character, err)
	}
	p.idle()
}

// driveDuo lets the second character's stance see what the server has sent and
// send its answer: a ping first, so the character's replica holds everything the
// server sent before now, and one after, so the answers are taken in.
func (p *Player) driveDuo(ctx context.Context) {
	t := p.w.T
	t.Helper()
	s := p.duo.Session()
	if err := s.Ping(ctx); err != nil {
		t.Fatalf("%s: waiting for the server: %v", p.second, err)
	}
	// Step waits for the results of what it sends: the clock moves for them.
	stepped := make(chan struct{})
	go func() { defer close(stepped); p.duo.Step(ctx) }()
	for done := false; !done; {
		select {
		case <-stepped:
			done = true
		case <-time.After(200 * time.Microsecond):
			if p.waitingOn(s) {
				p.tick(s)
			}
		}
	}
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

// idle returns when the program has handled every key and heard everything the
// server sent. The program's commands wait for their results, and under the
// Manual clock only the scenario moves it: while the program waits on an intent
// of its own, idle gives the server one tick (see tick).
func (p *Player) idle() {
	t := p.w.T
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		ok, commanding := false, false
		p.probe(func(a *app.App) { ok, commanding = a.Idle(p.keys), a.Commanding() })
		if ok {
			return
		}
		// Only a command of the program's own moves the clock: an intent the
		// scenario sent itself waits for its own advance.
		if commanding && p.waiting() {
			p.tick(p.sessions()...)
			continue
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: the program did not finish handling %d key presses", p.Character, p.keys)
		}
		time.Sleep(200 * time.Microsecond)
	}
}

// waiting says an intent this player sent has no result yet: the program's
// command is waiting for the clock.
func (p *Player) waiting() bool { return p.waitingOn(p.sessions()...) }

// waitingOn is waiting for the intents of the sessions.
func (p *Player) waitingOn(ss ...*client.Session) bool {
	n := 0
	for _, s := range ss {
		if s.Writing() {
			return false // not on the wire yet: look again
		}
		s.View(func(m *model.Model) { n += len(m.PendingIntents()) })
	}
	return n > 0
}

// sessions are the player's own and, once it is in play, its second character's.
func (p *Player) sessions() []*client.Session {
	ss := []*client.Session{p.Sess}
	if p.duo != nil && p.duo.Session() != nil {
		ss = append(ss, p.duo.Session())
	}
	return ss
}

// tick moves the server's clock by one tick (100 ms) for the intents the
// program waits on, after the barrier: every player's intents are in before the
// clock moves, and land in the same tick, whatever the speed of the sockets.
func (p *Player) tick(ss ...*client.Session) {
	// What the server answers without a tick (a slot set, say) may still be on
	// its way: take it in first, and move the clock only for what is still
	// waiting for it, or the number of ticks would depend on the race.
	p.w.barrier()
	if !p.waitingOn(ss...) {
		return
	}
	p.w.Srv.Advance(Tick)
	// What the tick sent is on the sockets, not yet in the models: a pending
	// intent the tick answered must not look pending to the next look.
	p.w.barrier()
}

// barrier returns when the server has taken in everything every player's
// sessions sent so far and every player has everything the server sent: a
// ping on each session (a session answers in the order it was written to), then
// the server idle, then a ping again. The clock does not move.
func (w *World) barrier() {
	t := w.T
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pingAll := func() {
		for _, q := range w.players {
			for _, s := range q.sessions() {
				if err := s.Ping(ctx); err != nil {
					t.Fatalf("%s: waiting for the server: %v", q.Character, err)
				}
			}
		}
	}
	pingAll()
	w.Srv.Idle()
	pingAll()
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

// Repeat writes n presses of an arrow or letter key, as a terminal repeating it
// sends them, before the player syncs once.
func (p *Player) Repeat(key string, n int) {
	p.w.T.Helper()
	b, ok := keyBytes[key]
	if !ok {
		b = key
	}
	for range n {
		if _, err := io.WriteString(p.in, b); err != nil {
			p.w.T.Fatalf("%s: pressing %q: %v", p.Character, key, err)
		}
	}
	p.keys += n
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

// connectStamp is the time of a server log's connect line. The server stamps it
// when the session starts, which can be before or after the first 100 ms tick
// of the pump, so its fraction of a second differs between two runs.
var connectStamp = regexp.MustCompile(`(\d\d:\d\d:\d\d\.)\d{3}(  connect:)`)

// maskAges replaces the digits of every age in the screen ("24.1s", the event
// log's) with N, keeping its width. An age is a difference of game times, and
// the server's pump used to add a tick of 100 ms to one between two runs; the
// clock is the scenario's now, but an age is still not what a golden screen is
// about.
func maskAges(s string) string {
	s = connectStamp.ReplaceAllString(s, "${1}NNN${2}")
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

// Resize makes the window w x h cells, as the terminal tells the program.
func (p *Player) Resize(w, h int) {
	p.w.T.Helper()
	p.probe(func(a *app.App) { a.Resize(w, h) })
}

// mouseEvent writes one SGR mouse report, the bytes a terminal sends for the
// pointer at the screen cell (x, y), counted from 0: code is the button and
// modifier bits, press says M (press, motion) rather than m (release).
func (p *Player) mouseEvent(code, x, y int, press bool) {
	p.w.T.Helper()
	end := "m"
	if press {
		end = "M"
	}
	if _, err := io.WriteString(p.in, fmt.Sprintf("\x1b[<%d;%d;%d%s", code, x+1, y+1, end)); err != nil {
		p.w.T.Fatalf("%s: the mouse at %d,%d: %v", p.Character, x, y, err)
	}
	p.keys++
	p.Sync()
}

const (
	mouseMotion = 32
	mouseAlt    = 8
	mouseUp     = 64
	mouseDown   = 65
)

// Click is a left click at the screen cell (x, y): the press and the release.
func (p *Player) Click(x, y int) {
	p.w.T.Helper()
	p.mouseEvent(0, x, y, true)
	p.mouseEvent(0, x, y, false)
}

// RightClick is a click of the right button.
func (p *Player) RightClick(x, y int) {
	p.w.T.Helper()
	p.mouseEvent(2, x, y, true)
	p.mouseEvent(2, x, y, false)
}

// Hover moves the pointer to the cell with no button held; alt says Alt is held.
func (p *Player) Hover(x, y int, alt bool) {
	p.w.T.Helper()
	code := 3 + mouseMotion // no button
	if alt {
		code += mouseAlt
	}
	p.mouseEvent(code, x, y, true)
}

// Wheel turns the scroll wheel over the cell: up zooms in.
func (p *Player) Wheel(x, y int, up bool) {
	p.w.T.Helper()
	code := mouseDown
	if up {
		code = mouseUp
	}
	p.mouseEvent(code, x, y, true)
}

// Where is the screen cell at which the program draws the world place (millimetres).
func (p *Player) Where(pos *pb.Pos) (x, y int, ok bool) {
	p.probe(func(a *app.App) { x, y, ok = a.CellOf(pos) })
	return
}

// Mouse is what the mouse state holds: see app.App.MouseState.
func (p *Player) Mouse() (zoom, bigZoom int, card string, labels bool, tab int, drawer bool) {
	p.probe(func(a *app.App) { zoom, bigZoom, card, labels, tab, drawer = a.MouseState() })
	return
}
