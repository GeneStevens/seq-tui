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
	Actor      string // the entity id that acted (combat attacker, denied actor), if any
	Target     string // the entity id acted on, if any
	// Quiet marks routine lines (regeneration, movement, ticks, and the moments
	// the screen shows in their own place): the short view of the log hides them.
	Quiet bool
}

func (m *Model) logEvent(e *pb.Event) {
	m.mobNotes(e) // what a mob did, in words, ahead of the event's own line
	kind, text, reason := describe(e, m.label, m.abilityOf)
	mob := false // the line is the client's own: it says more than the server's words
	if k, t, r, ok := m.describeInstance(e); ok {
		kind, text, reason, mob = k, t, r, true // who played what in a round, who is in the party
	}
	if k, t, r, ok := describeCombat(e, m.label, m.abilityOf); ok {
		kind, text, reason = k, t, r
	}
	if k, t, r, ok := m.describeEconomy(e); ok {
		kind, text, reason = k, t, r
	}
	if k, t, r, ok := m.describeContracts(e); ok {
		kind, text, reason = k, t, r
	}
	if k, t, r, ok := m.describeQuest(e); ok {
		kind, text, reason = k, t, r
	}
	if k, t, r, ok := m.describeCompanion(e); ok {
		kind, text, reason = k, t, r
	}
	if k, t, r, ok := m.describeMobs(e); ok {
		kind, text, reason, mob = k, t, r, true
	}
	// The server's own words (D76) say what happened to this character with names
	// only; the lines above are the fallback for an event it left without, and
	// for what a mob did, which only the client works out (D70).
	if t := e.GetText(); t != "" && !mob {
		text = t
	}
	quiet := false
	switch b := e.GetBody().(type) {
	case *pb.Event_LootDropped:
		if d := b.LootDropped; e.GetText() != "" && d.GetItem() != "" {
			text += m.itemTag(d.GetItem()) // the tier and binding, which the server's words leave out
			if !d.GetCanLoot() {
				text += ": not yours to take"
			}
		}
	case *pb.Event_Combat:
		quiet = b.Combat.GetKind() == pb.CombatKind_COMBAT_KIND_HEAL && b.Combat.GetAttacker() == "" // regeneration
	case *pb.Event_CombatState:
		kind = "combat_state"
	case *pb.Event_AttackBegan:
		kind = "attack_began"
	case *pb.Event_PlaceChanged:
		kind, quiet = "place_changed", true
	case *pb.Event_ObjectiveHints:
		kind, quiet = "objective_hints", true
	case *pb.Event_LevelUp:
		kind = "level_up"
	case *pb.Event_HandIn:
		kind, quiet = "hand_in", true // the board panel shows it, and the hand-in's reward line is in the log
	}
	if kind == "npc_appeared" || kind == "npc_left" {
		quiet = true // guards and townsfolk coming and going: the short log keeps the mobs
	}
	if (kind == "unknown" && e.GetText() == "") || kind == "entity_moved" || kind == "rules_tick" || kind == "resource_changed" {
		quiet = true
	}
	var actor, target string
	switch b := e.GetBody().(type) {
	case *pb.Event_Combat:
		actor, target = b.Combat.GetAttacker(), b.Combat.GetTarget()
	case *pb.Event_Denied:
		actor, target = b.Denied.GetActor(), b.Denied.GetTarget()
	case *pb.Event_AbilityUsed:
		actor, target = b.AbilityUsed.GetActor(), b.AbilityUsed.GetTarget()
	case *pb.Event_CastCancelled:
		actor, target = b.CastCancelled.GetActor(), b.CastCancelled.GetTarget()
	case *pb.Event_CastInterrupted:
		actor, target = b.CastInterrupted.GetActor(), b.CastInterrupted.GetTarget()
	}
	if w := e.GetReward(); w != nil && (w.GetKind() == "loot" || w.GetKind() == "crowns") {
		name := m.ItemName(w.GetItem())
		if m.mergeLoot(e, name) {
			return // one line lists everything a loot-all took (D78)
		}
	}
	m.addLog(LogEntry{Seq: e.GetSeq(), GameTimeMs: e.GetGameTimeMs(), Kind: kind, Text: text, Reason: reason, Actor: actor, Target: target, Quiet: quiet})
}

// label is what an entity is called in the log: its name if the model has seen
// it (even if it has since left view), else its id.
func (m *Model) label(id string) string {
	if n := m.names[id]; n != "" {
		return n
	}
	return id
}

