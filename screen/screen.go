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

	// Menu is the instance menu, nil when it is closed. Plan is the plan
	// being built in a turn-based instance. Now is the client's clock, for the
	// countdown to a round's deadline (a date from the server).
	Menu *Menu
	Plan Plan
	Now  time.Time

	// Slots is the slot screen and Create the character creation screen, nil
	// when closed. GameNow is the game time the cooldowns and the cast bar read
	// (0: the last time the server said).
	Slots   *Slots
	Create  *Create
	GameNow int64

	// PlanSlots are the slot changes the plan being built holds (a turn-based
	// instance changes slots between rounds, with the plan): slot 1 to 8 to the
	// ability it will hold, "" for emptied.
	PlanSlots map[int]string

	// Inventory, Traders and Trade are the inventory and gear screen, the
	// traders' screen and the trade window, nil when closed.
	Inventory *Inventory
	Traders   *Traders
	Trade     *TradeUI
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
	switch {
	case v.Create != nil:
		rows = box("New character", w, th+lh, createLines(m, v.Create, w-2, th+lh-2, st), st)
	case v.Help:
		rows = box("Keys", w, th+lh, helpLines(w-2, th+lh-2, st), st)
	case v.Slots != nil:
		rows = box("Ability slots", w, th+lh, slotScreenLines(m, v, w-2, th+lh-2, st), st)
	case v.Inventory != nil:
		rows = box("Inventory and gear", w, th+lh, inventoryScreenLines(m, v.Inventory, w-2, th+lh-2, st), st)
	case v.Traders != nil:
		rows = box("Traders", w, th+lh, tradersScreenLines(m, v.Traders, w-2, th+lh-2, st), st)
	case v.Trade != nil:
		rows = box("Trade", w, th+lh, tradeScreenLines(m, v.Trade, w-2, th+lh-2, st), st)
	case v.Menu != nil:
		rows = box("Instances", w, th+lh, menuLines(m, v, w-2, th+lh-2, st), st)
	default:
		mapW := w - rw
		if m.HasKit() {
			th -= abilityRows
		}
		mp := box(mapTitle(m), mapW, th, mapPanel(m, v, mapW-2, th-2, st), st)
		right := rightColumn(m, v, rw, th, st)
		for i := range mp {
			rows = append(rows, mp[i]+right[i])
		}
		if m.HasKit() {
			rows = append(rows, box("Abilities (keys 1-8, B sets the slots)", w, abilityRows, abilityLines(m, v, w-2, st), st)...)
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

func mapPanel(m *model.Model, v View, cw, ch int, st Styler) []string {
	if m.Map == nil || m.Self.Pos == nil {
		rows := make([]string, ch)
		for i := range rows {
			rows[i] = strings.Repeat(" ", cw)
		}
		rows[ch/2] = fit("  waiting for the map…", cw)
		return rows
	}
	return mapRows(m, v, cw, ch, st)
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
		"Target, attack   Tab cycles the target (Esc clears); F or Space: auto-attack on / off",
		"Loot, doors      L: the nearest drop you may take; E: the nearest door (stand at it)",
		"Release          R: when you are dead, give up your body and respawn",
		"Abilities        1-8 use that slot on your target; B sets slots (out of combat)",
		"Instances        I: this zone's menu (Enter opens one); then E at its door",
		"Gear, traders    C: wear and take off gear; N: buy, sell, train, bind, bank",
		"Trade            T: ask a character in reach; each side offers, then confirms",
		"Plans            in a turn-based instance the keys build a plan, not act:",
		"                 arrows, Tab target, F attack, 1-8 ability, L loot, E door,",
		"                 X stop, R release; B sets slots (they change when the round opens).",
		"                 One move or door, one ability or attack. Enter submits it,",
		"                 Backspace removes a step, Esc clears it",
		"Leave            O logs out of the world; Q or Ctrl+C quits the screen only",
		"Event log        V: also show movement, rules ticks and mana      Help  ?",
		"Map: north is up, your position is always the centre.",
		"  @ you  m mob  M your target  x dead mob  + end of your plan  p another character",
		"  * a drop  ~ a drop you may not take  $ a trader  D a door  # wall",
		"  Nearby lists the same, nearest first, with a distance and a compass point.",
		"Event log markers:  ! denied (with the reason) or a cast that did not land",
		"                    < a hit on you   x a kill   + a reward or a level",
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
