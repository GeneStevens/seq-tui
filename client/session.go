package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Config says how a Session connects.
type Config struct {
	Dial       Dialer
	Credential string // a human session token (doc 04 §3)
	Character  string
	// Session names this play session; a resume names it again. Empty: a
	// random one is made.
	Session string
	// Record, if set, receives every incoming message.
	Record *record.Writer
	// Model is the model to feed; nil makes a new one.
	Model *model.Model
}

// Session is one play session: a connection that feeds the model, and that can
// be brought back and resumed on the same session id.
type Session struct {
	cfg Config

	mu      sync.Mutex
	changed chan struct{} // closed and replaced on every change
	m       *model.Model
	tr      Transport
	gen     int   // the connection number, to ignore a dead pump's last words
	closed  bool  // the current connection has ended
	err     error // why
	sent    map[string]*pb.Intent
	seq     uint64
	idn     int
	out     bool // logout sent
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
	return &Session{cfg: cfg, m: cfg.Model, changed: make(chan struct{}), sent: map[string]*pb.Intent{}}
}

// SessionID is the id of this play session.
func (s *Session) SessionID() string { return s.cfg.Session }

// View runs f on the model with the session locked.
func (s *Session) View(f func(*model.Model)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s.m)
}

// Login connects, presents the credential, and selects the character; it
// returns when the first snapshot has arrived.
func (s *Session) Login(ctx context.Context) error {
	if err := s.connect(ctx); err != nil {
		return err
	}
	if err := s.send(ctx, &pb.ClientMessage{Body: &pb.ClientMessage_Select{Select: &pb.Select{Character: s.cfg.Character, Session: s.cfg.Session}}}); err != nil {
		return err
	}
	nerr := s.errCount()
	return s.wait(ctx, "the snapshot", func() bool { return s.m.Ready() || len(s.m.Errors) > nerr }, func() error {
		if !s.m.Ready() {
			return fmt.Errorf("select refused: %s", errText(s.m.Errors[len(s.m.Errors)-1]))
		}
		return nil
	})
}

// Reconnect brings a dropped connection back: it dials, presents the
// credential, asks to resume after the last event the model applied (D39), and
// sends again, under their own ids, the intents with no result yet.
func (s *Session) Reconnect(ctx context.Context) error {
	if err := s.connect(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	last, pending := s.m.Last(), s.m.PendingIntents()
	s.mu.Unlock()
	if err := s.send(ctx, &pb.ClientMessage{Body: &pb.ClientMessage_Resume{Resume: &pb.Resume{Session: s.cfg.Session, LastSeq: last}}}); err != nil {
		return err
	}
	for _, id := range pending {
		s.mu.Lock()
		in := s.sent[id]
		s.mu.Unlock()
		if in == nil {
			continue
		}
		if err := s.send(ctx, &pb.ClientMessage{Body: &pb.ClientMessage_Intent{Intent: in}}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) connect(ctx context.Context) error {
	tr, err := s.cfg.Dial(ctx)
	if err != nil {
		return describeErr("dialing", err)
	}
	s.mu.Lock()
	old := s.tr
	s.tr, s.closed, s.err = tr, false, nil
	s.gen++
	gen := s.gen
	nerr, before := len(s.m.Errors), s.m.Connections
	s.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	go s.pump(tr, gen)
	if err := s.send(ctx, &pb.ClientMessage{Body: &pb.ClientMessage_Hello{Hello: &pb.Hello{ProtocolVersion: ProtocolVersion, Credential: s.cfg.Credential}}}); err != nil {
		return err
	}
	return s.wait(ctx, "the welcome", func() bool {
		return s.m.Connections > before || len(s.m.Errors) > nerr
	}, func() error {
		if len(s.m.Errors) > nerr {
			return fmt.Errorf("hello refused: %s", errText(s.m.Errors[len(s.m.Errors)-1]))
		}
		return nil
	})
}

func (s *Session) errCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.m.Errors)
}

func errText(e *pb.Error) string { return e.GetCode().String() + ": " + e.GetMessage() }

// pump reads a connection until it ends, feeding the recorder and the model.
func (s *Session) pump(tr Transport, gen int) {
	for {
		msg, err := tr.Recv(context.Background())
		s.mu.Lock()
		if s.gen != gen {
			s.mu.Unlock()
			return
		}
		if err != nil {
			if errors.Is(err, errBadMessage) {
				s.m.Errors = append(s.m.Errors, &pb.Error{Code: pb.ErrorCode_ERROR_CODE_BAD_MESSAGE, Message: err.Error()})
				s.wake()
				s.mu.Unlock()
				continue
			}
			s.closed, s.err = true, err
			s.wake()
			s.mu.Unlock()
			return
		}
		if s.cfg.Record != nil {
			_ = s.cfg.Record.Write(msg)
		}
		s.m.Apply(msg)
		s.wake()
		s.mu.Unlock()
	}
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
		s.mu.Lock()
		ok, ch, closed, cerr := pred(), s.changed, s.closed, s.err
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

func (s *Session) send(ctx context.Context, m *pb.ClientMessage) error {
	s.mu.Lock()
	tr := s.tr
	s.mu.Unlock()
	if tr == nil {
		return errors.New("not connected")
	}
	if err := tr.Send(ctx, m); err != nil {
		return describeErr("sending", err)
	}
	return nil
}

// Drop ends the connection without a goodbye, as a network failure would; the
// session can be brought back with Reconnect.
func (s *Session) Drop() {
	s.mu.Lock()
	tr := s.tr
	s.mu.Unlock()
	if tr != nil {
		_ = tr.Close()
	}
}

// Close ends the connection politely.
func (s *Session) Close() { s.Drop() }

// Closed reports whether the connection has ended, and why.
func (s *Session) Closed() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed, s.err
}
