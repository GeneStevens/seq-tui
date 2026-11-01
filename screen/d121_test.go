package screen_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// D121: the places that matter are named on the large map at any zoom (the Downs
// goldens, testdata/downs, show them); names the map has no room for are counted
// under the nearest name, "Goblin Hut +2"; and the tip names what is under the
// pointer or the cursor.

func overviewOf(m *model.Model, v screen.View, w, h int) string {
	v.Overview = true
	return screen.Render(m, v, w, h, screen.Plain())
}

func TestImportantSmallPlacesAreNamedAtZoomZero(t *testing.T) {
	north := &pb.QuestInfo{Id: "highfold-watch", Name: "The Shepherds' Watch", Step: 2, Steps: 4, Kind: "watch", Text: "Watch the fold.", TargetKind: "place", Target: "highfold", TargetName: "Highfold", TargetPos: &pb.Pos{X: 100000, Y: 400000}}
	for _, sz := range [][2]int{{80, 24}, {120, 40}, {200, 50}} {
		out := overviewOf(downsWorldOn(downsWide(), 100000, 406000, north), view(), sz[0], sz[1])
		for _, name := range []string{"Barleycross", "Eldbury Ring", "Eelmere", "Fordwatch", "Goblin Stockade"} {
			if !strings.Contains(out, name) {
				t.Errorf("%v: %q is not named at zoom 0:\n%s", sz, name, out)
			}
		}
	}
}

// Places left out are counted under a name near them, and the tip lists them.
func TestHiddenPlacesAreGroupedUnderTheNearestName(t *testing.T) {
	m := downsWorldOn(downsWide(), 100000, 406000, nil)
	h := &screen.Hits{}
	v := view()
	v.Hits = h
	out := overviewOf(m, v, 120, 40)
	plus := regexp.MustCompile(`Goblin Hut \+(\d)`).FindStringSubmatch(out)
	if plus == nil {
		t.Fatalf("no \"Goblin Hut +N\" in the Downs:\n%s", out)
	}
	// The tip of the cell on the name lists it and what is counted under it.
	var tip string
	for y, row := range strings.Split(out, "\n") {
		if x := strings.Index(row, "Goblin Hut +"); x >= 0 {
			tip = h.TipAt(len([]rune(row[:x]))+1, y)
		}
	}
	if !strings.HasPrefix(tip, "Goblin Hut") || !strings.Contains(tip, "Goblin Rocks") || strings.Count(tip, "·") != int(plus[1][0]-'0') {
		t.Errorf("the tip of \"Goblin Hut +%s\" is %q", plus[1], tip)
	}
	// With everything named (Ctrl+A) nothing is left out, so nothing is counted.
	v.Labels = true
	if all := overviewOf(m, v, 200, 50); regexp.MustCompile(` \+\d`).MatchString(strings.Split(all, "Option-click")[0]) && strings.Contains(all, "Goblin Hut +") {
		t.Errorf("a name is counted with everything named:\n%s", all)
	}
}

// The mouse over a spot of the large map shows a one-line tip naming the places there.
func TestTheTipFollowsThePointer(t *testing.T) {
	m := downsWorldOn(downsWide(), 100000, 406000, nil)
	h := &screen.Hits{}
	v := view()
	v.Hits = h
	plain := overviewOf(m, v, 120, 40)
	// Over the stockade's cell: its name, in a tip beside it.
	var x, y int
	for r, row := range strings.Split(plain, "\n") {
		if i := strings.Index(row, "Goblin Stockade"); i >= 0 {
			x, y = len([]rune(row[:i]))+3, r
		}
	}
	if got := h.TipAt(x, y); !strings.HasPrefix(got, "Goblin Stockade") {
		t.Fatalf("the tip over the stockade's name is %q", got)
	}
	// Open ground has no tip.
	if got := h.TipAt(h.Map.X+1, h.Map.Y+1); got != "" {
		t.Errorf("the tip over open ground is %q", got)
	}
	v.Hover = &screen.Pt{X: x, Y: y}
	hovered := overviewOf(m, v, 120, 40)
	if strings.Count(hovered, "Goblin Stockade") != 2 {
		t.Errorf("the pointer's tip is not drawn beside the name:\n%s", hovered)
	}
}

// The keyboard cursor does the same where there is no mouse motion.
func TestTheTipFollowsTheCursor(t *testing.T) {
	m := downsWorldOn(downsWide(), 100000, 406000, nil)
	v := view()
	before := overviewOf(m, v, 120, 40)
	v.Cursor = &pb.Pos{X: 364000, Y: 144000} // the stockade
	after := overviewOf(m, v, 120, 40)
	if strings.Contains(before, " Goblin Stockade ") || !strings.Contains(after, " Goblin Stockade ") {
		t.Errorf("the cursor on the stockade shows no tip:\n%s", after)
	}
	v.Cursor = &pb.Pos{X: 327500, Y: 114500} // the rocks, which no name shows
	if out := overviewOf(m, v, 120, 40); !strings.Contains(out, " Goblin Rocks ") {
		t.Errorf("the cursor on the rocks does not name them:\n%s", out)
	}
}
