package model

import (
	"fmt"
	"sort"
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

// CompanionMana is the companion's mana and its most (0, 0: it has none, or is
// away), from its group entry, else from the snapshot's companion state.
func (m *Model) CompanionMana() (cur, max int64) {
	if g := m.CompanionMember(); g != nil && g.MaxMana > 0 {
		return g.Mana, g.MaxMana
	}
	c := m.Companion()
	if c.GetPresent() {
		return c.GetMana(), c.GetMaxMana()
	}
	return 0, 0
}

// QuestStepState says where a step of the quest stands.
type QuestStepState string

const (
	StepOffered QuestStepState = "offered" // the quest giver's board offers it
	StepHeld    QuestStepState = "held"    // taken, not yet done
	StepReady   QuestStepState = "ready"   // done: hand it in at the quest giver
	StepDone    QuestStepState = "done"    // handed in
)

// QuestStep is one contract of a quest, as seen. Quest names the quest and Step
// is the step's place in it (from 1), both as the server said (D55).
type QuestStep struct {
	ID, Quest, Name, Description, Kind string
	Step                               int64
	Have, Need                         int64
	State                              QuestStepState
}

// questSeen is a step of a quest the character has been told of.
type questSeen struct {
	quest, name, desc, kind string
	step, need              int64
}

// noteQuestOffers remembers the quest steps a board offers (D55: an offer says
// which quest and step it is). The offers in view now are the steps not yet
// taken. A board that has offered a step is the quest giver.
func (m *Model) noteQuestOffers(n *pb.NpcInfo) {
	if n.GetId() == m.questGiver {
		m.questOffered = map[string]bool{}
	}
	for _, o := range n.GetOffers() {
		if o.GetQuest() == "" {
			continue
		}
		m.questGiver = n.GetId()
		if m.questOffered == nil {
			m.questOffered = map[string]bool{}
		}
		m.questOffered[o.GetId()] = true
		m.seeQuestStep(o.GetId(), questSeen{o.GetQuest(), o.GetName(), o.GetDescription(), o.GetKind(), o.GetStep(), o.GetNeed()})
	}
}

// noteQuestHeld remembers the quest steps the character holds.
func (m *Model) noteQuestHeld(cs []*pb.ContractInfo) {
	for _, c := range cs {
		if c.GetQuest() != "" {
			m.seeQuestStep(c.GetId(), questSeen{c.GetQuest(), c.GetName(), c.GetDescription(), c.GetKind(), c.GetStep(), c.GetNeed()})
		}
	}
}

func (m *Model) seeQuestStep(id string, q questSeen) {
	if m.questSteps == nil {
		m.questSteps = map[string]questSeen{}
	}
	if _, seen := m.questSteps[id]; !seen {
		m.questOrder = append(m.questOrder, id)
	}
	m.questSteps[id] = q
}

// QuestGiver is the NPC id of the board that offered a quest step, "" if none
// has been seen.
func (m *Model) QuestGiver() string { return m.questGiver }

// QuestSteps lists the steps of the quests the character has been told of, by
// step. A step the character neither holds nor is offered is done.
func (m *Model) QuestSteps() []QuestStep {
	var out []QuestStep
	for _, id := range m.questOrder {
		q := m.questSteps[id]
		s := QuestStep{ID: id, Quest: q.quest, Name: q.name, Description: q.desc, Kind: q.kind, Step: q.step, Need: q.need, State: StepDone}
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
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Quest != out[j].Quest {
			return out[i].Quest < out[j].Quest
		}
		return out[i].Step < out[j].Step
	})
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
