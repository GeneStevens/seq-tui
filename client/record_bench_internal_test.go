package client

import (
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func benchEvent(n uint64) *pb.ServerMessage {
	return &pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: n, GameTimeMs: int64(n) * 100,
		Body: &pb.Event_Combat{Combat: &pb.Combat{Kind: pb.CombatKind_COMBAT_KIND_HIT, Attacker: "mob", Target: "me", Amount: 3}}}}}
}

// BenchmarkRecordOne is what recording one event costs the session lock (T3):
// the time onMessage holds the lock, with the recording going to a writer that
// takes its time on each flush.
func BenchmarkRecordOne(b *testing.B) {
	s := NewSession(Config{Record: record.NewWriter(discardSlow{}), Model: model.New()})
	msgs := make([]*pb.ServerMessage, 1024)
	for i := range msgs {
		msgs[i] = benchEvent(uint64(i + 1))
	}
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		s.onMessage(nil, msgs[i%len(msgs)])
		i++
	}
	b.StopTimer()
	s.Close()
}

// discardSlow is a disk that takes 20 µs a write.
type discardSlow struct{}

func (discardSlow) Write(p []byte) (int, error) {
	time.Sleep(20 * time.Microsecond)
	return len(p), nil
}
