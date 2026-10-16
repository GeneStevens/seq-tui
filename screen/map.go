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
	kDoor
	kDrop
	kPlan
	kCharacter
	kMob
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
	case e.GetKind() == pb.EntityKind_ENTITY_KIND_MOB:
		return glyphMob
	}
	return glyphCharacter
}

func upper(r rune) rune { return []rune(strings.ToUpper(string(r)))[0] }

// mapRows draws the map panel's interior: cw x ch cells, styled.
func mapRows(m *model.Model, v View, cw, ch int, st Styler) []string {
	g := newGrid(newScale(m.Map.GetViewRadiusMm(), ch), cw, ch, m.Self.Pos)
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
	for _, d := range m.Map.GetDoors() {
		c, r := g.cellOf(d.GetPos())
		g.put(c, r, glyphDoor, kDoor)
	}
	for _, id := range m.DropIDs() {
		c, r := g.cellOf(m.Drops[id].GetPos())
		gl := glyphDrop
		if !m.Drops[id].GetCanLoot() {
			gl = glyphLocked
		}
		g.put(c, r, gl, kDrop)
	}
	for _, n := range m.NPCs {
		c, r := g.cellOf(n.GetPos())
		g.put(c, r, glyphTrader, kDoor)
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
	g.put(int64(cw/2), int64(ch/2), glyphSelf, kSelf)

	rows := make([]string, ch)
	for r := range g.cells {
		var b strings.Builder
		for _, c := range g.cells[r] {
			s := string(c.r)
			switch c.k {
			case kEmpty:
				s = " "
			case kBounds:
				s = st.Dim(s)
			case kDoor:
				s = st.Cyan(s)
			case kDrop:
				s = st.Green(s)
			case kPlan:
				s = st.Yellow(s)
			case kMob:
				s = st.Red(s)
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
