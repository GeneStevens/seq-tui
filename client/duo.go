package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/GeneStevens/seq-tui/model"
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
	// In a turn-based instance (D55) the character is played by a plan each
	// round: the one the stance makes of the replica. own says it goes as the
	// character's own plan (a human session may not send a stand-in); played is
	// the last round it was sent for, plan what it held and planRound the round
	// that plan was for.
	own       bool
	played    uint64
	plan      []string
	planRound uint64
	follow    agentkit.Stance // the stance's driver
	assist    agentkit.Stance // the driver an attack order uses
	hold      agentkit.Stance // the driver a stop order uses
	quit      chan struct{}
}

// NewDuo returns the second character cfg.Character of the account, a follower
// of leader (the character this client plays), out of play until Summon.
//
// With cfg.Agent set the character logs in with that agent credential, and in a
// turn-based instance its stance sends a stand-in plan each round (D10, D55);
// otherwise it logs in as the player and sends its plans as its own.
func NewDuo(cfg Config, leader string) *Duo {
	cfg.Session, cfg.Record, cfg.Model = "", nil, nil
	own := cfg.Agent == ""
	if !own {
		cfg.Credential = cfg.Agent
	}
	return &Duo{cfg: cfg, leader: leader, stance: "follow", own: own,
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
	d.played, d.plan, d.planRound = 0, nil, 0
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

// UseDoor sends the second character through the same door as the player, if it
// is in play. The server decides (it must be within reach of the door); the
// answer is "" when the door took it, else the reason in words, and "" too when
// the character is out of play.
func (d *Duo) UseDoor(ctx context.Context, door string) string {
	s := d.Session()
	if s == nil {
		return ""
	}
	id, err := s.UseDoor(ctx, door)
	if err != nil {
		return err.Error()
	}
	st, err := s.Result(ctx, id)
	switch {
	case err != nil:
		return err.Error()
	case st.State == model.IntentDenied:
		return model.Words(st.Reason)
	}
	return ""
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

// Plan is what the stance sent for the round it last played in a turn-based
// instance, as words ("assist: target X, attack on"), and the round; empty and 0
// if it has played none.
func (d *Duo) Plan() ([]string, uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.plan, d.planRound
}

// Step sends what the stance (or the order) says the character does now, if
// the game clock has moved duoStepMs since the last time. In a turn-based
// instance it sends the stance's plan once for each round that opens (D55).
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
	var round uint64
	var steps []agentkit.Step
	var words []string
	s.Replica(func(r *agentkit.Replica) {
		d.mu.Lock()
		defer d.mu.Unlock()
		if r.Round != nil || r.Instance.GetTimeMode() == "turn_based" {
			// Game time is paused between rounds, so the plan is not held to the
			// clock: one plan for each round, as it opens.
			if r.Round != nil && r.Round.Open && r.Round.Number > d.played {
				round, steps = r.Round.Number, agentkit.PlanSteps(d.next(r))
				words = stepWords(steps, func(id string) string {
					if e := r.Entities[id]; e != nil && e.GetName() != "" {
						return e.GetName()
					}
					return id
				})
			}
			return
		}
		if r.GameTimeMs < d.last+duoStepMs && r.GameTimeMs >= d.last {
			return
		}
		d.last = r.GameTimeMs
		ins = d.next(r)
	})
	if round > 0 {
		d.playRound(ctx, s, round, steps, words)
		return
	}
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

// playRound sends the stance's plan for the round that is open. The server's
// answer is kept with the plan: a plan sent after the round closed is the next
// round's to make, and any other refusal is shown as the reason it gave.
func (d *Duo) playRound(ctx context.Context, s *Session, round uint64, steps []agentkit.Step, words []string) {
	in := agentkit.StandIn(steps...)
	if d.own {
		in = agentkit.TurnSubmit(steps...)
	}
	id, err := s.Send(ctx, "duo plan", in)
	if err != nil {
		return
	}
	st, err := s.Result(ctx, id)
	if err != nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.played = round
	if st.State == model.IntentDenied {
		if st.Reason == "round_closed" {
			return
		}
		words = []string{"refused: " + model.Words(st.Reason)}
	}
	d.plan, d.planRound = words, round
}

// stepWords lists a plan's steps in words; name gives an entity's name.
func stepWords(steps []agentkit.Step, name func(string) string) []string {
	var out []string
	for _, st := range steps {
		var p pb.PlanStep
		st(&p)
		switch b := p.GetBody().(type) {
		case *pb.PlanStep_Move:
			out = append(out, "move")
		case *pb.PlanStep_Stop:
			out = append(out, "stop")
		case *pb.PlanStep_Target:
			out = append(out, "target "+name(b.Target.GetTarget()))
		case *pb.PlanStep_Attack:
			if b.Attack.GetOn() {
				out = append(out, "attack")
			} else {
				out = append(out, "stop attacking")
			}
		case *pb.PlanStep_UseAbility:
			out = append(out, fmt.Sprintf("ability %d", b.UseAbility.GetSlot()))
		}
	}
	if len(out) == 0 {
		out = []string{"wait"}
	}
	return out
}
