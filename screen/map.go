package screen

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

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
	glyphTrader    = '$' // a vendor
	glyphTrainer   = 'T' // a trainer (D76: each role has a glyph of its own)
	glyphBanker    = 'B'
	glyphBinder    = 'H' // where you set your bind point (a hearth)
	glyphGuard     = 'g'
	glyphTalker    = 'Q' // someone who only talks (a quest giver)
	glyphBoard     = '&' // a contract board
	glyphOffer     = '!' // something new for you (gold)
	glyphAsk       = '?' // a hand-in that is ready (gold), or a quest in progress (grey)
	glyphMarker    = '%' // where a held scout contract sends you
	glyphScout     = ';' // any other scout marker of the zone map
	glyphSpawn     = 'v' // where a world event's raiders appear
	glyphGoal      = 'G' // where a world event's raiders are heading
	glyphCamp      = 'c' // a camp: where the zone's mobs are found (the zone map's)
	glyphRare      = 'r' // a roaming rare in view
	glyphDoor      = 'D'
	glyphWall      = '#'
	glyphBounds    = '░' // the edge of the zone: where walking ends
	glyphPlan      = '+' // where the plan being built ends
	// The ground (D53). Each says what it is without colour, and none is a
	// glyph above; a building is a solid block like a wall, a road a dotted
	// band, a forest a stand of quotes, water waves, a ford its stepping
	// stones and a bridge its planks. In a dungeon (floor) the floor is
	// dots, the rock beside it is walls, and the rest is blank.
	glyphBuilding = '#'
	glyphRock     = '^'
	glyphWater    = '≈'
	glyphFord     = ':'
	glyphBridge   = '='
	glyphForest   = '"'
	glyphRoad     = '·'
	glyphFloor    = '.'
	// The target is its glyph in upper case: M, X, P.
)

const defaultViewRadiusMm = 20000

type cellKind int

const (
	kEmpty cellKind = iota
	kGround
	kPlace // a named place's name on the large map
	kBounds
	kWall
	kCamp
	kDoor
	kDrop
	kPlan
	kMarkGrey // a quest in progress (D76)
	kMarkGold // something new, or ready to hand in; the board that has it
	kNextDoor // the next door on the way to the objective
	kCharacter
	kMob
	kQuestMob // a mob that counts for a held contract
	kRare
	kTarget
	kPointer // the arrow to an objective that is off the map
	kSelf
)

