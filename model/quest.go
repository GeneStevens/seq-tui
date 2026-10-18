package model

import (
	"fmt"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Quest chains (D62, D68) and the walk a fight stops (D66). The chain the
// character is on is the snapshot's; the quest events say in words what changed
// and the fresh snapshot after each one carries the new step.

// walkWindowMs is how recently the character's own moves must have been told
// for it to count as walking when no snapshot says where it is going.
const walkWindowMs = 1000

// Quest is the quest chain the character is on, nil when it is on none.
func (m *Model) Quest() *pb.QuestInfo { return m.Self.Quest }

// QuestProgress says where the chain stands: "First Steps, step 2 of 4".
func QuestProgress(q *pb.QuestInfo) string {
	if q.GetSteps() > 0 {
		return fmt.Sprintf("%s, step %d of %d", q.GetName(), q.GetStep(), q.GetSteps())
	}
	return q.GetName()
}

func (m *Model) describeQuest(e *pb.Event) (kind, text, reason string, ok bool) {
	q, is := e.GetBody().(*pb.Event_Quest)
	if !is {
		return "", "", "", false
	}
	b := q.Quest
	name := b.GetName()
	if name == "" {
		name = b.GetQuest()
	}
	paid := ""
	if HasReward(b.GetReward()) {
		paid = " (paid " + m.RewardWords(b.GetReward()) + ")"
	}
	switch b.GetReason() {
	case "offered":
		return "quest", fmt.Sprintf("Quest chain offered: %s. First: %s It is optional; /dismiss gives it up for good.", name, b.GetText()), "", true
	case "step":
		return "quest", fmt.Sprintf("%s: step %d done%s. Next: %s", name, b.GetStep(), paid, b.GetText()), "", true
	case "done":
		return "quest", fmt.Sprintf("%s: finished%s.", name, paid), "", true
	case "dismissed":
		return "quest", fmt.Sprintf("%s dismissed; it will not be offered again.", name), "", true
	}
	return "quest", fmt.Sprintf("%s: %s", name, b.GetReason()), b.GetReason(), true
}

// walking reports whether the character is on its way somewhere as far as the
// server has said: a snapshot's destination, or its own moves told lately.
func (m *Model) walking() bool {
	return m.Self.HasDestination || (m.lastSelfMoveMs > 0 && m.GameTimeMs-m.lastSelfMoveMs <= walkWindowMs)
}

// noteWalkStop logs that a fight stopped the character's walk (D66), after the
// line of the event that did it: the server cancels the order and says nothing
// else, so the log says it in plain words.
func (m *Model) noteWalkStop(e *pb.Event) {
	if !m.walking() {
		return
	}
	var why string
	switch b := e.GetBody().(type) {
	case *pb.Event_EncounterOpened:
		o := b.EncounterOpened
		if o.GetOwner() != m.Self.ID && m.Group[o.GetOwner()] == nil {
			return
		}
		why = m.label(o.GetMob()) + " attacked"
		if o.GetOwner() == m.Self.ID {
			why = "you are fighting " + m.label(o.GetMob())
		}
	case *pb.Event_AutoAttackChanged:
		a := b.AutoAttackChanged
		if a.GetActor() != m.Self.ID || !a.GetOn() {
			return
		}
		why = "you began attacking"
	case *pb.Event_AbilityUsed:
		a := b.AbilityUsed
		if a.GetActor() != m.Self.ID || !m.hostile(a.GetTarget()) {
			return
		}
		why = "you used an ability on " + m.label(a.GetTarget())
	default:
		return
	}
	m.Self.HasDestination, m.Self.Destination, m.lastSelfMoveMs = false, nil, 0
	m.Log = append(m.Log, LogEntry{Kind: "walk_stopped", GameTimeMs: e.GetGameTimeMs(), Seq: e.GetSeq(),
		Text: "Your walk stopped: " + why + "; a fight stops a walk. Move again to go on."})
	if over := len(m.Log) - m.logCap; over > 0 {
		m.Log = append([]LogEntry(nil), m.Log[over:]...)
	}
}

// hostile reports whether id is a mob in view.
func (m *Model) hostile(id string) bool {
	e := m.Entities[id]
	return e != nil && e.GetKind() == pb.EntityKind_ENTITY_KIND_MOB
}

// Name is what an entity is called, by its id: its name if the model has seen
// it, else the id.
func (m *Model) Name(id string) string { return m.label(id) }

// HasReward reports whether a reward pays anything.
func HasReward(r *pb.RewardInfo) bool {
	return r.GetXp() > 0 || r.GetCrowns() > 0 || r.GetItem() != ""
}
