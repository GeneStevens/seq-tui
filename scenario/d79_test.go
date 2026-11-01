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

// D79 in the TUI: the Warren's Venture delve is given a group tier and a Stir
// rule here (no pack declares one yet), and three accounts play it: ann and bob
// open and enter it, cora, a friend in the party, finds no slot, then takes the
// one bob leaves, and ann and cora fell the first camp, each told what became
// of the account's personal reward.

const (
	stirText  = "The Warren stirs: more voices answer the Diggers' call."
	groupTier = "Venture together"
)

// stirred gives the Venture delve a group tier of two players that earns a bonus
// roll for each of them, and a Stir rule for two.
func stirred(space string, z *core.Zone) {
	z.NoCompanions = true
	for i := range z.MobTemplates {
		z.MobTemplates[i].TierBonus = true // D82: only a marked encounter pays the bonus roll
	}
	for i := range z.InstancePresets {
		if z.InstancePresets[i].ID == "venture-warren" {
			z.InstancePresets[i].GroupTier = &core.GroupTierRule{ID: "venture-together", Name: groupTier, MinAccounts: 2, Bonus: "bonus-roll"}
			z.InstancePresets[i].Stir = []core.StirRule{{MinAccounts: 2, Text: stirText}}
		}
	}
}

func TestTheGroupTierStirSlotsStatusAndEntitlements(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{scenario.Warren}, Zone: stirred, Accounts: []testserver.Account{
		{Name: "ann", Home: scenario.Warren, Classed: map[string]string{"hale": "warrior"}},
		{Name: "bob", Home: scenario.Warren, Classed: map[string]string{"bram": "cleric"}},
		{Name: "cass", Home: scenario.Warren, Classed: map[string]string{"cora": "warrior"}}}})
	w.Place("ann", "hale", 11000, 70000)
	w.Place("bob", "bram", 11000, 69000)
	w.Place("cass", "cora", 11000, 68000)
	w.Srv.Give("hale", levelUp(12, hardy))
	w.Srv.Give("bram", levelUpAs("cleric", 12, hardy))
	w.Srv.Give("cora", levelUp(12, hardy))
	ann := w.Login("ann", "hale")
	bob := w.Login("bob", "bram")
	cass := w.Login("cass", "cora")
	has := func(p *scenario.Player, f func(*model.Model) bool) bool {
		ok := false
		p.Model(func(m *model.Model) { ok = f(m) })
		return ok
	}

	w.Step("two accounts open the delve and go in: the tier, the Stir text and the slots", func() {
		command(ann, "/invite bram")
		w.Settle()
		bob.Press("y")
		w.Settle()
		for _, p := range []*scenario.Player{ann, bob} {
			p.Eventually("the party of two", func(m *model.Model) bool { return len(m.PartyMembers()) == 2 })
		}
		w.Advance(time.Second)
		openPreset(t, bob, "venture-warren", true)
		bob.Eventually("the delve is open for two players", func(m *model.Model) bool {
			return m.Opened != nil && m.Opened.Players == 2 && m.Opened.GroupTier == "venture-together" && m.Opened.Slots == 2
		})
		w.Advance(time.Second)
		bob.Press("e")
		bob.Eventually("bram in the delve", inside)
		w.Advance(500 * time.Millisecond)
		ann.Press("e")
		ann.Eventually("hale in the delve", inside)
		ann.Eventually("told what the delve did", func(m *model.Model) bool { return len(m.Stirred()) == 1 })
		w.Advance(time.Second)
		ann.Model(func(m *model.Model) {
			if m.Players() != 2 || m.GroupTierName() == "" || m.Stirred()[0] != stirText || m.SlotCount() != 2 {
				t.Fatalf("players %d, tier %q, stirred %q, slots %d", m.Players(), m.GroupTierName(), m.Stirred(), m.SlotCount())
			}
		})
		screen := ann.Screen(140, 45)
		for _, want := range []string{"Venture together | 2 players", "stirred: " + stirText[:30], "Slots: 2, 2 players", "hale (you)  Player 2, you", "bram  Player 1", "replacement open; no slot is empty"} {
			if !strings.Contains(screen, want) {
				t.Errorf("hale's screen lacks %q:\n%s", want, screen)
			}
		}
		ann.Golden("d79_stirred_slots")
	})

	w.Step("a friend finds the delve full; /status is a word and nothing more", func() {
		command(ann, "/invite cora")
		w.Settle()
		cass.Press("y")
		w.Settle()
		cass.Eventually("in the party of three", func(m *model.Model) bool { return len(m.PartyMembers()) == 3 })
		w.Advance(time.Second)
		cass.Press("e")
		cass.Eventually("refused: every slot is held", func(m *model.Model) bool {
			return m.Refusal != nil && m.Refusal.Reason == "no_slot" && strings.Contains(m.Refusal.Text, "taken")
		})
		if n, bad := cass.Notice(); !bad || n == "" {
			t.Errorf("the status line says %q (denied %v)", n, bad)
		}
		cass.Golden("d79_refused_full")

		command(ann, "/status away")
		ann.Eventually("hale is away", func(m *model.Model) bool { return m.Status == model.StatusAway })
		bob.Eventually("bram sees hale away", func(m *model.Model) bool {
			for _, mt := range m.Mates() {
				if mt.Status == model.StatusAway {
					return true
				}
			}
			return false
		})
		notice(t, ann, "no effect", false)
		ann.Golden("d79_status_away")
		if scr := bob.Screen(140, 45); !strings.Contains(scr, "[away]") {
			t.Errorf("bram's Party panel does not mark hale away:\n%s", scr)
		}
		command(ann, "/status clear")
		ann.Eventually("hale is back", func(m *model.Model) bool { return m.Status == "" })
		command(ann, "/status sleeping")
		notice(t, ann, "usage: /status", true)
	})

	w.Step("a replacement takes the slot bob leaves", func() {
		bob.Press("e")
		w.Advance(time.Second)
		bob.Eventually("bram out of the delve", func(m *model.Model) bool {
			return !m.InInstance() && m.Ready() && m.Map != nil && m.Map.GetSpace() == m.Space
		})
		w.Advance(time.Second)
		ann.Eventually("bram's slot is empty, open to a replacement", func(m *model.Model) bool {
			for _, s := range m.Slots() {
				if s.State == model.HolderEmpty && s.Player == "Player 1" {
					return true
				}
			}
			return false
		})
		if scr := ann.Screen(140, 45); !strings.Contains(scr, "empty  was Player 1's, open") {
			t.Errorf("hale's screen does not show the empty slot:\n%s", scr)
		}
		cass.Press("e")
		cass.Eventually("cora in the delve", inside)
		w.Advance(time.Second)
		ann.Eventually("cora holds the slot", func(m *model.Model) bool {
			for _, s := range m.Slots() {
				if s.Holder == "cora" && s.State == model.HolderHeld && s.Player == "Player 1" {
					return true
				}
			}
			return false
		})
		// The replacement's panel carries the tier and the Stir state of the instance
		// it joined (D82); the server's snapshot says them.
		if scr := cass.Screen(140, 45); !strings.Contains(scr, "Venture together | 2 players") || !strings.Contains(scr, "stirred: "+stirText[:30]) {
			t.Errorf("cora's panel lacks the tier and the Stir:\n%s", scr)
		}
		w.Advance(time.Second)
		if has(cass, func(m *model.Model) bool { return m.Refusal != nil }) {
			t.Error("cora's refusal outlived her going in")
		}
		ann.Golden("d79_replacement_in")
	})

	w.Step("one expedition for each account", func() {
		openPreset(t, bob, "venture-warren", true)
		bob.Eventually("refused: bob is on an expedition", func(m *model.Model) bool {
			return m.Refusal != nil && strings.Contains(m.Refusal.Text, "already on an expedition")
		})
		bob.Golden("d79_refused_expedition")
	})

	w.Step("the first camp falls and each account is told what became of its reward", func() {
		from := [2]int64{11000, 70000}
		walkCamp(t, w, &from, warrenCamps[0], ann, cass)
		killCamp(t, w, warrenCamps[0].id, ann, cass)
		w.Advance(time.Second)
		for _, p := range []*scenario.Player{ann, cass} {
			p.Eventually("told the bonus roll", func(m *model.Model) bool {
				for _, l := range m.Log {
					if l.Kind == "reward" && strings.HasPrefix(l.Text, "Your account's bonus roll for ") {
						return true
					}
				}
				return false
			})
		}
		cass.Eventually("told, by name, whose roll it was", func(m *model.Model) bool {
			for _, l := range m.Log {
				if strings.Contains(l.Text, "hale used their own bonus roll for this encounter") {
					return true
				}
			}
			return false
		})
		ann.Golden("d79_entitlement")
	})
}
