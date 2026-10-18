package model_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func chain(step int64, text string) *pb.QuestInfo {
	return &pb.QuestInfo{Id: "first-steps", Name: "First Steps", Step: step, Steps: 4, Text: text}
}

func questEvent(m *model.Model, ms int64, reason string, step int64, text string, r *pb.RewardInfo) {
	event(m, ms, &pb.Event{Body: &pb.Event_Quest{Quest: &pb.QuestEvent{Quest: "first-steps", Name: "First Steps", Reason: reason, Step: step, Text: text, Reward: r}}})
}

func logLine(m *model.Model, kind string) string {
	for i := len(m.Log) - 1; i >= 0; i-- {
		if m.Log[i].Kind == kind {
			return m.Log[i].Text
		}
	}
	return ""
}

// The snapshot's chain is the status; the quest events say in words what changed
// (D62), and the ledger reward the server records beside them says nothing more.
func TestQuestChainStatusAndLogLines(t *testing.T) {
	m := model.New()
	snap(m, 1, "town", &pb.SelfState{Quest: chain(1, "Speak to the greeter.")}, nil)
	if q := m.Quest(); q == nil || model.QuestProgress(q) != "First Steps, step 1 of 4" {
		t.Fatalf("the chain: %v", m.Quest())
	}
	if got := logLine(m, "quest"); !strings.Contains(got, "Quest chain offered: First Steps. First: Speak to the greeter.") || !strings.Contains(got, "optional") {
		t.Errorf("the chain a character logs in on is not in the log: %q", got)
	}

	questEvent(m, 100, "step", 1, "Visit your trainer.", nil)
	if got := logLine(m, "quest"); got != "First Steps: step 1 done. Next: Visit your trainer." {
		t.Errorf("step: %q", got)
	}
	questEvent(m, 200, "step", 2, "Take a contract.", &pb.RewardInfo{Crowns: 15})
	if got := logLine(m, "quest"); got != "First Steps: step 2 done (paid 15 crowns). Next: Take a contract." {
		t.Errorf("paid step: %q", got)
	}
	n := len(m.Log)
	event(m, 200, &pb.Event{Body: &pb.Event_Reward{Reward: &pb.Reward{Recipient: "me", Kind: "quest"}}})
	if len(m.Log) != n {
		t.Errorf("the ledger reward of a quest change is in the log: %v", m.Log[n:])
	}
	questEvent(m, 300, "done", 4, "", &pb.RewardInfo{Xp: 20})
	if got := logLine(m, "quest"); got != "First Steps: finished (paid 20 XP)." {
		t.Errorf("done: %q", got)
	}
	questEvent(m, 400, "dismissed", 0, "", nil)
	if got := logLine(m, "quest"); got != "First Steps dismissed; it will not be offered again." {
		t.Errorf("dismissed: %q", got)
	}
	questEvent(m, 500, "offered", 1, "Speak to the greeter.", nil)
	if got := logLine(m, "quest"); !strings.HasPrefix(got, "Quest chain offered: First Steps.") {
		t.Errorf("offered: %q", got)
	}
	snap(m, 2, "town", &pb.SelfState{}, nil)
	if m.Quest() != nil {
		t.Errorf("a snapshot with no chain leaves one: %v", m.Quest())
	}
}

