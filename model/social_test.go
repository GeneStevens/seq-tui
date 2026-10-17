package model

import (
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
