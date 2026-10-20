package screen_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// D79 on the screen, from messages the server sends: nothing here needs a
// server, so the cases the live scenario cannot reach (an encounter open, a
// refusal in the server's words, the help page) are held here.

func apply(m *model.Model, msgs ...*pb.ServerMessage) {
	for _, msg := range msgs {
		m.Apply(msg)
	}
}

func event(seq uint64, body func(*pb.Event)) *pb.ServerMessage {
	e := &pb.Event{Seq: seq, GameTimeMs: int64(seq) * 100}
	body(e)
	return &pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: e}}
}

var warrenPreset = &pb.InstancePreset{
	Id: "peril-warren", Name: "The Warren (Peril)", Tier: "peril", PartyCap: 4, TimeMode: "realtime", Difficulty: "hard",
	PartySize: []*pb.PartySizeRule{{Size: 1}, {Size: 2, MobPct: 120, MobHpPct: 110}, {Size: 4, MobPct: 150}},
	GroupTier: &pb.GroupTierRule{Id: "peril-together", Name: "Peril together", MinPlayers: 2, Bonus: "bonus-roll"},
}

// insideTheStirredWarren is hale in a delve of three slots held by hale, bram and
// a new player, cora; one slot is empty.
func insideTheStirredWarren() *model.Model {
	m := model.New()
	char := func(id, name string, x int64) *pb.EntityState {
		return &pb.EntityState{Id: id, Name: name, Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, Pos: &pb.Pos{X: x, Y: 10000}, Hp: 100, MaxHp: 100}
	}
	apply(m,
		&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: "inst-1", Name: "Warren", ViewRadiusMm: 20000, InstancePresets: []*pb.InstancePreset{warrenPreset},
			Bounds: &pb.Rect{MinX: 0, MinY: 0, MaxX: 40000, MaxY: 40000}}}},
		&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "inst-1", Seq: 1, GameTimeMs: 1000,
			Self:  &pb.SelfState{Entity: char("hale", "Hale", 10000)},
			Group: []*pb.GroupMember{{Entity: char("bram", "Bram", 11000)}, {Entity: char("cora", "Cora", 12000)}},
			Instance: &pb.InstanceInfo{Preset: "peril-warren", PartyCap: 4, TimeMode: "realtime", Difficulty: "hard", Slots: 4, Players: 3, GroupTier: "peril-together",
				Stirred: []string{"The Warren stirs: a lieutenant answers the Chief's call."},
				Holders: []*pb.SlotHolder{
					{Slot: 1, Player: "Player 1", Character: "hale", Name: "Hale", State: "held"},
					{Slot: 2, Player: "Player 2", Character: "bram", Name: "Bram", State: "held"},
					{Slot: 3, Player: "Player 1", Character: "cora", Name: "Cora", State: "held"},
					{Slot: 4, Player: "Player 3", Name: "Dane", State: "empty"}}}}}},
	)
	return m
}

func TestTheInstancePanelSaysTheTierThePlayersAndWhatItStirred(t *testing.T) {
	m := insideTheStirredWarren()
	for _, sz := range screen.Sizes {
		out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, sz[0], sz[1], screen.Plain())
		for _, want := range []string{"Peril together", "stirred: The Warren"} {
			if !strings.Contains(out, want) {
				t.Errorf("%dx%d lacks %q:\n%s", sz[0], sz[1], want, out)
			}
		}
	}
	// With the group tier not reached the panel says what would reach it.
	m.Instance.GroupTier, m.Instance.Players, m.Instance.Stirred = "", 1, nil
	out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain())
	if !strings.Contains(out, "no group tier | 1 player") || !strings.Contains(out, "Peril together needs 2") {
		t.Errorf("the lone player is not told:\n%s", out)
	}
}

func TestTheSlotsAreListedAndReplacementOpensAndWaits(t *testing.T) {
	m := insideTheStirredWarren()
	m.SetOwn("Cora") // the player's second character
	view := func() string { return screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain()) }
	out := view()
	for _, want := range []string{"Slots: 4, 3 players", "1  Hale (you)  Player 1, you", "2  Bram  Player 2", "3  Cora  Player 1, yours", "4  empty  was Player 3's, open", "replacement open between encounters"} {
		if !strings.Contains(out, want) {
			t.Errorf("between encounters lacks %q:\n%s", want, out)
		}
	}
	apply(m, event(2, func(e *pb.Event) {
		e.Body = &pb.Event_EncounterOpened{EncounterOpened: &pb.EncounterOpened{Id: "enc-1", Mob: "mob-1", Owner: "hale"}}
	}))
	if out = view(); !strings.Contains(out, "replacement waits for the encounter") || strings.Contains(out, "replacement open between") {
		t.Errorf("during an encounter:\n%s", out)
	}
	apply(m, event(3, func(e *pb.Event) {
		e.Body = &pb.Event_EncounterClosed{EncounterClosed: &pb.EncounterClosed{Id: "enc-1", Mob: "mob-1"}}
	}))
	if out = view(); !strings.Contains(out, "replacement open between encounters") {
		t.Errorf("after it:\n%s", out)
	}
}

