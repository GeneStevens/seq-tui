package model

import (
	"fmt"
	"slices"
	"strings"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Consumables, bank slots and passage (D115). The server tells a consumable's
// effect and cooldown in the item's description, but not how long a cooldown
// has left or whether a regen is running; the client keeps both from the answer
// to the use (a reward of kind "use"), the same way it keeps the quest items.

// UseConsumedMark follows a use reward's detail when the item left the bags.
const UseConsumedMark = ";consumed"

// Regen is a resting bonus a regen item gave, until EndsAtMs (game time).
type Regen struct {
	HP, Mana int64 // per second, out of combat only
	EndsAtMs int64
}

// Use is what using the item does, nil for an item that is not a consumable.
func (m *Model) Use(inst string) *pb.UseInfo {
	if it := m.Item(inst); it != nil {
		return it.GetUse()
	}
	return nil
}

// CooldownLeftMs is how long until the consumable can be used again, 0 when it
// is ready or has no cooldown.
func (m *Model) CooldownLeftMs(inst string) int64 {
	u := m.Use(inst)
	if u == nil || u.GetCooldownGroup() == "" {
		return 0
	}
	return max(m.itemReady[u.GetCooldownGroup()]-m.GameTimeMs, 0)
}

// RegenActive is the regen bonus the character has now, nil when none runs.
func (m *Model) RegenActive() *Regen {
	if r := m.regen; r != nil && r.EndsAtMs > m.GameTimeMs {
		return r
	}
	return nil
}

// UseRefusal says why the consumable cannot be used right now, from what the
// client knows; "" when it can be tried (the server decides).
func (m *Model) UseRefusal(inst string) string {
	u := m.Use(inst)
	switch {
	case u == nil:
		return "that can't be used"
	case m.Self.InCombat && u.GetInCombat() == "forbidden":
		return m.ItemName(inst) + " can't be used in a fight"
	case m.CooldownLeftMs(inst) > 0:
		return fmt.Sprintf("%s isn't ready for %ds", m.ItemName(inst), (m.CooldownLeftMs(inst)+999)/1000)
	case u.GetEffect() == "heal" && m.Self.HP >= m.Self.MaxHP:
		return "you're already at full health"
	}
	return ""
}

// HealingItem is the first carried item that heals at once and can be used
// now, else the first that heals at all; "" when none.
func (m *Model) HealingItem() string {
	first := ""
	for _, sk := range m.Stacks(m.Self.Inventory) {
		if u := m.Use(sk.Inst); u != nil && u.GetEffect() == "heal" {
			if m.UseRefusal(sk.Inst) == "" {
				return sk.Inst
			}
			if first == "" {
				first = sk.Inst
			}
		}
	}
	return first
}

// UseWords says what a consumable does, for its card: "heals 40 hit points;
// cooldown 30 s; not in a fight".
func (m *Model) UseWords(inst string) string {
	u := m.Use(inst)
	if u == nil {
		return ""
	}
	var parts []string
	switch u.GetEffect() {
	case "heal":
		parts = append(parts, fmt.Sprintf("heals %d hit points", u.GetAmount()))
	case "regen":
		w := fmt.Sprintf("%d hit points a second", u.GetAmount())
		if u.GetManaAmount() > 0 {
			w += fmt.Sprintf(" and %d mana a second", u.GetManaAmount())
		}
		parts = append(parts, fmt.Sprintf("rest bonus of %s for %d s (out of combat)", w, u.GetDurationMs()/1000))
	default:
		parts = append(parts, u.GetEffect())
	}
	if u.GetCooldownGroup() != "" && u.GetCooldownMs() > 0 {
		parts = append(parts, fmt.Sprintf("cooldown %d s", u.GetCooldownMs()/1000))
	}
	if u.GetInCombat() == "forbidden" {
		parts = append(parts, "not in a fight")
	}
	if !u.GetConsumed() {
		parts = append(parts, "kept after use")
	}
	return strings.Join(parts, "; ")
}

// NextBankTier is the tier of bank slots the banker sells next to this
// character: its index from 1, the tier, and false when it has bought all.
func (m *Model) NextBankTier(n *pb.NpcInfo) (int, *pb.BankTierInfo, bool) {
	for i, t := range n.GetBankTiers() {
		if !t.GetBought() {
			return i + 1, t, true
		}
	}
	return 0, nil, false
}

// noteUse keeps the cooldown and the regen a use of the character's own started.
func (m *Model) noteUse(e *pb.Event) {
	r, ok := e.GetBody().(*pb.Event_Reward)
	if !ok || r.Reward.GetKind() != "use" || r.Reward.GetRecipient() != m.Self.ID {
		return
	}
	w := r.Reward
	u := m.Use(w.GetItem())
	if u == nil {
		return
	}
	now := e.GetGameTimeMs()
	if g := u.GetCooldownGroup(); g != "" {
		if m.itemReady == nil {
			m.itemReady = map[string]int64{}
		}
		m.itemReady[g] = now + u.GetCooldownMs()
	}
	if u.GetEffect() == "regen" {
		m.regen = &Regen{HP: u.GetAmount(), Mana: u.GetManaAmount(), EndsAtMs: now + u.GetDurationMs()}
	}
}

// describeUse is the log line of a use, a bank expansion or a passage.
func (m *Model) describeUse(w *pb.Reward) (string, bool) {
	who := m.label(w.GetRecipient())
	if w.GetRecipient() == m.Self.ID {
		who = "you"
	}
	switch w.GetKind() {
	case "use":
		eff, consumed := strings.CutSuffix(w.GetDetail(), UseConsumedMark)
		item := m.ItemName(w.GetItem())
		tail := ""
		if consumed {
			tail = " (used up)"
		}
		switch eff {
		case "heal":
			return fmt.Sprintf("%s used %s: healed %d%s", who, item, w.GetAmount(), tail), true
		case "regen":
			return fmt.Sprintf("%s used %s: rest bonus for %d s, out of combat%s", who, item, w.GetAmount(), tail), true
		}
		return fmt.Sprintf("%s used %s%s", who, item, tail), true
	case "bank_expand":
		tier, slots, _ := strings.Cut(w.GetDetail(), ";")
		return fmt.Sprintf("%s bought bank tier %s: %s more slots for %s", who, tier, slots, crowns(w.GetAmount())), true
	case "travel":
		return fmt.Sprintf("%s paid %s for passage (%s)", who, crowns(w.GetAmount()), w.GetDetail()), true
	}
	return "", false
}

// Service is what an NPC does for the traders' window: its role, or "ferry"
// for a quest giver who also sells passage (also: [ferry] in the pack), which
// the server lists in roles.
func Service(n *pb.NpcInfo) string {
	if n.GetRole() == "quest_giver" && slices.Contains(n.GetRoles(), "ferry") {
		return "ferry"
	}
	return n.GetRole()
}
