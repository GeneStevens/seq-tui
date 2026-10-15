package screen

import (
	"fmt"
	"strings"
)

// fit truncates s to n cells (with an ellipsis) and pads it with spaces to n.
// Every glyph the screen draws is one cell wide; control characters in
// server-supplied text are replaced so they cannot move the cursor.
func fit(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(clean(s))
	if len(r) > n {
		if n == 1 {
			return "…"
		}
		r = append(r[:n-1], '…')
	}
	return string(r) + strings.Repeat(" ", n-len(r))
}

func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			return '?'
		}
		return r
	}, s)
}

func width(s string) int { return len([]rune(s)) }

// bar draws a gauge of n cells for cur of max: "#####-----".
func bar(cur, max int64, n int) string {
	if n < 1 {
		return ""
	}
	fill := 0
	if max > 0 {
		fill = int(min(max, cur) * int64(n) / max)
		if cur > 0 && fill == 0 {
			fill = 1
		}
	}
	return strings.Repeat("#", fill) + strings.Repeat("-", n-fill)
}

// bearing names the direction of (dx, dy) as a compass point; y grows north.
func bearing(dx, dy int64) string {
	if dx == 0 && dy == 0 {
		return "here"
	}
	// eight 45-degree sectors, by integer comparison against tan(22.5°) ≈ 0.4142
	ax, ay := abs(dx), abs(dy)
	var h, v string
	if dx > 0 {
		h = "E"
	} else if dx < 0 {
		h = "W"
	}
	if dy > 0 {
		v = "N"
	} else if dy < 0 {
		v = "S"
	}
	switch {
	case ay*1000 <= ax*414:
		return h
	case ax*1000 <= ay*414:
		return v
	}
	return v + h
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func isqrt(n int64) int64 {
	if n <= 0 {
		return 0
	}
	x := n
	y := (x + 1) / 2
	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return x
}

// dist is the ground distance in millimetres.
func dist(ax, ay, bx, by int64) int64 {
	dx, dy := ax-bx, ay-by
	return isqrt(dx*dx + dy*dy)
}

func metres(mm int64) string { return fmt.Sprintf("%dm", (mm+500)/1000) }
