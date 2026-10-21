package screen

import (
	"fmt"
	"strings"
	"time"
)

// PickerRow is one character the picker lists.
type PickerRow struct {
	ID, Name, Class string
	Level           int64
	Zone            string
	Played          time.Time
	Online          bool // a connection plays it now: picking it is refused
}

// Picker is the character picker, when it is open (a session with no character
// chosen, and characters to choose from): one row for each character, the last
// played first and chosen to begin with, then "Create a new character" as the
// row after the last. Sel is the row the cursor is on.
type Picker struct {
	Rows []PickerRow
	Sel  int
	// Busy: the chosen character is being logged in. Err is why the last try failed.
	Busy bool
	Err  string
}

// Create says the cursor is on the "Create a new character" row.
func (p *Picker) Create() bool { return p.Sel >= len(p.Rows) }

// Move moves the cursor by d rows, stopping at the first and the create row.
func (p *Picker) Move(d int) { p.Sel = max(0, min(p.Sel+d, len(p.Rows))) }

// Chosen is the row under the cursor, or nil on the create row.
func (p *Picker) Chosen() *PickerRow {
	if p.Create() {
		return nil
	}
	return &p.Rows[p.Sel]
}

// ago says how long before now a time was, in words a table holds.
func ago(t, now time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%d h ago", int(d.Hours()))
	}
	return fmt.Sprintf("%d days ago", int(d.Hours()/24))
}

// lastPlayed is the last column: "online" for a character a connection plays
// now (picking it is refused), else when it was last played.
func lastPlayed(r PickerRow, now time.Time) string {
	if r.Online {
		return "online (in use)"
	}
	return ago(r.Played, now)
}

var pickerKeys = [][2]string{
	{"Up, Down", "choose a character (the last one you played is chosen to begin with); 1-9 jump to one"},
	{"Enter", "play the chosen character, or make a new one (an online one is in use: the server refuses it)"},
	{"N", "make a new character"},
	{"Ctrl+L", "the sessions view"},
	{"Ctrl+C", "quit"},
}

func pickerLines(p *Picker, now time.Time, w, h int, st Styler) []string {
	var rows []string
	add := func(s string) { rows = append(rows, fit(s, w)) }
	add(" Who will you play?")
	add("")
	head := fmt.Sprintf("    %-16s %-9s %5s  %-16s %s", "Name", "Class", "Level", "Zone", "Last played")
	rows = append(rows, st.Bold(fit(head, w)))
	for i, r := range p.Rows {
		mark := "  "
		if i == p.Sel {
			mark = "> "
		}
		zone := r.Zone
		if zone == "" {
			zone = "-"
		}
		class := r.Class
		if class == "" {
			class = "-"
		}
		line := fit(fmt.Sprintf(" %s%-16s %-9s %5d  %-16s %s", mark, fit(r.Name, 16), fit(class, 9), r.Level, fit(zone, 16), lastPlayed(r, now)), w)
		if i == p.Sel {
			line = st.Reverse(line)
		}
		rows = append(rows, line)
	}
	create := fit("   + Create a new character", w)
	if p.Create() {
		create = st.Reverse(fit(" > + Create a new character", w))
	}
	rows = append(rows, create, fit("", w))
	var tail []string
	switch {
	case p.Busy:
		tail = append(tail, fit(" Logging in…", w))
	case p.Err != "":
		tail = append(tail, st.Red(st.Bold(fit(" ! "+p.Err, w))))
	default:
		tail = append(tail, fit("", w))
	}
	for _, l := range keyTable(pickerKeys, w) {
		tail = append(tail, help(l, w, st))
	}
	for len(rows)+len(tail) < h {
		rows = append(rows, strings.Repeat(" ", w))
	}
	rows = rows[:min(len(rows), max(h-len(tail), 0))]
	return append(rows, tail...)
}
