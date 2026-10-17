package model_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func event(m *model.Model, ms int64, e *pb.Event) {
	e.Seq, e.GameTimeMs = m.Last()+1, ms
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: e}})
}

func lastText(m *model.Model) string { return m.Log[len(m.Log)-1].Text }

// The snapshot carries the economy: crowns, what is worn and banked, the items
// the server described, the traders in view and the trade window.
func TestSnapshotCarriesTheEconomy(t *testing.T) {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{
		Space: "z", Seq: 1,
		Self: &pb.SelfState{
			Entity: &pb.EntityState{Id: "me", Name: "Me"}, Crowns: 84, Inventory: []string{"worn-cap!", "mystery-gem"},
			Equipped: []string{"worn-cap", "", "", "", "", "", "", "", "scuttler-blade", ""}, Bank: []string{"buckler"}, BindZone: "testgrounds", InventorySize: 20,
		},
		Items: []*pb.ItemInfo{{Id: "worn-cap", Name: "Worn Cap", Tier: "common", Slot: "head"}, {Id: "scuttler-blade", Name: "Scuttler Blade", Tier: "rare", BindOnPickup: true}},
		Npcs:  []*pb.NpcInfo{{Id: "vendor", Name: "Vendor", Role: "vendor"}},
		Trade: &pb.TradeState{With: "bram", Incoming: true},
	}}})
	s := m.Self
	if s.Crowns != 84 || s.Bank[0] != "buckler" || s.BindZone != "testgrounds" || s.InventorySize != 20 || m.Worn(0) != "worn-cap" || m.Worn(8) != "scuttler-blade" || m.Worn(1) != "" {
		t.Fatalf("self %+v", s)
	}
	if m.ItemName("worn-cap!") != "Worn Cap" || m.ItemName("mystery-gem") != "mystery-gem" || !m.Bound("worn-cap!") || m.Bound("worn-cap") || !m.Bound("scuttler-blade") {
		t.Errorf("names or binding: %q %q", m.ItemName("worn-cap!"), m.ItemName("mystery-gem"))
	}
	if m.NPC("vendor").GetRole() != "vendor" || m.Trade.GetWith() != "bram" || !m.Trade.GetIncoming() {
		t.Errorf("npcs %v trade %v", m.NPCs, m.Trade)
	}
	// The next snapshot says the trade is over.
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 2, Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me"}}}}})
	if m.Trade != nil || len(m.NPCs) != 0 || m.ItemName("worn-cap") != "Worn Cap" {
		t.Errorf("trade %v, npcs %v; the catalogue should be kept", m.Trade, m.NPCs)
	}
}

// Loot that dropped says whether this character may take it; coin is a drop
// and a reward of its own.
func TestLootDropsSayWhetherYouMayTakeThem(t *testing.T) {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 1, Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me"}}}}})
	event(m, 100, &pb.Event{Body: &pb.Event_LootDropped{LootDropped: &pb.LootDropped{Drop: "d1", Mob: "mob", Item: "scuttler-blade", Pos: &pb.Pos{},
		ItemInfo: &pb.ItemInfo{Id: "scuttler-blade", Name: "Scuttler Blade", Tier: "rare", BindOnPickup: true}}}})
	event(m, 100, &pb.Event{Body: &pb.Event_LootDropped{LootDropped: &pb.LootDropped{Drop: "d2", Mob: "mob", Crowns: 5, CanLoot: true, Pos: &pb.Pos{}}}})
	if m.Drops["d1"].GetCanLoot() || !m.Drops["d2"].GetCanLoot() || m.Drops["d2"].GetCrowns() != 5 {
		t.Fatalf("drops %v", m.Drops)
	}
	if !strings.Contains(m.Log[0].Text, "Scuttler Blade") || !strings.Contains(m.Log[0].Text, "[Rare, bound on pickup]") || !strings.Contains(m.Log[0].Text, "not yours to take") || !strings.Contains(m.Log[1].Text, "5 crowns") {
		t.Errorf("log %q / %q", m.Log[0].Text, m.Log[1].Text)
	}
	if m.DropName(m.Drops["d2"]) != "5 crowns" {
		t.Errorf("drop name %q", m.DropName(m.Drops["d2"]))
	}
	event(m, 200, &pb.Event{Body: &pb.Event_Reward{Reward: &pb.Reward{Recipient: "me", Source: "d2", Kind: "crowns", Amount: 5}}})
	if m.Self.Crowns != 5 || len(m.Drops) != 1 || lastText(m) != "you took 5 crowns" {
		t.Errorf("crowns %d, drops %v, %q", m.Self.Crowns, m.Drops, lastText(m))
	}
}

