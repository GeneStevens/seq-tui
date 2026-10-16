package screen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Inventory is the inventory and gear screen, when it is open: which of the
// two lists the keys go to (the gear worn, or the items carried) and the cursor
// in each.
type Inventory struct {
	Carried bool // the cursor is on the items carried; else on the gear worn
	Worn    int  // index into model.GearSlots
	Item    int  // index into the carried items
}

// Traders is the screen for the traders in view: the list of them, and the
// window of one when it is open (Open). Pane and Cur are the lists inside a
// window: the first and the second, and the cursor in each.
type Traders struct {
	Sel  int
	Open bool
	Pane int
	Cur  [2]int
}

// TradeUI is the trade window's screen state: who the cursor is on while there
// is no window, then the list the keys go to (the items carried, or the offer)
// and the cursor in each, and the crowns being typed.
type TradeUI struct {
	Who     int
	Offer   bool // the cursor is on the offer; else on the items carried
	Item    int
	Mine    int
	Typing  bool
	Crowns  string
	MaxCash int
}

// tierWord is an item's tier as a player reads it.
func tierWord(tier string) string {
	if tier == "" {
		return ""
	}
	return strings.ToUpper(tier[:1]) + tier[1:]
}

// statsWords lists the non-zero stats an item adds, in few words.
func statsWords(s *pb.Stats) string {
	if s == nil {
		return ""
	}
	var out []string
	add := func(name string, v int64) {
		if v != 0 {
			out = append(out, fmt.Sprintf("%s %+d", name, v))
		}
	}
	if s.GetWeaponMax() > 0 {
		w := fmt.Sprintf("damage %d-%d", s.GetWeaponMin(), s.GetWeaponMax())
		if d := s.GetWeaponDelayMs(); d > 0 {
			w += fmt.Sprintf(" / %.1fs", float64(d)/1000)
		}
		out = append(out, w)
	}
	add("armor", s.GetArmor())
	add("hp", s.GetMaxHp())
	add("mana", s.GetMaxMana())
	add("str", s.GetStrength())
	add("agi", s.GetAgility())
	add("sta", s.GetStamina())
	add("int", s.GetIntellect())
	add("wis", s.GetWisdom())
	add("accuracy", s.GetAccuracy())
	add("evasion", s.GetEvasion())
	add("regen", s.GetRegen())
	add("mana regen", s.GetManaRegen())
	return strings.Join(out, ", ")
}

// bindingWords says whether an item can be traded: Rare and Named items are
// bound on pickup, the rest can be traded exactly once and are bound after it.
func bindingWords(m *model.Model, inst string) string {
	switch it := m.Item(inst); {
	case model.Traded(inst):
		return "bound: traded once"
	case it != nil && it.GetBindOnPickup():
		return "bound on pickup"
	case it == nil:
		return ""
	}
	return "tradable once"
}

// itemRow is one item in a list: its name, tier and binding marks, then what it
// adds, cut to w cells.
func itemRow(m *model.Model, inst string, w int) string {
	it := m.Item(inst)
	name := m.ItemName(inst)
	if model.Traded(inst) {
		name += model.BoundMark
	}
	if it == nil {
		return fit(name, w)
	}
	mark := ""
	if m.Bound(inst) {
		mark = " [bound]"
	}
	left := fmt.Sprintf("%s %s%s", name, tierWord(it.GetTier()), mark)
	if width(left) > w { // no room for the tier: the name matters most
		return fit(name, w)
	}
	if w >= 60 {
		if s := statsWords(it.GetStats()); s != "" {
			left += "  " + s
		}
	}
	return fit(left, w)
}

// itemDetail are the lines that explain one item: what it is, what it adds and
// what a vendor pays.
func itemDetail(m *model.Model, inst string, w int) []string {
	it := m.Item(inst)
	if inst == "" || it == nil {
		return []string{fit("", w), fit("", w)}
	}
	slot := it.GetSlot()
	if slot == "" || slot == "none" {
		slot = "no gear slot"
	}
	l1 := fmt.Sprintf(" %s: %s, %s; %s", m.ItemName(inst), tierWord(it.GetTier()), slot, bindingWords(m, inst))
	l2 := " " + statsWords(it.GetStats())
	if s := it.GetSellPrice(); s > 0 {
		if l2 != " " {
			l2 += "; "
		}
		l2 += fmt.Sprintf("a vendor pays %d", s)
	}
	return []string{fit(l1, w), fit(l2, w)}
}

