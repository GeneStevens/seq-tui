package model

import (
	"cmp"
	"math"
	"slices"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Significance is how much a named place matters to the player (D121), the order
// the large map names and groups places in: the current objective first, then a
// quest person, then any other NPC, then a door or a building, then size alone.
const (
	SigSize = iota
	SigDoorOrBuilding
	SigNPC
	SigQuestPerson
	SigObjective
)

// ClusterGapMm is the cluster test (D121): the pieces of a place (its areas of
// one name) are one cluster when each can be reached from any other by hops of no
// more than this gap between their bounding boxes, 40 m. Eldbury Ring's four
// banks stand 20 to 33 m apart and so are one ring; two copses a field apart are
// two clusters.
const ClusterGapMm = 40_000

// nearMm is how close to a piece an NPC, a door or the objective must be to
// belong to its place: standing at its edge counts.
const nearMm = 4_000

// PlaceLabel is a named place of the zone and where its name is written on the
// large map (D120, D121): every area of the zone with the same name is one
// place. When its pieces form one cluster (ClusterGapMm) the label is the middle
// of their combined bounding box and Size, DX and DY are the box's; otherwise
// the place is scattered and it is named beside its largest piece, and they are
// that piece's.
type PlaceLabel struct {
	Name      string
	Pos       *pb.Pos
	Size      int64 // the area of the box, square millimetres
	DX, DY    int64 // the box's width and height, millimetres
	Areas     int   // how many named areas make up the place
	Scattered bool  // the pieces do not form one cluster
	Sig       int   // how much it matters; see Sig*
	Why       string
	Polys     [][]*pb.Pos // the pieces
	Kind      AreaKind    // the kind of its largest piece
}

type box struct{ x0, x1, y0, y1 int64 }

func (b box) grow(x, y int64) box {
	return box{min(b.x0, x), max(b.x1, x), min(b.y0, y), max(b.y1, y)}
}

func (b box) area() int64 { return (b.x1 - b.x0) * (b.y1 - b.y0) }

// gap is the distance between two boxes, 0 when they touch or overlap.
func (b box) gap(o box) float64 {
	dx := max(b.x0-o.x1, o.x0-b.x1, 0)
	dy := max(b.y0-o.y1, o.y0-b.y1, 0)
	return math.Hypot(float64(dx), float64(dy))
}

func polyBox(poly []*pb.Pos) box {
	p0 := poly[0]
	b := box{p0.GetX(), p0.GetX(), p0.GetY(), p0.GetY()}
	for _, p := range poly {
		b = b.grow(p.GetX(), p.GetY())
	}
	return b
}

// clusters splits the boxes into the groups that reach each other by hops of at
// most ClusterGapMm (single linkage), and returns each box's group number.
func clusters(boxes []box) (group []int, n int) {
	parent := make([]int, len(boxes))
	for i := range parent {
		parent[i] = i
	}
	find := func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	for i := range boxes {
		for j := i + 1; j < len(boxes); j++ {
			if boxes[i].gap(boxes[j]) <= ClusterGapMm {
				parent[find(j)] = find(i)
			}
		}
	}
	group = make([]int, len(boxes))
	seen := map[int]int{}
	for i := range boxes {
		r := find(i)
		if _, ok := seen[r]; !ok {
			seen[r] = len(seen)
		}
		group[i] = seen[r]
	}
	return group, len(seen)
}

// PlaceLabels are the named places of the zone map (D76): streets, squares,
// buildings, gates and fields, one for each name, in the order each first
// appears, with how much each matters (D121).
func (m *Model) PlaceLabels() []PlaceLabel {
	var order []string
	pieces := map[string][][]*pb.Pos{}
	kinds := map[string][]AreaKind{}
	for _, a := range m.Map.GetAreas() {
		if a.GetName() == "" || len(a.GetPoly()) == 0 {
			continue
		}
		if _, ok := pieces[a.GetName()]; !ok {
			order = append(order, a.GetName())
		}
		pieces[a.GetName()] = append(pieces[a.GetName()], a.GetPoly())
		kinds[a.GetName()] = append(kinds[a.GetName()], AreaKind(a.GetKind()))
	}
	out := make([]PlaceLabel, 0, len(order))
	for _, name := range order {
		polys := pieces[name]
		boxes := make([]box, len(polys))
		whole := polyBox(polys[0])
		largest := 0
		for i, p := range polys {
			boxes[i] = polyBox(p)
			whole = whole.grow(boxes[i].x0, boxes[i].y0).grow(boxes[i].x1, boxes[i].y1)
			if boxes[i].area() > boxes[largest].area() {
				largest = i
			}
		}
		_, n := clusters(boxes)
		at := whole
		if n > 1 {
			at = boxes[largest] // scattered: beside its largest piece
		}
		pl := PlaceLabel{Name: name, Pos: &pb.Pos{X: (at.x0 + at.x1) / 2, Y: (at.y0 + at.y1) / 2},
			Size: at.area(), DX: at.x1 - at.x0, DY: at.y1 - at.y0, Areas: len(polys), Scattered: n > 1,
			Polys: polys, Kind: kinds[name][largest]}
		pl.Sig, pl.Why = m.significance(&pl, kinds[name])
		out = append(out, pl)
	}
	return out
}

// travelled says a kind of ground is one a person follows or crosses: a road, water,
// a ford or a bridge. Standing on one makes no place important; they are named by size.
func travelled(k AreaKind) bool {
	return k == Road || k == Water || k == Ford || k == Bridge
}

// significance is how much the place matters: the objective, a quest person or
// other NPC, a door or a building (D121). Size is the tie-break and is not here.
func (m *Model) significance(pl *PlaceLabel, kinds []AreaKind) (sig int, why string) {
	raise := func(s int, w string) {
		if s > sig {
			sig, why = s, w
		}
	}
	// near says a point belongs to a piece that is not a road or water.
	near := func(p *pb.Pos) bool {
		for i, poly := range pl.Polys {
			if !travelled(kinds[i]) && nearPoly(p, poly, nearMm) {
				return true
			}
		}
		return false
	}
	for _, k := range kinds {
		if k == Building {
			raise(SigDoorOrBuilding, "building")
		}
	}
	for _, d := range m.Map.GetDoors() {
		if near(d.GetPos()) {
			raise(SigDoorOrBuilding, "door")
		}
	}
	for _, k := range m.Map.GetMarkers() {
		if (k.GetKind() == "zone_line" || k.GetKind() == "instance_entrance") && near(k.GetPos()) {
			raise(SigDoorOrBuilding, "door")
		}
	}
	for _, n := range m.KnownNPCs() {
		if !near(n.GetPos()) {
			continue
		}
		if n.GetRole() == "quest_giver" || m.MarkOf(n.GetId()) != "" {
			raise(SigQuestPerson, "quest person")
		} else {
			raise(SigNPC, "person")
		}
	}
	if h := m.Objective(); h != nil && h.GetPos() != nil && near(h.GetPos()) {
		raise(SigObjective, "objective")
	}
	return sig, why
}

// RankPlaces sorts places most significant first: by Sig, then size.
func RankPlaces(places []PlaceLabel) {
	slices.SortStableFunc(places, func(a, b PlaceLabel) int {
		return cmp.Or(cmp.Compare(b.Sig, a.Sig), cmp.Compare(b.Size, a.Size))
	})
}

// nearPoly says p is inside the polygon or within d of its edge.
func nearPoly(p *pb.Pos, poly []*pb.Pos, d int64) bool {
	if inPoly(p, poly) {
		return true
	}
	for i := range poly {
		a, b := poly[i], poly[(i+1)%len(poly)]
		if segDist(p, a, b) <= float64(d) {
			return true
		}
	}
	return false
}

// inPoly is the even-odd rule.
func inPoly(p *pb.Pos, poly []*pb.Pos) bool {
	in := false
	x, y := float64(p.GetX()), float64(p.GetY())
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		xi, yi, xj, yj := float64(poly[i].GetX()), float64(poly[i].GetY()), float64(poly[j].GetX()), float64(poly[j].GetY())
		if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			in = !in
		}
	}
	return in
}

