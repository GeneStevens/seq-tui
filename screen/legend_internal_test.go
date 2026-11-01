package screen

import (
	"strings"
	"testing"
)

// The legend draws each glyph through the style table the map draws through, so a
// glyph looks in the legend as it does on the map.
func TestLegendGlyphsAreInTheirMapStyles(t *testing.T) {
	st := ANSI()
	text := strings.Join(legendLines(nil, 120, st, nil), "\n")
	for _, g := range legendGroups {
		for _, it := range g.items {
			// The map's own path: a cell of this kind in a grid, styled.
			grid := &grid{cw: 1, ch: 1, cells: [][]cell{{{r: it.glyph, k: it.kind, a: it.area}}}}
			onMap := grid.styled(st)[0]
			if want := cellStyle(it.kind, it.area, st)(string(it.glyph)); onMap != want {
				t.Errorf("%q: the map draws %q, the style table %q", it.word, onMap, want)
			}
			if !strings.Contains(text, onMap+" "+it.word) {
				t.Errorf("the legend does not draw %q as the map does (%q)", it.word, onMap)
			}
		}
	}
}

func TestLegendIsGroupedByKind(t *testing.T) {
	lines := legendLines(nil, 76, Plain(), nil)
	for _, label := range []string{"You", "People", "Quest", "Events", "Creatures", "Ground"} {
		n := 0
		for _, l := range lines {
			if strings.HasPrefix(l, label+strings.Repeat(" ", legendLabelW-len(label))) {
				n++
			}
		}
		if n != 1 {
			t.Errorf("%q labels %d lines", label, n)
		}
	}
	for _, l := range lines {
		if width(l) > 76 {
			t.Errorf("too wide: %q", l)
		}
	}
	// A world event's marks have a colour no person or quest mark has.
	st := ANSI()
	ev := cellStyle(kEvent, "", st)("v")
	for _, k := range []cellKind{kPlan, kDoor, kMarkGold, kMob, kDrop, kQuestMob, kCharacter} {
		if cellStyle(k, "", st)("v") == ev {
			t.Errorf("the event colour is also kind %d's", k)
		}
	}
	if glyphGoal == glyphGuard || glyphGoal == 'G' {
		t.Errorf("the event goal is %q, too like the guard", glyphGoal)
	}
}
