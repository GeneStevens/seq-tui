package client

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func event(n uint64) *pb.ServerMessage {
	return &pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: n, GameTimeMs: int64(n) * 100,
		Body: &pb.Event_Combat{Combat: &pb.Combat{Kind: pb.CombatKind_COMBAT_KIND_HIT, Attacker: "mob", Target: "me", Amount: 3}}}}}
}

// slowWriter is a disk that waits on a gate.
type slowWriter struct {
	mu   sync.Mutex
	gate chan struct{}
	buf  bytes.Buffer
}

func (w *slowWriter) Write(p []byte) (int, error) {
	<-w.gate
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *slowWriter) String() string { w.mu.Lock(); defer w.mu.Unlock(); return w.buf.String() }

// Every message is in both recordings, once and in order, after a clean close.
func TestRecordingIsCompleteAfterClose(t *testing.T) {
	var a, b bytes.Buffer
	s := NewSession(Config{Record: record.NewWriter(&a), Log: record.NewWriter(&b)})
	const n = 3000
	for i := range n {
		s.onMessage(nil, event(uint64(i+1)))
	}
	s.Close()
	for name, buf := range map[string]*bytes.Buffer{"record": &a, "log": &b} {
		msgs, err := record.Read(bytes.NewReader(buf.Bytes()))
		if err != nil || len(msgs) != n {
			t.Fatalf("%s: %d messages, %v; want %d", name, len(msgs), err, n)
		}
		for i, m := range msgs {
			if m.GetEvent().GetSeq() != uint64(i+1) {
				t.Fatalf("%s: message %d is event %d", name, i, m.GetEvent().GetSeq())
			}
		}
	}
	s.Close()                    // again: harmless
	s.onMessage(nil, event(n+1)) // after close: not recorded, not a panic
}

// A disk that has stopped never holds the session: onMessage returns, the model
// goes on, and the drop is counted in the recording.
func TestAFullQueueNeverBlocksTheSession(t *testing.T) {
	w := &slowWriter{gate: make(chan struct{})}
	s := NewSession(Config{Record: record.NewWriter(w), Model: model.New()})
	done := make(chan struct{})
	go func() {
		for i := range 3 * recorderQueue {
			s.onMessage(nil, event(uint64(i+1)))
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("onMessage blocked on a stalled recording")
	}
	s.View(func(m *model.Model) {
		if m.Last() == 0 {
			t.Error("the model did not take the messages")
		}
	})
	if s.rec.dropped.Load() == 0 {
		t.Error("nothing was dropped from a stalled queue")
	}
	close(w.gate)
	s.Close()
	if !strings.Contains(w.String(), "# dropped ") {
		t.Error("the recording does not say messages were dropped")
	}
	if _, err := record.Read(strings.NewReader(w.String())); err != nil {
		t.Errorf("a recording with a drop line does not read: %v", err)
	}
}

// A frame is one message and so one change: the screen is told once, with every
// event of the frame already in the model, where messages of old told it once
// for each (D89).
func TestAFrameWakesTheScreenOnceWithAllItsEvents(t *testing.T) {
	s := NewSession(Config{Model: model.New()})
	s.onMessage(nil, &pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Seq: 4, Space: "z", Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Pos: &pb.Pos{}}}}}})
	before := s.Changed()
	s.onMessage(nil, &pb.ServerMessage{Body: &pb.ServerMessage_TickFrame{TickFrame: &pb.TickFrame{BaseSeq: 5, Events: []*pb.Event{event(0).GetEvent(), event(0).GetEvent(), event(0).GetEvent()}}}})
	select {
	case <-before:
	default:
		t.Fatal("the frame did not wake the screen")
	}
	select {
	case <-s.Changed():
		t.Error("the frame woke the screen more than once")
	default:
	}
	s.View(func(m *model.Model) {
		if m.Last() != 7 {
			t.Errorf("the model is at event %d when the screen is woken, want 7", m.Last())
		}
	})
}
