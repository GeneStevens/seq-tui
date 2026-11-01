package screen

import "testing"

// window settles a scroll position against the rows of a tab: the rows 1,1,2,3,3,3,4,5
// are five lines, the third taking three rows.
func TestScrollWindow(t *testing.T) {
	seqs := []int{1, 1, 2, 3, 3, 3, 4, 5}
	for _, tc := range []struct {
		name         string
		pos          ScrollPos
		h            int
		lo, hi, new  int
		anchor, back int
	}{
		{"live shows the newest", ScrollPos{}, 3, 5, 8, 0, 0, 0},
		{"live with room to spare", ScrollPos{}, 20, 0, 8, 0, 0, 0},
		{"one row back", ScrollPos{Anchor: scrollNewest, Back: 1}, 3, 4, 7, 1, 4, 0},
		{"back into the middle of a line", ScrollPos{Anchor: scrollNewest, Back: 4}, 3, 1, 4, 2, 3, 2},
		{"home stops at the first page", ScrollPos{Anchor: scrollNewest, Back: 1 << 30}, 3, 0, 3, 3, 2, 0},
		{"forward past the end is live", ScrollPos{Anchor: 4, Back: -5}, 3, 5, 8, 0, 0, 0},
		{"the anchor line was cut off the old end", ScrollPos{Anchor: 0 + 1, Back: 9}, 3, 0, 3, 3, 2, 0},
	} {
		p := tc.pos
		lo, hi, n := p.window(seqs, tc.h)
		if lo != tc.lo || hi != tc.hi || n != tc.new || p.Anchor != tc.anchor || p.Back != tc.back {
			t.Errorf("%s: rows %d-%d, %d new, pos %+v; want %d-%d, %d, anchor %d back %d", tc.name, lo, hi, n, p, tc.lo, tc.hi, tc.new, tc.anchor, tc.back)
		}
	}
}

// Lines that arrive while the panel is scrolled back leave the rows where they are.
func TestScrolledBackStaysPutAsLinesArrive(t *testing.T) {
	seqs := []int{1, 2, 3, 4, 5, 6}
	p := ScrollPos{}
	p.Scroll(2)
	lo, hi, _ := p.window(seqs, 3)
	for i := 7; i <= 9; i++ {
		seqs = append(seqs, i)
	}
	lo2, hi2, fresh := p.window(seqs, 3)
	if lo2 != lo || hi2 != hi || fresh != 5 {
		t.Errorf("rows %d-%d, %d new after three arrived; were %d-%d, want 5 new", lo2, hi2, fresh, lo, hi)
	}
	p.Reset()
	if lo, hi, _ = p.window(seqs, 3); hi != len(seqs) || !p.Live() {
		t.Errorf("Reset is not live: rows %d-%d", lo, hi)
	}
}

func TestScrollMarkIsPlainText(t *testing.T) {
	for n, want := range map[int]string{0: "↓ more below · End", 12: "↓ 12 new · End"} {
		if got := scrollMark(n); got != want {
			t.Errorf("scrollMark(%d) = %q, want %q", n, got, want)
		}
	}
}
