package screen

import (
	"strings"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The glyph vocabulary of the map. Each glyph says what a thing is without
// colour; colour and emphasis only repeat it. Whether another character is
// driven by a person or an agent is not in the protocol and never shown (D17).
const (
	glyphSelf      = '@'
	glyphMob       = 'm'
	glyphDeadMob   = 'x'
	glyphCharacter = 'p' // another character
	glyphDrop      = '*'
	glyphLocked    = '~' // a drop this character may not take (a Rare one)
	glyphTrader    = '$' // a vendor, trainer, binder or banker
	glyphBoard     = '&' // a contract board
	glyphTalker    = '?' // someone who only talks (a quest giver)
	glyphMarker    = '%' // where a held scout contract sends you
	glyphCamp      = 'c' // a camp: where the zone's mobs are found (the zone map's)
	glyphRare      = 'r' // a roaming rare in view
	glyphDoor      = 'D'
	glyphWall      = '#'
	glyphBounds    = '.'
	glyphPlan      = '+' // where the plan being built ends
	// The target is its glyph in upper case: M, X, P.
)

const defaultViewRadiusMm = 20000

type cellKind int

const (
	kEmpty cellKind = iota
	kBounds
	kWall
	kCamp
	kDoor
	kDrop
	kPlan
	kCharacter
	kMob
	kRare
	kTarget
	kSelf
)

type cell struct {
	r rune
	k cellKind
}

// scale is the map's projection: how many millimetres a cell spans. A cell is
// about twice as tall as wide, so a row spans twice the millimetres of a column
// and the picture keeps its shape. The scale is fixed by the viewport size and
// the view radius, never by the player's position, so the picture only ever
// shifts by whole cells.
type scale struct {
	col, row int64 // mm per column, mm per row
}

func newScale(viewRadiusMm int64, rows int) scale {
	if viewRadiusMm <= 0 {
		viewRadiusMm = defaultViewRadiusMm
	}
	row := max(2*viewRadiusMm/int64(max(rows, 1)), 2)
	return scale{col: max(row/2, 1), row: row}
}

func roundDiv(a, b int64) int64 {
	if a < 0 {
		return -((-a + b/2) / b)
	}
	return (a + b/2) / b
}

// grid is the viewport: a window of cw x ch cells centred on the player's cell.
// World to cell is a global integer grid (round(x/col), round(y/row)) and the
// window's origin is the player's own cell, so the centre is snapped: moving
// less than a cell never moves the map (the old mapgeo technique, kept; its
// data is not).
type grid struct {
	sc         scale
	cw, ch     int
	pcol, prow int64 // the player's cell in the global grid
	cells      [][]cell
}

func newGrid(sc scale, cw, ch int, self *pb.Pos) *grid {
	g := &grid{sc: sc, cw: cw, ch: ch, pcol: roundDiv(self.GetX(), sc.col), prow: roundDiv(self.GetY(), sc.row)}
	g.cells = make([][]cell, ch)
	for i := range g.cells {
		g.cells[i] = make([]cell, cw)
	}
	return g
}

// cellOf maps a world position to window coordinates; north is up.
func (g *grid) cellOf(p *pb.Pos) (c, r int64) {
	return roundDiv(p.GetX(), g.sc.col) - g.pcol + int64(g.cw/2), int64(g.ch/2) - (roundDiv(p.GetY(), g.sc.row) - g.prow)
}

func (g *grid) put(c, r int64, ch rune, k cellKind) {
	if c < 0 || r < 0 || c >= int64(g.cw) || r >= int64(g.ch) {
		return
	}
	if cur := &g.cells[r][c]; k >= cur.k {
		*cur = cell{ch, k}
	}
}

// line rasterizes a segment between two world positions (Bresenham).
func (g *grid) line(a, b *pb.Pos, ch rune, k cellKind) {
	c0, r0 := g.cellOf(a)
	c1, r1 := g.cellOf(b)
	// A segment wholly on one side of the window draws nothing.
	if (c0 < 0 && c1 < 0) || (r0 < 0 && r1 < 0) || (c0 >= int64(g.cw) && c1 >= int64(g.cw)) || (r0 >= int64(g.ch) && r1 >= int64(g.ch)) {
		return
	}
	dc, dr := abs(c1-c0), abs(r1-r0)
	sc, sr := int64(1), int64(1)
	if c0 > c1 {
		sc = -1
	}
	if r0 > r1 {
		sr = -1
	}
	e := dc - dr
	for {
		g.put(c0, r0, ch, k)
		if c0 == c1 && r0 == r1 {
			return
		}
		e2 := 2 * e
		if e2 > -dr {
			e -= dr
			c0 += sc
		}
		if e2 < dc {
			e += dc
			r0 += sr
		}
	}
}

// entityGlyph is the glyph for an entity in view.
func entityGlyph(e *pb.EntityState) rune {
	switch {
	case e.GetKind() == pb.EntityKind_ENTITY_KIND_MOB && e.GetDead():
		return glyphDeadMob
	case model.IsRare(e):
		return glyphRare
	case e.GetKind() == pb.EntityKind_ENTITY_KIND_MOB:
		return glyphMob
	}
	return glyphCharacter
}

func upper(r rune) rune { return []rune(strings.ToUpper(string(r)))[0] }

// npcGlyph is the glyph of an NPC by what it does.
func npcGlyph(n *pb.NpcInfo) rune {
	switch n.GetRole() {
	case "contract_board":
		return glyphBoard
	case "quest_giver":
		return glyphTalker
	}
	return glyphTrader
}

// mapRows draws the map panel's interior: cw x ch cells, styled. It scrolls: the
// picture is centred on the character and the scale never changes, so a zone
// larger than the panel is seen a window at a time.
func mapRows(m *model.Model, v View, cw, ch int, st Styler) []string {
	g := newGrid(newScale(m.Map.GetViewRadiusMm(), ch), cw, ch, m.Self.Pos)
	paintWorld(g, m, v)
	g.put(int64(cw/2), int64(ch/2), glyphSelf, kSelf)
	return g.styled(st)
}

// overviewRows draws the whole zone in cw x ch cells: the bounds fit the panel,
// so one cell spans many metres. What the character has not seen is not there.
func overviewRows(m *model.Model, v View, cw, ch int, st Styler) []string {
	b := m.Map.GetBounds()
	w, h := b.GetMaxX()-b.GetMinX(), b.GetMaxY()-b.GetMinY()
	col := max((w+int64(cw)-1)/int64(max(cw, 1)), (h+2*int64(ch)-1)/(2*int64(max(ch, 1))), 1)
	centre := &pb.Pos{X: (b.GetMinX() + b.GetMaxX()) / 2, Y: (b.GetMinY() + b.GetMaxY()) / 2}
	g := newGrid(scale{col: col, row: 2 * col}, cw, ch, centre)
	paintWorld(g, m, v)
	c, r := g.cellOf(m.Self.Pos)
	g.put(c, r, glyphSelf, kSelf)
	return g.styled(st)
}

// paintWorld puts everything the map shows on the grid except the character:
// the bounds, walls, doors, the NPCs seen (remembered ones too), camps, the
// markers of held scout contracts, drops, the plan, and what is in view.
func paintWorld(g *grid, m *model.Model, v View) {
	if b := m.Map.GetBounds(); b != nil {
		corners := []*pb.Pos{{X: b.GetMinX(), Y: b.GetMinY()}, {X: b.GetMaxX(), Y: b.GetMinY()}, {X: b.GetMaxX(), Y: b.GetMaxY()}, {X: b.GetMinX(), Y: b.GetMaxY()}}
		if b.GetMinX() != b.GetMaxX() || b.GetMinY() != b.GetMaxY() {
			for i := range corners {
				g.line(corners[i], corners[(i+1)%4], glyphBounds, kBounds)
			}
		}
	}
	for _, w := range m.Map.GetWalls() {
		g.line(w.GetA(), w.GetB(), glyphWall, kWall)
	}
	for _, camp := range m.Camps() {
		c, r := g.cellOf(camp.GetPos())
		g.put(c, r, glyphCamp, kCamp)
	}
	for _, d := range m.Map.GetDoors() {
		c, r := g.cellOf(d.GetPos())
		g.put(c, r, glyphDoor, kDoor)
	}
	for _, n := range m.KnownNPCs() {
		c, r := g.cellOf(n.GetPos())
		g.put(c, r, npcGlyph(n), kDoor)
	}
	for _, k := range m.Self.Contracts {
		if p := m.Marker(k); p != nil {
			c, r := g.cellOf(p)
			g.put(c, r, glyphMarker, kPlan)
		}
	}
	for _, id := range m.DropIDs() {
		c, r := g.cellOf(m.Drops[id].GetPos())
		gl := glyphDrop
		if !m.Drops[id].GetCanLoot() {
			gl = glyphLocked
		}
		g.put(c, r, gl, kDrop)
	}
	if v.Plan.Dest != nil {
		c, r := g.cellOf(v.Plan.Dest)
		g.put(c, r, glyphPlan, kPlan)
	}
	put := func(e *pb.EntityState) {
		c, r := g.cellOf(e.GetPos())
		gl, k := entityGlyph(e), kCharacter
		if e.GetKind() == pb.EntityKind_ENTITY_KIND_MOB {
			k = kMob
			if model.IsRare(e) && !e.GetDead() {
				k = kRare
			}
		}
		if e.GetId() == m.Self.Target {
			gl, k = upper(gl), kTarget
		}
		g.put(c, r, gl, k)
	}
	for _, id := range m.GroupIDs() {
		put(m.Group[id].Entity)
	}
	for _, id := range m.EntityIDs() {
		put(m.Entities[id])
	}
}

// styled turns the grid into lines, with the colour of each kind of thing.
func (g *grid) styled(st Styler) []string {
	rows := make([]string, len(g.cells))
	for r := range g.cells {
		var b strings.Builder
		for _, c := range g.cells[r] {
			s := string(c.r)
			switch c.k {
			case kEmpty:
				s = " "
			case kBounds, kCamp:
				s = st.Dim(s)
			case kDoor:
				s = st.Cyan(s)
			case kDrop:
				s = st.Green(s)
			case kPlan:
				s = st.Yellow(s)
			case kMob:
				s = st.Red(s)
			case kRare:
				s = st.Yellow(st.Bold(s))
			case kTarget:
				s = st.Reverse(st.Bold(s))
			case kSelf:
				s = st.Bold(s)
			}
			b.WriteString(s)
		}
		rows[r] = b.String()
	}
	return rows
}