// cols puts two lists of rows side by side, each w1 and w2 cells wide, with a
// bar between.
func cols(left, right []string, w1, w2 int) []string {
	n := max(len(left), len(right))
	out := make([]string, n)
	for i := range out {
		l, r := strings.Repeat(" ", w1), strings.Repeat(" ", w2)
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		out[i] = l + "│" + r
	}
	return out
}

// window shows rows [from, from+n) of list so that cursor is inside.
func window(total, cursor, n int) (from, to int) {
	if n <= 0 {
		return 0, 0
	}
	from = max(min(cursor-n+1, total-n), 0)
	if cursor < from {
		from = cursor
	}
	return from, min(from+n, total)
}

func inCrowns(m *model.Model) string { return crownsWord(m.Self.Crowns) }

func crownsWord(n int64) string {
	if n == 1 {
		return "1 crown"
	}
	return fmt.Sprintf("%d crowns", n)
}

// listRows draws rows[i] for i in the window around cur with a cursor mark; the
// row under the cursor is reversed when focus is set.
func listRows(rows []string, cur, n, w int, focus bool, st Styler) []string {
	var out []string
	if len(rows) == 0 {
		return out
	}
	from, to := window(len(rows), cur, n)
	for i := from; i < to; i++ {
		mark := "  "
		if i == cur {
			mark = "> "
		}
		line := fit(mark+rows[i], w)
		if i == cur && focus {
			line = st.Reverse(line)
		}
		out = append(out, line)
	}
	return out
}

func inventoryScreenLines(m *model.Model, inv *Inventory, w, h int, st Styler) []string {
	s := m.Self
	var rows []string
	add := func(l string) { rows = append(rows, fit(l, w)) }
	add(fmt.Sprintf(" Crowns %d | carrying %d of %d | bank %d | bind point %s", s.Crowns, len(s.Inventory), s.InventorySize, len(s.Bank), bindWords(m)))
	lw := (w - 1) / 2
	rw := w - 1 - lw
	// The gear worn, ten slots, and the items carried.
	n := len(model.GearSlots)
	worn := []string{fit(" Gear worn", lw)}
	for i, slot := range model.GearSlots {
		item := m.Worn(i)
		text := "-"
		if item != "" {
			text = itemRow(m, item, max(lw-14, 1))
		}
		mark := "  "
		if !inv.Carried && i == inv.Worn {
			mark = "> "
		}
		line := fit(fmt.Sprintf(" %s%-9s %s", mark, slot, strings.TrimRight(text, " ")), lw)
		if !inv.Carried && i == inv.Worn {
			line = st.Reverse(line)
		}
		worn = append(worn, line)
	}
	carried := []string{fit(" Items carried", rw)}
	room := max(n, 1)
	if len(s.Inventory) == 0 {
		carried = append(carried, fit("  (nothing)", rw))
	} else {
		var texts []string
		for _, inst := range s.Inventory {
			texts = append(texts, itemRow(m, inst, max(rw-4, 1)))
		}
		carried = append(carried, listRows(texts, inv.Item, room, rw, inv.Carried, st)...)
	}
	rows = append(rows, cols(worn, carried, lw, rw)...)
	sel := ""
	if inv.Carried && inv.Item < len(s.Inventory) {
		sel = s.Inventory[inv.Item]
	} else if !inv.Carried {
		sel = m.Worn(inv.Worn)
	}
	rows = append(rows, strings.Repeat("─", w))
	for _, l := range itemDetail(m, sel, w) {
		rows = append(rows, l)
	}
	add(" Tab: gear / items. Up / Down choose. Enter wears it or takes it off.")
	add(" Out of combat only. Esc or C closes. N: traders. T: trade.")
	return pad(rows[:min(len(rows), h)], w, h)
}

func bindWords(m *model.Model) string {
	if m.Self.BindZone == "" {
		return "none yet"
	}
	return m.Self.BindZone
}

// roleWord is what a trader does, in a word.
func roleWord(role string) string {
	switch role {
	case "vendor":
		return "sells and buys"
	case "trainer":
		return "teaches"
	case "binder":
		return "sets your bind point"
	case "banker":
		return "keeps your bank"
	}
	return role
}

