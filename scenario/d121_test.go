package scenario_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/testserver"
)

// D121: on the large map the keyboard cursor, moved onto a spot, names the place
// or places there in a one-line tip; the keyboard works wherever the terminal
// reports no mouse motion.
func TestLargeMapCursorNamesThePlaceUnderIt(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: []testserver.Account{{Name: "ann", Classed: map[string]string{"hale": "warrior"}}}})
	p := w.Login("ann", "hale")
	p.Resize(140, 45)
	var here []string
	p.Model(func(m *model.Model) {
		self := m.Self.Pos
		for _, pl := range m.PlaceLabels() {
			if pl.Touches(self.GetX()-2000, self.GetY()-2000, self.GetX()+2000, self.GetY()+2000) {
				here = append(here, pl.Name)
			}
		}
	})
	if len(here) == 0 {
		t.Fatal("the character stands in no named place of Thornwick")
	}
	p.Press("m")
	before := p.Screen(140, 45)
	p.Press("right", "left") // the cursor appears on the character's own cell
	after := p.Screen(140, 45)
	for _, name := range here {
		if strings.Count(after, name) <= strings.Count(before, name) {
			t.Errorf("the cursor on the character does not name %q in a tip:\n%s", name, after)
		}
	}
}
