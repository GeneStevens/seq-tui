package model_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func snap(m *model.Model, seq uint64, space string, self *pb.SelfState, npcs []*pb.NpcInfo, ents ...*pb.EntityState) {
	self.Entity = &pb.EntityState{Id: "me", Name: "Me", Pos: &pb.Pos{X: 1000, Y: 1000}}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: space, Seq: seq, Self: self, Npcs: npcs, Entities: ents}}})
}

// A held contract carries its own name, text and reward wherever the character
// is (D52): away from the board the model has nothing to fill in.
func TestHeldContractsAreNamedAwayFromTheBoard(t *testing.T) {
	m := model.New()
	board := &pb.NpcInfo{Id: "board", Name: "The Board", Role: "contract_board", Pos: &pb.Pos{X: 2000, Y: 1000}, Offers: []*pb.ContractOffer{
		{Id: "hunt-wolves", Name: "Thin the Wolves", Description: "Kill 3 wolves.", Kind: "hunt", Need: 3, MinLevel: 1, MaxLevel: 3, Reward: &pb.RewardInfo{Xp: 30, Crowns: 12, Item: "tin-ring"}},
	}}
	snap(m, 1, "town", &pb.SelfState{}, []*pb.NpcInfo{board})
	m.Items["tin-ring"] = &pb.ItemInfo{Id: "tin-ring", Name: "Tin Ring"}
	if got := m.RewardWords(board.Offers[0].Reward); got != "30 XP, 12 crowns and Tin Ring" {
		t.Errorf("reward words %q", got)
	}
	// Away, in the Downs: the snapshot describes it in full.
	snap(m, 3, "downs", &pb.SelfState{Contracts: []*pb.ContractInfo{{Id: "hunt-wolves", Name: "Thin the Wolves", Description: "Kill 3 wolves.", Kind: "hunt", Have: 2, Need: 3, Reward: &pb.RewardInfo{Xp: 30, Crowns: 12}}}}, nil)
	c := m.Contract("hunt-wolves")
	if c.GetName() != "Thin the Wolves" || c.GetDescription() != "Kill 3 wolves." || c.GetReward().GetCrowns() != 12 || c.GetHave() != 2 {
		t.Errorf("held contract away from the board: %v", c)
	}
	if model.Progress(c) != "2 of 3 killed" {
		t.Errorf("progress %q", model.Progress(c))
	}
	if m.ContractName("hunt-wolves") != "Thin the Wolves" || m.ContractName("gather-dust") != "gather-dust" {
		t.Errorf("names %q %q", m.ContractName("hunt-wolves"), m.ContractName("gather-dust"))
	}
}

func TestProgressInWords(t *testing.T) {
	for _, tc := range []struct {
		c    *pb.ContractInfo
		want string
	}{
		{&pb.ContractInfo{Kind: "hunt", Have: 1, Need: 4}, "1 of 4 killed"},
		{&pb.ContractInfo{Kind: "bounty", Have: 0, Need: 1}, "0 of 1 killed"},
		{&pb.ContractInfo{Kind: "gather", Have: 2, Need: 3}, "2 of 3 carried"},
		{&pb.ContractInfo{Kind: "scout", Have: 0, Need: 1}, "go to the marker"},
		{&pb.ContractInfo{Kind: "scout", Have: 1, Need: 1}, "marker reached"},
	} {
		if got := model.Progress(tc.c); got != tc.want {
			t.Errorf("%v: %q, want %q", tc.c, got, tc.want)
		}
	}
}

