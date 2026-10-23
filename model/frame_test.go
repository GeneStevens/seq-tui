package model_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func load(t *testing.T, name string) []*pb.ServerMessage {
	t.Helper()
	msgs, err := record.Load(filepath.Join("..", "testdata", "logs"), name)
	if err != nil {
		t.Fatal(err)
	}
	return msgs
}

// framed makes the stream of a recorded log the stream of a server that sends
// tick frames: each run of events numbered one after another is one frame, its
// events carrying no number of their own. Everything else stays as it was.
func framed(msgs []*pb.ServerMessage) []*pb.ServerMessage {
	var out []*pb.ServerMessage
	var run *pb.TickFrame
	flush := func() {
		if run != nil {
			out = append(out, &pb.ServerMessage{Body: &pb.ServerMessage_TickFrame{TickFrame: run}})
			run = nil
		}
	}
	for _, m := range msgs {
		ev := m.GetEvent()
		if ev == nil || ev.GetSeq() == 0 || (run != nil && ev.GetSeq() != run.BaseSeq+uint64(len(run.Events))) {
			flush()
		}
		if ev == nil || ev.GetSeq() == 0 {
			out = append(out, m)
			continue
		}
		if run == nil {
			run = &pb.TickFrame{BaseSeq: ev.GetSeq()}
		}
		bare := proto.Clone(ev).(*pb.Event)
		bare.Seq = 0
		run.Events = append(run.Events, bare)
	}
	flush()
	return out
}

func apply(msgs []*pb.ServerMessage) *model.Model {
	m := model.New()
	for _, msg := range msgs {
		m.Apply(msg)
	}
	return m
}

// digest is what a model says of itself that a frame must not change.
func digest(m *model.Model) []any {
	s := m.Self
	return []any{m.Log, m.Last(), m.Faults, m.Duplicates, m.Intents, s.XP, s.HP, s.Level, s.Pos.String(), s.Target, s.Crowns, s.Inventory,
		len(m.Entities), len(m.Group), len(m.Drops), m.GameTimeMs, m.SelfMoves}
}

// A frame is the messages of old, in one: the model it makes is the model of the
// stream of messages, over every kind of recorded session.
func TestFramesApplyAsTheStreamOfMessages(t *testing.T) {
	for _, name := range []string{"fight_loot.jsonl", "companion_reconnect.jsonl", "drop_resume.jsonl", "party_warren.jsonl", "levelup.jsonl", "loot_equip.jsonl"} {
		msgs := load(t, name)
		f := framed(msgs)
		if len(f) >= len(msgs) {
			t.Errorf("%s: %d messages made %d: nothing was framed", name, len(msgs), len(f))
		}
		if got, want := digest(apply(f)), digest(apply(msgs)); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: frames and messages give different models:\n got %v\nwant %v", name, got, want)
		}
	}
}

// A resume repeats nothing (D39), and a model that is sent a frame that does
// repeat or skip is exactly as sure as for an event out of turn.
func TestFramesAreExactlyOnce(t *testing.T) {
	ev := func(n uint64) *pb.Event {
		return &pb.Event{Body: &pb.Event_RulesTick{RulesTick: &pb.RulesTick{}}, GameTimeMs: int64(n)}
	}
	frame := func(base uint64, n int) *pb.ServerMessage {
		f := &pb.TickFrame{BaseSeq: base}
		for i := range n {
			f.Events = append(f.Events, ev(base+uint64(i)))
		}
		return &pb.ServerMessage{Body: &pb.ServerMessage_TickFrame{TickFrame: f}}
	}
	for _, tc := range []struct {
		name         string
		frames       []*pb.ServerMessage
		last         uint64
		dups, faults int
	}{
		{"in order", []*pb.ServerMessage{frame(11, 2), frame(13, 1)}, 13, 0, 0},
		{"a resume's frame overlaps the last", []*pb.ServerMessage{frame(11, 2), frame(12, 2)}, 13, 1, 0},
		{"a frame that is all repeats", []*pb.ServerMessage{frame(11, 3), frame(11, 3)}, 13, 3, 0},
		{"a frame skips", []*pb.ServerMessage{frame(11, 1), frame(14, 1)}, 14, 0, 1},
		{"a frame with no events", []*pb.ServerMessage{frame(11, 1), frame(12, 0)}, 11, 0, 0},
	} {
		m := model.New()
		m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Seq: 10, Space: "z", Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Pos: &pb.Pos{}}}}}})
		for _, f := range tc.frames {
			m.Apply(f)
		}
		if m.Last() != tc.last || m.Duplicates != tc.dups || len(m.Faults) != tc.faults {
			t.Errorf("%s: last %d dups %d faults %v; want %d, %d, %d", tc.name, m.Last(), m.Duplicates, m.Faults, tc.last, tc.dups, tc.faults)
		}
	}
}

func text[T proto.Message](list []T) string {
	var out string
	for _, p := range list {
		out += prototext.MarshalOptions{}.Format(p) + ";"
	}
	return out
}