// Every economic reward is told in words; the fee and its refund too.
func TestEconomyRewardsInTheLog(t *testing.T) {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 1,
		Self:  &pb.SelfState{Entity: &pb.EntityState{Id: "me"}},
		Items: []*pb.ItemInfo{{Id: "worn-cap", Name: "Worn Cap"}},
		Npcs:  []*pb.NpcInfo{{Id: "vendor", Name: "Testgrounds Vendor"}}}}})
	event(m, 50, &pb.Event{Body: &pb.Event_InstanceOpenFailed{InstanceOpenFailed: &pb.InstanceOpenFailed{Reason: "instance_unavailable"}}})
	for _, tc := range []struct {
		r    *pb.Reward
		want string
	}{
		{&pb.Reward{Kind: "buy", Source: "vendor", Item: "worn-cap", Amount: 8}, "you bought Worn Cap for 8 crowns from Testgrounds Vendor"},
		{&pb.Reward{Kind: "sell", Source: "vendor", Item: "worn-cap", Amount: 1}, "you sold Worn Cap for 1 crown to Testgrounds Vendor"},
		{&pb.Reward{Kind: "train", Source: "vendor", Amount: 45, Detail: "wizard-arcane-ward"}, "you learned wizard-arcane-ward for 45 crowns from Testgrounds Vendor"},
		{&pb.Reward{Kind: "bind", Source: "vendor", Amount: 20}, "you set the bind point at Testgrounds Vendor for 20 crowns"},
		{&pb.Reward{Kind: "fee", Amount: 50, Detail: "peril-den"}, "you paid an entry fee of 50 crowns"},
		{&pb.Reward{Kind: "refund", Amount: 50, Detail: "peril-den"}, "the entry fee of 50 crowns was refunded to you: the instance could not be opened"},
		{&pb.Reward{Kind: "equip", Item: "worn-cap", Detail: "head"}, "you put on Worn Cap (head)"},
		{&pb.Reward{Kind: "equip", Item: "worn-cap"}, "you took off Worn Cap"},
		{&pb.Reward{Kind: "bank", Item: "worn-cap", Detail: "deposit"}, "you put Worn Cap in the bank"},
		{&pb.Reward{Kind: "bank", Item: "worn-cap", Detail: "withdraw"}, "you took Worn Cap out of the bank"},
	} {
		tc.r.Recipient = "me"
		event(m, 100, &pb.Event{Body: &pb.Event_Reward{Reward: tc.r}})
		got := lastText(m)
		if got != tc.want {
			t.Errorf("%s: %q, want %q", tc.r.Kind, got, tc.want)
		}
	}
}

func TestTradeEventsInTheLog(t *testing.T) {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 1,
		Self:     &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me"}},
		Entities: []*pb.EntityState{{Id: "bram", Name: "Bram"}}}}})
	for _, tc := range []struct {
		e    *pb.TradeEvent
		want string
	}{
		{&pb.TradeEvent{Kind: "request", Actor: "bram", Target: "me"}, "Bram asked you for a trade"},
		{&pb.TradeEvent{Kind: "opened", Actor: "me", Target: "bram"}, "the trade window with Bram is open"},
		{&pb.TradeEvent{Kind: "updated", Actor: "bram", Target: "me", Amount: 7}, "Bram changed the offer: 7 crowns"},
		{&pb.TradeEvent{Kind: "confirmed", Actor: "me", Target: "bram"}, "you confirmed"},
		{&pb.TradeEvent{Kind: "closed", Actor: "bram", Target: "me", Reason: "out_of_range"}, "the trade with Bram closed: too far apart"},
	} {
		event(m, 100, &pb.Event{Body: &pb.Event_Trade{Trade: tc.e}})
		if got := lastText(m); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.e.Kind, got, tc.want)
		}
	}
	if e := m.Log[len(m.Log)-1]; e.Reason != "out_of_range" {
		t.Errorf("the reason is kept: %+v", e)
	}
}

