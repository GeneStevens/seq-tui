package client

import (
	"context"
	"fmt"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Commands: one function for each intent the protocol has. Each sends the
// intent and returns its id at once; Result waits for the server's answer and
// matches it by that id. The server decides the outcome; the client does not
// check a rule first (doc 07 §1).

// Move walks toward a point (millimetres).
func (s *Session) Move(ctx context.Context, x, y int64) (string, error) {
	return s.intent(ctx, "move", func(i *pb.Intent) { i.Body = &pb.Intent_Move{Move: &pb.Move{Destination: &pb.Pos{X: x, Y: y}}} })
}

// Stop cancels a move in progress.
func (s *Session) Stop(ctx context.Context) (string, error) {
	return s.intent(ctx, "stop", func(i *pb.Intent) { i.Body = &pb.Intent_Stop{Stop: &pb.Stop{}} })
}

// Target selects a target; "" clears it.
func (s *Session) Target(ctx context.Context, id string) (string, error) {
	return s.intent(ctx, "target", func(i *pb.Intent) { i.Body = &pb.Intent_Target{Target: &pb.SetTarget{Target: id}} })
}

// Attack turns auto-attack on or off.
func (s *Session) Attack(ctx context.Context, on bool) (string, error) {
	return s.intent(ctx, "attack", func(i *pb.Intent) { i.Body = &pb.Intent_Attack{Attack: &pb.Attack{On: on}} })
}

// Loot takes the item from a drop.
func (s *Session) Loot(ctx context.Context, drop string) (string, error) {
	return s.intent(ctx, "loot", func(i *pb.Intent) { i.Body = &pb.Intent_Loot{Loot: &pb.Loot{Drop: drop}} })
}

// UseDoor crosses a zone line or enters or leaves an instance (D39).
func (s *Session) UseDoor(ctx context.Context, door string) (string, error) {
	return s.intent(ctx, "use_door", func(i *pb.Intent) { i.Body = &pb.Intent_UseDoor{UseDoor: &pb.UseDoor{Door: door}} })
}

// Logout leaves the world at once (D39); the server ends the connection when
// the logout is journaled.
func (s *Session) Logout(ctx context.Context) error {
	s.mu.Lock()
	s.out = true
	s.mu.Unlock()
	return s.send(ctx, &pb.ClientMessage{Body: &pb.ClientMessage_Logout{Logout: &pb.Logout{}}})
}

// LoggedOut reports whether Logout was sent.
func (s *Session) LoggedOut() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.out
}

func (s *Session) intent(ctx context.Context, kind string, fill func(*pb.Intent)) (string, error) {
	s.mu.Lock()
	s.seq++
	s.idn++
	in := &pb.Intent{IntentId: fmt.Sprintf("tui-%d", s.idn), Seq: s.seq}
	fill(in)
	s.sent[in.IntentId] = in
	s.m.IntentSent(in.IntentId, in.Seq, kind)
	s.mu.Unlock()
	return in.IntentId, s.send(ctx, &pb.ClientMessage{Body: &pb.ClientMessage_Intent{Intent: in}})
}

// Result waits for the intent_result of the intent with id and returns its status.
func (s *Session) Result(ctx context.Context, id string) (model.IntentStatus, error) {
	var st model.IntentStatus
	err := s.wait(ctx, "the result of "+id, func() bool {
		i, ok := s.m.Intent(id)
		return ok && i.State != model.IntentPending
	}, func() error {
		st, _ = s.m.Intent(id)
		return nil
	})
	return st, err
}

// Ping asks the server for a pong, so a quiet connection is seen to be alive.
func (s *Session) Ping(ctx context.Context) error {
	s.mu.Lock()
	s.idn++
	n := uint64(s.idn)
	s.mu.Unlock()
	return s.send(ctx, &pb.ClientMessage{Body: &pb.ClientMessage_Ping{Ping: &pb.Ping{Nonce: n}}})
}
