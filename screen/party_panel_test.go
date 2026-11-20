package screen_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// partyWorld is Hale in Dale with a party: Cora (mana, in view), Brann (no
// mana, in view), Dova (mana, another zone) and Eli (offline).
func partyWorld() *model.Model {
	m := model.New()
	ent := func(id, name string, x int64) *pb.EntityState {
		return &pb.EntityState{Id: id, Name: name, Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, Pos: &pb.Pos{X: x, Y: 10000}, Hp: 10, MaxHp: 10}
	}
	apply(m,
		&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: "z", Name: "Dale", ViewRadiusMm: 20000, Bounds: &pb.Rect{MaxX: 40000, MaxY: 40000}}}},
		&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 1, GameTimeMs: 1000,
			Self:     &pb.SelfState{Entity: ent("hale", "Hale", 10000)},
			Group:    []*pb.GroupMember{{Entity: ent("cora", "Cora", 10500)}},
			Entities: []*pb.EntityState{ent("brann", "Brann", 10600)}}}},
		&pb.ServerMessage{Body: &pb.ServerMessage_Party{Party: &pb.PartyState{Id: "p", Leader: "Hale", Members: []*pb.PartyMemberInfo{
			{Name: "Hale", Online: true, Leader: true},
			{Name: "Cora", Online: true, Zone: "Dale", Level: 5, Hp: 40, MaxHp: 50, Mana: 30, MaxMana: 40},
			{Name: "Brann", Online: true, Zone: "Dale", Level: 6, Hp: 80, MaxHp: 90},
			{Name: "Dova", Online: true, Zone: "Hollow Warren", Level: 7, Hp: 20, MaxHp: 70, Mana: 12, MaxMana: 60},
			{Name: "Eli"}}}}})
	m.Self.Name = "Hale"
	return m
}

// The Party panel gives each other member a key (F2 to F6), their level, health
// and mana for a class with mana, and the zone only when it is another zone; a
// line fits the panel's width at any size.
func TestThePartyPanelLinesShowKeyLevelHealthManaAndZone(t *testing.T) {
	m := partyWorld()
	for _, size := range [][2]int{{80, 24}, {140, 45}} {
		out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, size[0], size[1], screen.Plain())
		for _, l := range strings.Split(out, "\n") {
			if w := []rune(strings.TrimRight(l, " ")); len(w) > size[0] {
				t.Errorf("%dx%d: a line is wider than the screen: %q", size[0], size[1], l)
			}
		}
		if size[0] < 140 {
			continue // the small screen has no room for the whole panel
		}
		for _, s := range []string{"F2 ", "Cora  Lv 5", "HP 40/50", "MP 30/40", "F3 ", "Brann  Lv 6", "HP 80/90",
			"F4 ", "Dova  Lv 7  Hollow Warren", "MP 12/60", "F5 ", "Eli  away"} {
			if !strings.Contains(out, s) {
				t.Errorf("%dx%d: the panel lacks %q:\n%s", size[0], size[1], s, out)
			}
		}
		for _, s := range []string{"Cora  Lv 5  Dale", "Brann  Lv 6  Dale", "HP 80/90 ######## MP"} {
			if strings.Contains(out, s) {
				t.Errorf("%dx%d: unexpected %q", size[0], size[1], s)
			}
		}
	}
}

// The Party panel's lines are hit areas: one per line of a member, the member's
// entity for one in view and none for one who is not.
func TestThePartyPanelNotesWhereEachMemberIsDrawn(t *testing.T) {
	m := partyWorld()
	var h screen.Hits
	screen.Render(m, screen.View{Conn: screen.ConnOnline, Hits: &h}, 140, 45, screen.Plain())
	got := map[string]string{}
	rows := map[int]string{}
	for _, p := range h.Party {
		got[p.Name] = p.ID
		if p.Rect.W <= 0 || p.Rect.H != 1 {
			t.Errorf("%s: rect %+v", p.Name, p.Rect)
		}
		rows[p.Rect.Y] = p.Name
	}
	for name, id := range map[string]string{"Cora": "cora", "Brann": "brann", "Dova": "", "Eli": ""} {
		if gid, ok := got[name]; !ok || gid != id {
			t.Errorf("%s: id %q (drawn %v), want %q", name, gid, ok, id)
		}
	}
	if _, ok := got["Hale"]; ok {
		t.Error("you have no line in the Party panel")
	}
	lines := strings.Split(screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain()), "\n")
	for y, name := range rows {
		if !strings.Contains(lines[y], name) && !strings.Contains(lines[y], "HP ") {
			t.Errorf("row %d is not %s's: %q", y, name, lines[y])
		}
	}
}
