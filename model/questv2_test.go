package model_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Quests v2 (D105): each step kind is told in words, with its count.
func TestEachStepKindIsTold(t *testing.T) {
	m := model.New()
	m.Items["charm-bead"] = &pb.ItemInfo{Id: "charm-bead", Name: "charm bead", Tier: "common", QuestItem: true}
	for _, c := range []struct {
		name     string
		q        *pb.QuestInfo
		bag      []string // the bags: a deliver or collect step counts what they hold
		words    string
		progress string
	}{
		{"deliver", &pb.QuestInfo{Kind: "deliver", Item: "charm-bead", Need: 3, Progress: 2, TargetName: "Barnaby Tuck"}, []string{"charm-bead", "charm-bead"}, "Bring 3 charm bead to Barnaby Tuck", "Held: 2 of 3 charm bead"},
		{"deliver, more than needed", &pb.QuestInfo{Kind: "deliver", Item: "charm-bead", Need: 2, TargetName: "Barnaby Tuck"}, []string{"charm-bead", "charm-bead", "charm-bead"}, "Bring 2 charm bead to Barnaby Tuck", "Held: 2 of 2 charm bead"},
		{"deliver, server text", &pb.QuestInfo{Kind: "deliver", Item: "charm-bead", Need: 3, Text: "Bring 3 charm beads to Barnaby Tuck.", TargetName: "Barnaby Tuck"}, nil, "Bring 3 charm beads to Barnaby Tuck.", "Held: 0 of 3 charm bead"},
		{"collect", &pb.QuestInfo{Kind: "collect", Item: "charm-bead", Need: 4, Progress: 0}, []string{"charm-bead", "tin-ring"}, "Collect 4 charm bead", "Collected: 1 of 4 charm bead"},
		{"kill", &pb.QuestInfo{Kind: "kill", Need: 5, Progress: 5, Unit: "kills"}, nil, "Defeat 5", "Kills: 5 of 5"},
		{"watch", &pb.QuestInfo{Kind: "watch", Need: 20, Progress: 12, Unit: "seconds", TargetName: "the shore"}, nil, "Watch from the shore for 20 s", "Watching… 12 of 20 s"},
		{"talk", &pb.QuestInfo{Kind: "talk_to", TargetName: "Tamsin Hale"}, nil, "Tamsin Hale", ""},
		{"deliver without an item", &pb.QuestInfo{Kind: "deliver", Need: 1, TargetName: "Barnaby Tuck"}, []string{"tin-ring"}, "Bring 1 items to Barnaby Tuck", "Held: 0 of 1 items"},
	} {
		m.Self.Inventory = c.bag
		if got := m.QuestStepWords(c.q); got != c.words {
			t.Errorf("%s: words %q, want %q", c.name, got, c.words)
		}
		if got := m.QuestProgressLine(c.q); got != c.progress {
			t.Errorf("%s: progress %q, want %q", c.name, got, c.progress)
		}
	}
}

// A locked offer says why in a few words; an open one says nothing.
func TestLockedOffersSayWhy(t *testing.T) {
	for _, c := range []struct {
		o    *pb.QuestOfferInfo
		want string
	}{
		{&pb.QuestOfferInfo{}, ""},
		{&pb.QuestOfferInfo{Locked: "level", MinLevel: 8, Text: "Reach level 8"}, "needs level 8"},
		{&pb.QuestOfferInfo{Locked: "requires", Text: "Finish first: Lost Charms"}, "finish Lost Charms first"},
		{&pb.QuestOfferInfo{Locked: "requires", Text: "Finish first: A, B"}, "finish A, B first"},
		{&pb.QuestOfferInfo{Locked: "requires"}, "finish an earlier quest first"},
		{&pb.QuestOfferInfo{Locked: "condition", Text: "Not yet"}, "not yet: come back later"},
		{&pb.QuestOfferInfo{Locked: "other"}, "other"},
	} {
		if got := model.OfferLocked(c.o); got != c.want {
			t.Errorf("%v: %q, want %q", c.o, got, c.want)
		}
	}
	n := &pb.NpcInfo{QuestOffers: []*pb.QuestOfferInfo{{Id: "a", Locked: "level"}, {Id: "b"}, {Id: "c", Locked: "requires"}}}
	var ids []string
	for _, o := range model.QuestOffers(n) {
		ids = append(ids, o.GetId())
	}
	if strings.Join(ids, "") != "bac" {
		t.Errorf("open offers come first: %v", ids)
	}
}

// A step in another zone names the zone and, when the map has it, the door.
func TestAStepInAnotherZoneNamesTheDoor(t *testing.T) {
	m := model.New()
	m.Map = &pb.ZoneMap{Doors: []*pb.Door{{Id: "town-gate", Name: "Town Gate", Pos: &pb.Pos{X: 6000, Y: 225000}}}, Markers: []*pb.Marker{{Id: "m", Name: "Marker", Pos: &pb.Pos{X: 1, Y: 2}}}}
	away := func(p *pb.Pos) *pb.QuestInfo {
		return &pb.QuestInfo{Zone: "thornwick", ZoneName: "Thornwick", TargetKind: "zone", Target: "thornwick", TargetPos: p}
	}
	for _, c := range []struct {
		name string
		q    *pb.QuestInfo
		door string
		ok   bool
	}{
		{"the door", away(&pb.Pos{X: 6000, Y: 225000}), "Town Gate", true},
		{"a marker", away(&pb.Pos{X: 1, Y: 2}), "Marker", true},
		{"a spot with no name", away(&pb.Pos{X: 9, Y: 9}), "door", true},
		{"no door from here", away(nil), "", false},
		{"same zone", &pb.QuestInfo{TargetPos: &pb.Pos{X: 6000, Y: 225000}}, "", false},
	} {
		name, pos := m.QuestDoor(c.q)
		if name != c.door || (pos != nil) != c.ok {
			t.Errorf("%s: door %q %v", c.name, name, pos)
		}
	}
	if !model.QuestAway(away(nil)) || model.QuestAway(&pb.QuestInfo{}) {
		t.Error("QuestAway")
	}
}

