// Package client is the TUI's connection to a SEQ server: a Session that feeds
// the model from the agent toolkit's connection, and the commands that send
// intents. The WebSocket, the login, the reconnect and the resume are the
// toolkit's (agentkit in the seq module); this package has no transport.
package client

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
	"github.com/GeneStevens/seq/agentkit"
	"github.com/GeneStevens/seq/protocol"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// ProtocolVersion is the protocol version this client speaks.
const ProtocolVersion = agentkit.ProtocolVersion

// Config says how a Session connects.
type Config struct {
	URL        string      // wss://host:port/seq/v1
	TLS        *tls.Config // nil: the system roots
	Credential string      // a human session token (doc 04 §3)
	Character  string
	// Session names this play session; a resume names it again. Empty: a
	// random one is made.
	Session string
	// Record, if set, receives every incoming message.
	Record *record.Writer
	// Log, if set, receives them too: the session's own log (D64).
	Log *record.Writer
	// Agent, for a second character on a stance (client.Duo), is an agent
	// credential for it: with one its turn-based plans are stand-ins (D10);
	// without, they are the character's own, sent as the player.
	Agent string
	// Model is the model to feed; nil makes a new one.
	Model *model.Model
	// NoFrames asks the server for one message for each event, as before
	// SEQ-119, in place of one frame for each tick with the self view sent as
	// deltas (D89). A recording made either way replays.
	NoFrames bool
}

// features are what hello asks for.
func (c Config) features() []string {
	if c.NoFrames {
		return []string{} // asks for nothing (nil would take the toolkit's default)
	}
	return []string{protocol.FeatureTickFrames}
}

// Session is one play session: the agent toolkit's connection (D41) with the
// model fed from its raw-message hook. The toolkit owns the WebSocket, the
// login and the resume; the session adds the model, the recording, and one
// command per intent.
type Session struct {
	cfg Config

	mu      sync.Mutex
	changed chan struct{} // closed and replaced on every change
	m       *model.Model
	agent   *agentkit.Agent
	cur     *agentkit.Client // the agent's connection as of the last login or reconnect
	sent    map[string]agentkit.Sent
	out     bool // logout sent
	nping   uint64

	rec recorder
}

// NewSessionID makes a play-session id (D64).
func NewSessionID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "tui-" + hex.EncodeToString(b[:])
}

// NewSession returns a Session; Login connects it.
func NewSession(cfg Config) *Session {
	if cfg.Session == "" {
		cfg.Session = NewSessionID()
	}
	if cfg.Model == nil {
		cfg.Model = model.New()
	}
	s := &Session{cfg: cfg, m: cfg.Model, changed: make(chan struct{}), sent: map[string]agentkit.Sent{}}
	if cfg.Record != nil || cfg.Log != nil {
		s.rec.start(cfg.Record, cfg.Log)
	}
	return s
}

// SessionID is the id of this play session.
func (s *Session) SessionID() string { return s.cfg.Session }

// View runs f on the model with the session locked.
func (s *Session) View(f func(*model.Model)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s.m)
}

// Changed returns a channel that is closed at the next change to the model or
// the connection. Take it before reading the model and wait on it after, and
// no change is missed.
func (s *Session) Changed() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.changed
}

