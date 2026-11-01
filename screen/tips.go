package screen

import (
	"strings"

	"github.com/GeneStevens/seq-tui/model"
)

// tipSep joins the names in a tip.
const tipSep = " · "

// placesAt are the names of the places at the cell (c, r) of the large map,
// most significant first (D121): those whose ground lies in the cell, and the
// places left out that are counted under a name when the cell is on that name.
func (g *grid) placesAt(c, r int64) []string {
	if c < 0 || r < 0 || c >= int64(g.cw) || r >= int64(g.ch) {
		return nil
	}
	mid := g.worldOf(c, r)
	x0, x1, y0, y1 := mid.GetX()-g.sc.col/2, mid.GetX()+g.sc.col/2, mid.GetY()-g.sc.row/2, mid.GetY()+g.sc.row/2
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	for _, n := range g.named {
		if n.r == r && c >= n.c && c < n.c+int64(n.w) {
			add(n.name)
			for _, h := range n.hidden {
				add(h)
			}
		}
	}
	for i := range g.places {
		if p := &g.places[i]; p.Touches(x0, y0, x1, y1) {
			add(p.Name)
		}
	}
	return out
}

// TipAt is the one-line tip for the screen cell (x, y) of the large map: the
// places there, "Goblin Hut · Goblin Rocks", "" when there is none or the cell
// is not on the large map (D121).
func (h *Hits) TipAt(x, y int) string {
	if h == nil || h.g == nil || h.g.places == nil || !h.Map.In(x, y) {
		return ""
	}
	return strings.Join(h.g.placesAt(int64(x-h.Map.X), int64(y-h.Map.Y)), tipSep)
}

// paintTips draws the tip of the cell the pointer is over and of the cell the
// keyboard cursor is on, each beside its cell. A cell with something on it that
// has a label of its own (a person, a mob, a door) shows that label instead.
func paintTips(g *grid, m *model.Model, v View) {
	var cells [][2]int64
	if v.Hits != nil && v.Hover != nil {
		cells = append(cells, [2]int64{int64(v.Hover.X - v.Hits.org.X), int64(v.Hover.Y - v.Hits.org.Y)})
	}
	if v.Cursor != nil {
		c, r := g.cellOf(v.Cursor)
		cells = append(cells, [2]int64{c, r})
	}
	if len(cells) == 0 {
		return
	}
	ts := things(g, m)
	var used map[[2]int64]bool
	for _, at := range cells {
		if _, ok := thingAt(ts, at[0], at[1]); ok {
			continue
		}
		tip := strings.Join(g.placesAt(at[0], at[1]), tipSep)
		if tip == "" {
			continue
		}
		if used == nil {
			used = map[[2]int64]bool{}
		}
		g.tip(used, at[0], at[1], tip)
	}
}

// tip writes a label for the cell (c, r): beside it, right then left, else on
// the row above or below, kept inside the map. A tip longer than the map is cut.
func (g *grid) tip(used map[[2]int64]bool, c, r int64, tip string) {
	text := " " + strings.TrimRight(fit(tip, max(g.cw-2, 1)), " ") + " "
	w := int64(width(text))
	mid := min(max(c-w/2, 0), int64(g.cw)-w)
	for _, at := range [][2]int64{{c + 1, r}, {c - w, r}, {mid, r - 1}, {mid, r + 1}} {
		if g.labelRoom(used, at[0], at[1], int(w)) {
			g.text(at[0], at[1], text, kLabel)
			for i := int64(0); i < w; i++ {
				used[[2]int64{at[0] + i, at[1]}] = true
			}
			return
		}
	}
}