// What an NPC says is kept for the one who talked; the answer for someone else's
// talk is a log line only.
func TestDialogueAndRewardsInTheLog(t *testing.T) {
	m := model.New()
	snap(m, 1, "town", &pb.SelfState{}, []*pb.NpcInfo{{Id: "board", Name: "The Board", Role: "contract_board", Pos: &pb.Pos{X: 2000, Y: 1000}}})
	event(m, 1000, &pb.Event{Body: &pb.Event_Dialogue{Dialogue: &pb.Dialogue{Actor: "me", Npc: "board", Lines: []string{"Short jobs."}, Options: []*pb.DialogueOption{{Action: "contract_accept", Label: "Take a contract"}}}}})
	d := m.Said("board")
	if d == nil || d.Lines[0] != "Short jobs." || d.Options[0].GetAction() != "contract_accept" || m.Talks != 1 {
		t.Fatalf("dialogue %+v talks %d", d, m.Talks)
	}
	if got := lastText(m); got != "The Board says: Short jobs." {
		t.Errorf("log: %q", got)
	}
	event(m, 1100, &pb.Event{Body: &pb.Event_Dialogue{Dialogue: &pb.Dialogue{Actor: "bram", Npc: "board", Lines: []string{"Other"}}}})
	if m.Said("board").Lines[0] != "Short jobs." || m.Talks != 1 {
		t.Errorf("another character's talk replaced mine: %+v", m.Said("board"))
	}

	event(m, 1500, &pb.Event{Body: &pb.Event_WorldEvent{WorldEvent: &pb.WorldEvent{Event: "harvest-raid", Name: "Harvest Raid", Kind: "won", Text: "The farms are safe."}}})

	// Rewards of the contract kinds.
	for _, tc := range []struct {
		r    *pb.Reward
		want string
	}{
		{&pb.Reward{Recipient: "me", Source: "board", Kind: "contract_accept", Detail: "hunt-wolves"}, "you took the contract hunt-wolves"},
		{&pb.Reward{Recipient: "me", Kind: "contract_abandon", Detail: "hunt-wolves"}, "you gave up the contract hunt-wolves"},
		{&pb.Reward{Recipient: "me", Source: "board", Kind: "contract_turn_in", Detail: "hunt-wolves", Xp: 30, Amount: 12, Item: "tin-ring"}, "you handed in hunt-wolves to The Board and got 30 XP, 12 crowns and tin-ring"},
		{&pb.Reward{Recipient: "me", Source: "raid", Kind: "world_event", Detail: "harvest-raid", Xp: 120, Amount: 60}, "you was paid for the Harvest Raid: 120 XP and 60 crowns"},
	} {
		event(m, 2000, &pb.Event{Body: &pb.Event_Reward{Reward: tc.r}})
		if got := lastText(m); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.r.Kind, got, tc.want)
		}
	}
}

// Pay that waited for a join (D55) is told as owed: the reward of an event whose
// end this model was not told, and a refund with no failed opening before it.
func TestOwedPayIsToldAsOwed(t *testing.T) {
	m := model.New()
	snap(m, 1, "town", &pb.SelfState{}, nil)
	event(m, 100, &pb.Event{Body: &pb.Event_Reward{Reward: &pb.Reward{Recipient: "me", Source: "raid", Kind: "world_event", Detail: "harvest-raid", Xp: 120, Amount: 60}}})
	if got := lastText(m); got != "owed pay arrived for the harvest-raid, which ended while you were away: 120 XP and 60 crowns" {
		t.Errorf("owed event pay: %q", got)
	}
	event(m, 200, &pb.Event{Body: &pb.Event_Reward{Reward: &pb.Reward{Recipient: "me", Kind: "refund", Amount: 50, Detail: "peril-den"}}})
	if got := lastText(m); !strings.Contains(got, "an owed refund arrived") {
		t.Errorf("owed refund: %q", got)
	}
	// A refund that answers a failed opening is not owed.
	event(m, 300, &pb.Event{Body: &pb.Event_InstanceOpenFailed{InstanceOpenFailed: &pb.InstanceOpenFailed{Reason: "instance_unavailable"}}})
	event(m, 310, &pb.Event{Body: &pb.Event_Reward{Reward: &pb.Reward{Recipient: "me", Kind: "refund", Amount: 50, Detail: "peril-den"}}})
	if got := lastText(m); !strings.Contains(got, "was refunded to you") {
		t.Errorf("refund after a failure: %q", got)
	}
}

