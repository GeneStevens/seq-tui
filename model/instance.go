package model

import (
	"slices"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Opened is an instance the character's group opened (instance_opened).
type Opened struct {
	Space    string // the instance's space name
	Preset   string
	PartyCap int64
}

// Ended is the last instance that ended and why (instance_ended).
type Ended struct {
	Space  string
	Preset string // "" if the model never saw it opened
	Reason string // "cleared", "abandoned" or "expired"
}

// Round is a turn-based instance's round as the character was told (D30): game
// time is paused while Open, until every participant has submitted a plan of
// their own or the deadline passes. A snapshot does not carry it; the server
// repeats round_started after one.
type Round struct {
	Number     uint64
	Open       bool
	DeadlineMs int64    // a date, milliseconds since the Unix epoch
	Submitted  []string // the characters with a plan of their own, as the last round event said
	// Participants is how many characters the round waits for (0: the server
	// did not say). Until the first plan of round 1 it can rise (D49).
	Participants int64

	// Resolved is the last round seen resolved (0: none), by what, and who had
	// a plan of their own.
	Resolved          uint64
	ResolvedBy        string // "all_submitted", "deadline" or "instance_ended"
	ResolvedSubmitted []string
	// ResolvedParticipants and Plays say how many characters the last resolved
	// round had and what each played (D49), in the server's order.
	ResolvedParticipants int64
	Plays                []Play
}

// Play is whose plan a character played in a round (D49): "own_plan",
// "stand_in" or "safe_default".
type Play struct {
	Character string
	Source    string
}

// PlayedBy says what the character played in the last resolved round: "" if
// the server did not say.
func (r *Round) PlayedBy(id string) string {
	if r == nil {
		return ""
	}
	for _, p := range r.Plays {
		if p.Character == id {
			return p.Source
		}
	}
	return ""
}

// HasSubmitted says whether id has a plan of its own in the round.
func (r *Round) HasSubmitted(id string) bool { return r != nil && slices.Contains(r.Submitted, id) }

// Preset returns the menu entry named id, if the model has seen a zone map that lists it.
func (m *Model) Preset(id string) *pb.InstancePreset {
	for _, p := range m.Presets {
		if p.GetId() == id {
			return p
		}
	}
	return nil
}

// InInstance reports whether the character is inside an instance: the server
// says so in the snapshot (D49), and a round says so as well.
func (m *Model) InInstance() bool { return m.Instance != nil || m.Round != nil }

// InstanceName is the name of the instance the character is in: its preset's
// name from the zone's menu, else the preset's id ("" outside one).
func (m *Model) InstanceName() string {
	if m.Instance == nil {
		return ""
	}
	if p := m.Preset(m.Instance.GetPreset()); p != nil {
		return p.GetName()
	}
	return m.Instance.GetPreset()
}

// Current is the menu entry of the instance the character is in, or nil when it
// is in no instance or the model never saw the menu (a fresh session that
// resumed inside one). What the instance is (party cap, time mode, difficulty,
// sleep limit) is Instance, from the server.
func (m *Model) Current() *pb.InstancePreset {
	if m.Instance == nil {
		return nil
	}
	return m.Preset(m.Instance.GetPreset())
}

// TurnBased reports whether bare intents are refused here and plans are
// played: the instance says so, or a round has been announced.
func (m *Model) TurnBased() bool {
	return m.Round != nil || m.Instance.GetTimeMode() == "turn_based"
}

// repeatsRound reports whether e is a round_started for the round already open.
func (m *Model) repeatsRound(e *pb.Event) bool {
	o := e.GetRoundStarted()
	return o != nil && m.Round != nil && m.Round.Open && m.Round.Number == o.GetRound() && m.Round.DeadlineMs == o.GetDeadlineMs() && m.Round.Participants == o.GetParticipants()
}

func (m *Model) applyInstanceEvent(e *pb.Event) {
	switch b := e.GetBody().(type) {
	case *pb.Event_InstanceOpened:
		o := b.InstanceOpened
		m.Opened = &Opened{Space: o.GetInstance(), Preset: o.GetPreset(), PartyCap: o.GetPartyCap()}
		m.Ended = nil
	case *pb.Event_InstanceEnded:
		o := b.InstanceEnded
		end := &Ended{Space: o.GetInstance(), Reason: o.GetReason()}
		if m.Opened != nil && m.Opened.Space == o.GetInstance() {
			end.Preset, m.Opened = m.Opened.Preset, nil
		}
		m.Ended = end
		if m.Space == o.GetInstance() {
			m.Round = nil
		}
	case *pb.Event_RoundStarted:
		o := b.RoundStarted
		var prev *Round
		if m.Round != nil {
			prev = m.Round
		}
		r := &Round{Number: o.GetRound(), Open: true, DeadlineMs: o.GetDeadlineMs(), Submitted: slices.Clone(o.GetSubmitted()), Participants: o.GetParticipants()}
		if prev != nil {
			r.Resolved, r.ResolvedBy, r.ResolvedSubmitted = prev.Resolved, prev.ResolvedBy, prev.ResolvedSubmitted
			r.ResolvedParticipants, r.Plays = prev.ResolvedParticipants, prev.Plays
		}
		m.Round = r
	case *pb.Event_RoundResolved:
		o := b.RoundResolved
		if m.Round == nil {
			m.Round = &Round{}
		}
		m.Round.Number, m.Round.Open, m.Round.DeadlineMs = o.GetRound(), false, o.GetDeadlineMs()
		m.Round.Submitted = slices.Clone(o.GetSubmitted())
		m.Round.Participants = o.GetParticipants()
		m.Round.ResolvedParticipants, m.Round.Plays = o.GetParticipants(), nil
		for _, p := range o.GetPlays() {
			m.Round.Plays = append(m.Round.Plays, Play{Character: p.GetCharacter(), Source: p.GetSource()})
		}
		m.Round.Resolved, m.Round.ResolvedBy, m.Round.ResolvedSubmitted = o.GetRound(), o.GetReason(), slices.Clone(o.GetSubmitted())
	}
}
