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
	// Model is the model to feed; nil makes a new one.
	Model *model.Model
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
	sent    map[string]agentkit.Sent
	out     bool // logout sent
	nping   uint64
}

// NewSession returns a Session; Login connects it.
func NewSession(cfg Config) *Session {
	if cfg.Session == "" {
		var b [8]byte
		_, _ = rand.Read(b[:])
		cfg.Session = "tui-" + hex.EncodeToString(b[:])
	}
	if cfg.Model == nil {
		cfg.Model = model.New()
	}
	return &Session{cfg: cfg, m: cfg.Model, changed: make(chan struct{}), sent: map[string]agentkit.Sent{}}
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if msg == nil {
		s.m.Errors = append(s.m.Errors, &pb.Error{Code: pb.ErrorCode_ERROR_CODE_BAD_MESSAGE, Message: "the server sent something that is not a protocol message"})
	} else {
		if s.cfg.Record != nil {
			_ = s.cfg.Record.Write(msg)
		}
		s.m.Apply(msg)
	}
	s.wake()
}

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
		Session: s.cfg.Session, OnMessage: s.onMessage,
	})
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.agent = a
	s.mu.Unlock()
	s.watch(a.Client())
	return nil
}

func (s *Session) client() *agentkit.Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.agent == nil {
		return nil
	}
	return s.agent.Client()
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
	s.watch(a.Client())
	s.mu.Lock()
	var pending []agentkit.Sent
	for _, id := range s.m.PendingIntents() {
		if in, ok := s.sent[id]; ok {
			pending = append(pending, in)
		}
	}
	s.mu.Unlock()
	for _, in := range pending {
		if err := a.Client().Resend(ctx, in); err != nil {
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
}

// Closed reports whether the connection has ended, and why.
func (s *Session) Closed() (bool, error) {
	if c := s.client(); c != nil {
		return c.Closed()
	}
	return false, nil
}
