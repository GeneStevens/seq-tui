package scenario_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq-tui/screen"
	"github.com/GeneStevens/seq/content"
	"github.com/GeneStevens/seq/core"
	"github.com/GeneStevens/seq/testserver"
)

// The scenarios of items, vendors and trade (SEQ-040, SEQ-042). Each presses
// its own keys, as a player does.

func inventoryOf(p *scenario.Player) (items []string, crowns int64) {
	p.Model(func(m *model.Model) { items, crowns = slices.Clone(m.Self.Inventory), m.Self.Crowns })
	return
}

func indexOf(t *testing.T, list []string, want string) int {
	t.Helper()
	i := slices.IndexFunc(list, func(s string) bool { return model.ItemID(s) == want })
	if i < 0 {
		t.Fatalf("%q is not in %q", want, list)
	}
	return i
}

// openTrader presses N and walks the list to the trader with that name, then
// opens its window.
func openTrader(t *testing.T, p *scenario.Player, name string) {
	t.Helper()
	idx := -1
	p.Model(func(m *model.Model) {
		for i, n := range screen.TradersInView(m) {
			if n.GetName() == name {
				idx = i
			}
		}
	})
	if idx < 0 {
		t.Fatalf("no trader called %q in view", name)
	}
	p.Press("n")
	for range idx {
		p.Press("down")
	}
	p.Press("enter")
}

// Kill the trophy scuttler, take its coin and its two drops with L, then wear
// them from the inventory screen. A bystander in view of the kill sees the Rare
// drop as one it may not take, and L says so without asking the server.
func TestEquipALootedItem(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds},
		Accounts: []testserver.Account{{Name: "ann", Characters: []string{"c1"}}, {Name: "bob", Characters: []string{"c2"}}}})
	w.Fixture("loot_equip.jsonl")
	w.Place("ann", "c1", 98000, 50000)
	w.Place("bob", "c2", 100000, 66000)
	w.Srv.Attend("ann")
	p := w.Login("ann", "c1")
	b := w.Login("bob", "c2")

	attack(p)
	for i := 0; i < 60; i++ {
		var drops int
		p.Model(func(m *model.Model) { drops = len(m.Drops) })
		if drops >= 3 {
			break
		}
		w.Advance(time.Second)
	}
	b.Eventually("the bystander sees the drops", func(m *model.Model) bool { return len(m.Drops) >= 3 })
	// The drop event describes the item (D52): its tier, binding and stats are
	// known at once, before it is looted and before any snapshot comes.
	p.Model(func(m *model.Model) {
		it := m.Item("scuttler-blade")
		if it == nil || it.GetTier() != "rare" || !it.GetBindOnPickup() || it.GetStats() == nil || !hasLog(m, "loot_dropped", "Scuttler Blade [Rare, bound on pickup]") {
			t.Fatalf("the dropped blade is described as %v", it)
		}
	})
	b.Model(func(m *model.Model) {
		for id, d := range m.Drops {
			if d.GetCanLoot() {
				t.Fatalf("the bystander may take %s", id)
			}
		}
	})
	b.Press("l")
	notice(t, b, "not lootable", false)
	b.Golden("loot_not_yours")

	// The participant takes all three with one L (loot-all, D78), as one log line.
	p.Press("l")
	items, crowns := inventoryOf(p)
	if len(items) != 2 || crowns != 5 {
		t.Fatalf("after looting: %q, %d crowns", items, crowns)
	}
	p.Model(func(m *model.Model) {
		if len(m.Drops) != 0 || !hasLog(m, "reward", "You took") {
			t.Fatalf("drops %d; log ends %+v", len(m.Drops), m.Log[len(m.Log)-3:])
		}
	})

	// C: the character view opens on the bags, when they hold anything; Enter wears the one
	// under the cursor, as the server accepts it.
	p.Press("c")
	p.Golden("inventory_looted")
	for range indexOf(t, items, "scuttler-blade") {
		p.Press("down")
	}
	p.Press("enter")
	notice(t, p, "wear Scuttler Blade ok", false)
	p.Model(func(m *model.Model) {
		if m.Worn(8) != "scuttler-blade" || len(m.Self.Inventory) != 1 || !hasLog(m, "reward", "You equip Scuttler Blade") {
			t.Fatalf("worn %q, carried %q", m.Self.Equipped, m.Self.Inventory)
		}
	})
	p.Press("enter")
	notice(t, p, "wear Worn Cap ok", false)
	p.Press("1") // the Gear tab
	p.Golden("inventory_worn")
	p.Press("enter")
	notice(t, p, "take off Worn Cap ok", false)
	p.Model(func(m *model.Model) {
		if m.Worn(0) != "" || len(m.Self.Inventory) != 1 {
			t.Fatalf("worn %q, carried %q", m.Self.Equipped, m.Self.Inventory)
		}
	})
}

