package model

import (
	"fmt"
	"testing"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func chat(channel, from, to, text string) *pb.ServerMessage {
	return &pb.ServerMessage{Body: &pb.ServerMessage_Chat{Chat: &pb.ChatMessage{Channel: channel, From: from, To: to, Text: text}}}
}

func notice(kind, from string) *pb.ServerMessage {
	return &pb.ServerMessage{Body: &pb.ServerMessage_Notice{Notice: &pb.SocialNotice{Kind: kind, From: from}}}
}

func partyOf(id, leader string, members ...string) *pb.ServerMessage {
	p := &pb.PartyState{Id: id, Leader: leader}
	for _, n := range members {
		p.Members = append(p.Members, &pb.PartyMemberInfo{Name: n, Online: true, Leader: n == leader})
	}
	return &pb.ServerMessage{Body: &pb.ServerMessage_Party{Party: p}}
}

func TestChatIsKeptWithItsChannelAndOwnerAndBounded(t *testing.T) {
	m := New()
	m.Self.Name = "Ann"
	m.Apply(chat("say", "ann", "", "hello"))
	m.Apply(chat("tell", "bob", "ann", "psst"))
	m.Apply(chat("tell", "ann", "bob", "hi bob"))
	if len(m.Social.Chat) != 3 || !m.Social.Chat[0].Mine || m.Social.Chat[1].Mine || !m.Social.Chat[2].Mine {
		t.Fatalf("chat: %+v", m.Social.Chat)
	}
	if m.Social.LastTell != "bob" {
		t.Fatalf("reply goes to %q", m.Social.LastTell)
	}
	for i, want := range []string{"[say] ann: hello", "[tell] bob: psst", "[tell] to bob: hi bob"} {
		if got := m.Social.Chat[i].String(); got != want {
			t.Errorf("line %d: %q, want %q", i, got, want)
		}
	}
	for range ChatCap + 10 {
		m.Apply(chat("say", "bob", "", "x"))
	}
	if len(m.Social.Chat) != ChatCap {
		t.Fatalf("the scrollback holds %d lines", len(m.Social.Chat))
	}
}

func TestAnInvitationWaitsForAnAnswerAndAPartyStateClearsIt(t *testing.T) {
	m := New()
	m.Self.Name = "bob"
	m.Apply(notice("party_invite", "ann"))
	if m.Social.Invite == nil || m.Social.Invite.From != "ann" {
		t.Fatalf("invite: %+v", m.Social.Invite)
	}
	if got := m.Log[len(m.Log)-1]; got.Kind != "notice_party_invite" || got.Text != "ann invites you to a party" {
		t.Fatalf("log: %+v", got)
	}
	m.Apply(partyOf("p1", "ann", "ann", "bob"))
	if m.Social.Invite != nil || !m.InParty() || m.IsLeader() {
		t.Fatalf("after joining: invite %+v party %v", m.Social.Invite, m.Social.Party)
	}
	m.Apply(notice("party_invite", "cy"))
	m.ClearInvite()
	if m.Social.Invite != nil {
		t.Fatal("an answered invitation stays")
	}
	m.Apply(partyOf("", ""))
	if m.InParty() {
		t.Fatal("an empty party id is no party")
	}
}

func TestMatesCarryThePartyStatesVitals(t *testing.T) {
	m := New()
	m.Self = Self{ID: "ann1", Name: "ann1", Level: 3, HP: 10, MaxHP: 20}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Party{Party: &pb.PartyState{Id: "p1", Leader: "ann1", Members: []*pb.PartyMemberInfo{
		{Name: "ann1", Online: true, Leader: true},
		{Name: "bob1", Online: true, Zone: "Testgrounds", Level: 4, Hp: 15, MaxHp: 30, Mana: 5, MaxMana: 9},
		{Name: "cy1", Online: true, Zone: "Downs", Level: 6, Hp: 0, MaxHp: 40},
		{Name: "dan1"}}}}})
	mates := m.Mates()
	if len(mates) != 4 || !mates[0].Me || !mates[0].Leader || mates[0].Level != 3 {
		t.Fatalf("mates: %+v", mates)
	}
	if b := mates[1]; b.Zone != "Testgrounds" || b.Level != 4 || b.HP != 15 || b.MaxMana != 9 || b.Leader {
		t.Fatalf("bob: %+v", b)
	}
	if c := mates[2]; c.Zone != "Downs" || c.Level != 6 || !c.Dead {
		t.Fatalf("cy: %+v", c)
	}
	if d := mates[3]; d.Online || d.Zone != "" {
		t.Fatalf("dan: %+v", d)
	}
	if got := m.PartyIDs(); len(got) != 3 || got[0] != "ann1" {
		t.Fatalf("ids %v", got)
	}
}

