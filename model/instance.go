package model

import (
	"slices"
	"strings"

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

	// Resolved is the last round seen resolved (0: none), by what, and who had
	// a plan of their own.
	Resolved          uint64
	ResolvedBy        string // "all_submitted", "deadline" or "instance_ended"
	ResolvedSubmitted []string
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

// instanceName splits the name of an instance's space, "Zone (Preset name)",
// the form the server gives it. The protocol has no field that says a space is
// an instance, so this and a round are how the model knows.
func instanceName(space string) (zone, name string, ok bool) {
	i := strings.LastIndex(space, " (")
	if i < 0 || !strings.HasSuffix(space, ")") {
		return "", "", false
	}
	return space[:i], space[i+2 : len(space)-1], true
}

// InInstance reports whether the character is inside an instance.
func (m *Model) InInstance() bool {
	_, _, ok := instanceName(m.Space)
	return ok || m.Round != nil
}

// InstanceName is the name of the instance the character is in, from its
// space ("" outside one).
func (m *Model) InstanceName() string {
	_, name, _ := instanceName(m.Space)
	return name
}

// Current is the preset of the instance the character is in, or nil when it is
// in no instance or the model never saw the menu (a fresh session that
// resumed inside one).
func (m *Model) Current() *pb.InstancePreset {
	name := m.InstanceName()
	if name == "" {
		return nil
	}
	for _, p := range m.Presets {
		if p.GetName() == name {
			return p
		}
	}
	return nil
}

// TurnBased reports whether bare intents are refused here and plans are
// played: a round has been announced, or the instance's preset says so.
func (m *Model) TurnBased() bool {
	if m.Round != nil {
		return true
	}
	p := m.Current()
	return p != nil && p.GetTimeMode() == "turn_based"
}

// repeatsRound reports whether e is a round_started for the round already open.
func (m *Model) repeatsRound(e *pb.Event) bool {
	o := e.GetRoundStarted()
	return o != nil && m.Round != nil && m.Round.Open && m.Round.Number == o.GetRound() && m.Round.DeadlineMs == o.GetDeadlineMs()
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
		r := &Round{Number: o.GetRound(), Open: true, DeadlineMs: o.GetDeadlineMs(), Submitted: slices.Clone(o.GetSubmitted())}
		if prev != nil {
			r.Resolved, r.ResolvedBy, r.ResolvedSubmitted = prev.Resolved, prev.ResolvedBy, prev.ResolvedSubmitted
		}
		m.Round = r
	case *pb.Event_RoundResolved:
		o := b.RoundResolved
		if m.Round == nil {
			m.Round = &Round{}
		}
		m.Round.Number, m.Round.Open, m.Round.DeadlineMs = o.GetRound(), false, o.GetDeadlineMs()
		m.Round.Submitted = slices.Clone(o.GetSubmitted())
		m.Round.Resolved, m.Round.ResolvedBy, m.Round.ResolvedSubmitted = o.GetRound(), o.GetReason(), slices.Clone(o.GetSubmitted())
	}
}
