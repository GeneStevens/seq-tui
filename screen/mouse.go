package screen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// D77, Diablo-style mouse play, on the screen: where the map was drawn (so a
// click can be turned into a place or a thing), the labels by the pointer, the
// info card of what was clicked, and the zoom. The screen only reports what it
// drew; the program decides what a click does, and the server what is allowed.

// Pt is a place on the screen, in cells from the top left.
type Pt struct{ X, Y int }

// Rect is a block of cells.
type Rect struct{ X, Y, W, H int }

// In says the cell (x, y) is inside the block.
func (r Rect) In(x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

// Tab is a tab of the character sidebar and where its name was drawn.
type Tab struct {
	Rect Rect
	Tab  int
}

// Hits is what the last drawing laid down: the map's block and the grid behind
// it, and the sidebar's tabs. The program keeps one and the drawing fills it.
type Hits struct {
	Map  Rect // the map's inside; W is 0 when the map was not drawn
	Side Rect // the sidebar's body, W is 0 when there is none
	Tabs []Tab
	// Panel is the tabs of the bottom panel, Drawer the chat drawer's block (W is 0
	// when it is not drawn).
	Panel  []Tab
	Drawer Rect
	// Log is the bottom panel, borders and all (W is 0 when it is not drawn), LogTab the tab
	// it shows and LogPage how many rows a page of it is.
	Log     Rect
	LogTab  int
	LogPage int
	// Names are the names in the chat as drawn, top to bottom (D101 §9): a click
	// on one opens that character's card, and the keyboard steps through them.
	// Card is the character card's block and CardActs where its actions were
	// drawn (Tab is the action's number); W is 0 when there is no card.
	Names    []NameHit
	Card     Rect
	CardActs []Tab
	nameN    int
	// Seen is how far the drawing read each chat tab it showed (the screen, the
	// bottom panel, the drawer); zero for what it did not show. Marks.Merge applies it.
	Seen Marks
	g    *grid
	org  Pt // where the next map is drawn
	// spare is the grid of the drawing before, for the next to draw on (T5).
	spare *grid
}

// begin readies the record for a drawing.
func (h *Hits) begin() {
	if h.g != nil {
		h.spare = h.g
	}
	h.Map, h.Side, h.Tabs, h.g = Rect{}, Rect{}, h.Tabs[:0], nil
	h.Panel, h.Drawer, h.Log = h.Panel[:0], Rect{}, Rect{}
	h.Names, h.Card, h.CardActs, h.nameN = h.Names[:0], Rect{}, h.CardActs[:0], 0
	h.Seen = Marks{}
}

// at says where the map or screen about to be drawn starts.
func (h *Hits) at(x, y int) {
	if h != nil {
		h.org = Pt{x, y}
	}
}

// shift moves where the map about to be drawn starts, dx cells to the right.
func (h *Hits) shift(dx int) {
	if h != nil {
		h.org.X += dx
	}
}

// setGrid notes the grid just drawn as the map.
func (h *Hits) setGrid(g *grid) {
	if h != nil {
		h.g = g
		h.Map = Rect{h.org.X, h.org.Y, g.cw, g.ch}
	}
}

// PickKind says what a place on the map holds.
type PickKind int

const (
	PickGround PickKind = iota // open ground: an Option-click walks there
	PickMob
	PickNPC
	PickDrop
	PickDoor
	PickPlayer
	PickSelf   // the character's own mark: hover only
	PickMarker // a named place of the zone map (a scout point, a world event's spawn or goal): an Option-click walks there
)

// Pick is what the pointer is over on the map: a thing, or the ground at Pos.
type Pick struct {
	Kind PickKind
	ID   string
	Pos  *pb.Pos // the thing's place, or the centre of the cell for ground
	Name string
}

// thing is something the map draws a glyph for, and how it is labelled.
type thing struct {
	c, r  int64
	pick  Pick
	label string // by the pointer: name and title
	dead  bool   // a corpse: it is not selected, a click goes through it
}

// things lists what the map shows that can be pointed at, in the order of
// priority a click takes them: mobs, people, then NPCs, drops and doors.
func things(g *grid, m *model.Model) []thing {
	var out []thing
	dead := false
	add := func(p Pick, label string) {
		c, r := g.cellOf(p.Pos)
		if c < 0 || r < 0 || c >= int64(g.cw) || r >= int64(g.ch) {
			return
		}
		out = append(out, thing{c: c, r: r, pick: p, label: label, dead: dead})
	}
	ent := func(e *pb.EntityState) {
		kind, label := PickPlayer, e.GetName()
		if e.GetKind() == pb.EntityKind_ENTITY_KIND_MOB {
			kind = PickMob
			label = fmt.Sprintf("%s  Lv %d", e.GetName(), e.GetLevel())
			switch {
			case e.GetDead():
				label = e.GetName() + " (dead)"
			case e.GetCon() != "":
				label += ", " + e.GetCon()
			}
			if model.IsRare(e) && !e.GetDead() {
				label += " RARE"
			}
		}
		dead = e.GetDead()
		add(Pick{Kind: kind, ID: e.GetId(), Pos: e.GetPos(), Name: e.GetName()}, label)
		dead = false
	}
	if m.Self.Pos != nil {
		add(Pick{Kind: PickSelf, ID: m.Self.ID, Pos: m.Self.Pos, Name: m.Self.Name}, SelfLabel(m))
	}
	for _, id := range m.GroupIDs() {
		ent(m.Group[id].Entity)
	}
	for _, id := range m.EntityIDs() {
		ent(m.Entities[id])
	}
	for _, n := range m.KnownNPCs() {
		add(Pick{Kind: PickNPC, ID: n.GetId(), Pos: n.GetPos(), Name: n.GetName()}, model.NPCLabel(n))
	}
	for _, id := range m.DropIDs() {
		d := m.Drops[id]
		label := m.DropName(d)
		if mk := m.Mark(d.GetItem()); mk != "" {
			label += " " + mk
		}
		if !d.GetCanLoot() {
			label += " (not yours)"
		}
		add(Pick{Kind: PickDrop, ID: id, Pos: d.GetPos(), Name: m.DropName(d)}, label)
	}
	for _, d := range m.Map.GetDoors() {
		name := d.GetLabel()
		if name == "" {
			name = d.GetName()
		}
		if name == "" {
			name = m.MarkerName(d.GetId())
		}
		label := name
		if usable, _ := m.DoorUsable(d.GetId()); !usable {
			label += " (shut)"
		}
		add(Pick{Kind: PickDoor, ID: d.GetId(), Pos: d.GetPos(), Name: name}, label)
	}
	for _, k := range m.Map.GetMarkers() {
		if label := MarkerLabel(m, k); label != "" {
			add(Pick{Kind: PickMarker, ID: k.GetId(), Pos: k.GetPos(), Name: m.MarkerName(k.GetId())}, label)
		}
	}
	return out
}

// SelfLabel is what hovering the character's own mark says: "You: Aria, level 3,
// at 12,40 m", and where it is walking to when it is.
func SelfLabel(m *model.Model) string {
	s := fmt.Sprintf("You: %s, level %d, at %d,%d m", m.Self.Name, m.Self.Level, m.Self.Pos.GetX()/1000, m.Self.Pos.GetY()/1000)
	if m.Self.HasDestination && m.Self.Destination != nil {
		s += fmt.Sprintf(", walking to %d,%d m", m.Self.Destination.GetX()/1000, m.Self.Destination.GetY()/1000)
	}
	return s
}

// MarkerLabel is what hovering a marker of the zone map says, "The Overlook: a
// scout point", "" for the kinds that are doors (the doors list names those).
func MarkerLabel(m *model.Model, k *pb.Marker) string {
	var what string
	switch k.GetKind() {
	case "scout":
		what = "a scout point"
	case "event_spawn":
		what = "raiders gather here (world event)"
	case "event_goal":
		what = "raiders head here (world event)"
	default:
		return ""
	}
	if name := m.MarkerName(k.GetId()); name != k.GetId() {
		return name + ": " + what
	}
	return strings.ToUpper(what[:1]) + what[1:]
}

// worldOf is the middle of the cell (c, r) of the grid, in the world.
func (g *grid) worldOf(c, r int64) *pb.Pos {
	return &pb.Pos{X: (c - int64(g.cw/2) + g.pcol) * g.sc.col, Y: (int64(g.ch/2) - r + g.prow) * g.sc.row}
}

// thingAt is what sits in the cell, the highest priority first.
func thingAt(ts []thing, c, r int64) (thing, bool) {
	for _, t := range ts {
		if t.c == c && t.r == r && !t.dead {
			return t, true
		}
	}
	return thing{}, false
}

// nearMiss is the mob or person within one cell of (c, r), the nearest of them
// when there are several (the first in priority order on a tie): a click that
// just misses a moving mob still counts as a click on it.
func nearMiss(ts []thing, c, r int64) (thing, bool) {
	best, bestD, found := thing{}, int64(0), false
	for _, t := range ts {
		if t.dead || (t.pick.Kind != PickMob && t.pick.Kind != PickNPC) {
			continue
		}
		dc, dr := t.c-c, t.r-r
		if dc < -1 || dc > 1 || dr < -1 || dr > 1 {
			continue
		}
		if d := dc*dc + dr*dr; !found || d < bestD {
			best, bestD, found = t, d, true
		}
	}
	return best, found
}

// PickAt says what is under the screen cell (x, y): a mob, person, NPC, drop or
// door, else a mob or NPC within one cell of it, else the ground there. ok is false when the cell is not on the map.
func (h *Hits) PickAt(m *model.Model, x, y int) (Pick, bool) {
	if h == nil || h.g == nil || !h.Map.In(x, y) {
		return Pick{}, false
	}
	c, r := int64(x-h.Map.X), int64(y-h.Map.Y)
	ts := things(h.g, m)
	if t, ok := thingAt(ts, c, r); ok {
		return t.pick, true
	}
	if t, ok := nearMiss(ts, c, r); ok {
		return t.pick, true
	}
	return Pick{Kind: PickGround, Pos: h.g.worldOf(c, r)}, true
}

// CellAt is the world place at the screen cell, for a keyboard cursor.
func (h *Hits) WorldAt(x, y int) *pb.Pos {
	if h == nil || h.g == nil || !h.Map.In(x, y) {
		return nil
	}
	return h.g.worldOf(int64(x-h.Map.X), int64(y-h.Map.Y))
}

// CellOf is where the world place lies on the screen.
func (h *Hits) CellOf(p *pb.Pos) (Pt, bool) {
	if h == nil || h.g == nil || p == nil {
		return Pt{}, false
	}
	c, r := h.g.cellOf(p)
	if c < 0 || r < 0 || c >= int64(h.g.cw) || r >= int64(h.g.ch) {
		return Pt{}, false
	}
	return Pt{h.Map.X + int(c), h.Map.Y + int(r)}, true
}

// kinds above every glyph: a label by the pointer, and a card.
const (
	kLabel cellKind = iota + 100
	kCard
	kCursor
	kBubble // a speech bubble: open ground only, under labels
)

// text writes s into the grid from (c, r), as the kind k.
func (g *grid) text(c, r int64, s string, k cellKind) {
	for i, rn := range []rune(s) {
		g.put(c+int64(i), r, rn, k)
	}
}

// labelRoom says the cells from c on, w of them, in row r are free of labels.
func (g *grid) labelRoom(used map[[2]int64]bool, c, r int64, w int) bool {
	if r < 0 || r >= int64(g.ch) || c < 0 || c+int64(w) > int64(g.cw) {
		return false
	}
	for i := 0; i < w; i++ {
		if used[[2]int64{c + int64(i), r}] {
			return false
		}
	}
	return true
}

// paintOverlays draws what lies over the map: the labels of everything while
// Alt is held (or the toggle is on), the label by the pointer, the keyboard
// cursor and the info card of what was clicked.
func paintOverlays(g *grid, m *model.Model, v View) {
	// What can be pointed at is listed only for what needs it: the labels and the
	// label by the pointer. A frame with neither draws no list (T5).
	var ts []thing
	if v.Labels || (v.Hits != nil && v.Hover != nil) {
		ts = things(g, m)
	}
	var used map[[2]int64]bool
	place := func(t thing, text string) bool {
		if used == nil {
			used = map[[2]int64]bool{}
		}
		text = " " + text + " "
		w := width(text)
		// Beside the thing, right then left; failing that (a long label by a thing
		// in the middle), on the row above or below it, kept inside the map.
		for _, at := range [][2]int64{{t.c + 1, t.r}, {t.c - int64(w), t.r}, {min(max(t.c-int64(w)/2, 0), int64(g.cw-w)), t.r - 1}, {min(max(t.c-int64(w)/2, 0), int64(g.cw-w)), t.r + 1}} {
			if g.labelRoom(used, at[0], at[1], w) {
				g.text(at[0], at[1], text, kLabel)
				for i := 0; i < w; i++ {
					used[[2]int64{at[0] + int64(i), at[1]}] = true
				}
				return true
			}
		}
		return false
	}
	if v.Labels {
		sort.SliceStable(ts, func(i, j int) bool {
			if ts[i].r != ts[j].r {
				return ts[i].r < ts[j].r
			}
			return ts[i].c < ts[j].c
		})
		for _, t := range ts {
			place(t, t.label)
		}
	}
	if v.Hits != nil && v.Hover != nil {
		c, r := int64(v.Hover.X-v.Hits.org.X), int64(v.Hover.Y-v.Hits.org.Y)
		if t, ok := thingAt(ts, c, r); ok {
			for i := range used {
				delete(used, i)
			}
			place(t, t.label)
			if t.pick.Kind == PickDrop {
				if f := m.FactWords(m.Drops[t.pick.ID].GetItem()); f != "" {
					g.text(t.c+1, t.r+1, " "+f+" ", kLabel)
				}
			}
		}
	}
	if len(v.Bubbles) > 0 {
		paintBubbles(g, m, v.Bubbles, used)
	}
	if v.Cursor != nil {
		c, r := g.cellOf(v.Cursor)
		g.put(c, r, '+', kCursor)
	}
	if v.Card != "" {
		paintCard(g, m, v.Card)
	}
}

// paintCard draws the info card of the mob or NPC with that id in a corner of
// the map: the one its subject is not in.
func paintCard(g *grid, m *model.Model, id string) {
	lines := CardLines(m, id)
	if len(lines) == 0 {
		return
	}
	w := 0
	for i, l := range lines {
		lines[i] = " " + l + " "
		w = max(w, width(lines[i]))
	}
	w = min(w+2, g.cw)
	if w < 6 || len(lines)+2 > g.ch {
		return
	}
	box := make([]string, 0, len(lines)+2)
	box = append(box, "┌"+strings.Repeat("─", w-2)+"┐")
	for _, l := range lines {
		box = append(box, "│"+fit(l, w-2)+"│")
	}
	box = append(box, "└"+strings.Repeat("─", w-2)+"┘")
	c0 := int64(0)
	for _, t := range things(g, m) {
		if t.pick.ID == id && t.c < int64(w)+1 && t.r < int64(len(box))+1 {
			c0 = int64(g.cw - w) // the subject is under the left corner: use the right
		}
	}
	for i, l := range box {
		g.text(c0, int64(i), l, kCard)
	}
}

// CardLines are the lines of the info card of a mob or NPC (D77): its name and
// title, level and difficulty, and what it offers or how to act on it.
func CardLines(m *model.Model, id string) []string {
	sp := m.Self.Pos
	away := func(p *pb.Pos) string {
		if p == nil || sp == nil {
			return ""
		}
		return metres(dist(p.GetX(), p.GetY(), sp.GetX(), sp.GetY())) + " " + bearing(p.GetX()-sp.GetX(), p.GetY()-sp.GetY())
	}
	if e := m.Entities[id]; e != nil || m.Group[id] != nil {
		if e == nil {
			e = m.Group[id].Entity
		}
		if e.GetKind() != pb.EntityKind_ENTITY_KIND_MOB {
			return []string{e.GetName(), fmt.Sprintf("Level %d, %s", e.GetLevel(), away(e.GetPos()))}
		}
		lvl := fmt.Sprintf("Level %d", e.GetLevel())
		if e.GetCon() != "" && !e.GetDead() {
			lvl += ", " + e.GetCon()
		}
		if model.IsRare(e) && !e.GetDead() {
			lvl += ", RARE"
		}
		hp := fmt.Sprintf("HP %d/%d, %s", e.GetHp(), e.GetMaxHp(), away(e.GetPos()))
		act := "Click again or F: attack"
		switch {
		case e.GetDead():
			hp, act = "Dead, "+away(e.GetPos()), ""
		case m.Self.AutoAttack && m.Self.Target == id:
			act = "Attacking · F: stop"
		}
		out := []string{e.GetName(), lvl, hp}
		if act != "" {
			out = append(out, act)
		}
		return out
	}
	if n := m.NPC(id); n != nil {
		offers := roleWord(n.GetRole())
		out := []string{model.NPCLabel(n), "Offers: " + offers}
		switch m.MarkOf(id) {
		case model.MarkOffer:
			out = append(out, "Has something new for you")
		case model.MarkReady:
			out = append(out, "Ready for you to hand in")
		case model.MarkInProgress:
			out = append(out, "Your quest is in progress")
		}
		out = append(out, away(n.GetPos()))
		if m.InReach(n) {
			out = append(out, "Click again or G: talk")
		} else {
			out = append(out, "Option-click to walk up to them")
		}
		return out
	}
	return nil
}
