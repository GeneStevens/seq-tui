package screen_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// TUI-038: D86 and the level cap on the screen, tested on the final output.

func d86Model(level int64, atCap bool) *model.Model {
	m := model.New()
	pos := &pb.Pos{X: 10000, Y: 10000}
	next := int64(500)
	if atCap {
		next = 0
	}
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "Z", Seq: 1, GameTimeMs: 1000,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Aria", Level: level, Pos: pos, Hp: 10, MaxHp: 10},
			Class: "warrior", Xp: 300, XpLevelStart: 100, XpNextLevel: next, AtCap: atCap},
		Npcs: []*pb.NpcInfo{{Id: "board", Name: "Notice Board", Title: "Contract Board", Role: "contract_board", ReachMm: 5000, Pos: pos,
			Offers: []*pb.ContractOffer{
				{Id: "far", Name: "Far Hunt", Kind: "hunt", MinLevel: 6, Need: 3},
				{Id: "near", Name: "Near Hunt", Kind: "hunt", MinLevel: 1, Need: 3},
				{Id: "also", Name: "Also Open", Kind: "gather", MinLevel: 3, Need: 2},
			}}}}}})
	return m
}

func xpEvent(m *model.Model, paidAs int64, text string) {
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: 5, GameTimeMs: 2000, Text: text,
		Body: &pb.Event_Reward{Reward: &pb.Reward{Recipient: "me", Kind: "xp", Amount: 22, Level: 3, PaidAsLevel: paidAs}}}}})
}

func TestACappedXPLineIsMutedNotRed(t *testing.T) {
	m := d86Model(3, false)
	xpEvent(m, 6, "You gain 22 experience (capped: paid as a level 6 creature)")
	xpEvent(m, 0, "You gain 40 experience")
	v := screen.View{Conn: screen.ConnOnline}
	plain := screen.Render(m, v, 140, 45, screen.Plain())
	if !strings.Contains(plain, "You gain 22 experience (capped: paid as a level 6 creature)") {
		t.Errorf("the server's line is not in Events:\n%s", plain)
	}
	ansi := screen.Render(m, v, 140, 45, screen.ANSI())
	if !strings.Contains(ansi, "experience \x1b[0m\x1b[2m(capped: paid as a level 6 creature)") && !strings.Contains(ansi, "\x1b[2m(capped: paid as a level 6 creature)") {
		t.Errorf("the capped part is not dim:\n%q", ansi)
	}
	for _, l := range strings.Split(ansi, "\n") {
		if strings.Contains(l, "(capped") && (strings.Contains(l, "\x1b[31m") || strings.Contains(l, "\x1b[1;97;41m")) {
			t.Errorf("a capped line reads as a refusal: %q", l)
		}
		if strings.Contains(l, "You gain 40 experience") && !strings.Contains(l, "\x1b[32m") {
			t.Errorf("an uncapped line lost its green: %q", l)
		}
	}
}

func TestAnOfferThatOpensLaterIsDimAndSortedLast(t *testing.T) {
	m := d86Model(3, false)
	v := screen.View{Conn: screen.ConnOnline, Board: &screen.Board{NPC: "board", Cur: [2]int{2, 0}}}
	out := screen.Render(m, v, 140, 45, screen.Plain())
	iNear, iAlso, iFar := strings.Index(out, "Near Hunt"), strings.Index(out, "Also Open"), strings.Index(out, "Far Hunt")
	if iNear < 0 || iAlso < iNear || iFar < iAlso {
		t.Errorf("the takeable offers are not listed first (%d %d %d):\n%s", iNear, iAlso, iFar, out)
	}
	if !strings.Contains(out, "opens at level 6") {
		t.Errorf("no 'opens at level 6':\n%s", out)
	}
	if !strings.Contains(out, "This contract opens at level 6.") {
		t.Errorf("the reason is not shown under the cursor:\n%s", out)
	}
	if strings.Contains(out, "Near Hunt") && strings.Contains(strings.Split(out, "Near Hunt")[1][:40], "opens at") {
		t.Errorf("a takeable offer says it opens later:\n%s", out)
	}
	v.Board.Cur[0] = 0 // the cursor row is drawn reversed, not dim
	ansi := screen.Render(m, v, 140, 45, screen.ANSI())
	dim := false
	for _, l := range strings.Split(ansi, "\n") {
		if strings.Contains(l, "\x1b[2m  Far Hunt") {
			dim = true
		}
		if strings.Contains(l, "\x1b[2m  Near Hunt") || strings.Contains(l, "\x1b[2m  Also Open") {
			t.Errorf("a takeable offer is dim: %q", l)
		}
	}
	if !dim {
		t.Errorf("the later offer is not dim:\n%q", ansi)
	}
}

func TestMaxLevelIsSaidInBothLayouts(t *testing.T) {
	m := d86Model(12, true)
	for _, sz := range [][2]int{{80, 24}, {140, 45}} {
		out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, sz[0], sz[1], screen.Plain())
		if !strings.Contains(out, "Lv 12 (max)") || !strings.Contains(out, "max") {
			t.Errorf("%v: no Lv 12 (max):\n%s", sz, out)
		}
		if strings.Contains(out, "to Lv 13") || strings.Contains(out, "top level") {
			t.Errorf("%v: a bar to the next level at the cap:\n%s", sz, out)
		}
	}
	if out := screen.Render(d86Model(5, false), screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain()); strings.Contains(out, "(max)") {
		t.Errorf("a character below the cap shows max or no bar:\n%s", out)
	}
	// A character with no class (no kit) says it too.
	m.Self.Class = ""
	if out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain()); !strings.Contains(out, "Lv 12 (max)") {
		t.Errorf("no max without a kit:\n%s", out)
	}
}

func TestTheHelpHasGroupingAndExperience(t *testing.T) {
	m := d86Model(3, false)
	var out string
	for p := 0; p < screen.HelpPages(); p++ {
		o := screen.Render(m, screen.View{Conn: screen.ConnOnline, Help: true, HelpPage: p}, 140, 45, screen.Plain())
		if strings.Contains(o, "Grouping and experience") && strings.Contains(o, "Level 12") {
			out = o
		}
	}
	for _, want := range []string{"other accounts shares the experience, with a bonus", "3 levels above you would", "open at most 3 levels above you", "no more experience, but crowns and items"} {
		if !strings.Contains(out, want) {
			t.Errorf("the grouping page lacks %q:\n%s", want, out)
		}
	}
}
