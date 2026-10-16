package client

import (
	"testing"

	"github.com/GeneStevens/seq/agentkit"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// What the second character is told to do, over a replica built by hand: the
// stance when there is no order, and an order over the stance.

func duoRig() (*Duo, *agentkit.Replica) {
	pos := func(x, y int64) *pb.Pos { return &pb.Pos{X: x, Y: y} }
	mob := func(id string, x int64) *pb.EntityState {
		return &pb.EntityState{Id: id, Kind: pb.EntityKind_ENTITY_KIND_MOB, Pos: pos(x, 0), Hp: 50, MaxHp: 50}
	}
	r := &agentkit.Replica{
		Self: "c2", Pos: pos(10000, 0), HP: 100, MaxHP: 100, MeleeMm: 2500,
		Entities: map[string]*pb.EntityState{
			"c1":    {Id: "c1", Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, Pos: pos(0, 0), Hp: 100, MaxHp: 100},
			"mob-1": mob("mob-1", 12000),
			"mob-2": mob("mob-2", -9000),
		},
		Targets: map[string]string{"c1": "mob-1"}, AttackedBy: map[string]string{},
		Abilities: map[string]*pb.AbilityInfo{}, Slots: map[int]string{}, ReadyAtMs: map[string]int64{},
	}
	return NewDuo(Config{Character: "c2"}, "c1"), r
}

// kinds names the intents: "target mob-2", "move", "stop", "attack".
func kinds(ins []agentkit.Intent) []string {
	var out []string
	for _, in := range ins {
		m := &pb.Intent{}
		in(m)
		switch b := m.GetBody().(type) {
		case *pb.Intent_Move:
			out = append(out, "move")
		case *pb.Intent_Stop:
			out = append(out, "stop")
		case *pb.Intent_Target:
			out = append(out, "target "+b.Target.GetTarget())
		case *pb.Intent_Attack:
			out = append(out, "attack")
		}
	}
	return out
}

func same(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDuoStanceWithNoOrder(t *testing.T) {
	d, r := duoRig()
	d.SetStance("assist")
	d.mu.Lock()
	defer d.mu.Unlock()
	if got := kinds(d.next(r)); !same(got, []string{"target mob-1", "attack"}) {
		t.Errorf("assist, the leader on mob-1: %v", got)
	}
}

func TestDuoAttackOrderNamesItsTargetAndLeavesTheLeadersAlone(t *testing.T) {
	d, r := duoRig()
	d.SetStance("follow")
	d.SetOrder(Order{Kind: "attack", Target: "mob-2"})
	d.mu.Lock()
	defer d.mu.Unlock()
	if got := kinds(d.next(r)); !same(got, []string{"target mob-2", "move"}) {
		t.Errorf("attack mob-2: %v", got)
	}
	if r.Targets["c1"] != "mob-1" {
		t.Errorf("the leader's target was left at %q", r.Targets["c1"])
	}
	// The target gone: the order is over and the stance goes on.
	r.Entities["mob-2"].Dead = true
	d.next(r)
	if d.order.Kind != "" {
		t.Errorf("the order stands with its target dead: %+v", d.order)
	}
}

func TestDuoMoveOrderIsSentOnceAndEndsOnArrival(t *testing.T) {
	d, r := duoRig()
	d.SetOrder(Order{Kind: "move", Dest: &pb.Pos{X: 4000, Y: 0}})
	d.mu.Lock()
	defer d.mu.Unlock()
	if got := kinds(d.next(r)); !same(got, []string{"move"}) {
		t.Errorf("first: %v", got)
	}
	if got := kinds(d.next(r)); len(got) != 0 || d.order.Kind != "move" {
		t.Errorf("on its way: %v, %+v", got, d.order)
	}
	r.Pos = &pb.Pos{X: 4500, Y: 0}
	d.next(r)
	if d.order.Kind != "" {
		t.Errorf("arrived, and the order stands: %+v", d.order)
	}
}

func TestDuoStopOrderStopsOnceAndHoldsEmptyHanded(t *testing.T) {
	d, r := duoRig()
	r.Target, r.AutoAttack = "mob-1", true
	d.SetStance("assist")
	d.SetOrder(Order{Kind: "stop"})
	d.mu.Lock()
	defer d.mu.Unlock()
	if got := kinds(d.next(r)); !same(got, []string{"stop", "target "}) {
		t.Errorf("first: %v", got)
	}
	r.Target, r.AutoAttack = "", false
	if got := kinds(d.next(r)); len(got) != 0 {
		t.Errorf("held: %v", got)
	}
	// A new stance drops the order.
	d.mu.Unlock()
	d.SetStance("guard")
	d.mu.Lock()
	if d.order.Kind != "" {
		t.Errorf("a stance left the order: %+v", d.order)
	}
}
