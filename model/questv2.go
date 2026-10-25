package model

import (
	"fmt"
	"strings"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Quests v2 (D105): the steps beyond talk and go (deliver, kill, collect,
// watch), steps in another zone, quest items and the offers an NPC holds back.
// The server counts and decides; this file only says it in words.

// QuestTag is the tag a quest item carries in the bags, beside its colour.
const QuestTag = "[quest]"

// IsQuestItem says whether the item, as the inventory lists it, is a quest's
// own: it cannot be sold, traded or banked, and leaves when its quest ends.
func (m *Model) IsQuestItem(inst string) bool { return m.Item(inst).GetQuestItem() }

// QuestItemRefusal is why a quest item is not offered to a vendor, a banker or
// a trade, "" for any other item.
func (m *Model) QuestItemRefusal(inst string) string {
	if !m.IsQuestItem(inst) {
		return ""
	}
	return m.ItemName(inst) + " is a quest item: it cannot be sold, traded or banked, and it goes when its quest ends"
}

// QuestProgressLine says how far the current step is: "Kills: 2 of 5", "Held:
// 2 of 3 charm beads", "Watching… 12 of 20 s". Empty for a step with no count.
func (m *Model) QuestProgressLine(q *pb.QuestInfo) string {
	if q.GetNeed() <= 0 {
		return ""
	}
	of := fmt.Sprintf("%d of %d", m.stepProgress(q), q.GetNeed())
	item := m.questItemName(q.GetItem())
	switch q.GetKind() {
	case "deliver":
		return fmt.Sprintf("Held: %s %s", of, item)
	case "collect":
		return fmt.Sprintf("Collected: %s %s", of, item)
	case "kill":
		return "Kills: " + of
	case "watch":
		return fmt.Sprintf("Watching… %d of %d s", q.GetProgress(), q.GetNeed())
	}
	return ""
}

// stepProgress is how far the step is. The server tells a quest's progress with
// the quest, when a step or the chain changes, not as a count rises; what the
// bags hold the character knows itself, so a deliver or collect step counts the
// item in the bags now. Kills and the watch are the server's last word.
func (m *Model) stepProgress(q *pb.QuestInfo) int64 {
	if (q.GetKind() == "deliver" || q.GetKind() == "collect") && q.GetItem() != "" {
		held := int64(0)
		for _, inst := range m.Self.Inventory {
			if ItemID(inst) == q.GetItem() {
				held++
			}
		}
		return min(held, q.GetNeed())
	}
	return q.GetProgress()
}

// questItemName is an item's name, or "items" for a step that names none.
func (m *Model) questItemName(id string) string {
	if id == "" {
		return "items"
	}
	return m.ItemName(id)
}

// QuestStepWords is the step in one plain sentence when the server's own text
// is empty: "Bring 3 charm beads to Barnaby Tuck".
func (m *Model) QuestStepWords(q *pb.QuestInfo) string {
	if t := q.GetText(); t != "" {
		return t
	}
	n := q.GetTargetName()
	switch q.GetKind() {
	case "deliver":
		return fmt.Sprintf("Bring %d %s to %s", q.GetNeed(), m.questItemName(q.GetItem()), n)
	case "collect":
		return fmt.Sprintf("Collect %d %s", q.GetNeed(), m.questItemName(q.GetItem()))
	case "kill":
		return fmt.Sprintf("Defeat %d", q.GetNeed())
	case "watch":
		return fmt.Sprintf("Watch from %s for %d s", n, q.GetNeed())
	}
	return n
}

// QuestAway says the current step is done in another zone than this one.
func QuestAway(q *pb.QuestInfo) bool { return q.GetZone() != "" }

// QuestDoor is the door the quest view names toward a step in another zone:
// its name, and its place on this map; "" when the zone has no door from here.
func (m *Model) QuestDoor(q *pb.QuestInfo) (string, *pb.Pos) {
	p := q.GetTargetPos()
	if !QuestAway(q) || p == nil {
		return "", nil
	}
	for _, d := range m.Map.GetDoors() {
		if d.GetPos().GetX() == p.GetX() && d.GetPos().GetY() == p.GetY() {
			if d.GetName() != "" {
				return d.GetName(), p
			}
			return m.MarkerName(d.GetId()), p
		}
	}
	for _, k := range m.Map.GetMarkers() {
		if k.GetPos().GetX() == p.GetX() && k.GetPos().GetY() == p.GetY() {
			return k.GetName(), p
		}
	}
	return "door", p
}

// WatchHint is the plain note under a watch step that has gone back to the
// start, "" while it runs or for any other step: why, and what to do.
func (m *Model) WatchHint(q *pb.QuestInfo) string {
	if q.GetKind() != "watch" {
		return ""
	}
	switch {
	case m.Self.InCombat && q.GetProgress() == 0:
		return "The watch is stopped while you are in combat. It starts again from nothing when the fight is over."
	case m.watchReset != nil && m.watchReset.quest == q.GetId() && m.watchReset.step == q.GetStep() && q.GetProgress() < m.watchReset.peak:
		return fmt.Sprintf("The watch started over at %d s: you %s. Stay within the spot, out of combat, for the whole time.", m.watchReset.peak, m.watchReset.why)
	}
	return ""
}

// watchNote remembers that a watch went back: the quest and step, how far it
// had come and why.
type watchNote struct {
	quest string
	step  int64
	peak  int64
	why   string
}

// noteQuestChange compares the quest view before and after: a watch whose
// seconds fell has been reset by leaving the spot or by a fight.
func (m *Model) noteQuestChange(old, now *pb.QuestInfo) {
	if now.GetKind() != "watch" || old.GetKind() != "watch" || old.GetId() != now.GetId() || old.GetStep() != now.GetStep() {
		m.watchReset = nil
		return
	}
	if now.GetProgress() < old.GetProgress() {
		why := "left the spot"
		if m.Self.InCombat || len(m.Self.Attackers) > 0 {
			why = "were in combat"
		}
		peak := old.GetProgress()
		if r := m.watchReset; r != nil && r.quest == now.GetId() && r.step == now.GetStep() {
			peak = max(peak, r.peak)
		}
		m.watchReset = &watchNote{quest: now.GetId(), step: now.GetStep(), peak: peak, why: why}
	}
}

// OfferLocked is why an NPC's quest offer is held back, in a few words: "needs
// level 8", "finish Lost Charms first". "" for an open offer.
func OfferLocked(o *pb.QuestOfferInfo) string {
	switch o.GetLocked() {
	case "":
		return ""
	case "level":
		return fmt.Sprintf("needs level %d", o.GetMinLevel())
	case "requires":
		if who := strings.TrimSpace(strings.TrimPrefix(o.GetText(), "Finish first:")); who != "" {
			return "finish " + who + " first"
		}
		return "finish an earlier quest first"
	case "condition":
		return "not yet: come back later"
	}
	return o.GetLocked()
}

// QuestOffers are the chains an NPC gives that the character has not taken,
// open ones first.
func QuestOffers(n *pb.NpcInfo) []*pb.QuestOfferInfo {
	var open, locked []*pb.QuestOfferInfo
	for _, o := range n.GetQuestOffers() {
		if o.GetLocked() == "" {
			open = append(open, o)
		} else {
			locked = append(locked, o)
		}
	}
	return append(open, locked...)
}

// noteQuestItemsGone logs the quest items the bags lost between two inventories
// (handed in, taken at the end of a chain, or given up): the server takes them
// and says nothing else a player can read.
func (m *Model) noteQuestItemsGone(before []string, gameMs int64) {
	now := map[string]int{}
	for _, inst := range m.Self.Inventory {
		now[inst]++
	}
	seen := map[string]bool{}
	for _, inst := range before {
		if seen[inst] || !m.IsQuestItem(inst) {
			continue
		}
		seen[inst] = true
		had := 0
		for _, b := range before {
			if b == inst {
				had++
			}
		}
		if gone := had - now[inst]; gone > 0 {
			m.addLog(LogEntry{Kind: "quest", GameTimeMs: gameMs, Seq: m.evSeq,
				Text: fmt.Sprintf("Quest item removed from your bags: %d %s.", gone, m.ItemName(inst))})
		}
	}
}