// A world event is told to the zone: the model keeps the last announcement of
// each, active until it is won or lost, and a recent end is still shown.
func TestWorldEventAnnouncements(t *testing.T) {
	m := model.New()
	snap(m, 1, "downs", &pb.SelfState{}, nil)
	ann := func(ms int64, kind string, leaks int64) {
		event(m, ms, &pb.Event{Body: &pb.Event_WorldEvent{WorldEvent: &pb.WorldEvent{Event: "harvest-raid", Kind: kind, Text: "Defend the farms!", Leaks: leaks}}})
	}
	if len(m.ActiveEvents()) != 0 || m.RecentEvent(0, 30000) != nil {
		t.Fatal("an event before any announcement")
	}
	ann(10_000, "started", 0)
	if len(m.ActiveEvents()) != 1 || !strings.Contains(lastText(m), "has begun") {
		t.Errorf("started: %v / %q", m.ActiveEvents(), lastText(m))
	}
	ann(40_000, "leak", 1)
	if e := m.WorldEvents["harvest-raid"]; !e.Active || e.Leaks != 1 || !strings.Contains(lastText(m), "got through (1 so far)") {
		t.Errorf("leak: %+v / %q", e, lastText(m))
	}
	ann(100_000, "won", 1)
	if len(m.ActiveEvents()) != 0 || !strings.Contains(lastText(m), "is won") {
		t.Errorf("won: %v / %q", m.ActiveEvents(), lastText(m))
	}
	if m.RecentEvent(120_000, 30_000) == nil || m.RecentEvent(131_000, 30_000) != nil {
		t.Error("the end of an event is recent for thirty seconds")
	}
}

// A rare is the mob the server flags (D52); one coming into view is a log line of
// its own. The camps are the zone map's. NPCs stream: one comes into view and goes
// out of it by event, and what has been seen stays known.
func TestRaresCampsAndPeopleAreRemembered(t *testing.T) {
	m := model.New()
	wolf := &pb.EntityState{Id: "mob-a-1", Kind: pb.EntityKind_ENTITY_KIND_MOB, Name: "Wolf", Hp: 10, MaxHp: 10, Pos: &pb.Pos{X: 30000, Y: 30000}}
	snap(m, 1, "downs", &pb.SelfState{}, []*pb.NpcInfo{{Id: "captain", Name: "Captain", Role: "quest_giver", Pos: &pb.Pos{X: 5000, Y: 5000}}}, wolf)
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: "downs",
		Camps:   []*pb.Camp{{Id: "wolves", Mob: "Wolf", Level: 2, Count: 3, Pos: &pb.Pos{X: 30000, Y: 30000}}},
		Markers: []*pb.Marker{{Id: "old-oak", Kind: "scout", Pos: &pb.Pos{X: 70000, Y: 1000}}}}}})
	if c := m.Camps(); len(c) != 1 || c[0].GetMob() != "Wolf" {
		t.Errorf("camps: %v", c)
	}
	event(m, 1000, &pb.Event{Body: &pb.Event_EntityAppeared{EntityAppeared: &pb.EntityAppeared{Entity: &pb.EntityState{
		Id: "mob-grimtusk-1", Kind: pb.EntityKind_ENTITY_KIND_MOB, Name: "Old Grimtusk", Level: 6, Hp: 300, MaxHp: 300, Rare: true, Pos: &pb.Pos{X: 50000, Y: 50000}}}}})
	if r := m.RaresInView(); len(r) != 1 || r[0].GetName() != "Old Grimtusk" {
		t.Fatalf("rares in view: %v", r)
	}
	if e := m.Log[len(m.Log)-1]; e.Kind != "rare_appeared" || !strings.Contains(e.Text, "a rare, Old Grimtusk (Lv 6)") {
		t.Errorf("log: %+v", e)
	}
	// A rare is the flag, never the name.
	if !model.IsRare(m.Entities["mob-grimtusk-1"]) || model.IsRare(wolf) || model.IsRare(&pb.EntityState{Kind: pb.EntityKind_ENTITY_KIND_MOB, Name: "Old Grimtusk"}) {
		t.Error("IsRare")
	}
	// A scout contract's marker is the zone map's spot.
	held := &pb.ContractInfo{Id: "scout", Marker: "old-oak", Need: 1}
	if p := m.Marker(held); p.GetX() != 70000 {
		t.Errorf("marker: %v", p)
	}
	held.Have = 1
	if m.Marker(held) != nil {
		t.Error("a reached marker is not drawn")
	}

	// An NPC walks into view by event, with the items it names, and out of it.
	event(m, 2000, &pb.Event{Body: &pb.Event_NpcAppeared{NpcAppeared: &pb.NpcAppeared{
		Npc:   &pb.NpcInfo{Id: "smith", Name: "Smith", Role: "vendor", ReachMm: 5000, Pos: &pb.Pos{X: 9000, Y: 9000}},
		Items: []*pb.ItemInfo{{Id: "hammer", Name: "Hammer", Tier: "fine"}}}}})
	if m.NPC("smith") == nil || len(m.NPCs) != 2 || m.ItemName("hammer") != "Hammer" {
		t.Fatalf("npc_appeared: %v, %q", m.NPCs, m.ItemName("hammer"))
	}
	event(m, 3000, &pb.Event{Body: &pb.Event_NpcLeft{NpcLeft: &pb.NpcLeft{Id: "smith"}}})
	if len(m.NPCs) != 1 || m.NPCs[0].GetId() != "captain" {
		t.Errorf("npc_left: %v", m.NPCs)
	}
	// The people seen stay known after a snapshot that no longer lists them.
	snap(m, 2, "downs", &pb.SelfState{}, nil)
	if n := m.KnownNPCs(); len(n) != 2 || m.NPC("captain") == nil {
		t.Errorf("known people: %v", n)
	}
}

