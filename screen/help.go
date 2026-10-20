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
	{Keys: []string{"up", "down", "left", "right", "w", "a", "s", "d"}, Show: "Arrows, W A S D", What: "tap for one 3 m step, hold to walk that way and let go to stop (Esc stops a click-walk)", Group: "Moving"},
	{Keys: []string{"e", "enter"}, Show: "E, Enter", What: "go through the nearest door (stand at it)", Group: "Moving"},
	{Keys: []string{"m"}, Show: "M", What: "the large map: click a spot to walk there (arrows and Enter do too)", Group: "Moving"},
	{Keys: []string{"+", "=", "-", "_"}, Show: "+ / -", What: "zoom the map in and out (the scroll wheel does too)", Group: "Moving"},
	{Keys: []string{"ctrl+a"}, Show: "Ctrl+A", What: "name everything on the map, on or off (holding Alt over the map does too)", Group: "Moving"},
	{Keys: []string{"tab"}, Show: "Tab", What: "choose the next target", Group: "Fighting"},
	{Keys: []string{"esc"}, Show: "Esc", What: "clear the target", Group: "Fighting"},
	{Keys: []string{"f", "space"}, Show: "F, Space", What: "auto-attack on or off (none targeted: what attacks you)", Group: "Fighting"},
	{Keys: []string{"ctrl+r"}, Show: "Ctrl+R", What: "auto-retaliate on or off: fight back at once when something attacks you (on by default)", Group: "Fighting"},
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
// The lines are markup: a line with no indent is a section title, {braces} mark a
// key, and a line led by preStyled is already drawn (the legend's).
type helpPage struct {
	title string
	lines func(m *model.Model, v View, w int, st Styler) []string
}

// preStyled leads a help line that is already styled and is only indented.
const preStyled = "\x03"

var helpPages = []helpPage{
	{"Moving, doors, people", func(_ *model.Model, _ View, w int, _ Styler) []string { return sections(w, guidePlaying) }},
	{"The map and its legend", mapPage},
	{"The mouse and the sidebar", func(_ *model.Model, _ View, w int, _ Styler) []string { return sections(w, guideMouse) }},
	{"Quest chains and fighting", func(_ *model.Model, _ View, w int, _ Styler) []string { return sections(w, guideChains) }},
	{"Mobs and difficulty", func(_ *model.Model, _ View, w int, _ Styler) []string { return sections(w, guideMobs) }},
	{"Logs and replay", func(_ *model.Model, _ View, w int, _ Styler) []string { return sections(w, guideLogs) }},
	{"Every key", func(m *model.Model, v View, w int, _ Styler) []string { return keyLines(m, v, w) }},
	{"Chat, parties, who and friends", func(_ *model.Model, _ View, w int, _ Styler) []string { return socialHelp() }},
	{"Group rules", func(m *model.Model, _ View, w int, _ Styler) []string { return groupRulesPage(m, w) }},
}

// HelpPages is how many pages the help has.
func HelpPages() int { return len(helpPages) }

// section is a titled part of a page: the keys of it as a table, then a little prose.
type section struct {
	head string
	keys [][2]string // key, what it does
	body []string
}

// keyColW is the width of the key column of the tables.
const keyColW = 18

var guidePlaying = []section{
	{"Moving and doors", [][2]string{
		{"Arrows, W A S D", "tap for one 3 m step; hold to walk that way, let go to stop"},
		{"Esc", "stop a walk begun by a click"},
		{"E, Enter", "go through the nearest door (stand at it)"},
		{"M", "the large map: click a spot to walk there"},
	}, []string{
		"A terminal that reports key releases (Kitty, Ghostty, WezTerm, recent iTerm2) stops you the moment you let go. Any other steps once on a single press and walks while the key repeats, stopping about a tenth of a second after the repeats stop. Two keys held at once walk a diagonal where your terminal reports them.",
		"Your own mark runs ahead of the server's word so a key shows at once, and it never walks through a wall; the server settles where you are. The status line says how fast you go on the ground you stand on.",
		"A door (D) leads to the next zone or to an instance. A zone line (the east gate, say) takes you across; E at the far side brings you back. A door that cannot be used says why in the line under the log, and Nearby marks it (shut).",
		"Nearby lists what is around you by name and distance: people, doors and places, and the town's fixed people even before you have seen them.",
	}},
	{"Talking, trainers and contracts", [][2]string{
		{"G", "talk to the nearest person; their options open a trader, a trainer or a board"},
		{"N", "traders in view: buy, sell, learn an ability, set your bind point, bank"},
		{"K", "the contracts you hold"},
	}, []string{
		"A contract board offers hunts, gathering and scouting. Hand a finished one in at any contract board in town, not only the one that posted it. Nothing here is required and nothing waits for you: play in any order.",
	}},
	{"When something is refused", nil, []string{
		"The server's reason shows in red with a ! in Events, and the status line goes red with it until your next key. Pressing E at a shut door still asks the server; the reason is its answer.",
	}},
}

