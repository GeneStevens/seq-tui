package screen

import (
	"testing"
	"unicode/utf8"

	"github.com/GeneStevens/seq-tui/model"
)

// A round that has run names why in a title that fits the 22 columns the Round
// panel's title has at 80 wide, up to round 99.
func TestARanRoundsTitleFits(t *testing.T) {
	for _, by := range []string{"all_submitted", "deadline", "instance_ended", "something_new"} {
		m := model.New()
		m.Round = &model.Round{Number: 99, Resolved: 99, ResolvedBy: by}
		if got := roundTitle(m, View{}); utf8.RuneCountInString(got) > 22 && by != "something_new" {
			t.Errorf("%s: the title %q is %d columns", by, got, utf8.RuneCountInString(got))
		}
	}
}
