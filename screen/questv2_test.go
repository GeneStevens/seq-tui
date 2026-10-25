package screen_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
	"google.golang.org/protobuf/proto"
)

// Quests v2 on the screen (D105): the quest chain screen for each step kind, the
// locked offers on a person, quest items in the bags and at a vendor. The
// goldens are testdata/quests, rewritten by `make golden`.

func questWorld(q *pb.QuestInfo) *model.Model {
	m := downsWorld(100000, 44000, q)
	m.Items["charm-bead"] = &pb.ItemInfo{Id: "charm-bead", Name: "charm bead", Tier: "common", QuestItem: true}
	m.Items["tin-ring"] = &pb.ItemInfo{Id: "tin-ring", Name: "Tin Ring", Slot: "finger", SellPrice: 3}
	if q.GetItem() != "" { // a deliver or collect step counts what the bags hold
		for range q.GetProgress() {
			m.Self.Inventory = append(m.Self.Inventory, q.GetItem())
		}
	}
	return m
}

var questSteps = map[string]*pb.QuestInfo{
	"deliver": {Id: "beads", Name: "Beads for Barnaby", Step: 3, Steps: 4, Kind: "deliver", Item: "charm-bead", Need: 3, Progress: 2, Unit: "items",
		Text: "Bring 3 charm beads to Barnaby Tuck.", TargetKind: "npc", Target: "barnaby", TargetName: "Barnaby Tuck", TargetPos: &pb.Pos{X: 104000, Y: 40000},
		StepReward: &pb.RewardInfo{Xp: 40}},
	"kill": {Id: "wolves", Name: "Wolves at the Fold", Step: 2, Steps: 3, Kind: "kill", Need: 5, Progress: 2, Unit: "kills",
		Text: "Drive off the hill wolves near Highfold.", TargetKind: "place", Target: "highfold", TargetName: "Highfold", TargetPos: &pb.Pos{X: 100000, Y: 50000}},
	"collect": {Id: "reeds", Name: "Reeds for the Roof", Step: 1, Steps: 3, Kind: "collect", Item: "charm-bead", Need: 4, Progress: 1, Unit: "items",
		Text: "Gather 4 charm beads from the Eel Mere shore."},
	"watch": {Id: "watch", Name: "The Shepherds' Watch", Step: 2, Steps: 4, Kind: "watch", Need: 20, Progress: 12, Unit: "seconds",
		Text: "Watch the shore from the green.", TargetKind: "place", Target: "highfold", TargetName: "Highfold", TargetPos: &pb.Pos{X: 100000, Y: 50000}},
	"zone": {Id: "ferry", Name: "A Boat for the Ferryman", Step: 4, Steps: 5, Kind: "talk_to", Zone: "thornwick", ZoneName: "Thornwick",
		Text: "Take the news to Captain Hartley.", TargetKind: "zone", Target: "thornwick", TargetName: "Thornwick", TargetPos: &pb.Pos{X: 6000, Y: 225000}},
}