// onMessage is the toolkit's hook: it sees every message of every connection,
// in order, before the toolkit acts on it. It records the message and folds it
// into the model.
func (s *Session) onMessage(_ []byte, msg *pb.ServerMessage) {
	var line []byte
	if msg.GetTickFrame() != nil && s.rec.recording() {
		// The toolkit numbers the events of a frame in place once this returns, so
		// the line is made now, here and not under the lock or on the recorder's
		// goroutine, which would read the message while it is written to.
		line, _ = record.Encode(msg)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if msg == nil {
		s.m.Errors = append(s.m.Errors, &pb.Error{Code: pb.ErrorCode_ERROR_CODE_BAD_MESSAGE, Message: "the server sent something that is not a protocol message"})
	} else {
		if line != nil {
			s.rec.submitLine(line)
		} else {
			s.rec.submit(msg) // never blocks: the writing is the recorder's, off this lock
		}
		s.m.Apply(msg)
	}
	s.wake()
}

// Hear takes msg as if the server had sent it: the seam for a test of a message
// the test server cannot be made to send (a game master's announcement).
func (s *Session) Hear(msg *pb.ServerMessage) { s.onMessage(nil, msg) }

// watch wakes the waiters when c's connection ends.
func (s *Session) watch(c *agentkit.Client) {
	go func() {
		_, _ = c.Ended(context.Background())
		s.mu.Lock()
		s.wake()
		s.mu.Unlock()
	}()
}

// Login connects, presents the credential, and selects the character; it
// returns when the first snapshot has arrived.
func (s *Session) Login(ctx context.Context) error {
	a, _, err := agentkit.Login(ctx, agentkit.Config{
		URL: s.cfg.URL, TLS: s.cfg.TLS, Credential: s.cfg.Credential, Character: s.cfg.Character,
		Session: s.cfg.Session, OnMessage: s.onMessage, Features: s.cfg.features(),
	})
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.agent, s.cur = a, a.Client()
	s.mu.Unlock()
	s.watch(a.Client())
	return nil
}

// Connect dials and presents the credential without choosing a character: the
// connection a player makes a new character on (CreateCharacter, then Select).
func (s *Session) Connect(ctx context.Context) error {
	a, err := agentkit.Connect(ctx, agentkit.Config{
		URL: s.cfg.URL, TLS: s.cfg.TLS, Credential: s.cfg.Credential, Character: s.cfg.Character,
		Session: s.cfg.Session, OnMessage: s.onMessage, Features: s.cfg.features(),
	})
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.agent, s.cur = a, a.Client()
	s.mu.Unlock()
	s.watch(a.Client())
	return nil
}

// ListCharacters asks the server for the account's characters (id, name, class,
// level, zone, last played, online), the one played most recently first. It
// works after Connect, before a character is chosen.
func (s *Session) ListCharacters(ctx context.Context) ([]*pb.CharacterEntry, error) {
	a := s.agentOrNil()
	if a == nil {
		return nil, errors.New("not connected")
	}
	return a.ListCharacters(ctx)
}

// MySessions lists the account's own sessions from the server's session log
// (D64); the server answers for the account of the credential only. A refusal
// is an error carrying its reason.
func (s *Session) MySessions(ctx context.Context) ([]*pb.SessionSummary, error) {
	a := s.agentOrNil()
	if a == nil {
		return nil, errors.New("not connected")
	}
	rep, err := a.MySessions(ctx, 0)
	if err != nil {
		return nil, err
	}
	if r := rep.GetRefused(); r != "" {
		return nil, fmt.Errorf("the server refused: %s", model.Words(r))
	}
	return rep.GetSessions(), nil
}

// SessionLog fetches the server's log of one of the account's own sessions.
func (s *Session) SessionLog(ctx context.Context, session string) (*pb.SessionsReply, error) {
	a := s.agentOrNil()
	if a == nil {
		return nil, errors.New("not connected")
	}
	rep, err := a.SessionLog(ctx, session, 0)
	if err != nil {
		return nil, err
	}
	if r := rep.GetRefused(); r != "" {
		return nil, fmt.Errorf("the server refused: %s", model.Words(r))
	}
	return rep, nil
}

func (s *Session) agentOrNil() *agentkit.Agent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.agent
}

// Refused is a create_character the server refused: Reason is its code
// ("bad_name", "name_taken", "unknown_class", "too_many_characters").
type Refused struct{ Reason string }

func (r Refused) Error() string { return "refused: " + r.Reason }

// CreateCharacter makes a character of the account and returns its id, which
// Select takes (D46). A refusal is a Refused.
func (s *Session) CreateCharacter(ctx context.Context, name, class string) (string, error) {
	s.mu.Lock()
	a, before := s.agent, len(s.m.Errors)
	s.mu.Unlock()
	if a == nil {
		return "", errors.New("not connected")
	}
	id, err := a.CreateCharacter(ctx, name, class)
	if err != nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		for _, e := range s.m.Errors[min(before, len(s.m.Errors)):] {
			if e.GetCode() == pb.ErrorCode_ERROR_CODE_CREATE_REFUSED {
				return "", Refused{e.GetMessage()}
			}
		}
		return "", err
	}
	return id, nil
}