func watchSnap(m *model.Model, seq uint64, progress int64, combat bool) {
	snap(m, seq, "downs", &pb.SelfState{InCombat: combat, Quest: &pb.QuestInfo{Id: "q", Step: 2, Kind: "watch", Need: 20, Progress: progress, Unit: "seconds"}}, nil)
}

// A watch that falls back to nothing says why, and keeps saying it until it is
// past where it was; a fight on the timer says so at once.
func TestAWatchThatResetsSaysWhy(t *testing.T) {
	m := model.New()
	hint := func() string { return m.WatchHint(m.Quest()) }
	watchSnap(m, 1, 12, false)
	if hint() != "" {
		t.Errorf("a running watch: %q", hint())
	}
	watchSnap(m, 2, 0, false)
	if h := hint(); !strings.Contains(h, "started over at 12 s") || !strings.Contains(h, "left the spot") {
		t.Errorf("left the spot: %q", h)
	}
	watchSnap(m, 3, 5, false)
	if h := hint(); !strings.Contains(h, "started over at 12 s") {
		t.Errorf("still short of the old mark: %q", h)
	}
	watchSnap(m, 4, 12, false)
	if hint() != "" {
		t.Errorf("past it: %q", hint())
	}
	watchSnap(m, 5, 8, false)
	watchSnap(m, 6, 0, true)
	if h := hint(); !strings.Contains(h, "in combat") {
		t.Errorf("in a fight: %q", h)
	}
	// A delta carries the same news as a snapshot.
	watchSnap(m, 7, 9, false)
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_TickFrame{TickFrame: &pb.TickFrame{Deltas: []*pb.SelfDelta{{Part: &pb.SelfDelta_Quest{Quest: &pb.QuestDelta{Quest: &pb.QuestInfo{Id: "q", Step: 2, Kind: "watch", Need: 20, Progress: 0}}}}}}}})
	if h := hint(); !strings.Contains(h, "started over at 12 s") || !strings.Contains(h, "left the spot") {
		t.Errorf("a delta's reset (the old mark of 12 s stands): %q", h)
	}
	// The next step starts clean.
	snap(m, 8, "downs", &pb.SelfState{Quest: &pb.QuestInfo{Id: "q", Step: 3, Kind: "kill", Need: 3}}, nil)
	if hint() != "" {
		t.Errorf("another step: %q", hint())
	}
}

// Quest items are known by the description, refused for sale, and their removal
// is logged whether a snapshot or a bags delta brings it.
func TestQuestItemsAreRefusedAndTheirRemovalIsLogged(t *testing.T) {
	m := model.New()
	m.Items["charm-bead"] = &pb.ItemInfo{Id: "charm-bead", Name: "charm bead", Tier: "common", QuestItem: true}
	m.Items["tin-ring"] = &pb.ItemInfo{Id: "tin-ring", Name: "Tin Ring"}
	snap(m, 1, "town", &pb.SelfState{Inventory: []string{"charm-bead", "charm-bead", "charm-bead", "tin-ring"}}, nil)
	if !m.IsQuestItem("charm-bead") || m.IsQuestItem("tin-ring") || m.IsQuestItem("unknown") {
		t.Error("IsQuestItem")
	}
	if why := m.QuestItemRefusal("charm-bead"); !strings.Contains(why, "charm bead is a quest item") || m.QuestItemRefusal("tin-ring") != "" {
		t.Errorf("refusal %q", m.QuestItemRefusal("charm-bead"))
	}
	// Handed in: two of the three go.
	snap(m, 2, "town", &pb.SelfState{Inventory: []string{"charm-bead", "tin-ring"}}, nil)
	if got := lastText(m); got != "Quest item removed from your bags: 2 charm bead." {
		t.Errorf("after a hand-in: %q", got)
	}
	// The quest ends: the last one goes, by a bags delta.
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_TickFrame{TickFrame: &pb.TickFrame{Deltas: []*pb.SelfDelta{{Part: &pb.SelfDelta_Inventory{Inventory: &pb.InventoryDelta{Inventory: []string{"tin-ring"}}}}}}}})
	if got := lastText(m); got != "Quest item removed from your bags: 1 charm bead." {
		t.Errorf("after the quest ends: %q", got)
	}
	// A sold ring is no quest item and says nothing here.
	n := len(m.Log)
	snap(m, 3, "town", &pb.SelfState{}, nil)
	if len(m.Log) != n {
		t.Errorf("an ordinary item left and the log grew: %q", lastText(m))
	}
}
