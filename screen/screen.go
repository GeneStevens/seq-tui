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

// panelMinWidth is the narrowest a service panel may be, beside the You,
// Target and Nearby panels; below it the service takes the whole width.
const panelMinWidth = 60

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
	ConnReplay       Conn = "REPLAY"
)

// View is everything the screen shows that is not in the model.
type View struct {
	Help bool
	// HelpPage is the page of the help that is open, HelpScroll how far it is
	// scrolled.
	HelpPage, HelpScroll int
	Verbose              bool // log routine movement and rules ticks too
	Conn                 Conn
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

	// Talk, Board and Contracts are the conversation, the contract board and
	// the Contracts screen, nil when closed; Overview shows the whole zone.
	Talk      *Talk
	Board     *Board
	Contracts *ContractsUI
	Overview  bool

	// Role is the screen for choosing the recruit's role, nil when closed. Duo is
	// the account's second character on a stance, nil when there is none, and
	// KeysToDuo says the companion keys talk to it rather than to the house
	// companion (when the character has both).
	Role      *Role
	Duo       *DuoView
	KeysToDuo bool

	// Directing is the name of the second character while the keys and the
	// screen are its (D56): the screen is then its own, and the status line says
	// whom the player is directing.
	Directing string

	// Line is the command line, Social the Social screen: nil when closed.
	Line   *Line
	Social *SocialUI

	// Sessions is the sessions view (D64), nil when closed. Quest is the quest
	// chain screen (D62), nil when closed.
	Sessions *SessionsUI
	Quest    *QuestUI
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
	if b := eventBanner(m, v, w, st); b != "" {
		rows, th = append(rows, b), th-1
	}
	if b := inviteBanner(m, w, st); b != "" {
		rows, th = append(rows, b), th-1
	}
	if b := momentBanner(m, v, w, st); len(b) > 0 {
		rows, th = append(rows, b...), th-len(b)
	}
	// A screen that takes the whole window keeps one line of vitals (D75).
	strip := m.Self.ID != ""
	takeover := func(title string, lines func(w, h int) []string) {
		bh := th + lh
		if strip {
			bh--
		}
		body := lines(w-2, bh-2)
		if !v.Help {
			body = withNotice(body, v, w-2, st)
		}
		rows = append(rows, boxA(title, w, bh, body, st, alarmed(m))...)
		if strip {
			rows = append(rows, vitalsStrip(m, w, st))
		}
	}
	switch {
	case v.Help:
		takeover(HelpTitle(v.HelpPage), func(w, h int) []string { return helpLines(m, v, w, h, st) })
	case v.Sessions != nil:
		title := "Sessions: your logs"
		if v.Sessions.Log != nil {
			title = "Sessions: the server's log"
		}
		rows = append(rows, box(title, w, th+lh, sessionsScreenLines(v.Sessions, w-2, th+lh-2, st), st)...)
	case v.Quest != nil:
		takeover("Quest chain", func(w, h int) []string { return questScreenLines(m, v.Quest, w, h, st) })
	case v.Create != nil:
		rows = append(rows, box("New character", w, th+lh, createLines(m, v.Create, w-2, th+lh-2, st), st)...)
	case v.Social != nil:
		takeover("Social: "+tabNames[v.Social.Tab], func(w, h int) []string { return socialScreenLines(m, v.Social, w, h, st) })
	case v.Slots != nil:
		takeover("Ability slots", func(w, h int) []string { return slotScreenLines(m, v, w, h, st) })
	case v.Role != nil:
		takeover("Recruit's role", func(w, h int) []string { return roleLines(m, v.Role, w, h, st) })
	case v.Overview && m.Map != nil && m.Self.Pos != nil:
		takeover("Zone overview", func(w, h int) []string { return overviewLines(m, w, h, st) })
	case v.Menu != nil:
		takeover("Instances", func(w, h int) []string { return menuLines(m, v, w, h, st) })
	case serviceOpen(v) && w-rw-2 < panelMinWidth:
		// Too narrow for a panel beside the other panels: the service takes the
		// width, and keeps the vitals strip and the alarm (D75).
		bh := th + lh - 1
		title, lines, _ := servicePanel(m, v, w-2, bh-2, st)
		rows = append(rows, boxA(title, w, bh, lines, st, alarmed(m))...)
		rows = append(rows, vitalsStrip(m, w, st))
	default:
		mapW := w - rw
		if title, lines, ok := servicePanel(m, v, mapW-2, th-2, st); ok {
			// Services are panels in the map's area (D75): the You, Target and
			// Nearby panels stay beside them, and the log below.
			mp := boxA(title, mapW, th, lines, st, alarmed(m))
			right := rightColumn(m, v, rw, th, st)
			for i := range mp {
				rows = append(rows, mp[i]+right[i])
			}
		} else {
			th-- // the line of key hints
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
		}
		evh := lh
		if chatShown(m, v) {
			// Chat takes the right part of the log's row once anything has been said.
			cw := min(max(w*5/12, 30), 60)
			ev := box(fmt.Sprintf("Events (%s)", logMode(v)), w-cw, evh, logLines(m, v, w-cw-2, evh-2, st), st)
			ch := box("Chat (/ types)", cw, evh, chatLines(m, cw-2, evh-2, st), st)
			for i := range ev {
				rows = append(rows, ev[i]+ch[i])
			}
		} else {
			rows = append(rows, box(fmt.Sprintf("Events (%s)", logMode(v)), w, evh, logLines(m, v, w-2, evh-2, st), st)...)
		}
		if !v.anyScreen() {
			rows = append(rows, hintLine(m, v, w, st))
		}
	}
	if v.Line != nil {
		rows = append(rows, lineRow(v.Line, w, st))
	} else {
		rows = append(rows, statusLine(m, v, w, st))
	}
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
	case m.Map != nil && m.Map.GetName() != "" && m.Self.Pos != nil:
		name := m.Map.GetName()
		if p := m.Self.Place; p != "" {
			name += " — " + p
		}
		return fmt.Sprintf("%s, at %d,%d m, north up", name, m.Self.Pos.GetX()/1000, m.Self.Pos.GetY()/1000)
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
	return boxWith(title, w, h, inner, st.Dim)
}

