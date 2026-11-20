package screen_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The Nearby list says whom a mob is attacking (D213): "→ you", "→ <party member>",
// nothing for anyone else, and the mark follows target_changed and target_cleared.
func TestNearbyMarksWhomAMobAttacks(t *testing.T) {
	m := model.New()
	ent := func(id, name string, kind pb.EntityKind, x int64, target string) *pb.EntityState {
		return &pb.EntityState{Id: id, Name: name, Kind: kind, Pos: &pb.Pos{X: x, Y: 10000}, Hp: 10, MaxHp: 10, Target: target}
	}
	const mob, chr = pb.EntityKind_ENTITY_KIND_MOB, pb.EntityKind_ENTITY_KIND_CHARACTER
	apply(m,
		&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: "z", Name: "Dale", ViewRadiusMm: 20000, Bounds: &pb.Rect{MaxX: 40000, MaxY: 40000}}}},
		&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 1, GameTimeMs: 1000,
			Self:  &pb.SelfState{Entity: ent("hale", "Hale", chr, 10000, "")},
			Group: []*pb.GroupMember{{Entity: ent("cora", "Cora", chr, 10500, "")}},
			Entities: []*pb.EntityState{
				ent("m1", "Rat", mob, 11000, "hale"),
				ent("m2", "Snake", mob, 11100, "cora"),
				ent("m3", "Wolf", mob, 11200, "stranger"),
				ent("m4", "Bat", mob, 11300, ""),
				ent("m5", "Ancient Cave Spider Matriarch", mob, 10200, "hale"),
			}}}})
	rows := func() map[string]string {
		out := map[string]string{}
		for _, l := range strings.Split(screen.Render(m, screen.View{Conn: screen.ConnOnline}, 100, 45, screen.Plain()), "\n") {
			for _, n := range []string{"Rat", "Snake", "Wolf", "Bat", "Ancien"} {
				if strings.Contains(l, n) && strings.Contains(l, "m ") {
					out[n] = l
				}
			}
		}
		return out
	}
	for name, want := range map[string]string{"Rat": "→ you", "Snake": "→ Cora", "Wolf": "", "Bat": "", "Ancien": "→ you"} {
		l := rows()[name]
		if l == "" || strings.Contains(l, "→") != (want != "") || !strings.Contains(l, want) {
			t.Errorf("%s: want %q in %q\n%s", name, want, l, screen.Render(m, screen.View{Conn: screen.ConnOnline}, 100, 45, screen.Plain()))
		}
	}
	apply(m, event(2, func(e *pb.Event) {
		e.Body = &pb.Event_TargetChanged{TargetChanged: &pb.TargetChanged{Actor: "m4", Target: "cora"}}
	}), event(3, func(e *pb.Event) {
		e.Body = &pb.Event_TargetCleared{TargetCleared: &pb.TargetCleared{Actor: "m1", Target: "hale"}}
	}))
	if r := rows(); !strings.Contains(r["Bat"], "→ Cora") || strings.Contains(r["Rat"], "→") {
		t.Errorf("after the events: %q / %q", r["Bat"], r["Rat"])
	}
}
