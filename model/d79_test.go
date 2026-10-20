package model_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func inAnInstance(t *testing.T) *model.Model {
	t.Helper()
	m := model.New()
	char := func(id, name string) *pb.EntityState {
		return &pb.EntityState{Id: id, Name: name, Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, Hp: 10, MaxHp: 10, Pos: &pb.Pos{X: 1000, Y: 1000}}
	}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: "inst-1", InstancePresets: []*pb.InstancePreset{{
		Id: "warren", Name: "The Warren", GroupTier: &pb.GroupTierRule{Id: "together", Name: "Together", MinPlayers: 2, Bonus: "rare-roll"}}}}}})
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "inst-1", Seq: 1,
		Self:     &pb.SelfState{Entity: char("hale", "Hale")},
		Group:    []*pb.GroupMember{{Entity: char("zehen", "Zehen")}, {Entity: char("cass", "Cass")}},
		Instance: &pb.InstanceInfo{Preset: "warren", Slots: 4, Players: 2, GroupTier: "together"}}}})
	return m
}

func reward(m *model.Model, seq uint64, to string) {
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: seq, Body: &pb.Event_Reward{Reward: &pb.Reward{Recipient: to, Kind: "personal", Amount: 1}}}}})
}

// The entitlement notice names the reward, and says which character used it:
// this one, the player's other character, or somebody else's.
func TestTheEntitlementNoticeSaysWhoUsedIt(t *testing.T) {
	m := inAnInstance(t)
	m.SetOwn("Zehen")
	reward(m, 2, "hale")
	reward(m, 3, "zehen")
	reward(m, 4, "cass")
	var got []string
	for _, l := range m.Log {
		if l.Kind == "reward" {
			got = append(got, l.Text)
		}
	}
	want := []string{
		"Your account's rare roll for this encounter went to you",
		"Your account's rare roll for this encounter went to your other character, Zehen",
		"Cass used their own rare roll for this encounter",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestSlotsHoldersAndWhenAReplacementMayJoin(t *testing.T) {
	m := inAnInstance(t)
	m.SetOwn("zehen")
	slots := m.Slots()
	if len(slots) != 4 || !slots[0].Me || slots[0].Holder != "Hale" || slots[1].Own || slots[1].Holder != "Cass" || !slots[2].Own || slots[2].Holder != "Zehen" || slots[3].Holder != "" {
		t.Fatalf("slots %+v", slots)
	}
	if m.Vacant() != 1 || m.EncounterOpen() || m.ReplacementWords() != "replacement open between encounters" {
		t.Errorf("between encounters: vacant %d, %q", m.Vacant(), m.ReplacementWords())
	}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: 2, Body: &pb.Event_EncounterOpened{EncounterOpened: &pb.EncounterOpened{Id: "e", Mob: "mob", Owner: "hale"}}}}})
	if !m.EncounterOpen() || m.ReplacementWords() != "replacement waits for the encounter" {
		t.Errorf("in an encounter: %q", m.ReplacementWords())
	}
}

// A refusal stays until an instance opens or the character goes in; a denial of
// another sort is no refusal of the expedition rules.
func TestARefusalStaysUntilTheCharacterIsIn(t *testing.T) {
	m := model.New()
	deny := func(seq uint64, reason, text string) {
		m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_IntentResult{IntentResult: &pb.IntentResult{IntentId: "i", Reason: reason, Text: text}}})
	}
	deny(1, "out_of_range", "too far")
	if m.Refusal != nil {
		t.Fatalf("an unrelated denial is a refusal: %+v", m.Refusal)
	}
	deny(2, "expedition_active:The Warren", "")
	if m.Refusal == nil || m.Refusal.Text != "You are already on an expedition: The Warren. Finish it or leave it first" {
		t.Fatalf("refusal %+v", m.Refusal)
	}
	deny(3, "slot_account", "the server's own words")
	if m.Refusal.Text != "the server's own words" {
		t.Errorf("the server's words were replaced: %q", m.Refusal.Text)
	}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "inst-2", Seq: 1,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "hale", Name: "Hale"}}, Instance: &pb.InstanceInfo{Preset: "warren"}}}})
	if m.Refusal != nil {
		t.Errorf("the refusal outlived going in: %+v", m.Refusal)
	}
}

func TestStirredTextsAreKeptOnceAndForgottenWhenTheInstanceEnds(t *testing.T) {
	m := inAnInstance(t)
	stir := func(seq uint64) {
		m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: seq, Body: &pb.Event_InstanceStirred{InstanceStirred: &pb.InstanceStirred{Text: "The Warren stirs."}}}}})
	}
	stir(2)
	stir(3)
	if got := m.Stirred(); len(got) != 1 || got[0] != "The Warren stirs." {
		t.Errorf("stirred %q", got)
	}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: 4, Body: &pb.Event_InstanceEnded{InstanceEnded: &pb.InstanceEnded{Instance: "inst-1", Reason: "cleared"}}}}})
	m.Instance = nil
	if len(m.Stirred()) != 0 {
		t.Errorf("stirred after the end: %q", m.Stirred())
	}
}