// A refusal shows the server's own words in the Instance panel, whether it came
// as a result of the door, a denied event or a failed opening.
func TestRefusalsShowTheServersText(t *testing.T) {
	for _, tc := range []struct {
		name string
		msg  *pb.ServerMessage
		want string
	}{
		{"mid-encounter join", &pb.ServerMessage{Body: &pb.ServerMessage_IntentResult{IntentResult: &pb.IntentResult{IntentId: "i1", Reason: "encounter_open", Text: "A fight is under way: join between encounters"}}}, "A fight is under way"},
		{"a slot of another account", event(5, func(e *pb.Event) {
			e.Body = &pb.Event_Denied{Denied: &pb.Denied{Actor: "me", Reason: "slot_account", Text: "That slot was an account's who left; yours is in already"}}
		}), "That slot was an account's"},
		{"one expedition each", event(6, func(e *pb.Event) {
			e.Body = &pb.Event_InstanceOpenFailed{InstanceOpenFailed: &pb.InstanceOpenFailed{Reason: "instance_exists", Instance: "inst-9", Name: "The Warren (Venture)"}}
		}), "You are already on an"},
	} {
		m := model.New()
		apply(m, &pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "Z", Seq: 1, GameTimeMs: 1000,
			Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Pos: &pb.Pos{X: 10000, Y: 10000}, Hp: 10, MaxHp: 10}}}}}, tc.msg)
		out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain())
		if !strings.Contains(out, "refused: ") || !strings.Contains(out, tc.want) || m.Refusal == nil {
			t.Errorf("%s: the panel lacks %q:\n%s", tc.name, tc.want, out)
		}
	}
}

func TestStatusShowsInYouTheWhoListAndTheParty(t *testing.T) {
	m := insideTheStirredWarren()
	apply(m,
		&pb.ServerMessage{Body: &pb.ServerMessage_Status{Status: &pb.StatusState{Status: "autopilot"}}},
		&pb.ServerMessage{Body: &pb.ServerMessage_Party{Party: &pb.PartyState{Id: "p", Leader: "Hale", Members: []*pb.PartyMemberInfo{
			{Name: "Hale", Online: true, Leader: true}, {Name: "Bram", Online: true, Zone: "Warren", Level: 5, Status: "away"}}}}})
	out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain())
	for _, want := range []string{"Status: on autopilot", "Bram [away]"} {
		if !strings.Contains(out, want) {
			t.Errorf("lacks %q:\n%s", want, out)
		}
	}
	who := screen.Render(m, screen.View{Conn: screen.ConnOnline, Social: &screen.SocialUI{Tab: screen.TabWho, Who: []*pb.WhoEntry{{Name: "Bram", ClassId: "cleric", Level: 5, Zone: "Warren", Status: "away"}}}}, 140, 45, screen.Plain())
	if !strings.Contains(who, "Warren [away]") {
		t.Errorf("who lacks the status:\n%s", who)
	}
	apply(m, &pb.ServerMessage{Body: &pb.ServerMessage_Status{Status: &pb.StatusState{}}})
	if out = screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain()); strings.Contains(out, "Status:") {
		t.Errorf("a cleared status is still shown:\n%s", out)
	}
}

func TestTheGroupRulesPageIsDrawnFromTheServersMenu(t *testing.T) {
	m := insideTheStirredWarren()
	page := -1
	for i := 0; i < screen.HelpPages(); i++ {
		if strings.Contains(screen.HelpTitle(i), "Group rules") {
			page = i
		}
	}
	if page < 0 {
		t.Fatal("the help has no Group rules page")
	}
	out := screen.Render(m, screen.View{Conn: screen.ConnOnline, Help: true, HelpPage: page}, 140, 45, screen.Plain())
	for _, want := range []string{"The Warren (Peril), Peril tier, party up to 4", "Party of 2: mobs at 120%, mob health at 110%", "Party of 4: mobs at 150%", `Group tier "Peril together": 2 or more players`, "One entitlement per", "/status away", "no reward effect"} {
		if !strings.Contains(out, want) {
			t.Errorf("the page lacks %q:\n%s", want, out)
		}
	}
	for _, sz := range screen.Sizes {
		if got := screen.Render(m, screen.View{Conn: screen.ConnOnline, Help: true, HelpPage: page}, sz[0], sz[1], screen.Plain()); strings.Count(got, "\n")+1 != sz[1] {
			t.Errorf("%v: the page does not fit", sz)
		}
	}
}