// D50: the snapshot lists the effects on everything in view, so a login in the
// middle of a fight shows a rooted mob; a cast that lands in a later round
// says which.
func TestSnapshotEffectsOnEntitiesAndNextRoundCasts(t *testing.T) {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 1, GameTimeMs: 1000,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me"}, Class: "wizard"},
		Entities: []*pb.EntityState{{Id: "mob", Kind: pb.EntityKind_ENTITY_KIND_MOB, Effects: []*pb.EffectState{
			{Ability: "wizard-root", Kind: "root", ExpiresAtMs: 13000}, {Ability: "druid-swarm", Kind: "dot", TicksLeft: 2, Tags: []string{"poison"}}}}},
	}}})
	fx := m.EffectsOn("mob")
	if len(fx) != 2 || fx[0].Kind != "root" || fx[0].EndsAtMs != 13000 || fx[1].Tags[0] != "poison" || !fx[1].Approx {
		t.Fatalf("effects on the mob: %+v", fx)
	}
	event(m, 1100, &pb.Event{Body: &pb.Event_CastStarted{CastStarted: &pb.CastStarted{Actor: "me", Ability: "wizard-frost-bolt", CastMs: 2000, LandsInRound: 3}}})
	if c := m.Self.Cast; c == nil || c.LandsInRound != 3 {
		t.Fatalf("cast %+v", m.Self.Cast)
	}
	if got := lastText(m); !strings.Contains(got, "lands in round 3") {
		t.Errorf("log %q", got)
	}
	// A trainer's lesson names the ability the character does not know (D52); an
	// ability nobody described has only its id, never a guessed name.
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 2,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me"}, Class: "wizard"},
		Npcs: []*pb.NpcInfo{{Id: "t", Role: "trainer", Trains: []*pb.TrainInfo{{Ability: "wizard-ward", Info: &pb.AbilityInfo{Id: "wizard-ward", Name: "Ward of Glass", Description: "Absorbs damage."}}}}}}}})
	if m.AbilityName("wizard-ward") != "Ward of Glass" || m.AbilityName("wizard-arcane-ward") != "wizard-arcane-ward" {
		t.Errorf("names %q %q", m.AbilityName("wizard-ward"), m.AbilityName("wizard-arcane-ward"))
	}
	// A cast already under way at login says the round it lands in.
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 3,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me"}, Cast: &pb.CastState{Ability: "wizard-frost-bolt", Slot: 1, EndsAtMs: 5000, LandsInRound: 4}}}}})
	if c := m.Self.Cast; c == nil || c.LandsInRound != 4 {
		t.Errorf("cast after a fresh login: %+v", m.Self.Cast)
	}
}

// The class menu is the welcome's: the model offers it and looks classes up.
func TestClassMenuComesFromTheWelcome(t *testing.T) {
	m := model.New()
	if len(m.Classes()) != 0 || m.Class("druid") != nil {
		t.Fatal("a menu before the welcome")
	}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Welcome{Welcome: &pb.Welcome{Classes: []*pb.ClassInfo{{Id: "druid", Name: "Druid", Roles: []string{"healer", "spell"}}}}}})
	if len(m.Classes()) != 1 || m.Class("druid").GetName() != "Druid" {
		t.Errorf("classes %v", m.Classes())
	}
}

// A round_started that only repeats the round (after a snapshot) is not logged
// again, but one that says another character has submitted is.
func TestRoundRepeatedWithANewSubmissionIsLogged(t *testing.T) {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 1, Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me"}}}}})
	start := func(submitted ...string) {
		event(m, 100, &pb.Event{Body: &pb.Event_RoundStarted{RoundStarted: &pb.RoundStarted{Round: 1, DeadlineMs: 5000, Participants: 2, Submitted: submitted}}})
	}
	start()
	start()
	n := len(m.Log)
	if n != 1 {
		t.Fatalf("an unchanged repeat was logged: %d lines", n)
	}
	start("bram")
	if len(m.Log) != 2 || !m.Round.HasSubmitted("bram") || !strings.Contains(lastText(m), "1 of 2 submitted: bram") {
		t.Errorf("log %d lines: %q", len(m.Log), lastText(m))
	}
}
