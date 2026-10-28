package scenario_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/content"
	"github.com/GeneStevens/seq/core"
	"github.com/GeneStevens/seq/testserver"
)

// TUI-029: the polish of the Operator's second session. These run in Thornwick
// with a level 3 warrior beside Barnaby Tuck's stall (rings, pendants, orbs).

// besideBarnaby is a level 3 warrior with crowns and whatever edit gives it, on
// the map, a metre from Barnaby Tuck.
func besideBarnaby(t *testing.T, crowns int, edit func(*core.CharacterSave)) (*scenario.World, *scenario.Player) {
	t.Helper()
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: []testserver.Account{{Name: "ann", Classed: map[string]string{"hale": "warrior"}}}})
	w.Place("ann", "hale", 49000, 40000)
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
	p.Resize(140, 45)
	return w, p
}

// C on the map opens the character view: Gear, Bags and Stats as a panel with the
// vitals strip under it. It opens on the bags, which hold the finds; Enter wears
// the item under the cursor, and on the Gear tab takes it off.
func TestCOpensTheCharacterViewAndEnterWearsAndTakesOff(t *testing.T) {
	_, p := besideBarnaby(t, 5, func(cs *core.CharacterSave) { cs.Inventory = []string{"wolf-pelt", "short-sword"} })
	p.Press("c")
	s := p.Screen(140, 45)
	for _, want := range []string{"Your character", "[Bags]", "Items carried", "Wolf Pelt", "▲ Short Sword", "HP ", "Esc or C closes"} {
		if !strings.Contains(s, want) {
			t.Errorf("the character view lacks %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "Inventory and gear") {
		t.Errorf("the old inventory screen is still there:\n%s", s)
	}
	p.Golden("character_view_bags")

	p.Press("down", "enter") // the sword is the second line
	notice(t, p, "wear Short Sword ok", false)
	p.Press("1") // the Gear tab: the cursor is on the head
	p.Press("down", "down", "down", "down", "down", "down", "down", "down")
	s = p.Screen(140, 45)
	if !strings.Contains(s, "[Gear]") || !strings.Contains(s, "> Main hand Short Sword") {
		t.Errorf("the sword is not worn in the main hand, under the cursor:\n%s", s)
	}
	p.Golden("character_view_gear")
	p.Press("enter")
	notice(t, p, "take off Short Sword ok", false)

	p.Press("3") // Stats: the sheet the server sent
	s = p.Screen(140, 45)
	for _, want := range []string{"[Stats]", "Level 3 Warrior", "Armour", "Strength"} {
		if !strings.Contains(s, want) {
			t.Errorf("the Stats tab lacks %q:\n%s", want, s)
		}
	}
	p.Press("c")
	if s := p.Screen(140, 45); strings.Contains(s, "Your character") {
		t.Errorf("C did not close the character view:\n%s", s)
	}
}

// The character sheet shows every attribute the server sent. A warrior's content
// has Strength only, so the rest are 0 in the sheet: that is the server's number.
func TestTheSheetShowsEveryAttributeTheServerSends(t *testing.T) {
	_, p := besideBarnaby(t, 0, nil)
	p.Model(func(m *model.Model) {
		if m.Self.Sheet == nil {
			t.Fatal("no sheet")
		}
		var names []string
		for _, a := range m.Self.Sheet.GetAttributes() {
			names = append(names, a.GetName())
		}
		if strings.Join(names, ",") != "strength,agility,stamina,intellect,wisdom" {
			t.Fatalf("the server sent the attributes %v", names)
		}
	})
	p.Press("c", "3")
	s := p.Screen(140, 45)
	for _, want := range []string{"Strength", "Agility", "Stamina", "Intellect", "Wisdom"} {
		if !strings.Contains(s, want) {
			t.Errorf("the Stats tab lacks %s:\n%s", want, s)
		}
	}
}

// Every item at Barnaby's stall carries its mark, including the ones that cannot
// be afforded, and its card says what it would change: a ring is "over an empty
// ring" (the server's slot is ring1), a pendant over an empty neck; an orb is for
// a caster (D80), so a warrior's mark is x and its card says who may use it.
func TestEveryVendorItemHasItsMarkAndComparison(t *testing.T) {
	_, p := besideBarnaby(t, 5, nil)
	p.Press("g", "1")
	s := p.Screen(140, 45)
	for _, want := range []string{"▲ Tin Ring", "▲ Brass Ring", "▲ Silver Ring", "▲ Cord Pendant", "▲ Horn Pendant", "x Carved Orb", "x Rune Orb", "short 7", "short 79"} {
		if !strings.Contains(s, want) {
			t.Errorf("the stall lacks %q (a mark on every row, afforded or not):\n%s", want, s)
		}
	}
	at := 0
	for _, c := range []struct {
		idx        int
		item, want string
	}{
		{0, "tin-ring", "upgrade over an empty ring: +10 health, +1 stamina"},
		{3, "cord-pendant", "upgrade over an empty neck: +6 health"},
		{8, "rune-orb", "can't use: wizard, cleric, druid only"}, // D80: orbs are for casters
	} {
		for ; at < c.idx; at++ {
			p.Press("down")
		}
		s := p.Screen(140, 45)
		if !strings.Contains(s, c.want) || strings.Contains(s, "ring1") {
			t.Errorf("the card of %s lacks %q:\n%s", c.item, c.want, s)
		}
		if c.item == "tin-ring" {
			p.Golden("vendor_ring_comparison")
		}
	}
}

// toStockOf moves the cursor of the open vendor window to the item.
func toStockOf(t *testing.T, p *scenario.Player, npc, item string) {
	t.Helper()
	idx := -1
	p.Model(func(m *model.Model) {
		for _, n := range m.NPCs {
			if n.GetId() == npc {
				for i, s := range n.GetStock() {
					if s.GetItem() == item {
						idx = i
					}
				}
			}
		}
	})
	if idx < 0 {
		t.Fatalf("%s does not sell %s", npc, item)
	}
	for range idx {
		p.Press("down")
	}
}

// Nearby gives each person one line, the title beside the name in dim, so that
// several fit where one and "+17 more" did.
func TestNearbyIsOneLinePerEntry(t *testing.T) {
	_, p := besideBarnaby(t, 0, nil)
	p.Resize(140, 45)
	lines := strings.Split(p.Screen(140, 45), "\n")
	var entries []string
	in := false
	for _, l := range lines {
		if strings.Contains(l, "Nearby") {
			in = true
			continue
		}
		if !in {
			continue
		}
		if i := strings.Index(l, "││"); i >= 0 {
			l = l[i+len("││"):]
		}
		if strings.Contains(l, "└") {
			break
		}
		if strings.Contains(l, " more") {
			continue
		}
		if re := regexp.MustCompile(`^ \S .*\d+m [NSEW]{1,2}`); re.MatchString(strings.TrimRight(l, "│ ")) {
			entries = append(entries, l)
		}
	}
	if len(entries) < 4 {
		t.Errorf("only %d entries fit in Nearby:\n%s", len(entries), strings.Join(lines, "\n"))
	}
	titled := 0
	for _, e := range entries {
		if strings.Contains(e, "<") && strings.Contains(e, ">") {
			titled++
		}
	}
	if titled == 0 {
		t.Errorf("no entry has its title beside the name:\n%s", strings.Join(entries, "\n"))
	}
	p.Golden("nearby_one_line")
}

// The contract board's help names X (give up), and a held scout contract reads in
// words where to go.
func TestTheBoardHelpNamesXAndAScoutReadsInWords(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})
	p := w.NewCharacter("ann") // no recording: thornwick_scout.jsonl is the scout test's own
	makeWarrior(p, "Hale")
	toTheBoard(w, p)
	p.Press("g", "1")
	s := p.Screen(140, 45)
	if !strings.Contains(s, "Tab offered / held · Enter take / hand in · X give up · Esc back") {
		t.Errorf("the board's help lacks X:\n%s", s)
	}
	for range offered(t, p, "board", "Word from Wickett Mill") {
		p.Press("down")
	}
	p.Press("enter")
	notice(t, p, "take Word from Wickett Mill ok", false)
	w.Advance(time.Second) // the server records the take on its next tick
	p.Press("tab")
	s = p.Screen(140, 45)
	if !strings.Contains(s, "0/1 — go to the marker") {
		t.Errorf("the held scout contract does not read in words:\n%s", s)
	}
	if strings.Contains(s, " go\n") || regexp.MustCompile(`Mi… +go +│`).MatchString(s) {
		t.Errorf("the scout still reads \"go\":\n%s", s)
	}
	p.Golden("board_held_scout")
}
