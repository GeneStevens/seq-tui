package model

import (
	"fmt"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The cast cue (presentation A3): what someone else is casting, on whom, how
// long is left, how far its reach goes, and how it ended. The server sends a
// cast's start with its length and nothing else: no deadline, no reach, and no
// message when it lands, so the cue is worked out from the events.
//
//	landed       an ability_used from the caster with that ability (a combat
//	             miss with it at the same time turns that into "missed")
//	interrupted  cast_interrupted
//	stunned      cast_cancelled, reason "stunned"
//	out of reach cast_cancelled, reason "out_of_range" (at the deadline only)
//
// Every ending is read from the event's structured reason, never from the
// words the server wrote, which change.

// CastGraceMs is how long past its deadline a cast stays on the cue while the
// event that ends it is on its way: a landed cast's ability_used and an
// out-of-reach cancel both arrive at the deadline, a little behind the local
// clock. After it the cue goes, ending event or not.
const CastGraceMs = 600

// CastEndShownMs is how long an ending is shown before the panel goes back to
// the plain target.
const CastEndShownMs = 4000

// How a cast ended (CastEndInfo.Kind).
const (
	EndLanded     = "landed"
	EndMissed     = "missed"
	EndInterrupt  = "interrupted"
	EndStunned    = "stunned"
	EndOutOfReach = "out of reach"
	EndCancelled  = "cancelled" // another reason: Reason says which
)

// clientReachMm is how far a cast's blow reaches, in millimetres, for the casts
// the cue explains. It is a CLIENT-SIDE fact: the server sends no reach with a
// cast, and the client keeps its own figure, shown as such. The test
// TestClientReachMatchesTheWorldPack (scenario) holds each figure to the
// range_mm the pinned seq-world pack gives the ability, so it cannot drift
// unseen.
var clientReachMm = map[string]int64{
	"chief-brow-breaker": 2500,
}

// ClientReachMm is the client-side reach of an ability's cast, 0 when the client
// does not know one.
func ClientReachMm(ability string) int64 { return clientReachMm[ability] }

// ClientReachAbilities are the abilities the client keeps a reach for.
func ClientReachAbilities() []string {
	out := make([]string, 0, len(clientReachMm))
	for id := range clientReachMm {
		out = append(out, id)
	}
	return out
}

// OtherCast is a cast by someone other than the character: a mob's, or another
// player's. EndsAtMs is game time, started + length, so a start that arrives late
// is shorter and never restarts.
type OtherCast struct {
	Actor, Target, Ability string
	EndsAtMs, CastMs       int64
	// LandsInRound is the round of a turn-based instance the cast lands in, when
	// it is a later one (0: it counts down in game time).
	LandsInRound uint64
}

// Left is how much of the cast is left at game time now, never negative.
func (c *OtherCast) Left(now int64) int64 { return max(c.EndsAtMs-now, 0) }

// Live says the cast still counts at game time now: before its deadline, or
// within the grace past it for its ending to arrive.
func (c *OtherCast) Live(now int64) bool {
	return c.LandsInRound > 0 || now < c.EndsAtMs+CastGraceMs
}

// CastEndInfo is how a cast by someone else ended.
type CastEndInfo struct {
	Actor, Target, Ability string
	Kind                   string // one of the End constants
	Reason                 string // the server's reason for a cancel or interrupt
	AtMs                   int64
}

// Shown says the ending is still on the panel at game time now.
func (e *CastEndInfo) Shown(now int64) bool { return now-e.AtMs <= CastEndShownMs }

// CastOf is the cast the actor has in progress, nil when none; a cast is dropped
// once it is past its deadline and grace, by the next event.
func (m *Model) CastOf(actor string) *OtherCast { return m.casts[actor] }

// CastEndOf is how the actor's last cast ended, nil when none (or when it began
// another since).
func (m *Model) CastEndOf(actor string) *CastEndInfo { return m.castEnds[actor] }

// CastName is a cast's move for display: the ability's name as the character's
// kit gives it, else a mob's, made from its id without the owner's prefix.
func (m *Model) CastName(ability string) string { return m.abilityOf(ability) }

// CastTargetLabel is who a cast is on: "you" for the character, else the name.
func (m *Model) CastTargetLabel(target string) string {
	if target == m.Self.ID {
		return "you"
	}
	return m.label(target)
}

// CastReach is what the cue says of a cast's reach: the client-side reach in
// millimetres (0: none known), the distance between the caster and its target,
// and whether that distance is known (both are in view).
func (m *Model) CastReach(c *OtherCast) (reachMm, distMm int64, known bool) {
	reachMm = ClientReachMm(c.Ability)
	if reachMm == 0 {
		return 0, 0, false
	}
	a, b := m.posOf(c.Actor), m.posOf(c.Target)
	if a == nil || b == nil {
		return reachMm, 0, false
	}
	return reachMm, dist(a, b), true
}

// noteCasts keeps the casts of others in step with the events.
func (m *Model) noteCasts(e *pb.Event) {
	me := m.Self.ID
	at := e.GetGameTimeMs()
	m.expireCasts()
	switch b := e.GetBody().(type) {
	case *pb.Event_CastStarted:
		o := b.CastStarted
		if o.GetActor() == me || o.GetActor() == "" {
			return
		}
		delete(m.castEnds, o.GetActor())
		delete(m.casts, o.GetActor())
		c := &OtherCast{Actor: o.GetActor(), Target: o.GetTarget(), Ability: o.GetAbility(), EndsAtMs: at + o.GetCastMs(), CastMs: o.GetCastMs(), LandsInRound: o.GetLandsInRound()}
		// A start that comes after its own deadline (the game time already moved
		// past it, from a snapshot or a later event) has nothing left to show.
		if c.LandsInRound == 0 && c.EndsAtMs <= m.GameTimeMs {
			return
		}
		if m.casts == nil {
			m.casts = map[string]*OtherCast{}
		}
		m.casts[c.Actor] = c
	case *pb.Event_AbilityUsed:
		o := b.AbilityUsed
		if c := m.casts[o.GetActor()]; c != nil && c.Ability == o.GetAbility() {
			m.endCast(c, EndLanded, "", at)
		}
	case *pb.Event_Combat:
		c := b.Combat
		switch c.GetKind() {
		case pb.CombatKind_COMBAT_KIND_MISS:
			// The cast landed and the blow missed: say so.
			if end := m.castEnds[c.GetAttacker()]; end != nil && end.Kind == EndLanded && end.Ability == c.GetAbility() && end.AtMs == at {
				end.Kind = EndMissed
			}
		case pb.CombatKind_COMBAT_KIND_DEATH:
			m.forgetCasts(c.GetTarget())
		}
	case *pb.Event_CastInterrupted:
		o := b.CastInterrupted
		m.endCastOf(o, EndInterrupt, at)
	case *pb.Event_CastCancelled:
		o := b.CastCancelled
		switch o.GetReason() {
		case "stunned":
			m.endCastOf(o, EndStunned, at)
		case "out_of_range":
			m.endCastOf(o, EndOutOfReach, at)
		default:
			m.endCastOf(o, EndCancelled, at)
		}
	case *pb.Event_EntityLeft:
		m.forgetCasts(b.EntityLeft.GetId())
	case *pb.Event_MobReset:
		m.forgetCasts(b.MobReset.GetId())
	case *pb.Event_SpaceChanged:
		m.casts, m.castEnds = nil, nil
	}
}

// endCastOf ends the cast an ending event names. The ending is shown even when the
// cast's start was never seen (a login in the middle of it).
func (m *Model) endCastOf(o *pb.CastEnded, kind string, at int64) {
	if o.GetActor() == m.Self.ID || o.GetActor() == "" {
		return
	}
	c := m.casts[o.GetActor()]
	if c == nil {
		c = &OtherCast{Actor: o.GetActor(), Target: o.GetTarget(), Ability: o.GetAbility()}
	}
	if c.Ability != o.GetAbility() { // an ending for another cast than the one shown
		return
	}
	m.endCast(c, kind, o.GetReason(), at)
}

func (m *Model) endCast(c *OtherCast, kind, reason string, at int64) {
	delete(m.casts, c.Actor)
	if m.castEnds == nil {
		m.castEnds = map[string]*CastEndInfo{}
	}
	m.castEnds[c.Actor] = &CastEndInfo{Actor: c.Actor, Target: c.Target, Ability: c.Ability, Kind: kind, Reason: reason, AtMs: at}
}

// forgetCasts drops what the cue holds for someone who died, left or reset.
func (m *Model) forgetCasts(id string) {
	delete(m.casts, id)
	delete(m.castEnds, id)
}

// pruneCasts is expireCasts after a snapshot, which also drops the casts of
// anyone no longer in view: the snapshot carries no casts, so nothing else says
// they ended.
func (m *Model) pruneCasts() {
	for id := range m.casts {
		if m.entity(id) == nil {
			delete(m.casts, id)
		}
	}
	for id := range m.castEnds {
		if m.entity(id) == nil {
			delete(m.castEnds, id)
		}
	}
	m.expireCasts()
}

// expireCasts drops the casts whose deadline and grace the game time has passed,
// and the endings shown long enough, so a lost ending event does not leave a cue.
func (m *Model) expireCasts() {
	for id, c := range m.casts {
		if !c.Live(m.GameTimeMs) {
			delete(m.casts, id)
		}
	}
	for id, e := range m.castEnds {
		if m.GameTimeMs-e.AtMs > 4*CastEndShownMs {
			delete(m.castEnds, id)
		}
	}
}

// CastEndWords is the cue's line for an ending, plain words that differ from one
// ending to another and never rest on colour: what happened and what it means.
func CastEndWords(e *CastEndInfo, move string) string {
	switch e.Kind {
	case EndLanded:
		return move + " LANDED"
	case EndMissed:
		return move + " MISSED"
	case EndInterrupt:
		return move + " INTERRUPTED"
	case EndStunned:
		return move + " STUNNED: cast broken"
	case EndOutOfReach:
		return move + " OUT OF REACH: it fell short"
	}
	return fmt.Sprintf("%s CANCELLED: %s", move, castWhy(e.Reason))
}

// CastEndShort is CastEndWords for a panel with no room for the explanation.
func CastEndShort(e *CastEndInfo, move string) string {
	switch e.Kind {
	case EndStunned:
		return move + " STUNNED"
	case EndOutOfReach:
		return move + " OUT OF REACH"
	case EndCancelled:
		return move + " CANCELLED"
	}
	return CastEndWords(e, move)
}
