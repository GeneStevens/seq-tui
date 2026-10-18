package screen

import (
	"fmt"
	"slices"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
)

// The ? help screen (D62): short plain sections, then every key. It opens from
// every view and closes from it; the keys page is drawn from Bindings, the same
// table the program's tests hold the key handler to, so it cannot drift.

// Binding is a key of the main view, and what it does.
type Binding struct {
	Keys  []string // the names the program receives for it
	Show  string   // how the help writes it
	What  string
	Group string
	// When limits the row: "" always, "companion" a character with a house
	// companion, "duo" a client with a second character, "turn" a turn-based round
	// (its row replaces nothing: the plan keys are listed with the rest).
	When string
}

// Groups are the headings of the keys page, in order.
var Groups = []string{"Moving", "Fighting", "People and things", "Screens", "Leaving and help", "Plans (turn-based rounds)", "Helpers"}

// Bindings is every key of the main view. Screens that open from it list their
// own keys on their last line.
var Bindings = []Binding{
	{Keys: []string{"up", "down", "left", "right", "w", "a", "s", "d"}, Show: "Arrows, W A S D", What: "walk a few metres that way", Group: "Moving"},
	{Keys: []string{"e", "enter"}, Show: "E, Enter", What: "go through the nearest door (stand at it)", Group: "Moving"},
	{Keys: []string{"m"}, Show: "M", What: "the whole zone you have seen", Group: "Moving"},
	{Keys: []string{"tab"}, Show: "Tab", What: "choose the next target", Group: "Fighting"},
	{Keys: []string{"esc"}, Show: "Esc", What: "clear the target", Group: "Fighting"},
	{Keys: []string{"f", "space"}, Show: "F, Space", What: "auto-attack on or off; with no target, the one attacking you", Group: "Fighting"},
	{Keys: []string{"1", "2", "3", "4", "5", "6", "7", "8"}, Show: "1-8", What: "use that slot's ability on your target (none: what attacks you)", Group: "Fighting"},
	{Keys: []string{"b"}, Show: "B", What: "set the ability slots (out of combat)", Group: "Fighting"},
	{Keys: []string{"r"}, Show: "R", What: "when you are dead: give up your body and respawn", Group: "Fighting"},
	{Keys: []string{"g"}, Show: "G", What: "talk to the nearest person", Group: "People and things"},
	{Keys: []string{"l"}, Show: "L", What: "loot the nearest drop you may take", Group: "People and things"},
	{Keys: []string{"k"}, Show: "K", What: "contracts you hold (X gives one up)", Group: "People and things"},
	{Keys: []string{"ctrl+k"}, Show: "Ctrl+K", What: "your quest chain: its step; X dismisses it for good", Group: "People and things"},
	{Keys: []string{"n"}, Show: "N", What: "traders: buy, sell, train, bind, bank", Group: "People and things"},
	{Keys: []string{"c"}, Show: "C", What: "inventory: wear and take off gear", Group: "People and things"},
	{Keys: []string{"t"}, Show: "T", What: "trade with a character in reach", Group: "People and things"},
	{Keys: []string{"i"}, Show: "I", What: "this zone's instance menu (Enter opens one)", Group: "People and things"},
	{Keys: []string{"/"}, Show: "/", What: "the command line: chat, parties, /quest, /dismiss", Group: "Screens"},
	{Keys: []string{"ctrl+l"}, Show: "Ctrl+L", What: "your session logs: replay, read, export", Group: "Screens"},
	{Keys: []string{"v"}, Show: "V", What: "more log: routine movement and rules ticks", Group: "Screens"},
	{Keys: []string{"?", "f1"}, Show: "? or F1", What: "this help (from any screen); ? turns the page, Esc closes", Group: "Leaving and help"},
	{Keys: []string{"o"}, Show: "O", What: "log out", Group: "Leaving and help"},
	{Keys: []string{"q", "ctrl+c"}, Show: "Q, Ctrl+C", What: "quit the screen only (your character stays in the world)", Group: "Leaving and help"},
	{Keys: []string{"x"}, Show: "X", What: "in a round: stop where you are", Group: "Plans (turn-based rounds)", When: "turn"},
	{Keys: []string{"enter"}, Show: "Enter", What: "submit the plan", Group: "Plans (turn-based rounds)", When: "turn"},
	{Keys: []string{"backspace"}, Show: "Backspace", What: "drop the last step", Group: "Plans (turn-based rounds)", When: "turn"},
	{Keys: []string{"esc"}, Show: "Esc", What: "clear the plan", Group: "Plans (turn-based rounds)", When: "turn"},
	{Keys: []string{"p"}, Show: "P", What: "call your companion, or send it away", Group: "Helpers", When: "companion"},
	{Keys: []string{"h"}, Show: "H", What: "its stance", Group: "Helpers", When: "companion"},
	{Keys: []string{"y"}, Show: "Y", What: "attack my target", Group: "Helpers", When: "companion"},
	{Keys: []string{"u"}, Show: "U", What: "come here", Group: "Helpers", When: "companion"},
	{Keys: []string{"j"}, Show: "J", What: "stop", Group: "Helpers", When: "companion"},
	{Keys: []string{"z"}, Show: "Z", What: "drop the order", Group: "Helpers", When: "companion"},
	{Keys: []string{"."}, Show: ".", What: "the other helper (when you have two)", Group: "Helpers", When: "companion"},
	{Keys: []string{"p"}, Show: "P", What: "bring your second character in or out", Group: "Helpers", When: "duo"},
	{Keys: []string{"h", "y", "u", "j", "z"}, Show: "H Y U J Z", What: "its stance, attack my target, come here, stop, back to its stance", Group: "Helpers", When: "duo"},
	{Keys: []string{"x"}, Show: "X", What: "direct it (its screen, its keys); X again: back to you", Group: "Helpers", When: "duo"},
}

