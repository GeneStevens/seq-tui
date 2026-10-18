package model

import (
	"fmt"
	"strings"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// What a mob does in a fight, in plain words (D70). The server sends no event
// for it: a mob that flees is a mob that moves away while hurt, one that turns
// back is a mob that swings again, a shot from range is a hit from further than
// a blade reaches, a linked camp or a rally is an encounter that opens beside
// another. The lines are inferred from the events the server already sends, and
// say nothing the events do not show.

// shotFromMm is how far a mob's hit must come from to be called a shot: past
// any melee reach in the packs (2.5 m for a character).
const shotFromMm = 4000

// joinWithinMs is how close in game time an encounter opens to the one that
// pulled it, or to a mob's ability, to be a join.
const joinWithinMs = 100

// mobTrack is what the model remembers about mobs for these lines.
type mobTrack struct {
	fled    map[string]bool // fled once this fight (the server lets a mob flee once)
	fleeing map[string]bool // is fleeing now
	shot    map[string]bool // has been seen to shoot from range
	opened  map[string]openedAt
	used    usedAbility // the last ability a mob used
	mine    usedAbility // the last ability a character of ours used
}

type openedAt struct {
	mob string
	at  int64
}

type usedAbility struct {
	actor, ability, target string
	at                     int64
}

func (t *mobTrack) init() {
	if t.fled == nil {
		t.fled, t.fleeing, t.shot, t.opened = map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]openedAt{}
	}
}

// forget drops what is remembered about a mob that died, left or reset.
func (t *mobTrack) forget(id string) {
	delete(t.fled, id)
	delete(t.fleeing, id)
	delete(t.shot, id)
}

