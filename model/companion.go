package model

import (
	"fmt"
	"strings"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The house companion and the Watch Captain's quest (SEQ-049, D35, D43, D46).
// As everywhere in this package the model keeps what the server said: the
// snapshot's companion state, the companion as a member of the group (it is
// the entity that carries the character as its owner), and the contracts the
// quest giver offers. It decides no rule.

// Companion is the character's house companion state from the last snapshot,
// nil until the quest has been begun.
func (m *Model) Companion() *pb.CompanionState { return m.Self.Companion }

// CompanionMember is the house companion in the group: the member that carries
// this character as its owner. Nil when it is away.
func (m *Model) CompanionMember() *Member {
	for _, id := range m.GroupIDs() {
		if g := m.Group[id]; g.Entity.GetOwner() == m.Self.ID && m.Self.ID != "" {
			return g
		}
	}
	return nil
}

// QuestStepState says where a step of the quest stands.
type QuestStepState string

const (
	StepOffered QuestStepState = "offered" // the quest giver's board offers it
	StepHeld    QuestStepState = "held"    // taken, not yet done
	StepReady   QuestStepState = "ready"   // done: hand it in at the quest giver
	StepDone    QuestStepState = "done"    // handed in
)

// QuestStep is one contract of a quest giver's chain, as seen.
type QuestStep struct {
	ID, Name, Description, Kind string
	Have, Need                  int64
	State                       QuestStepState
}

// noteQuestGiver remembers what a quest giver offers. The protocol marks the
// quest giver by its role and no contract as a quest step, so a step is a
// contract that a quest giver's board offered; the board offers a step only
// once the one before is handed in, so the offers in view now are the steps not
// yet taken.
func (m *Model) noteQuestGiver(n *pb.NpcInfo) {
	if n.GetRole() != "quest_giver" {
		return
	}
	m.questGiver = n.GetId()
	m.questOffered = map[string]bool{}
	for _, o := range n.GetOffers() {
		m.questOffered[o.GetId()] = true
		if _, seen := m.questSteps[o.GetId()]; !seen {
			if m.questSteps == nil {
				m.questSteps = map[string]*pb.ContractOffer{}
			}
			m.questSteps[o.GetId()] = o
			m.questOrder = append(m.questOrder, o.GetId())
		}
	}
}

// QuestGiver is the NPC id of the quest giver the character has seen, "" if none.
func (m *Model) QuestGiver() string { return m.questGiver }

// QuestSteps lists the steps of the companion quest the character has seen, in
// the order the quest giver offered them.
func (m *Model) QuestSteps() []QuestStep {
	var out []QuestStep
	for _, id := range m.questOrder {
		o := m.questSteps[id]
		s := QuestStep{ID: id, Name: o.GetName(), Description: o.GetDescription(), Kind: o.GetKind(), Need: o.GetNeed(), State: StepDone}
		switch c := m.Contract(id); {
		case c != nil:
			s.Name, s.Description, s.Kind, s.Have, s.Need, s.State = c.GetName(), c.GetDescription(), c.GetKind(), c.GetHave(), c.GetNeed(), StepHeld
			if c.GetReady() {
				s.State = StepReady
			}
		case m.questOffered[id]:
			s.State = StepOffered
		}
		out = append(out, s)
	}
	return out
}

// QuestShown says whether the Quest panel has something to tell: a step seen
// and the recruit not yet chosen, or a companion unlocked and its role not yet
// chosen.
func (m *Model) QuestShown() bool {
	c := m.Companion()
	if c.GetUnlocked() && c.GetRole() != "" {
		return false
	}
	return len(m.questOrder) > 0 || c != nil
}

// describeCompanion is the log line of a companion event.
func (m *Model) describeCompanion(e *pb.Event) (kind, text, reason string, ok bool) {
	b, is := e.GetBody().(*pb.Event_Companion)
	if !is {
		return
	}
	o := b.Companion
	who := m.label(o.GetOwner())
	if o.GetOwner() == m.Self.ID {
		who = "your"
	} else {
		who += "'s"
	}
	name := o.GetText()
	if n := m.label(o.GetCompanion()); o.GetCompanion() != "" && n != o.GetCompanion() {
		name = n
	}
	switch o.GetReason() {
	case "unlocked":
		return "companion", who + " quest is done: choose your recruit's role at the Watch Captain", "", true
	case "role":
		return "companion", fmt.Sprintf("%s recruit is now a %s", who, o.GetText()), "", true
	case "summoned":
		return "companion", fmt.Sprintf("%s companion %s came to its side", who, name), "", true
	case "back":
		return "companion", fmt.Sprintf("%s companion %s is back", who, name), "", true
	case "dismissed":
		return "companion", who + " companion was sent away", "", true
	case "away":
		return "companion", fmt.Sprintf("%s companion %s went with its owner's connection", who, name), "", true
	case "stance":
		return "companion", fmt.Sprintf("%s companion is now on %s", who, o.GetText()), "", true
	case "order":
		return "companion", fmt.Sprintf("%s companion is ordered: %s", who, o.GetText()), "", true
	case "level":
		return "companion", fmt.Sprintf("%s companion %s is now level %d", who, name, o.GetAmount()), "", true
	}
	return "companion", strings.TrimSpace(fmt.Sprintf("%s companion: %s", who, o.GetReason())), "", true
}

// HelperAction says in words what a companion or a duo member is doing now: its
// order if it has one, else the target the group says it has, else what its
// stance does.
func HelperAction(m *Model, g *Member, order, stance string) string {
	switch order {
	case "attack":
		return "ordered to attack"
	case "move":
		return "ordered to move"
	case "stop":
		return "ordered to stand"
	case "ability":
		return "using an ability"
	}
	if g != nil && g.Target != "" {
		name := g.Target
		if e := m.Entities[g.Target]; e != nil {
			name = e.GetName()
		}
		return "fighting " + name
	}
	switch stance {
	case "passive":
		return "standing"
	case "heal":
		return "watching for the hurt"
	case "assist", "guard":
		return "waiting for a fight"
	case "follow":
		return "following you"
	}
	return "idle"
}

// NameOf is the name the model has seen for an entity id, "" if none.
func (m *Model) NameOf(id string) string { return m.names[id] }

// InView is the entity with that id as the model has it, whether it is in the
// group or only in view (a character of the account that joined after the
// snapshot is in view until the next one lists it with the group), or nil.
func (m *Model) InView(id string) *Member {
	if g := m.Group[id]; g != nil {
		return g
	}
	if e := m.Entities[id]; e != nil {
		return &Member{Entity: e}
	}
	return nil
}
