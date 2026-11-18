package app

import (
	"reflect"
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// guardWorld is a character beside a guard that has spoken with the options given.
func guardWorld(options ...*pb.DialogueOption) (*App, *model.Model) {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "Z", Seq: 1, GameTimeMs: 1000,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Pos: &pb.Pos{X: 10000}, Hp: 50, MaxHp: 50}},
		Npcs: []*pb.NpcInfo{{Id: "guard", Name: "Town Guard", Title: "Guard", Role: "guard", ReachMm: 5000, Pos: &pb.Pos{X: 12000}}},
	}}})
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: 2, GameTimeMs: 2000,
		Body: &pb.Event_Dialogue{Dialogue: &pb.Dialogue{Actor: "me", Npc: "guard", Lines: []string{"Halt."}, Options: options}}}}})
	return New(client.NewSession(client.Config{Model: m})), m
}

var directionsOption = &pb.DialogueOption{Action: "directions", Label: "Directions"}

func theList(m *model.Model) {
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: 3, GameTimeMs: 3000, Body: &pb.Event_Directions{Directions: &pb.DirectionsList{Actor: "me", Npc: "guard", Entries: []*pb.DirectionEntry{
		{Id: "trainers:gorran", Group: "trainers", Name: "Gorran Stoutarm", Title: "Warrior Trainer"},
		{Id: "place:gate", Group: "places", Name: "East Gate"},
	}}}}}})
}

// The talk screen offers Directions only for a guard that gives them; Enter on it
// opens the list and asks; the arrows and Enter choose an entry of the list, and
// that sends the entry's id; Esc goes back to the talk.
func TestTheTalkScreenOffersDirectionsAndAChoiceSendsTheEntry(t *testing.T) {
	a, _ := guardWorld(&pb.DialogueOption{Action: "buy", Label: "Buy"})
	a.talk = &screen.Talk{NPC: "guard"}
	if s := screenNow(a); strings.Contains(s, "Directions") {
		t.Errorf("a guard that gives none offers them:\n%s", s)
	}
	a, m := guardWorld(directionsOption)
	a.talk = &screen.Talk{NPC: "guard"}
	if s := screenNow(a); !strings.Contains(s, "Directions") {
		t.Fatalf("no Directions on the talk screen:\n%s", s)
	}
	if act := a.option(m, "guard", directionsOption); act.Kind != OpenDirections || act.NPC != "guard" {
		t.Fatalf("the option: %+v", act)
	}
	a.do(Action{Kind: OpenDirections, NPC: "guard", Note: "Town Guard"})
	if a.dirs == nil || a.talk != nil {
		t.Fatalf("the list did not open over the talk: dirs %v, talk %v", a.dirs, a.talk)
	}
	theList(m)
	s := screenNow(a)
	if i, j := strings.Index(s, "Trainers"), strings.Index(s, "Places"); i < 0 || j < i {
		t.Errorf("the groups are not in the server's order:\n%s", s)
	}
	for _, tc := range []struct {
		keys []string
		want Action
	}{
		{[]string{"enter"}, Action{Kind: ChooseDirection, NPC: "guard", ID: "trainers:gorran", Note: "Gorran Stoutarm"}},
		{[]string{"down", "enter"}, Action{Kind: ChooseDirection, NPC: "guard", ID: "place:gate", Note: "East Gate"}},
		{[]string{"down", "down", "up", "enter"}, Action{Kind: ChooseDirection, NPC: "guard", ID: "trainers:gorran", Note: "Gorran Stoutarm"}}, // the list ends at its last entry
	} {
		a.dirs = &screen.Dirs{NPC: "guard"}
		var got Action
		for _, k := range tc.keys {
			a.sess.View(func(m *model.Model) { got = a.dirsKey(k, m) })
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%v: %+v, want %+v", tc.keys, got, tc.want)
		}
	}
	a.dirs = &screen.Dirs{NPC: "guard"}
	a.sess.View(func(m *model.Model) { a.dirsKey("esc", m) })
	if a.dirs != nil || a.talk == nil || a.talk.NPC != "guard" {
		t.Errorf("Esc: dirs %v, talk %v", a.dirs, a.talk)
	}
}

// Clearing is a key anywhere, and on the large map X marks the cursor's spot.
func TestClearAndSetDestinationKeys(t *testing.T) {
	a, m := guardWorld()
	if act := Plan("X", m); act.Kind != ClearDestination {
		t.Errorf("ctrl+d: %+v", act)
	}
	if act := a.overviewKey("x", m); act.Kind != None || act.Note == "" {
		t.Errorf("x with no cursor: %+v", act)
	}
	a.mouse.cursor = &pb.Pos{X: 4000, Y: 5000}
	if act := a.overviewKey("x", m); !reflect.DeepEqual(act, Action{Kind: SetDestination, X: 4000, Y: 5000}) {
		t.Errorf("x on the cursor: %+v", act)
	}
}
