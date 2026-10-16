package model

import (
	"fmt"
	"math"
	"slices"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Slots is how many ability slots a character has (keys 1 to 8).
const Slots = 8

// RulesTickMs is the length of a rules tick (D42). Buffs and damage or heal
// over time count these; the snapshot says how many are left, not when.
const RulesTickMs = 6000

// Cast is a cast in progress and the game time it lands.
type Cast struct {
	Ability  string
	Target   string
	Slot     int
	EndsAtMs int64
	// LandsInRound is the round a cast in a turn-based instance lands in, when
	// the server said it is a later one (0: this round, or real time).
	LandsInRound uint64
}

// Effect is an effect lying on someone: from the snapshot for the character,
// from effect_applied for anyone else in view. EndsAtMs is the game time it
// ends (0: until broken); Approx says that time is counted in rules ticks, so
// it is the latest the effect can last, not an exact time.
type Effect struct {
	Ability string
	Source  string
	Kind    string // buff, debuff, dot, hot, stun, root, snare, absorb, lifesteal, stealth
	Stat    string
	// EndsAtMs is the game time the effect ends, 0 when it lasts until broken.
	EndsAtMs int64
	Approx   bool
	// Tags say what kind of affliction it is: poison, disease or curse (D50).
	Tags []string
}

func (e Effect) same(o Effect) bool {
	return e.Ability == o.Ability && e.Kind == o.Kind && e.Stat == o.Stat && e.Source == o.Source
}

// CastEnd is how the last cast ended when it did not land: the line the cast
// bar shows until the next cast starts.
type CastEnd struct {
	Ability string
	Reason  string // "moved", "stunned", "damage", ... as the server says
	Broken  string // "cancelled" or "interrupted"
}

// AbilityByID is the ability the character knows with that id, or nil.
func (m *Model) AbilityByID(id string) *pb.AbilityInfo {
	for _, a := range m.Self.Abilities {
		if a.GetId() == id {
			return a
		}
	}
	return nil
}

// AbilityName is the ability's name, as the character's snapshot or a trainer's
// lesson gave it, or its id if the server never described it.
func (m *Model) AbilityName(id string) string {
	if a := m.AbilityByID(id); a != nil && a.GetName() != "" {
		return a.GetName()
	}
	if a := m.Lessons[id]; a.GetName() != "" {
		return a.GetName() // a trainer's lesson the character does not know yet (D52)
	}
	return id
}

// SlotOf is the slot (1 to 8) an ability is in, 0 if none.
func (m *Model) SlotOf(ability string) int {
	for i, s := range m.Self.Slots {
		if s.Ability == ability && ability != "" {
			return i + 1
		}
	}
	return 0
}

// HasKit says whether the character has abilities to show: a character with
// none (a template that predates classes) gets no ability bar.
func (m *Model) HasKit() bool { return len(m.Self.Abilities) > 0 }

// EffectsOn lists the effects the client knows of on an entity: the
// character's own from the snapshot and the events, anyone else's from the
// events since the client saw them.
func (m *Model) EffectsOn(id string) []Effect {
	if id == m.Self.ID {
		return m.Self.Effects
	}
	return m.TargetEffects[id]
}

// Gauge is what the ability bar says of one slot, a hint for the player: the
// server decides when the ability is used, and a key always sends the intent.
type Gauge int

const (
	SlotEmpty Gauge = iota
	SlotReady
	SlotCasting  // another cast is in progress
	SlotCooldown // the ability's own cooldown runs
	SlotGCD      // the global cooldown runs
	SlotNoMana
	SlotNoTarget // an enemy ability and nothing targeted
	SlotRange    // the target is farther than the ability reaches
)

// Slot reports the state of slot n (1 to 8) at game time now, and the time left
// when it waits on a cooldown. The range check uses the straight distance to
// the target, as the server's does; the server decides.
func (m *Model) Slot(n int, now int64) (g Gauge, leftMs int64) {
	if n < 1 || n > Slots {
		return SlotEmpty, 0
	}
	s := m.Self.Slots[n-1]
	a := m.AbilityByID(s.Ability)
	if a == nil {
		return SlotEmpty, 0
	}
	if r := m.Self.ReadyAtMs[s.Ability]; r > now {
		return SlotCooldown, r - now
	}
	if m.Self.Cast != nil {
		return SlotCasting, max(m.Self.Cast.EndsAtMs-now, 0)
	}
	if m.Self.GCDUntilMs > now {
		return SlotGCD, m.Self.GCDUntilMs - now
	}
	if a.GetManaCost() > m.Self.Mana {
		return SlotNoMana, 0
	}
	if a.GetTarget() == "enemy" {
		t := m.Entities[m.Self.Target]
		switch {
		case t == nil || t.GetDead():
			return SlotNoTarget, 0
		case a.GetRangeMm() > 0 && dist(t.GetPos(), m.Self.Pos) > a.GetRangeMm():
			return SlotRange, 0
		}
	}
	return SlotReady, 0
}

func dist(a, b *pb.Pos) int64 {
	dx, dy := float64(a.GetX()-b.GetX()), float64(a.GetY()-b.GetY())
	return int64(math.Round(math.Hypot(dx, dy)))
}

// setKit takes the combat part of a snapshot's self.
func (s *Self) setKit(self *pb.SelfState, now int64) {
	s.Class, s.XPStart, s.XPNext = self.GetClass(), self.GetXpLevelStart(), self.GetXpNextLevel()
	s.Mana, s.MaxMana, s.GCDUntilMs, s.GCDMs = self.GetMana(), self.GetMaxMana(), self.GetGcdUntilMs(), self.GetGcdMs()
	s.Abilities = slices.Clone(self.GetAbilities())
	s.Slots = [Slots]SlotState{}
	s.ReadyAtMs = map[string]int64{}
	for _, sl := range self.GetSlots() {
		if n := int(sl.GetSlot()); n >= 1 && n <= Slots {
			s.Slots[n-1] = SlotState{Ability: sl.GetAbility()}
			s.ReadyAtMs[sl.GetAbility()] = sl.GetReadyAtMs()
		}
	}
	s.Cast = nil
	if c := self.GetCast(); c.GetAbility() != "" {
		s.Cast = &Cast{Ability: c.GetAbility(), Target: c.GetTarget(), Slot: int(c.GetSlot()), EndsAtMs: c.GetEndsAtMs(), LandsInRound: c.GetLandsInRound()}
	}
	s.Effects = effectsFrom(self.GetEffects(), now)
}

// effectsFrom reads the effects the snapshot lists on the character or on any
// entity in view (D50).
func effectsFrom(list []*pb.EffectState, now int64) []Effect {
	var out []Effect
	for _, e := range list {
		fx := Effect{Ability: e.GetAbility(), Source: e.GetSource(), Kind: e.GetKind(), Stat: e.GetStat(), EndsAtMs: e.GetExpiresAtMs(), Tags: slices.Clone(e.GetTags())}
		if e.GetTicksLeft() > 0 {
			fx.EndsAtMs, fx.Approx = now+e.GetTicksLeft()*RulesTickMs, true
		}
		out = append(out, fx)
	}
	return out
}

// levelUp is the log line of a snapshot that raised the character's level: the
// abilities the new level taught (SEQ-039).
func (m *Model) levelUp(old Self) {
	var learned []string
	for _, a := range m.Self.Abilities {
		if !slices.ContainsFunc(old.Abilities, func(o *pb.AbilityInfo) bool { return o.GetId() == a.GetId() }) {
			n := a.GetName()
			if s := m.SlotOf(a.GetId()); s > 0 {
				n = fmt.Sprintf("%s (key %d)", n, s)
			}
			learned = append(learned, n)
		}
	}
	text := fmt.Sprintf("you reached level %d", m.Self.Level)
	if len(learned) > 0 {
		text += ": new ability " + joinAnd(learned)
	}
	m.Log = append(m.Log, LogEntry{Seq: m.last, GameTimeMs: m.GameTimeMs, Kind: "level_up", Text: text, Actor: m.Self.ID})
	if over := len(m.Log) - m.logCap; over > 0 {
		m.Log = append([]LogEntry(nil), m.Log[over:]...)
	}
}

func joinAnd(s []string) string {
	switch len(s) {
	case 0:
		return ""
	case 1:
		return s[0]
	}
	out := s[0]
	for _, x := range s[1 : len(s)-1] {
		out += ", " + x
	}
	return out + " and " + s[len(s)-1]
}

// applyCombatEvent folds the combat model's events (SEQ-038) into the model; it
// reports whether e was one of them.
func (m *Model) applyCombatEvent(e *pb.Event) {
	me := m.Self.ID
	at := e.GetGameTimeMs()
	switch b := e.GetBody().(type) {
	case *pb.Event_ResourceChanged:
		if o := b.ResourceChanged; o.GetActor() == me && o.GetResource() == "mana" {
			m.Self.Mana = max(m.Self.Mana+o.GetDelta(), 0)
		}
	case *pb.Event_SlotSet:
		if o := b.SlotSet; o.GetActor() == me && o.GetSlot() >= 1 && o.GetSlot() <= Slots {
			m.Self.Slots[o.GetSlot()-1] = SlotState{Ability: o.GetAbility()}
			if m.Self.ReadyAtMs == nil {
				m.Self.ReadyAtMs = map[string]int64{}
			}
			if _, ok := m.Self.ReadyAtMs[o.GetAbility()]; !ok && o.GetAbility() != "" {
				m.Self.ReadyAtMs[o.GetAbility()] = 0
			}
		}
	case *pb.Event_CastStarted:
		if o := b.CastStarted; o.GetActor() == me {
			m.Self.GCDUntilMs = at + m.Self.GCDMs
			m.Self.Cast = &Cast{Ability: o.GetAbility(), Target: o.GetTarget(), Slot: m.SlotOf(o.GetAbility()), EndsAtMs: at + o.GetCastMs(), LandsInRound: o.GetLandsInRound()}
			m.Self.LastCastEnd = nil
		}
	case *pb.Event_CastCancelled:
		if o := b.CastCancelled; o.GetActor() == me {
			m.Self.Cast = nil
			m.Self.LastCastEnd = &CastEnd{Ability: o.GetAbility(), Reason: o.GetReason(), Broken: "cancelled"}
		}
	case *pb.Event_CastInterrupted:
		if o := b.CastInterrupted; o.GetActor() == me {
			m.Self.Cast = nil
			m.Self.LastCastEnd = &CastEnd{Ability: o.GetAbility(), Reason: o.GetReason(), Broken: "interrupted"}
		}
	case *pb.Event_AbilityUsed:
		if o := b.AbilityUsed; o.GetActor() == me {
			if m.Self.Cast == nil {
				m.Self.GCDUntilMs = at + m.Self.GCDMs // an instant ability starts it when it is used
			}
			m.Self.Cast = nil
			m.Self.LastCastEnd = nil
			if a := m.AbilityByID(o.GetAbility()); a != nil && a.GetCooldownMs() > 0 && m.Self.ReadyAtMs != nil {
				m.Self.ReadyAtMs[o.GetAbility()] = at + a.GetCooldownMs()
			}
		}
	case *pb.Event_EffectApplied:
		o := b.EffectApplied
		fx := Effect{Ability: o.GetAbility(), Source: o.GetSource(), Kind: o.GetKind(), Stat: o.GetStat(), Tags: slices.Clone(o.GetTags())}
		if o.GetDurationMs() > 0 {
			fx.EndsAtMs = at + o.GetDurationMs()
		}
		if o.GetTarget() == me {
			m.Self.Effects = withEffect(m.Self.Effects, fx)
		} else {
			if m.TargetEffects == nil {
				m.TargetEffects = map[string][]Effect{}
			}
			m.TargetEffects[o.GetTarget()] = withEffect(m.TargetEffects[o.GetTarget()], fx)
		}
	case *pb.Event_EffectExpired:
		o := b.EffectExpired
		gone := Effect{Ability: o.GetAbility(), Source: o.GetSource(), Kind: o.GetKind(), Stat: o.GetStat()}
		if o.GetTarget() == me {
			m.Self.Effects = withoutEffect(m.Self.Effects, gone)
		} else if fx := m.TargetEffects[o.GetTarget()]; fx != nil {
			if fx = withoutEffect(fx, gone); len(fx) == 0 {
				delete(m.TargetEffects, o.GetTarget())
			} else {
				m.TargetEffects[o.GetTarget()] = fx
			}
		}
	}
}

func withEffect(l []Effect, fx Effect) []Effect {
	return append(withoutEffect(l, fx), fx)
}

func withoutEffect(l []Effect, fx Effect) []Effect {
	return slices.DeleteFunc(slices.Clone(l), func(o Effect) bool { return o.same(fx) })
}

// describeCombat is the log line of an event of the combat model.
func describeCombat(e *pb.Event, label func(string) string, ability func(string) string) (kind, text, reason string, ok bool) {
	switch b := e.GetBody().(type) {
	case *pb.Event_AbilityUsed:
		o := b.AbilityUsed
		text = fmt.Sprintf("%s used %s", label(o.GetActor()), ability(o.GetAbility()))
		if o.GetTarget() != "" && o.GetTarget() != o.GetActor() {
			text += " on " + label(o.GetTarget())
		}
		return "ability_used", text, "", true
	case *pb.Event_CastStarted:
		o := b.CastStarted
		text = fmt.Sprintf("%s began casting %s (%.1f s)", label(o.GetActor()), ability(o.GetAbility()), float64(o.GetCastMs())/1000)
		if r := o.GetLandsInRound(); r > 0 {
			text += fmt.Sprintf(", lands in round %d", r)
		}
		return "cast_started", text, "", true
	case *pb.Event_CastCancelled:
		o := b.CastCancelled
		return "cast_cancelled", fmt.Sprintf("%s's %s was cancelled: %s", label(o.GetActor()), ability(o.GetAbility()), castWhy(o.GetReason())), o.GetReason(), true
	case *pb.Event_CastInterrupted:
		o := b.CastInterrupted
		return "cast_interrupted", fmt.Sprintf("%s's %s was interrupted: %s", label(o.GetActor()), ability(o.GetAbility()), castWhy(o.GetReason())), o.GetReason(), true
	case *pb.Event_EffectApplied:
		o := b.EffectApplied
		text = fmt.Sprintf("%s: %s on %s", ability(o.GetAbility()), effectWord(o.GetKind(), o.GetStat()), label(o.GetTarget()))
		if o.GetDurationMs() > 0 {
			text += fmt.Sprintf(" for %d s", (o.GetDurationMs()+999)/1000)
		}
		return "effect_applied", text, "", true
	case *pb.Event_EffectExpired:
		o := b.EffectExpired
		return "effect_expired", fmt.Sprintf("%s: %s on %s ended", ability(o.GetAbility()), effectWord(o.GetKind(), o.GetStat()), label(o.GetTarget())), "", true
	case *pb.Event_ResourceChanged:
		o := b.ResourceChanged
		return "resource_changed", fmt.Sprintf("%s %+d %s (%s)", label(o.GetActor()), o.GetDelta(), o.GetResource(), o.GetReason()), o.GetReason(), true
	case *pb.Event_SlotSet:
		o := b.SlotSet
		if o.GetAbility() == "" {
			return "slot_set", fmt.Sprintf("slot %d cleared", o.GetSlot()), "", true
		}
		return "slot_set", fmt.Sprintf("slot %d holds %s", o.GetSlot(), ability(o.GetAbility())), "", true
	}
	return "", "", "", false
}

// castWhy says why a cast did not land.
func castWhy(reason string) string {
	switch reason {
	case "moved":
		return "you moved"
	case "damage":
		return "damage broke it"
	case "interrupt":
		return "an interrupt broke it"
	case "stunned":
		return "stunned"
	case "target_gone":
		return "the target is gone"
	case "no_mana":
		return "not enough mana"
	case "out_of_range":
		return "the target is out of range"
	case "no_line_of_sight":
		return "no line of sight to the target"
	case "died":
		return "the caster died"
	}
	return reason
}

// CastWhy is castWhy for the screen.
func CastWhy(reason string) string { return castWhy(reason) }

// effectWord names an effect in a word or two: a stun, a root and a snare by
// their kind, a buff or debuff by its stat.
func effectWord(kind, stat string) string {
	switch kind {
	case "buff", "debuff":
		if stat != "" {
			return kind + " " + stat
		}
	}
	return kind
}

// EffectWord is effectWord for the screen.
func EffectWord(kind, stat string) string { return effectWord(kind, stat) }
