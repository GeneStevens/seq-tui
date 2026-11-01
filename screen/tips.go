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
// keyboard cursor is on, each beside its cell. Over a thing with a label of its
// own (a person, a mob, a door) the pointer shows that label, drawn with the
// others; the cursor, which has none, says the thing too, then the places.
func paintTips(g *grid, m *model.Model, v View) {
	type spot struct {
		c, r   int64
		cursor bool
	}
	var spots []spot
	if v.Hits != nil && v.Hover != nil {
		spots = append(spots, spot{int64(v.Hover.X - v.Hits.org.X), int64(v.Hover.Y - v.Hits.org.Y), false})
	}
	if v.Cursor != nil {
		c, r := g.cellOf(v.Cursor)
		spots = append(spots, spot{c, r, true})
	}
	if len(spots) == 0 {
		return
	}
	ts := things(g, m)
	used := map[[2]int64]bool{}
	for _, at := range spots {
		var parts []string
		if t, ok := thingAt(ts, at.c, at.r); ok {
			if !at.cursor {
				continue
			}
			parts = append(parts, t.label)
		}
		if parts = append(parts, g.placesAt(at.c, at.r)...); len(parts) > 0 {
			g.tip(used, at.c, at.r, strings.Join(parts, tipSep))
		}
	}
}

// tip writes a label for the cell (c, r): beside it, right then left, else on
// the rows above or below, kept inside the map and off the names, the character
// and the cursor. A tip longer than the map is cut.
func (g *grid) tip(used map[[2]int64]bool, c, r int64, tip string) {
	text := " " + strings.TrimRight(fit(tip, max(g.cw-2, 1)), " ") + " "
	w := int64(width(text))
	mid := min(max(c-w/2, 0), int64(g.cw)-w)
	for _, at := range [][2]int64{{c + 1, r}, {c - w, r}, {mid, r - 1}, {mid, r + 1}, {mid, r - 2}, {mid, r + 2}} {
		if g.labelRoom(used, at[0], at[1], int(w)) && g.clearOfNames(at[0], at[1], int(w)) {
			g.text(at[0], at[1], text, kLabel)
			for i := int64(0); i < w; i++ {
				used[[2]int64{at[0] + i, at[1]}] = true
			}
			return
		}
	}
}

// clearOfNames says w cells from (c, r) hold no name, nor the character, nor the
// cursor: a tip does not cover them.
func (g *grid) clearOfNames(c, r int64, w int) bool {
	for i := 0; i < w; i++ {
		switch g.cells[r][c+int64(i)].k {
		case kPlace, kSelf, kCursor, kLabel:
			return false
		}
	}
	return true
}
