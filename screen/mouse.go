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
	g    *grid
	org  Pt // where the next map is drawn
}

// begin readies the record for a drawing.
func (h *Hits) begin() {
	h.Map, h.Side, h.Tabs, h.g = Rect{}, Rect{}, h.Tabs[:0], nil
}

// at says where the map or screen about to be drawn starts.
func (h *Hits) at(x, y int) {
	if h != nil {
		h.org = Pt{x, y}
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
	PickGround PickKind = iota // open ground: a click walks there
	PickMob
	PickNPC
	PickDrop
	PickDoor
	PickPlayer
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
}

// things lists what the map shows that can be pointed at, in the order of
// priority a click takes them: mobs, people, then NPCs, drops and doors.
func things(g *grid, m *model.Model) []thing {
	var out []thing
	add := func(p Pick, label string) {
		c, r := g.cellOf(p.Pos)
		if c < 0 || r < 0 || c >= int64(g.cw) || r >= int64(g.ch) {
			return
		}
		out = append(out, thing{c: c, r: r, pick: p, label: label})
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
		add(Pick{Kind: kind, ID: e.GetId(), Pos: e.GetPos(), Name: e.GetName()}, label)
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
		add(Pick{Kind: PickDoor, ID: d.GetId(), Pos: d.GetPos(), Name: name}, name)
	}
	return out
}

// worldOf is the middle of the cell (c, r) of the grid, in the world.
func (g *grid) worldOf(c, r int64) *pb.Pos {
	return &pb.Pos{X: (c - int64(g.cw/2) + g.pcol) * g.sc.col, Y: (int64(g.ch/2) - r + g.prow) * g.sc.row}
}

// thingAt is what sits in the cell, the highest priority first.
func thingAt(ts []thing, c, r int64) (thing, bool) {
	for _, t := range ts {
		if t.c == c && t.r == r && !(t.pick.Kind == PickMob && strings.HasSuffix(t.label, "(dead)")) {
			return t, true
		}
	}
	return thing{}, false
}

// PickAt says what is under the screen cell (x, y): a mob, person, NPC, drop or
// door, else the ground there. ok is false when the cell is not on the map.
func (h *Hits) PickAt(m *model.Model, x, y int) (Pick, bool) {
	if h == nil || h.g == nil || !h.Map.In(x, y) {
		return Pick{}, false
	}
	c, r := int64(x-h.Map.X), int64(y-h.Map.Y)
	if t, ok := thingAt(things(h.g, m), c, r); ok {
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
	ts := things(g, m)
	used := map[[2]int64]bool{}
	place := func(t thing, text string) bool {
		text = " " + text + " "
		w := width(text)
		for _, c := range []int64{t.c + 1, t.c - int64(w)} {
			if g.labelRoom(used, c, t.r, w) {
				g.text(c, t.r, text, kLabel)
				for i := 0; i < w; i++ {
					used[[2]int64{c + int64(i), t.r}] = true
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
		if e.GetDead() {
			hp, act = "Dead, "+away(e.GetPos()), ""
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
			out = append(out, "Click again to walk up to them")
		}
		return out
	}
	return nil
}