// Buy at the vendor, sell what is not wanted, train the next ability, bind at
// the binder and bank an item: every one a window of the traders' screen.
func TestBuySellTrainBindAndBank(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds},
		Accounts: []testserver.Account{{Name: "ann", Classed: map[string]string{"zedd": "wizard"}}}})
	w.Fixture("traders.jsonl")
	w.Place("ann", "zedd", 14000, 60000)
	z, _ := content.Testgrounds()
	w.Srv.Give("zedd", func(cs *core.CharacterSave) {
		cl := z.Class("wizard")
		cs.Stats, cs.Known, cs.Slots, cs.XP = core.ClassCharacter(cl, *z.CharacterTemplate("wizard"), 3)
		cs.Known, cs.Slots = cl.KnownAt(1), core.DefaultSlots(cl.KnownAt(1))
		cs.Crowns = 120
	})
	p := w.Login("ann", "zedd")
	p.Model(func(m *model.Model) {
		if m.Self.Crowns != 120 || len(m.NPCs) < 6 || m.Self.Level != 3 {
			t.Fatalf("crowns %d, %d traders, level %d", m.Self.Crowns, len(m.NPCs), m.Self.Level)
		}
	})

	// The vendor: Worn Cap is the second thing on sale.
	openTrader(t, p, "Testgrounds Vendor")
	p.Press("down")
	p.Golden("vendor")
	p.Press("enter")
	notice(t, p, "buy Worn Cap ok", false)
	items, crowns := inventoryOf(p)
	if !slices.Equal(items, []string{"worn-cap"}) || crowns != 112 {
		t.Fatalf("after buying: %q, %d crowns", items, crowns)
	}
	// Something it cannot sell is refused with the reason. Then Tab and Enter
	// sell the cap back: a vendor pays a quarter.
	p.Press("tab", "enter")
	notice(t, p, "sell Worn Cap ok", false)
	items, crowns = inventoryOf(p)
	if len(items) != 0 || crowns != 114 {
		t.Fatalf("after selling: %q, %d crowns", items, crowns)
	}
	p.Press("tab")
	for range 8 { // the Buckler is the last of the ten on sale
		p.Press("down")
	}
	p.Press("enter")
	notice(t, p, "buy Buckler ok", false)
	p.Press("esc", "esc")

	lesson := 0
	// The wizard trainer teaches Glyphshield, level 3, for 45 crowns, into the first empty slot.
	openTrader(t, p, "Wizard Trainer")
	p.Model(func(m *model.Model) {
		for i, tr := range m.NPC("trainer-wizard").GetTrains() {
			if tr.GetAbility() == "wizard-glyphshield" {
				lesson = i
			}
		}
	})
	for range lesson {
		p.Press("down")
	}
	p.Golden("trainer")
	p.Press("enter")
	notice(t, p, "train Glyphshield ok", false)
	p.Model(func(m *model.Model) {
		if m.AbilityByID("wizard-glyphshield") == nil || m.SlotOf("wizard-glyphshield") != 3 || m.Self.Crowns != 114-12-45 || !hasLog(m, "reward", "You learn Glyphshield") {
			t.Fatalf("after training: slots %s, %d crowns", slotsOf(m), m.Self.Crowns)
		}
	})
	p.Press("enter")
	notice(t, p, "You already know that", true)
	p.Press("esc", "esc")

	// The binder sets the bind point; the banker keeps the buckler.
	openTrader(t, p, "Testgrounds Binder")
	p.Press("enter")
	notice(t, p, "bind ok", false)
	p.Model(func(m *model.Model) {
		if m.Self.BindZone == "" || m.Self.Crowns != 114-12-45-20 {
			t.Fatalf("bound at %q with %d crowns", m.Self.BindZone, m.Self.Crowns)
		}
	})
	p.Press("esc", "esc")
	openTrader(t, p, "Testgrounds Banker")
	p.Press("enter")
	notice(t, p, "bank Buckler ok", false)
	p.Golden("bank")
	p.Press("tab", "enter")
	notice(t, p, "take out Buckler ok", false)
	p.Model(func(m *model.Model) {
		if len(m.Self.Bank) != 0 || len(m.Self.Inventory) != 1 {
			t.Fatalf("bank %q, carried %q", m.Self.Bank, m.Self.Inventory)
		}
	})
}

