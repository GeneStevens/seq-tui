package screen_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// TUI-034: the playtest fixes on the screen. world is a small zone with a gate
// beside the character, three named markers and a destination.
func world(t *testing.T, usable bool) *model.Model {
	t.Helper()
	m := model.New()
	pos := func(x, y int64) *pb.Pos { return &pb.Pos{X: 10000 + x, Y: 10000 + y} }
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "Z", Seq: 1, GameTimeMs: 1000,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Aria", Level: 3, Pos: pos(0, 0), Hp: 10, MaxHp: 10}}}}})
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: "Z", Name: "Z", ViewRadiusMm: 20000,
		Bounds: &pb.Rect{MinX: -20000, MinY: -20000, MaxX: 40000, MaxY: 40000},
		Doors:  []*pb.Door{{Id: "gate-north", Name: "Gate North", Kind: pb.DoorKind_DOOR_KIND_INSTANCE_ENTRANCE, Pos: pos(1000, 0)}},
		Markers: []*pb.Marker{
			{Id: "overlook", Name: "The Overlook", Kind: "scout", Pos: pos(-6000, 0)},
			{Id: "rise", Name: "Brigands' Rise", Kind: "event_spawn", Pos: pos(6000, 0)},
			{Id: "goal", Name: "The Farms", Kind: "event_goal", Pos: pos(9000, 0)},
		}}}})
	why := "Gate North is shut: no expedition is open beyond it. Start one first."
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_DoorStates{DoorStates: &pb.DoorStates{Doors: []*pb.DoorState{{Id: "gate-north", Usable: usable, Why: map[bool]string{false: why}[usable]}}}}})
	return m
}

func lastRow(out string) string {
	rows := strings.Split(out, "\n")
	return rows[len(rows)-1]
}

func hintRow(out string) string {
	rows := strings.Split(out, "\n")
	return strings.TrimSpace(rows[len(rows)-2])
}

func TestAShutDoorIsNeverOfferedAsAnAction(t *testing.T) {
	v := screen.View{Conn: screen.ConnOnline}
	if h := hintRow(screen.Render(world(t, true), v, 140, 45, screen.Plain())); !strings.Contains(h, "E: enter Gate North") {
		t.Errorf("a usable door is not offered: %q", h)
	}
	m := world(t, false)
	h := hintRow(screen.Render(m, v, 140, 45, screen.Plain()))
	if strings.Contains(h, "E:") || !strings.Contains(h, "Gate North is shut: no expedition is open beyond it") {
		t.Errorf("a shut door is offered or its reason is missing: %q", h)
	}
	// The reason is in the warning style (red), not the key style.
	if h := hintRow(screen.Render(m, v, 140, 45, screen.ANSI())); !strings.Contains(h, "\x1b[31mGate North is shut") {
		t.Errorf("the reason is not in red: %q", h)
	}
	// Nearby marks it, dim.
	out := screen.Render(m, v, 140, 45, screen.Plain())
	if !strings.Contains(out, "Gate North") || !strings.Contains(out, "(shut)") {
		t.Errorf("Nearby does not mark the shut door:\n%s", out)
	}
	if out := screen.Render(world(t, true), v, 140, 45, screen.Plain()); strings.Contains(out, "(shut)") {
		t.Errorf("a usable door is marked shut:\n%s", out)
	}
	if ansi := screen.Render(m, v, 140, 45, screen.ANSI()); !strings.Contains(ansi, "\x1b[2m") {
		t.Errorf("no dim line in the screen")
	}
}

// A door the server has said nothing of is taken as usable: it decides on E.
func TestADoorWithNoStateIsOffered(t *testing.T) {
	m := world(t, true)
	m.Doors = nil
	if h := hintRow(screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain())); !strings.Contains(h, "E: enter Gate North") {
		t.Errorf("%q", h)
	}
}

func TestARefusalIsRedOnTheStatusLine(t *testing.T) {
	m := world(t, true)
	v := screen.View{Conn: screen.ConnOnline, HasTick: true, Notice: "Gate North is shut: no expedition is open beyond it. Start one first.", NoticeBad: true}
	row := lastRow(screen.Render(m, v, 140, 45, screen.ANSI()))
	want := "\x1b[1;97;41m"
	i := strings.Index(row, want)
	if i < 0 || !strings.Contains(row[i:], "! Gate North is shut: no expedition is open beyond it. Start one first.") {
		t.Fatalf("the status line has no red-background white bold refusal: %q", row)
	}
	// A plain notice is not an alert.
	v.NoticeBad = false
	if row := lastRow(screen.Render(m, v, 140, 45, screen.ANSI())); strings.Contains(row, want) {
		t.Errorf("a notice that is not a refusal is red: %q", row)
	}
	// Without colour the refusal is still told apart: bold reverse, and the "!".
	v.NoticeBad = true
	if row := lastRow(screen.Render(m, v, 140, 45, screen.Mono())); !strings.Contains(row, "\x1b[1;7m") || !strings.Contains(row, "! Gate North") {
		t.Errorf("mono: %q", row)
	}
	// At 80 cells it is still the part that shows.
	if row := lastRow(screen.Render(m, v, 80, 24, screen.Plain())); !strings.Contains(row, "! Gate North is shut") {
		t.Errorf("80 cells: %q", row)
	}
}