// Traders lists the vendors, trainers, binders and bankers in view, nearest first.
func tradersInView(m *model.Model) []*pb.NpcInfo {
	list := append([]*pb.NpcInfo(nil), m.NPCs...)
	sp := m.Self.Pos
	sort.SliceStable(list, func(i, j int) bool {
		di := dist(list[i].GetPos().GetX(), list[i].GetPos().GetY(), sp.GetX(), sp.GetY())
		dj := dist(list[j].GetPos().GetX(), list[j].GetPos().GetY(), sp.GetX(), sp.GetY())
		if di != dj {
			return di < dj
		}
		return list[i].GetId() < list[j].GetId()
	})
	return list
}

// TradersInView is the traders the screen lists, in its order, for the keys.
func TradersInView(m *model.Model) []*pb.NpcInfo { return tradersInView(m) }

func tradersScreenLines(m *model.Model, t *Traders, w, h int, st Styler) []string {
	list := tradersInView(m)
	var rows []string
	add := func(l string) { rows = append(rows, fit(l, w)) }
	sp := m.Self.Pos
	if !t.Open || t.Sel >= len(list) {
		add(" Traders in view. Up / Down choose, Enter opens, Esc closes. Walk up to one first.")
		add("")
		if len(list) == 0 {
			add(" Nobody here trades, teaches or binds.")
		}
		for i, n := range list {
			d := dist(n.GetPos().GetX(), n.GetPos().GetY(), sp.GetX(), sp.GetY())
			mark := "  "
			if i == t.Sel {
				mark = "> "
			}
			line := fit(fmt.Sprintf(" %s%-24s %-22s %s %s", mark, n.GetName(), roleWord(n.GetRole()), metres(d), bearing(n.GetPos().GetX()-sp.GetX(), n.GetPos().GetY()-sp.GetY())), w)
			if i == t.Sel {
				line = st.Reverse(line)
			}
			rows = append(rows, line)
		}
		return pad(rows[:min(len(rows), h)], w, h)
	}
	n := list[t.Sel]
	d := dist(n.GetPos().GetX(), n.GetPos().GetY(), sp.GetX(), sp.GetY())
	add(fmt.Sprintf(" %s, %s, %s %s away. You have %s.", n.GetName(), n.GetRole(), metres(d), bearing(n.GetPos().GetX()-sp.GetX(), n.GetPos().GetY()-sp.GetY()), inCrowns(m)))
	add("")
	body := h - len(rows) - 3
	switch n.GetRole() {
	case "vendor":
		rows = append(rows, vendorLines(m, n, t, w, body, st)...)
	case "trainer":
		rows = append(rows, trainerLines(m, n, t, w, body, st)...)
	case "binder":
		add(fmt.Sprintf(" Bind point now: %s.", bindWords(m)))
		add(" Enter sets it here: you respawn and gate to the nearest bind point you set. It costs crowns.")
	case "banker":
		rows = append(rows, bankLines(m, t, w, body, st)...)
	}
	for len(rows) < h-2 {
		rows = append(rows, strings.Repeat(" ", w))
	}
	rows = rows[:h-2]
	switch n.GetRole() {
	case "vendor", "banker":
		add(" Tab switches list; Up / Down choose; Enter acts on the item. Esc goes back.")
	default:
		add(" Enter acts. Esc goes back.")
	}
	add(" The answer is on the status line; every denial says why.")
	return pad(rows[:min(len(rows), h)], w, h)
}