// selfView is the parts of the self view a delta can set, as text.
func selfView(m *model.Model) string {
	s := m.Self
	group := ""
	for _, id := range m.GroupIDs() {
		group += prototext.MarshalOptions{}.Format(m.Group[id].Entity) + fmt.Sprint(m.Group[id].Target, m.Group[id].XP, m.Group[id].Mana) + ";"
	}
	return fmt.Sprint(s.Inventory, s.Equipped, s.Bank, s.InventorySize, s.BagSlots, s.BindZone, "|", s.Crowns, text(s.Owed), "|",
		s.Level, s.HP, s.MaxHP, s.Dead, s.XP, s.XPStart, s.XPNext, s.AtCap, s.Class, s.Mana, s.MaxMana, s.GCDMs, s.Slots, s.ReadyAtMs, text(s.Abilities), s.Pos, s.Ground, s.SpeedPct, s.Cast, s.Effects, "|",
		s.Quest, "|", s.Companion, "|", text(s.Contracts), text(m.NPCs), "|", group, "|", m.Trade, "|", len(m.Items), m.Items["ring"])
}

func selfSnap(f func(*pb.Snapshot)) *pb.ServerMessage {
	s := &pb.Snapshot{Space: "z", Seq: 5, Tick: 7, GameTimeMs: 7000,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Level: 3, Hp: 20, MaxHp: 30, Pos: &pb.Pos{X: 1000, Y: 2000}},
			Xp: 50, XpLevelStart: 40, XpNextLevel: 90, Class: "warrior", Mana: 3, MaxMana: 5, GcdMs: 1500,
			Abilities: []*pb.AbilityInfo{{Id: "bash", Name: "Bash"}}, Slots: []*pb.AbilitySlot{{Slot: 1, Ability: "bash"}},
			Inventory: []string{"ring#1"}, InventorySize: 10, BagSlots: 1, Crowns: 12, BindZone: "town"},
		Items: []*pb.ItemInfo{{Id: "ring", Name: "Ring"}}}
	f(s)
	return &pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: s}}
}

// A delta replaces the part of the self view it names, and the model it makes is
// the model of the snapshot it stands in for: every part is a row.
func TestDeltasMakeTheModelOfTheSnapshot(t *testing.T) {
	member := &pb.GroupMember{Entity: &pb.EntityState{Id: "mate", Name: "Mate", Level: 2, Hp: 9, MaxHp: 9, Pos: &pb.Pos{X: 5}}, Xp: 7, Mana: 2, MaxMana: 4}
	quest := &pb.QuestInfo{}
	for _, tc := range []struct {
		part  string
		mod   func(*pb.Snapshot)
		delta *pb.SelfDelta
	}{
		{"inventory", func(s *pb.Snapshot) {
			s.Self.Inventory, s.Self.Equipped, s.Self.Bank, s.Self.InventorySize, s.Self.BagSlots, s.Self.BindZone = []string{"ring#1", "ring#2"}, []string{"", "ring#3"}, []string{"ring#4"}, 12, 2, "camp"
		}, &pb.SelfDelta{Part: &pb.SelfDelta_Inventory{Inventory: &pb.InventoryDelta{Inventory: []string{"ring#1", "ring#2"}, Equipped: []string{"", "ring#3"}, Bank: []string{"ring#4"}, InventorySize: 12, BagSlots: 2, BindZone: "camp"}}}},
		{"money", func(s *pb.Snapshot) {
			s.Self.Crowns, s.Self.Owed = 99, []*pb.OwedPay{{Crowns: 5}}
		}, &pb.SelfDelta{Part: &pb.SelfDelta_Money{Money: &pb.MoneyDelta{Crowns: 99, Owed: []*pb.OwedPay{{Crowns: 5}}}}}},
		{"stats", func(s *pb.Snapshot) {
			s.Self.Entity.Level, s.Self.Entity.Hp, s.Self.Entity.MaxHp, s.Self.Xp, s.Self.XpLevelStart, s.Self.XpNextLevel, s.Self.MaxMana = 4, 40, 40, 91, 90, 150, 9
			s.Self.Abilities = append(s.Self.Abilities, &pb.AbilityInfo{Id: "slam", Name: "Slam"})
			s.Self.Slots = append(s.Self.Slots, &pb.AbilitySlot{Slot: 2, Ability: "slam", ReadyAtMs: 9000})
		}, &pb.SelfDelta{Part: &pb.SelfDelta_Stats{Stats: &pb.StatsDelta{State: &pb.SelfState{
			Entity: &pb.EntityState{Level: 4, Hp: 40, MaxHp: 40, Pos: &pb.Pos{X: 1000, Y: 2000}, Name: "Me"}, Xp: 91, XpLevelStart: 90, XpNextLevel: 150, Class: "warrior", Mana: 3, MaxMana: 9, GcdMs: 1500,
			Abilities: []*pb.AbilityInfo{{Id: "bash", Name: "Bash"}, {Id: "slam", Name: "Slam"}}, Slots: []*pb.AbilitySlot{{Slot: 1, Ability: "bash"}, {Slot: 2, Ability: "slam", ReadyAtMs: 9000}}}}}}},
		{"quest", func(s *pb.Snapshot) { s.Self.Quest = quest }, &pb.SelfDelta{Part: &pb.SelfDelta_Quest{Quest: &pb.QuestDelta{Quest: quest}}}},
		{"companion", func(s *pb.Snapshot) { s.Self.Companion = &pb.CompanionState{Stage: 2, Role: "melee", Present: true} },
			&pb.SelfDelta{Part: &pb.SelfDelta_Companion{Companion: &pb.CompanionDelta{Companion: &pb.CompanionState{Stage: 2, Role: "melee", Present: true}}}}},
		{"offers", func(s *pb.Snapshot) {
			s.Self.Contracts = []*pb.ContractInfo{{Id: "c1", Name: "Boars"}}
			s.Npcs = []*pb.NpcInfo{{Id: "n1", Name: "Board", Role: "contract_board"}}
		}, &pb.SelfDelta{Part: &pb.SelfDelta_Offers{Offers: &pb.OffersDelta{Contracts: []*pb.ContractInfo{{Id: "c1", Name: "Boars"}}, Npcs: []*pb.NpcInfo{{Id: "n1", Name: "Board", Role: "contract_board"}}}}}},
		{"group", func(s *pb.Snapshot) { s.Group = []*pb.GroupMember{member} },
			&pb.SelfDelta{Part: &pb.SelfDelta_Group{Group: &pb.GroupDelta{Group: []*pb.GroupMember{member}}}}},
		{"trade", func(s *pb.Snapshot) { s.Trade = &pb.TradeState{With: "mate", Open: true} },
			&pb.SelfDelta{Part: &pb.SelfDelta_Trade{Trade: &pb.TradeDelta{Trade: &pb.TradeState{With: "mate", Open: true}}}}},
		{"items", func(s *pb.Snapshot) { s.Items = append(s.Items, &pb.ItemInfo{Id: "ring", Name: "Ring of Ash"}) },
			&pb.SelfDelta{Part: &pb.SelfDelta_Items{Items: &pb.ItemsDelta{Items: []*pb.ItemInfo{{Id: "ring", Name: "Ring of Ash"}}}}}},
	} {
		base := selfSnap(func(*pb.Snapshot) {})
		bySnapshot, byDelta := apply([]*pb.ServerMessage{base}), apply([]*pb.ServerMessage{base})
		bySnapshot.Apply(selfSnap(tc.mod))
		byDelta.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_TickFrame{TickFrame: &pb.TickFrame{BaseSeq: 6, Deltas: []*pb.SelfDelta{tc.delta}}}})
		if got, want := selfView(byDelta), selfView(bySnapshot); got != want {
			t.Errorf("%s: the delta and the snapshot give different self views:\n got %s\nwant %s", tc.part, got, want)
		}
		if selfView(bySnapshot) == selfView(apply([]*pb.ServerMessage{base})) {
			t.Errorf("%s: the row changes nothing, so proves nothing", tc.part)
		}
	}
}

