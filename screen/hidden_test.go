package screen_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Hidden (D211): a hidden game master reads "Hidden" on the status line, and another
// game master sees "(hidden)" beside its name in nearby and in who.
func TestHiddenShowsOnTheStatusLineNearbyAndWho(t *testing.T) {
	m := model.New()
	char := func(id, name string, x int64, hidden bool) *pb.EntityState {
		return &pb.EntityState{Id: id, Name: name, Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, Pos: &pb.Pos{X: x, Y: 10000}, Hp: 100, MaxHp: 100, Hidden: hidden}
	}
	apply(m,
		&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: "z", Name: "Dale", ViewRadiusMm: 20000, Bounds: &pb.Rect{MaxX: 40000, MaxY: 40000}}}},
		&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 1, GameTimeMs: 1000,
			Self:     &pb.SelfState{Entity: char("hale", "Hale", 10000, false)},
			Entities: []*pb.EntityState{char("bram", "Bram", 11000, true), char("cora", "Cora", 12000, false)}}}})
	render := func(ui *screen.SocialUI) string {
		return screen.Render(m, screen.View{Conn: screen.ConnOnline, Social: ui}, 140, 45, screen.Plain())
	}
	if out := render(nil); strings.Count(out, "(hidden)") != 1 || strings.Contains(out, "| Hidden") {
		t.Errorf("nearby should mark bram alone:\n%s", out)
	}
	who := &screen.SocialUI{Tab: screen.TabWho, Who: []*pb.WhoEntry{{Name: "Bram", ClassId: "cleric", Level: 5, Zone: "Warren", Hidden: true}, {Name: "Cora", ClassId: "cleric", Level: 5, Zone: "Warren"}}}
	if out := render(who); !strings.Contains(out, "Warren (hidden)") || strings.Count(out, "Warren (hidden)") != 1 {
		t.Errorf("who rows:\n%s", out)
	}
	apply(m, event(2, func(e *pb.Event) {
		e.Body = &pb.Event_HiddenChanged{HiddenChanged: &pb.HiddenChanged{Actor: "hale", On: true}}
	}))
	if out := render(nil); !strings.Contains(out, "| Hidden") {
		t.Errorf("the status line should say Hidden:\n%s", out)
	}
	apply(m, event(3, func(e *pb.Event) {
		e.Body = &pb.Event_HiddenChanged{HiddenChanged: &pb.HiddenChanged{Actor: "hale", On: false}}
	}))
	if out := render(nil); strings.Contains(out, "| Hidden") {
		t.Errorf("the status line should clear:\n%s", out)
	}
}
