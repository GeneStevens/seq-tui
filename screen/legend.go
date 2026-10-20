package screen

import (
	"strings"

	"github.com/GeneStevens/seq-tui/model"
)

// legendItem is one entry of the legend: a glyph, drawn in the style it has on
// the map (kind and area go through cellStyle, as the map's cells do), and what
// it means.
type legendItem struct {
	glyph rune
	kind  cellKind
	area  model.AreaKind
	word  string
}

// legendGroup is a labelled line of the legend.
type legendGroup struct {
	label string
	items []legendItem
}

// legendGroups is the key to the map's glyphs and marks (D76), by kind. Every
// glyph is a constant of the map and every kind the one its cell is put down
// with, so the legend and the map cannot draw a glyph differently.
var legendGroups = []legendGroup{
	{"You", []legendItem{
		{glyphSelf, kSelf, "", "you"}, {glyphMarker, kPlan, "", "scout marker"}, {glyphPlan, kPlan, "", "plan end"},
		{'◄', kPointer, "", "objective"},
	}},
	{"People", []legendItem{
		{glyphCharacter, kCharacter, "", "player"}, {glyphTrainer, kDoor, "", "trainer"}, {glyphTrader, kDoor, "", "vendor"},
		{glyphBanker, kDoor, "", "banker"}, {glyphBinder, kDoor, "", "binder"}, {glyphGuard, kDoor, "", "guard"},
		{glyphTalker, kDoor, "", "quest person"}, {glyphBoard, kDoor, "", "board"}, {glyphDoor, kDoor, "", "door"},
	}},
	{"Quest", []legendItem{
		{glyphOffer, kMarkGold, "", "new"}, {glyphAsk, kMarkGold, "", "hand in"}, {glyphAsk, kMarkGrey, "", "in progress"},
		{glyphMob, kQuestMob, "", "quest mob"}, {glyphDoor, kNextDoor, "", "next door"},
	}},
	{"Events", []legendItem{
		{glyphScout, kPlan, "", "scout point"}, {glyphSpawn, kEvent, "", "raiders appear"}, {glyphGoal, kEvent, "", "raiders head for"},
	}},
	{"Creatures", []legendItem{
		{glyphMob, kMob, "", "mob"}, {'M', kTarget, "", "target"}, {glyphDeadMob, kMob, "", "dead"}, {glyphRare, kRare, "", "rare"},
		{glyphDrop, kDrop, "", "loot"}, {glyphLocked, kDrop, "", "not yours"}, {glyphCamp, kCamp, "", "camp"},
	}},
	{"Ground", []legendItem{
		{glyphBuilding, kGround, model.Building, "building"}, {glyphRock, kGround, model.Rock, "rock"}, {glyphWater, kGround, model.Water, "water"},
		{glyphFord, kGround, model.Ford, "ford"}, {glyphBridge, kGround, model.Bridge, "bridge"}, {glyphForest, kGround, model.Forest, "forest"},
		{glyphRoad, kGround, model.Road, "road"}, {glyphFloor, kGround, model.Floor, "floor"}, {glyphBounds, kBounds, "", "zone edge"},
	}},
}

const legendLabelW = 10

// legendLines is the legend at w cells, one labelled line for each kind of thing
// (wrapped under its label when it is too long), the glyphs drawn as the map
// draws them.
func legendLines(m *model.Model, w int, st Styler) []string {
	var out []string
	for _, g := range legendGroups {
		label := g.label
		start := true
		line, plain := "", 0
		emit := func() {
			pre := strings.Repeat(" ", legendLabelW)
			if start {
				pre = st.Bold(fit(label, legendLabelW))
				start = false
			}
			out = append(out, pre+line)
			line, plain = "", 0
		}
		for _, it := range g.items {
			text := string(it.glyph) + " " + it.word
			if plain > 0 && legendLabelW+plain+2+width(text) > w {
				emit()
			}
			if plain > 0 {
				line += "  "
				plain += 2
			}
			line += cellStyle(it.kind, it.area, st)(string(it.glyph)) + " " + it.word
			plain += width(text)
		}
		emit()
	}
	return out
}
