package app

import (
	"testing"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func TestPartyKeyNamesWhoItTargets(t *testing.T) {
	for k, want := range map[string]int{"f1": 0, "f2": 1, "f6": 5, "f7": -1, "f10": -1, "f": -1, "1": -1, "?": -1} {
		if got := partyKey(k); got != want {
			t.Errorf("partyKey(%q) = %d, want %d", k, got, want)
		}
	}
}

// partyApp is Hale with Cora in view, Dova in another zone and Eli away.
func partyApp() (*App, *model.Model) {
	m := model.New()
	ent := func(id, name string) *pb.EntityState {
		return &pb.EntityState{Id: id, Name: name, Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, Pos: &pb.Pos{X: 1000}, Hp: 10, MaxHp: 10}
	}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "Z", Seq: 1, GameTimeMs: 1000,
		Self:  &pb.SelfState{Entity: ent("hale", "Hale")},
		Group: []*pb.GroupMember{{Entity: ent("cora", "Cora")}}}}})
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Party{Party: &pb.PartyState{Id: "p", Leader: "Hale", Members: []*pb.PartyMemberInfo{
		{Name: "Hale", Online: true, Leader: true},
		{Name: "Cora", Online: true, Zone: "Z", Level: 5, Hp: 40, MaxHp: 50},
		{Name: "Dova", Online: true, Zone: "Elsewhere", Level: 7, Hp: 20, MaxHp: 70},
		{Name: "Eli"}}}}})
	return New(client.NewSession(client.Config{Model: m})), m
}

func TestTheFunctionKeysTargetYouAndThePartyInThePanelsOrder(t *testing.T) {
	_, m := partyApp()
	for _, tc := range []struct {
		key  string
		want Action
	}{
		{"f1", Action{Kind: Target, ID: "hale"}},
		{"f2", Action{Kind: Target, ID: "cora"}},
		{"f3", Action{Note: "Dova is not in view"}},
		{"f4", Action{Note: "Eli is not in view"}},
		{"f5", Action{Note: "no party member for F5"}},
		{"f6", Action{Note: "no party member for F6"}},
	} {
		if got := Plan(tc.key, m); got.Kind != tc.want.Kind || got.ID != tc.want.ID || got.Note != tc.want.Note {
			t.Errorf("%s: %+v, want %+v", tc.key, got, tc.want)
		}
	}
	m.Self.Target = "cora"
	if got := Plan("f2", m); got.Kind != None || got.Note != "Cora is already your target" {
		t.Errorf("f2 on the target: %+v", got)
	}
}

func TestClickingAPartyLineTargetsThatMember(t *testing.T) {
	a, _ := partyApp()
	a.w, a.h = 140, 45
	a.refresh()
	want := map[string]Action{"Cora": {Kind: Target, ID: "cora"}, "Dova": {Note: "Dova is not in view"}}
	seen := map[string]bool{}
	for _, p := range a.mouse.hits.Party {
		w, ok := want[p.Name]
		if !ok {
			continue
		}
		seen[p.Name] = true
		if got, hit := a.partyClick(p.Rect.X+2, p.Rect.Y); !hit || got.Kind != w.Kind || got.ID != w.ID || got.Note != w.Note {
			t.Errorf("click on %s: %+v (hit %v), want %+v", p.Name, got, hit, w)
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("drawn lines: %+v", a.mouse.hits.Party)
	}
	if _, hit := a.partyClick(0, 0); hit {
		t.Error("a click off the panel hit a member")
	}
}