type cell struct {
	r rune
	k cellKind
	a model.AreaKind // for kGround: which ground
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
		*cur = cell{r: ch, k: k}
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

// npcGlyph is the glyph of an NPC by what it does (D76).
func npcGlyph(n *pb.NpcInfo) rune {
	switch n.GetRole() {
	case "contract_board":
		return glyphBoard
	case "quest_giver":
		return glyphTalker
	case "trainer":
		return glyphTrainer
	case "banker":
		return glyphBanker
	case "binder":
		return glyphBinder
	case "guard":
		return glyphGuard
	}
	return glyphTrader
}

// npcMark is the glyph and kind an NPC is drawn with: its role's glyph, or the
// mark the server's hints put on it (D76): a gold ! for something new, a gold ?
// for a hand-in that is ready, a grey ? for a quest in progress; a board keeps
// its glyph and is gold when something can be taken or handed in.
func npcMark(m *model.Model, n *pb.NpcInfo) (rune, cellKind) {
	gl, mark := npcGlyph(n), m.MarkOf(n.GetId())
	board := n.GetRole() == "contract_board"
	switch {
	case mark == model.MarkReady && board, mark == model.MarkOffer && board:
		return gl, kMarkGold
	case mark == model.MarkOffer:
		return glyphOffer, kMarkGold
	case mark == model.MarkReady:
		return glyphAsk, kMarkGold
	case mark == model.MarkInProgress && !board:
		return glyphAsk, kMarkGrey
	}
	return gl, kDoor
}

// mapRows draws the map panel's interior: cw x ch cells, styled. It scrolls: the
// picture is centred on the character and the scale never changes, so a zone
// larger than the panel is seen a window at a time.
func mapRows(m *model.Model, v View, cw, ch int, st Styler) []string {
	g := newGrid(newScale(zoomed(m.Map.GetViewRadiusMm(), v.Zoom), ch), cw, ch, m.Self.Pos)
	paintWorld(g, m, v, false)
	g.pointToObjective(m)
	g.put(int64(cw/2), int64(ch/2), glyphSelf, kSelf)
	v.Hits.setGrid(g)
	paintOverlays(g, m, v)
	return g.styled(st)
}

// Zoom steps: the scroll wheel (or + and -) halves or doubles what the map shows.
const (
	MinZoom = -2
	MaxZoom = 3
)

// zoomed is the radius the map shows at a zoom: each step in halves it, each
// step out doubles it.
func zoomed(radius int64, zoom int) int64 {
	if radius <= 0 {
		radius = defaultViewRadiusMm
	}
	zoom = min(max(zoom, MinZoom), MaxZoom)
	if zoom >= 0 {
		return max(radius>>uint(zoom), 1000)
	}
	return radius << uint(-zoom)
}

// overviewRows draws the whole zone in cw x ch cells: the bounds fit the panel,
// so one cell spans many metres. What the character has not seen is not there.
func overviewRows(m *model.Model, v View, cw, ch int, st Styler) []string {
	b := m.Map.GetBounds()
	w, h := b.GetMaxX()-b.GetMinX(), b.GetMaxY()-b.GetMinY()
	col := max((w+int64(cw)-1)/int64(max(cw, 1)), (h+2*int64(ch)-1)/(2*int64(max(ch, 1))), 1)
	centre := &pb.Pos{X: (b.GetMinX() + b.GetMaxX()) / 2, Y: (b.GetMinY() + b.GetMaxY()) / 2}
	if z := min(max(v.BigZoom, 0), MaxZoom); z > 0 {
		// Zoomed in, the map is centred on the character and each cell spans less.
		col, centre = max(col>>uint(z), 1), m.Self.Pos
	}
	g := newGrid(scale{col: col, row: 2 * col}, cw, ch, centre)
	paintWorld(g, m, v, true)
	c, r := g.cellOf(m.Self.Pos)
	g.put(c, r, glyphSelf, kSelf)
	paintPlaces(g, m)
	v.Hits.setGrid(g)
	paintOverlays(g, m, v)
	return g.styled(st)
}

// paintPlaces writes the names of the zone's named places (D76) on the large
// map, centred on each place and under every glyph: a name is written only where
// it covers open ground or building, never over another name or a glyph, and is
// moved a row up or down to find room. The smaller places are written first.
func paintPlaces(g *grid, m *model.Model) {
	places := m.PlaceLabels()
	sort.SliceStable(places, func(i, j int) bool { return places[i].Size < places[j].Size })
	for _, p := range places {
		c, r := g.cellOf(p.Pos)
		c0 := c - int64(width(p.Name)/2)
		for _, dr := range []int64{0, -1, 1, -2, 2} {
			if g.roomFor(c0, r+dr, width(p.Name)) {
				g.text(c0, r+dr, p.Name, kPlace)
				break
			}
		}
	}
}

// roomFor says w cells from (c, r) are inside the grid and show no more than
// ground: nothing is drawn over them.
func (g *grid) roomFor(c, r int64, w int) bool {
	if r < 0 || r >= int64(g.ch) || c < 0 || c+int64(w) > int64(g.cw) {
		return false
	}
	for i := 0; i < w; i++ {
		if g.cells[r][c+int64(i)].k > kGround {
			return false
		}
	}
	return true
}

// groundGlyph is the glyph of a kind of ground; 0 for open ground.
func groundGlyph(k model.AreaKind) rune {
	switch k {
	case model.Building:
		return glyphBuilding
	case model.Rock:
		return glyphRock
	case model.Water:
		return glyphWater
	case model.Ford:
		return glyphFord
	case model.Bridge:
		return glyphBridge
	case model.Forest:
		return glyphForest
	case model.Road:
		return glyphRoad
	case model.Floor:
		return glyphFloor
	}
	return 0
}

// groundShown is what a cell shows when several kinds of ground fall in it:
// what blocks the way first, then what crosses water, then the rest, so a thin
// ford, bridge, road or hedge is not lost to the open ground round it.
var groundShown = []model.AreaKind{model.Building, model.Rock, model.Bridge, model.Ford, model.Water, model.Road, model.Forest, model.Floor}

// paintGround puts the zone's ground on the grid. Each cell samples nine points
// across it, so a feature narrower than a cell still shows. In a dungeon (a
// zone with floor, where floor is the only walkable ground) the floor is drawn,
// the rock that touches it is drawn as a wall, and the rest stays blank.
func paintGround(g *grid, t *model.Terrain) {
	key := groundKey{t, g.sc, g.pcol, g.prow, g.cw, g.ch}
	groundMu.Lock()
	rows, ok := groundMemo[key]
	groundMu.Unlock()
	if !ok {
		scratch := &grid{sc: g.sc, cw: g.cw, ch: g.ch, pcol: g.pcol, prow: g.prow, cells: make([][]cell, g.ch)}
		for i := range scratch.cells {
			scratch.cells[i] = make([]cell, g.cw)
		}
		rasterGround(scratch, t)
		rows = scratch.cells
		groundMu.Lock()
		if len(groundMemo) >= 16 {
			clear(groundMemo)
		}
		groundMemo[key] = rows
		groundMu.Unlock()
	}
	for r := range rows {
		copy(g.cells[r], rows[r])
	}
}

// The program draws the same ground again and again while the character stands
// still, so the last few rasters are kept, by terrain, scale and place.
type groundKey struct {
	t          *model.Terrain
	sc         scale
	pcol, prow int64
	cw, ch     int
}

var (
	groundMu   sync.Mutex
	groundMemo = map[groundKey][][]cell{}
)

func rasterGround(g *grid, t *model.Terrain) {
	dungeon := t.HasFloor()
	// Only the areas that reach the view are asked about the cells in it: a
	// dungeon has fifty, and the view is a few of them.
	c0, c1 := (int64(0)-int64(g.cw/2)+g.pcol-1)*g.sc.col, (int64(g.cw)-int64(g.cw/2)+g.pcol+1)*g.sc.col
	r0, r1 := (int64(g.ch/2)-int64(g.ch)+g.prow-1)*g.sc.row, (int64(g.ch/2)+g.prow+1)*g.sc.row
	t = t.Within(c0, r0, c1, r1)
	sub := [3]int64{-1, 0, 1}
	kindAt := func(c, r int) model.AreaKind {
		var seen [8]bool
		cx := (int64(c) - int64(g.cw/2) + g.pcol) * g.sc.col
		cy := (int64(g.ch/2) - int64(r) + g.prow) * g.sc.row
		for _, dx := range sub {
			for _, dy := range sub {
				k := t.Ground(cx+dx*g.sc.col/3, cy+dy*g.sc.row/3)
				for i, s := range groundShown {
					if s == k {
						seen[i] = true
					}
				}
			}
		}
		for i, s := range groundShown {
			if seen[i] {
				return s
			}
		}
		return ""
	}
	if dungeon {
		floor := make([][]bool, g.ch)
		for r := range floor {
			floor[r] = make([]bool, g.cw)
			for c := range floor[r] {
				floor[r][c] = kindAt(c, r) == model.Floor
			}
		}
		for r := range floor {
			for c := range floor[r] {
				switch {
				case floor[r][c]:
					g.cells[r][c] = cell{r: glyphFloor, k: kGround, a: model.Floor}
				case touches(floor, c, r):
					g.cells[r][c] = cell{r: glyphBuilding, k: kGround, a: model.Rock}
				}
			}
		}
		return
	}
	for r := range g.ch {
		for c := range g.cw {
			if k := kindAt(c, r); groundGlyph(k) != 0 {
				g.cells[r][c] = cell{r: groundGlyph(k), k: kGround, a: k}
			}
		}
	}
}

// touches says a cell has floor in one of the eight cells round it.
func touches(floor [][]bool, c, r int) bool {
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			rr, cc := r+dr, c+dc
			if rr >= 0 && rr < len(floor) && cc >= 0 && cc < len(floor[rr]) && floor[rr][cc] {
				return true
			}
		}
	}
	return false
}

