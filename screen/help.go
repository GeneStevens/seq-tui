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
	{Keys: []string{"m"}, Show: "M", What: "the large map: click a spot to walk there (arrows and Enter do too)", Group: "Moving"},
	{Keys: []string{"+", "=", "-", "_"}, Show: "+ / -", What: "zoom the map in and out (the scroll wheel does too)", Group: "Moving"},
	{Keys: []string{"ctrl+a"}, Show: "Ctrl+A", What: "name everything on the map, on or off (holding Alt over the map does too)", Group: "Moving"},
	{Keys: []string{"tab"}, Show: "Tab", What: "choose the next target", Group: "Fighting"},
	{Keys: []string{"esc"}, Show: "Esc", What: "clear the target", Group: "Fighting"},
	{Keys: []string{"f", "space"}, Show: "F, Space", What: "auto-attack on or off (none targeted: what attacks you)", Group: "Fighting"},
	{Keys: []string{"1", "2", "3", "4", "5", "6", "7", "8"}, Show: "1-8", What: "use that slot's ability on your target", Group: "Fighting"},
	{Keys: []string{"b"}, Show: "B", What: "set the ability slots (out of combat)", Group: "Fighting"},
	{Keys: []string{"r"}, Show: "R", What: "when you are dead: give up your body and respawn", Group: "Fighting"},
	{Keys: []string{"g"}, Show: "G", What: "talk to the nearest person", Group: "People and things"},
	{Keys: []string{"l"}, Show: "L", What: "loot everything within reach that you may take", Group: "People and things"},
	{Keys: []string{"k"}, Show: "K", What: "contracts you hold (X gives one up)", Group: "People and things"},
	{Keys: []string{"ctrl+k"}, Show: "Ctrl+K", What: "your quest chain: its step; X dismisses it for good", Group: "People and things"},
	{Keys: []string{"n"}, Show: "N", What: "traders: buy, sell, train, bind, bank", Group: "People and things"},
	{Keys: []string{"c"}, Show: "C", What: "your character: gear, bags and stats; Enter wears or takes off (beside a trader, C turns the sidebar tab)", Group: "People and things"},
	{Keys: []string{"t"}, Show: "T", What: "trade with a character in reach", Group: "People and things"},
	{Keys: []string{"i"}, Show: "I", What: "this zone's instance menu (Enter opens one)", Group: "People and things"},
	{Keys: []string{"/"}, Show: "/", What: "the command line: chat, parties, /quest, /dismiss", Group: "Screens"},
	{Keys: []string{"ctrl+l"}, Show: "Ctrl+L", What: "your session logs: replay, read, export", Group: "Screens"},
	{Keys: []string{"v"}, Show: "V", What: "the log, short or all: routine movement, rules ticks and people coming into view (the Events title says which)", Group: "Screens"},
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
	{"The map and its legend", func(m *model.Model, _ View, w int) []string { return mapPage(m, w) }},
	{"The mouse and the sidebar", func(_ *model.Model, _ View, w int) []string { return sections(w, guideMouse) }},
	{"Quest chains and fighting", func(_ *model.Model, _ View, w int) []string { return sections(w, guideChains) }},
	{"Mobs and difficulty", func(_ *model.Model, _ View, w int) []string { return sections(w, guideMobs) }},
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
		"M shows the whole zone you have seen. A door (D) leads to the next zone or to an instance: stand at it and press E or Enter. A zone line (the east gate, say) takes you across; E at the far side brings you back.",
		"Nearby lists what is around you by name and distance: people, doors and places, and the town's fixed people even before you have seen them.",
	}},
	{"Talking, trainers and contracts", []string{
		"G talks to the nearest person; their options open a trader, a trainer or a board. N opens the traders in view: buy, sell, learn an ability from a trainer for crowns, set your bind point, bank.",
		"A contract board offers hunts, gathering and scouting. K lists the contracts you hold. Hand a finished one in at any contract board in town, not only the one that posted it.",
		"Nothing here is required and nothing waits for you: play in any order.",
	}},
}