// The slot list is the server's (D82): an instance_slots event replaces it, a
// slot nobody has entered is reserved, and a replacement who fills the empty
// slot is told the tier it joined.
func TestSlotHoldersFollowTheServerAndAReservedSlotIsToldAsOne(t *testing.T) {
	m := insideTheStirredWarren()
	view := func() string { return screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain()) }
	apply(m, event(2, func(e *pb.Event) {
		e.Body = &pb.Event_InstanceSlots{InstanceSlots: &pb.InstanceSlots{Actor: "dane", Holders: []*pb.SlotHolder{
			{Slot: 1, Player: "Player 1", Character: "hale", Name: "Hale", State: "held"},
			{Slot: 2, Player: "Player 2", Character: "bram", Name: "Bram", State: "held"},
			{Slot: 3, Player: "Player 1", Character: "cora", Name: "Cora", State: "held"},
			{Slot: 4, Player: "Player 3", Character: "dane", Name: "Dane", State: "held"}}}}
	}))
	out := view()
	for _, want := range []string{"4  Dane  Player 3", "replacement open; no slot is empty"} {
		if !strings.Contains(out, want) {
			t.Errorf("after the slot was taken lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "empty  Player 3") {
		t.Errorf("the slot taken is still shown empty:\n%s", out)
	}
	apply(m, event(3, func(e *pb.Event) {
		e.Body = &pb.Event_InstanceSlots{InstanceSlots: &pb.InstanceSlots{Holders: []*pb.SlotHolder{
			{Slot: 1, Player: "Player 1", Character: "hale", Name: "Hale", State: "held"},
			{Slot: 2, Player: "Player 2", State: "reserved"}}}}
	}))
	if out = view(); !strings.Contains(out, "2  reserved  Player 2, not yet in") {
		t.Errorf("a reserved slot is not told:\n%s", out)
	}
}

// A replacement is told what it joined, in the server's words, and the Instance
// panel carries the tier and the Stir state from the instance it joined.
func TestAReplacementIsToldTheTierAndTheStir(t *testing.T) {
	m := insideTheStirredWarren()
	apply(m, event(2, func(e *pb.Event) {
		e.Text = "You take a place in The Warren (Peril): Peril tier, 3 players, group tier Peril together, stirred"
		e.Body = &pb.Event_InstanceJoined{InstanceJoined: &pb.InstanceJoined{Actor: "hale", Instance: "inst-1", Preset: "peril-warren", Tier: "peril", Players: 3, GroupTier: "peril-together", Stirred: 1, Slots: 4}}
	}))
	var got string
	for _, l := range m.Log {
		if l.Kind == "instance_joined" {
			got = l.Text
		}
	}
	if !strings.HasPrefix(got, "You take a place in The Warren (Peril): Peril tier") {
		t.Errorf("the log says %q", got)
	}
	if m.Joined == nil || m.Joined.GroupTier != "peril-together" || m.Joined.Stirred != 1 {
		t.Errorf("joined %+v", m.Joined)
	}
	out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain())
	for _, want := range []string{"Peril together | 3 players", "stirred: The Warren stirs"} {
		if !strings.Contains(out, want) {
			t.Errorf("the panel lacks %q:\n%s", want, out)
		}
	}
	// With words left out the client says it in its own.
	m2 := insideTheStirredWarren()
	apply(m2, event(2, func(e *pb.Event) {
		e.Body = &pb.Event_InstanceJoined{InstanceJoined: &pb.InstanceJoined{Preset: "peril-warren", Tier: "peril", Players: 3, GroupTier: "Peril together", Stirred: 1}}
	}))
	if l := m2.Log[len(m2.Log)-1]; l.Kind != "instance_joined" || l.Text != "You take a place in The Warren (Peril): peril tier, 3 players, group tier Peril together, stirred" {
		t.Errorf("fallback words %+v", l)
	}
}

// A personal reward is shown in the server's first-person words, with the item
// when it names one; the account's other character taking it is told too.
func TestPersonalRewardsAreShownInTheServersFirstPerson(t *testing.T) {
	m := insideTheStirredWarren()
	apply(m,
		event(2, func(e *pb.Event) {
			e.Text = "Your account's bonus roll for the Hollow Warren's Chief: Pate Cleaver."
			e.Body = &pb.Event_Reward{Reward: &pb.Reward{Recipient: "hale", Kind: "personal", Detail: "bonus-roll", Item: "pate-cleaver", Entitlement: "inst-1.enc-4/entitlement/bonus-roll"}}
		}),
		event(3, func(e *pb.Event) {
			e.Text = "Your other character, Cora, took this account's bonus roll"
			e.Body = &pb.Event_EntitlementTaken{EntitlementTaken: &pb.EntitlementTaken{Actor: "hale", Taker: "cora", Entitlement: "inst-1.enc-4/entitlement/bonus-roll", Kind: "bonus-roll"}}
		}),
		event(4, func(e *pb.Event) {
			e.Body = &pb.Event_EntitlementTaken{EntitlementTaken: &pb.EntitlementTaken{Actor: "hale", Taker: "cora", Kind: "bonus-roll"}}
		}),
	)
	var got []string
	for _, l := range m.Log {
		if l.Kind == "reward" || l.Kind == "entitlement_taken" {
			got = append(got, l.Text)
		}
	}
	want := []string{
		"Your account's bonus roll for the Hollow Warren's Chief: Pate Cleaver.",
		"Your other character, Cora, took this account's bonus roll",
		"Your other character, Cora, took this account's bonus roll",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