func segDist(p, a, b *pb.Pos) float64 {
	px, py, ax, ay, bx, by := float64(p.GetX()), float64(p.GetY()), float64(a.GetX()), float64(a.GetY()), float64(b.GetX()), float64(b.GetY())
	dx, dy := bx-ax, by-ay
	t := 0.0
	if l := dx*dx + dy*dy; l > 0 {
		t = min(max(((px-ax)*dx+(py-ay)*dy)/l, 0), 1)
	}
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
}

// Touches says some piece of the place lies in the rectangle [x0,x1] x [y0,y1]
// (millimetres): a vertex inside it, its middle inside the piece, or an edge
// crossing it. It is how a cell of the large map says it is over the place.
func (pl *PlaceLabel) Touches(x0, y0, x1, y1 int64) bool {
	in := func(p *pb.Pos) bool { return p.GetX() >= x0 && p.GetX() <= x1 && p.GetY() >= y0 && p.GetY() <= y1 }
	mid := &pb.Pos{X: (x0 + x1) / 2, Y: (y0 + y1) / 2}
	for _, poly := range pl.Polys {
		if inPoly(mid, poly) {
			return true
		}
		for i := range poly {
			a, b := poly[i], poly[(i+1)%len(poly)]
			if in(a) || segHitsRect(a, b, x0, y0, x1, y1) {
				return true
			}
		}
	}
	return false
}

// segHitsRect is Liang-Barsky: the segment ab meets the rectangle.
func segHitsRect(a, b *pb.Pos, x0, y0, x1, y1 int64) bool {
	ax, ay := float64(a.GetX()), float64(a.GetY())
	dx, dy := float64(b.GetX())-ax, float64(b.GetY())-ay
	t0, t1 := 0.0, 1.0
	for _, c := range [][2]float64{{-dx, ax - float64(x0)}, {dx, float64(x1) - ax}, {-dy, ay - float64(y0)}, {dy, float64(y1) - ay}} {
		p, q := c[0], c[1]
		if p == 0 {
			if q < 0 {
				return false
			}
			continue
		}
		r := q / p
		if p < 0 {
			t0 = max(t0, r)
		} else {
			t1 = min(t1, r)
		}
		if t0 > t1 {
			return false
		}
	}
	return true
}
