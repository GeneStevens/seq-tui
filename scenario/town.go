package scenario

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/GeneStevens/seq/content"
	"github.com/GeneStevens/seq/core"
	"github.com/GeneStevens/seq/server"
	"github.com/GeneStevens/seq/sim"
	"github.com/GeneStevens/seq/store"
	"github.com/GeneStevens/seq/testserver"
)

// Server is what a World needs of the server it plays on: the clock, a setup
// for a character before it logs in, and a look at it afterwards. Both
// testserver.Server (the placeholder zones) and Town (Thornwick) are one.
type Server interface {
	URL() string
	TLS() *tls.Config
	SessionToken(account string) string
	NowMs() int64
	Advance(d time.Duration)
	Place(account, id string, at core.Pos)
	Give(id string, edit func(*core.CharacterSave))
	Attend(account string)
	Character(id string) (testserver.Character, bool)
}

// The spaces of the town server, as in cmd/seqd.
const (
	Thornwick = "thornwick"
	Downs     = "thornwick-downs"
)

// TownOptions say what a Town starts with.
type TownOptions struct {
	// Seed seeds the world; 0 means 1.
	Seed uint64
	// Accounts and their characters are made before the test starts. A
	// character is homed in Thornwick unless the account's Home says otherwise;
	// Classed gives a character a class (its id is the class's).
	Accounts []testserver.Account
	// Zone, when set, is called with each zone before the world starts (the
	// server's own copy), to adjust it.
	Zone func(space string, z *core.Zone)
	// State, when set, is called with each space's first state, to adjust what
	// the content leaves to a timer (a rare's first spawn, an event's first start).
	State func(space string, st *sim.State)
}

// Town is a test server like seq's testserver, but running Thornwick and the
// Thornwick Downs, which the testserver does not know (it runs the placeholder
// zones). It is built the way testserver is, from the same packages and the same
// manual clock, so a scenario moves time the same way: Advance, never a sleep.
type Town struct {
	t     testing.TB
	clock *server.ManualClock
	pacer *server.ManualPacer
	env   server.Env
	cl    *server.Cluster
	sh    *server.Shell
	gw    *server.Gateway
	srv   *server.Server
	sched *server.Scheduler
	acc   store.Accounts
	dir   store.Directory
	tls   *tls.Config
	zone  map[string]*core.Zone
	mu    sync.Mutex
	done  chan struct{}
	wg    sync.WaitGroup
}

const (
	stepMs    = 100
	maxPasses = 10_000
)

// StartTown starts the town server and stops it when the test ends.
func StartTown(t testing.TB, o TownOptions) *Town {
	t.Helper()
	s, err := startTown(t, o)
	if err != nil {
		t.Fatalf("town server: %v", err)
	}
	t.Cleanup(s.Stop)
	return s
}

func startTown(t testing.TB, o TownOptions) (*Town, error) {
	seed := o.Seed
	if seed == 0 {
		seed = 1
	}
	s := &Town{t: t, clock: &server.ManualClock{}, pacer: &server.ManualPacer{}, zone: map[string]*core.Zone{}}
	s.dir = store.NewMemoryDirectory()
	s.acc = store.NewMemoryAccounts()
	s.env = server.Env{
		LogLevel: server.LogLevelError, Clock: s.clock, Pacer: s.pacer,
		Journal: store.NewMemoryJournal(), Store: store.NewMemory(),
		HandoffTimeoutMs: 30_000, HibernateAfterMs: server.DefaultHibernateAfterMs,
		Present: server.PresentFrom(s.acc),
	}
	var specs []server.SpaceSpec
	for i, name := range []string{Thornwick, Downs} {
		z, err := content.Zone(zoneID(name))
		if err != nil {
			return nil, err
		}
		if o.Zone != nil {
			o.Zone(name, z)
		}
		st, err := sim.NewState(z, seed+uint64(i), nil)
		if err != nil {
			return nil, err
		}
		if o.State != nil {
			o.State(name, &st)
		}
		s.zone[name] = z
		specs = append(specs, server.SpaceSpec{Name: name, Initial: st, CheckpointEveryMs: server.DefaultCheckpointEveryMs})
	}
	s.cl = server.NewCluster(s.dir, s.clock)
	sh, err := s.cl.StartNode("town", s.env, specs)
	if err != nil {
		return nil, err
	}
	s.sh = sh
	for _, sp := range specs {
		s.cl.Serve(zoneID(sp.Name), sp.Name)
	}
	rounds, ok := s.env.Store.(store.Rounds)
	if !ok {
		s.cl.Stop()
		return nil, errors.New("the memory store keeps no rounds")
	}
	s.sched = s.cl.Scheduler(rounds)
	if s.gw, err = sh.StartGateway(server.GatewayConfig{Accounts: s.acc, DefaultSpace: Thornwick, StartSpace: Thornwick, RevocationCheck: -1}); err != nil {
		s.cl.Stop()
		return nil, err
	}
	cert, pool, err := server.SelfSignedCert()
	if err != nil {
		s.cl.Stop()
		return nil, err
	}
	s.tls = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS13}
	if s.srv, err = s.gw.Listen("127.0.0.1:0", cert); err != nil {
		s.cl.Stop()
		return nil, err
	}
	for _, a := range o.Accounts {
		if err := s.addAccount(a); err != nil {
			s.Stop()
			return nil, err
		}
	}
	s.Idle()
	s.done = make(chan struct{})
	s.wg.Add(1)
	go s.pump()
	return s, nil
}