func TestQuestV2Goldens(t *testing.T) {
	cases := []struct {
		name  string
		q     string
		set   func(m *model.Model, v *screen.View)
		wants []string
	}{
		{"quest_deliver", "deliver", func(m *model.Model, v *screen.View) { v.Quest = &screen.QuestUI{} }, []string{"Bring 3 charm beads to Barnaby Tuck.", "Held: 2 of 3 charm bead", "(npc: Barnaby Tuck)", "This step pays 40 XP"}},
		{"quest_kill", "kill", func(m *model.Model, v *screen.View) { v.Quest = &screen.QuestUI{} }, []string{"Kills: 2 of 5"}},
		{"quest_collect", "collect", func(m *model.Model, v *screen.View) { v.Quest = &screen.QuestUI{} }, []string{"Collected: 1 of 4 charm bead"}},
		{"quest_watch", "watch", func(m *model.Model, v *screen.View) { v.Quest = &screen.QuestUI{} }, []string{"Watching… 12 of 20 s"}},
		{"quest_watch_reset", "watch", func(m *model.Model, v *screen.View) {
			v.Quest = &screen.QuestUI{}
			cut := proto.Clone(questSteps["watch"]).(*pb.QuestInfo)
			cut.Progress = 0
			snapQuest(m, cut)
		}, []string{"Watching… 0 of 20 s", "started over at 12 s", "you left the spot"}},
		{"quest_zone", "zone", func(m *model.Model, v *screen.View) { v.Quest = &screen.QuestUI{} }, []string{"This step is in another zone: Thornwick.", "Take the Town Gate: 204m NW of you"}},
		{"quest_panel_watch", "watch", func(m *model.Model, v *screen.View) {}, []string{"Watching… 12 of 20 s"}},
		{"quest_panel_zone", "zone", func(m *model.Model, v *screen.View) {}, []string{"in another zone:"}},
		{"quest_offers_locked", "kill", func(m *model.Model, v *screen.View) {
			m.NPCs = []*pb.NpcInfo{{Id: "reeve", Name: "Maud Fenner", Title: "Highfold Reeve", Role: "quest_giver", Pos: &pb.Pos{X: 100000, Y: 45000}, ReachMm: 3000, QuestOffers: []*pb.QuestOfferInfo{
				{Id: "lost-beads", Name: "The Lost Beads", Locked: "level", MinLevel: 12, Text: "Reach level 12"},
				{Id: "wolf-hunt", Name: "Wolves at the Fold", Locked: "requires", Text: "Finish first: The Lost Beads"},
				{Id: "new-fold", Name: "A New Fold"},
			}}}
			m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: 2, GameTimeMs: 2000, Body: &pb.Event_Dialogue{Dialogue: &pb.Dialogue{Actor: "me", Npc: "reeve", Lines: []string{"The wolves are bold this year."}}}}}})
			v.Talk = &screen.Talk{NPC: "reeve"}
		}, []string{"[locked] The Lost Beads: needs level 12", "[locked] Wolves at the Fold: finish The Lost Beads first", "[new] A New Fold"}},
		{"quest_items_bags", "deliver", func(m *model.Model, v *screen.View) {
			m.Self.Inventory = []string{"charm-bead", "charm-bead", "tin-ring"}
			v.Char, v.Side = true, &screen.Sidebar{Tab: screen.TabBags}
		}, []string{"charm bead ×2 Common [quest]", "Tin Ring"}},
		{"quest_items_vendor", "deliver", func(m *model.Model, v *screen.View) {
			m.Self.Inventory = []string{"charm-bead", "charm-bead", "tin-ring"}
			m.NPCs = []*pb.NpcInfo{{Id: "vendor", Name: "Tilda", Title: "Provisioner", Role: "vendor", Pos: &pb.Pos{X: 100000, Y: 45000}, ReachMm: 3000}}
			v.Traders = &screen.Traders{Open: true, NPC: "vendor", Pane: 1}
		}, []string{"charm bead ×2", "[quest]", "no sale"}},
	}
	for _, c := range cases {
		for _, sz := range [][2]int{{80, 24}, {120, 40}} {
			m := questWorld(questSteps[c.q])
			v := view()
			c.set(m, &v)
			got := screen.Render(m, v, sz[0], sz[1], screen.Plain()) + "\n"
			if sz[0] > 80 || !strings.HasPrefix(c.name, "quest_panel") {
				for _, want := range c.wants {
					if !strings.Contains(got, want) {
						t.Errorf("%s at %v lacks %q:\n%s", c.name, sz, want, got)
					}
				}
			}
			file := filepath.Join("../testdata/quests", c.name+"_"+strconv.Itoa(sz[0])+"x"+strconv.Itoa(sz[1])+".txt")
			if os.Getenv("SEQ_GOLDEN") == "1" {
				if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			want, err := os.ReadFile(file)
			if err != nil {
				t.Errorf("%s: %v (run make golden)", file, err)
				continue
			}
			if got != string(want) {
				t.Errorf("%s differs from its golden (make golden rewrites it after review)\n--- got\n%s--- want\n%s", file, got, want)
			}
		}
	}
}

// snapQuest applies a new quest view the way a delta does.
func snapQuest(m *model.Model, q *pb.QuestInfo) {
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_TickFrame{TickFrame: &pb.TickFrame{Deltas: []*pb.SelfDelta{{Part: &pb.SelfDelta_Quest{Quest: &pb.QuestDelta{Quest: q}}}}}}})
}

// A quest item is refused wherever it could be sold, banked or traded, in the
// words of the quest, and every other item is not.
func TestQuestItemsAreNotOffered(t *testing.T) {
	m := questWorld(questSteps["deliver"])
	m.Self.Inventory = []string{"charm-bead", "tin-ring"}
	if why := m.QuestItemRefusal("charm-bead"); !strings.Contains(why, "quest item") {
		t.Errorf("refusal %q", why)
	}
	if why := m.QuestItemRefusal("tin-ring"); why != "" {
		t.Errorf("an ordinary item refused: %q", why)
	}
}
