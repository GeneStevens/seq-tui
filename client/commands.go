package client

import (
	"context"
	"errors"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq/agentkit"
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

// Release gives up a dead character's body; it respawns at the zone's start
// point after the zone's delay (D41).
func (s *Session) Release(ctx context.Context) (string, error) {
	return s.intent(ctx, "release", func(i *pb.Intent) { i.Body = &pb.Intent_Release{Release: &pb.Release{}} })
}

// Logout leaves the world at once (D39); the server ends the connection when
// the logout is journaled.
func (s *Session) Logout(ctx context.Context) error {
	c := s.client()
	if c == nil {
		return errors.New("not connected")
	}
	s.mu.Lock()
	s.out = true
	s.mu.Unlock()
	return c.Logout(ctx)
}

// LoggedOut reports whether Logout was sent.
func (s *Session) LoggedOut() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.out
}

// intent sends a new intent through the toolkit. The toolkit numbers it and
// calls the body with the id and sequence number set, just before the write;
// the body is where the model learns of it, so a fast result finds it there.
func (s *Session) intent(ctx context.Context, kind string, fill func(*pb.Intent)) (string, error) {
	c := s.client()
	if c == nil {
		return "", errors.New("not connected")
	}
	body := func(i *pb.Intent) {
		fill(i)
		s.mu.Lock()
		if _, ok := s.m.Intent(i.IntentId); !ok { // a resend names it again
			s.m.IntentSent(i.IntentId, i.Seq, kind)
		}
		s.mu.Unlock()
	}
	sent, err := c.Send(ctx, "tui", body)
	if sent.ID != "" {
		s.mu.Lock()
		s.sent[sent.ID] = sent
		s.mu.Unlock()
	}
	return sent.ID, err
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
	c := s.client()
	if c == nil {
		return errors.New("not connected")
	}
	s.mu.Lock()
	s.nping++
	n := 100 + s.nping // 1 is the toolkit's own, on a reconnect
	s.mu.Unlock()
	return c.Ping(ctx, n)
}

// OpenInstance asks for an instance of the zone from one of the presets its
// map lists (D11). An accepted intent means the request was taken: the
// instance_opened or instance_open_failed event follows.
func (s *Session) OpenInstance(ctx context.Context, preset string) (string, error) {
	return s.intent(ctx, "instance_open", agentkit.InstanceOpen(preset))
}

// SubmitPlan sends the plan for the next round of a turn-based instance (D30).
// The intent's result is the plan's durable receipt (D37). A human session
// never marks a plan as a stand-in.
func (s *Session) SubmitPlan(ctx context.Context, steps ...agentkit.Step) (string, error) {
	return s.intent(ctx, "plan", agentkit.TurnSubmit(steps...))
}