func zoneID(space string) string {
	if space == Downs {
		return content.DownsID
	}
	return content.ThornwickID
}

func (s *Town) addAccount(a testserver.Account) error {
	if _, err := s.acc.CreateAccount(a.Name, a.Name); err != nil && !errors.Is(err, store.ErrAccountExists) {
		return err
	}
	home := a.Home
	if home == "" {
		home = Thornwick
	}
	nameOf := func(id string) string {
		if n := a.Names[id]; n != "" {
			return n
		}
		return id
	}
	ids := make([]string, 0, len(a.Classed))
	for id := range a.Classed {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		class := a.Classed[id]
		if err := s.acc.AddCharacter(store.AccountCharacter{ID: id, Account: a.Name, Name: nameOf(id), Template: class, Class: class, Home: home}); err != nil {
			return fmt.Errorf("character %s: %w", id, err)
		}
	}
	return nil
}

// pump keeps a client from waiting on a clock nobody is moving: while a space
// holds work waiting for its tick it gives the world one tick of 100 ms.
func (s *Town) pump() {
	defer s.wg.Done()
	for {
		select {
		case <-s.done:
			return
		case <-time.After(200 * time.Microsecond):
		}
		s.mu.Lock()
		if s.cl != nil && s.waiting() {
			s.clock.Advance(stepMs)
			if err := s.settle(); err != nil {
				s.t.Errorf("town server: %v", err)
			}
		}
		s.mu.Unlock()
	}
}

func (s *Town) waiting() bool {
	for _, sp := range s.pacer.Keys() {
		if snap, err := s.cl.Snapshot(sp); err == nil && (snap.Pending > 0 || snap.PendingRoster > 0) {
			return true
		}
	}
	return false
}

