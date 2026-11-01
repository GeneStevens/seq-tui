package scenario_test

import (
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/content"
	"github.com/GeneStevens/seq/core"
	"github.com/GeneStevens/seq/testserver"
)

// D78: the service screens keep the character in view. These run in Thornwick at
// Dunstan Reeve's weapon stall, with a level 3 warrior with bare hands.

// atTheWeaponStall is a level 3 warrior with crowns beside Dunstan Reeve, and
// whatever the edit gives it.
func atTheWeaponStall(t *testing.T, crowns int, edit func(*core.CharacterSave)) (*scenario.World, *scenario.Player) {
	t.Helper()
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: []testserver.Account{{Name: "ann", Classed: map[string]string{"hale": "warrior"}}}})
	w.Place("ann", "hale", 28000, 51500)
	z, _ := content.Zone(scenario.Thornwick)
	w.Srv.Give("hale", func(cs *core.CharacterSave) {
		cl := z.Class("warrior")
		cs.Stats, cs.Known, cs.Slots, cs.XP = core.ClassCharacter(cl, *z.CharacterTemplate("warrior"), 3)
		cs.Known, cs.Slots = cl.KnownAt(1), core.DefaultSlots(cl.KnownAt(1))
		cs.Crowns = crowns
		if edit != nil {
			edit(cs)
		}
	})
	p := w.Login("ann", "hale")
	p.Press("g", "1") // talk to Dunstan, open the weapon stall
	return w, p
}

// row moves the vendor's cursor down to the item with that name in the stock.
func toStock(t *testing.T, p *scenario.Player, item string) {
	t.Helper()
	idx := -1
	p.Model(func(m *model.Model) {
		for _, n := range m.NPCs {
			if n.GetId() == "vendor-weapons" {
				for i, s := range n.GetStock() {
					if s.GetItem() == item {
						idx = i
					}
				}
			}
		}
	})
	if idx < 0 {
		t.Fatalf("the stall does not sell %s", item)
	}
	for range idx {
		p.Press("down")
	}
}

// The vendor screen with the sidebar: a Short Sword is ▲ over bare hands, the
// comparison card says by how much, what cannot be afforded is greyed out with
// how far short, and the Stats tab gives the swing with bare hands.
func TestVendorSidebarAndComparisonBareHandsAgainstAShortSword(t *testing.T) {
	_, p := atTheWeaponStall(t, 30, nil)
	p.Resize(140, 45)
	s := p.Screen(140, 45)
	for _, want := range []string{"Traders", "[Gear]", "Your character", "Main hand", "▲ Short Sword", "short 15"} {
		if !strings.Contains(s, want) {
			t.Errorf("the vendor screen lacks %q:\n%s", want, s)
		}
	}
	toStock(t, p, "short-sword")
	s = p.Screen(140, 45)
	for _, want := range []string{"▲ Short Sword: Common, Main hand", "upgrade over an empty main hand", "damage/s", "costs 60 crowns"} {
		if !strings.Contains(s, want) {
			t.Errorf("the comparison card lacks %q:\n%s", want, s)
		}
	}
	p.Golden("vendor_sidebar_comparison")

	// A flanged mace is dearer than the 30 crowns: the card says how far short.
	p.Press("down", "down", "down")
	s = p.Screen(140, 45)
	if !strings.Contains(s, "Flanged Mace") || !strings.Contains(s, "short 80 crowns") {
		t.Errorf("an item that cannot be afforded does not say how far short:\n%s", s)
	}

	// The Stats tab: the swing with bare hands, and what gear adds.
	p.Press("c", "c")
	if _, _, _, _, tab, _ := p.Mouse(); tab != 2 {
		t.Fatalf("C twice is tab %d, want Stats (2)", tab)
	}
	s = p.Screen(140, 45)
	for _, want := range []string{"[Stats]", "Fists: ", "–", "/ 2.0s", "damage/s", "Armour", "Strength"} {
		if !strings.Contains(s, want) {
			t.Errorf("the Stats tab lacks %q:\n%s", want, s)
		}
	}
	p.Golden("sidebar_stats_bare_hands")

	// Clicking a tab picks it.
	x, y := tabCell(t, p, 140, 45, "Bags")
	p.Click(x, y)
	if _, _, _, _, tab, _ := p.Mouse(); tab != 1 {
		t.Errorf("clicking Bags gave tab %d, want 1", tab)
	}
}

// tabCell finds a sidebar tab's name on the screen.
func tabCell(t *testing.T, p *scenario.Player, w, h int, name string) (x, y int) {
	t.Helper()
	for row, l := range strings.Split(p.Screen(w, h), "\n") {
		if i := strings.Index(l, name); i >= 0 && strings.Contains(l, "Gear") && strings.Contains(l, "Stats") {
			return len([]rune(l[:i])) + 1, row
		}
	}
	t.Fatalf("no tab %q on the screen", name)
	return
}

