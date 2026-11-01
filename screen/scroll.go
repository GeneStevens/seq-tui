package screen

import (
	"fmt"
	"math"
	"sort"
)

// ScrollPos is how far back a tab of the bottom panel is scrolled. A zero one is
// live: the newest lines show. Otherwise the panel's last row is the one Back rows
// before the end of the line numbered Anchor, so lines that arrive (or are cut off
// the old end) never move it. The app turns it by Scroll, Home and End; drawing
// settles it, so it always names a line that is kept.
type ScrollPos struct{ Anchor, Back int }

// scrollNewest is an Anchor that names the newest line, whatever its number.
const scrollNewest = math.MaxInt

// Live says the tab shows the newest lines.
func (p ScrollPos) Live() bool { return p.Anchor == 0 }

// Scroll moves the view back by rows (up, positive) or forward (down, negative).
func (p *ScrollPos) Scroll(rows int) {
	if p.Live() {
		if rows <= 0 {
			return
		}
		p.Anchor = scrollNewest
	}
	p.Back += rows
}

// Oldest scrolls to the oldest line kept (Home).
func (p *ScrollPos) Oldest() { *p = ScrollPos{Anchor: scrollNewest, Back: math.MaxInt32} }

// Reset returns the tab to the newest lines (End).
func (p *ScrollPos) Reset() { *p = ScrollPos{} }

// window settles p against the rows of a tab, whose line numbers are seqs (rising),
// for a body h rows high: the rows lo to hi show, and fresh counts the lines past
// the last one in view. A nil or live p shows the newest h.
func (p *ScrollPos) window(seqs []int, h int) (lo, hi, fresh int) {
	n := len(seqs)
	if p == nil || p.Live() || n == 0 {
		if p != nil && n == 0 {
			p.Reset()
		}
		return max(n-h, 0), n, 0
	}
	end := sort.Search(n, func(i int) bool { return seqs[i] > p.Anchor })
	bottom := end - p.Back
	if bottom >= n {
		p.Reset()
		return max(n-h, 0), n, 0
	}
	bottom = max(bottom, min(h, n))
	a := seqs[bottom-1]
	endA := sort.Search(n, func(i int) bool { return seqs[i] > a })
	p.Anchor, p.Back = a, endA-bottom
	for i := endA; i < n; i++ {
		if i == endA || seqs[i] != seqs[i-1] {
			fresh++
		}
	}
	return max(bottom-h, 0), bottom, fresh
}

// scrollMark is the words on the panel's border while it is scrolled back. They
// are plain text: the mark never rests on colour.
func scrollMark(fresh int) string {
	if fresh > 0 {
		return fmt.Sprintf("↓ %d new · End", fresh)
	}
	return "↓ more below · End"
}

// pos is the scroll of the bottom panel's tab t, nil (live) when the view has none.
func (v View) pos(t int) *ScrollPos {
	if v.Back == nil || t < 0 || t >= PanelTabs {
		return nil
	}
	return &v.Back[t]
}