// Reach is the NPC's own (D52), not a number the client knows.
func TestInReachIsTheNPCsOwn(t *testing.T) {
	m := model.New()
	snap(m, 1, "town", &pb.SelfState{}, nil)
	m.Self.Pos = &pb.Pos{}
	near := &pb.NpcInfo{Pos: &pb.Pos{X: 3000, Y: 0}, ReachMm: 3000}
	far := &pb.NpcInfo{Pos: &pb.Pos{X: 3001, Y: 0}, ReachMm: 3000}
	if !m.InReach(near) || m.InReach(far) || m.InReach(&pb.NpcInfo{Pos: &pb.Pos{}}) {
		t.Error("InReach")
	}
}

// A world event's name is the server's (D55); a snapshot after a join that shows
// more than the character had is owed pay arriving; an offer of no quest is no
// quest step; party and companion mana are the group's.
func TestD55Reads(t *testing.T) {
	m := model.New()
	snap(m, 1, "downs", &pb.SelfState{Crowns: 5}, nil)
	event(m, 100, &pb.Event{Body: &pb.Event_WorldEvent{WorldEvent: &pb.WorldEvent{Event: "harvest-raid", Name: "Harvest Raid", Kind: "started", Text: "Defend!"}}})
	if got := m.EventName("harvest-raid"); got != "Harvest Raid" || m.WorldEvents["harvest-raid"].Title() != "Harvest Raid" {
		t.Errorf("the event's name: %q", got)
	}
	if got := m.EventName("unheard-of"); got != "unheard-of" {
		t.Errorf("an event never announced is its id: %q", got)
	}

	// Across a door the snapshot shows 30 crowns more: the join paid them.
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Body: &pb.Event_SpaceChanged{SpaceChanged: &pb.SpaceChanged{Space: "town"}}}}})
	snap(m, 1, "town", &pb.SelfState{Crowns: 35, Xp: 60}, nil)
	if got := lastText(m); got != "owed pay arrived with the join: 60 XP and 30 crowns" {
		t.Errorf("after the join: %q", got)
	}
	before := len(m.Log)
	snap(m, 3, "town", &pb.SelfState{Crowns: 99, Xp: 99}, nil)
	if len(m.Log) != before {
		t.Errorf("a snapshot with no join added a line: %q", lastText(m))
	}

	// An offer that is no quest step is none, whoever offers it.
	board := &pb.NpcInfo{Id: "board", Name: "Board", Role: "contract_board", Pos: &pb.Pos{X: 2000, Y: 1000}, Offers: []*pb.ContractOffer{{Id: "hunt", Name: "Hunt", Kind: "hunt", Need: 3}}}
	snap(m, 4, "town", &pb.SelfState{}, []*pb.NpcInfo{board})
	if len(m.QuestSteps()) != 0 || m.QuestGiver() != "" {
		t.Errorf("quest steps from a plain board: %+v", m.QuestSteps())
	}

	// Mana of the group's members.
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "town", Seq: 5,
		Self:  &pb.SelfState{Entity: &pb.EntityState{Id: "me"}, Companion: &pb.CompanionState{Unlocked: true, Role: "healer", Present: true, Id: "comp-me", Mana: 40, MaxMana: 100}},
		Group: []*pb.GroupMember{{Entity: &pb.EntityState{Id: "comp-me", Owner: "me"}, Mana: 40, MaxMana: 100}, {Entity: &pb.EntityState{Id: "bram"}, Mana: 7, MaxMana: 20}}}}})
	if cur, most := m.CompanionMana(); cur != 40 || most != 100 || m.Group["bram"].Mana != 7 || m.Group["bram"].MaxMana != 20 {
		t.Errorf("mana: companion %d of %d, bram %+v", cur, most, m.Group["bram"])
	}
	event(m, 200, &pb.Event{Body: &pb.Event_ResourceChanged{ResourceChanged: &pb.ResourceChanged{Actor: "bram", Resource: "mana", Delta: -5}}})
	if m.Group["bram"].Mana != 2 {
		t.Errorf("bram's mana after an event: %d", m.Group["bram"].Mana)
	}
}