// helpPage is one page of the help: a title and its lines at the width given.
type helpPage struct {
	title string
	lines func(m *model.Model, v View, w int) []string
}

var helpPages = []helpPage{
	{"Moving, doors, people", func(_ *model.Model, _ View, w int) []string { return sections(w, guidePlaying) }},
	{"Quest chains and fighting", func(_ *model.Model, _ View, w int) []string { return sections(w, guideChains) }},
	{"Logs and replay", func(_ *model.Model, _ View, w int) []string { return sections(w, guideLogs) }},
	{"Every key", func(m *model.Model, v View, w int) []string { return KeyLines(m, v, w) }},
	{"Chat, parties, who and friends", func(_ *model.Model, _ View, w int) []string { return socialHelp() }},
}

// HelpPages is how many pages the help has.
func HelpPages() int { return len(helpPages) }

type section struct {
	head string
	body []string
}

var guidePlaying = []section{
	{"Moving and doors", []string{
		"Walk with the arrow keys or W A S D; each press sends you a few metres on. The status line says how fast you go on the ground you stand on.",
		"M shows the whole zone you have seen. A door (D) leads to the next zone or to an instance: stand at it and press E or Enter. A zone line, the east gate say, takes you across; E at the far side brings you back.",
		"Nearby lists what is around you by name and distance: people, doors and places, and the town's fixed people even before you have seen them.",
	}},
	{"Talking, trainers and contracts", []string{
		"G talks to the nearest person; their options open a trader, a trainer or a board. N opens the traders in view: buy, sell, learn an ability from a trainer for crowns, set your bind point, bank.",
		"A contract board offers hunts, gathering and scouting. K lists the contracts you hold. Hand a finished one in at any contract board in town, not only the one that posted it.",
		"Nothing here is required and nothing waits for you: play in any order.",
	}},
}

var guideChains = []section{
	{"Quest chains", []string{
		"A new character is offered First Steps in Thornwick: talk to the Watch greeter in Hawthorn Square, talk to your class trainer, take a contract from the board, then go out through the east gate. It pays a small reward at the end.",
		"It is optional. Nothing walks you anywhere and nothing reminds you; the Quest chain panel shows the step you are on, and the log tells you when one is done.",
		"Ctrl+K (or /quest) shows the chain. X there (or /dismiss) gives it up for good, after one question: a dismissed chain is never offered again.",
	}},
	{"Fighting", []string{
		"Tab chooses a target (Esc clears it). F or Space turns auto-attack on or off. 1-8 use the abilities in your slots on your target.",
		"With no target, an attack or an ability goes to whatever is attacking you, the nearest first; the status line names the one picked. If nothing is attacking you it is refused: Tab to a target first.",
		"A fight stops your walk: attacking, using an ability on an enemy, or something attacking you. The log says \"Your walk stopped\"; press a move key to go on.",
		"To back off: F turns auto-attack off, Esc clears the target, then walk away. If you die, R gives up your body and you wake at your bind point.",
	}},
}

