package screen

import (
	"fmt"
	"strings"
	"time"

	"github.com/GeneStevens/seq-tui/sessionlog"
)

// The sessions view (D64): the player's own logs, the recordings on this
// machine and the server's list of the account's sessions. The screen shows
// what the app loaded and decides nothing.

// SessionRow is one session in the list. Local rows are recordings on disk;
// server rows are the server's log of the account's sessions.
type SessionRow struct {
	Local     bool
	ID        string
	Character string
	Zone      string
	Start     time.Time
	Length    time.Duration
	Lines     int // server: lines of its log
	Ended     bool
}

// SessionsUI is the sessions view, nil when closed. Local rows come first, then
// the server's. Log, when set, is the server's log of one session as readable
// lines, scrolled up by Scroll.
type SessionsUI struct {
	Dir      string // where the recordings live
	Rows     []SessionRow
	Sel      int
	Loading  bool   // the server's list is on its way
	Err      string // why the server's list is missing
	Log      []string
	LogTitle string
	Scroll   int
	// LogID and Lines are the log being read, to export it.
	LogID string
	Lines []sessionlog.ServerLine
}

// Selected is the chosen row, if any.
func (s *SessionsUI) Selected() (SessionRow, bool) {
	if s.Sel < 0 || s.Sel >= len(s.Rows) {
		return SessionRow{}, false
	}
	return s.Rows[s.Sel], true
}

// sessionLength says how long a session ran, in words that stay short.
func sessionLength(d time.Duration) string {
	d = d.Round(time.Second)
	switch {
	case d <= 0:
		return "-"
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}

func sessionsScreenLines(ui *SessionsUI, w, h int, st Styler) []string {
	rows := make([]string, 0, h)
	add := func(s string) { rows = append(rows, fit(" "+s, w)) }
	if ui.Log != nil {
		add(st.Bold(ui.LogTitle))
		body := h - 3
		end := max(len(ui.Log)-ui.Scroll, 0)
		start := max(end-body, 0)
		for _, l := range ui.Log[start:end] {
			add(l)
		}
		for len(rows) < h-1 {
			add("")
		}
		add(st.Dim("Up, Down, PgUp, PgDn scroll   E exports this log to a file   Esc back"))
		return padRows(rows, w, h)
	}
	dir := []rune(ui.Dir)
	if room := w - 16; len(dir) > room && room > 1 {
		dir = append([]rune("…"), dir[len(dir)-room+1:]...)
	}
	add(st.Dim("Logs live in " + string(dir)))
	head := func(t string) { add(st.Bold(t)) }
	line := func(i int, r SessionRow) string {
		mark := "  "
		if i == ui.Sel {
			mark = "> "
		}
		when := "-"
		if !r.Start.IsZero() {
			when = r.Start.UTC().Format("2006-01-02 15:04")
		}
		state := ""
		if !r.Local && !r.Ended {
			state = " (open)"
		}
		zone := r.Zone
		if zone == "" {
			zone = "-"
		}
		who := r.Character
		if who == "" {
			who = "-"
		}
		s := fmt.Sprintf("%s%-16s  %-10s  %-14s  %-7s  %s%s", mark, when, fit(who, 10), fit(zone, 14), sessionLength(r.Length), r.ID, state)
		if i == ui.Sel {
			return st.Reverse(fit(" "+s, w))
		}
		return fit(" "+s, w)
	}
	section := func(local bool, title, empty string) {
		head(title)
		n := 0
		for i, r := range ui.Rows {
			if r.Local == local && len(rows) < h-2 {
				rows = append(rows, line(i, r))
				n++
			}
		}
		if n == 0 {
			add("  " + empty)
		}
	}
	section(true, "On this machine (replay, export)", "no recordings yet")
	switch {
	case ui.Loading:
		head("On the server")
		add("  asking the server…")
	case ui.Err != "":
		head("On the server")
		add("  not available: " + ui.Err)
	default:
		section(false, "On the server (its log of your sessions: view, export)", "the server has none of yours")
	}
	for len(rows) < h-1 {
		add("")
	}
	add(st.Dim("Up, Down choose   Enter: replay a recording, read a server log   E export   R refresh   Esc back"))
	return padRows(rows, w, h)
}

func padRows(rows []string, w, h int) []string {
	for len(rows) < h {
		rows = append(rows, strings.Repeat(" ", w))
	}
	return rows[:h]
}