func describe(e *pb.Event, label, ability func(string) string) (kind, text, reason string) {
	switch b := e.GetBody().(type) {
	case *pb.Event_EntityAppeared:
		en := b.EntityAppeared.GetEntity()
		return "entity_appeared", fmt.Sprintf("%s appeared", en.GetName()), ""
	case *pb.Event_EntityMoved:
		return "entity_moved", fmt.Sprintf("%s moved to (%d, %d)", label(b.EntityMoved.GetId()), b.EntityMoved.GetPos().GetX(), b.EntityMoved.GetPos().GetY()), ""
	case *pb.Event_EntityLeft:
		return "entity_left", label(b.EntityLeft.GetId()) + " left view", ""
	case *pb.Event_Combat:
		c := b.Combat
		switch c.GetKind() {
		case pb.CombatKind_COMBAT_KIND_HIT:
			if c.GetAbility() != "" {
				return "combat_hit", fmt.Sprintf("%s's %s hit %s for %d", label(c.GetAttacker()), ability(c.GetAbility()), label(c.GetTarget()), c.GetAmount()), ""
			}
			return "combat_hit", fmt.Sprintf("%s hit %s for %d", label(c.GetAttacker()), label(c.GetTarget()), c.GetAmount()), ""
		case pb.CombatKind_COMBAT_KIND_MISS:
			return "combat_miss", fmt.Sprintf("%s missed %s", label(c.GetAttacker()), label(c.GetTarget())), ""
		case pb.CombatKind_COMBAT_KIND_DEATH:
			return "combat_death", fmt.Sprintf("%s killed %s", label(c.GetAttacker()), label(c.GetTarget())), ""
		case pb.CombatKind_COMBAT_KIND_HEAL:
			if c.GetAbility() != "" {
				return "combat_heal", fmt.Sprintf("%s's %s healed %s for %d", label(c.GetAttacker()), ability(c.GetAbility()), label(c.GetTarget()), c.GetAmount()), ""
			}
			return "combat_heal", fmt.Sprintf("%s healed %d", label(c.GetTarget()), c.GetAmount()), ""
		case pb.CombatKind_COMBAT_KIND_ABSORB:
			return "combat_absorb", fmt.Sprintf("a shield on %s took %d", label(c.GetTarget()), c.GetAmount()), ""
		}
		return "combat", "combat", ""
	case *pb.Event_EncounterOpened:
		o := b.EncounterOpened
		return "encounter_opened", fmt.Sprintf("%s began fighting %s", label(o.GetOwner()), label(o.GetMob())), ""
	case *pb.Event_EncounterClosed:
		o := b.EncounterClosed
		return "encounter_closed", fmt.Sprintf("the fight with %s is over: %s", label(o.GetMob()), reasonWords(o.GetReason())), o.GetReason()
	case *pb.Event_Denied:
		d := b.Denied
		return "denied", fmt.Sprintf("%s was refused %s on %s: %s", label(d.GetActor()), d.GetIntent(), label(d.GetTarget()), reasonWords(d.GetReason())), d.GetReason()
	case *pb.Event_Reward:
		w := b.Reward
		if w.GetKind() == "loot" {
			return "reward", fmt.Sprintf("%s received %s", label(w.GetRecipient()), w.GetItem()), ""
		}
		text = fmt.Sprintf("%s received %d %s from %s", label(w.GetRecipient()), w.GetAmount(), w.GetKind(), label(w.GetSource()))
		if w.GetLevel() > 0 {
			text += fmt.Sprintf(" and reached level %d", w.GetLevel())
		}
		return "reward", text, ""
	case *pb.Event_RulesTick:
		return "rules_tick", "rules tick", ""
	case *pb.Event_SpaceChanged:
		return "space_changed", "entered " + b.SpaceChanged.GetSpace(), ""
	case *pb.Event_TargetChanged:
		t := b.TargetChanged
		if t.GetTarget() == "" {
			return "target_changed", label(t.GetActor()) + " cleared the target", ""
		}
		return "target_changed", fmt.Sprintf("%s targeted %s", label(t.GetActor()), label(t.GetTarget())), ""
	case *pb.Event_TargetCleared:
		t := b.TargetCleared
		return "target_cleared", fmt.Sprintf("%s's target %s cleared: %s", label(t.GetActor()), label(t.GetTarget()), t.GetReason()), t.GetReason()
	case *pb.Event_AutoAttackChanged:
		a := b.AutoAttackChanged
		if a.GetOn() {
			return "auto_attack_changed", fmt.Sprintf("%s began attacking %s", label(a.GetActor()), label(a.GetTarget())), ""
		}
		return "auto_attack_changed", label(a.GetActor()) + " stopped attacking", ""
	case *pb.Event_LootDropped:
		d := b.LootDropped
		return "loot_dropped", fmt.Sprintf("%s dropped %s", label(d.GetMob()), d.GetItem()), ""
	case *pb.Event_LootGone:
		g := b.LootGone
		return "loot_gone", fmt.Sprintf("a drop is gone: %s", reasonWords(g.GetReason())), g.GetReason()
	case *pb.Event_Released:
		r := b.Released
		return "released", fmt.Sprintf("%s released; respawns in %d s", label(r.GetActor()), (r.GetRespawnInMs()+999)/1000), ""
	case *pb.Event_Respawned:
		r := b.Respawned
		return "respawned", fmt.Sprintf("%s respawned at (%d, %d) with %d hp", label(r.GetActor()), r.GetPos().GetX(), r.GetPos().GetY(), r.GetHp()), ""
	case *pb.Event_MobReset:
		return "mob_reset", fmt.Sprintf("%s reset to %d hp", label(b.MobReset.GetId()), b.MobReset.GetHp()), ""
	}
	return "unknown", "an event this client does not know", "" // additive protocol: ignore what we do not know
}

// reasonWords is a reason code in plain words, else the code.
func reasonWords(code string) string {
	if w, ok := reasons[code]; ok {
		return w
	}
	return code
}