// A fight stops a walk (D66): the server cancels the order and says nothing but
// the event that did it, so the log says it.
func TestAFightThatStopsAWalkIsInTheLog(t *testing.T) {
	setup := func() *model.Model {
		m := model.New()
		snap(m, 1, "z", &pb.SelfState{HasDestination: true, Destination: &pb.Pos{X: 9000, Y: 1000}}, nil,
			&pb.EntityState{Id: "wolf", Name: "Wolf", Kind: pb.EntityKind_ENTITY_KIND_MOB, Hp: 10, MaxHp: 10, Pos: &pb.Pos{X: 5000, Y: 1000}})
		return m
	}
	opened := func(owner string) *pb.Event {
		return &pb.Event{Body: &pb.Event_EncounterOpened{EncounterOpened: &pb.EncounterOpened{Id: "enc-1", Mob: "wolf", Owner: owner}}}
	}

	m := setup()
	event(m, 100, opened("me"))
	if got := logLine(m, "walk_stopped"); !strings.HasPrefix(got, "Your walk stopped: you are fighting Wolf") {
		t.Errorf("engaged: %q", got)
	}
	if m.Self.HasDestination {
		t.Error("the destination stands after the walk was stopped")
	}
	if m.Log[len(m.Log)-1].Kind != "walk_stopped" || m.Log[len(m.Log)-2].Kind != "encounter_opened" {
		t.Errorf("the walk line does not follow the event that caused it: %v", m.Log)
	}

	// Attacking, and a hostile ability, stop it too; a heal on oneself does not.
	m = setup()
	event(m, 100, &pb.Event{Body: &pb.Event_AutoAttackChanged{AutoAttackChanged: &pb.AutoAttackChanged{Actor: "me", On: true, Target: "wolf"}}})
	if got := logLine(m, "walk_stopped"); !strings.Contains(got, "you began attacking") {
		t.Errorf("attack: %q", got)
	}
	m = setup()
	event(m, 100, &pb.Event{Body: &pb.Event_AbilityUsed{AbilityUsed: &pb.AbilityUsed{Actor: "me", Ability: "bash", Target: "me"}}})
	if got := logLine(m, "walk_stopped"); got != "" {
		t.Errorf("an ability on oneself stopped the walk: %q", got)
	}
	event(m, 200, &pb.Event{Body: &pb.Event_AbilityUsed{AbilityUsed: &pb.AbilityUsed{Actor: "me", Ability: "bash", Target: "wolf"}}})
	if got := logLine(m, "walk_stopped"); !strings.Contains(got, "you used an ability on Wolf") {
		t.Errorf("ability: %q", got)
	}

	// Standing still, or someone else's fight, says nothing; so does a second
	// event after the walk is already stopped.
	m = model.New()
	snap(m, 1, "z", &pb.SelfState{}, nil)
	event(m, 100, opened("me"))
	if got := logLine(m, "walk_stopped"); got != "" {
		t.Errorf("a character that was not walking: %q", got)
	}
	m = setup()
	event(m, 100, opened("stranger"))
	if got := logLine(m, "walk_stopped"); got != "" {
		t.Errorf("a stranger's fight: %q", got)
	}
	m = setup()
	event(m, 100, opened("me"))
	event(m, 110, &pb.Event{Body: &pb.Event_AutoAttackChanged{AutoAttackChanged: &pb.AutoAttackChanged{Actor: "me", On: true, Target: "wolf"}}})
	n := 0
	for _, e := range m.Log {
		if e.Kind == "walk_stopped" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("%d walk lines for one walk", n)
	}
}

// The zone map's standing NPCs are known before they are seen (D69).
func TestStandingNPCsOfTheZoneMapAreKnownBeforeSeen(t *testing.T) {
	m := model.New()
	snap(m, 1, "town", &pb.SelfState{}, nil)
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: "town", Npcs: []*pb.NpcPlace{
		{Id: "greeter", Name: "Tamsin Hale", Role: "guard", Pos: &pb.Pos{X: 5000, Y: 5000}},
		{Id: "board", Name: "Contract Board", Role: "contract_board", Pos: &pb.Pos{X: 9000, Y: 5000}},
	}}}})
	var names []string
	for _, n := range m.KnownNPCs() {
		names = append(names, n.GetName())
	}
	if strings.Join(names, ",") != "Contract Board,Tamsin Hale" {
		t.Errorf("known NPCs: %v", names)
	}
	// One seen in view keeps its own description.
	snap(m, 2, "town", &pb.SelfState{}, []*pb.NpcInfo{{Id: "board", Name: "Contract Board", Role: "contract_board", Pos: &pb.Pos{X: 9000, Y: 5000}, Offers: []*pb.ContractOffer{{Id: "x"}}}})
	for _, n := range m.KnownNPCs() {
		if n.GetId() == "board" && len(n.GetOffers()) != 1 {
			t.Errorf("the NPC in view lost its offers: %v", n)
		}
	}
}
