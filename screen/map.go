package screen

import (
	"fmt"
	"math"
	"slices"
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
	glyphGoal      = '◆' // where a world event's raiders are heading (not G: it read as g, the guard)
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
	kEvent    // a world event's spawn and goal: magenta, no person's or quest's colour
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
	flat       []cell // what the rows of cells are cut from
	// The large map's places, most significant first, and the names written (D121).
	places []model.PlaceLabel
	named  []namedPlace
}

func newGrid(sc scale, cw, ch int, self *pb.Pos) *grid {
	g := &grid{sc: sc, cw: cw, ch: ch, pcol: roundDiv(self.GetX(), sc.col), prow: roundDiv(self.GetY(), sc.row)}
	g.flat = make([]cell, cw*ch)
	g.cells = make([][]cell, ch)
	for i := range g.cells {
		g.cells[i] = g.flat[i*cw : (i+1)*cw : (i+1)*cw]
	}
	return g
}

// gridFor is a blank grid for the map about to be drawn: the one the map before
// it was drawn on when the window is the same size (the grid is the same memory
// from one frame to the next), else a new one (T5).
func gridFor(h *Hits, sc scale, cw, ch int, self *pb.Pos) *grid {
	if h != nil && h.spare != nil && h.spare.cw == cw && h.spare.ch == ch {
		g := h.spare
		h.spare = nil
		g.sc, g.pcol, g.prow = sc, roundDiv(self.GetX(), sc.col), roundDiv(self.GetY(), sc.row)
		clear(g.flat)
		g.places, g.named = nil, g.named[:0]
		return g
	}
	return newGrid(sc, cw, ch, self)
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
	g := gridFor(v.Hits, newScale(zoomed(m.Map.GetViewRadiusMm(), v.Zoom), ch), cw, ch, m.Self.Pos)
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
// It also says which marks are on the picture, for the legend to list.
func overviewRows(m *model.Model, v View, cw, ch int, st Styler) ([]string, marks) {
	g := overviewGrid(m, v, cw, ch)
	c, r := g.cellOf(m.Self.Pos)
	g.put(c, r, glyphSelf, kSelf)
	paintPlaces(g, m, v)
	present := g.marksShown()
	v.Hits.setGrid(g)
	paintOverlays(g, m, v)
	paintTips(g, m, v)
	return g.styled(st), present
}

// overviewScale is how many millimetres a column spans on the large map when it
// is cw x ch cells: the bounds fit, keeping the true aspect.
func overviewScale(b *pb.Rect, cw, ch int) int64 {
	w, h := b.GetMaxX()-b.GetMinX(), b.GetMaxY()-b.GetMinY()
	// The bounds span an even number of cells either side of the character's
	// centre cell: the zone's edges are drawn on cells, so a zone as tall as the
	// panel (the 450 m Downs at 80 x 24) would otherwise lose its south edge.
	cols, rows := int64(max((cw-1)/2*2, 1)), int64(max((ch-1)/2*2, 1))
	return max((w+cols-1)/cols, (h+2*rows-1)/(2*rows), 1)
}

// overviewWidth is the width the large map needs at a height, at most max:
// where the zone is taller than wide the map is narrower than the window and
// the legend sits beside it, not far off.
func overviewWidth(m *model.Model, v View, maxW, ch int) int {
	if v.BigZoom > 0 {
		return maxW
	}
	b := m.Map.GetBounds()
	col := overviewScale(b, maxW, ch)
	// The columns the zone spans at that scale, and the edges drawn on cells either
	// side of them, as overviewScale counts them.
	cells := int((b.GetMaxX()-b.GetMinX()+col-1)/col) + 2
	return min(cells|1, maxW)
}

// overviewGrid paints the zone as the large map shows it, without the character
// and the names.
func overviewGrid(m *model.Model, v View, cw, ch int) *grid {
	b := m.Map.GetBounds()
	col := overviewScale(b, cw, ch)
	centre := &pb.Pos{X: (b.GetMinX() + b.GetMaxX()) / 2, Y: (b.GetMinY() + b.GetMaxY()) / 2}
	if z := min(max(v.BigZoom, 0), MaxZoom); z > 0 {
		// Zoomed in, the map is centred on the character and each cell spans less.
		col, centre = max(col>>uint(z), 1), m.Self.Pos
	}
	g := gridFor(v.Hits, scale{col: col, row: 2 * col}, cw, ch, centre)
	paintWorld(g, m, v, true)
	return g
}

// placeMinCells is the least a place must span on the large map to be named
// when the map is not showing everything (D120): its longer side, counted in
// columns (a row is two columns tall), at least this many. A lake or a wood
// clears it at the widest zoom; a hut, a stone or a sheepfold pen does not, and
// each step in halves what a cell spans, so doubles the cells a place spans.
// A place that matters (model.Sig*: it has a building, a person, a door or the
// objective) is named at any zoom whatever its size (D121).
const placeMinCells = 8

// groupCols and groupRows are how near a place left unnamed must be to a name,
// in cells from the name's middle, to be counted under it as "Name +N" (D121).
const (
	groupCols = 12
	groupRows = 6
)

// searchMoreCols and searchMoreRows are how far beside a place its name may go,
// beyond its own extent and the name's own width, when the spots inside are
// taken; the spots nearest the place are tried first.
const (
	searchMoreCols = 6
	searchMoreRows = 2
)

// namedPlace is a name written on the large map and the places left out that
// are counted under it.
type namedPlace struct {
	name   string // the place's own name
	pos    *pb.Pos
	text   string // what is written: the name, or "name +N"
	c, r   int64  // the cell its text starts in
	w      int
	hidden []string // the names of the places left out near it, most significant first
}

// paintPlaces writes the names of the zone's named places (D76) on the large
// map, one for each place (D120): where the place has several pieces in one
// cluster it is the middle of them all, else beside the largest (D121). A name
// is written only where it covers open ground, a building, rock or forest, and
// never road, water, a ford or bridge, a door, the character, the objective, a
// mark or another name; if the middle does not fit, the nearest spot inside or
// beside the place does, and if none does the name is left out. The places that
// matter most are written first, then the larger. A place that is small and
// matters to nobody is not named, unless everything is (Alt held); the ones
// left out near a name are counted under it, "Mill +2" (D121).
func paintPlaces(g *grid, m *model.Model, v View) {
	places := m.PlaceLabels()
	model.RankPlaces(places)
	g.places, g.named = places, g.named[:0]
	var cands []int // indexes into places
	for i, p := range places {
		span := max(p.DX, p.DY/2) / g.sc.col // columns
		if v.Labels || p.Sig > model.SigSize || span >= placeMinCells {
			cands = append(cands, i)
		}
	}
	snap := slices.Clone(g.flat)
	named := g.nameAll(m, cands, nil)
	// The places left out, and the ones named that found no room, are counted
	// under the nearest name that matters most; then the names are written again
	// with the counts, which may make a name wider.
	byName := map[string]int{}
	for i, n := range named {
		byName[n.name] = i
	}
	grouped := false
	for i := range places {
		if _, ok := byName[places[i].Name]; ok {
			continue
		}
		hc, hr := g.cellOf(places[i].Pos)
		for j := range named {
			if c, r := g.cellOf(named[j].pos); abs(c-hc) <= groupCols && abs(r-hr) <= groupRows {
				named[j].hidden, grouped = append(named[j].hidden, places[i].Name), true
				break
			}
		}
	}
	if !grouped {
		g.named = named
		return
	}
	copy(g.flat, snap)
	more := map[string][]string{}
	for _, n := range named {
		more[n.name] = n.hidden
	}
	g.named = g.nameAll(m, cands, more)
}

// nameAll writes the names of the places cands (indexes into g.places, most
// significant first) and returns those that found room. group has the names left
// out under a name, and the name is then written "Name +N".
func (g *grid) nameAll(m *model.Model, cands []int, group map[string][]string) []namedPlace {
	b := m.Map.GetBounds()
	_, top := g.cellOf(&pb.Pos{X: b.GetMinX(), Y: b.GetMaxY()})
	_, bottom := g.cellOf(&pb.Pos{X: b.GetMinX(), Y: b.GetMinY()})
	var objective [2]int64
	haveObjective := false
	if h := m.Objective(); h != nil && h.GetPos() != nil {
		objective[0], objective[1] = g.cellOf(h.GetPos())
		haveObjective = true
	}
	var out []namedPlace
	for _, i := range cands {
		p := &g.places[i]
		texts := []string{p.Name}
		if hid := group[p.Name]; len(hid) > 0 {
			texts = []string{fmt.Sprintf("%s +%d", p.Name, len(hid)), p.Name}
		}
		c, r := g.cellOf(p.Pos)
		for ti, text := range texts {
			w := width(text)
			halfC := max(p.DX/g.sc.col/2, 2) + int64(w)/2 + searchMoreCols
			halfR := max(p.DY/g.sc.row/2, 0) + 1 + searchMoreRows
			type spot struct{ dc, dr, cost int64 }
			var spots []spot
			for dr := -halfR; dr <= halfR; dr++ {
				for dc := -halfC; dc <= halfC; dc++ {
					spots = append(spots, spot{dc, dr, dc*dc + 4*dr*dr})
				}
			}
			sort.SliceStable(spots, func(i, j int) bool { return spots[i].cost < spots[j].cost })
			done := false
			for _, s := range spots {
				c0, row := c+s.dc-int64(w/2), r+s.dr
				if row <= top || row >= bottom || !g.roomFor(c0, row, w) || g.nameAt(c0-1, row) || g.nameAt(c0+int64(w), row) {
					continue
				}
				if haveObjective && row == objective[1] && objective[0] >= c0 && objective[0] < c0+int64(w) {
					continue
				}
				g.text(c0, row, text, kPlace)
				n := namedPlace{name: p.Name, pos: p.Pos, text: text, c: c0, r: row, w: w}
				if ti == 0 {
					n.hidden = group[p.Name]
				}
				out = append(out, n)
				done = true
				break
			}
			if done {
				break
			}
		}
	}
	return out
}

// nameAt says the cell holds a letter of a place's name: the next name keeps a
// cell clear of it.
func (g *grid) nameAt(c, r int64) bool {
	return r >= 0 && r < int64(g.ch) && c >= 0 && c < int64(g.cw) && g.cells[r][c].k == kPlace
}

// nameable says a name may be written over the cell (D120): over open ground,
// or a building, rock or forest, and over nothing else. Road, water, ford and
// bridge are ground too, and a name over them breaks the line a person follows.
func (c cell) nameable() bool {
	switch c.k {
	case kEmpty:
		return true
	case kGround:
		switch c.a {
		case model.Building, model.Rock, model.Forest, model.Floor:
			return true
		}
	}
	return false
}

// roomFor says w cells from (c, r) are inside the grid and show nothing a name
// may not cover: nothing else is drawn over them.
func (g *grid) roomFor(c, r int64, w int) bool {
	if r < 0 || r >= int64(g.ch) || c < 0 || c+int64(w) > int64(g.cw) {
		return false
	}
	for i := 0; i < w; i++ {
		if !g.cells[r][c+int64(i)].nameable() {
			return false
		}
	}
	return true
}

// A mark on the picture: what the legend lists is what is here.
type markKey struct {
	k cellKind
	r rune
}

type marks map[markKey]bool

// marksShown is the marks on the grid: each kind and glyph that has a cell.
func (g *grid) marksShown() marks {
	out := marks{}
	for _, row := range g.cells {
		for _, c := range row {
			if c.k != kEmpty && c.r != ' ' {
				out[markKey{c.k, c.r}] = true
			}
		}
	}
	return out
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

// paintStatic puts what does not move on the grid: the ground, the bounds and
// the walls. They are the same for the same zone map, scale and cell the
// character stands in, so the layer is made once for each and copied (T5).
func paintStatic(g *grid, m *model.Model, overview bool) {
	copy(g.flat, staticLayer(g, m.Ground(), m.Map, overview))
}

// The program draws the same ground again and again while the character stands
// still, so the last few layers are kept, by zone map, scale and place.
type groundKey struct {
	t          *model.Terrain
	zm         *pb.ZoneMap
	sc         scale
	pcol, prow int64
	cw, ch     int
	overview   bool // the large map's ground (D120), not the panel's
}

var (
	groundMu   sync.Mutex
	groundMemo = map[groundKey][]cell{}
)

// staticLayer is the layer for g's place and scale: the ground, and with a zone
// map the bounds and walls over it. Its cells are shared: callers copy them.
func staticLayer(g *grid, t *model.Terrain, zm *pb.ZoneMap, overview bool) []cell {
	key := groundKey{t, zm, g.sc, g.pcol, g.prow, g.cw, g.ch, overview}
	groundMu.Lock()
	cells, ok := groundMemo[key]
	groundMu.Unlock()
	if ok {
		return cells
	}
	scratch := newGrid(g.sc, g.cw, g.ch, &pb.Pos{})
	scratch.pcol, scratch.prow = g.pcol, g.prow
	if t != nil {
		rasterGround(scratch, t, overview)
	}
	if zm != nil {
		paintLines(scratch, zm)
	}
	groundMu.Lock()
	if len(groundMemo) >= 16 {
		clear(groundMemo)
	}
	groundMemo[key] = scratch.flat
	groundMu.Unlock()
	return scratch.flat
}

// rasterGround puts the zone's ground on the grid. Each cell samples nine points
// across it, so a feature narrower than a cell still shows. In a dungeon (a
// zone with floor, where floor is the only walkable ground) the floor is drawn,
// the rock that touches it is drawn as a wall, and the rest stays blank.
func rasterGround(g *grid, t *model.Terrain, overview bool) {
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
				if overview && k == model.Forest && forestGap(g, c, r) {
					continue
				}
				g.cells[r][c] = cell{r: groundGlyph(k), k: kGround, a: k}
			}
		}
	}
}

