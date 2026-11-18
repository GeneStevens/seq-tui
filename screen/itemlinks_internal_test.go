package screen

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
)

// D206: a link is its name in brackets, in the item's rarity colour, in the chat
// and on the talk screen; without colour the brackets still say it.

var sgrRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

func linkParts(tier string) []model.ChatPart {
	return []model.ChatPart{{Text: "try the "}, {Text: "Hearth Draught", ID: "hearth-draught", Kind: model.KindItem, Tier: tier}, {Text: " now"}}
}

func TestAnItemLinkIsBracketedAndInItsRarityColour(t *testing.T) {
	for _, tc := range []struct {
		tier, sgr string
	}{
		{"common", ""}, {"fine", "\x1b[32m"}, {"rare", "\x1b[94m"}, {"named", "\x1b[1;35m"}, {"", ""},
	} {
		l := model.ChatLine{Channel: "say", From: "Mira", Text: "try the [Hearth Draught] now", Parts: append([]model.ChatPart{}, linkParts(tc.tier)...), Seq: 1}
		m := model.New()
		hits := &Hits{}
		np := &namePlace{h: hits, x: 0, y: 0}
		coloured := strings.Join(np.paint(chatLineRows(m, l, 60, ANSI()), 0, ANSI()), "\n")
		plain := strings.Join((*namePlace)(nil).paint(chatLineRows(m, l, 60, Plain()), 0, Plain()), "\n")
		if !strings.Contains(plain, "try the [Hearth Draught] now") || strings.Contains(plain, "\x1b") {
			t.Errorf("%s: without colour: %q", tc.tier, plain)
		}
		if want := tc.sgr + "[Hearth Draught]"; !strings.Contains(coloured, want) {
			t.Errorf("%s: with colour %q lacks %q", tc.tier, coloured, want)
		}
		if len(hits.Items) != 1 || hits.Items[0].ID != "hearth-draught" || len(hits.Names) != 1 || hits.Names[0].Name != "Mira" { // the item is no name to open a card for
			t.Errorf("%s: hits %+v names %+v", tc.tier, hits.Items, hits.Names)
		}
	}
}

func TestTalkLinesColourTheLinkAndWrapEveryRowToTheWidth(t *testing.T) {
	text := "Take a [Hearth Draught] to the east gate and show it to the guard there."
	parts := append(linkParts("rare"), model.ChatPart{Text: ", then rest."})
	for _, tc := range []struct {
		name string
		st   Styler
		want string
	}{
		{"plain", Plain(), "[Hearth Draught]"},
		{"colour", ANSI(), "\x1b[94m[Hearth Draught]\x1b["},
	} {
		rows := speechRows(text, parts, 30, tc.st)
		joined := strings.Join(rows, "\n")
		if !strings.Contains(joined, tc.want) || !strings.HasPrefix(strings.TrimLeft(joined, " "), "“try the") {
			t.Errorf("%s: %q", tc.name, joined)
		}
		for _, r := range rows {
			if n := width(sgrRe.ReplaceAllString(r, "")); n != 30 {
				t.Errorf("%s: row %q is %d cells wide, want 30", tc.name, r, n)
			}
		}
	}
	if rows := speechRows("Hello.", nil, 30, Plain()); len(rows) != 1 || !strings.Contains(rows[0], "“Hello.”") {
		t.Errorf("a line with no links: %q", rows)
	}
}

func TestTheItemScreenShowsTheCardAndTheChoice(t *testing.T) {
	m := model.New()
	pick := &ItemUI{Pick: []model.ChatPart{{Text: "Hearth Draught", ID: "h", Tier: "fine"}, {Text: "Iron Blade", ID: "i", Tier: "rare"}}, Sel: 1}
	got := strings.Join(itemLines(m, pick, 60, 8, Plain()), "\n")
	for _, want := range []string{"1  [Hearth Draught] Fine", "> 2  [Iron Blade] Rare"} {
		if !strings.Contains(got, want) {
			t.Errorf("the choice lacks %q:\n%s", want, got)
		}
	}
	if got := strings.Join(itemLines(m, &ItemUI{ID: "unseen"}, 60, 8, Plain()), "\n"); !strings.Contains(got, "…") {
		t.Errorf("an item not yet answered:\n%s", got)
	}
}