// mapPage is the page on reading the map: the legend, then the marks and the alarm.
func mapPage(m *model.Model, w int) []string {
	out := append([]string{"Legend"}, indentAll(legendLines(m, w-2), "  ")...)
	out = append(out, "", "Marks and pointers")
	for _, p := range []string{
		"A gold ! is somebody with something new for you; a gold ? is a hand-in that is ready, a grey ? a quest in progress. A board that has something to take or hand in is gold. Mobs that count for a contract you hold are gold, and the next door on your way is a gold D. Shape says it as well as colour.",
		"When your objective is off the map an arrow at the edge points to it, with its name and distance (◄ Gorran 23m); the status line names it too. Nearby lists what matters to your quest first, tagged [step 2] or [2/3].",
		"The zone edge is the ▒ line: walking ends there. Beside it, a zone line is a door (D).",
		"The line under the log changes with what you stand next to: the door, the person, the drop, the mobs in view.",
	} {
		out = append(out, wrapIndent("  "+p, w, "  ")...)
	}
	out = append(out, "", "Danger")
	for _, p := range []string{
		"Services (traders, talk, the board, the character view, contracts) are panels in the map's area: the You, Target and Nearby panels stay beside them. A screen never hides a fight. When something attacks you its border turns red, the status line names it and your health, and nothing closes by itself. Esc leaves the screen; the attacker is already your target, so F fights it. -bell rings the terminal's bell too.",
		"The You panel says \"in combat\" and who is attacking. A hard or deadly mob is called out as DANGER.",
	} {
		out = append(out, wrapIndent("  "+p, w, "  ")...)
	}
	return out
}

func indentAll(lines []string, indent string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = indent + l
	}
	return out
}

var guideMouse = []section{
	{"The mouse", []string{
		"Click the ground to walk there; the server plans the route, inside this zone. The first click on a mob or a person selects it and opens a small card (name, title, level, difficulty, what they offer); a second click, or F (attack) or G (talk), acts. A first click never attacks. Click loot to pick up everything within reach.",
		"Hover for a name and title beside the pointer. Hold Alt over the map to name everything on it; if your terminal does not send Alt with the mouse, Ctrl+A turns the names on and off. The wheel zooms (+ and - do too). M opens the large map: click a spot to walk there, or move the cursor with the arrows and press Enter.",
		"Right-click lets go of the selection, as Esc does. Shift+click is never used: terminals keep it for selecting text.",
		"Every mouse action has a key: arrows or WASD walk, Tab selects, F and G act, L loots, + and - zoom, Ctrl+A names. The game is fully playable without a mouse.",
	}},
	{"The character sidebar", []string{
		"Beside a trader, a trainer, the bank, a contract board or a conversation sits your character: Gear (what you wear), Bags (what you carry, stacked, \"Wolf Pelt ×2\") and Stats (health, armour, accuracy, what your swing does now, bare hands too, and what gear adds to each attribute). C turns to the next tab; click a tab to pick it; PgUp and PgDn or the wheel scroll. In a narrow window C holds the sidebar open as a drawer and C or Esc puts it away.",
		"Every item says how it compares with what you wear in its slot: ▲ better, ▼ worse, = about the same, x you cannot use it (the card says why). What you cannot afford is greyed out with how many crowns you are short. At a vendor, J sells all your junk (things with no gear slot that no contract of yours wants), A sells the whole stack under the cursor.",
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

var guideMobs = []section{
	{"What mobs do", []string{
		"Difficulty: each mob is trivial, easy, even, hard or deadly to you, by its level against yours (Nearby and the Target panel say it in words). Trivial mobs leave you alone and drop nothing rare; deadly ones are four or more levels above you.",
		"Casters: a mob's bolt takes a few seconds to cast. A stun or an interrupting blow (a warrior's Bash) breaks it off, and the log says \"your Bash breaks off the hexer's bolt\".",
		"Healers: a shaman heals the mob it fights beside. The log says who heals whom; kill the healer first, or break its cast.",
		"Fleeing: a hurt goblin runs for home once, then turns back to fight. Follow it, or let it go: it comes back by itself.",
		"Linked camps: pulling one mob can bring its partner camp, and a chief's cry can call the guards of his wing. The log says \"joins the fight\" or \"answers\" for each.",
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
	lead := s[:len(s)-len(strings.TrimLeft(s, " "))]
	var out []string
	line := lead
	for _, word := range strings.Fields(s) {
		switch {
		case strings.TrimSpace(line) == "":
			line += word
		case width(line)+1+width(word) <= w:
			line += " " + word
		default:
			out = append(out, line)
			line = indent + word
		}
	}
	if strings.TrimSpace(line) != "" {
		out = append(out, line)
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
	out = append(out, "", "Map: the legend is on its own page (? turns the page).",
		"Log markers  ! denied, a cast that did not land, a walk a fight stopped  < hit on you",
		"  x kill  + reward, level or quest step  * world event")
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
