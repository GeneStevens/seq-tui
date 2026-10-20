package model

import (
	"fmt"
	"math"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Terrain is the ground of a zone (D53, D55): areas with a kind. It is read from
// the zone map the server sent, never from anywhere else; a zone map with no
// areas gives none, and the map then shows walls, camps and doors only. The
// paths mobs and guards walk are not told (D56): a client sees them walk.
type Terrain struct {
	Areas []Area
}

// AreaKind is what an area is.
type AreaKind string

// The kinds of area, as seq's content names them.
const (
	Building AreaKind = "building" // blocks movement and sight
	Rock     AreaKind = "rock"     // blocks movement and sight
	Water    AreaKind = "water"    // blocks movement, except under a ford or bridge
	Ford     AreaKind = "ford"     // walkable ground across water
	Bridge   AreaKind = "bridge"   // walkable ground across water
	Forest   AreaKind = "forest"   // slows movement
	Road     AreaKind = "road"     // speeds movement
	Floor    AreaKind = "floor"    // a dungeon's only walkable ground
)

// Area is a polygon on the ground plane, in millimetres; the last corner joins
// the first.
type Area struct {
	ID   string
	Kind AreaKind
	Poly []*pb.Pos
	// minX..maxY is the polygon's bounding box, for a quick miss.
	minX, minY, maxX, maxY int64
}

// NewArea makes an area and works out its bounding box.
func NewArea(id string, kind AreaKind, poly []*pb.Pos) Area {
	a := Area{ID: id, Kind: kind, Poly: poly}
	for i, p := range poly {
		if i == 0 || p.GetX() < a.minX {
			a.minX = p.GetX()
		}
		if i == 0 || p.GetX() > a.maxX {
			a.maxX = p.GetX()
		}
		if i == 0 || p.GetY() < a.minY {
			a.minY = p.GetY()
		}
		if i == 0 || p.GetY() > a.maxY {
			a.maxY = p.GetY()
		}
	}
	return a
}

// Contains reports whether the point is inside the area, its edge included.
func (a *Area) Contains(x, y int64) bool {
	if x < a.minX || x > a.maxX || y < a.minY || y > a.maxY {
		return false
	}
	in := false
	n := len(a.Poly)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		p, q := a.Poly[j], a.Poly[i]
		cross := (q.GetX()-p.GetX())*(y-p.GetY()) - (q.GetY()-p.GetY())*(x-p.GetX())
		if cross == 0 && x >= min(p.GetX(), q.GetX()) && x <= max(p.GetX(), q.GetX()) && y >= min(p.GetY(), q.GetY()) && y <= max(p.GetY(), q.GetY()) {
			return true // on an edge
		}
		if (p.GetY() > y) != (q.GetY() > y) {
			lhs := (x - p.GetX()) * (q.GetY() - p.GetY())
			rhs := (q.GetX() - p.GetX()) * (y - p.GetY())
			if (q.GetY() > p.GetY()) == (lhs < rhs) {
				in = !in
			}
		}
	}
	return in
}

// Ground is what the ground is at a point, by the rules of doc 02 §4a: a
// building or rock first; then a bridge or ford, then water; a road wins over a
// forest (a road through a forest is a road); then a forest; then a floor.
// The zero kind is open ground.
func (t *Terrain) Ground(x, y int64) AreaKind {
	if t == nil {
		return ""
	}
	var has [8]bool
	for i := range t.Areas {
		a := &t.Areas[i]
		if k := kindRank(a.Kind); k >= 0 && !has[k] && a.Contains(x, y) {
			has[k] = true
		}
	}
	for k, kind := range groundOrder {
		if has[k] {
			return kind
		}
	}
	return ""
}

// groundOrder lists the kinds from the one that wins to the one that loses.
var groundOrder = [8]AreaKind{Building, Rock, Bridge, Ford, Water, Road, Forest, Floor}

func kindRank(k AreaKind) int {
	for i, g := range groundOrder {
		if g == k {
			return i
		}
	}
	return -1
}

// Within is the terrain of the areas that reach the box (millimetres, corners in
// any order): what is asked about points inside the box is answered the same, from
// fewer areas.
func (t *Terrain) Within(x0, y0, x1, y1 int64) *Terrain {
	if t == nil {
		return nil
	}
	x0, x1 = min(x0, x1), max(x0, x1)
	y0, y1 = min(y0, y1), max(y0, y1)
	out := &Terrain{}
	for _, a := range t.Areas {
		if a.maxX >= x0 && a.minX <= x1 && a.maxY >= y0 && a.minY <= y1 {
			out.Areas = append(out.Areas, a)
		}
	}
	return out
}

// HasFloor says the zone is a dungeon: it has floor, which is then the only
// walkable ground.
func (t *Terrain) HasFloor() bool {
	if t == nil {
		return false
	}
	for i := range t.Areas {
		if t.Areas[i].Kind == Floor {
			return true
		}
	}
	return false
}

// TerrainOf reads the ground out of a zone map: its areas. It returns nil
// for a map with none.
func TerrainOf(zm *pb.ZoneMap) *Terrain {
	if len(zm.GetAreas()) == 0 {
		return nil
	}
	t := &Terrain{}
	for _, a := range zm.GetAreas() {
		t.Areas = append(t.Areas, NewArea(a.GetId(), AreaKind(a.GetKind()), a.GetPoly()))
	}
	return t
}

