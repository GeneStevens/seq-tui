package app

import (
	"reflect"
	"testing"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func withCompanion(c *pb.CompanionState, target string) *model.Model {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "downs", Seq: 1,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Pos: &pb.Pos{X: 5000, Y: 6000}}, Target: target, Companion: c}}}})
	return m
}

// The helper keys against the model: what they send for the house companion, and
// the notes they give when there is nothing to command.
func TestHelperKeysCommandTheHouseCompanion(t *testing.T) {
	a := New(nil)
	ready := &pb.CompanionState{Stage: 2, Unlocked: true, Role: "healer", Stance: "assist"}
	m := withCompanion(ready, "mob-1")
	for _, tc := range []struct {
		key  string
		want Action
	}{
		{"p", Action{Kind: CompanionSummon}},
		{"h", Action{Kind: CompanionStance, ID: "guard"}},
		{"y", Action{Kind: CompanionOrder, ID: "attack", Target: "mob-1", Note: "mob-1"}},
		{"u", Action{Kind: CompanionOrder, ID: "move", X: 5000, Y: 6000}},
		{"j", Action{Kind: CompanionOrder, ID: "stop"}},
		{"z", Action{Kind: CompanionOrder, ID: "clear"}},
	} {
		if got := a.helperAction(tc.key, m); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %+v, want %+v", tc.key, got, tc.want)
		}
	}
	// With it at the character's side, P sends it away; the last stance goes round to the first.
	here := &pb.CompanionState{Stage: 2, Unlocked: true, Role: "healer", Stance: "passive", Present: true, Summoned: true}
	m = withCompanion(here, "")
	if got := a.helperAction("p", m); got.Kind != CompanionDismiss {
		t.Errorf("p, with it here: %+v", got)
	}
	if got := a.helperAction("h", m); got.Kind != CompanionStance || got.ID != "follow" {
		t.Errorf("h, from passive: %+v", got)
	}
	if got := a.helperAction("y", m); got.Kind != None || got.Note == "" {
		t.Errorf("y with no target: %+v", got)
	}
}

func TestHelperKeysWithNothingToCommandSayWhy(t *testing.T) {
	a := New(nil)
	for _, c := range []*pb.CompanionState{nil, {Stage: 1}, {Stage: 2, Unlocked: true}} { // no quest, half, no role
		m := withCompanion(c, "")
		if got := a.helperAction("p", m); got.Kind != None || got.Note == "" {
			t.Errorf("%v: %+v", c, got)
		}
	}
}

// In a turn-based instance the keys add plan steps instead, and a plan holds one
// stance and one order for the companion.
func TestHelperKeysInATurnBasedRoundAddPlanSteps(t *testing.T) {
	a := New(nil)
	m := withCompanion(&pb.CompanionState{Stage: 2, Unlocked: true, Role: "tank", Stance: "follow", Present: true}, "mob-1")
	m.Instance = &pb.InstanceInfo{TimeMode: "turn_based"}
	act := a.helperAction("h", m)
	if act.Kind != PlanAdd || act.Step.Kind != CompanionStance || act.Step.ID != "assist" {
		t.Fatalf("h: %+v", act)
	}
	var d Draft
	if why := d.Add(*act.Step); why != "" {
		t.Fatal(why)
	}
	if why := d.Add(*act.Step); why == "" {
		t.Error("a second stance was added")
	}
	if why := d.Add(Action{Kind: CompanionOrder, ID: "attack", Target: "mob-1"}); why != "" {
		t.Errorf("the order after the stance: %s", why)
	}
	if why := d.Add(Action{Kind: CompanionOrder, ID: "stop"}); why == "" {
		t.Error("a second order was added")
	}
	if why := d.Add(Action{Kind: Move, X: 1, Y: 2}); why != "" {
		t.Errorf("a move beside the companion's steps: %s", why)
	}
	if lines := d.Lines(m); len(lines) != 3 || lines[0] != "companion: stance assist" || lines[1] != "companion: attack mob-1" {
		t.Errorf("lines %q", lines)
	}
	steps := d.Agent()
	if len(steps) != 3 {
		t.Fatalf("steps %d", len(steps))
	}
	var s0, s1 pb.PlanStep
	steps[0](&s0)
	steps[1](&s1)
	if s0.GetCompanionStance().GetStance() != "assist" || s1.GetCompanionOrder().GetOrder() != "attack" || s1.GetCompanionOrder().GetTarget() != "mob-1" {
		t.Errorf("toolkit steps: %v %v", &s0, &s1)
	}
}

// A second character is commanded by the same keys when there is no house
// companion, or when the keys are pointed at it with the full stop.
func TestHelperKeysCommandTheSecondCharacter(t *testing.T) {
	a := New(nil)
	a.SetDuo(client.NewDuo(client.Config{Character: "bram"}, "me"))
	m := withCompanion(nil, "mob-1")
	if got := a.helperAction("p", m); got.Kind != DuoSummon {
		t.Errorf("p: %+v", got)
	}
	if got := a.helperAction("h", m); got.Kind != DuoStance || got.ID != "assist" {
		t.Errorf("h: %+v", got)
	}
	if got := a.helperAction("y", m); got.Kind != DuoOrder || got.ID != "attack" || got.Target != "mob-1" {
		t.Errorf("y: %+v", got)
	}
	if got := a.helperAction(".", m); got.Kind != None {
		t.Errorf(". with one helper: %+v", got)
	}
	// With a house companion as well, the full stop turns the keys from one to the other.
	m = withCompanion(&pb.CompanionState{Stage: 2, Unlocked: true, Role: "tank", Stance: "follow"}, "")
	if got := a.helperAction("p", m); got.Kind != CompanionSummon {
		t.Errorf("p, with both: %+v", got)
	}
	a.helperAction(".", m)
	if got := a.helperAction("p", m); got.Kind != DuoSummon {
		t.Errorf("p, pointed at the second character: %+v", got)
	}
	// A second character cannot plan in a turn-based round.
	m.Instance = &pb.InstanceInfo{TimeMode: "turn_based"}
	if got := a.helperAction("h", m); got.Kind != None || got.Note == "" {
		t.Errorf("h in a round: %+v", got)
	}
}