// The cleanup of M8 (D56): a reward names its world event itself, an NPC the
// server streams moves, marker names are the pack's, owed pay is listed once at
// login, and the House Companion's recruit option is not offered when the
// snapshot carries no companion.
func TestM8CleanupReads(t *testing.T) {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "downs", Seq: 1,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Pos: &pb.Pos{X: 1000, Y: 1000}}, Owed: []*pb.OwedPay{
			{Kind: "event", Name: "Harvest Raid", Xp: 30, Crowns: 15}, {Kind: "refund", Name: "Warren Peril Delve", Crowns: 50}}},
		Npcs: []*pb.NpcInfo{{Id: "watchman", Name: "Watchman", Pos: &pb.Pos{X: 5000, Y: 5000}}}}}})
	if got := strings.Count(logText(m), "owed pay waiting"); got != 2 {
		t.Fatalf("%d owed pay lines at login: %q", got, logText(m))
	}
	if !strings.Contains(logText(m), "owed pay waiting from the Harvest Raid: 30 XP and 15 crowns, paid at your next join") ||
		!strings.Contains(logText(m), "owed pay waiting from a refund for Warren Peril Delve: 50 crowns, paid at your next join") {
		t.Errorf("the owed pay lines: %q", logText(m))
	}
	// A later snapshot of the same character is not a login.
	n := len(m.Log)
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "downs", Seq: 2,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me"}, Owed: []*pb.OwedPay{{Kind: "event", Name: "Harvest Raid", Xp: 30}}}}}})
	if len(m.Log) != n {
		t.Errorf("a second snapshot listed the owed pay again: %q", logText(m))
	}

	// A guard the server streams moves where it is told.
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "downs", Seq: 3,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me"}},
		Npcs: []*pb.NpcInfo{{Id: "watchman", Name: "Watchman", Pos: &pb.Pos{X: 5000, Y: 5000}}}}}})
	event(m, 300, &pb.Event{Body: &pb.Event_NpcMoved{NpcMoved: &pb.NpcMoved{Id: "watchman", Pos: &pb.Pos{X: 6500, Y: 5000}}}})
	if g := m.NPC("watchman"); g.GetPos().GetX() != 6500 || g.GetName() != "Watchman" {
		t.Errorf("the guard after npc_moved: %v", g)
	}
	if k := m.KnownNPCs(); len(k) != 1 || k[0].GetPos().GetX() != 6500 {
		t.Errorf("the guard remembered for the map: %v", k)
	}
	event(m, 400, &pb.Event{Body: &pb.Event_NpcMoved{NpcMoved: &pb.NpcMoved{Id: "nobody", Pos: &pb.Pos{X: 1}}}}) // not in view: nothing

	// A world event's reward uses the event's name it carries, else the announced one.
	event(m, 500, &pb.Event{Body: &pb.Event_WorldEvent{WorldEvent: &pb.WorldEvent{Event: "raid", Name: "Announced Raid", Kind: "started"}}})
	event(m, 600, &pb.Event{Body: &pb.Event_Reward{Reward: &pb.Reward{Recipient: "me", Kind: "world_event", Detail: "raid", EventName: "Harvest Raid", Xp: 60, Amount: 30}}})
	if got := lastText(m); !strings.Contains(got, "for the Harvest Raid") {
		t.Errorf("the reward line %q", got)
	}
	event(m, 700, &pb.Event{Body: &pb.Event_Reward{Reward: &pb.Reward{Recipient: "me", Kind: "world_event", Detail: "raid", Xp: 60, Amount: 30}}})
	if got := lastText(m); !strings.Contains(got, "for the Announced Raid") {
		t.Errorf("the reward line without the event's own name %q", got)
	}

	// Markers show the names the zone map gives; one without a name shows its id.
	m.Map = &pb.ZoneMap{Space: "downs", Markers: []*pb.Marker{{Id: "goblin-lookout", Kind: "scout", Name: "Goblin Lookout"}, {Id: "plain", Kind: "scout"}}}
	if m.MarkerName("goblin-lookout") != "Goblin Lookout" || m.MarkerName("plain") != "plain" || m.MarkerName("unknown") != "unknown" {
		t.Errorf("marker names: %q %q %q", m.MarkerName("goblin-lookout"), m.MarkerName("plain"), m.MarkerName("unknown"))
	}

	// With no companion in the snapshot the Captain's role option is dropped.
	event(m, 800, &pb.Event{Body: &pb.Event_Dialogue{Dialogue: &pb.Dialogue{Actor: "me", Npc: "captain",
		Options: []*pb.DialogueOption{{Action: "companion_role", Label: "Choose your recruit's role"}, {Action: "contract_accept", Label: "Take a contract"}}}}})
	if d := m.Said("captain"); d == nil || len(d.Options) != 1 || d.Options[0].GetAction() != "contract_accept" {
		t.Errorf("the Captain's options: %+v", d)
	}
}

func logText(m *model.Model) string {
	var b strings.Builder
	for _, e := range m.Log {
		b.WriteString(e.Text + "\n")
	}
	return b.String()
}

// The short log leaves out people coming into and going out of view: the model
// marks those lines quiet, and keeps them for the long view.
func TestNPCsInAndOutOfViewAreQuietLines(t *testing.T) {
	m := model.New()
	event(m, 2000, &pb.Event{Text: "Smith <Smith> comes into view", Body: &pb.Event_NpcAppeared{NpcAppeared: &pb.NpcAppeared{
		Npc: &pb.NpcInfo{Id: "smith", Name: "Smith", Role: "vendor", Pos: &pb.Pos{X: 9000, Y: 9000}}}}})
	event(m, 3000, &pb.Event{Text: "Smith is out of view", Body: &pb.Event_NpcLeft{NpcLeft: &pb.NpcLeft{Id: "smith"}}})
	if len(m.Log) != 2 {
		t.Fatalf("log: %+v", m.Log)
	}
	for _, e := range m.Log {
		if !e.Quiet {
			t.Errorf("%s %q is not quiet", e.Kind, e.Text)
		}
	}
}
