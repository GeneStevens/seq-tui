package model

import (
	"fmt"
	"strconv"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func itoa(n uint64) string { return strconv.FormatUint(n, 10) }

// LogEntry is one event in the log, with the reason the server gave for it
// (Pillar 8) kept as its own field.
type LogEntry struct {
	Seq        uint64 // the character's stream number; 0 for a space_changed
	GameTimeMs int64
	Kind       string // the event's kind, as the protocol names it
	Text       string // one readable line
	Reason     string // the server's reason, if the event carries one
}

func (m *Model) logEvent(e *pb.Event) {
	kind, text, reason := describe(e)
	m.Log = append(m.Log, LogEntry{Seq: e.GetSeq(), GameTimeMs: e.GetGameTimeMs(), Kind: kind, Text: text, Reason: reason})
	if over := len(m.Log) - m.logCap; over > 0 {
		m.Log = append([]LogEntry(nil), m.Log[over:]...)
	}
}

func describe(e *pb.Event) (kind, text, reason string) {
	switch b := e.GetBody().(type) {
	case *pb.Event_EntityAppeared:
		en := b.EntityAppeared.GetEntity()
		return "entity_appeared", fmt.Sprintf("%s (%s) appeared", en.GetName(), en.GetId()), ""
	case *pb.Event_EntityMoved:
		return "entity_moved", fmt.Sprintf("%s moved to (%d, %d)", b.EntityMoved.GetId(), b.EntityMoved.GetPos().GetX(), b.EntityMoved.GetPos().GetY()), ""
	case *pb.Event_EntityLeft:
		return "entity_left", b.EntityLeft.GetId() + " left view", ""
	case *pb.Event_Combat:
		c := b.Combat
		switch c.GetKind() {
		case pb.CombatKind_COMBAT_KIND_HIT:
			return "combat_hit", fmt.Sprintf("%s hit %s for %d", c.GetAttacker(), c.GetTarget(), c.GetAmount()), ""
		case pb.CombatKind_COMBAT_KIND_MISS:
			return "combat_miss", fmt.Sprintf("%s missed %s", c.GetAttacker(), c.GetTarget()), ""
		case pb.CombatKind_COMBAT_KIND_DEATH:
			return "combat_death", fmt.Sprintf("%s killed %s", c.GetAttacker(), c.GetTarget()), ""
		case pb.CombatKind_COMBAT_KIND_HEAL:
			return "combat_heal", fmt.Sprintf("%s healed %d", c.GetTarget(), c.GetAmount()), ""
		}
		return "combat", "combat", ""
	case *pb.Event_EncounterOpened:
		o := b.EncounterOpened
		return "encounter_opened", fmt.Sprintf("%s opened an encounter with %s (%s)", o.GetOwner(), o.GetMob(), o.GetId()), ""
	case *pb.Event_EncounterClosed:
		o := b.EncounterClosed
		return "encounter_closed", fmt.Sprintf("encounter %s with %s closed: %s", o.GetId(), o.GetMob(), o.GetReason()), o.GetReason()
	case *pb.Event_Denied:
		d := b.Denied
		return "denied", fmt.Sprintf("%s denied %s on %s: %s", d.GetActor(), d.GetIntent(), d.GetTarget(), d.GetReason()), d.GetReason()
	case *pb.Event_Reward:
		w := b.Reward
		if w.GetKind() == "loot" {
			return "reward", fmt.Sprintf("%s received %s from %s", w.GetRecipient(), w.GetItem(), w.GetSource()), ""
		}
		return "reward", fmt.Sprintf("%s received %d %s from %s", w.GetRecipient(), w.GetAmount(), w.GetKind(), w.GetSource()), ""
	case *pb.Event_RulesTick:
		return "rules_tick", "rules tick", ""
	case *pb.Event_SpaceChanged:
		return "space_changed", "entered " + b.SpaceChanged.GetSpace(), ""
	case *pb.Event_TargetChanged:
		t := b.TargetChanged
		if t.GetTarget() == "" {
			return "target_changed", t.GetActor() + " cleared the target", ""
		}
		return "target_changed", fmt.Sprintf("%s targeted %s", t.GetActor(), t.GetTarget()), ""
	case *pb.Event_TargetCleared:
		t := b.TargetCleared
		return "target_cleared", fmt.Sprintf("%s's target %s cleared: %s", t.GetActor(), t.GetTarget(), t.GetReason()), t.GetReason()
	case *pb.Event_AutoAttackChanged:
		a := b.AutoAttackChanged
		if a.GetOn() {
			return "auto_attack_changed", fmt.Sprintf("%s began attacking %s", a.GetActor(), a.GetTarget()), ""
		}
		return "auto_attack_changed", a.GetActor() + " stopped attacking", ""
	case *pb.Event_LootDropped:
		d := b.LootDropped
		return "loot_dropped", fmt.Sprintf("%s dropped %s (%s)", d.GetMob(), d.GetItem(), d.GetDrop()), ""
	case *pb.Event_MobReset:
		return "mob_reset", fmt.Sprintf("%s reset to %d hp", b.MobReset.GetId(), b.MobReset.GetHp()), ""
	}
	return "unknown", "an event this client does not know", "" // additive protocol: ignore what we do not know
}