var guideLogs = []section{
	{"Logs and replay", []string{
		"Every session is recorded by default, under its id, in seq-tui/sessions in your data dir (-log-dir moves it, -no-record turns it off). A log never holds your credential.",
		"Ctrl+L lists your sessions, yours and the server's log of them: Enter replays one in place (Left and Right step, Home and End jump, Esc comes back), R reads the server's log, E exports one to a file.",
		"seq-tui -replay FILE (or a session id) draws a recording with no server; -text prints it as plain text.",
		"V shows more of the event log: routine movement and rules ticks.",
	}},
}

// sections wraps the sections to w cells.
func sections(w int, ss []section) []string {
	var out []string
	for i, s := range ss {
		if i > 0 {
			out = append(out, "")
		}
		out = append(out, s.head)
		for _, p := range s.body {
			out = append(out, wrapIndent("  "+p, w, "  ")...)
		}
	}
	return out
}

// wrapIndent breaks s at spaces into lines of at most w cells, each after the first
// starting with indent.
func wrapIndent(s string, w int, indent string) []string {
	var out []string
	line := ""
	for _, word := range strings.Fields(s) {
		switch {
		case line == "":
			line = word
		case width(line)+1+width(word) <= w:
			line += " " + word
		default:
			out = append(out, line)
			line = indent + word
		}
	}
	if line != "" {
		out = append(out, line)
	}
	// Fields ate the first line's indent: put it back.
	if len(out) > 0 && strings.HasPrefix(s, indent) && !strings.HasPrefix(out[0], indent) {
		out[0] = indent + out[0]
	}
	return out
}

// KeyLines is the keys page: the Bindings the character can use, grouped.
func KeyLines(m *model.Model, v View, w int) []string {
	var out []string
	for _, g := range Groups {
		var rows []string
		for _, b := range Bindings {
			if b.Group != g || !bindingShown(b, m, v) {
				continue
			}
			rows = append(rows, fmt.Sprintf("  %-16s %s", b.Show, b.What))
		}
		if len(rows) == 0 {
			continue
		}
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, g)
		for _, r := range rows {
			out = append(out, fit(r, w))
		}
	}
	out = append(out, "", "Map glyphs  @ you  m mob  M target  r rare  c camp  x dead  p player  + plan end",
		"  * drop  ~ a drop you may not take  $ trader  & board  ? talker  D door  # wall",
		"  % your marker  ; scout marker  ! event spawn, goal",
		"Ground  # building  ^ rock  ≈ water  : ford  = bridge  \" forest  · road  . floor",
		"Log markers  ! denied or a cast that did not land  < hit on you  x kill  + reward  * world event")
	return out
}

func bindingShown(b Binding, m *model.Model, v View) bool {
	switch b.When {
	case "companion":
		return m.Companion() != nil
	case "duo":
		return v.Duo != nil || v.Directing != ""
	case "turn":
		return true
	}
	return true
}

// helpLines draws the open page of the help, scrolled.
func helpLines(m *model.Model, v View, w, h int, st Styler) []string {
	page := helpPages[min(max(v.HelpPage, 0), len(helpPages)-1)]
	lines := page.lines(m, v, w-2)
	body := h - 1
	top := min(max(v.HelpScroll, 0), max(len(lines)-body, 0))
	rows := make([]string, 0, h)
	for _, l := range lines[top:min(top+body, len(lines))] {
		rows = append(rows, fit(" "+l, w))
	}
	for len(rows) < body {
		rows = append(rows, strings.Repeat(" ", w))
	}
	more := ""
	if len(lines) > body {
		more = "  Up, Down scroll"
	}
	return append(rows, st.Dim(fit(fmt.Sprintf(" Page %d of %d%s  ? next page  Left, Right turn  Esc closes", v.HelpPage+1, len(helpPages), more), w)))
}

// HelpTitle is the title of the help box.
func HelpTitle(page int) string {
	page = min(max(page, 0), len(helpPages)-1)
	return "Help: " + helpPages[page].title
}

// HelpBindingKeys lists every key name the Bindings name, for a test.
func HelpBindingKeys() []string {
	var out []string
	for _, b := range Bindings {
		for _, k := range b.Keys {
			if !slices.Contains(out, k) {
				out = append(out, k)
			}
		}
	}
	return out
}

// HelpMaxScroll is how far the open page of the help can scroll at w x h.
func HelpMaxScroll(m *model.Model, v View, w, h int) int {
	page := helpPages[min(max(v.HelpPage, 0), len(helpPages)-1)]
	lh := min(6+(h-MinHeight)/3, 14)
	th := h - lh - 1
	body := th + lh - 2 - 1
	return max(len(page.lines(m, v, w-4))-body, 0)
}
