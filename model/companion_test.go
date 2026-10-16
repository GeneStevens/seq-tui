package model_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func captain(offers ...*pb.ContractOffer) *pb.NpcInfo {
	return &pb.NpcInfo{Id: "captain", Name: "Captain Hartley", Role: "quest_giver", Pos: &pb.Pos{X: 2000, Y: 1000}, Offers: offers}
}

var (
	stepOne = &pb.ContractOffer{Id: "lookout", Name: "Eyes on the Goblins", Description: "Walk out to the lookout.", Kind: "scout", Need: 1}
	stepTwo = &pb.ContractOffer{Id: "raiders", Name: "Break the Raiding Party", Description: "Kill 5 goblins.", Kind: "hunt", Need: 5}
)

// The quest is the contracts a quest giver's board offers: the protocol marks no
// contract as a step, so a step is one that board offered. Each state of a step
// follows from what the snapshot says is offered and held.
func TestQuestStepsFollowTheBoardAndWhatIsHeld(t *testing.T) {
	m := model.New()
	if m.QuestShown() {
		t.Fatal("a quest is shown before anything is known of it")
	}
	snap(m, 1, "town", &pb.SelfState{}, []*pb.NpcInfo{captain(stepOne)})
	states := func() []model.QuestStepState {
		var out []model.QuestStepState
		for _, s := range m.QuestSteps() {
			out = append(out, s.State)
		}
		return out
	}
	want := func(w ...model.QuestStepState) {
		t.Helper()
		got := states()
		if len(got) != len(w) {
			t.Fatalf("steps %v, want %v", got, w)
		}
		for i := range w {
			if got[i] != w[i] {
				t.Fatalf("steps %v, want %v", got, w)
			}
		}
	}
	want(model.StepOffered)
	if !m.QuestShown() || m.QuestGiver() != "captain" {
		t.Fatalf("shown %v, giver %q", m.QuestShown(), m.QuestGiver())
	}

	// Taken: the board no longer offers it, the character holds it.
	held := &pb.ContractInfo{Id: "lookout", Name: "Eyes on the Goblins", Kind: "scout", Have: 0, Need: 1}
	snap(m, 2, "town", &pb.SelfState{Contracts: []*pb.ContractInfo{held}, Companion: &pb.CompanionState{}}, []*pb.NpcInfo{captain()})
	want(model.StepHeld)
	held = &pb.ContractInfo{Id: "lookout", Name: "Eyes on the Goblins", Kind: "scout", Have: 1, Need: 1, Ready: true}
	snap(m, 3, "town", &pb.SelfState{Contracts: []*pb.ContractInfo{held}, Companion: &pb.CompanionState{}}, []*pb.NpcInfo{captain()})
	want(model.StepReady)

	// Handed in: the second step is offered, the first is done.
	snap(m, 4, "town", &pb.SelfState{Companion: &pb.CompanionState{Stage: 1}}, []*pb.NpcInfo{captain(stepTwo)})
	want(model.StepDone, model.StepOffered)
	if s := m.QuestSteps()[1]; s.Name != "Break the Raiding Party" || s.Need != 5 || s.Kind != "hunt" {
		t.Errorf("the second step: %+v", s)
	}

	// The last step handed in: unlocked, and the role not chosen: still shown.
	snap(m, 5, "town", &pb.SelfState{Companion: &pb.CompanionState{Stage: 2, Unlocked: true}}, []*pb.NpcInfo{captain()})
	want(model.StepDone, model.StepDone)
	if !m.QuestShown() {
		t.Error("the quest is hidden before the role is chosen")
	}
	snap(m, 6, "town", &pb.SelfState{Companion: &pb.CompanionState{Stage: 2, Unlocked: true, Role: "healer"}}, []*pb.NpcInfo{captain()})
	if m.QuestShown() {
		t.Error("the quest is shown after the role is chosen")
	}
}

// The companion is the group member that carries the character as its owner; a
// character of the account that is not one is not taken for it.
func TestTheCompanionIsTheGroupMemberThatIsOwnedByTheCharacter(t *testing.T) {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "downs", Seq: 1,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Pos: &pb.Pos{}}, Companion: &pb.CompanionState{Unlocked: true, Role: "tank", Present: true, Id: "comp-me"}},
		Group: []*pb.GroupMember{
			{Entity: &pb.EntityState{Id: "bram", Name: "Bram", Kind: pb.EntityKind_ENTITY_KIND_CHARACTER}},
			{Entity: &pb.EntityState{Id: "comp-me", Name: "Watch Shieldman", Owner: "me", Hp: 90, MaxHp: 100, Level: 5}, Target: "mob-1"},
		}}}})
	g := m.CompanionMember()
	if g == nil || g.Entity.GetName() != "Watch Shieldman" || g.Entity.GetOwner() != "me" || g.Target != "mob-1" {
		t.Fatalf("companion member %+v", g)
	}
	if m.InView("bram") == nil || m.InView("nobody") != nil {
		t.Error("InView")
	}
	if c := m.Companion(); c.GetRole() != "tank" || !c.GetPresent() {
		t.Errorf("companion state %v", c)
	}
}

func TestCompanionEventsAreInTheLogInWords(t *testing.T) {
	m := model.New()
	snap(m, 1, "town", &pb.SelfState{}, nil)
	for _, tc := range []struct {
		ev   *pb.CompanionEvent
		want string
	}{
		{&pb.CompanionEvent{Owner: "me", Reason: "unlocked"}, "your quest is done: choose your recruit's role"},
		{&pb.CompanionEvent{Owner: "me", Reason: "role", Text: "healer"}, "your recruit is now a healer"},
		{&pb.CompanionEvent{Owner: "me", Companion: "comp-me", Reason: "summoned", Text: "Watch Medic"}, "your companion Watch Medic came to its side"},
		{&pb.CompanionEvent{Owner: "me", Reason: "stance", Text: "guard"}, "your companion is now on guard"},
		{&pb.CompanionEvent{Owner: "me", Reason: "order", Text: "attack"}, "your companion is ordered: attack"},
		{&pb.CompanionEvent{Owner: "me", Reason: "dismissed"}, "your companion was sent away"},
		{&pb.CompanionEvent{Owner: "other", Reason: "stance", Text: "assist"}, "other's companion is now on assist"},
	} {
		event(m, 1000, &pb.Event{Body: &pb.Event_Companion{Companion: tc.ev}})
		if e := m.Log[len(m.Log)-1]; e.Kind != "companion" || !strings.Contains(e.Text, tc.want) {
			t.Errorf("%s: logged %q", tc.ev.Reason, e.Text)
		}
	}
}

func TestHelperActionIsInWords(t *testing.T) {
	m := model.New()
	snap(m, 1, "downs", &pb.SelfState{}, nil, &pb.EntityState{Id: "mob-1", Name: "Tusker", Kind: pb.EntityKind_ENTITY_KIND_MOB})
	fighting := &model.Member{Target: "mob-1"}
	for _, tc := range []struct {
		g      *model.Member
		order  string
		stance string
		want   string
	}{
		{fighting, "stop", "assist", "ordered to stand"},
		{fighting, "", "assist", "fighting Tusker"},
		{&model.Member{}, "", "follow", "following you"},
		{&model.Member{}, "", "passive", "standing"},
		{nil, "", "guard", "waiting for a fight"},
		{nil, "move", "follow", "ordered to move"},
	} {
		if got := model.HelperAction(m, tc.g, tc.order, tc.stance); got != tc.want {
			t.Errorf("order %q stance %q: %q, want %q", tc.order, tc.stance, got, tc.want)
		}
	}
}
