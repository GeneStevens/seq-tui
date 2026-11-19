package app

import (
	"reflect"
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// D206: item links in the terminal: making them in the chat line, and showing a
// linked item's details.

var (
	hearth = &pb.ItemInfo{Id: "hearth-draught", Name: "Hearth Draught", Tier: "fine"}
	cap_   = &pb.ItemInfo{Id: "worn-cap", Name: "Worn Cap", Tier: "common", Slot: "head"}
	blade  = &pb.ItemInfo{Id: "iron-blade", Name: "Iron Blade", Tier: "rare", Slot: "main_hand"}
)

// linkWorld is a character carrying a draught and a blade and wearing a cap.
func linkWorld() (*App, *model.Model) {
	a, m, _ := chatApp()
	for _, it := range []*pb.ItemInfo{hearth, cap_, blade} {
		m.Items[it.Id] = it
	}
	m.Self.Inventory = []string{"hearth-draught", "iron-blade"}
	m.Self.Equipped = []string{"worn-cap"}
	return a, m
}

func hearItems(a *App, m *model.Model, parts ...*pb.ChatPart) {
	a.w, a.h, a.paint.hitsStale = 140, 45, true // the next key reads the links as drawn
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Chat{Chat: &pb.ChatMessage{Channel: "say", From: "Mira", Text: "look", Parts: parts}}})
}

func item(it *pb.ItemInfo) *pb.ChatPart {
	return &pb.ChatPart{Body: &pb.ChatPart_Item{Item: &pb.ItemRef{Id: it.Id, Name: it.Name, Tier: it.Tier}}}
}

func words(s string) *pb.ChatPart { return &pb.ChatPart{Body: &pb.ChatPart_Text{Text: s}} }