// paintWorld puts everything the map shows on the grid except the character:
// the bounds, walls, doors, the NPCs seen (remembered ones too), camps, the
// markers of held scout contracts, drops, the plan, and what is in view.
func paintWorld(g *grid, m *model.Model, v View, overview bool) {
	if t := m.Ground(); t != nil {
		paintGround(g, t)
	}
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
		k := kDoor
		if m.IsNextDoor(d.GetId()) {
			k = kNextDoor
		}
		g.put(c, r, glyphDoor, k)
	}
	// Every marker the zone map lists (D55): zone lines and instance entrances
	// are doors, a scout marker is yours when a held contract names it, and a
	// world event has its spawn and its goal.
	for _, k := range m.Map.GetMarkers() {
		c, r := g.cellOf(k.GetPos())
		switch k.GetKind() {
		case "zone_line", "instance_entrance":
			if m.IsNextDoor(k.GetId()) {
				g.put(c, r, glyphDoor, kNextDoor)
			} else {
				g.put(c, r, glyphDoor, kDoor)
			}
		case "scout":
			g.put(c, r, glyphScout, kPlan)
		case "event_spawn":
			g.put(c, r, glyphSpawn, kPlan)
		case "event_goal":
			g.put(c, r, glyphGoal, kPlan)
		}
	}
	for _, n := range m.KnownNPCs() {
		c, r := g.cellOf(n.GetPos())
		gl, k := npcMark(m, n)
		g.put(c, r, gl, k)
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
			switch {
			case model.IsRare(e) && !e.GetDead():
				k = kRare
			case !e.GetDead() && m.QuestMob(e.GetName()) != nil:
				k = kQuestMob
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
			case kGround:
				s = groundStyle(c.a, st)(s)
			case kPlace:
				s = st.Bold(s)
			case kLabel:
				s = st.Reverse(s)
			case kCard:
				s = st.Bold(s)
			case kCursor:
				s = st.Reverse(st.Bold(s))
			case kBounds, kCamp:
				s = st.Dim(s)
			case kDoor:
				s = st.Cyan(s)
			case kMarkGold, kNextDoor, kPointer:
				s = st.Yellow(st.Bold(s))
			case kMarkGrey:
				s = st.Dim(s)
			case kQuestMob:
				s = st.Yellow(s)
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

// groundStyle is the colour of a kind of ground; it only repeats the glyph.
func groundStyle(k model.AreaKind, st Styler) func(string) string {
	switch k {
	case model.Water:
		return st.Blue
	case model.Ford, model.Bridge:
		return st.Cyan
	case model.Forest:
		return st.Green
	case model.Rock, model.Road, model.Floor:
		return st.Dim
	}
	return func(s string) string { return s }
}

// pointToObjective draws the arrow to the objective's next waypoint at the edge
// of the map when it lies off it (D76): "◄ Gorran 23m". The label is the
// objective's name, short, and the distance in metres.
func (g *grid) pointToObjective(m *model.Model) {
	h := m.Waypoint()
	if h == nil {
		h = m.Objective()
	}
	if h == nil || h.GetPos() == nil {
		return
	}
	c, r := g.cellOf(h.GetPos())
	west, east := c < 0, c >= int64(g.cw)
	north, south := r < 0, r >= int64(g.ch)
	if !west && !east && !north && !south {
		return // on the map: its own mark shows where it is
	}
	arrow := pointerGlyph(west, east, north, south)
	ec, er := min(max(c, 0), int64(g.cw-1)), min(max(r, 0), int64(g.ch-1))
	label := fmt.Sprintf("%s %dm", pointerLabel(m, h), dist64(h.GetPos(), m.Self.Pos)/1000)
	s := string(arrow) + " " + label
	start := int64(0)
	switch {
	case east && !west:
		s = label + " " + string(arrow)
		start = int64(g.cw) - int64(width(s))
	case west:
	default: // above or below: centred on the column the objective lies in
		start = ec - int64(width(s))/2
	}
	start = min(max(start, 0), max(int64(g.cw)-int64(width(s)), 0))
	for i, rn := range []rune(s) {
		g.put(start+int64(i), er, rn, kPointer)
	}
}

func dist64(a, b *pb.Pos) int64 {
	dx, dy := float64(a.GetX()-b.GetX()), float64(a.GetY()-b.GetY())
	return int64(math.Round(math.Hypot(dx, dy)))
}

// pointerGlyph is the arrow for the side or corner the objective lies beyond.
func pointerGlyph(west, east, north, south bool) rune {
	switch {
	case west && north:
		return '◤'
	case east && north:
		return '◥'
	case west && south:
		return '◣'
	case east && south:
		return '◢'
	case west:
		return '◄'
	case east:
		return '►'
	case north:
		return '▲'
	}
	return '▼'
}

// pointerLabel is the objective's name in a few cells: "Gorran" for "Talk to Gorran Stoutarm".
func pointerLabel(m *model.Model, h *pb.Hint) string {
	l := h.GetLabel()
	person := false
	for _, p := range []string{"Talk to ", "Hand in: ", "Take: ", "Report back: "} {
		if strings.HasPrefix(l, p) {
			l, person = strings.TrimPrefix(l, p), p == "Talk to "
		}
	}
	if i := strings.Index(l, " — "); i > 0 {
		l = l[:i]
	}
	if f := strings.Fields(l); person && len(f) > 1 {
		l = f[0] // the first name is enough to know who
	}
	return strings.TrimSpace(fit(l, 16))
}
