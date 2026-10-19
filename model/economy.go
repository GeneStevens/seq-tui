package model

import (
	"fmt"
	"strings"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// GearSlots are the ten gear slots, in the order the snapshot's equipped list
// has them (doc 05 §7).
var GearSlots = []string{"head", "chest", "legs", "hands", "feet", "ring1", "ring2", "neck", "main_hand", "off_hand"}

// BoundMark is what the server puts after an item's id once it has been traded:
// the item is bound to its owner and can never be traded again (D45).
const BoundMark = "!"

// ItemID is the item's id without the mark of a traded item.
func ItemID(inst string) string { return strings.TrimSuffix(inst, BoundMark) }

// Traded says whether an item, as the inventory lists it, carries the mark of a
// trade.
func Traded(inst string) bool { return strings.HasSuffix(inst, BoundMark) }

// Item describes an item as the inventory lists it (an id, with the mark of a
// trade after it), from the catalogue the snapshots brought; nil if the server
// never described it.
func (m *Model) Item(inst string) *pb.ItemInfo { return m.Items[ItemID(inst)] }

// ItemName is the item's name, or its id if it was never described.
func (m *Model) ItemName(inst string) string {
	if it := m.Item(inst); it != nil && it.GetName() != "" {
		return it.GetName()
	}
	return ItemID(inst)
}

// itemTag says an item's tier and binding in a few words for a log line, as
// the server described them (D52): " [Rare, bound on pickup]". Empty for an
// ordinary tradable item, or one never described.
func (m *Model) itemTag(inst string) string {
	it := m.Item(inst)
	if it == nil {
		return ""
	}
	var parts []string
	if t := it.GetTier(); t != "" && t != "common" {
		parts = append(parts, strings.ToUpper(t[:1])+t[1:])
	}
	if it.GetBindOnPickup() {
		parts = append(parts, "bound on pickup")
	}
	if len(parts) == 0 {
		return ""
	}
	return " [" + strings.Join(parts, ", ") + "]"
}

// noteDescribed keeps the items an event describes (D52): a drop, a reward, the
// stock and lessons of an NPC that came into view.
func (m *Model) noteDescribed(e *pb.Event) {
	switch b := e.GetBody().(type) {
	case *pb.Event_LootDropped:
		m.noteItem(b.LootDropped.GetItemInfo())
	case *pb.Event_Reward:
		m.noteItem(b.Reward.GetItemInfo())
	case *pb.Event_NpcAppeared:
		for _, it := range b.NpcAppeared.GetItems() {
			m.noteItem(it)
		}
	}
}

// noteItem keeps an item a loot or reward event described (D52).
func (m *Model) noteItem(it *pb.ItemInfo) {
	if it.GetId() != "" {
		m.Items[it.GetId()] = it
	}
}

// Bound says whether the item cannot be traded: it is Rare or Named (bound on
// pickup), or it has already been traded once (D45, D50).
func (m *Model) Bound(inst string) bool {
	if Traded(inst) {
		return true
	}
	it := m.Item(inst)
	return it != nil && it.GetBindOnPickup()
}

// NPC is the NPC with that id: the one in the last snapshot (which the server
// sends after each talk and each trade, with the stock and the offers as they
// are then), else the one the character has seen in this space; nil if neither.
func (m *Model) NPC(id string) *pb.NpcInfo {
	for _, n := range m.NPCs {
		if n.GetId() == id {
			return n
		}
	}
	return m.Known().NPCs[id]
}

// InReach says whether the character stands close enough to deal with the NPC:
// the NPC's own reach (D52), measured from the NPC's place. Whether the server
// agrees is its own call; this only tells the screen what to grey out.
func (m *Model) InReach(n *pb.NpcInfo) bool {
	if m.Self.Pos == nil || n.GetPos() == nil || n.GetReachMm() <= 0 {
		return false
	}
	dx, dy := n.GetPos().GetX()-m.Self.Pos.GetX(), n.GetPos().GetY()-m.Self.Pos.GetY()
	r := n.GetReachMm()
	return dx*dx+dy*dy <= r*r
}

// Worn is what is worn in the gear slot at index i of GearSlots, "" if nothing.
func (m *Model) Worn(i int) string {
	if i < 0 || i >= len(m.Self.Equipped) {
		return ""
	}
	return m.Self.Equipped[i]
}

// DropName is what a drop is called: its item, or its coin.
func (m *Model) DropName(d *pb.LootDropState) string {
	if d.GetItem() == "" {
		return fmt.Sprintf("%d crowns", d.GetCrowns())
	}
	return m.ItemName(d.GetItem())
}

// crowns says an amount of the currency.
func crowns(n int64) string {
	if n == 1 {
		return "1 crown"
	}
	return fmt.Sprintf("%d crowns", n)
}

// describeEconomy is the log line of an event of the economy (SEQ-040, SEQ-042):
// loot that dropped, the rewards of NPCs and fees, and trade windows.
func (m *Model) describeEconomy(e *pb.Event) (kind, text, reason string, ok bool) {
	switch b := e.GetBody().(type) {
	case *pb.Event_LootDropped:
		d := b.LootDropped
		what := m.ItemName(d.GetItem())
		if d.GetItem() == "" {
			what = crowns(d.GetCrowns())
		}
		text = fmt.Sprintf("%s dropped %s%s", m.label(d.GetMob()), what, m.itemTag(d.GetItem()))
		if !d.GetCanLoot() {
			text += ": not yours to take"
		}
		return "loot_dropped", text, "", true
	case *pb.Event_EncounterLeft:
		o := b.EncounterLeft
		return "encounter_left", fmt.Sprintf("%s left the fight with %s (it goes on for the others)", m.label(o.GetActor()), m.label(o.GetMob())), "", true
	case *pb.Event_Reward:
		w := b.Reward
		who := m.label(w.GetRecipient())
		if w.GetRecipient() == m.Self.ID {
			who = "you"
		}
		npc := m.label(w.GetSource())
		if n := m.NPC(w.GetSource()); n != nil && n.GetName() != "" {
			npc = n.GetName()
		}
		item := m.ItemName(w.GetItem())
		switch w.GetKind() {
		case "loot":
			return "reward", fmt.Sprintf("%s received %s%s", m.label(w.GetRecipient()), item, m.itemTag(w.GetItem())), "", true
		case "crowns":
			return "reward", fmt.Sprintf("%s took %s", who, crowns(w.GetAmount())), "", true
		case "buy":
			return "reward", fmt.Sprintf("%s bought %s for %s from %s", who, item, crowns(w.GetAmount()), npc), "", true
		case "sell":
			return "reward", fmt.Sprintf("%s sold %s for %s to %s", who, item, crowns(w.GetAmount()), npc), "", true
		case "train":
			return "reward", fmt.Sprintf("%s learned %s for %s from %s", who, m.AbilityName(w.GetDetail()), crowns(w.GetAmount()), npc), "", true
		case "bind":
			return "reward", fmt.Sprintf("%s set the bind point at %s for %s", who, npc, crowns(w.GetAmount())), "", true
		case "fee":
			return "reward", fmt.Sprintf("%s paid an entry fee of %s", who, crowns(w.GetAmount())), "", true
		case "refund":
			if m.refunds >= m.openFailed { // no failed opening told: the refund was owed, and paid at this join (D55)
				return "reward", fmt.Sprintf("an owed refund arrived: the entry fee of %s for an instance that could not be opened", crowns(w.GetAmount())), "", true
			}
			return "reward", fmt.Sprintf("the entry fee of %s was refunded to %s: the instance could not be opened", crowns(w.GetAmount()), who), "", true
		case "equip":
			if w.GetDetail() == "" {
				return "reward", fmt.Sprintf("%s took off %s", who, item), "", true
			}
			return "reward", fmt.Sprintf("%s put on %s (%s)", who, item, w.GetDetail()), "", true
		case "bank":
			if w.GetDetail() == "withdraw" {
				return "reward", fmt.Sprintf("%s took %s out of the bank", who, item), "", true
			}
			return "reward", fmt.Sprintf("%s put %s in the bank", who, item), "", true
		case "trade":
			return "reward", fmt.Sprintf("the trade between %s and %s is saved", m.label(w.GetRecipient()), m.label(w.GetSource())), "", true
		}
	case *pb.Event_Trade:
		o := b.Trade
		actor, target := m.label(o.GetActor()), m.label(o.GetTarget())
		if o.GetActor() == m.Self.ID {
			actor = "you"
		}
		if o.GetTarget() == m.Self.ID {
			target = "you"
		}
		switch o.GetKind() {
		case "request":
			return "trade", fmt.Sprintf("%s asked %s for a trade", actor, target), "", true
		case "opened":
			return "trade", fmt.Sprintf("the trade window with %s is open", other(m, o)), "", true
		case "updated":
			what := m.ItemName(o.GetItem())
			if o.GetItem() == "" {
				what = crowns(o.GetAmount())
			}
			return "trade", fmt.Sprintf("%s changed the offer: %s", actor, what), "", true
		case "confirmed":
			return "trade", fmt.Sprintf("%s confirmed", actor), "", true
		case "done":
			return "trade", fmt.Sprintf("the trade with %s is done", other(m, o)), "", true
		case "closed":
			return "trade", fmt.Sprintf("the trade with %s closed: %s", other(m, o), tradeWhy(o.GetReason())), o.GetReason(), true
		}
		return "trade", fmt.Sprintf("trade %s: %s with %s", o.GetKind(), actor, target), "", true
	}
	return "", "", "", false
}

// other is the party to a trade event that is not the character.
func other(m *Model, o *pb.TradeEvent) string {
	if o.GetActor() == m.Self.ID {
		return m.label(o.GetTarget())
	}
	return m.label(o.GetActor())
}

// tradeWhy says why a trade window closed.
func tradeWhy(reason string) string {
	switch reason {
	case "cancelled":
		return "cancelled"
	case "left":
		return "someone left"
	case "dead":
		return "someone died"
	case "out_of_range":
		return "too far apart"
	}
	return reason
}

// Label is what a character or mob is called: its name if the model has seen
// it, else its id.
func (m *Model) Label(id string) string { return m.label(id) }

// Classes is the class menu the server's welcome gave (D50): every class a
// character can be made of, with its roles, a line about it and the kit it
// starts with. Empty before the welcome, or from a server that has no menu.
func (m *Model) Classes() []*pb.ClassInfo { return m.Welcome.GetClasses() }

// Class is the menu entry with that id, or nil.
func (m *Model) Class(id string) *pb.ClassInfo {
	for _, c := range m.Classes() {
		if c.GetId() == id {
			return c
		}
	}
	return nil
}
