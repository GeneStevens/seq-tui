package model

import (
	"strings"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// What D203 adds to the model: the list a guard gave (by NPC), and the
// destination the character has, which arrives as two more kinds of the objective
// hints. The server decides who answers, what is listed and when a destination
// ends; the model keeps what it said.

// DirectionsOption is the dialogue option that leads to a guard's list.
const DirectionsOption = "directions"

func (m *Model) applyDirections(d *pb.DirectionsList) {
	if d.GetActor() != m.Self.ID {
		return
	}
	if m.DirectionLists == nil {
		m.DirectionLists = map[string]*pb.DirectionsList{}
	}
	m.DirectionLists[d.GetNpc()] = d
}

// DirectionsOf is the last list the NPC gave the character, nil if it gave none.
func (m *Model) DirectionsOf(npc string) *pb.DirectionsList { return m.DirectionLists[npc] }

// GivesDirections says the NPC's last words to the character offered Directions.
func (m *Model) GivesDirections(npc string) bool {
	if d := m.Said(npc); d != nil {
		for _, o := range d.Options {
			if o.GetAction() == DirectionsOption {
				return true
			}
		}
	}
	return false
}

// Destination is the character's destination, nil when it has none. In another
// zone it is the door toward it, with the far zone's name in Zone.
func (m *Model) Destination() *pb.Hint { return m.hintOf("destination") }

// DestinationWaypoint is the point to draw: the door toward a destination in
// another zone, else the destination itself.
func (m *Model) DestinationWaypoint() *pb.Hint {
	if w := m.hintOf("destination_waypoint"); w != nil && w.GetPos() != nil {
		return w
	}
	return m.Destination()
}

// DoorName is the door a destination in another zone goes through, "East Gate"
// for the label "East Gate — to the Thornwick Downs".
func DoorName(w *pb.Hint) string {
	l := w.GetLabel()
	if i := strings.Index(l, " — "); i > 0 {
		l = l[:i]
	}
	return l
}
