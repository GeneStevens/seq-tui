package screen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Traders is the screen for the traders in view: the list of them, and the
// window of one when it is open (Open). Pane and Cur are the lists inside a
// window: the first and the second, and the cursor in each.
//
// A window stays with the NPC it was opened on (D78): NPC is its id, not a place
// in the list, which is sorted by distance and changes as people walk by.
type Traders struct {
	Sel  int
	NPC  string
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
func itemRow(m *model.Model, inst string, w int) string { return itemRowN(m, inst, 1, w) }

// markCol is the item's mark as the first cell of a row: ▲ upgrade, ▼ downgrade,
// = sidegrade, x not usable by this character, a blank for what is not gear (D78).
func markCol(m *model.Model, inst string) string {
	if mk := m.Mark(inst); mk != "" {
		return mk
	}
	return " "
}

// itemRowM is itemRow with the item's mark in front, for every list of items to
// be compared: bags, vendor lists, loot (D78). A stack shows its count.
func itemRowM(m *model.Model, inst string, count, w int) string {
	return markCol(m, inst) + " " + itemRowN(m, inst, count, max(w-2, 1))
}

// greyed marks a row to be drawn dim: what cannot be used or afforded (D78).
const greyed = "\x01"

// greyIf marks the row greyed out when cond holds.
func greyIf(cond bool, row string) string {
	if cond {
		return greyed + row
	}
	return row
}

func itemRowN(m *model.Model, inst string, count, w int) string {
	it := m.Item(inst)
	name := m.ItemName(inst)
	if model.Traded(inst) {
		name += model.BoundMark
	}
	if count > 1 {
		name += fmt.Sprintf(" ×%d", count)
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
	if m.NotUsable(inst) {
		if why := m.FactWords(inst); why != "" && width(left)+2+width(why) <= w {
			left += "  " + why // the reason it is greyed out
		}
	} else if w >= 60 {
		if s := statsWords(it.GetStats()); s != "" {
			left += "  " + s
		}
	}
	return fit(left, w)
}

// detailRows is how many lines itemDetail gives.
const detailRows = 3

// itemDetail is the card of one item, in three lines (D78): what it is, what it
// adds and what a vendor says of it, and how it compares with what is worn in
// its slot. price, if set, goes after what it adds.
func itemDetail(m *model.Model, inst string, w int, price string) []string {
	it := m.Item(inst)
	if inst == "" || it == nil {
		return []string{fit("", w), fit("", w), fit("", w)}
	}
	slot := model.SlotWord(it.GetSlot())
	mark := ""
	if mk := m.Mark(inst); mk != "" {
		mark = mk + " "
	}
	l1 := fmt.Sprintf(" %s%s: %s, %s; %s", mark, m.ItemName(inst), tierWord(it.GetTier()), slot, bindingWords(m, inst))
	l2 := " " + statsWords(it.GetStats())
	if price != "" {
		if l2 != " " {
			l2 += "; "
		}
		l2 += price
	}
	l3 := " " + m.FactWords(inst)
	if f := m.Fact(inst); f == nil || m.FactWords(inst) == "" {
		l3 = ""
	}
	return []string{fit(l1, w), fit(l2, w), fit(l3, w)}
}

// sellWords is what a vendor pays for an item, for its card.
func sellWords(m *model.Model, inst string) string {
	if it := m.Item(inst); it != nil && it.GetSellPrice() > 0 {
		return fmt.Sprintf("a vendor pays %s", crownsWord(it.GetSellPrice()))
	}
	return ""
}

// priceWords is what a vendor asks for an item and whether the character can pay.
func priceWords(m *model.Model, price int64) string {
	if short := price - m.Self.Crowns; short > 0 {
		return fmt.Sprintf("costs %s, short %s", crownsWord(price), crownsWord(short))
	}
	return "costs " + crownsWord(price)
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

// dimKeepMark draws a greyed row dim except for the item's mark (▲ ▼ = x), which
// keeps its weight: what cannot be afforded or used is still worth comparing.
func dimKeepMark(line string, st Styler) string {
	r := []rune(line)
	if len(r) > 3 && r[3] == ' ' && strings.ContainsRune("▲▼=x", r[2]) {
		return st.Dim(string(r[:2])) + st.Bold(string(r[2])) + st.Dim(string(r[3:]))
	}
	return st.Dim(line)
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
		grey := strings.HasPrefix(rows[i], greyed)
		line := fit(mark+strings.TrimPrefix(rows[i], greyed), w)
		switch {
		case i == cur && focus:
			line = st.Reverse(line)
		case grey:
			line = dimKeepMark(line, st)
		}
		out = append(out, line)
	}
	return out
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
	case "contract_board":
		return "posts contracts"
	case "quest_giver":
		return "talks"
	case "guard":
		return "keeps watch over the town"
	}
	return role
}

// tradersInView lists the NPCs the character has seen in this space, nearest first.
func tradersInView(m *model.Model) []*pb.NpcInfo {
	list := m.KnownNPCs()
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

// titled is an NPC's name and title.
func titled(n *pb.NpcInfo) string { return model.NPCLabel(n) }

// TradersInView is the traders the screen lists, in its order, for the keys.
func TradersInView(m *model.Model) []*pb.NpcInfo { return tradersInView(m) }

func tradersScreenLines(m *model.Model, t *Traders, w, h int, st Styler) []string {
	list := tradersInView(m)
	var rows []string
	add := func(l string) { rows = append(rows, fit(l, w)) }
	sp := m.Self.Pos
	var n *pb.NpcInfo
	if t.Open {
		n = m.NPC(t.NPC)
	}
	if n == nil {
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
			far := ""
			if !m.InReach(n) {
				far = " too far"
			}
			right := metres(d) + " " + bearing(n.GetPos().GetX()-sp.GetX(), n.GetPos().GetY()-sp.GetY()) + far
			line := fit(fit(fmt.Sprintf(" %s%c %s: %s", mark, npcGlyph(n), titled(n), roleWord(n.GetRole())), w-width(right)-1)+" "+right, w)
			switch {
			case i == t.Sel:
				line = st.Reverse(line)
			case far != "":
				line = st.Dim(line) // beyond its reach: greyed out
			}
			rows = append(rows, line)
		}
		return pad(rows[:min(len(rows), h)], w, h)
	}
	d := dist(n.GetPos().GetX(), n.GetPos().GetY(), sp.GetX(), sp.GetY())
	add(fmt.Sprintf(" %s, %s %s away. You have %s.", model.NPCLabel(n), metres(d), bearing(n.GetPos().GetX()-sp.GetX(), n.GetPos().GetY()-sp.GetY()), inCrowns(m)))
	if !m.InReach(n) {
		add(fmt.Sprintf(" Too far: stand within %s of %s to deal with them.", metres(n.GetReachMm()), n.GetName()))
	} else {
		add("")
	}
	body := h - len(rows) - 3
	switch n.GetRole() {
	case "vendor":
		rows = append(rows, vendorLines(m, n, t, w, body, st)...)
	case "trainer":
		rows = append(rows, trainerLines(m, n, t, w, body, st)...)
	case "binder":
		add(fmt.Sprintf(" Bind point now: %s.", bindWords(m)))
		add(fmt.Sprintf(" Enter sets it here for %s: you respawn there, and Homeward brings you back to it.", crownsWord(n.GetBindFee())))
	case "banker":
		rows = append(rows, bankLines(m, t, w, body, st)...)
	case "contract_board":
		add(" Contracts: Enter opens the board, where you take them and hand them in.")
	default:
		add(" Nothing to buy or sell here. G talks to the nearest person.")
	}
	for len(rows) < h-2 {
		rows = append(rows, strings.Repeat(" ", w))
	}
	rows = rows[:h-2]
	switch n.GetRole() {
	case "vendor":
		rows = append(rows, help(" {Tab} list · {Enter} buy / sell one · {A} sell the stack · {J} sell all junk · {Esc} back", w, st))
	case "banker":
		rows = append(rows, help(" {Tab} switches list · {Up} / {Down} choose · {Enter} acts on the item · {Esc} back", w, st))
	default:
		rows = append(rows, help(" {Enter} acts · {Esc} back", w, st))
	}
	rows = append(rows, help(" The answer is on the status line; every denial says why.", w, st))
	return pad(rows[:min(len(rows), h)], w, h)
}

func vendorLines(m *model.Model, n *pb.NpcInfo, t *Traders, w, h int, st Styler) []string {
	lw := (w - 1) / 2
	rw := w - 1 - lw
	listN := max(h-4, 1)
	sale := []string{st.Bold(fit(" For sale (Enter buys one)", lw))}
	var texts []string
	for _, s := range n.GetStock() {
		price := fmt.Sprintf("%4d", s.GetPrice())
		poor := s.GetPrice() > m.Self.Crowns
		if poor {
			price = fmt.Sprintf("%d, short %d", s.GetPrice(), s.GetPrice()-m.Self.Crowns)
		}
		room := max(lw-4-width(price), 8)
		texts = append(texts, greyIf(poor || m.NotUsable(s.GetItem()), fmt.Sprintf("%-*s %s", room, strings.TrimRight(itemRowM(m, s.GetItem(), 1, room), " "), price)))
	}
	if len(texts) == 0 {
		sale = append(sale, fit("  (nothing)", lw))
	}
	sale = append(sale, listRows(texts, t.Cur[0], listN, lw, t.Pane == 0, st)...)
	yours := []string{st.Bold(fit(" You carry (Enter sells one)", rw))}
	texts = nil
	stacks := m.Stacks(m.Self.Inventory)
	for _, sk := range stacks {
		price := ""
		if it := m.Item(sk.Inst); it != nil {
			price = fmt.Sprintf("pays %d", it.GetSellPrice())
		}
		room := max(rw-4-width(price), 8)
		texts = append(texts, fmt.Sprintf("%-*s %s", room, strings.TrimRight(itemRowM(m, sk.Inst, sk.Count, room), " "), price))
	}
	if len(texts) == 0 {
		yours = append(yours, fit("  (nothing)", rw))
	}
	yours = append(yours, listRows(texts, t.Cur[1], listN, rw, t.Pane == 1, st)...)
	rows := cols(sale, yours, lw, rw)
	sel, price := "", ""
	switch {
	case t.Pane == 0 && t.Cur[0] < len(n.GetStock()):
		s := n.GetStock()[t.Cur[0]]
		sel, price = s.GetItem(), priceWords(m, s.GetPrice())
	case t.Pane == 1 && t.Cur[1] < len(stacks):
		sel = stacks[t.Cur[1]].Inst
		price = sellWords(m, sel)
	}
	rows = append(rows, strings.Repeat("─", w))
	return append(rows, itemDetail(m, sel, w, price)...)
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
	rows = append(rows, listRows(texts, t.Cur[0], max(h-4, 1), w, true, st)...)
	if lessons := n.GetTrains(); t.Cur[0] < len(lessons) {
		// What the ability is, as the trainer describes it (D52).
		rows = append(rows, strings.Repeat("─", w))
		rows = append(rows, wrapped(lessonWords(lessons[t.Cur[0]].GetInfo()), w, " ")...)
	}
	return rows
}

func bankLines(m *model.Model, t *Traders, w, h int, st Styler) []string {
	lw := (w - 1) / 2
	rw := w - 1 - lw
	left := []string{st.Bold(fit(" You carry (Enter banks one)", lw))}
	var texts []string
	for _, sk := range m.Stacks(m.Self.Inventory) {
		texts = append(texts, itemRowM(m, sk.Inst, sk.Count, max(lw-4, 1)))
	}
	if len(texts) == 0 {
		left = append(left, fit("  (nothing)", lw))
	}
	left = append(left, listRows(texts, t.Cur[0], max(h-3, 1), lw, t.Pane == 0, st)...)
	right := []string{st.Bold(fit(fmt.Sprintf(" In the bank, %d (Enter takes one)", len(m.Self.Bank)), rw))}
	texts = nil
	for _, sk := range m.Stacks(m.Self.Bank) {
		texts = append(texts, itemRowM(m, sk.Inst, sk.Count, max(rw-4, 1)))
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
	rows = append(rows, st.Bold(fit(fmt.Sprintf(" You carry, %s (Enter puts the item in the offer)", inCrowns(m)), w)))
	var texts []string
	for _, sk := range m.Stacks(m.Self.Inventory) {
		texts = append(texts, itemRowM(m, sk.Inst, sk.Count, max(w-4, 1)))
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
		rows = append(rows, help(" {Tab} list · {Enter} add / take back · {C} crowns · {Y} confirm · {X} cancel trade", w, st))
	}
	rows = append(rows, help(" {Esc} closes the screen and leaves the trade open.", w, st))
	rows = append(rows, help(" The server refuses Rare and Named items, and ones already traded.", w, st))
	return pad(rows[:min(len(rows), h)], w, h)
}

// lessonWords is the trainer's description of an ability: its description and
// numbers, as the server wrote them.
func lessonWords(a *pb.AbilityInfo) string {
	if a.GetDescription() == "" {
		return a.GetName()
	}
	return a.GetName() + ": " + a.GetDescription()
}
