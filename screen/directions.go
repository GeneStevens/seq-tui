package screen

import (
	"fmt"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// D203 on the screen: the guard's list on the talk screen's side, and the
// character's destination in the You panel. The map's mark and edge marker are
// in map.go. Everything here is what the server said.

// Dirs is the Directions screen, nil when closed: the list the NPC NPC gave and
// the entry Sel the cursor is on.
type Dirs struct {
	NPC string
	Sel int
}

// directionGroups are the names of the server's groups.
var directionGroups = map[string]string{"trainers": "Trainers", "merchants": "Merchants", "services": "Services", "places": "Places"}

// DirectionEntries are the entries of the list the NPC gave, in the server's order.
func DirectionEntries(m *model.Model, npc string) []*pb.DirectionEntry {
	return m.DirectionsOf(npc).GetEntries()
}

// entryWords is an entry as the list shows it: "Gorran Stoutarm, Warrior Trainer",
// "East Gate (Thornwick Downs)".
func entryWords(e *pb.DirectionEntry) string {
	s := e.GetName()
	if e.GetTitle() != "" && !strings.Contains(s, e.GetTitle()) {
		s += ", " + e.GetTitle()
	}
	if e.GetZone() != "" {
		s += " (" + e.GetZone() + ")"
	}
	return s
}

// dirsLines is the Directions screen: the entries in the server's groups and
// order, a heading for each group, the cursor's row in reverse.
func dirsLines(m *model.Model, d *Dirs, w, h int, st Styler) []string {
	add := func(rows []string, l string) []string { return append(rows, fit(l, w)) }
	rows := []string{st.Bold(fit(" Directions from "+m.NPCTitled(d.NPC), w)), ""}
	list := m.DirectionsOf(d.NPC)
	if list == nil {
		rows = add(rows, " …")
	} else if len(list.GetEntries()) == 0 {
		rows = add(rows, " They know of nothing to point you to.")
	}
	entries := list.GetEntries()
	sel := max(min(d.Sel, len(entries)-1), 0)
	var body []string
	cursorAt := 0
	group := ""
	for i, e := range entries {
		if e.GetGroup() != group {
			group = e.GetGroup()
			name := directionGroups[group]
			if name == "" {
				name = group
			}
			if len(body) > 0 {
				body = append(body, "")
			}
			body = append(body, st.Bold(fit(" "+name, w)))
		}
		mark := "  "
		if i == sel {
			mark = "> "
			cursorAt = len(body)
		}
		line := fit("   "+mark+entryWords(e), w)
		if i == sel {
			line = st.Reverse(line)
		}
		body = append(body, line)
	}
	room := max(h-len(rows)-2, 1)
	from := min(max(cursorAt-room+1, 0), max(len(body)-room, 0))
	rows = append(rows, body[from:min(from+room, len(body))]...)
	for len(rows) < h-2 {
		rows = append(rows, strings.Repeat(" ", w))
	}
	rows = append(rows, help(" {Up} / {Down} choose · {Enter} marks it on your map · {Esc} back", w, st))
	rows = append(rows, help(" The way there is on your map and in the You panel; {Shift+X} clears it.", w, st))
	return pad(rows[:min(len(rows), h)], w, h)
}

// DestinationWords is the destination as the You panel names it, "" for none:
// "Destination: Ottilie Penn, NE 24 m", and for another zone the door that leads
// there, "Destination: Barleycross, via East Gate".
func DestinationWords(m *model.Model) string {
	d := m.Destination()
	if d == nil {
		return ""
	}
	label := d.GetLabel()
	if d.GetZone() != "" {
		if w := m.DestinationWaypoint(); w != nil && w.GetKind() == "destination_waypoint" {
			return fmt.Sprintf("Destination: %s, via %s", label, model.DoorName(w))
		}
		return fmt.Sprintf("Destination: %s, in %s", label, d.GetZone())
	}
	sp := m.Self.Pos
	if d.GetPos() == nil || sp == nil {
		return "Destination: " + label
	}
	dx, dy := d.GetPos().GetX()-sp.GetX(), d.GetPos().GetY()-sp.GetY()
	return fmt.Sprintf("Destination: %s, %s %d m", label, bearing(dx, dy), (dist(dx, dy, 0, 0)+500)/1000)
}

// withDestination adds the You panel's destination row.
func withDestination(m *model.Model, rows []string, w int, st Styler) []string {
	if s := DestinationWords(m); s != "" {
		return append(rows, st.Yellow(fit(" "+s, w)))
	}
	return rows
}