// Stop ends the server; Start has registered it for the end of the test.
func (s *Town) Stop() {
	if s.done != nil {
		close(s.done)
		s.wg.Wait()
		s.done = nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.srv != nil {
		_ = s.srv.Close()
		s.srv = nil
	}
	if s.cl != nil {
		s.cl.Stop()
		s.cl = nil
	}
}

// URL is the WebSocket URL a client dials.
func (s *Town) URL() string { return "wss://" + s.srv.Addr + server.GatewayPath }

// TLS is the trust config a client needs for the server's throwaway certificate.
func (s *Town) TLS() *tls.Config { return s.tls.Clone() }

// SessionToken issues a human session token for the account.
func (s *Town) SessionToken(account string) string {
	s.t.Helper()
	tok, err := s.acc.IssueSessionToken(account, s.clock.NowMs(), store.DevLoginTTLMs)
	if err != nil {
		s.t.Fatalf("town server: session token for %s: %v", account, err)
	}
	return tok
}

// NowMs is the test clock's reading.
func (s *Town) NowMs() int64 { return s.clock.NowMs() }

// Advance moves game and wall time together by d and returns once the server has
// handled everything that came due, in steps of at most 10 s.
func (s *Town) Advance(d time.Duration) {
	s.t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	left := d.Milliseconds()
	for first := true; left > 0 || first; first = false {
		step := min(left, 100*stepMs)
		s.clock.Advance(step)
		left -= step
		if err := s.settle(); err != nil {
			s.t.Fatalf("town server: %v", err)
		}
	}
}

// Idle returns once the server has handled everything sent to it; time does
// not move.
func (s *Town) Idle() {
	s.t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncAll()
}

func (s *Town) settle() error {
	last := ""
	for pass := 0; ; pass++ {
		if pass >= maxPasses {
			return fmt.Errorf("the server never settled at %d ms", s.clock.NowMs())
		}
		if _, err := s.sched.Poll(); err != nil {
			return fmt.Errorf("scheduler: %w", err)
		}
		s.syncAll()
		for _, sp := range s.pacer.Keys() {
			s.pacer.Pulse(sp)
		}
		s.syncAll()
		if now := s.fingerprint(); now == last {
			return nil
		} else {
			last = now
		}
	}
}

func (s *Town) fingerprint() string {
	var b []byte
	for _, sp := range s.pacer.Keys() {
		snap, err := s.cl.Snapshot(sp)
		if err != nil {
			continue
		}
		b = fmt.Appendf(b, "%s=%d,%d;", sp, snap.State.Tick, snap.Pending)
	}
	return string(b)
}

func (s *Town) syncAll() {
	for range 2 {
		for _, sp := range s.pacer.Keys() {
			s.sh.SyncUnlessGone(sp)
		}
		s.gw.Settle()
	}
}

// Character reads a character's server-side state for an assertion.
func (s *Town) Character(id string) (testserver.Character, bool) {
	s.t.Helper()
	own, ok, err := s.dir.Ownership(id)
	if err != nil {
		s.t.Fatalf("town server: ownership of %s: %v", id, err)
	}
	if !ok || own.State != store.InSpace {
		return testserver.Character{}, false
	}
	snap, err := s.cl.Snapshot(own.Space)
	if err != nil {
		return testserver.Character{}, false
	}
	e := snap.State.Find(core.EntityID(id))
	if e == nil {
		return testserver.Character{}, false
	}
	return testserver.Character{Space: own.Space, Entity: e.Clone()}, true
}

// Place puts a character, which must not be in any space yet, at a position in
// its home space, for a scenario's setup.
func (s *Town) Place(account, id string, at core.Pos) {
	s.t.Helper()
	chars, err := s.acc.Characters(account)
	if err != nil {
		s.t.Fatalf("town server: characters of %s: %v", account, err)
	}
	var home, template, name string
	for _, c := range chars {
		if c.ID == id {
			home, template, name = c.Home, c.Template, c.Name
		}
	}
	z := s.zone[home]
	if z == nil {
		s.t.Fatalf("town server: %s is not a character of %s, or its home is not loaded", id, account)
	}
	if _, ok, _ := s.dir.Ownership(id); ok {
		s.t.Fatalf("town server: Place is for setup: %s is already in the world", id)
	}
	cs, err := sim.Arrival(z, sim.Join{ID: core.EntityID(id), Template: template, Account: account, Name: name})
	if err != nil {
		s.t.Fatalf("town server: %v", err)
	}
	cs.Pos = at
	data, err := json.Marshal(cs)
	if err != nil {
		s.t.Fatal(err)
	}
	if _, err := s.env.Store.SaveCharacter(id, 0, data); err != nil {
		s.t.Fatalf("town server: placing %s: %v", id, err)
	}
}

// Give changes the saved state of a character that was Placed and is not in the
// world yet: crowns, items, a level.
func (s *Town) Give(id string, edit func(*core.CharacterSave)) {
	s.t.Helper()
	rec, ok, err := s.env.Store.LoadCharacter(id)
	if err != nil || !ok {
		s.t.Fatalf("town server: Give is for a character that was Placed: %s (%v)", id, err)
	}
	var cs core.CharacterSave
	if err := json.Unmarshal(rec.Data, &cs); err != nil {
		s.t.Fatalf("town server: character %s: %v", id, err)
	}
	edit(&cs)
	data, err := json.Marshal(cs)
	if err != nil {
		s.t.Fatal(err)
	}
	if _, err := s.env.Store.SaveCharacter(id, rec.Version, data); err != nil {
		s.t.Fatalf("town server: giving %s: %v", id, err)
	}
}

// Attend records that the account's human gave input now (D8, D31).
func (s *Town) Attend(account string) {
	s.t.Helper()
	if err := s.acc.RecordPresence(account, s.clock.UnixMs()); err != nil {
		s.t.Fatalf("town server: presence of %s: %v", account, err)
	}
}