func TestFriendNoticesAreInTheLogAndMoveTheList(t *testing.T) {
	m := New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Friends{Friends: &pb.FriendList{Friends: []*pb.FriendEntry{{Name: "cy1", Online: true, Zone: "Downs"}, {Name: "dan1"}}}}})
	m.Apply(notice("friend_offline", "cy1"))
	if m.Social.Friends[0].GetOnline() || m.Log[len(m.Log)-1].Text != "friend cy1 logged out" {
		t.Fatalf("after logout: %v %v", m.Social.Friends, m.Log)
	}
	m.Apply(notice("friend_online", "DAN1"))
	if !m.Social.Friends[1].GetOnline() || m.Log[len(m.Log)-1].Text != "friend DAN1 logged in" {
		t.Fatalf("after login: %v %v", m.Social.Friends, m.Log)
	}
}

func TestSocialReasonsAreInWords(t *testing.T) {
	for _, r := range []string{"rate_limited", "not_online", "no_such_character", "not_in_party", "not_leader", "party_full", "no_invitation"} {
		if got := SocialReason(r); got == r || len(got) < len(r) {
			t.Errorf("%s: %q", r, got)
		}
	}
	if got := SocialReason("something_new"); got != "something_new" {
		t.Errorf("an unknown reason is shown as sent, not %q", got)
	}
}

func TestPartyVitalsChangeOnlyTheFieldsTheyCarry(t *testing.T) {
	m := New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Party{Party: &pb.PartyState{Id: "p", Leader: "ann1", Members: []*pb.PartyMemberInfo{
		{Name: "ann1", Online: true, Zone: "Testgrounds", Level: 3, Hp: 90, MaxHp: 100, Mana: 20, MaxMana: 30}}}}})
	before := m.Social.Party
	hp := int32(55)
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_PartyVitals{PartyVitals: &pb.PartyVitals{Name: "Ann1", Hp: &hp}}})
	mt := m.Mates()[0]
	if mt.HP != 55 || mt.MaxHP != 100 || mt.Mana != 20 || mt.MaxMana != 30 || mt.Zone != "Testgrounds" || mt.Level != 3 {
		t.Fatalf("mate: %+v", mt)
	}
	if before.Members[0].Hp != 90 {
		t.Fatal("the party handed out earlier was edited")
	}
	// A member the party does not have, and no party at all: nothing happens.
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_PartyVitals{PartyVitals: &pb.PartyVitals{Name: "zed", Hp: &hp}}})
	New().Apply(&pb.ServerMessage{Body: &pb.ServerMessage_PartyVitals{PartyVitals: &pb.PartyVitals{Name: "zed", Hp: &hp}}})
}

// D101: every line is numbered, a mention is found when it arrives, and the other
// side of the latest tell is kept either way.
func TestChatLinesAreNumberedAndFlaggedForMentionsAndTells(t *testing.T) {
	m := New()
	m.Self.Name = "Ann"
	m.Apply(chat("say", "bob", "", "hello ANN, well met"))
	m.Apply(chat("say", "ann", "", "ann says hi"))
	m.Apply(chat("tell", "ann", "cy", "psst"))
	m.Apply(chat("tell", "bob", "ann", "back"))
	m.Apply(chat("say", "bob", "", "Anna?"))
	var seqs []int
	var mentions []bool
	for _, l := range m.Social.Chat {
		seqs, mentions = append(seqs, l.Seq), append(mentions, l.Mention)
	}
	if want := []int{1, 2, 3, 4, 5}; !equalInts(seqs, want) || m.Social.ChatN != 5 {
		t.Errorf("numbers %v, ChatN %d", seqs, m.Social.ChatN)
	}
	if want := []bool{true, false, false, false, false}; fmt.Sprint(mentions) != fmt.Sprint(want) {
		t.Errorf("mentions %v: own lines and 'Anna' are not mentions", mentions)
	}
	if m.Social.LastTell != "bob" || m.Social.TellPeer != "bob" {
		t.Errorf("last tell %q peer %q", m.Social.LastTell, m.Social.TellPeer)
	}
	m.Apply(chat("tell", "ann", "cy", "psst"))
	if m.Social.LastTell != "bob" || m.Social.TellPeer != "cy" {
		t.Errorf("after telling cy: last tell %q peer %q", m.Social.LastTell, m.Social.TellPeer)
	}
	if all, party, tell := m.ChatSince(2); all != 2 || party != 0 || tell != 1 {
		t.Errorf("since 2: %d %d %d (own lines are not counted)", all, party, tell)
	}
}

