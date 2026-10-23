package client_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// frame is a tick frame of n ticks' events from base, as the server sends it.
func (f *fake) frame(base uint64, n int) {
	fr := &pb.TickFrame{BaseSeq: base}
	for i := range n {
		fr.Events = append(fr.Events, &pb.Event{GameTimeMs: int64(base) + int64(i), Body: &pb.Event_RulesTick{RulesTick: &pb.RulesTick{}}})
	}
	b, _ := proto.Marshal(&pb.ServerMessage{Body: &pb.ServerMessage_TickFrame{TickFrame: fr}})
	_ = f.conn.Write(context.Background(), websocket.MessageBinary, b)
}

// The session asks for tick frames in hello and, dropped and resumed, applies
// each event once: the resume names the last event of the last frame, the
// server's replay may begin a frame early, and the recording of both
// connections replays to the same model (T-frames, D89).
func TestFramedDropResumeIsExactlyOnce(t *testing.T) {
	srv := newServer(t)
	var buf bytes.Buffer
	s := newSession(srv, record.NewWriter(&buf))
	fc := make(chan *fake, 1)
	go func() {
		f := srv.accept(t)
		fc <- f
		if h := f.next(t).GetHello(); len(h.GetFeatures()) != 1 || h.GetFeatures()[0] != "tick_frames" {
			t.Errorf("hello asks for %v, want tick_frames", h.GetFeatures())
		}
		f.say(&pb.Welcome{ProtocolVersion: 1, Features: []string{"tick_frames"}})
		f.next(t) // select
		f.say(snap(10))
	}()
	if err := s.Login(ctx(t)); err != nil {
		t.Fatal(err)
	}
	f1 := <-fc
	f1.frame(11, 3) // events 11, 12, 13
	for range 500 {
		var last uint64
		s.View(func(m *model.Model) { last = m.Last() })
		if last == 13 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	f1.Close()
	for closed, _ := s.Closed(); !closed; closed, _ = s.Closed() {
		time.Sleep(5 * time.Millisecond)
	}
	go func() {
		f2 := srv.accept(t)
		f2.next(t) // hello
		f2.say(&pb.Welcome{ProtocolVersion: 1})
		if r := f2.next(t).GetResume(); r.GetLastSeq() != 13 {
			t.Errorf("resume after %d, want 13: the last event of the last frame", r.GetLastSeq())
		}
		f2.next(t) // the ping
		f2.say(&pb.Pong{Nonce: 1})
		f2.frame(14, 2) // events 14, 15
	}()
	if err := s.Reconnect(ctx(t)); err != nil {
		t.Fatal(err)
	}
	for range 500 {
		var last uint64
		s.View(func(m *model.Model) { last = m.Last() })
		if last == 15 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	s.View(func(m *model.Model) {
		if m.Last() != 15 || len(m.Faults) != 0 || m.Duplicates != 0 || m.Connections != 2 {
			t.Errorf("last %d faults %v dups %d connections %d", m.Last(), m.Faults, m.Duplicates, m.Connections)
		}
	})
	s.Close()
	msgs, err := record.Read(&buf)
	if err != nil {
		t.Fatal(err)
	}
	rm := model.New()
	for _, m := range msgs {
		rm.Apply(m)
	}
	if rm.Last() != 15 || len(rm.Faults) != 0 {
		t.Errorf("the recording replays to event %d, faults %v; want 15", rm.Last(), rm.Faults)
	}
}
