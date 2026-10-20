package screen

import (
	"fmt"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
)

// The character sidebar (D78): the gear worn, the bags and the character sheet,
// beside the vendor, trainer, bank, board and talk screens, so that nothing is
// decided blind. In a window too narrow for it, it is a drawer that takes the
// screen's place while C holds it open.

// The tabs of the sidebar.
const (
	TabGear = iota
	TabBags
	TabStats
	Tabs
)

var sideTabs = [Tabs]string{"Gear", "Bags", "Stats"}

// Sidebar is the sidebar's state: the tab, whether the drawer is open (a narrow
// window), and how far each tab is scrolled.
//
// Cur is the cursor in each tab: it is the character view's (C from the map),
// where Enter wears or takes off the item under it. The sidebar beside a service
// has no cursor.
type Sidebar struct {
	Tab    int
	Drawer bool
	Scroll [Tabs]int
	Cur    [Tabs]int
}

// sidebarOpen says the screen is one a sidebar goes beside: the traders (list,
// trainer, bank, vendor), a conversation and the contract board. The trade
// window lists the bags itself, and keeps C for the crowns.
func sidebarOpen(v View) bool {
	return v.Side != nil && (v.Traders != nil || v.Talk != nil || v.Board != nil)
}

// SidebarOpen is sidebarOpen for the program's keys.
func SidebarOpen(v View) bool { return sidebarOpen(v) }

// tabHeader is the line of tab names, the open one in brackets, and where each
// starts on it.
func tabHeader(tab int) (string, [Tabs]int) {
	var at [Tabs]int
	var b strings.Builder
	b.WriteString(" ")
	for i, n := range sideTabs {
		at[i] = width(b.String())
		if i == tab {
			b.WriteString("[" + n + "]")
		} else {
			b.WriteString(" " + n + " ")
		}
		b.WriteString(" ")
	}
	return b.String(), at
}

// sidebarLines draws the sidebar's inside, w x h cells: the tabs and the open
// one. x, y is where the inside starts on the screen, for the tabs' hit blocks.
func sidebarLines(m *model.Model, v View, w, h, x, y int, st Styler) []string {
	sb := v.Side
	head, at := tabHeader(sb.Tab)
	if v.Hits != nil {
		for i, n := range sideTabs {
			v.Hits.Tabs = append(v.Hits.Tabs, Tab{Rect: Rect{x + at[i], y, width(n) + 2, 1}, Tab: i})
		}
		v.Hits.Side = Rect{x, y, w, h}
	}
	rows := []string{st.Bold(fit(head, w))}
	var body []string
	switch sb.Tab {
	case TabGear:
		body = gearLines(m, w, -1, st)
	case TabBags:
		body = bagLines(m, w, "", -1, st)
	default:
		body = statLines(m, w, st)
	}
	room := h - 1
	from := min(max(sb.Scroll[sb.Tab], 0), max(len(body)-room, 0))
	sb.Scroll[sb.Tab] = from
	body = body[from:min(from+room, len(body))]
	rows = append(rows, body...)
	return pad(rows, w, h)
}

// gearLines are the ten gear slots and what is worn in each. cur is the slot the
// cursor is on, -1 for none (the sidebar has no cursor).
func gearLines(m *model.Model, w, cur int, st Styler) []string {
	var rows []string
	for i, slot := range model.GearSlots {
		text := "-"
		pre, cut := " ", 12
		if cur >= 0 {
			pre, cut = "   ", 14
			if i == cur {
				pre = " > "
			}
		}
		if item := m.Worn(i); item != "" {
			text = itemRow(m, item, max(w-cut, 1))
		}
		line := fit(fmt.Sprintf("%s%-9s %s", pre, model.SlotWord(slot), strings.TrimRight(text, " ")), w)
		if i == cur {
			line = st.Reverse(line)
		}
		rows = append(rows, line)
	}
	return rows
}