// In an 80x24 window there is no room beside the screen: C holds the sidebar as a
// drawer, over the vendor's screen, and puts it away again without leaving it.
func TestTheSidebarIsADrawerInANarrowWindow(t *testing.T) {
	_, p := atTheWeaponStall(t, 45, nil)
	s := p.Screen(80, 24)
	if !strings.Contains(s, "C: your character") || strings.Contains(s, "[Gear]") {
		t.Errorf("a narrow window shows the vendor and a hint for C:\n%s", s)
	}
	p.Press("c")
	s = p.Screen(80, 24)
	if !strings.Contains(s, "Your character") || !strings.Contains(s, "[Gear]") || !strings.Contains(s, "HP ") {
		t.Errorf("the drawer is not open, or lost the vitals strip:\n%s", s)
	}
	p.Golden("sidebar_drawer")
	p.Press("tab") // the drawer's own keys
	if _, _, _, _, tab, drawer := p.Mouse(); tab != 1 || !drawer {
		t.Errorf("Tab in the drawer: tab %d, drawer %v", tab, drawer)
	}
	p.Press("c")
	s = p.Screen(80, 24)
	if !strings.Contains(s, "Traders") || strings.Contains(s, "[Bags]") {
		t.Errorf("C did not put the drawer away:\n%s", s)
	}
	// Still in the vendor's window: Enter buys the item under the cursor.
	p.Press("enter")
	notice(t, p, "buy ", false)
}

// Stacks, sell all junk and the stack sale: identical slotless items share a line
// ("Wolf Pelt ×3"); J sells every one a contract does not want; A sells the stack
// under the cursor.
func TestStacksAndSellingAllTheJunk(t *testing.T) {
	_, p := atTheWeaponStall(t, 0, func(cs *core.CharacterSave) {
		cs.Inventory = []string{"wolf-pelt", "wolf-pelt", "wolf-pelt", "short-sword"}
	})
	p.Resize(140, 45)
	s := p.Screen(140, 45)
	for _, want := range []string{"Wolf Pelt ×3", "▲ Short Sword", "pays 1"} {
		if !strings.Contains(s, want) {
			t.Errorf("the vendor screen lacks %q:\n%s", want, s)
		}
	}
	p.Press("tab") // to the list carried
	p.Press("a")   // sell the stack under the cursor: the pelts
	notice(t, p, "sell Wolf Pelt ok", false)
	items, crowns := inventoryOf(p)
	if len(items) != 1 || model.ItemID(items[0]) != "short-sword" || crowns != 3 {
		t.Fatalf("A sold the stack: %q, %d crowns (want the sword and 3 crowns)", items, crowns)
	}

	// A second run: one key sells all the junk and leaves the sword.
	_, q := atTheWeaponStall(t, 0, func(cs *core.CharacterSave) {
		cs.Inventory = []string{"wolf-pelt", "short-sword", "wolf-pelt"}
	})
	q.Press("J")
	notice(t, q, "sell all junk ok", false)
	items, crowns = inventoryOf(q)
	if len(items) != 1 || model.ItemID(items[0]) != "short-sword" || crowns != 2 {
		t.Fatalf("J sold the junk: %q, %d crowns (want the sword and 2 crowns)", items, crowns)
	}
	q.Model(func(m *model.Model) {
		if !hasLog(m, "reward", "junk") {
			t.Errorf("the log does not say what was sold: %+v", m.Log[len(m.Log)-3:])
		}
	})
	q.Golden("sold_all_junk")
}

// Buying a dirk and wearing it: the Stats tab names the weapon and says what gear
// adds to an attribute, and the Short Sword's mark follows what is worn.
func TestWearingGearChangesTheSheetAndTheMarks(t *testing.T) {
	w, p := atTheWeaponStall(t, 100, nil)
	p.Resize(140, 45)
	toStock(t, p, "dirk")
	p.Press("enter")
	notice(t, p, "buy Dirk ok", false)
	p.Press("esc", "esc")
	p.Press("c") // the inventory screen: the dirk is the only item
	p.Press("enter")
	notice(t, p, "wear Dirk ok", false)
	p.Press("esc")
	w.Advance(time.Second)
	p.Press("g", "1")
	p.Press("c", "c")
	s := p.Screen(140, 45)
	for _, want := range []string{"Dirk: ", "gear)", "Agility"} {
		if !strings.Contains(s, want) {
			t.Errorf("the sheet after wearing a dirk lacks %q:\n%s", want, s)
		}
	}
	p.Press("c")
	p.Model(func(m *model.Model) {
		if m.Mark("short-sword") == "" {
			t.Error("the Short Sword has no mark after the dirk was worn")
		}
	})
}
