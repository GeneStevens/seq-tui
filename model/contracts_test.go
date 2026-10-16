package model_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func snap(m *model.Model, seq uint64, space string, self *pb.SelfState, npcs []*pb.NpcInfo, ents ...*pb.EntityState) {
	self.Entity = &pb.EntityState{Id: "me", Name: "Me", Pos: &pb.Pos{X: 1000, Y: 1000}}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: space, Seq: seq, Self: self, Npcs: npcs, Entities: ents}}})
}

// The board's offers come with the snapshot; the contracts held are named, and a
// held one that a zone without the board describes by its id is filled in from
// what the board offered.
func TestHeldContractsAreNamedAwayFromTheBoard(t *testing.T) {
	m := model.New()
	board := &pb.NpcInfo{Id: "board", Name: "The Board", Role: "contract_board", Pos: &pb.Pos{X: 2000, Y: 1000}, Offers: []*pb.ContractOffer{
		{Id: "hunt-wolves", Name: "Thin the Wolves", Description: "Kill 3 wolves.", Kind: "hunt", Need: 3, MinLevel: 1, MaxLevel: 3, Reward: &pb.RewardInfo{Xp: 30, Crowns: 12, Item: "tin-ring"}},
	}}
	snap(m, 1, "town", &pb.SelfState{}, []*pb.NpcInfo{board})
	if got := m.RewardWords(board.Offers[0].Reward); got != "30 XP, 12 crowns and Tin Ring" {
		t.Errorf("reward words %q", got)
	}
	// Held, in the town: described in full.
	snap(m, 2, "town", &pb.SelfState{Contracts: []*pb.ContractInfo{{Id: "hunt-wolves", Name: "Thin the Wolves", Kind: "hunt", Have: 1, Need: 3, Reward: &pb.RewardInfo{Xp: 30, Crowns: 12, Item: "tin-ring"}}}}, nil)
	// Away: only the id.
	snap(m, 3, "downs", &pb.SelfState{Contracts: []*pb.ContractInfo{{Id: "hunt-wolves", Name: "hunt-wolves", Kind: "hunt", Have: 2, Need: 3}}}, nil)
	c := m.Contract("hunt-wolves")
	if c.GetName() != "Thin the Wolves" || c.GetDescription() != "Kill 3 wolves." || c.GetReward().GetCrowns() != 12 || c.GetHave() != 2 {
		t.Errorf("held contract away from the board: %v", c)
	}
	if model.Progress(c) != "2 of 3 killed" {
		t.Errorf("progress %q", model.Progress(c))
	}
	if m.ContractName("hunt-wolves") != "Thin the Wolves" || m.ContractName("gather-dust") != "Gather Dust" {
		t.Errorf("names %q %q", m.ContractName("hunt-wolves"), m.ContractName("gather-dust"))
	}
}

func TestProgressInWords(t *testing.T) {
	for _, tc := range []struct {
		c    *pb.ContractInfo
		want string
	}{
		{&pb.ContractInfo{Kind: "hunt", Have: 1, Need: 4}, "1 of 4 killed"},
		{&pb.ContractInfo{Kind: "bounty", Have: 0, Need: 1}, "0 of 1 killed"},
		{&pb.ContractInfo{Kind: "gather", Have: 2, Need: 3}, "2 of 3 carried"},
		{&pb.ContractInfo{Kind: "scout", Have: 0, Need: 1}, "go to the marker"},
		{&pb.ContractInfo{Kind: "scout", Have: 1, Need: 1}, "marker reached"},
	} {
		if got := model.Progress(tc.c); got != tc.want {
			t.Errorf("%v: %q, want %q", tc.c, got, tc.want)
		}
	}
}

// What an NPC says is kept for the one who talked; the answer for someone else's
// talk is a log line only.
func TestDialogueAndRewardsInTheLog(t *testing.T) {
	m := model.New()
	snap(m, 1, "town", &pb.SelfState{}, []*pb.NpcInfo{{Id: "board", Name: "The Board", Role: "contract_board", Pos: &pb.Pos{X: 2000, Y: 1000}}})
	event(m, 1000, &pb.Event{Body: &pb.Event_Dialogue{Dialogue: &pb.Dialogue{Actor: "me", Npc: "board", Lines: []string{"Short jobs."}, Options: []*pb.DialogueOption{{Action: "contract_accept", Label: "Take a contract"}}}}})
	d := m.Said("board")
	if d == nil || d.Lines[0] != "Short jobs." || d.Options[0].GetAction() != "contract_accept" || m.Talks != 1 {
		t.Fatalf("dialogue %+v talks %d", d, m.Talks)
	}
	if got := lastText(m); got != "The Board says: Short jobs." {
		t.Errorf("log: %q", got)
	}
	event(m, 1100, &pb.Event{Body: &pb.Event_Dialogue{Dialogue: &pb.Dialogue{Actor: "bram", Npc: "board", Lines: []string{"Other"}}}})
	if m.Said("board").Lines[0] != "Short jobs." || m.Talks != 1 {
		t.Errorf("another character's talk replaced mine: %+v", m.Said("board"))
	}

	// Rewards of the contract kinds.
	for _, tc := range []struct {
		r    *pb.Reward
		want string
	}{
		{&pb.Reward{Recipient: "me", Source: "board", Kind: "contract_accept", Detail: "hunt-wolves"}, "you took the contract Hunt Wolves"},
		{&pb.Reward{Recipient: "me", Kind: "contract_abandon", Detail: "hunt-wolves"}, "you gave up the contract Hunt Wolves"},
		{&pb.Reward{Recipient: "me", Source: "board", Kind: "contract_turn_in", Detail: "hunt-wolves", Xp: 30, Amount: 12, Item: "tin-ring"}, "you handed in Hunt Wolves to The Board and got 30 XP, 12 crowns and Tin Ring"},
		{&pb.Reward{Recipient: "me", Source: "raid", Kind: "world_event", Detail: "harvest-raid", Xp: 120, Amount: 60}, "you was paid for the Harvest Raid: 120 XP and 60 crowns"},
	} {
		event(m, 2000, &pb.Event{Body: &pb.Event_Reward{Reward: tc.r}})
		if got := lastText(m); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.r.Kind, got, tc.want)
		}
	}
}

