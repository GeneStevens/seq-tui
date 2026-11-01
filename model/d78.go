package model

import (
	"fmt"
	"strings"

	"github.com/GeneStevens/seq/agentkit"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// D78 in the model: the character sheet and the item comparison facts the server
// told, stacks of identical items, and the log line of a loot-all. The server
// computes the sheet and every verdict; this only keeps and words them.

// slotWords are the gear slots as a player says them.
var slotWords = map[string]string{
	"head": "Head", "chest": "Chest", "legs": "Legs", "hands": "Hands", "feet": "Feet",
	"ring1": "Ring 1", "ring2": "Ring 2", "ring": "Ring", "neck": "Neck", "main_hand": "Main hand", "off_hand": "Off hand",
	"none": "no gear slot", "": "no gear slot",
}

// SlotWord is a gear slot in words: "Main hand", "Ring 1".
func SlotWord(slot string) string {
	if w, ok := slotWords[slot]; ok {
		return w
	}
	return strings.ToUpper(slot[:1]) + strings.ReplaceAll(slot[1:], "_", " ")
}

// Stack is one line of a list of items: the entry that stands for it and how
// many identical copies the character holds. The server keeps one entry for each
// copy; a slotless item with a definition shares a slot among its copies (D78),
// and the screen shows them as "Wolf Pelt ×2".
type Stack struct {
	Inst  string
	Count int
}

// Stackable says an item shares a bag slot with identical copies: it has a
// definition and no gear slot.
func (m *Model) Stackable(inst string) bool {
	it := m.Item(inst)
	return it != nil && (it.GetSlot() == "" || it.GetSlot() == "none")
}

// Stacks groups a list of items (the inventory, the bank) into the lines the
// screen shows, in the order each first appears. A traded copy, which is marked,
// stacks apart from an unmarked one (D45).
func (m *Model) Stacks(list []string) []Stack {
	var out []Stack
	at := map[string]int{}
	for _, inst := range list {
		if !m.Stackable(inst) {
			out = append(out, Stack{Inst: inst, Count: 1})
			continue
		}
		if i, ok := at[inst]; ok {
			out[i].Count++
			continue
		}
		at[inst] = len(out)
		out = append(out, Stack{Inst: inst, Count: 1})
	}
	return out
}

// Fact is what the server says wearing the item would change for this
// character; nil when it said nothing (a trade good it has no definition for).
func (m *Model) Fact(inst string) *pb.ItemFact { return m.Self.Facts[ItemID(inst)] }

// Mark is the shape that says an item against what is worn in its slot: ▲
// upgrade, ▼ downgrade, = sidegrade, x not usable by this character; "" for what
// is not gear.
func (m *Model) Mark(inst string) string {
	if f := m.Fact(inst); f != nil {
		return agentkit.FactMark(f)
	}
	return ""
}

// FactWords is the comparison in words ("upgrade over an empty main hand:
// +1.5 damage/s"), "" when there is none.
func (m *Model) FactWords(inst string) string {
	f := m.Fact(inst)
	if f == nil {
		return ""
	}
	text := agentkit.FactText(f)
	// The server names the ring slot it compared with ("ring1"), and the item worn
	// there by its id; the player says "an empty ring" and the item's name.
	if f.GetAgainst() == "" {
		if slot := f.GetSlot(); slot == "ring1" || slot == "ring2" {
			text = strings.Replace(text, "an empty "+slot, "an empty ring", 1)
		}
	} else {
		text = strings.Replace(text, " over "+f.GetAgainst(), " over "+m.ItemName(f.GetAgainst()), 1)
	}
	return text
}

// NotUsable says the server told this character cannot use the item.
func (m *Model) NotUsable(inst string) bool {
	return m.Fact(inst).GetVerdict() == "not_usable"
}

// factsByItem keeps the facts the server sent by item id.
func factsByItem(fs []*pb.ItemFact) map[string]*pb.ItemFact {
	out := make(map[string]*pb.ItemFact, len(fs))
	for _, f := range fs {
		out[f.GetItem()] = f
	}
	return out
}

// applySheetEvent replaces the sheet or the facts with those the event brings
// when they are this character's (the server re-sends the whole list when it
// changes, D78).
func (m *Model) applySheetEvent(e *pb.Event) {
	switch b := e.GetBody().(type) {
	case *pb.Event_CharacterSheet:
		if b.CharacterSheet.GetActor() == m.Self.ID {
			m.Self.Sheet = b.CharacterSheet.GetSheet()
		}
	case *pb.Event_ItemFacts:
		if b.ItemFacts.GetActor() == m.Self.ID {
			m.Self.Facts = factsByItem(b.ItemFacts.GetFacts())
		}
	}
}

// Slots used in the bags: what the server counts, else one for each entry.
func (m *Model) BagSlotsUsed() int {
	if m.Self.BagSlots > 0 {
		return int(m.Self.BagSlots)
	}
	return len(m.Stacks(m.Self.Inventory))
}

// tradeTip is said once, the first time the character holds an item that can be
// traded (D78: explain "tradable once" the first time it appears).
const tradeTip = `Tip: tradable once means one trade to another player, then it is bound to them.`

// noteTradeTip logs the tip once, when the character holds a tradable item.
func (m *Model) noteTradeTip(gameMs int64) {
	if m.tipped {
		return
	}
	for _, inst := range m.Self.Inventory {
		if it := m.Item(inst); it != nil && !m.Bound(inst) {
			m.tipped = true
			m.addLog(LogEntry{GameTimeMs: gameMs, Kind: "tip", Text: tradeTip})
			return
		}
	}
}

// lootRun is the loot-all in progress: the rewards it sent one by one, in the one
// log line that lists them.
type lootRun struct {
	seq    uint64
	atMs   int64
	items  []string // names in the order taken, with their counts
	counts map[string]int
	crowns int64
}

// mergeLoot folds a reward of loot or coin this character took into the line of
// the loot-all it belongs to. It returns false when the reward is not one, or is
// the first of a run (then the caller logs the server's own line).
func (m *Model) mergeLoot(e *pb.Event, name string) bool {
	w := e.GetReward()
	if w.GetRecipient() != m.Self.ID || (w.GetKind() != "loot" && w.GetKind() != "crowns") {
		m.run = nil
		return false
	}
	add := func(r *lootRun) {
		if w.GetKind() == "crowns" {
			r.crowns += w.GetAmount()
			return
		}
		if r.counts[name] == 0 {
			r.items = append(r.items, name)
		}
		r.counts[name]++
	}
	if r := m.run; r != nil && r.atMs == e.GetGameTimeMs() {
		// The line is among the last few: a tip or a quiet event may follow it.
		for i := len(m.Log) - 1; i >= 0 && i >= len(m.Log)-8; i-- {
			if m.Log[i].Seq != r.seq || m.Log[i].Kind != "reward" {
				continue
			}
			add(r)
			m.Log[i].Text, m.Log[i].Seq = r.text(), m.evSeq
			r.seq = m.evSeq
			return true
		}
	}
	r := &lootRun{seq: m.evSeq, atMs: e.GetGameTimeMs(), counts: map[string]int{}}
	add(r)
	m.run = r
	return false
}

// text is the one line that lists what the loot-all took.
func (r *lootRun) text() string {
	var parts []string
	for _, n := range r.items {
		if c := r.counts[n]; c > 1 {
			n = fmt.Sprintf("%s ×%d", n, c)
		}
		parts = append(parts, n)
	}
	if r.crowns > 0 {
		parts = append(parts, crowns(r.crowns))
	}
	return "You took " + strings.Join(parts, ", ")
}