// A link's span is where its "[Name]" stands in the text sent, in code points; a
// name the player edited away is no link.
func TestLinkSpansAreWhereTheBracketedNamesStand(t *testing.T) {
	both := []screen.LineLink{{Item: "a", Name: "Hearth Draught"}, {Item: "b", Name: "Iron Blade"}}
	for _, tc := range []struct {
		name  string
		text  string
		links []screen.LineLink
		want  [][3]any
	}{
		{"none", "hello", nil, nil},
		{"one", "try the [Hearth Draught] now", both[:1], [][3]any{{"a", 8, 24}}},
		{"two in order", "[Hearth Draught] and [Iron Blade]", both, [][3]any{{"a", 0, 16}, {"b", 21, 33}}},
		{"edited away", "try the Hearth Draught", both[:1], nil},
		{"one of two edited away", "no [Iron Blade]", both, [][3]any{{"b", 3, 15}}},
		{"the same item twice", "[Iron Blade] [Iron Blade]", []screen.LineLink{both[1], both[1]}, [][3]any{{"b", 0, 12}, {"b", 13, 25}}},
		{"after wide letters", "Écoute [Hearth Draught]", both[:1], [][3]any{{"a", 7, 23}}},
	} {
		var got [][3]any
		for _, l := range linkSpans(tc.text, tc.links) {
			got = append(got, [3]any{l.Item, int(l.Start), int(l.End)})
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
}

// In the bags or the worn list, L links the item under the cursor into the chat
// line, opening it when it is shut; Ctrl+Y adds another with the line open.
func TestLinkingFromTheBagsAndTheWornList(t *testing.T) {
	for _, tc := range []struct {
		name       string
		tab, cur   int
		wantLink   string
		wantName   string
		wantAction Kind
	}{
		{"bags", screen.TabBags, 1, "iron-blade", "Iron Blade", LinkItem},
		{"worn", screen.TabGear, 0, "worn-cap", "Worn Cap", LinkItem},
		{"nothing worn there", screen.TabGear, 3, "", "", None},
		{"stats", screen.TabStats, 0, "", "", None},
	} {
		a, _ := linkWorld()
		a.char = true
		a.mouse.charTab.Tab = tc.tab
		a.mouse.charTab.Cur[tc.tab] = tc.cur
		a.key("l")
		switch {
		case tc.wantAction == None:
			if a.line != nil {
				t.Errorf("%s: the line opened with %+v", tc.name, a.line)
			}
			continue
		case a.line == nil || len(a.line.Links) != 1:
			t.Fatalf("%s: the key did not link: %+v", tc.name, a.line)
		}
		want := []screen.LineLink{{Item: tc.wantLink, Name: tc.wantName}}
		if !reflect.DeepEqual(a.line.Links, want) || !strings.Contains(a.line.Text, "["+tc.wantName+"]") || a.line.Channel != screen.ChanSay {
			t.Errorf("%s: line %+v, want %+v written in brackets", tc.name, a.line, want)
		}
	}
	a, _ := linkWorld()
	a.char = true
	a.mouse.charTab.Tab = screen.TabBags
	a.key("L")
	a.lineKey(keyPress("down"))
	a.lineKey(keyPress("ctrl+y"))
	if got := a.line.Links; len(got) != 2 || got[0].Item != "hearth-draught" || got[1].Item != "iron-blade" {
		t.Errorf("a second link with the line open: %+v", a.line)
	}
	for i := 0; i < 5; i++ {
		a.lineKey(keyPress("ctrl+y"))
	}
	if len(a.line.Links) != maxLinks || !strings.Contains(screenNow(a), "at most 5") {
		t.Errorf("the line holds %d links, and the screen does not say five is the most", len(a.line.Links))
	}
}

// What the line sends: the item ids and where their names stand in the text the
// channel carries, whichever channel and whatever the player typed round them.
func TestTheMessageSendsIdsAndTheirSpans(t *testing.T) {
	links := []screen.LineLink{{Item: "hearth-draught", Name: "Hearth Draught"}}
	for _, tc := range []struct {
		name string
		l    screen.Line
		want []int
	}{
		{"say", screen.Line{Text: "  try [Hearth Draught] now  "}, []int{4, 20}},
		{"party", screen.Line{Text: "try [Hearth Draught]", Channel: "party"}, []int{4, 20}},
		{"tell", screen.Line{Text: "try [Hearth Draught]", Channel: "tell", Target: "Mira"}, []int{4, 20}},
		{"ooc", screen.Line{Text: "try [Hearth Draught]", Channel: "ooc"}, []int{4, 20}},
	} {
		tc.l.Links = links
		text := lineWire(&tc.l)
		_, arg := splitLine(text)
		verb := lineVerb(strings.TrimPrefix(strings.Fields(text + " x")[0], "/"))
		if verb == "tell" {
			_, arg = firstWord(arg)
		}
		got := linkSpans(arg, tc.l.Links)
		if len(got) != 1 || got[0].Item != "hearth-draught" || int(got[0].Start) != tc.want[0] || int(got[0].End) != tc.want[1] {
			t.Errorf("%s: sends %v for %q, want id hearth-draught at %v", tc.name, got, arg, tc.want)
		}
	}
	a, _ := linkWorld()
	a.line = &screen.Line{Text: "see [Hearth Draught]", Links: links}
	if _, cmd := a.lineKey(keyPress("enter")); cmd == nil || a.line != nil {
		t.Errorf("Enter did not send the line")
	}
}

// Shift+I shows the details of the one item in view; with several it lists them
// and a number picks one. The details are the bags' card with D78's comparison.
func TestTheDetailsOfOneLinkAndOfSeveral(t *testing.T) {
	a, m := linkWorld()
	m.Self.Facts = map[string]*pb.ItemFact{}
	a.view.Panel = screen.PanelChat
	a.key("I")
	if a.items != nil || !strings.Contains(a.view.Notice, "no item is linked") {
		t.Errorf("with nothing linked: %v, %q", a.items, a.view.Notice)
	}
	hearItems(a, m, words("try the "), item(hearth), words(" now"))
	a.key("I")
	if a.items == nil || a.items.ID != "hearth-draught" || len(a.items.Pick) != 0 {
		t.Fatalf("one link did not open its details: %+v", a.items)
	}
	m.NoteItemDetails(&pb.ItemDetails{Item: blade, Fact: &pb.ItemFact{Item: "iron-blade", Verdict: "upgrade", Slot: "main_hand"}})
	a.items = &screen.ItemUI{ID: "iron-blade"}
	s := screenNow(a)
	for _, want := range []string{"[Iron Blade]", "Rare", "upgrade"} {
		if !strings.Contains(s, want) {
			t.Errorf("the details lack %q:\n%s", want, s)
		}
	}
	a.key("esc")
	if a.items != nil {
		t.Fatalf("Esc did not close the details")
	}
	hearItems(a, m, item(blade), words(" and "), item(hearth))
	a.key("I")
	if a.items == nil || len(a.items.Pick) != 2 {
		t.Fatalf("several links did not open a choice: %+v", a.items)
	}
	s = screenNow(a)
	i, j := strings.Index(s, "1  [Hearth Draught]"), strings.Index(s, "2  [Iron Blade]") // each once, as drawn from the top
	if i < 0 || j < i {
		t.Errorf("the choice does not list the links by number in the order they stand:\n%s", s)
	}
	a.key("2")
	if a.items == nil || a.items.ID != "iron-blade" || len(a.items.Pick) != 0 {
		t.Errorf("2 did not choose the second: %+v", a.items)
	}
}

// What a person said on the talk screen: I shows the item they linked.
func TestTheDetailsOfAnItemAPersonNamed(t *testing.T) {
	a, m := guardWorld()
	m.Items[hearth.Id] = hearth
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: 3, GameTimeMs: 3000,
		Body: &pb.Event_Dialogue{Dialogue: &pb.Dialogue{Actor: "me", Npc: "guard", Lines: []string{"Take a [Hearth Draught]."},
			LinesParts: []*pb.LineParts{{Parts: []*pb.ChatPart{words("Take a "), item(hearth), words(".")}}}}}}}})
	a.talk = &screen.Talk{NPC: "guard"}
	if s := screenNow(a); !strings.Contains(s, "“Take a [Hearth Draught].”") || !strings.Contains(s, "{I}") && !strings.Contains(s, "I shows an item") {
		t.Errorf("the talk does not show the link and its key:\n%s", s)
	}
	a.key("I")
	if a.items == nil || a.items.ID != "hearth-draught" {
		t.Errorf("I on the talk screen: %+v", a.items)
	}
}
