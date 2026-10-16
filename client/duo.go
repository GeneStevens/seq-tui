package client

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/GeneStevens/seq/agentkit"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// A duo on built-in stances (SEQ-049, D42): the account's second character is
// played from this client by the toolkit's stance driver, so a duo needs no AI
// agent. The character is the player's own: it has its own connection and its
// commands count as directed play while the player is present (D31).

const (
	// duoStepMs is how much game time passes between two things the driver
	// sends: about a tick of the player's own stance, no faster.
	duoStepMs = 500
	// duoNearMm is how close to a spot a "come here" order counts as done.
	duoNearMm = 2000
)

// Order is a direct command to the second character; it overrides the stance
// until it is done or replaced, as a companion's does.
type Order struct {
	Kind   string // "attack", "move" or "stop"; "" is none
	Target string
	Dest   *pb.Pos
}

// Duo is the second character: a session of its own, a stance and an order.
type Duo struct {
	cfg    Config
	leader string

	mu     sync.Mutex
	sess   *Session
	stance string
	order  Order
	sent   bool // the move order has been sent
	last   int64
	follow agentkit.Stance // the stance's driver
	assist agentkit.Stance // the driver an attack order uses
	hold   agentkit.Stance // the driver a stop order uses
	quit   chan struct{}
}

// NewDuo returns the second character cfg.Character of the account, a follower
// of leader (the character this client plays), out of play until Summon.
func NewDuo(cfg Config, leader string) *Duo {
	cfg.Session, cfg.Record, cfg.Model = "", nil, nil
	return &Duo{cfg: cfg, leader: leader, stance: "follow",
		follow: agentkit.Stance{Leader: leader}, assist: agentkit.Stance{Name: "assist", Leader: leader}, hold: agentkit.Stance{Name: "passive", Leader: leader}}
}

// Character is the id of the second character.
func (d *Duo) Character() string { return d.cfg.Character }

// Active says the second character is in play.
func (d *Duo) Active() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sess != nil
}

// Summon puts the second character in play: it logs in on a connection of its
// own. Its stance is the one it had.
func (d *Duo) Summon(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.sess != nil {
		return errors.New("already in play")
	}
	s := NewSession(d.cfg)
	if err := s.Login(ctx); err != nil {
		return err
	}
	d.sess, d.last, d.sent, d.order = s, 0, false, Order{}
	return nil
}

// Dismiss logs the second character out.
func (d *Duo) Dismiss(ctx context.Context) error {
	d.mu.Lock()
	s := d.sess
	d.sess, d.order = nil, Order{}
	d.mu.Unlock()
	if s == nil {
		return errors.New("not in play")
	}
	return s.LogoutAndWait(ctx)
}

// Close ends the second character's connection, leaving it in the world as
// quitting the screen does for the first.
func (d *Duo) Close() {
	d.mu.Lock()
	s := d.sess
	d.sess = nil
	d.mu.Unlock()
	if s != nil {
		s.Close()
	}
}

// Stance is the second character's stance.
func (d *Duo) Stance() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.stance
}

// SetStance sets the stance and drops any standing order, as the server does for
// a companion.
func (d *Duo) SetStance(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stance, d.order = name, Order{}
}

// Order is the standing order.
func (d *Duo) Order() Order {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.order
}

// SetOrder gives an order; an empty one drops the standing order.
func (d *Duo) SetOrder(o Order) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.order, d.sent = o, false
}

// Session is the second character's session, nil out of play.
func (d *Duo) Session() *Session {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sess
}

// Run drives the character until ctx ends: about every tenth of a second it
// sends what the stance says, once the game clock has moved enough.
func (d *Duo) Run(ctx context.Context) {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			d.Step(ctx)
		}
	}
}

// Step sends what the stance (or the order) says the character does now, if
// the game clock has moved duoStepMs since the last time. It does nothing in a
// turn-based instance: the stance cannot plan.
func (d *Duo) Step(ctx context.Context) {
	d.mu.Lock()
	s := d.sess
	d.mu.Unlock()
	if s == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var ins []agentkit.Intent
	s.Replica(func(r *agentkit.Replica) {
		d.mu.Lock()
		defer d.mu.Unlock()
		if r.GameTimeMs < d.last+duoStepMs && r.GameTimeMs >= d.last {
			return
		}
		d.last = r.GameTimeMs
		if r.Round != nil || r.Instance.GetTimeMode() == "turn_based" {
			return
		}
		ins = d.next(r)
	})
	// Each is answered before the next goes: the character does what the stance
	// said and no more than that before the clock moves again.
	for _, in := range ins {
		id, err := s.Send(ctx, "duo", in)
		if err != nil {
			return
		}
		if _, err := s.Result(ctx, id); err != nil {
			return
		}
	}
}

// next is what the character does now; the caller holds d.mu.
func (d *Duo) next(r *agentkit.Replica) []agentkit.Intent {
	switch o := d.order; o.Kind {
	case "attack":
		if e := r.Entities[o.Target]; e != nil && !e.GetDead() {
			// The assist driver fights what the leader targets; the order names
			// the target, so it is put there for the one call.
			old, had := r.Targets[d.leader]
			r.Targets[d.leader] = o.Target
			ins := d.assist.Next(r)
			if had {
				r.Targets[d.leader] = old
			} else {
				delete(r.Targets, d.leader)
			}
			return ins
		}
		d.order = Order{}
	case "move":
		if !d.sent {
			d.sent = true
			return []agentkit.Intent{agentkit.MoveTo(o.Dest.GetX(), o.Dest.GetY())}
		}
		if r.Dist(o.Dest) <= duoNearMm {
			d.order = Order{}
		}
		return nil
	case "stop":
		ins := d.hold.Next(r)
		if !d.sent { // whatever it was doing, it stops
			d.sent = true
			ins = append([]agentkit.Intent{agentkit.Stop()}, ins...)
		}
		return ins
	}
	d.follow.Name = d.stance
	return d.follow.Next(r)
}