// Where the server follows an event with no delta for what a snapshot of old
// would have changed, a frame's events make the change; the stream of messages
// is read as it was (a snapshot follows).
func TestFramesCompleteWhatNoDeltaCarries(t *testing.T) {
	trainer := &pb.NpcInfo{Id: "t1", Name: "Trainer", Role: "trainer", Trains: []*pb.TrainInfo{{Ability: "ward", Cost: 45}, {Ability: "bash", Cost: 5}}}
	join := func() *model.Model {
		m := apply([]*pb.ServerMessage{selfSnap(func(s *pb.Snapshot) { s.Npcs = []*pb.NpcInfo{trainer} })})
		return m
	}
	appeared := &pb.Event{Body: &pb.Event_EntityAppeared{EntityAppeared: &pb.EntityAppeared{Entity: &pb.EntityState{Id: "comp", Name: "Comp", Owner: "me", Hp: 5, MaxHp: 5, Pos: &pb.Pos{}}}}}
	learned := &pb.Event{Body: &pb.Event_Reward{Reward: &pb.Reward{Kind: "train", Recipient: "me", Detail: "ward", Amount: 45}}}
	for _, tc := range []struct {
		name  string
		event *pb.Event
		check func(*model.Model) bool
	}{
		{"the companion comes into the group", appeared, func(m *model.Model) bool { return m.CompanionMember() != nil && m.Entities["comp"] == nil }},
		{"the lesson learned is known", learned, func(m *model.Model) bool {
			tr := m.NPC("t1").GetTrains()
			return tr[0].GetKnown() && !tr[1].GetKnown() && !trainer.Trains[0].GetKnown() // the message is left as it was
		}},
	} {
		inFrame := join()
		inFrame.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_TickFrame{TickFrame: &pb.TickFrame{BaseSeq: 6, Events: []*pb.Event{tc.event}}}})
		if !tc.check(inFrame) {
			t.Errorf("%s: not so in a frame", tc.name)
		}
		alone := join()
		e := proto.Clone(tc.event).(*pb.Event)
		e.Seq = 6
		alone.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: e}})
		if tc.check(alone) {
			t.Errorf("%s: also so in a stream of messages, where a snapshot follows", tc.name)
		}
	}
}
