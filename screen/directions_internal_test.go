package screen

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func withHints(hs ...*pb.Hint) *model.Model {
	m := worldModelInternal()
	m.Self.Hints = hs
	return m
}

var (
	nearSpot = &pb.Hint{Kind: "destination", Id: "ottilie", Label: "Ottilie Penn", Pos: &pb.Pos{X: 17000, Y: 17000}}
	farSpot  = &pb.Hint{Kind: "destination", Id: "ottilie", Label: "Ottilie Penn", Pos: &pb.Pos{X: 120000, Y: 0}}
	farZone  = []*pb.Hint{
		{Kind: "destination", Id: "gate", Label: "Barleycross", Zone: "Thornwick Downs", Pos: &pb.Pos{X: -200000, Y: 0}},
		{Kind: "destination_waypoint", Id: "gate", Label: "East Gate — to the Thornwick Downs", Note: "Barleycross", Zone: "Thornwick Downs", Pos: &pb.Pos{X: -200000, Y: 0}},
	}
)

// The sidebar's line names the destination with its compass direction and
// distance, or the door for another zone.
func TestTheDestinationLineNamesDirectionAndDistanceOrTheDoor(t *testing.T) {
	for _, tc := range []struct {
		name  string
		hints []*pb.Hint
		want  string
	}{
		{"none", nil, ""},
		{"near", []*pb.Hint{nearSpot}, "Destination: Ottilie Penn, NE 24 m"},
		{"far", []*pb.Hint{farSpot}, "Destination: Ottilie Penn, E 120 m"},
		{"another zone", farZone, "Destination: Barleycross, via East Gate"},
		{"only an objective", []*pb.Hint{{Kind: "objective", Label: "Talk to Gorran"}}, ""},
	} {
		if got := DestinationWords(withHints(tc.hints...)); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
	m := withHints(nearSpot)
	if rows := strings.Join(youLines(m, View{}, 60, Plain()), "\n"); !strings.Contains(rows, "Destination: Ottilie Penn, NE 24 m") {
		t.Errorf("the You panel lacks the line:\n%s", rows)
	}
}

// The mark is a ◉ at the destination, an arrow with its own mark at the edge when
// it lies beyond the map, and the door's for another zone; the objective's arrow
// gives way when it points the same way.
func TestTheDestinationIsMarkedOnTheMapAndAtItsEdge(t *testing.T) {
	objective := &pb.Hint{Kind: "objective", Id: "o", Label: "Talk to Gorran Stoutarm", Pos: &pb.Pos{X: 130000, Y: 0}}
	for _, tc := range []struct {
		name  string
		hints []*pb.Hint
		want  []string
		not   []string
	}{
		{"none", nil, nil, []string{"◉"}},
		{"near, on the map", []*pb.Hint{nearSpot}, []string{"◉"}, []string{"►", "◄"}},
		{"far, off the map", []*pb.Hint{farSpot}, []string{"Ottilie Penn 120m ◉►"}, nil},
		{"another zone, the door", farZone, []string{"◄◉ East Gate 200m"}, nil},
		{"the objective beyond the same edge gives way", []*pb.Hint{farSpot, objective}, []string{"Ottilie Penn 120m ◉►"}, []string{"Gorran 130m"}},
		{"the objective alone", []*pb.Hint{objective}, []string{"Gorran 130m ►"}, []string{"◉"}},
	} {
		out := strings.Join(mapRows(withHints(tc.hints...), View{Hits: &Hits{}}, 60, 20, Plain()), "\n")
		for _, w := range tc.want {
			if !strings.Contains(out, w) {
				t.Errorf("%s: no %q on the map:\n%s", tc.name, w, out)
			}
		}
		for _, w := range tc.not {
			if strings.Contains(out, w) {
				t.Errorf("%s: %q on the map:\n%s", tc.name, w, out)
			}
		}
	}
}

// The list shows the server's groups and order, with the cursor on its entry.
func TestTheDirectionsScreenListsTheGroupsInTheServersOrder(t *testing.T) {
	m := withHints()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: 2, Body: &pb.Event_Directions{Directions: &pb.DirectionsList{Actor: "me", Npc: "guard", Entries: []*pb.DirectionEntry{
		{Id: "trainers:t", Group: "trainers", Name: "Gorran Stoutarm", Title: "Warrior Trainer"},
		{Id: "merchants:v", Group: "merchants", Name: "Hilda Brasswick", Title: "Vendor"},
		{Id: "place:g", Group: "places", Name: "East Gate"},
		{Id: "place:b", Group: "places", Name: "Barleycross", Zone: "Thornwick Downs"},
	}}}}}})
	out := strings.Join(dirsLines(m, &Dirs{NPC: "guard", Sel: 3}, 70, 20, Plain()), "\n")
	last := -1
	for _, w := range []string{"Trainers", "Gorran Stoutarm, Warrior Trainer", "Merchants", "Hilda Brasswick, Vendor", "Places", "East Gate", "> Barleycross (Thornwick Downs)"} {
		i := strings.Index(out, w)
		if i < last {
			t.Errorf("%q is out of order or missing:\n%s", w, out)
		}
		last = max(i, last)
	}
	if got := dirsLines(m, &Dirs{NPC: "nobody"}, 70, 8, Plain()); !strings.Contains(strings.Join(got, "\n"), "…") {
		t.Errorf("no list yet should say so:\n%s", strings.Join(got, "\n"))
	}
}
