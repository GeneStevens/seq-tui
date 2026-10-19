package screen

import (
	"fmt"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The screens of the people and the world of M8 (SEQ-044, SEQ-045): talking,
// the contract board, the Contracts panel, the whole-zone overview and the
// banner of a world event.

// Talk is the conversation screen, when it is open: with whom, and the option
// the cursor is on.
type Talk struct {
	NPC string
	Sel int
}

// Board is the contract board's screen, when it is open: whose board, which
// list the keys go to (0 the contracts on offer, 1 the ones held) and the
// cursor in each.
type Board struct {
	NPC  string
	Pane int
	Cur  [2]int
}

// ContractsUI is the Contracts panel as a screen, when it is open.
type ContractsUI struct{ Sel int }

// eventWindowMs is how long, in game time, the banner of an event that has
// ended stays up.
const eventWindowMs = 30_000

// wrapped wraps text into lines of at most w cells, each indented by one cell.
func wrapped(text string, w int, indent string) []string {
	var out []string
	for _, l := range wrap(clean(text), max(w-width(indent), 8)) {
		out = append(out, fit(indent+l, w))
	}
	return out
}

// optionLabel is an option of a conversation as the screen lists it.
func optionLabel(o *pb.DialogueOption) string {
	if o.GetAction() == "" {
		return o.GetLabel() + " (nothing to do yet)"
	}
	return o.GetLabel()
}

func talkLines(m *model.Model, t *Talk, w, h int, st Styler) []string {
	var rows []string
	add := func(l string) { rows = append(rows, fit(l, w)) }
	name := m.NPCName(t.NPC)
	n := m.NPC(t.NPC)
	head := " " + name
	if n != nil {
		sp := m.Self.Pos
		head += fmt.Sprintf(", %s %s of you", metres(dist(n.GetPos().GetX(), n.GetPos().GetY(), sp.GetX(), sp.GetY())), bearing(n.GetPos().GetX()-sp.GetX(), n.GetPos().GetY()-sp.GetY()))
	}
	rows = append(rows, st.Bold(fit(head, w)))
	add("")
	d := m.Said(t.NPC)
	if d == nil {
		add(" …")
		return pad(rows, w, h)
	}
	for _, l := range d.Lines {
		rows = append(rows, wrapped("“"+l+"”", w, " ")...)
	}
	add("")
	if len(d.Options) == 0 {
		add(" They have nothing more to offer.")
	} else {
		add(" What they can do for you:")
	}
	for i, o := range d.Options {
		mark := "  "
		if i == t.Sel {
			mark = "> "
		}
		line := fit(fmt.Sprintf(" %s%d  %s", mark, i+1, optionLabel(o)), w)
		if i == t.Sel {
			line = st.Reverse(line)
		}
		rows = append(rows, line)
	}
	if len(rows) > h-2 {
		rows = rows[:h-2]
	}
	for len(rows) < h-2 {
		rows = append(rows, strings.Repeat(" ", w))
	}
	add(" Number keys or Up / Down choose, Enter does it. Esc closes. G talks again.")
	add(" The answer is on the status line; every refusal says why.")
	return pad(rows[:min(len(rows), h)], w, h)
}

// levels says the levels a contract is for: "L1-3", "L6+".
func levels(lo, hi int64) string {
	switch {
	case hi == 0:
		return fmt.Sprintf("L%d+", lo)
	case lo == hi:
		return fmt.Sprintf("L%d", lo)
	}
	return fmt.Sprintf("L%d-%d", lo, hi)
}

// kindWord is a contract's kind in a word a player reads.
func kindWord(kind string) string {
	switch kind {
	case "hunt":
		return "Hunt"
	case "gather":
		return "Gather"
	case "scout":
		return "Scout"
	case "bounty":
		return "Bounty"
	}
	return kind
}

// markerWords says where a held scout contract's marker is, from the character.
func markerWords(m *model.Model, c *pb.ContractInfo) string {
	if c.GetMarker() == "" || c.GetHave() >= c.GetNeed() {
		return ""
	}
	p := m.Marker(c)
	if p == nil || m.Self.Pos == nil {
		return "the marker is in another place"
	}
	sp := m.Self.Pos
	name := "the marker"
	if n := m.MarkerName(c.GetMarker()); n != c.GetMarker() {
		name = n
	}
	return fmt.Sprintf("%s is %s %s", name, metres(dist(p.GetX(), p.GetY(), sp.GetX(), sp.GetY())), bearing(p.GetX()-sp.GetX(), p.GetY()-sp.GetY()))
}

func offerRow(o *pb.ContractOffer, w int) string {
	lv := levels(o.GetMinLevel(), o.GetMaxLevel())
	rest := fmt.Sprintf(" %-6s %s", kindWord(o.GetKind()), lv)
	return fit(fit(o.GetName(), max(w-width(rest), 4))+rest, w)
}

func heldRow(c *pb.ContractInfo, w int) string {
	state := compactState(c)
	return fit(fit(c.GetName(), max(w-width(state)-1, 4))+" "+state, w)
}

func boardLines(m *model.Model, v View, b *Board, w, h int, st Styler) []string {
	n := m.NPC(b.NPC)
	var rows []string
	add := func(l string) { rows = append(rows, fit(l, w)) }
	if n == nil {
		add(" The board is out of view. Esc closes.")
		return pad(rows, w, h)
	}
	held := m.Self.Contracts
	add(fmt.Sprintf(" %s | you hold %d of 3 contracts | %s", model.NPCLabel(n), len(held), inCrowns(m)))
	moment := handInLines(m, v, w, st)
	rows = append(rows, moment...)
	lw := (w - 1) / 2
	rw := w - 1 - lw
	left := []string{fit(" On offer now (Enter takes it)", lw)}
	var texts []string
	for _, o := range n.GetOffers() {
		texts = append(texts, offerRow(o, max(lw-4, 1)))
	}
	if len(texts) == 0 {
		left = append(left, fit("  (nothing for your level)", lw))
	}
	room := max(h-9-len(moment), 1)
	left = append(left, listRows(texts, b.Cur[0], room, lw, b.Pane == 0, st)...)
	right := []string{fit(" You hold (Enter hands in, X gives up)", rw)}
	texts = nil
	for _, c := range held {
		texts = append(texts, heldRow(c, max(rw-4, 1)))
	}
	if len(texts) == 0 {
		right = append(right, fit("  (none)", rw))
	}
	right = append(right, listRows(texts, b.Cur[1], room, rw, b.Pane == 1, st)...)
	rows = append(rows, cols(left, right, lw, rw)...)
	for len(rows) < h-6 {
		rows = append(rows, strings.Repeat(" ", w))
	}
	rows = rows[:min(len(rows), h-6)]
	rows = append(rows, strings.Repeat("─", w))
	// The detail of the one under the cursor.
	switch {
	case b.Pane == 0 && b.Cur[0] < len(n.GetOffers()):
		o := n.GetOffers()[b.Cur[0]]
		add(fmt.Sprintf(" %s: %s, %s, for %s.", o.GetName(), kindWord(o.GetKind()), levels(o.GetMinLevel(), o.GetMaxLevel()), needWords(o.GetKind(), o.GetNeed())))
		rows = append(rows, wrapped(o.GetDescription(), w, " ")...)
		add(" Pays " + m.RewardWords(o.GetReward()) + ".")
	case b.Pane == 1 && b.Cur[1] < len(held):
		c := held[b.Cur[1]]
		add(fmt.Sprintf(" %s: %s, %s.", c.GetName(), kindWord(c.GetKind()), model.Progress(c)))
		rows = append(rows, wrapped(c.GetDescription(), w, " ")...)
		add(" Pays " + m.RewardWords(c.GetReward()) + ". " + readyWords(m, c))
	}
	if len(rows) > h-2 {
		rows = rows[:h-2]
	}
	for len(rows) < h-2 {
		rows = append(rows, strings.Repeat(" ", w))
	}
	add(" Tab: offered / held. Up / Down choose. Enter acts. Esc goes back.")
	add(" The answer is on the status line; every refusal says why.")
	return pad(rows[:min(len(rows), h)], w, h)
}

func needWords(kind string, need int64) string {
	switch kind {
	case "hunt", "bounty":
		return fmt.Sprintf("%d kills", need)
	case "gather":
		return fmt.Sprintf("%d items", need)
	case "scout":
		return "a visit to a marker"
	}
	return fmt.Sprintf("%d", need)
}

// readyWords says whether a held contract is finished, and where to hand it in
// (D76: the server's hint names the board).
func readyWords(m *model.Model, c *pb.ContractInfo) string {
	if c.GetReady() {
		return "Finished: " + handInWhere(m, c) + "."
	}
	return "Not finished yet."
}

// handInWhere is where a finished contract is handed in, "hand it in at Notice Board, 23m NE".
func handInWhere(m *model.Model, c *pb.ContractInfo) string {
	for _, h := range m.Self.Hints {
		if h.GetKind() == "ready" && strings.HasSuffix(h.GetLabel(), c.GetName()) {
			name := m.NPCTitled(h.GetId())
			if h.GetPos() != nil && m.Self.Pos != nil {
				sp := m.Self.Pos
				return fmt.Sprintf("hand it in at %s, %s %s", name, metres(dist(h.GetPos().GetX(), h.GetPos().GetY(), sp.GetX(), sp.GetY())), bearing(h.GetPos().GetX()-sp.GetX(), h.GetPos().GetY()-sp.GetY()))
			}
			return "hand it in at " + name
		}
	}
	return "hand it in at a contract board"
}

// contractRows are the held contracts as a list: one line each when compact,
// else the name and kind, the progress, where the marker is and the reward. The
// contract number sel (-1: none) is marked.
func contractRows(m *model.Model, w int, full bool, sel int, st Styler) []string {
	var rows []string
	for i, c := range m.Self.Contracts {
		mark := " "
		if i == sel {
			mark = ">"
		}
		if !full {
			state := compactState(c)
			rows = append(rows, fit(fit(" "+c.GetName(), max(w-width(state)-2, 1))+" "+state, w))
			continue
		}
		state := model.Progress(c)
		if c.GetReady() {
			state = "READY: " + handInWhere(m, c)
		}
		head := st.Bold(fit(fmt.Sprintf("%s %s (%s)", mark, c.GetName(), kindWord(c.GetKind())), w))
		if i == sel {
			head = st.Reverse(fit(fmt.Sprintf("%s %s (%s)", mark, c.GetName(), kindWord(c.GetKind())), w))
		}
		rows = append(rows, head)
		line := fit("   "+state, w)
		if c.GetReady() {
			line = st.Green(line)
		}
		rows = append(rows, line)
		if mk := markerWords(m, c); mk != "" {
			rows = append(rows, fit("   "+mk, w))
		}
		rows = append(rows, fit("   Pays "+m.RewardWords(c.GetReward()), w))
		rows = append(rows, wrapped(c.GetDescription(), w, "   ")...)
	}
	return rows
}

// compactState is a contract's progress in the fewest cells: "2/3", "go", "READY".
func compactState(c *pb.ContractInfo) string {
	switch {
	case c.GetReady():
		return "READY"
	case c.GetKind() == "scout":
		return "go"
	}
	return fmt.Sprintf("%d/%d", c.GetHave(), c.GetNeed())
}

func contractsScreenLines(m *model.Model, c *ContractsUI, w, h int, st Styler) []string {
	var rows []string
	add := func(l string) { rows = append(rows, fit(l, w)) }
	add(fmt.Sprintf(" You hold %d of 3 contracts. Take them at a contract board (G beside it); only your own kills count.", len(m.Self.Contracts)))
	add("")
	if len(m.Self.Contracts) == 0 {
		add(" You hold no contracts.")
	}
	rows = append(rows, contractRows(m, w, true, c.Sel, st)...)
	if len(rows) > h-2 {
		rows = rows[:h-2]
	}
	for len(rows) < h-2 {
		rows = append(rows, strings.Repeat(" ", w))
	}
	add(" Up / Down choose. X gives the contract up (anywhere). Hand a finished one in at its board.")
	add(" Esc closes.")
	return pad(rows[:min(len(rows), h)], w, h)
}

// contractsBox is the Contracts panel in the right column: its height for the
// contracts held (0 if none), and its lines.
func contractsBox(m *model.Model, w int, st Styler) (h int, lines []string) {
	if len(m.Self.Contracts) == 0 {
		return 0, nil
	}
	lines = contractRows(m, w, false, -1, st)
	return len(lines) + 2, lines
}

func overviewLines(m *model.Model, w, h int, st Styler) []string {
	legend := legendLines(m, w-2)
	foot := 2 + len(legend)
	rows := overviewRows(m, View{}, w, h-foot, st)
	add := func(l string) { rows = append(rows, fit(l, w)) }
	b := m.Map.GetBounds()
	sp := m.Self.Pos
	add(fmt.Sprintf(" %s, %dm by %dm. You: %dm E, %dm N of the south-west corner.", m.Map.GetName(),
		(b.GetMaxX()-b.GetMinX())/1000, (b.GetMaxY()-b.GetMinY())/1000, (sp.GetX()-b.GetMinX())/1000, (sp.GetY()-b.GetMinY())/1000))
	for _, l := range legend {
		add(" " + l)
	}
	add(" North is up. Only what you have seen is marked. M or Esc closes.")
	return pad(rows, w, h)
}

// eventBanner is the one-line banner of a world event that is on or just ended,
// "" when there is none. now is the game time.
func eventBanner(m *model.Model, v View, w int, st Styler) string {
	now := m.GameTimeMs
	if v.GameNow > now {
		now = v.GameNow
	}
	e := m.RecentEvent(now, eventWindowMs)
	if e == nil {
		return ""
	}
	name := e.Title()
	var text string
	switch e.Kind {
	case "started":
		text = fmt.Sprintf(" EVENT  %s: %s", name, e.Text)
	case "leak":
		text = fmt.Sprintf(" EVENT  %s: %d got through. %s", name, e.Leaks, e.Text)
	case "won":
		text = fmt.Sprintf(" EVENT OVER  %s won: %s", name, e.Text)
	default:
		text = fmt.Sprintf(" EVENT OVER  %s lost: %s", name, e.Text)
	}
	return st.Reverse(st.Bold(fit(text, w)))
}
