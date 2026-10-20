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
				Stirred: []string{"The Warren stirs: a lieutenant answers the Chief's call."}}}}},
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
	for _, want := range []string{"Slots: 4, 3 players", "1  Hale (you)  your account", "2  Bram  another player", "3  Cora  your account, other character", "4  empty slot", "replacement open between encounters"} {
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