func TestCatchUpWordsWhatCameInAFight(t *testing.T) {
	for _, c := range []struct {
		all, party, tell int
		want             string
	}{
		{0, 0, 0, ""},
		{3, 0, 0, "While fighting: 3 chat"},
		{4, 2, 0, "While fighting: 4 chat (2 party)"},
		{4, 0, 1, "While fighting: 4 chat (1 tells)"},
		{6, 2, 3, "While fighting: 6 chat (2 party, 3 tells)"},
	} {
		if got := CatchUp(c.all, c.party, c.tell); got != c.want {
			t.Errorf("CatchUp(%d, %d, %d) = %q", c.all, c.party, c.tell, got)
		}
	}
}

func TestTheLogCountsTheLinesTheShortViewShows(t *testing.T) {
	m := New()
	m.Notice("one")
	m.Notice("two")
	m.Log = append(m.Log, LogEntry{Kind: "entity_moved", Text: "quiet"})
	m.trimLog()
	if m.LogN != 2 {
		t.Errorf("LogN %d: the routine line is not counted", m.LogN)
	}
}

func equalInts(a, b []int) bool { return fmt.Sprint(a) == fmt.Sprint(b) }

func structuredLine(channel, from string, parts ...*pb.ChatPart) *pb.ServerMessage {
	return &pb.ServerMessage{Body: &pb.ServerMessage_Chat{Chat: &pb.ChatMessage{Channel: channel, From: from, Text: "x",
		Speaker: &pb.CharacterRef{Id: "s", Name: from, Kind: "gm"}, Parts: parts}}}
}

func TestStructuredChatKeepsItsPartsAndMentionsComeFromThem(t *testing.T) {
	ref := func(id, name, kind string) *pb.ChatPart {
		return &pb.ChatPart{Body: &pb.ChatPart_Character{Character: &pb.CharacterRef{Id: id, Name: name, Kind: kind}}}
	}
	txt := func(s string) *pb.ChatPart { return &pb.ChatPart{Body: &pb.ChatPart_Text{Text: s}} }
	item := &pb.ChatPart{Body: &pb.ChatPart_Item{Item: &pb.ItemRef{Id: "sw", Name: "sword"}}}
	for _, c := range []struct {
		name    string
		channel string
		parts   []*pb.ChatPart
		want    []ChatPart
		mention bool
	}{
		{"a reference to you", "say", []*pb.ChatPart{txt("hi "), ref("me", "Aria", "player")}, []ChatPart{{Text: "hi "}, {Text: "Aria", ID: "me", Kind: "player"}}, true},
		{"your name by name only, in any case", "ooc", []*pb.ChatPart{ref("", "aria", "")}, []ChatPart{{Text: "aria", Kind: "player"}}, true},
		{"someone else", "say", []*pb.ChatPart{ref("mira", "Mira", "npc"), item, {}}, []ChatPart{{Text: "Mira", ID: "mira", Kind: "npc"}, {Text: "sword", ID: "sw", Kind: "item"}}, false},
		{"text that only looks like your name is not a reference", "say", []*pb.ChatPart{txt("Aria")}, []ChatPart{{Text: "Aria"}}, false},
	} {
		m := New()
		m.Self.ID, m.Self.Name = "me", "Aria"
		m.Apply(structuredLine(c.channel, "Hale", c.parts...))
		l := m.Social.Chat[0]
		if !l.Structured() || l.SpeakerKind != "gm" || l.Mention != c.mention || len(l.Parts) != len(c.want) {
			t.Errorf("%s: %+v", c.name, l)
			continue
		}
		for i, p := range l.Parts {
			if p != c.want[i] {
				t.Errorf("%s: part %d is %+v, want %+v", c.name, i, p, c.want[i])
			}
		}
	}
}

func TestAnOldServersLineHasNoPartsAndKeepsTheTextMention(t *testing.T) {
	m := New()
	m.Self.ID, m.Self.Name = "me", "Aria"
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Chat{Chat: &pb.ChatMessage{Channel: "ooc", From: "Bram", Text: "hi aria!"}}})
	l := m.Social.Chat[0]
	if l.Structured() || !l.Mention || l.String() != "[ooc] Bram: hi aria!" {
		t.Errorf("%+v / %q", l, l.String())
	}
}

func TestTheServersReportAcknowledgementIsCountedAndLogged(t *testing.T) {
	m := New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Notice{Notice: &pb.SocialNotice{Kind: "report_filed", Text: "Your report about Bram has been filed (report 1)."}}})
	if m.Social.ReportAck != 1 || m.Log[len(m.Log)-1].Text != "Your report about Bram has been filed (report 1)." {
		t.Errorf("ack %d log %+v", m.Social.ReportAck, m.Log)
	}
	for reason, want := range map[string]string{"ooc_off": "ooc is switched off: /ooc on turns it back on"} {
		if got := SocialReason(reason); !strings.Contains(got, want) {
			t.Errorf("%s: %q", reason, got)
		}
	}
}