// A world event is told to the zone: the model keeps the last announcement of
// each, active until it is won or lost, and a recent end is still shown.
func TestWorldEventAnnouncements(t *testing.T) {
	m := model.New()
	snap(m, 1, "downs", &pb.SelfState{}, nil)
	ann := func(ms int64, kind string, leaks int64) {
		event(m, ms, &pb.Event{Body: &pb.Event_WorldEvent{WorldEvent: &pb.WorldEvent{Event: "harvest-raid", Kind: kind, Text: "Defend the farms!", Leaks: leaks}}})
	}
	if len(m.ActiveEvents()) != 0 || m.RecentEvent(0, 30000) != nil {
		t.Fatal("an event before any announcement")
	}
	ann(10_000, "started", 0)
	if len(m.ActiveEvents()) != 1 || !strings.Contains(lastText(m), "has begun") {
		t.Errorf("started: %v / %q", m.ActiveEvents(), lastText(m))
	}
	ann(40_000, "leak", 1)
	if e := m.WorldEvents["harvest-raid"]; !e.Active || e.Leaks != 1 || !strings.Contains(lastText(m), "got through (1 so far)") {
		t.Errorf("leak: %+v / %q", e, lastText(m))
	}
	ann(100_000, "won", 1)
	if len(m.ActiveEvents()) != 0 || !strings.Contains(lastText(m), "is won") {
		t.Errorf("won: %v / %q", m.ActiveEvents(), lastText(m))
	}
	if m.RecentEvent(120_000, 30_000) == nil || m.RecentEvent(131_000, 30_000) != nil {
		t.Error("the end of an event is recent for thirty seconds")
	}
}

// Rares are known by name; one coming into view is a log line of its own, and a
// camp is where a mob was first seen: not a rare, not an event's marching mob.
func TestRaresCampsAndPeopleAreRemembered(t *testing.T) {
	m := model.New()
	wolf := &pb.EntityState{Id: "mob-a-1", Kind: pb.EntityKind_ENTITY_KIND_MOB, Name: "Wolf", Hp: 10, MaxHp: 10, Pos: &pb.Pos{X: 30000, Y: 30000}}
	snap(m, 1, "downs", &pb.SelfState{}, []*pb.NpcInfo{{Id: "captain", Name: "Captain", Role: "quest_giver", Pos: &pb.Pos{X: 5000, Y: 5000}}}, wolf)
	event(m, 1000, &pb.Event{Body: &pb.Event_EntityAppeared{EntityAppeared: &pb.EntityAppeared{Entity: &pb.EntityState{
		Id: "mob-grimtusk-1", Kind: pb.EntityKind_ENTITY_KIND_MOB, Name: "Old Grimtusk", Level: 6, Hp: 300, MaxHp: 300, Pos: &pb.Pos{X: 50000, Y: 50000}}}}})
	if r := m.RaresInView(); len(r) != 1 || r[0].GetName() != "Old Grimtusk" {
		t.Fatalf("rares in view: %v", r)
	}
	if e := m.Log[len(m.Log)-1]; e.Kind != "rare_appeared" || !strings.Contains(e.Text, "a rare, Old Grimtusk (Lv 6)") {
		t.Errorf("log: %+v", e)
	}
	if !model.IsRare(m.Entities["mob-grimtusk-1"]) || model.IsRare(wolf) {
		t.Error("IsRare")
	}
	// The wolf's place is a camp, the rare's is not; a second wolf in the same
	// spot is the same camp; one that comes while an event is on is no camp.
	event(m, 2000, &pb.Event{Body: &pb.Event_EntityAppeared{EntityAppeared: &pb.EntityAppeared{Entity: &pb.EntityState{
		Id: "mob-a-2", Kind: pb.EntityKind_ENTITY_KIND_MOB, Name: "Wolf", Pos: &pb.Pos{X: 33000, Y: 30000}}}}})
	event(m, 3000, &pb.Event{Body: &pb.Event_WorldEvent{WorldEvent: &pb.WorldEvent{Event: "raid", Kind: "started"}}})
	event(m, 4000, &pb.Event{Body: &pb.Event_EntityAppeared{EntityAppeared: &pb.EntityAppeared{Entity: &pb.EntityState{
		Id: "mob-ev-1", Kind: pb.EntityKind_ENTITY_KIND_MOB, Name: "Raider", Pos: &pb.Pos{X: 90000, Y: 90000}}}}})
	if c := m.Known().Camps; len(c) != 1 || c[0].GetX() != 30000 {
		t.Errorf("camps: %v", c)
	}
	// The people seen stay known after a snapshot that no longer lists them.
	snap(m, 2, "downs", &pb.SelfState{}, nil)
	if n := m.KnownNPCs(); len(n) != 1 || n[0].GetId() != "captain" || m.NPC("captain") == nil || m.NearestNPC().GetId() != "captain" {
		t.Errorf("known people: %v", n)
	}
	if len(m.Known().Camps) != 1 {
		t.Error("camps are lost with a snapshot")
	}
}
