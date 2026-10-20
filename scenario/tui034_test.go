package scenario_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
	"github.com/GeneStevens/seq/testserver"
)

// TUI-034: a refusal at a door, against the real server. Standing at Gate North
// with no expedition open: the hint says the gate is shut and does not offer E,
// Nearby marks it, and pressing E still asks the server, whose plain line is in
// Events in red and on the status line, once.
func TestAShutGateIsHonestAndItsRefusalIsNotMissed(t *testing.T) {
	const why = "Gate North is shut: no expedition is open beyond it. Start one first."
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria")})
	w.Place("ann", "aria", gateX, gateY)
	p := w.Login("ann", "aria")
	w.Advance(2 * time.Second)
	p.Eventually("the server's word on the gate", func(m *model.Model) bool {
		u, _ := m.DoorUsable("gate-north")
		return len(m.Doors) > 0 && !u
	})

	s := p.Screen(140, 45)
	if h := hintRow(s); strings.Contains(h, "E:") || !strings.Contains(h, why) {
		t.Errorf("the hint offers a shut gate or omits why: %q", h)
	}
	if !strings.Contains(s, "(shut)") {
		t.Errorf("Nearby does not mark the gate:\n%s", s)
	}

	p.Press("e")
	notice(t, p, why, true)
	w.Advance(time.Second)
	s = p.Screen(140, 45)
	if !regexp.MustCompile(`! +[0-9.]+s ` + regexp.QuoteMeta(why)).MatchString(s) {
		t.Errorf("the refusal is not in Events:\n%s", s)
	}
	if !strings.Contains(lastRow(s), "! "+why) {
		t.Errorf("the refusal is not on the status line: %q", lastRow(s))
	}
	if strings.Contains(s, "use door denied") {
		t.Errorf("the internal prefix is still there:\n%s", s)
	}
	p.Model(func(m *model.Model) {
		n := 0
		for _, e := range m.Log {
			if e.Refused() && e.Text == why {
				n++
			}
		}
		if n != 1 {
			t.Errorf("the refusal is in the log %d times: %+v", n, m.Log)
		}
	})
	p.Golden("gate_shut_refused")
}

// A named marker of the zone (the Testgrounds' scout point) has a hover label with
// its name and kind, and a click walks there; hovering yourself says who you are.
func TestAMarkerHasAHoverAndAClickWalksThere(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria")})
	w.Place("ann", "aria", 84000, 50000)
	p := w.Login("ann", "aria")
	p.Resize(140, 45)
	var mk *pb.Marker
	var from *pb.Pos
	p.Model(func(m *model.Model) {
		from = m.Self.Pos
		for _, k := range m.Map.GetMarkers() {
			if k.GetKind() == "scout" && k.GetName() != "" {
				mk = k
			}
		}
	})
	if mk == nil {
		t.Fatal("the zone has no named scout marker")
	}
	// Zoom out so the marker is on the map.
	for range 3 {
		p.Press("-")
	}
	x, y := at(t, p, mk.GetPos())
	p.Hover(x, y, false)
	if s := p.Screen(140, 45); !strings.Contains(s, mk.GetName()+": a scout point") {
		t.Errorf("hovering the marker does not name it and its kind:\n%s", s)
	}
	sx, sy := at(t, p, from)
	p.Hover(sx, sy, false)
	if s := p.Screen(140, 45); !strings.Contains(s, "You: aria, level 1, at 84,50 m") {
		t.Errorf("hovering yourself:\n%s", s)
	}
	p.Click(x, y)
	notice(t, p, "move ok", false)
	w.Advance(3 * time.Second)
	p.Model(func(m *model.Model) {
		if !m.Self.HasDestination && m.Self.Pos.GetX() == from.GetX() && m.Self.Pos.GetY() == from.GetY() {
			t.Errorf("the click on the marker did not walk anywhere: %v", m.Self.Pos)
		}
	})
}
