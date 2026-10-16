package main

import (
	"fmt"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq/agentkit"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// toGate walks the test zone's character to the instance entrance, round the
// wall at x = 40 m.
func (c *capture) toGate() error {
	for _, p := range [][2]int64{{30000, 85000}, {99000, 88000}} {
		if err := c.walk(p[0], p[1]); err != nil {
			return err
		}
	}
	return nil
}

// inSpace waits for a snapshot and map of a space other than from.
func (c *capture) inOther(from string) error {
	return c.until("the other side", func(m *model.Model) bool {
		return m.Space != from && m.Ready() && m.Map != nil && m.Map.GetSpace() == m.Space
	})
}

func (c *capture) space() (s string) {
	c.s.View(func(m *model.Model) { s = m.Space })
	return
}

// openAndEnter opens preset from the menu, waits for instance_opened, and goes in.
func (c *capture) openAndEnter(preset string) error {
	world := c.space()
	if err := c.toGate(); err != nil {
		return err
	}
	if err := c.do("open "+preset, func() (string, error) { return c.s.OpenInstance(c.ctx, preset) }); err != nil {
		return err
	}
	if err := c.until("instance_opened", func(m *model.Model) bool { return m.Opened != nil }); err != nil {
		return err
	}
	time.Sleep(time.Second)
	if err := c.do("enter", func() (string, error) { return c.s.UseDoor(c.ctx, "gate-north") }); err != nil {
		return err
	}
	return c.inOther(world)
}

// solo plays solo-trial: open, enter, kill and loot the scuttler until the
// instance is cleared, and be handed back.
func (c *capture) solo() error {
	if err := c.openAndEnter("solo-trial"); err != nil {
		return err
	}
	time.Sleep(time.Second)
	var mob string
	c.s.View(func(m *model.Model) {
		for _, id := range m.EntityIDs() {
			mob = id
		}
	})
	if mob == "" {
		return fmt.Errorf("no mob in the instance")
	}
	if err := c.do("target", func() (string, error) { return c.s.Target(c.ctx, mob) }); err != nil {
		return err
	}
	if err := c.closeIn(mob); err != nil {
		return err
	}
	if err := c.do("attack", func() (string, error) { return c.s.Attack(c.ctx, true) }); err != nil {
		return err
	}
	if err := c.until("a drop", func(m *model.Model) bool { return len(m.Drops) > 0 }); err != nil {
		return err
	}
	var drop string
	c.s.View(func(m *model.Model) { drop = m.DropIDs()[0] })
	if err := c.do("loot", func() (string, error) { return c.s.Loot(c.ctx, drop) }); err != nil {
		return err
	}
	if err := c.until("the hand-back", func(m *model.Model) bool {
		return m.Ended != nil && !m.InInstance() && m.Map != nil && m.Map.GetSpace() == m.Space
	}); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return nil
}

// closeIn walks toward a mob until it is within melee reach.
func (c *capture) closeIn(mob string) error {
	for {
		var x, y int64
		var near bool
		c.s.View(func(m *model.Model) {
			e := m.Entities[mob]
			if e == nil {
				return
			}
			x, y = e.GetPos().GetX(), e.GetPos().GetY()
			dx, dy := x-m.Self.Pos.GetX(), y-m.Self.Pos.GetY()
			near = dx*dx+dy*dy < 2000*2000
		})
		if near {
			return nil
		}
		if _, err := c.s.Move(c.ctx, x, y); err != nil {
			return err
		}
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// round is the open round that is newer than after, or nil.
func (c *capture) openRound(after uint64) func(*model.Model) bool {
	return func(m *model.Model) bool { return m.Round != nil && m.Round.Open && m.Round.Number > after }
}

// plan builds this round's plan the way a player would: go to the mob, target
// it, attack; loot when it is dead.
func (c *capture) plan() []agentkit.Step {
	var steps []agentkit.Step
	c.s.View(func(m *model.Model) {
		if ids := m.DropIDs(); len(ids) > 0 {
			steps = append(steps, agentkit.StepLoot(ids[0]))
			return
		}
		for _, id := range m.EntityIDs() {
			e := m.Entities[id]
			if e.GetKind() != pb.EntityKind_ENTITY_KIND_MOB || e.GetDead() {
				continue
			}
			dx, dy := e.GetPos().GetX()-m.Self.Pos.GetX(), e.GetPos().GetY()-m.Self.Pos.GetY()
			if dx*dx+dy*dy > 2000*2000 {
				steps = append(steps, agentkit.StepMove(e.GetPos().GetX(), e.GetPos().GetY()))
			}
			steps = append(steps, agentkit.StepTarget(id), agentkit.StepAttack(true))
			return
		}
	})
	return steps
}

func (c *capture) submit(what string, steps ...agentkit.Step) error {
	return c.do(what, func() (string, error) { return c.s.SubmitPlan(c.ctx, steps...) })
}

func (c *capture) resolved(n uint64) error {
	return c.until(fmt.Sprintf("round %d to run", n), func(m *model.Model) bool { return m.Round != nil && m.Round.Resolved >= n })
}

// turns plays quick-turns: plans with receipts, a missed deadline (the safe
// default), a connection that drops and resumes, a fresh login into the open
// round (as after a hibernation), until the instance is cleared and the
// character handed back. The solo character's round runs as soon as its plan
// is in, so each reconnect is made while the round waits for a plan.
func (c *capture) turns() error {
	if err := c.openAndEnter("quick-turns"); err != nil {
		return err
	}
	var last uint64
	for round := 1; ; round++ {
		if err := c.until("an open round", func(m *model.Model) bool {
			return (m.Round != nil && m.Round.Open && m.Round.Number > last) || (m.Ended != nil && !m.InInstance())
		}); err != nil {
			return err
		}
		if c.cleared() {
			break
		}
		var n uint64
		c.s.View(func(m *model.Model) { n = m.Round.Number })
		last = n
		fmt.Printf("round %d (turn %d)\n", n, round)
		switch round {
		case 2: // the connection drops and resumes inside the round
			time.Sleep(time.Second)
			c.s.Drop()
			time.Sleep(2 * time.Second)
			if err := c.s.Reconnect(c.ctx); err != nil {
				return err
			}
			time.Sleep(time.Second)
		case 3: // a fresh login: the snapshot and the round again
			time.Sleep(time.Second)
			c.s.Close()
			time.Sleep(3 * time.Second)
			if err := c.login(); err != nil {
				return err
			}
			if err := c.until("the round again", func(m *model.Model) bool { return m.Ready() && m.Round != nil && m.Round.Open && m.Round.Number == n }); err != nil {
				return err
			}
			time.Sleep(time.Second)
		case 4: // miss the deadline: nothing is submitted
			if err := c.resolved(n); err != nil {
				return err
			}
			continue
		}
		if err := c.submit("plan", c.plan()...); err != nil {
			return err
		}
		if err := c.resolved(n); err != nil {
			return err
		}
	}
	return c.until("the hand-back", func(m *model.Model) bool {
		return m.Ended != nil && !m.InInstance() && m.Map != nil && m.Map.GetSpace() == m.Space
	})
}

func (c *capture) cleared() (b bool) {
	c.s.View(func(m *model.Model) { b = m.Ended != nil })
	return
}