func pickAt(t *testing.T, m *model.Model, pos *pb.Pos) (screen.Pick, string) {
	t.Helper()
	h := &screen.Hits{}
	v := screen.View{Conn: screen.ConnOnline, Hits: h}
	screen.Render(m, v, 140, 45, screen.Plain())
	pt, ok := h.CellOf(pos)
	if !ok {
		t.Fatalf("%v is not on the map", pos)
	}
	p, ok := h.PickAt(m, pt.X, pt.Y)
	if !ok {
		t.Fatalf("nothing at %v", pt)
	}
	v.Hover = &pt
	return p, screen.Render(m, v, 140, 45, screen.Plain())
}

func TestMarkersHaveHoverClickAndNearby(t *testing.T) {
	m := world(t, true)
	for _, c := range []struct {
		id, label string
		pos       *pb.Pos
	}{
		{"overlook", "The Overlook: a scout point", &pb.Pos{X: 4000, Y: 10000}},
		{"rise", "Brigands' Rise: raiders gather here (world event)", &pb.Pos{X: 16000, Y: 10000}},
		{"goal", "The Farms: raiders head here (world event)", &pb.Pos{X: 19000, Y: 10000}},
	} {
		p, out := pickAt(t, m, c.pos)
		if p.Kind != screen.PickMarker || p.ID != c.id || p.Pos.GetX() != c.pos.GetX() {
			t.Errorf("%s: picked %+v", c.id, p)
		}
		if !strings.Contains(out, c.label) {
			t.Errorf("hover over %s does not say %q:\n%s", c.id, c.label, out)
		}
		if !strings.Contains(out, " "+strings.SplitN(c.label, ":", 2)[0]) {
			t.Errorf("Nearby lacks %s", c.id)
		}
	}
	// Nearby lists them, the goal with a glyph that is not the guard's.
	out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain())
	for _, want := range []string{"; The Overlook", "v Brigands' Rise", "◆ The Farms"} {
		if !strings.Contains(out, want) {
			t.Errorf("Nearby lacks %q:\n%s", want, out)
		}
	}
}

func TestHoveringYourselfSaysWho(t *testing.T) {
	m := world(t, true)
	p, out := pickAt(t, m, m.Self.Pos)
	if p.Kind != screen.PickSelf {
		t.Fatalf("picked %+v", p)
	}
	if !strings.Contains(out, "You: Aria, level 3, at 10,10 m") || strings.Contains(out, "walking to") {
		t.Errorf("self hover:\n%s", out)
	}
	m.Self.HasDestination, m.Self.Destination = true, &pb.Pos{X: 30000, Y: 10000}
	if _, out := pickAt(t, m, m.Self.Pos); !strings.Contains(out, "You: Aria, level 3, at 10,10 m, walking to 30,10 m") {
		t.Errorf("self hover while walking:\n%s", out)
	}
}

func TestHelpTitlesAndKeysAreStyledAndWalkingIsCurrent(t *testing.T) {
	m := world(t, true)
	v := screen.View{Conn: screen.ConnOnline, Help: true}
	ansi := screen.Render(m, v, 140, 45, screen.ANSI())
	if !strings.Contains(ansi, "\x1b[1;36m Moving and doors") {
		t.Errorf("the section title is not in the panel-title style:\n%s", ansi)
	}
	if !strings.Contains(ansi, "\x1b[36mArrows, W A S D") {
		t.Errorf("a key is not in the hint bar's key style")
	}
	plain := screen.Render(m, v, 140, 45, screen.Plain())
	for _, want := range []string{"tap for one 3 m step", "hold to walk", "single press"} {
		if !strings.Contains(plain, want) {
			t.Errorf("the walking text lacks %q:\n%s", want, plain)
		}
	}
	// The key table is two columns on every page that has keys.
	for page := 0; page < screen.HelpPages(); page++ {
		v.HelpPage = page
		if out := screen.Render(m, v, 140, 45, screen.ANSI()); !strings.Contains(out, "\x1b[1;36m") {
			t.Errorf("page %d has no styled title", page)
		}
	}
	v.HelpPage = 6
	if out := screen.Render(m, v, 140, 45, screen.Plain()); !strings.Contains(out, "Ctrl+R") || !strings.Contains(out, "3 m step") {
		t.Errorf("the key page lacks Ctrl+R or the step:\n%s", out)
	}
}

func TestARefusalInEventsIsRedAndBold(t *testing.T) {
	m := world(t, true)
	m.IntentSent("i1", 1, "use_door")
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_IntentResult{IntentResult: &pb.IntentResult{IntentId: "i1", Reason: "no_instance", Text: "Gate North is shut: no expedition is open beyond it. Start one first."}}})
	out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.ANSI())
	if !strings.Contains(out, "\x1b[31m\x1b[1m ! 16:04:03.2 Gate North is shut") {
		for _, l := range strings.Split(out, "\n") {
			if strings.Contains(l, "Gate North is shut") {
				t.Logf("%q", l)
			}
		}
		t.Errorf("the refusal in Events is not red and bold")
	}
}
