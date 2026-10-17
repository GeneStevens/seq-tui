package model

import (
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Terrain is the ground of a zone (D53): areas with a kind, and the patrol
// paths of its mobs. The protocol's zone map does not carry either yet, so the
// model does not build them from messages: a TerrainSource gives them by the
// space's name (the `world` package reads them from seq-world's packs). With no
// source the map shows walls, camps and doors only, as before.
type Terrain struct {
	Areas   []Area
	Patrols []Patrol
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

// Patrol is the path a mob walks instead of wandering. Loop: it goes round
// and round; else it goes to the end and back.
type Patrol struct {
	Camp   string
	Loop   bool
	Points []*pb.Pos
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

// TerrainSource gives the terrain of a space by its name, nil if it has none.
type TerrainSource interface {
	Terrain(space string) *Terrain
}

// UseTerrain sets where the model finds the ground of the space it is in.
func (m *Model) UseTerrain(src TerrainSource) { m.terrain = src }

// Ground is the terrain of the space the character is in, nil if it has none.
func (m *Model) Ground() *Terrain {
	if m.terrain == nil || m.Space == "" {
		return nil
	}
	return m.terrain.Terrain(m.Space)
}

// Underfoot is the ground the character stands on; "" for open ground.
func (m *Model) Underfoot() AreaKind {
	if m.Self.Pos == nil {
		return ""
	}
	return m.Ground().Ground(m.Self.Pos.GetX(), m.Self.Pos.GetY())
}
