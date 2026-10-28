package model_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// useModel is a character at hp of 100 carrying a draught, a loaf and an ale.
func useModel(hp int64, inCombat bool) *model.Model {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 1, GameTimeMs: 1000,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Hp: hp, MaxHp: 100}, InCombat: inCombat,
			Inventory: []string{"draught", "loaf", "ale"}, BankSize: 20, BankTiers: 0},
		Items: []*pb.ItemInfo{
			{Id: "draught", Name: "Draught", Use: &pb.UseInfo{Effect: "heal", Amount: 40, Consumed: true, CooldownGroup: "draught", CooldownMs: 60000, InCombat: "allowed"}},
			{Id: "loaf", Name: "Loaf", Use: &pb.UseInfo{Effect: "regen", Amount: 2, DurationMs: 20000, Consumed: true, CooldownGroup: "food", CooldownMs: 20000, InCombat: "allowed"}},
			{Id: "ale", Name: "Ale", Use: &pb.UseInfo{Effect: "regen", Amount: 1, ManaAmount: 2, DurationMs: 20000, Consumed: true, InCombat: "forbidden"}},
		}}}})
	return m
}

func TestUseRefusalIsSaidFromWhatTheClientKnows(t *testing.T) {
	for _, c := range []struct {
		name     string
		hp       int64
		inCombat bool
		item     string
		want     string
	}{
		{"a heal when hurt", 50, false, "draught", ""},
		{"a heal at full health", 100, false, "draught", "already at full health"},
		{"a heal in a fight is allowed", 50, true, "draught", ""},
		{"an ale in a fight", 50, true, "ale", "can't be used in a fight"},
		{"a loaf at full health", 100, false, "loaf", ""},
		{"not an item", 50, false, "nothing", "can't be used"},
	} {
		if got := useModel(c.hp, c.inCombat).UseRefusal(c.item); (c.want == "") != (got == "") || !strings.Contains(got, c.want) {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

// A use of the character's own starts the cooldown of the item's group and, for
// a regen, the resting bonus; another character's use starts neither.
func TestUseStartsTheCooldownAndTheRegen(t *testing.T) {
	m := useModel(50, false)
	other := &pb.Event{Body: &pb.Event_Reward{Reward: &pb.Reward{Recipient: "bram", Kind: "use", Item: "draught", Amount: 40, Detail: "heal;consumed"}}}
	event(m, 2000, other)
	if m.CooldownLeftMs("draught") != 0 {
		t.Fatal("another character's use started a cooldown")
	}
	event(m, 2000, &pb.Event{Body: &pb.Event_Reward{Reward: &pb.Reward{Recipient: "me", Kind: "use", Item: "draught", Amount: 40, Detail: "heal;consumed"}}})
	if got := m.CooldownLeftMs("draught"); got != 60000 {
		t.Errorf("cooldown left %d, want 60000", got)
	}
	if m.CooldownLeftMs("loaf") != 0 || m.RegenActive() != nil {
		t.Error("a heal started the food cooldown or a regen")
	}
	if why := m.UseRefusal("draught"); !strings.Contains(why, "isn't ready for 60s") {
		t.Errorf("refusal %q", why)
	}
	event(m, 3000, &pb.Event{Body: &pb.Event_Reward{Reward: &pb.Reward{Recipient: "me", Kind: "use", Item: "loaf", Amount: 20, Detail: "regen;consumed"}}})
	if r := m.RegenActive(); r == nil || r.HP != 2 || r.EndsAtMs != 23000 {
		t.Errorf("regen %+v", r)
	}
	event(m, 24000, &pb.Event{Body: &pb.Event_Reward{Reward: &pb.Reward{Recipient: "me", Kind: "bank_expand", Amount: 100, Detail: "1;20"}}})
	if m.RegenActive() != nil {
		t.Error("the regen outlived its time")
	}
	if got := lastText(m); !strings.Contains(got, "bought bank tier 1: 20 more slots for 100 crowns") {
		t.Errorf("log %q", got)
	}
}

func TestHealingItemPrefersOneThatCanBeUsed(t *testing.T) {
	if got := useModel(50, false).HealingItem(); got != "draught" {
		t.Errorf("healing item %q", got)
	}
	if got := useModel(50, false).HealingItem(); got == "" {
		t.Error("none")
	}
	if got := (&model.Model{}).HealingItem(); got != "" {
		t.Errorf("empty model: %q", got)
	}
}

func TestNextBankTierAndService(t *testing.T) {
	m := useModel(50, false)
	n := &pb.NpcInfo{Role: "banker", BankTiers: []*pb.BankTierInfo{{Slots: 20, Price: 100, Bought: true}, {Slots: 20, Price: 250}, {Slots: 40, Price: 500}}}
	if i, tier, ok := m.NextBankTier(n); !ok || i != 2 || tier.GetPrice() != 250 {
		t.Errorf("next tier %d %v %v", i, tier, ok)
	}
	n.BankTiers[1].Bought, n.BankTiers[2].Bought = true, true
	if _, _, ok := m.NextBankTier(n); ok {
		t.Error("a tier after the last")
	}
	for _, c := range []struct {
		n    *pb.NpcInfo
		want string
	}{
		{&pb.NpcInfo{Role: "quest_giver", Roles: []string{"quest_giver", "ferry"}}, "ferry"},
		{&pb.NpcInfo{Role: "quest_giver"}, "quest_giver"},
		{&pb.NpcInfo{Role: "ferry"}, "ferry"},
		{&pb.NpcInfo{Role: "vendor", Roles: []string{"ferry"}}, "vendor"},
	} {
		if got := model.Service(c.n); got != c.want {
			t.Errorf("%v: service %q, want %q", c.n, got, c.want)
		}
	}
}