// Two characters trade: one asks and the other agrees, each puts items and
// crowns in, the Rare item is refused as bound, and both confirm. The cap
// arrives marked, bound to its new owner.
func TestTradeWithABoundItemRefused(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds},
		Accounts: []testserver.Account{{Name: "ann", Characters: []string{"c1"}}, {Name: "bob", Characters: []string{"c2"}}}})
	w.Fixture("trade.jsonl")
	w.Place("ann", "c1", 30000, 90000)
	w.Place("bob", "c2", 31000, 90000)
	w.Srv.Give("c1", func(cs *core.CharacterSave) { cs.Inventory = []string{"worn-cap", "scuttler-blade"} })
	w.Srv.Give("c2", func(cs *core.CharacterSave) { cs.Crowns = 20 })
	a := w.Login("ann", "c1")
	b := w.Login("bob", "c2")

	a.Press("t")
	a.Golden("trade_ask")
	a.Press("enter")
	notice(t, a, "ask c2 to trade ok", false)
	b.Eventually("the ask", func(m *model.Model) bool { return m.Trade != nil && m.Trade.GetIncoming() })
	b.Press("t")
	b.Golden("trade_asked")
	b.Press("enter")
	notice(t, b, "ask c1 to trade ok", false)
	a.Eventually("the window", func(m *model.Model) bool { return m.Trade != nil && m.Trade.GetOpen() })
	b.Eventually("the window", func(m *model.Model) bool { return m.Trade != nil && m.Trade.GetOpen() })

	// Ann offers the cap; the blade is bound and refused, with the reason.
	a.Press("enter")
	notice(t, a, "offer Worn Cap ok", false)
	a.Press("down", "enter")
	notice(t, a, "That item is bound to you and can't be traded", true)
	// Bob offers 7 crowns, typed.
	b.Press("c")
	b.Type("7")
	b.Press("enter")
	notice(t, b, "offer 7 crowns ok", false)
	a.Eventually("both offers", func(m *model.Model) bool {
		return m.Trade != nil && len(m.Trade.GetMine().GetItems()) == 1 && m.Trade.GetTheirs().GetCrowns() == 7
	})
	a.Golden("trade_window")

	// Each confirms; the trade applies at once, all of it.
	a.Press("y")
	notice(t, a, "confirm the trade ok", false)
	b.Eventually("ann's confirmation", func(m *model.Model) bool { return m.Trade != nil && m.Trade.GetTheirsConfirmed() })
	b.Golden("trade_confirmed")
	b.Press("y")
	b.Eventually("the cap, marked", func(m *model.Model) bool {
		return m.Trade == nil && slices.Equal(m.Self.Inventory, []string{"worn-cap!"})
	})
	a.Eventually("7 crowns", func(m *model.Model) bool { return m.Trade == nil && m.Self.Crowns == 7 })
	b.Model(func(m *model.Model) {
		if m.Self.Crowns != 13 || !hasLog(m, "trade", "is complete") || !m.Bound("worn-cap!") {
			t.Fatalf("bob has %d crowns; log ends %+v", m.Self.Crowns, m.Log[len(m.Log)-3:])
		}
	})
	b.Press("esc")
	b.Press("c", "1")
	b.Golden("inventory_traded")
}

// Opening a Peril instance takes its entry fee; the fee is in the log and the
// crowns are in the You panel. A character that cannot pay is told so.
func TestPerilEntryFee(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds},
		Accounts: []testserver.Account{{Name: "ann", Characters: []string{"c1"}}, {Name: "bob", Characters: []string{"c2"}}}})
	w.Fixture("peril.jsonl")
	w.Place("ann", "c1", gateX, gateY)
	w.Place("bob", "c2", gateX, gateY)
	w.Srv.Give("c1", func(cs *core.CharacterSave) { cs.Crowns = 80 })
	w.Srv.Give("c2", func(cs *core.CharacterSave) { cs.Crowns = 10 })
	p := w.Login("ann", "c1")
	poor := w.Login("bob", "c2")

	openPreset(t, p, "peril-den", false)
	p.Golden("peril_menu")
	p.Press("enter")
	w.Advance(time.Second)
	p.Model(func(m *model.Model) {
		if m.Opened == nil || m.Opened.Preset != "peril-den" || m.Self.Crowns != 30 || !hasLog(m, "reward", "You pay 50 crowns to enter") {
			t.Fatalf("opened %+v, %d crowns; log ends %+v", m.Opened, m.Self.Crowns, m.Log[len(m.Log)-3:])
		}
	})
	p.Golden("peril_paid")

	openPreset(t, poor, "peril-den", true)
	poor.Model(func(m *model.Model) {
		if m.Self.Crowns != 10 {
			t.Fatalf("the poor one has %d crowns", m.Self.Crowns)
		}
	})
	if got, bad := poor.Notice(); !bad || !strings.Contains(got, "can't afford") {
		// The server may answer in the event instead of the intent.
		poor.Model(func(m *model.Model) {
			if !hasLog(m, "instance_open_failed", "can't afford") && !hasLog(m, "denied", "can't afford") {
				t.Fatalf("no refusal: notice %q; log ends %+v", got, m.Log[len(m.Log)-3:])
			}
		})
	}
}

