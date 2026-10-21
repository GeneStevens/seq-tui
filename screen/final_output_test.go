package screen_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/GeneStevens/seq-tui/screen"
)

var (
	ansiSeq = regexp.MustCompile("\x1b\\[[0-9;]*m")
	litSGR  = regexp.MustCompile(`\[[0-9;]+m`)
	helpAll = 12
)

// TUI-035: the final terminal output of every screen TUI-034 touched carries
// its styling as escapes only: none is printed as text (the old fit turned the
// ESC byte into "?"), and no row is wider than the screen once the escapes are
// taken out.
func TestFinalOutputLeaksNoEscapeText(t *testing.T) {
	stylers := map[string]screen.Styler{"plain": screen.Plain(), "ansi": screen.ANSI(), "mono": screen.Mono()}
	for _, shut := range []bool{false, true} {
		for _, attacked := range []bool{false, true} {
			m := world(t, !shut)
			if attacked {
				m.Self.HP, m.Self.MaxHP = 5, 10
				m.Self.InCombat, m.Self.Attackers = true, []string{"mob-1"}
			}
			views := map[string]screen.View{"map": {}, "overview": {Overview: true}, "hover": {}}
			for p := 0; p < helpAll; p++ {
				views[fmt.Sprintf("help%d", p)] = screen.View{Help: true, HelpPage: p}
			}
			for name, v := range views {
				v.Conn = screen.ConnOnline
				for sn, st := range stylers {
					for _, sz := range [][2]int{{80, 24}, {200, 50}} {
						out := screen.Render(m, v, sz[0], sz[1], st)
						id := fmt.Sprintf("%s/%s shut=%v attacked=%v %v", name, sn, shut, attacked, sz)
						plain := ansiSeq.ReplaceAllString(out, "")
						if strings.ContainsRune(plain, 0x1b) || litSGR.MatchString(plain) || strings.Contains(plain, "?[") {
							t.Errorf("%s: escape text in the output:\n%s", id, plain)
						}
						for i, l := range strings.Split(plain, "\n") {
							if n := utf8.RuneCountInString(l); n != sz[0] {
								t.Errorf("%s: row %d is %d cells, not %d: %q", id, i, n, sz[0], l)
							}
						}
						if name == "overview" && sz[0] == 200 && !strings.Contains(plain, "zone edge") {
							t.Errorf("%s: the legend is cut short:\n%s", id, plain)
						}
					}
				}
			}
		}
	}
}
