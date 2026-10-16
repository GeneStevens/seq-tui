package app_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/app"
	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// shopper is a program on a model that carries things and stands among traders.
func shopper(t *testing.T) (*app.App, *model.Model) {
	t.Helper()
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "Z", Seq: 1, GameTimeMs: 1000,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Pos: &pb.Pos{X: 10000}}, Crowns: 30, Inventory: []string{"worn-cap", "scuttler-blade"},
			Equipped: []string{"", "", "", "", "", "", "", "", "", ""}, InventorySize: 20},
		Items: []*pb.ItemInfo{{Id: "worn-cap", Name: "Worn Cap", Tier: "common", Slot: "head", SellPrice: 2}, {Id: "scuttler-blade", Name: "Scuttler Blade", Tier: "rare", BindOnPickup: true}},
		Npcs: []*pb.NpcInfo{{Id: "vendor", Name: "Vendor", Role: "vendor", Pos: &pb.Pos{X: 12000}, Stock: []*pb.StockItem{{Item: "worn-cap", Price: 8}}},
			{Id: "binder", Name: "Binder", Role: "binder", Pos: &pb.Pos{X: 30000}}},
		Entities: []*pb.EntityState{{Id: "bram", Name: "Bram", Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, Pos: &pb.Pos{X: 11000}}},
	}}})
	return app.New(client.NewSession(client.Config{Model: m})), m
}

func TestInventoryScreenKeys(t *testing.T) {
	a, _ := shopper(t)
	press(a, "c")
	s := screenOf(a)
	for _, want := range []string{"Inventory and gear", "Items carried", "> Worn Cap", "Scuttler Blade Rare [bound]", "bound on pickup"} {
		if want == "bound on pickup" {
			press(a, "down")
			s = screenOf(a)
		}
		if !strings.Contains(s, want) {
			t.Errorf("missing %q:\n%s", want, s)
		}
	}
	// Enter wears the item under the cursor: with no server the intent goes nowhere, but it is sent.
	if cmd := press(a, "enter"); cmd == nil {
		t.Error("enter did not send the equip")
	}
	// Tab to the gear: nothing is worn there.
	press(a, "tab")
	if cmd := press(a, "enter"); cmd != nil || !strings.Contains(screenOf(a), "nothing is worn there") {
		t.Errorf("take off from an empty slot:\n%s", screenOf(a))
	}
	press(a, "esc")
	if strings.Contains(screenOf(a), "Inventory and gear") {
		t.Error("esc did not close the screen")
	}
}

func TestTradersScreenKeys(t *testing.T) {
	a, _ := shopper(t)
	press(a, "n")
	if s := screenOf(a); !strings.Contains(s, "Vendor") || !strings.Contains(s, "Binder") || !strings.Contains(s, "sets your bind point") {
		t.Fatalf("no list of traders:\n%s", s)
	}
	press(a, "enter") // the nearest: the vendor
	s := screenOf(a)
	for _, want := range []string{"For sale (Enter buys one)", "Worn Cap", "You carry (Enter sells one)", "You have 30 crowns"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q:\n%s", want, s)
		}
	}
	if cmd := press(a, "enter"); cmd == nil {
		t.Error("enter on the stock did not buy")
	}
	press(a, "tab")
	if cmd := press(a, "enter"); cmd == nil {
		t.Error("enter on what is carried did not sell")
	}
	press(a, "esc")
	press(a, "down")
	press(a, "enter") // the binder
	if cmd := press(a, "enter"); cmd == nil {
		t.Error("enter at the binder did not bind")
	}
	press(a, "esc")
	press(a, "esc")
	if strings.Contains(screenOf(a), "Traders") {
		t.Error("esc did not close the screen")
	}
}

func TestTradeScreenKeys(t *testing.T) {
	a, _ := shopper(t)
	press(a, "t")
	if s := screenOf(a); !strings.Contains(s, "Trade with another character in reach") || !strings.Contains(s, "> Bram") {
		t.Fatalf("no one to ask:\n%s", s)
	}
	if cmd := press(a, "enter"); cmd == nil {
		t.Error("enter did not ask Bram")
	}
	press(a, "esc")
	if strings.Contains(screenOf(a), "Trade with another") {
		t.Error("esc did not close the screen")
	}
}

func TestTradeWindowKeys(t *testing.T) {
	a, m := shopper(t)
	m.Trade = &pb.TradeState{With: "bram", Open: true, Mine: &pb.TradeOffer{Items: []string{"worn-cap"}}, Theirs: &pb.TradeOffer{Crowns: 4}, TheirsConfirmed: true}
	press(a, "t")
	s := screenOf(a)
	for _, want := range []string{"Trade with Bram", "Your offer", "Bram's offer", "CONFIRMED", "crowns 4", "Tab: list"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q:\n%s", want, s)
		}
	}
	if cmd := press(a, "enter"); cmd == nil {
		t.Error("enter did not offer the item")
	}
	// C types the crowns; only digits count; Enter sends the total.
	press(a, "c")
	for _, k := range []string{"1", "x", "2"} {
		press(a, k)
	}
	if !strings.Contains(screenOf(a), "Crowns to offer in all: 12_") {
		t.Errorf("typing crowns:\n%s", screenOf(a))
	}
	if cmd := press(a, "enter"); cmd == nil {
		t.Error("enter did not send the crowns")
	}
	if cmd := press(a, "y"); cmd == nil {
		t.Error("y did not confirm")
	}
	if cmd := press(a, "x"); cmd == nil {
		t.Error("x did not cancel")
	}
	// Tab to the offer: Enter takes the item back.
	press(a, "tab")
	if cmd := press(a, "enter"); cmd == nil {
		t.Error("enter did not take the item back")
	}
}

// A drop the server marks as not this character's to take is not sent: the key
// says so. Auto-loot does not pick another character's drop.
func TestLootKeyDoesNotSendADropYouMayNotTake(t *testing.T) {
	_, m := shopper(t)
	m.Drops["d1"] = &pb.LootDropState{Id: "d1", Item: "scuttler-blade", Pos: &pb.Pos{X: 10000}}
	if a := app.Plan("l", m); a.Kind != app.None || !strings.Contains(a.Note, "not lootable") {
		t.Errorf("%+v", a)
	}
	m.Drops["d2"] = &pb.LootDropState{Id: "d2", Item: "worn-cap", Pos: &pb.Pos{X: 50000}, CanLoot: true}
	if a := app.Plan("l", m); a.Kind != app.Loot || a.ID != "d2" {
		t.Errorf("the far drop that may be taken is preferred: %+v", a)
	}
}