// boxA is box with a red border when alarm is set (D75): a screen over the map
// while something attacks. The title says so too, so colour is not all there is.
func boxA(title string, w, h int, inner []string, st Styler, alarm bool) []string {
	if !alarm {
		return box(title, w, h, inner, st)
	}
	return boxWith(title+"  ⚠ UNDER ATTACK", w, h, inner, func(s string) string { return st.Red(st.Bold(s)) })
}

func boxWith(title string, w, h int, inner []string, border func(string) string) []string {
	top := []rune("┌─ " + strings.TrimSpace(fit(title, w-6)) + " " + strings.Repeat("─", w))
	rows := []string{border(string(top[:w-1]) + "┐")}
	side := border("│")
	for i := 0; i < h-2; i++ {
		line := strings.Repeat(" ", w-2)
		if i < len(inner) && inner[i] != "" {
			line = inner[i]
		}
		rows = append(rows, side+line+side)
	}
	return append(rows, border("└"+strings.Repeat("─", w-2)+"┘"))
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
	if a := alarmText(m, v); a != "" {
		// The alarm takes the line (D75). The answer to the last command goes with
		// it, whole: the alarm gives up its last words before the answer is cut.
		line := " " + a
		if note != "" {
			short := strings.TrimSuffix(strings.TrimSuffix(a, " — Esc to fight"), " — F to fight")
			switch {
			case width(line+" | "+note) <= w:
				line += " | " + note
			default: // the alarm is never the part that is cut
				line = " " + short + " | " + note
			}
		}
		return st.Reverse(st.Red(st.Bold(fit(line, w))))
	}
	if p := m.Self.Place; p != "" {
		space = p + ", " + space
	}
	objective := objectiveWords(m)
	// The line is the connection, the space, the clock and the last command's
	// answer. When it is short, the key hint goes first, then the space and the
	// clock: a denial's reason is never cut for them.
	var tries [][]string
	// Ground changes how fast the character walks (D53): the line says what the
	// server says of its speed (D55), and keeps saying so when it is short.
	ground := m.SpeedWords()
	conn := string(v.Conn)
	if v.Directing != "" {
		conn += " | directing " + v.Directing
	}
	quest := ""
	if q := m.Quest(); q != nil {
		quest = fmt.Sprintf("%s %d/%d", q.GetName(), q.GetStep(), q.GetSteps())
	}
	for _, parts := range [][]string{{conn, space, quest, objective, ground, heard, note}, {conn, quest, objective, ground, heard, note}, {conn, quest, objective, ground, note}, {conn, quest, objective, note}, {conn, objective, ground, note}, {conn, quest, ground, heard, note}, {conn, ground, heard, note}, {conn, ground, note}, {conn, heard, note}, {conn, note}} {
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
	return st.Reverse(fit(" "+conn+" | "+note, w))
}
