// Package screen draws the TUI: a pure function from the model and a little
// view state to a block of exactly W x H cells (seq-design doc 03 layer 5).
// It reads the model and changes nothing, so a recorded log can be replayed
// into the model and the screen compared with a golden file, with no backend
// and no terminal.
package screen

import (
	"fmt"
	"strings"
	"time"

	"github.com/GeneStevens/seq-tui/model"
)

// Smallest screen the layout is designed for.
const (
	MinWidth  = 80
	MinHeight = 24
)

// Conn is the connection as the status line says it, in words.
type Conn string

const (
	ConnOnline       Conn = "ONLINE"
	ConnReconnecting Conn = "RECONNECTING"
	ConnOffline      Conn = "OFFLINE"
	ConnLoggedOut    Conn = "LOGGED OUT"
)

// View is everything the screen shows that is not in the model.
type View struct {
	Help    bool
	Verbose bool // log routine movement and rules ticks too
	Conn    Conn
	// Since is how long ago the server last said anything (an event, a result,
	// a pong); HasTick is false before any.
	Since   time.Duration
	HasTick bool
	// Notice is the answer to the last command, shown on the status line.
	Notice    string
	NoticeBad bool // the command was denied: shown with "!" and its reason
}

// Render draws the whole screen: h lines of w cells each, joined by newlines.
func Render(m *model.Model, v View, w, h int, st Styler) string {
	if w < MinWidth || h < MinHeight {
		return tooSmall(w, h)
	}
	rw := min(max(w*3/10, 28), 44)
	lh := min(6+(h-MinHeight)/3, 14)
	th := h - lh - 1 // rows above the log
	var rows []string
	if v.Help {
		rows = box("Keys", w, th+lh, helpLines(w-2, th+lh-2, st), st)
	} else {
		mapW := w - rw
		mp := box(mapTitle(m), mapW, th, mapPanel(m, mapW-2, th-2, st), st)
		right := rightColumn(m, rw, th, st)
		for i := range mp {
			rows = append(rows, mp[i]+right[i])
		}
		rows = append(rows, box(fmt.Sprintf("Events (%s)", logMode(v)), w, lh, logLines(m, v, w-2, lh-2, st), st)...)
	}
	rows = append(rows, statusLine(m, v, w, st))
	return strings.Join(rows, "\n")
}

func tooSmall(w, h int) string {
	msg := fmt.Sprintf("Terminal too small: need %dx%d, have %dx%d", MinWidth, MinHeight, w, h)
	rows := make([]string, max(h, 1))
	for i := range rows {
		rows[i] = strings.Repeat(" ", max(w, 0))
	}
	if w > 0 {
		rows[0] = fit(msg, w)
	}
	return strings.Join(rows, "\n")
}

func logMode(v View) string {
	if v.Verbose {
		return "all"
	}
	return "v: all"
}

func mapTitle(m *model.Model) string {
	switch {
	case m.Map != nil && m.Map.GetName() != "":
		return m.Map.GetName() + ", north up"
	case m.Space != "":
		return m.Space
	}
	return "Map"
}

func mapPanel(m *model.Model, cw, ch int, st Styler) []string {
	if m.Map == nil || m.Self.Pos == nil {
		rows := make([]string, ch)
		for i := range rows {
			rows[i] = strings.Repeat(" ", cw)
		}
		rows[ch/2] = fit("  waiting for the map…", cw)
		return rows
	}
	return mapRows(m, cw, ch, st)
}

// box draws a titled frame of w x h cells around inner, whose lines are each
// exactly w-2 cells wide once styling is ignored.
func box(title string, w, h int, inner []string, st Styler) []string {
	top := []rune("┌─ " + strings.TrimSpace(fit(title, w-6)) + " " + strings.Repeat("─", w))
	rows := []string{st.Dim(string(top[:w-1]) + "┐")}
	side := st.Dim("│")
	for i := 0; i < h-2; i++ {
		line := strings.Repeat(" ", w-2)
		if i < len(inner) {
			line = inner[i]
		}
		rows = append(rows, side+line+side)
	}
	return append(rows, st.Dim("└"+strings.Repeat("─", w-2)+"┘"))
}

func helpLines(w, h int, st Styler) []string {
	keys := []string{
		"Move             arrow keys or W A S D",
		"Cycle target     Tab (Esc clears the target)",
		"Auto-attack      F or Space: on / off",
		"Loot             L: the nearest drop you may take",
		"Use a door       E: the nearest door; stand at it first",
		"Log out          O: leave the world",
		"Quit             Q or Ctrl+C: leave the screen, stay in the world",
		"Event log        V: also show movement and rules ticks",
		"Help             ?",
		"",
		"Map: north is up, your position is always the centre.",
		"  @ you     m mob     M your target     x dead mob",
		"  p another character     * a drop     D a door     # wall",
		"  Nearby lists the same things nearest first, with a distance",
		"  and a compass point, so nothing depends on colour.",
		"",
		"Event log markers:  ! denied (with the reason)   < a hit on you",
		"                    x a kill   + a reward",
	}
	rows := make([]string, h)
	for i := range rows {
		s := ""
		if i < len(keys) {
			s = keys[i]
		}
		rows[i] = fit(" "+s, w)
	}
	return rows
}

func statusLine(m *model.Model, v View, w int, st Styler) string {
	heard := "nothing heard yet"
	if v.HasTick {
		heard = fmt.Sprintf("heard %.1fs ago", v.Since.Seconds())
		if v.Since > 5*time.Second {
			heard += " (SILENT)"
		}
	}
	space := m.Space
	if space == "" {
		space = "no space"
	}
	note := v.Notice
	if note != "" && v.NoticeBad {
		note = "! " + note
	}
	// The line is the connection, the space, the clock and the last command's
	// answer. When it is short, the key hint goes first, then the space and the
	// clock: a denial's reason is never cut for them.
	var tries [][]string
	for _, parts := range [][]string{{string(v.Conn), space, heard, note}, {string(v.Conn), heard, note}, {string(v.Conn), note}} {
		var kept []string
		for _, p := range parts {
			if p != "" {
				kept = append(kept, p)
			}
		}
		tries = append(tries, kept)
	}
	for _, parts := range tries {
		left := " " + strings.Join(parts, " | ")
		for _, hint := range []string{"? help  q quit", "? help", ""} {
			if width(left)+1+width(hint) <= w {
				return st.Reverse(fit(fit(left, w-width(hint)-1)+" "+hint, w))
			}
		}
	}
	return st.Reverse(fit(" "+string(v.Conn)+" | "+note, w))
}