// bagLines are the bags: the slots used and the crowns (or the title, if one is
// given), and every stack with its mark. cur is the stack the cursor is on, -1 for
// none.
func bagLines(m *model.Model, w int, title string, cur int, st Styler) []string {
	head := fit(fmt.Sprintf(" %d of %d slots · %s", m.BagSlotsUsed(), m.Self.InventorySize, inCrowns(m)), w)
	if title != "" {
		head = st.Bold(fit(title, w))
	}
	rows := []string{head}
	stacks := m.Stacks(m.Self.Inventory)
	if len(stacks) == 0 {
		return append(rows, fit("  (nothing)", w))
	}
	for i, sk := range stacks {
		pre := " "
		if cur >= 0 {
			pre = "   "
			if i == cur {
				pre = " > "
			}
		}
		line := fit(pre+itemRowM(m, sk.Inst, sk.Count, w-width(pre)), w)
		switch {
		case i == cur:
			line = st.Reverse(line)
		case m.NotUsable(sk.Inst):
			line = st.Dim(line)
		}
		rows = append(rows, line)
	}
	return rows
}

// swingName is what the character swings: "Fists" with the main hand empty.
func swingName(m *model.Model) string {
	if inst := m.Worn(slotIndex("main_hand")); inst != "" {
		return m.ItemName(inst)
	}
	return "Fists"
}

func slotIndex(slot string) int {
	for i, s := range model.GearSlots {
		if s == slot {
			return i
		}
	}
	return -1
}

// statLines are the character sheet the server sent (D78): health, mana, armour,
// accuracy, the swing now (bare hands too) and each attribute with what gear adds.
func statLines(m *model.Model, w int, st Styler) []string {
	s := m.Self.Sheet
	if s == nil {
		return []string{fit(" The server has not sent your sheet.", w)}
	}
	rows := []string{
		st.Bold(fit(fmt.Sprintf(" Level %d %s", m.Self.Level, className(m, m.Self.Class)), w)),
		fit(fmt.Sprintf(" Health   %d  (regen %d a tick)", s.GetMaxHp(), s.GetRegen()), w),
	}
	if s.GetMaxMana() > 0 {
		rows = append(rows, fit(fmt.Sprintf(" Mana     %d  (regen %d, resting %d)", s.GetMaxMana(), s.GetManaRegen(), s.GetManaRegenResting()), w))
	}
	rows = append(rows,
		fit(fmt.Sprintf(" Armour   %d  (cuts %d%% of a hit)", s.GetArmor(), s.GetArmorCutPct()), w),
		fit(fmt.Sprintf(" Accuracy %d  (%d%% to hit)", s.GetAccuracy(), s.GetHitPct()), w),
		fit(fmt.Sprintf(" %s: %d–%d / %.1fs  (%.1f damage/s)", swingName(m), s.GetSwingMin(), s.GetSwingMax(), float64(s.GetDelayMs())/1000, float64(s.GetDpsTenths())/10), w),
	)
	for _, a := range s.GetAttributes() {
		line := fmt.Sprintf(" %-9s %d", tierWord(a.GetName()), a.GetBase()+a.GetGear())
		if a.GetGear() != 0 {
			line += fmt.Sprintf("  (%d base %+d gear)", a.GetBase(), a.GetGear())
		}
		rows = append(rows, fit(line, w))
	}
	return rows
}

// sidebarColumn is the right column of a service screen (D78): the You panel, so
// that health and what attacks stay in view (D75), and the character sidebar in
// the rest. x0, y0 is where the column starts on the screen.
func sidebarColumn(m *model.Model, v View, rw, th, x0, y0 int, st Styler) []string {
	iw := rw - 2
	you := youLines(m, v, iw, st)
	rows := box(youTitle(m), rw, len(you)+2, you, st)
	if left := th - len(rows); left >= 6 {
		title := "Your character (C: tab)"
		rows = append(rows, box(title, rw, left, sidebarLines(m, v, iw, left-2, x0+1, y0+len(rows)+1, st), st)...)
	}
	for len(rows) < th {
		rows = append(rows, strings.Repeat(" ", rw))
	}
	return rows[:th]
}
