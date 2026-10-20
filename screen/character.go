package screen

import (
	"fmt"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
)

// The character view (C from the map): the sidebar's Gear, Bags and Stats as a
// panel of its own, with the vitals strip under it. Enter takes off the gear
// under the cursor, or wears the item in the bags.

// charFixedRows are the rows the view keeps for itself: the crowns line, the
// tabs, a rule, the item card (detailRows) and two lines of help.
const charFixedRows = 1 + 1 + 1 + detailRows + 2

// CharSide is the sidebar state the character view uses; a View without one
// (a test of the screen) has the first tab.
func charSide(v View) *Sidebar {
	if v.Side != nil {
		return v.Side
	}
	return &Sidebar{}
}

// CharSelected is the item under the character view's cursor and where it is: the
// gear slot index when gear is true, else the instance in the bags. "" if none.
func CharSelected(m *model.Model, sb *Sidebar) (inst string, gear bool, slot int) {
	switch sb.Tab {
	case TabGear:
		slot = min(max(sb.Cur[TabGear], 0), len(model.GearSlots)-1)
		return m.Worn(slot), true, slot
	case TabBags:
		stacks := m.Stacks(m.Self.Inventory)
		if len(stacks) == 0 {
			return "", false, 0
		}
		return stacks[min(max(sb.Cur[TabBags], 0), len(stacks)-1)].Inst, false, 0
	}
	return "", false, 0
}

// CharRows is how many rows the cursor can move over in the open tab.
func CharRows(m *model.Model, tab int) int {
	switch tab {
	case TabGear:
		return len(model.GearSlots)
	case TabBags:
		return len(m.Stacks(m.Self.Inventory))
	}
	return 0
}

func charLines(m *model.Model, v View, w, h int, st Styler) []string {
	sb := charSide(v)
	s := m.Self
	rows := []string{fit(fmt.Sprintf(" Crowns %d | bags %d of %d slots | bank %d | bind point %s", s.Crowns, m.BagSlotsUsed(), s.InventorySize, len(s.Bank), bindWords(m)), w)}
	head, at := tabHeader(sb.Tab)
	rows = append(rows, st.Bold(fit(head, w)))
	if v.Hits != nil {
		x, y := v.Hits.org.X, v.Hits.org.Y
		for i, n := range sideTabs {
			v.Hits.Tabs = append(v.Hits.Tabs, Tab{Rect: Rect{x + at[i], y + 1, width(n) + 2, 1}, Tab: i})
		}
		v.Hits.Side = Rect{x, y + 1, w, h - 1}
	}
	room := max(h-charFixedRows, 1)
	var body []string
	cur := -1 // the row of the body the cursor is on
	cursor := min(max(sb.Cur[sb.Tab], 0), max(CharRows(m, sb.Tab)-1, 0))
	switch sb.Tab {
	case TabGear:
		body = append([]string{st.Bold(fit(" Gear worn", w))}, gearLines(m, w, cursor, st)...)
		cur = cursor + 1
	case TabBags:
		body = bagLines(m, w, " Items carried", cursor, st)
		if CharRows(m, TabBags) > 0 {
			cur = cursor + 1
		}
	default:
		body = statLines(m, w, st)
	}
	from := 0
	if cur >= 0 {
		from, _ = window(len(body), cur, room)
	} else {
		from = min(max(sb.Scroll[sb.Tab], 0), max(len(body)-room, 0))
		sb.Scroll[sb.Tab] = from
	}
	body = body[from:min(from+room, len(body))]
	rows = append(rows, pad(body, w, room)...)
	rows = append(rows, strings.Repeat("─", w))
	inst, gear, _ := CharSelected(m, sb)
	price := ""
	if !gear {
		price = sellWords(m, inst)
	}
	switch {
	case sb.Tab == TabStats:
		rows = append(rows, pad(nil, w, detailRows)...)
	case inst == "" && sb.Tab == TabGear:
		rows = append(rows, fit(" Nothing is worn there.", w))
		rows = append(rows, pad(nil, w, detailRows-1)...)
	default:
		rows = append(rows, itemDetail(m, inst, w, price)...)
	}
	switch sb.Tab {
	case TabGear:
		rows = append(rows, help(" {Tab} next tab · {Up} / {Down} choose · {Enter} takes it off · out of combat only", w, st))
	case TabBags:
		rows = append(rows, help(" {Tab} next tab · {Up} / {Down} choose · {Enter} wears it · out of combat only", w, st))
	default:
		rows = append(rows, help(" {Tab} next tab · {Up} / {Down} scroll", w, st))
	}
	rows = append(rows, help(" {Esc} or {C} closes · {N} traders · {T} trade", w, st))
	return pad(rows[:min(len(rows), h)], w, h)
}
