package model

import (
	"fmt"
	"strings"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// reasons are the server's reason codes in plain words. A code not listed is
// shown as it is: the server owns the vocabulary and may add to it.
var reasons = map[string]string{
	// instances and doors
	"no_such_preset":         "the zone offers no instance by that name",
	"party_too_large":        "your group is larger than this instance allows",
	"turn_based_unavailable": "turn-based instances are not available on this server",
	"not_world_zone":         "an instance can only be opened from a world zone, not from inside another instance",
	"dead":                   "you are dead",
	"instance_exists":        "your group already has an instance open",
	"instance_unavailable":   "the server could not make the instance just now",
	"not_in_party":           "that instance belongs to another group",
	"party_full":             "the instance is full",
	"no_instance":            "no instance is open at this entrance; open one first (I)",
	"instance_closing":       "the instance is closing",
	"no_such_door":           "there is no such door here",
	"out_of_reach":           "you are too far from the door; walk to it first",
	"door_unavailable":       "the door leads nowhere right now",
	"space_unavailable":      "the place behind the door is not available right now",
	// turn-based play
	"plan_required":      "this instance is turn-based: build a plan and submit it with Enter",
	"round_closed":       "that round has already run; plan for the next round",
	"not_turn_based":     "this is not a turn-based place; plans are for turn-based instances",
	"bad_plan":           "the server refused the plan as malformed",
	"stand_in_forbidden": "only your own agent may send a stand-in plan",
	"plan_replaced":      "a later plan replaced this one",
	// the rest
	"encounter_owned":  "that mob is in someone else's fight",
	"out_of_range":     "the target is out of range",
	"no_target":        "you have no target",
	"target_dead":      "the target is dead",
	"not_dead":         "you are not dead",
	"already_released": "you have already released",
	"not_participant":  "that drop belongs to a fight you were not in",
	"already_looted":   "that drop was already taken",
	"rate_limited":     "too many commands too fast",
}

// Words is the reason code in plain words followed by the code itself, as
// "words (code)"; a code with no words is returned as it is.
func Words(code string) string {
	w, ok := reasons[code]
	if !ok {
		return code
	}
	return fmt.Sprintf("%s (%s)", w, code)
}

// endedWords says what each way an instance ends means.
func endedWords(reason string) string {
	switch reason {
	case "cleared":
		return "cleared: every mob dead and the loot taken"
	case "abandoned":
		return "abandoned: nobody was inside for too long"
	case "expired":
		return "expired: it outlived its time limit"
	}
	return reason
}

// describeInstance is the log line of an instance or round event.
func (m *Model) describeInstance(e *pb.Event) (kind, text, reason string, ok bool) {
	switch b := e.GetBody().(type) {
	case *pb.Event_InstanceOpened:
		o := b.InstanceOpened
		name := o.GetPreset()
		door := ""
		if p := m.Preset(o.GetPreset()); p != nil {
			name, door = p.GetName(), p.GetEntrance()
		}
		text = fmt.Sprintf("instance opened: %s (up to %d in it)", name, o.GetPartyCap())
		if door != "" {
			text += "; enter at " + door + " with E"
		}
		return "instance_opened", text, "", true
	case *pb.Event_InstanceOpenFailed:
		o := b.InstanceOpenFailed
		text = "instance not opened: " + Words(o.GetReason())
		if o.GetInstance() != "" {
			text += ", " + o.GetInstance()
		}
		return "instance_open_failed", text, o.GetReason(), true
	case *pb.Event_InstanceEnded:
		o := b.InstanceEnded
		return "instance_ended", "instance ended, " + endedWords(o.GetReason()), o.GetReason(), true
	case *pb.Event_RoundStarted:
		o := b.RoundStarted
		who := "none yet"
		if len(o.GetSubmitted()) > 0 {
			who = m.nameList(o.GetSubmitted())
		}
		return "round_started", fmt.Sprintf("round %d is open: game time waits for plans (plans in: %s)", o.GetRound(), who), "", true
	case *pb.Event_RoundResolved:
		o := b.RoundResolved
		own := false
		for _, id := range o.GetSubmitted() {
			own = own || id == m.Self.ID
		}
		by := map[string]string{"all_submitted": "everyone had a plan", "deadline": "the deadline passed", "instance_ended": "the instance ended"}[o.GetReason()]
		if by == "" {
			by = o.GetReason()
		}
		fate := "your character played your own plan"
		if !own {
			fate = "you had no plan of your own, so your character played a stand-in plan if one was sent, else the safe default (stop, defend, start nothing)"
		}
		return "round_resolved", fmt.Sprintf("round %d ran (%s): %s", o.GetRound(), by, fate), o.GetReason(), true
	}
	return "", "", "", false
}

func (m *Model) nameList(ids []string) string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = m.label(id)
		if id == m.Self.ID {
			out[i] = "you"
		}
	}
	return strings.Join(out, ", ")
}
