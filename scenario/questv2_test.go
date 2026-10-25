package scenario_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/core"
	"github.com/GeneStevens/seq/testserver"
)

// Quests v2 (D105) on the Testgrounds, with a chain laid over it by hand (seq-world
// has no such chain yet): watch the Hollow Lookout, hand two relics to the captain,
// tell the Eastern Binder. The character holds three relics, so one is left to go
// when the chain ends.

var relic = core.ItemDef{ID: "relic", Name: "Old Relic", Tier: core.TierCommon, Binding: core.BindTradeOnce, QuestItem: true}

var relicChain = core.QuestDef{ID: "relics", Name: "The Old Relic", Giver: "captain", Steps: []core.QuestStep{
	{Kind: core.StepWatch, Place: "lookout", RadiusMM: 5000, DurationMs: 3000, Text: "Watch from the Hollow Lookout."},
	{Kind: core.StepDeliver, NPC: "captain", Item: "relic", Count: 2, Text: "Bring 2 old relics to the Testgrounds Captain.", Reward: core.ContractReward{Crowns: 7}},
	{Kind: core.StepTalkTo, NPC: "binder-east", Text: "Tell the Eastern Binder it is done."},
}}

func relicWorld(chain core.QuestDef) func(string, *core.Zone) {
	return func(_ string, z *core.Zone) {
		if z.ID != "testgrounds" {
			return
		}
		z.Items = append(slices.Clip(z.Items), relic)
		z.Quests = []core.QuestDef{chain}
		for i := range z.MobTemplates {
			z.MobTemplates[i].AggroRangeMM = 0 // quiet: nothing aggroes on the watch
		}
	}
}

func relicStep(t *testing.T, p *scenario.Player, step int64, kind string, progress int64) {
	t.Helper()
	p.Model(func(m *model.Model) {
		q := m.Quest()
		if q.GetStep() != step || q.GetKind() != kind || q.GetProgress() != progress {
			t.Fatalf("the quest is step %d %q progress %d; want step %d %q progress %d (%v)", q.GetStep(), q.GetKind(), q.GetProgress(), step, kind, progress, q)
		}
	})
}

func TestAWatchADeliverAndAQuestItemRemovedOnCompletion(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "hale"), Zone: relicWorld(relicChain)})
	w.Fixture("quest_v2.jsonl")
	w.Place("ann", "hale", 35000, 8000)
	w.Srv.Give("hale", func(cs *core.CharacterSave) {
		cs.Quests = []core.QuestState{{ID: "relics"}}
		cs.Inventory = append(cs.Inventory, "relic", "relic", "relic")
	})
	p := w.Login("ann", "hale")

	w.Step("the watch is on, and the quest tells it", func() {
		relicStep(t, p, 1, "watch", 0)
		if s := p.Screen(140, 45); !strings.Contains(s, "Watching… 0 of 3 s") {
			t.Fatalf("the Quest chain panel does not show the watch:\n%s", s)
		}
		p.GoldenWithout("quest_v2_watch", arrival)
		p.Press("c") // the bags, with the relics tagged
		if s := p.Screen(140, 45); !strings.Contains(s, "Old Relic ×3 Common [quest]") {
			t.Fatalf("the bags do not tag the relics:\n%s", s)
		}
		p.Press("esc")
	})

	w.Step("the watch runs to its end on the spot", func() {
		w.Advance(4 * time.Second)
		p.Model(func(m *model.Model) {
			if q := m.Quest(); q.GetStep() != 2 || q.GetKind() != "deliver" || q.GetNeed() != 2 {
				t.Fatalf("after the watch: %v", q)
			}
			if !hasLog(m, "quest", "step 1 done") {
				t.Fatalf("the log does not say the watch is done: %v", m.Log)
			}
		})
	})

	w.Step("the hand-in takes the relics and the quest item leaves the bags", func() {
		walkVia(t, w, p, [2]int64{20000, 8000}, [2]int64{20000, 57000})
		if s := p.Screen(140, 45); !strings.Contains(s, "Held: 2 of 2 Old Relic") { // it holds 3; the step counts what it needs
			t.Fatalf("the deliver step does not count the relics in the bags:\n%s", s)
		}
		p.GoldenWithout("quest_v2_deliver", arrival)
		talkTo(t, w, p, "captain")
		p.Press("esc")
		w.Advance(time.Second)
		p.Model(func(m *model.Model) {
			if q := m.Quest(); q.GetStep() != 3 || m.Self.Crowns < 7 {
				t.Fatalf("after the hand-in: %v, %d crowns", q, m.Self.Crowns)
			}
			if n := count(m.Self.Inventory, "relic"); n != 1 {
				t.Fatalf("%d relics left in the bags; want the 1 not handed in", n)
			}
			if !hasLog(m, "quest", "Quest item removed from your bags: 2 Old Relic.") {
				t.Fatalf("the log does not say the relics went: %v", m.Log)
			}
		})
	})

	w.Step("the last talk ends the chain and the spare relic goes", func() {
		walkVia(t, w, p, [2]int64{20000, 20000}, [2]int64{119000, 20000})
		talkTo(t, w, p, "binder-east")
		p.Press("esc")
		w.Advance(time.Second)
		p.Model(func(m *model.Model) {
			if m.Quest() != nil || count(m.Self.Inventory, "relic") != 0 {
				t.Fatalf("after the chain: %v, bags %v", m.Quest(), m.Self.Inventory)
			}
			if !hasLog(m, "quest", "Quest complete: The Old Relic") || !hasLog(m, "quest", "Quest item removed from your bags: 1 Old Relic.") {
				t.Fatalf("the log does not say it ended and the relic went: %v", m.Log)
			}
		})
		p.GoldenWithout("quest_v2_done", arrival)
	})
}

func count(list []string, s string) int {
	n := 0
	for _, x := range list {
		if x == s {
			n++
		}
	}
	return n
}