// Select chooses the character to play; the first snapshot is in the model
// when it returns.
func (s *Session) Select(ctx context.Context, id string) error {
	s.mu.Lock()
	a := s.agent
	s.mu.Unlock()
	if a == nil {
		return errors.New("not connected")
	}
	_, err := a.Select(ctx, id)
	return err
}

// client is the connection now in use, nil before a login. It is the one the
// session last saw up: the toolkit holds its own lock for the whole of a
// reconnect, while the pong it waits for comes through onMessage, which needs
// the session's, so the session must never wait on the agent under its lock
// (nor at all while a reconnect runs: a command sent then finds the old
// connection closed, as it should).
func (s *Session) client() *agentkit.Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cur
}

// Reconnect brings a dropped connection back: the toolkit dials, presents the
// credential and asks to resume after the last event it applied (D39); then
// the intents with no result yet are sent again, under their own ids.
func (s *Session) Reconnect(ctx context.Context) error {
	s.mu.Lock()
	a := s.agent
	s.mu.Unlock()
	if a == nil {
		return errors.New("not connected")
	}
	if err := a.Reconnect(ctx); err != nil {
		return err
	}
	nc := a.Client()
	s.mu.Lock()
	s.cur = nc
	s.mu.Unlock()
	s.watch(nc)
	s.mu.Lock()
	var pending []agentkit.Sent
	for _, id := range s.m.PendingIntents() {
		if in, ok := s.sent[id]; ok {
			pending = append(pending, in)
		}
	}
	s.mu.Unlock()
	for _, in := range pending {
		if err := nc.Resend(ctx, in); err != nil {
			return err
		}
	}
	return nil
}

// wake tells every waiter the model changed. Callers hold s.mu.
func (s *Session) wake() {
	close(s.changed)
	s.changed = make(chan struct{})
}

// wait blocks until pred (called with the session locked) holds, the
// connection ends, or ctx is done; done then runs with the session locked and
// its error is the result.
func (s *Session) wait(ctx context.Context, what string, pred func() bool, done func() error) error {
	for {
		closed, cerr := s.Closed()
		s.mu.Lock()
		ok, ch := pred(), s.changed
		var derr error
		if ok && done != nil {
			derr = done()
		}
		s.mu.Unlock()
		switch {
		case ok:
			return derr
		case closed:
			return fmt.Errorf("waiting for %s: the connection ended: %v", what, cerr)
		}
		select {
		case <-ch:
		case <-ctx.Done():
			return fmt.Errorf("waiting for %s: %w", what, ctx.Err())
		}
	}
}

// Drop ends the connection without a goodbye, as a network failure would; the
// session can be brought back with Reconnect.
func (s *Session) Drop() {
	if c := s.client(); c != nil {
		c.Drop()
	}
}

// Close ends the connection politely.
func (s *Session) Close() {
	if c := s.client(); c != nil {
		c.Close()
	}
	s.mu.Lock()
	s.rec.stop()
	s.mu.Unlock()
	s.rec.wait(recorderWait)
}

// Closed reports whether the connection has ended, and why.
func (s *Session) Closed() (bool, error) {
	if c := s.client(); c != nil {
		return c.Closed()
	}
	return false, nil
}

// LogoutAndWait logs out and returns once the server has ended the connection:
// the character is out of the world. A driver that has its own character log
// out uses it, so that nothing of it is left when the call returns.
func (s *Session) LogoutAndWait(ctx context.Context) error {
	c := s.client()
	if c == nil {
		return errors.New("not connected")
	}
	s.mu.Lock()
	s.out = true
	s.mu.Unlock()
	if err := c.Logout(ctx); err != nil {
		return err
	}
	_, err := c.Ended(ctx)
	return err
}