// MobAbilityName is a mob ability's name. The server describes only the
// abilities a character can learn, so a mob's is named from its id:
// "hexer-cinder-bolt" is "Cinder Bolt", "chief-pack-cry-rally" is "Pack Cry".
func MobAbilityName(id string) string {
	words := strings.Split(id, "-")
	if len(words) >= 3 {
		words = words[1:]
	}
	if n := len(words); n >= 2 && words[n-1] == "rally" {
		words = words[:n-1]
	}
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// abilityOf is an ability's name for the log: the character's own, else a mob's
// made from its id.
func (m *Model) abilityOf(id string) string {
	if n := m.AbilityName(id); n != id || !strings.Contains(id, "-") {
		return n
	}
	return MobAbilityName(id)
}

// partyPos is where the fight is: the owner's position, else the character's.
func (m *Model) fightPos(mob string) *pb.Pos {
	for _, enc := range m.Encounters {
		if enc.GetMob() != mob {
			continue
		}
		if enc.GetOwner() == m.Self.ID {
			return m.Self.Pos
		}
		if en := m.entity(enc.GetOwner()); en != nil {
			return en.GetPos()
		}
	}
	return nil
}

// mobNotes adds the lines about what a mob did, before the event's own line,
// and keeps what it needs to say the next. It runs once for each event, before
// the event is applied (the positions are still the old ones).
func (m *Model) mobNotes(e *pb.Event) {
	t := &m.mobs
	t.init()
	at := e.GetGameTimeMs()
	note := func(kind, text string) {
		m.addLog(LogEntry{Seq: e.GetSeq(), GameTimeMs: at, Kind: kind, Text: text})
	}
	switch b := e.GetBody().(type) {
	case *pb.Event_EntityMoved:
		mv := b.EntityMoved
		en := m.Entities[mv.GetId()]
		if en == nil || en.GetKind() != pb.EntityKind_ENTITY_KIND_MOB || en.GetDead() || t.fled[mv.GetId()] || t.shot[mv.GetId()] || en.GetHp() >= en.GetMaxHp() {
			return
		}
		fp := m.fightPos(mv.GetId())
		if fp == nil || dist(mv.GetPos(), fp) <= dist(en.GetPos(), fp) {
			return
		}
		t.fled[mv.GetId()], t.fleeing[mv.GetId()] = true, true
		note("mob_flees", m.label(mv.GetId())+" flees")
	case *pb.Event_Combat:
		c := b.Combat
		if c.GetKind() == pb.CombatKind_COMBAT_KIND_DEATH {
			t.forget(c.GetTarget())
			return
		}
		a := c.GetAttacker()
		if !m.hostile(a) {
			return
		}
		switch c.GetKind() {
		case pb.CombatKind_COMBAT_KIND_HIT, pb.CombatKind_COMBAT_KIND_MISS:
			if t.fleeing[a] {
				delete(t.fleeing, a)
				note("mob_returns", m.label(a)+" turns back to fight")
			}
			if c.GetAbility() == "" && !t.shot[a] {
				if tp := m.posOf(c.GetTarget()); tp != nil && dist(m.Entities[a].GetPos(), tp) > shotFromMm {
					t.shot[a] = true
					note("mob_shoots", m.label(a)+" shoots from range")
				}
			}
		}
	case *pb.Event_AbilityUsed:
		o := b.AbilityUsed
		u := usedAbility{o.GetActor(), o.GetAbility(), o.GetTarget(), at}
		if m.hostile(o.GetActor()) {
			t.used = u
			if t.fleeing[o.GetActor()] {
				delete(t.fleeing, o.GetActor())
				note("mob_returns", m.label(o.GetActor())+" turns back to fight")
			}
		} else {
			t.mine = u
		}
	case *pb.Event_EncounterOpened:
		o := b.EncounterOpened
		prev, linked := t.opened[o.GetOwner()]
		t.opened[o.GetOwner()] = openedAt{o.GetMob(), at}
		if u := t.used; u.actor != "" && u.actor != o.GetMob() && at-u.at <= joinWithinMs && at >= u.at {
			note("mob_joins", fmt.Sprintf("%s answers %s's %s", m.label(o.GetMob()), m.label(u.actor), m.abilityOf(u.ability)))
		} else if linked && prev.mob != o.GetMob() && at-prev.at <= joinWithinMs {
			note("mob_joins", fmt.Sprintf("%s, partner of %s, joins the fight", m.label(o.GetMob()), m.label(prev.mob)))
		}
	case *pb.Event_EntityLeft:
		t.forget(b.EntityLeft.GetId())
	case *pb.Event_MobReset:
		t.forget(b.MobReset.GetId())
	}
}

func (m *Model) posOf(id string) *pb.Pos {
	if id == m.Self.ID {
		return m.Self.Pos
	}
	if en := m.entity(id); en != nil {
		return en.GetPos()
	}
	return nil
}

// describeMobs is the line of an event that is a mob's doing, said in plain words.
func (m *Model) describeMobs(e *pb.Event) (kind, text, reason string, ok bool) {
	switch b := e.GetBody().(type) {
	case *pb.Event_CastInterrupted:
		return m.mobCastBroken("cast_interrupted", b.CastInterrupted, e.GetGameTimeMs())
	case *pb.Event_CastCancelled: // a Bash stuns the caster, and the server cancels the cast: "stunned"
		return m.mobCastBroken("cast_cancelled", b.CastCancelled, e.GetGameTimeMs())
	case *pb.Event_Combat:
		c := b.Combat
		if c.GetKind() == pb.CombatKind_COMBAT_KIND_HEAL && m.hostile(c.GetAttacker()) && m.hostile(c.GetTarget()) && c.GetAttacker() != c.GetTarget() {
			return "combat_heal", fmt.Sprintf("%s heals %s for %d", m.label(c.GetAttacker()), m.label(c.GetTarget()), c.GetAmount()), "", true
		}
	}
	return
}

// mobCastBroken is the line of a mob's cast that did not land, naming the blow
// that broke it when a character's ability was used on the mob a moment ago.
func (m *Model) mobCastBroken(kind string, o *pb.CastEnded, at int64) (string, string, string, bool) {
	if !m.hostile(o.GetActor()) {
		return "", "", "", false
	}
	cast := fmt.Sprintf("%s's %s", m.label(o.GetActor()), m.abilityOf(o.GetAbility()))
	if u := m.mobs.mine; (o.GetReason() == "interrupt" || o.GetReason() == "stunned") && u.target == o.GetActor() && at-u.at <= joinWithinMs {
		who := "your"
		if u.actor != m.Self.ID {
			who = m.label(u.actor) + "'s"
		}
		return kind, fmt.Sprintf("%s %s breaks off %s", who, m.abilityOf(u.ability), cast), o.GetReason(), true
	}
	return kind, fmt.Sprintf("%s is broken off: %s", cast, castWhy(o.GetReason())), o.GetReason(), true
}