// mapPage is the page on reading the map: the legend, then the marks and the alarm.
func mapPage(m *model.Model, _ View, w int, st Styler) []string {
	out := []string{"Legend"}
	for _, l := range legendLines(m, w-2, st) {
		out = append(out, preStyled+"  "+l)
	}
	out = append(out, "", "Marks and pointers")
	for _, p := range []string{
		"Gold is the quest's alone. A gold ! is somebody with something new for you; a gold ? is a hand-in that is ready, a grey ? a quest in progress. A board that has something to take or hand in is gold. Mobs that count for a contract you hold are gold, and the next door on your way is a gold D. Shape says it as well as colour.",
		"World events and scouting have their own marks. The scout points (;) are gold, as your own scout marker (%) is. A world event's raiders appear at a purple v and head for a purple ◆. Hover over any of them for its name, or click to walk there; Nearby lists the named ones.",
		"When your objective is off the map an arrow at the edge points to it, with its name and distance (◄ Gorran 23m); the status line names it too. Nearby lists what matters to your quest first, tagged [step 2] or [2/3].",
		"The zone edge is the ░ line: walking ends there. Beside it, a zone line is a door (D).",
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

var guideMouse = []section{
	{"The mouse", [][2]string{
		{"Click ground", "walk there; the server plans the route, inside this zone"},
		{"Click mob, person", "select it and open its card; click again (or F to attack, G to talk) to act"},
		{"Click loot", "pick up everything within reach"},
		{"Click a marker", "walk to it (a scout point, a world event's mark)"},
		{"Hover", "its name beside the pointer: a mob, person, door, marker, or you"},
		{"Alt, Ctrl+A", "name everything on the map (Ctrl+A if your terminal sends no Alt with the mouse)"},
		{"Wheel, + -", "zoom the map"},
		{"Right-click, Esc", "let go of the selection"},
	}, []string{
		"A first click never attacks. M opens the large map: click a spot to walk there, or move the cursor with the arrows and press Enter. Shift+click is never used: terminals keep it for selecting text. Every mouse action has a key; the game is fully playable without a mouse.",
	}},
	{"The character sidebar", [][2]string{
		{"C", "the next tab (in a narrow window, holds the sidebar open as a drawer; C or Esc puts it away)"},
		{"PgUp, PgDn", "scroll (the wheel does too); click a tab to pick it"},
		{"J", "at a vendor: sell all your junk (no gear slot, wanted by no contract of yours)"},
		{"A", "at a vendor: sell the whole stack under the cursor"},
	}, []string{
		"Beside a trader, a trainer, the bank, a contract board or a conversation sits your character: Gear (what you wear), Bags (what you carry, stacked, \"Wolf Pelt ×2\") and Stats (health, armour, accuracy, what your swing does now, and what gear adds to each attribute).",
		"Every item says how it compares with what you wear in its slot: ▲ better, ▼ worse, = about the same, x you cannot use it (the card says why). What you cannot afford is greyed out with how many crowns you are short.",
	}},
}

var guideChains = []section{
	{"Quest chains", [][2]string{
		{"Ctrl+K", "the quest chain screen (or /quest); X there gives it up for good, after one question"},
	}, []string{
		"A new character is offered First Steps in Thornwick: talk to the Watch greeter in Hawthorn Square, talk to your class trainer, take a contract from the board, then go out through the east gate. It pays a small reward at the end.",
		"It is optional. Nothing walks you anywhere and nothing reminds you; the Quest chain panel shows the step you are on, and the log tells you when one is done. A dismissed chain is never offered again.",
	}},
	{"Fighting", [][2]string{
		{"Tab", "choose the next target"},
		{"Esc", "clear the target"},
		{"F, Space", "auto-attack on or off"},
		{"Ctrl+R", "auto-retaliate on or off (on, you fight back when attacked, even with no input)"},
		{"1-8", "use the ability in that slot on your target"},
		{"R", "when you are dead: give up your body and wake at your bind point"},
	}, []string{
		"With no target, an attack or an ability goes to whatever is attacking you, the nearest first; the status line names the one picked. If nothing is attacking you it is refused: Tab to a target first.",
		"A fight stops your walk: attacking, using an ability on an enemy, or something attacking you. The log says \"Your walk stopped\"; hold a move key to go on. To back off: F turns auto-attack off, Esc clears the target, then walk away.",
	}},
}

var guideMobs = []section{
	{"What mobs do", nil, []string{
		"Difficulty: each mob is trivial, easy, even, hard or deadly to you, by its level against yours (Nearby and the Target panel say it in words). Trivial mobs leave you alone and drop nothing rare; deadly ones are four or more levels above you.",
		"Casters: a mob's bolt takes a few seconds to cast. A stun or an interrupting blow (a warrior's Bash) breaks it off, and the log says \"your Bash breaks off the hexer's bolt\".",
		"Healers: a shaman heals the mob it fights beside. The log says who heals whom; kill the healer first, or break its cast.",
		"Fleeing: a hurt goblin runs for home once, then turns back to fight. Follow it, or let it go: it comes back by itself.",
		"Linked camps: pulling one mob can bring its partner camp, and a chief's cry can call the guards of his wing. The log says \"joins the fight\" or \"answers\" for each.",
	}},
}

var guideLogs = []section{
	{"Logs and replay", [][2]string{
		{"Ctrl+L", "your sessions, and the server's log of them"},
		{"Enter", "replay the session in place (Left, Right step; Home, End jump; Esc comes back)"},
		{"R, E", "read the server's log of one; export one to a file"},
		{"V", "the log, short or all: routine movement and rules ticks"},
	}, []string{
		"Every session is recorded by default, under its id, in seq-tui/sessions in your data dir (-log-dir moves it, -no-record turns it off). A log never holds your credential.",
		"seq-tui -replay FILE (or a session id) draws a recording with no server; -text prints it as plain text.",
	}},
}

// keyTable lays out rows as a table of w cells: the key in a column of keyW, what
// it does beside it, wrapped under itself.
func keyTable(rows [][2]string, w int) []string {
	var out []string
	for _, r := range rows {
		key := "{" + r[0] + "}"
		pad := strings.Repeat(" ", max(keyColW-width(r[0]), 1))
		avail := max(w-2-keyColW, 10)
		for i, l := range wrapIndent(r[1], avail, "") {
			if i == 0 {
				out = append(out, "  "+key+pad+l)
				continue
			}
			out = append(out, strings.Repeat(" ", 2+keyColW)+l)
		}
	}
	return out
}

// sections wraps the sections to w cells: a title, its key table, then its prose.
func sections(w int, ss []section) []string {
	var out []string
	for i, s := range ss {
		if i > 0 {
			out = append(out, "")
		}
		out = append(out, s.head)
		out = append(out, keyTable(s.keys, w)...)
		if len(s.keys) > 0 && len(s.body) > 0 {
			out = append(out, "")
		}
		for _, p := range s.body {
			out = append(out, wrapIndent("  "+p, w, "  ")...)
		}
	}
	return out
}

// mwidth is the width of a line of markup as drawn: the braces and warning marks
// are not cells.
func mwidth(s string) int {
	return width(strings.NewReplacer("{", "", "}", "", warnOpen, "", warnClose, "").Replace(s))
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
		case mwidth(line)+1+mwidth(word) <= w:
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

// KeyLines is the keys page as plain text: the Bindings the character can use, grouped.
func KeyLines(m *model.Model, v View, w int) []string {
	lines := keyLines(m, v, w)
	for i, l := range lines {
		lines[i] = strings.NewReplacer("{", "", "}", "").Replace(l)
	}
	return lines
}

// keyLines is the keys page as markup.
func keyLines(m *model.Model, v View, w int) []string {
	var out []string
	for _, g := range Groups {
		var rows [][2]string
		for _, b := range Bindings {
			if b.Group == g && bindingShown(b, m, v) {
				rows = append(rows, [2]string{b.Show, b.What})
			}
		}
		if len(rows) == 0 {
			continue
		}
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, g)
		out = append(out, keyTable(rows, w)...)
	}
	out = append(out, "", "Map and log")
	out = append(out, keyTable([][2]string{
		{"Legend", "on its own page (? turns the page)"},
		{"! in the log", "a refusal, a cast that did not land, a walk a fight stopped"},
		{"< x + *", "a hit on you; a kill; a reward, level or quest step; a world event"},
	}, w)...)
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

// helpLines draws the open page of the help, scrolled: section titles in the
// panel-title style, key names as the hint bar shows them.
func helpLines(m *model.Model, v View, w, h int, st Styler) []string {
	page := helpPages[min(max(v.HelpPage, 0), len(helpPages)-1)]
	lines := page.lines(m, v, w-2, st)
	body := h - 1
	top := min(max(v.HelpScroll, 0), max(len(lines)-body, 0))
	rows := make([]string, 0, h)
	for _, l := range lines[top:min(top+body, len(lines))] {
		switch {
		case strings.HasPrefix(l, preStyled):
			l = " " + strings.TrimPrefix(l, preStyled)
			rows = append(rows, l+strings.Repeat(" ", max(w-visibleWidth(l), 0)))
		case l != "" && !strings.HasPrefix(l, " "):
			rows = append(rows, st.Accent(fit(" "+l, w)))
		default:
			rows = append(rows, keyed(" "+l, w, ident, st))
		}
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

// visibleWidth is the cells of a styled string: its escape sequences take none.
func visibleWidth(s string) int {
	n, in := 0, false
	for _, r := range s {
		switch {
		case r == 0x1b:
			in = true
		case in:
			in = r != 'm'
		default:
			n++
		}
	}
	return n
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
	return max(len(page.lines(m, v, w-4, Plain()))-body, 0)
}
