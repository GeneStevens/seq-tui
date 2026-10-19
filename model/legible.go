package model

import (
	"fmt"
	"regexp"
	"strings"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// What D75 and D76 add to the model: the combat state and the alert of a mob
// that began attacking, the named place, the objective hints, and the two
// moments (a level gained, a contract or quest step handed in). The server
// decides all of them; the model keeps what it said.

// Alert is the last mob that began attacking the character (D75).
type Alert struct {
	Mob, Name, Con string
	Level          int64
	Danger         bool // the band is hard or deadly
	AtMs           int64
}

// Moment is a level gained or a hand-in, with the game time it was told. Level
// is set for a level-up (and for the level-up a hand-in caused), Hand for a hand-in.
type Moment struct {
	Level *pb.LevelUp
	Hand  *pb.HandIn
	AtMs  int64
}

func (m *Model) applyLegibleEvent(e *pb.Event) {
	at := e.GetGameTimeMs()
	switch b := e.GetBody().(type) {
	case *pb.Event_CombatState:
		if c := b.CombatState; c.GetActor() == m.Self.ID {
			m.Self.InCombat, m.Self.Attackers = c.GetInCombat(), append([]string(nil), c.GetAttackers()...)
		}
	case *pb.Event_AttackBegan:
		if a := b.AttackBegan; a.GetActor() == m.Self.ID {
			m.Alert = &Alert{Mob: a.GetMob(), Name: a.GetName(), Con: a.GetCon(), Level: a.GetLevel(), Danger: a.GetDanger(), AtMs: at}
			m.Alerts++
		}
	case *pb.Event_PlaceChanged:
		if p := b.PlaceChanged; p.GetActor() == m.Self.ID {
			m.Self.Place = p.GetPlace()
		}
	case *pb.Event_ObjectiveHints:
		if h := b.ObjectiveHints; h.GetActor() == m.Self.ID {
			m.Self.Hints = h.GetHints()
		}
	case *pb.Event_LevelUp:
		if l := b.LevelUp; l.GetActor() == m.Self.ID {
			m.LevelUp, m.levelTold = &Moment{Level: l, AtMs: at}, true
		}
	case *pb.Event_HandIn:
		if h := b.HandIn; h.GetActor() == m.Self.ID {
			m.HandIn = &Moment{Hand: h, Level: h.GetLevelUp(), AtMs: at}
		}
	}
}

// UnderAttack says a mob is fighting the character.
func (m *Model) UnderAttack() bool { return m.Self.InCombat || len(m.Self.Attackers) > 0 }

// AttackerNames are the names of the mobs in a fight with the character.
func (m *Model) AttackerNames() []string {
	var out []string
	for _, id := range m.Self.Attackers {
		name := m.label(id)
		if name == id && m.Alert != nil && m.Alert.Mob == id {
			name = m.Alert.Name // the alert named it before the model had seen it
		}
		out = append(out, name)
	}
	return out
}

// NPCLabel is an NPC as a player reads it, with its title: "Gorran Stoutarm <Warrior Trainer>".
func NPCLabel(n *pb.NpcInfo) string {
	if n.GetTitle() == "" || strings.Contains(n.GetName(), n.GetTitle()) { // a name that already says it ("Warrior Trainer")
		return n.GetName()
	}
	return n.GetName() + " <" + n.GetTitle() + ">"
}

// NPCTitled is NPCLabel for the NPC with that id, else its name.
func (m *Model) NPCTitled(id string) string {
	if n := m.NPC(id); n != nil {
		return NPCLabel(n)
	}
	return m.NPCName(id)
}

// Mark kinds, from the hints.
const (
	MarkOffer      = "offer"
	MarkReady      = "ready"
	MarkInProgress = "in_progress"
)

// MarkOf is the mark an NPC or board carries (offer, ready, in progress), "" for none:
// a hand-in that is ready comes first, then something new, then work in progress.
func (m *Model) MarkOf(id string) string {
	best := ""
	rank := map[string]int{MarkInProgress: 1, MarkOffer: 2, MarkReady: 3}
	for _, h := range m.Self.Hints {
		if h.GetId() == id && rank[h.GetKind()] > rank[best] {
			best = h.GetKind()
		}
	}
	return best
}

// HintFor is the hint of that kind about the thing with that id, or nil.
func (m *Model) HintFor(kind, id string) *pb.Hint {
	for _, h := range m.Self.Hints {
		if h.GetKind() == kind && h.GetId() == id {
			return h
		}
	}
	return nil
}

// Objective is the character's current objective, nil when it has none.
func (m *Model) Objective() *pb.Hint { return m.hintOf("objective") }

// Waypoint is the next point toward the objective: the door to take when it
// lies in another zone, else the objective itself.
func (m *Model) Waypoint() *pb.Hint { return m.hintOf("waypoint") }

func (m *Model) hintOf(kind string) *pb.Hint {
	for _, h := range m.Self.Hints {
		if h.GetKind() == kind {
			return h
		}
	}
	return nil
}

// QuestMob is the hint that makes a mob count for a held contract (matched by
// the mob's name), nil for any other.
func (m *Model) QuestMob(name string) *pb.Hint {
	for _, h := range m.Self.Hints {
		if h.GetKind() == "target" && h.GetLabel() == name {
			return h
		}
	}
	return nil
}

// IsNextDoor says the door or marker with that id is the next one on the way
// to the objective.
func (m *Model) IsNextDoor(id string) bool {
	for _, h := range m.Self.Hints {
		if (h.GetKind() == "waypoint" || h.GetKind() == "objective") && h.GetId() == id && m.isDoor(id) {
			return true
		}
	}
	return false
}

func (m *Model) isDoor(id string) bool {
	for _, d := range m.Map.GetDoors() {
		if d.GetId() == id {
			return true
		}
	}
	for _, k := range m.Map.GetMarkers() {
		if k.GetId() == id && (k.GetKind() == "zone_line" || k.GetKind() == "instance_entrance") {
			return true
		}
	}
	return false
}

var progressTail = regexp.MustCompile(`(\d+) of (\d+)\s*$`)

// HintTag is the short tag Nearby puts after a thing a hint is about:
// "[step 2]" for the person the quest step names, "[2/3]" for progress,
// "[new]", "[hand in]".
func (m *Model) HintTag(h *pb.Hint) string {
	if h == nil {
		return ""
	}
	if o := m.Objective(); o != nil && o.GetId() == h.GetId() && m.Self.Quest != nil && h.GetKind() != "target" {
		return fmt.Sprintf("[step %d]", m.Self.Quest.GetStep())
	}
	if t := progressTail.FindStringSubmatch(h.GetNote()); t != nil {
		return "[" + t[1] + "/" + t[2] + "]"
	}
	switch h.GetKind() {
	case MarkOffer:
		return "[new]"
	case MarkReady:
		return "[hand in]"
	}
	return ""
}
