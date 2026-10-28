package scenario_test

import (
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/core"
	"github.com/GeneStevens/seq/testserver"
)

// The economy of D115 in the terminal: a draught from the bags (with its
// cooldown shown and its refusal said plainly), a regen item's resting bonus,
// vault space bought on confirm, and the punt across the mere.

// give puts the items and crowns in the character's bags, at half health.
func give(w *scenario.World, id string, crowns int, items ...string) {
	w.Srv.Give(id, func(cs *core.CharacterSave) {
		cs.Inventory, cs.Crowns = items, crowns
		cs.HP = core.MaxHP(cs.Stats) / 2
	})
}

func TestUseADraughtAndALoafFromTheBags(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: warriorAccount("hale"), Zone: noSpawns})
	w.Place("ann", "hale", 60000, 45000)
	give(w, "hale", 0, "small-hearth-draught", "hearth-draught", "smoked-eel")
	p := w.Login("ann", "hale")

	// The bags tab: U on the first stack uses it, and the card tells what it does.
	p.Press("c", "2")
	p.Golden("use_bags_card")
	var hp0 int64
	p.Model(func(m *model.Model) { hp0 = m.Self.HP })
	p.Press("u")
	w.Advance(time.Second)
	notice(t, p, "use Small Hearth Draught ok", false)
	p.Model(func(m *model.Model) {
		if m.Self.HP <= hp0 {
			t.Fatalf("the draught healed nothing: %d then %d", hp0, m.Self.HP)
		}
		if m.CooldownLeftMs(m.Self.Inventory[0]) <= 0 {
			t.Fatalf("no cooldown kept after the use")
		}
	})
	p.Golden("use_draught_cooling")

	// The other draught shares the cooldown: refused plainly, before the server is asked.
	p.Press("u")
	if got, _ := p.Notice(); !strings.Contains(got, "isn't ready") {
		t.Fatalf("the cooldown is not said: %q", got)
	}

	// The eel is another cooldown group: it gives the resting bonus, shown on the main screen.
	p.Press("down", "u")
	w.Advance(time.Second)
	notice(t, p, "use Smoked Eel ok", false)
	p.Press("esc")
	p.Model(func(m *model.Model) {
		if m.RegenActive() == nil {
			t.Fatalf("no resting bonus after the eel")
		}
	})
	p.Golden("use_regen_active")

	// H uses a healing draught anywhere; with the one on cooldown it says so.
	p.Press("H")
	if got, _ := p.Notice(); !strings.Contains(got, "isn't ready") {
		t.Fatalf("H on a cooldown says %q", got)
	}
}

func TestBuyVaultSpaceOnConfirm(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: warriorAccount("hale"), Zone: noSpawns})
	w.Place("ann", "hale", 88000, 57000)
	give(w, "hale", 120)
	p := w.Login("ann", "hale")

	openTrader(t, p, "Odalys Penn")
	p.Golden("vault_offer")
	// B asks; Esc leaves it.
	p.Press("B")
	p.Golden("vault_confirm")
	p.Press("esc")
	notice(t, p, "not bought", false)
	p.Model(func(m *model.Model) {
		if m.Self.BankTiers != 0 || m.Self.Crowns != 120 {
			t.Fatalf("nothing was confirmed: %d tiers, %d crowns", m.Self.BankTiers, m.Self.Crowns)
		}
	})
	// B, then Enter buys the first tier for 100 crowns.
	p.Press("B", "enter")
	w.Advance(time.Second)
	notice(t, p, "buy bank slots ok", false)
	p.Model(func(m *model.Model) {
		if m.Self.BankTiers != 1 || m.Self.Crowns != 20 {
			t.Fatalf("after the first tier: %d tiers, %d crowns", m.Self.BankTiers, m.Self.Crowns)
		}
	})
	p.Golden("vault_bought")
	// The second tier costs 250: the server refuses and the line says why.
	p.Press("B", "y")
	w.Advance(time.Second)
	if got, bad := p.Notice(); !bad || got == "" {
		t.Fatalf("an unaffordable tier is not refused: %q", got)
	}
}

func TestTakeNellsPunt(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: warrior(scenario.Downs), Zone: noSpawns})
	w.Place("ann", "hale", 119500, 411500)
	give(w, "hale", 25)
	p := w.Login("ann", "hale")

	openTrader(t, p, "Nell Rush")
	p.Golden("punt_routes")
	p.Press("enter")
	p.Golden("punt_confirm")
	p.Press("enter")
	w.Advance(time.Second)
	p.Model(func(m *model.Model) {
		if m.Self.Crowns != 15 || m.Self.Pos.GetX() < 190000 {
			t.Fatalf("after the punt: %d crowns at %v", m.Self.Crowns, m.Self.Pos)
		}
	})
}
