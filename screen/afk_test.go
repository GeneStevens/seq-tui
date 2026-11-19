package screen_test

import (
	"github.com/GeneStevens/seq-tui/model"
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// AFK (D208): the server's flag is shown beside a name where the terminal names
// characters, and on the character itself in the sidebar.

func afkChanged(seq uint64, actor string, on bool, text string) *pb.ServerMessage {
	return event(seq, func(e *pb.Event) {
		e.Text = text
		e.Body = &pb.Event_AfkChanged{AfkChanged: &pb.AfkChanged{Actor: actor, On: on}}
	})
}

func TestAFKShowsBesideANameAndOnYourself(t *testing.T) {
	m := model.New() // hale is you; bram and cora are in view
	char := func(id, name string, x int64) *pb.EntityState {
		return &pb.EntityState{Id: id, Name: name, Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, Pos: &pb.Pos{X: x, Y: 10000}, Hp: 100, MaxHp: 100}
	}
	apply(m,
		&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: "z", Name: "Dale", ViewRadiusMm: 20000, Bounds: &pb.Rect{MaxX: 40000, MaxY: 40000}}}},
		&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 1, GameTimeMs: 1000,
			Self:     &pb.SelfState{Entity: char("hale", "Hale", 10000)},
			Entities: []*pb.EntityState{char("bram", "Bram", 11000), char("cora", "Cora", 12000)}}}})
	apply(m, afkChanged(2, "bram", true, ""))
	if !m.Entities["bram"].GetAfk() || m.Entities["cora"].GetAfk() {
		t.Fatalf("the event did not reach bram alone")
	}
	out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain())
	if !strings.Contains(out, "Bram") || strings.Count(out, "AFK") != 1 || strings.Contains(out, "You are AFK.") {
		t.Errorf("nearby should mark only bram:\n%s", out)
	}
	who := func(afk bool) string {
		ui := &screen.SocialUI{Tab: screen.TabWho, Who: []*pb.WhoEntry{{Name: "Bram", ClassId: "cleric", Level: 5, Zone: "Warren", Afk: afk}}}
		return screen.Render(m, screen.View{Conn: screen.ConnOnline, Social: ui}, 140, 45, screen.Plain())
	}
	if !strings.Contains(who(true), "Warren AFK") || strings.Contains(who(false), "Warren AFK") {
		t.Errorf("who rows: AFK shown wrongly")
	}
	line := screen.LookLines(m, "bram", 60)[0]
	if !strings.Contains(line, "AFK") || strings.Contains(screen.LookLines(m, "cora", 60)[0], "AFK") {
		t.Errorf("look names: %q", line)
	}
	apply(m, afkChanged(3, "hale", true, "You are now AFK."))
	out = screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain())
	if !strings.Contains(out, "You are AFK.") || !strings.Contains(out, "You are now AFK.") {
		t.Errorf("your own flag and the server's line are not shown:\n%s", out)
	}
	apply(m, afkChanged(4, "hale", false, "You are no longer AFK."), afkChanged(5, "bram", false, ""))
	out = screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain())
	if strings.Contains(out, "You are AFK.") || m.Entities["bram"].GetAfk() || !strings.Contains(out, "You are no longer AFK.") {
		t.Errorf("clearing the flag:\n%s", out)
	}
}