func vendorLines(m *model.Model, n *pb.NpcInfo, t *Traders, w, h int, st Styler) []string {
	lw := (w - 1) / 2
	rw := w - 1 - lw
	sale := []string{fit(" For sale (Enter buys one)", lw)}
	var texts []string
	for _, s := range n.GetStock() {
		texts = append(texts, fmt.Sprintf("%-*s %4d", max(lw-10, 8), strings.TrimRight(itemRow(m, s.GetItem(), max(lw-10, 8)), " "), s.GetPrice()))
	}
	if len(texts) == 0 {
		sale = append(sale, fit("  (nothing)", lw))
	}
	sale = append(sale, listRows(texts, t.Cur[0], max(h-3, 1), lw, t.Pane == 0, st)...)
	yours := []string{fit(" You carry (Enter sells one)", rw)}
	texts = nil
	for _, inst := range m.Self.Inventory {
		price := ""
		if it := m.Item(inst); it != nil {
			price = fmt.Sprintf("%4d", it.GetSellPrice())
		}
		texts = append(texts, fmt.Sprintf("%-*s %s", max(rw-10, 8), strings.TrimRight(itemRow(m, inst, max(rw-10, 8)), " "), price))
	}
	if len(texts) == 0 {
		yours = append(yours, fit("  (nothing)", rw))
	}
	yours = append(yours, listRows(texts, t.Cur[1], max(h-3, 1), rw, t.Pane == 1, st)...)
	rows := cols(sale, yours, lw, rw)
	sel := ""
	switch {
	case t.Pane == 0 && t.Cur[0] < len(n.GetStock()):
		sel = n.GetStock()[t.Cur[0]].GetItem()
	case t.Pane == 1 && t.Cur[1] < len(m.Self.Inventory):
		sel = m.Self.Inventory[t.Cur[1]]
	}
	rows = append(rows, strings.Repeat("─", w))
	return append(rows, itemDetail(m, sel, w)...)
}

func trainerLines(m *model.Model, n *pb.NpcInfo, t *Traders, w, h int, st Styler) []string {
	rows := []string{fit(fmt.Sprintf(" You are level %d. Enter buys the ability under the cursor.", m.Self.Level), w)}
	var texts []string
	for _, tr := range n.GetTrains() {
		state := fmt.Sprintf("%d crowns", tr.GetCost())
		if tr.GetKnown() {
			state = "known"
		}
		texts = append(texts, fmt.Sprintf("%-22s level %-3d %s", m.AbilityName(tr.GetAbility()), tr.GetLevel(), state))
	}
	if len(texts) == 0 {
		rows = append(rows, fit("  (nothing to teach you)", w))
	}
	return append(rows, listRows(texts, t.Cur[0], max(h-1, 1), w, true, st)...)
}

func bankLines(m *model.Model, t *Traders, w, h int, st Styler) []string {
	lw := (w - 1) / 2
	rw := w - 1 - lw
	left := []string{fit(" You carry (Enter banks it)", lw)}
	var texts []string
	for _, inst := range m.Self.Inventory {
		texts = append(texts, itemRow(m, inst, max(lw-4, 1)))
	}
	if len(texts) == 0 {
		left = append(left, fit("  (nothing)", lw))
	}
	left = append(left, listRows(texts, t.Cur[0], max(h-3, 1), lw, t.Pane == 0, st)...)
	right := []string{fit(fmt.Sprintf(" In the bank, %d (Enter takes it)", len(m.Self.Bank)), rw)}
	texts = nil
	for _, inst := range m.Self.Bank {
		texts = append(texts, itemRow(m, inst, max(rw-4, 1)))
	}
	if len(texts) == 0 {
		right = append(right, fit("  (empty)", rw))
	}
	right = append(right, listRows(texts, t.Cur[1], max(h-3, 1), rw, t.Pane == 1, st)...)
	return cols(left, right, lw, rw)
}

// TraderPeers are the other characters the trade screen can ask: those in view
// and the character's own group, nearest first, without the dead.
func TraderPeers(m *model.Model) []*pb.EntityState {
	var out []*pb.EntityState
	for _, id := range m.GroupIDs() {
		if e := m.Group[id].Entity; !e.GetDead() {
			out = append(out, e)
		}
	}
	for _, id := range m.EntityIDs() {
		if e := m.Entities[id]; e.GetKind() == pb.EntityKind_ENTITY_KIND_CHARACTER && !e.GetDead() {
			out = append(out, e)
		}
	}
	sp := m.Self.Pos
	sort.SliceStable(out, func(i, j int) bool {
		di := dist(out[i].GetPos().GetX(), out[i].GetPos().GetY(), sp.GetX(), sp.GetY())
		dj := dist(out[j].GetPos().GetX(), out[j].GetPos().GetY(), sp.GetX(), sp.GetY())
		if di != dj {
			return di < dj
		}
		return out[i].GetId() < out[j].GetId()
	})
	return out
}