// Ground is the terrain of the zone map the character has, nil if it has none.
func (m *Model) Ground() *Terrain {
	if m.terrainOf != m.Map {
		m.terrain, m.terrainOf = TerrainOf(m.Map), m.Map
	}
	return m.terrain
}

// SpeedWords says in a few words how fast the server says the character moves,
// "forest: slowed to 75%", "road: faster, 125%"; "" when it moves at its normal
// speed or the server has not said. The server states ground and speed only in
// a snapshot, and a move does not refresh them, so they are used while the zone
// map's areas still put the character on the ground the snapshot named; once it
// has walked onto other ground, the line gives that ground and no speed until
// the next snapshot.
func (m *Model) SpeedWords() string {
	here := string(m.Underfoot())
	if here == "" {
		here = "open"
	}
	if m.Self.Ground != "" && m.Self.Ground != here {
		if here == "open" {
			return ""
		}
		return here
	}
	switch p := m.Self.SpeedPct; {
	case p == 0 || p == 100:
		return ""
	case p < 100:
		return fmt.Sprintf("%s: slowed to %d%%", here, p)
	default:
		return fmt.Sprintf("%s: faster, %d%%", here, p)
	}
}

// Underfoot is the ground the character stands on, read from the zone map's
// areas; "" for open ground. The server states the same fact as
// Self.Ground (and the speed it brings), which the status line uses.
func (m *Model) Underfoot() AreaKind {
	if m.Self.Pos == nil {
		return ""
	}
	return m.Ground().Ground(m.Self.Pos.GetX(), m.Self.Pos.GetY())
}

// moveWindowMs is the longest gap between two moves that still count as one
// walk when the speed is read off them: a heading walk moves every tick (100 ms).
const moveWindowMs = 250

// noteSelfSpeed reads the character's speed off a move that follows another
// within a walk's gap: millimetres moved over game time passed. The server does
// not say the speed in a moved event, and the snapshot's goes stale on other
// ground, so the client measures it (SelfSpeedMmS).
func (m *Model) noteSelfSpeed(to *pb.Pos, atMs int64) {
	from := m.Self.Pos
	dt := atMs - m.lastSelfMoveMs
	if from == nil || to == nil || m.lastSelfMoveMs == 0 || dt <= 0 || dt > moveWindowMs {
		return
	}
	dx, dy := float64(to.GetX()-from.GetX()), float64(to.GetY()-from.GetY())
	if d := math.Hypot(dx, dy); d > 0 {
		m.SelfSpeedMmS = int64(d * 1000 / float64(dt))
	}
}

// WalkSpeedMmS is how fast the character walks, for a client that predicts its
// own steps: the speed its last moves showed, else the snapshot's, 0 if neither.
func (m *Model) WalkSpeedMmS() int64 {
	if m.SelfSpeedMmS > 0 {
		return m.SelfSpeedMmS
	}
	return m.Self.EffectiveSpeedMmS
}

// CanStandAt says a character may stand at the point as far as the zone map
// tells: inside the bounds, not in a building, rock or water (a ford and a
// bridge are ground), and on floor in a dungeon. It is only a guide for
// prediction; the server decides.
func (m *Model) CanStandAt(x, y int64) bool {
	if b := m.Map.GetBounds(); b != nil && (b.GetMinX() != b.GetMaxX() || b.GetMinY() != b.GetMaxY()) {
		if x < b.GetMinX() || x > b.GetMaxX() || y < b.GetMinY() || y > b.GetMaxY() {
			return false
		}
	}
	t := m.Ground()
	switch k := t.Ground(x, y); k {
	case Building, Rock, Water:
		return false
	case "":
		return !t.HasFloor()
	}
	return true
}

// StepBlocked says a wall of the zone map lies between two points.
func (m *Model) StepBlocked(ax, ay, bx, by int64) bool {
	for _, w := range m.Map.GetWalls() {
		if segmentsCross(ax, ay, bx, by, w.GetA().GetX(), w.GetA().GetY(), w.GetB().GetX(), w.GetB().GetY()) {
			return true
		}
	}
	return false
}

// segmentsCross says the segment a-b meets the segment c-d.
func segmentsCross(ax, ay, bx, by, cx, cy, dx, dy int64) bool {
	side := func(px, py, qx, qy, rx, ry int64) int {
		v := (qx-px)*(ry-py) - (qy-py)*(rx-px)
		switch {
		case v > 0:
			return 1
		case v < 0:
			return -1
		}
		return 0
	}
	d1, d2 := side(ax, ay, bx, by, cx, cy), side(ax, ay, bx, by, dx, dy)
	d3, d4 := side(cx, cy, dx, dy, ax, ay), side(cx, cy, dx, dy, bx, by)
	if d1 != d2 && d3 != d4 {
		return true
	}
	on := func(px, py, qx, qy, rx, ry int64) bool {
		return rx >= min(px, qx) && rx <= max(px, qx) && ry >= min(py, qy) && ry <= max(py, qy)
	}
	return d1 == 0 && on(ax, ay, bx, by, cx, cy) || d2 == 0 && on(ax, ay, bx, by, dx, dy) ||
		d3 == 0 && on(cx, cy, dx, dy, ax, ay) || d4 == 0 && on(cx, cy, dx, dy, bx, by)
}