// forestGap says a forest cell of the large map is left blank (D120): the
// forest is a stipple of every other cell, which reads calmer than a solid run
// of quotes and needs no glyph a terminal might lack. The cell's place in the
// zone's own grid decides, not its place in the window, so the pattern does not
// flicker as the map scrolls.
func forestGap(g *grid, c, r int) bool {
	gc, gr := int64(c-g.cw/2)+g.pcol, g.prow-int64(r-g.ch/2)
	return (gc+gr)&1 != 0
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

// paintLines draws the zone's bounds and walls.
func paintLines(g *grid, zm *pb.ZoneMap) {
	if b := zm.GetBounds(); b != nil {
		corners := []*pb.Pos{{X: b.GetMinX(), Y: b.GetMinY()}, {X: b.GetMaxX(), Y: b.GetMinY()}, {X: b.GetMaxX(), Y: b.GetMaxY()}, {X: b.GetMinX(), Y: b.GetMaxY()}}
		if b.GetMinX() != b.GetMaxX() || b.GetMinY() != b.GetMaxY() {
			for i := range corners {
				g.line(corners[i], corners[(i+1)%4], glyphBounds, kBounds)
			}
		}
	}
	for _, w := range zm.GetWalls() {
		g.line(w.GetA(), w.GetB(), glyphWall, kWall)
	}
}

// paintWorld puts everything the map shows on the grid except the character:
// the bounds, walls, doors, the NPCs seen (remembered ones too), camps, the
// markers of held scout contracts, drops, the plan, and what is in view.
func paintWorld(g *grid, m *model.Model, v View, overview bool) {
	paintStatic(g, m, overview)
	// The camps are scattered all over the zone: the widest large map leaves
	// them out (D120) and shows them once zoomed in, or when everything is named.
	if !overview || v.BigZoom > 0 || v.Labels {
		for _, camp := range m.Camps() {
			c, r := g.cellOf(camp.GetPos())
			g.put(c, r, glyphCamp, kCamp)
		}
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
			g.put(c, r, glyphSpawn, kEvent)
		case "event_goal":
			g.put(c, r, glyphGoal, kEvent)
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

// styled turns the grid into lines, with the colour of each kind of thing. A
// run of cells of one kind is styled once, not a cell at a time.
func (g *grid) styled(st Styler) []string {
	rows := make([]string, len(g.cells))
	var b strings.Builder
	for r := range g.cells {
		b.Reset()
		cells := g.cells[r]
		for c := 0; c < len(cells); {
			k, a := cells[c].k, cells[c].a
			end := c + 1
			for end < len(cells) && cells[end].k == k && (k != kGround || cells[end].a == a) {
				end++
			}
			if k == kEmpty {
				for ; c < end; c++ {
					b.WriteByte(' ')
				}
				continue
			}
			var run strings.Builder
			run.Grow(end - c)
			for ; c < end; c++ {
				run.WriteRune(cells[c].r)
			}
			b.WriteString(cellStyle(k, a, st)(run.String()))
		}
		rows[r] = b.String()
	}
	return rows
}

// cellStyle is how a map cell of kind k is drawn. The map and the legend both
// draw through it, so a glyph in the legend looks as it does on the map.
func cellStyle(k cellKind, area model.AreaKind, st Styler) func(string) string {
	switch k {
	case kGround:
		return groundStyle(area, st)
	case kPlace, kCard:
		return st.Bold
	case kSelf:
		// The first thing the eye finds (D120), and not by colour.
		return func(s string) string { return st.Reverse(st.Bold(s)) }
	case kBubble:
		return func(x string) string { return st.Chat.Say(st.Bold(x)) }
	case kLabel:
		return st.Reverse
	case kCursor:
		return func(s string) string { return st.Reverse(st.Bold(s)) }
	case kBounds, kCamp, kMarkGrey:
		return st.Dim
	case kDoor:
		return st.Cyan
	case kMarkGold, kNextDoor, kPointer, kRare:
		return func(s string) string { return st.Yellow(st.Bold(s)) }
	case kQuestMob, kPlan:
		return st.Yellow
	case kDrop:
		return st.Green
	case kMob:
		return st.Red
	case kEvent:
		return st.Magenta
	case kTarget:
		return func(s string) string { return st.Reverse(st.Bold(s)) }
	}
	return ident
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
	case model.Road:
		return st.Bold // a road must not fade into the ground it crosses (D120)
	case model.Rock, model.Floor:
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