func offerRows(m *model.Model, o *pb.TradeOffer, confirmed bool, w int, sel int, focus bool, st Styler, owned bool) []string {
	var rows []string
	state := "not confirmed"
	if confirmed {
		state = "CONFIRMED"
	}
	rows = append(rows, fit(" "+state, w))
	rows = append(rows, fit(fmt.Sprintf(" crowns %d", o.GetCrowns()), w))
	var texts []string
	for _, it := range o.GetItems() {
		texts = append(texts, itemRow(m, it, max(w-4, 1)))
	}
	if len(texts) == 0 {
		rows = append(rows, fit("  (no items)", w))
	}
	if owned {
		return append(rows, listRows(texts, sel, 6, w, focus, st)...)
	}
	for _, t := range texts {
		rows = append(rows, fit("  "+t, w))
	}
	return rows
}

func tradeScreenLines(m *model.Model, tu *TradeUI, w, h int, st Styler) []string {
	var rows []string
	add := func(l string) { rows = append(rows, fit(l, w)) }
	tr := m.Trade
	peers := TraderPeers(m)
	if tr == nil || !tr.GetOpen() {
		add(fmt.Sprintf(" Trade with another character in reach. You have %s.", inCrowns(m)))
		if tr != nil && tr.GetIncoming() {
			add(fmt.Sprintf(" %s asked you for a trade: Enter agrees (choose them below).", m.Label(tr.GetWith())))
		} else {
			add(" Up / Down choose a character; Enter asks them. They ask back to open it.")
		}
		add("")
		for i, e := range peers {
			sp := m.Self.Pos
			mark := "  "
			if i == tu.Who {
				mark = "> "
			}
			extra := ""
			if tr != nil && tr.GetWith() == e.GetId() {
				extra = "  (asked you)"
				if !tr.GetIncoming() {
					extra = "  (you asked)"
				}
			}
			line := fit(fmt.Sprintf(" %s%-20s %s %s%s", mark, e.GetName(), metres(dist(e.GetPos().GetX(), e.GetPos().GetY(), sp.GetX(), sp.GetY())), bearing(e.GetPos().GetX()-sp.GetX(), e.GetPos().GetY()-sp.GetY()), extra), w)
			if i == tu.Who {
				line = st.Reverse(line)
			}
			rows = append(rows, line)
		}
		if len(peers) == 0 {
			add(" Nobody else is in view.")
		}
		for len(rows) < h-2 {
			rows = append(rows, strings.Repeat(" ", w))
		}
		rows = rows[:h-2]
		add(" Esc closes. Traded items are bound afterwards; Rare and Named never trade.")
		add(" The answer is on the status line; every denial says why.")
		return pad(rows[:min(len(rows), h)], w, h)
	}
	add(fmt.Sprintf(" Trade with %s. Changing an offer clears the confirmations.", m.Label(tr.GetWith())))
	lw := (w - 1) / 2
	rw := w - 1 - lw
	mine := append([]string{st.Bold(fit(" Your offer", lw))}, offerRows(m, tr.GetMine(), tr.GetMineConfirmed(), lw, tu.Mine, tu.Offer, st, true)...)
	theirs := append([]string{st.Bold(fit(" "+m.Label(tr.GetWith())+"'s offer", rw))}, offerRows(m, tr.GetTheirs(), tr.GetTheirsConfirmed(), rw, 0, false, st, false)...)
	rows = append(rows, cols(mine, theirs, lw, rw)...)
	rows = append(rows, strings.Repeat("─", w))
	add(fmt.Sprintf(" You carry, %s (Enter puts the item in the offer)", inCrowns(m)))
	var texts []string
	for _, inst := range m.Self.Inventory {
		texts = append(texts, itemRow(m, inst, max(w-4, 1)))
	}
	if len(texts) == 0 {
		add("  (nothing)")
	}
	room := h - len(rows) - 3
	rows = append(rows, listRows(texts, tu.Item, max(room, 1), w, !tu.Offer, st)...)
	for len(rows) < h-3 {
		rows = append(rows, strings.Repeat(" ", w))
	}
	rows = rows[:h-3]
	if tu.Typing {
		add(fmt.Sprintf(" Crowns to offer in all: %s_   Enter sets it, Esc stops.", tu.Crowns))
	} else {
		add(" Tab: list  Enter: add / take back  C: crowns  Y: confirm  X: cancel trade")
	}
	add(" Esc closes the screen and leaves the trade open.")
	add(" The server refuses Rare and Named items, and ones already traded.")
	return pad(rows[:min(len(rows), h)], w, h)
}