// A Wizard roots the scuttler and a fresh login, in the middle of the fight,
// shows the mob as rooted: the snapshot carries the effects on everything in
// view (D50).
func TestLoginMidFightShowsTheRootedMob(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{{Name: "ann", Classed: map[string]string{"zedd": "wizard"}}},
		Zone: both(learnAt("wizard", "wizard-root", 1), tough(400))})
	w.Fixture("mid_fight.jsonl")
	p := w.Login("ann", "zedd")
	p.Press("tab", "3")
	w.Advance(1800 * time.Millisecond)
	p.Model(func(m *model.Model) {
		if fx := m.EffectsOn(m.Self.Target); len(fx) != 1 || fx[0].Kind != "root" {
			t.Fatalf("effects on the target: %+v", fx)
		}
	})
	p.Fresh()
	p.Model(func(m *model.Model) {
		fx := m.EffectsOn(m.Self.Target)
		if m.Self.Target == "" || len(fx) != 1 || fx[0].Kind != "root" || fx[0].EndsAtMs <= m.GameTimeMs {
			t.Fatalf("after logging in again: target %q, effects %+v", m.Self.Target, fx)
		}
	})
	if s := p.Screen(80, 24); !strings.Contains(s, "Plain Scuttler: Rooted") {
		t.Fatalf("the screen does not show the rooted mob:\n%s", s)
	}
	p.Golden("login_rooted")
}

// A Wizard in a turn-based instance changes a slot with the plan: the slot
// screen adds the step, Enter submits it with the rest, and the slot holds
// from the round after.
func TestSlotsChangeBetweenRounds(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{{Name: "ann", Classed: map[string]string{"zedd": "wizard"}}},
		Zone: both(atGate, learnAt("wizard", "wizard-root", 1))})
	w.Fixture("slots_round.jsonl")
	p := w.Login("ann", "zedd")
	openPreset(t, p, "quick-turns", true)
	w.Advance(time.Second)
	p.Press("e")
	p.Eventually("round 1", openRound)

	// B, Enter on slot 1, down to Root, Enter: a step of the plan, not an intent.
	p.Press("b", "enter", "down", "down", "enter")
	if got := p.Plan(); len(got) != 1 || got[0] != "key 1: Root" {
		t.Fatalf("plan %q", got)
	}
	p.Model(func(m *model.Model) {
		if slotsOf(m) != "wizard-flame-dart,wizard-frost-bolt,wizard-root,,,,," {
			t.Fatalf("slots changed before the round: %s", slotsOf(m))
		}
	})
	p.Golden("slots_plan")
	p.Press("esc")
	p.Golden("slots_plan_round")
	p.Press("enter")
	notice(t, p, "plan accepted", false)
	w.Advance(100 * time.Millisecond)
	p.Model(func(m *model.Model) {
		if slotsOf(m) != "wizard-root,wizard-frost-bolt,wizard-root,,,,," || m.Round.Number != 2 || m.Round.PlayedBy("zedd") != "own_plan" {
			t.Fatalf("slots %s, round %+v", slotsOf(m), m.Round)
		}
	})
	p.Golden("slots_changed")
}

// Character creation reads the class menu the server sent: its classes, roles,
// descriptions and the kit each starts with. The class chosen on the screen is
// the character's, with exactly that kit in its slots.
func TestClassMenuFromTheServer(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{{Name: "ann"}}})
	w.Fixture("class_menu.jsonl")
	p := w.NewCharacter("ann")
	var menu []string
	var kit []string
	p.Model(func(m *model.Model) {
		for _, c := range m.Classes() {
			menu = append(menu, c.GetId())
		}
		if c := m.Class("druid"); c != nil {
			for _, a := range c.GetKit() {
				kit = append(kit, a.GetId())
			}
		}
	})
	if len(menu) != 6 || len(kit) == 0 {
		t.Fatalf("the class menu: %q; the druid's kit: %q", menu, kit)
	}
	p.Type("Mira")
	p.Press("tab")
	p.Press(string(rune('1' + slices.Index(menu, "druid"))))
	s := p.Screen(80, 24)
	for _, want := range []string{"Class (the server's menu)", "Druid", "healer + spell", "Starts with, free:", "Nature caster who heals over time"} {
		if !strings.Contains(s, want) {
			t.Fatalf("the class menu lacks %q:\n%s", want, s)
		}
	}
	p.Golden("class_menu")
	p.Press("enter")
	p.Created()
	p.Model(func(m *model.Model) {
		var slots []string
		for _, s := range m.Self.Slots {
			if s.Ability != "" {
				slots = append(slots, s.Ability)
			}
		}
		if m.Self.Class != "druid" || !slices.Equal(slots, kit) {
			t.Fatalf("the new character: class %q, slots %q; the menu's kit %q", m.Self.Class, slots, kit)
		}
	})
}
