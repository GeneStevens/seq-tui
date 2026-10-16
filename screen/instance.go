package screen

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// MaxPlanSteps is how many steps the server takes in a plan.
const MaxPlanSteps = 16

// Menu is the instance menu, when it is open: Sel is the preset the cursor is on.
type Menu struct{ Sel int }

// Plan is the plan being built for the open round, as lines of words. The
// plan itself belongs to the app; the screen only shows it.
type Plan struct {
	Steps []string
	// Sent: the server has taken this plan as the character's own for the
	// round (a receipt). Edits after that are not sent until Enter again.
	Sent bool
	// Dest is where the plan's last move ends, marked on the map; nil if it
	// has no move.
	Dest *pb.Pos
}

// duration says a span in the largest two units that matter.
func duration(d time.Duration) string {
	s := int64((d + 999*time.Millisecond) / time.Second)
	switch {
	case s <= 0:
		return "0 s"
	case s < 120:
		return fmt.Sprintf("%d s", s)
	case s < 2*3600:
		return unit(s/60, "min", s%60, "s")
	case s < 48*3600:
		return unit(s/3600, "h", s%3600/60, "min")
	}
	return unit(s/86400, "d", s%86400/3600, "h")
}

// unit writes "a A b B", or "a A" when b is zero.
func unit(a int64, an string, b int64, bn string) string {
	if b == 0 {
		return fmt.Sprintf("%d %s", a, an)
	}
	return fmt.Sprintf("%d %s %d %s", a, an, b, bn)
}

func ms(n int64) string { return duration(time.Duration(n) * time.Millisecond) }

// timeMode is the preset's time mode in words, with the round's length and the
// time the players have to plan.
func timeMode(p *pb.InstancePreset, tick time.Duration) string {
	if p.GetTimeMode() != "turn_based" {
		return "real-time"
	}
	return fmt.Sprintf("turn-based, %d-tick rounds (%s), %s to plan", p.GetRoundTicks(), duration(time.Duration(p.GetRoundTicks())*tick), ms(p.GetTurnDeadlineMs()))
}

// tickOf is the server's tick length, from the welcome (100 ms if unknown).
func tickOf(m *model.Model) time.Duration {
	if t := m.Welcome.GetTickMs(); t > 0 {
		return time.Duration(t) * time.Millisecond
	}
	return 100 * time.Millisecond
}

// perilWords says the preset's tier and, when there is one, the entry fee the
// server takes from the opener when it accepts the request (D44, D52).
func perilWords(p *pb.InstancePreset) string {
	words := ""
	if t := p.GetTier(); t != "" {
		words = "; " + strings.ToUpper(t[:1]) + t[1:]
	}
	if f := p.GetEntryFee(); f > 0 {
		words += ": entry fee " + crownsWord(f) + ", taken when it opens"
	}
	return words
}

func menuLines(m *model.Model, v View, w, h int, st Styler) []string {
	var rows []string
	add := func(s string) { rows = append(rows, fit(s, w)) }
	add(" A private copy of this zone for your group; then press E at its door.")
	add("")
	if len(m.Presets) == 0 {
		add(" This zone offers no instances (or its map has not arrived).")
	}
	// Three rows a preset; the list scrolls to keep the cursor in view.
	room := max((h-len(rows)-4)/3, 1)
	from, to := window(len(m.Presets), v.Menu.Sel, room)
	for i := from; i < to; i++ {
		p := m.Presets[i]
		mark := "  "
		if i == v.Menu.Sel {
			mark = "> "
		}
		head := fmt.Sprintf(" %s%d  %s  (%s), door %s", mark, i+1, p.GetName(), p.GetId(), p.GetEntrance())
		detail := fmt.Sprintf("      %s", timeMode(p, tickOf(m)))
		rules := fmt.Sprintf("      party %d; %s", p.GetPartyCap(), p.GetDifficulty())
		if sl := sleepWords(p.GetSleepLimitMs()); sl != "" {
			rules += "; " + sl
		}
		rules += perilWords(p)
		if i == v.Menu.Sel {
			rows = append(rows, st.Reverse(fit(head, w)), st.Reverse(fit(detail, w)), st.Reverse(fit(rules, w)))
		} else {
			rows = append(rows, st.Bold(fit(head, w)), fit(detail, w), fit(rules, w))
		}
	}
	for len(rows) < h-3 {
		rows = append(rows, strings.Repeat(" ", w))
	}
	rows = rows[:h-3]
	if len(m.Presets) > to-from {
		add(fmt.Sprintf(" Showing %d to %d of %d.", from+1, to, len(m.Presets)))
	} else {
		add("")
	}
	add(" Up / Down or the number choose; Enter opens it; Esc closes the menu.")
	add(" Denied inside an instance, and when your group already has one open.")
	return rows[:h]
}

// instanceShown says whether the Instance panel has anything to say.
func instanceShown(m *model.Model) bool {
	return m.InInstance() || m.Opened != nil || m.Ended != nil
}

func presetName(m *model.Model, id string) string {
	if p := m.Preset(id); p != nil {
		return p.GetName()
	}
	return id
}

// modeWord is a time mode as a player reads it.
func modeWord(mode string) string {
	if mode == "turn_based" {
		return "turn-based"
	}
	return "real-time"
}

// sleepWords says how long a turn-based instance may sleep unplayed before it
// is abandoned (D49); "" for a real-time one, which has no limit.
func sleepWords(ms int64) string {
	if ms <= 0 {
		return ""
	}
	return "sleeps up to " + duration(time.Duration(ms)*time.Millisecond)
}

// instanceInfoLines are the rows of the Instance panel: what the server says
// the instance is, not what its space is called (D49).
func instanceInfoLines(m *model.Model) []string {
	var rows []string
	switch {
	case m.Instance != nil:
		in := m.Instance
		rows = []string{m.InstanceName(), modeWord(in.GetTimeMode()) + " | inside",
			fmt.Sprintf("party up to %d | %s", in.GetPartyCap(), in.GetDifficulty())}
		if s := sleepWords(in.GetSleepLimitMs()); s != "" {
			rows = append(rows, s)
		}
	case m.InInstance(): // a round with no snapshot of the instance yet
		rows = []string{"Instance", "turn-based | inside"}
	case m.Opened != nil:
		rows = []string{presetName(m, m.Opened.Preset), "open"}
		if p := m.Preset(m.Opened.Preset); p != nil {
			rows[1] = modeWord(p.GetTimeMode()) + " | open"
			rows = append(rows, fmt.Sprintf("party up to %d | %s", m.Opened.PartyCap, p.GetDifficulty()))
			if s := sleepWords(p.GetSleepLimitMs()); s != "" {
				rows = append(rows, s)
			}
			rows = append(rows, "enter at "+p.GetEntrance()+" (E)")
		}
	default:
		name := presetName(m, m.Ended.Preset) + " ended"
		if m.Ended.Preset == "" {
			name = "Instance ended"
		}
		rows = []string{name, m.Ended.Reason, "you were handed back"}
	}
	return rows
}

// instanceRows is how many rows of text the Instance panel has to show.
func instanceRows(m *model.Model) int { return len(instanceInfoLines(m)) }

func instanceLines(m *model.Model, w, h int, st Styler) []string {
	var rows []string
	for i, l := range instanceInfoLines(m) {
		l = fit(" "+l, w)
		if i == 0 {
			l = st.Bold(l)
		}
		rows = append(rows, l)
	}
	return pad(rows[:min(len(rows), h)], w, h)
}

// roundTitle is the Round panel's title: the round and, while it is open, the
// countdown to its deadline.
func roundTitle(m *model.Model, v View) string {
	r := m.Round
	switch {
	case r == nil || r.Number == 0:
		return "Round"
	case r.Open:
		left := time.UnixMilli(r.DeadlineMs).Sub(v.Now)
		if left <= 0 {
			return fmt.Sprintf("Round %d, deadline now", r.Number)
		}
		return fmt.Sprintf("Round %d, %s left", r.Number, duration(left))
	}
	return fmt.Sprintf("Round %d ran: %s", r.Number, ranBy(r.ResolvedBy))
}

// roundLines is the Round panel: who has submitted and the plan being built.
func roundLines(m *model.Model, v View, w, h int, st Styler) []string {
	r := m.Round
	var rows []string
	add := func(s string) { rows = append(rows, fit(s, w)) }
	you := m.Self.ID
	if r == nil || r.Number == 0 {
		add(" waiting for the round…")
		return pad(rows, w, h)
	}
	if r.Open {
		// The server does not announce a plan to the others: what is known is the
		// list its last round event gave, and this character's own receipt.
		in := slices.Clone(r.Submitted)
		if v.Plan.Sent && !contains(in, you) {
			in = append(in, you)
		}
		who := "none yet"
		if len(in) > 0 {
			who = nameList(m, in)
		}
		if r.Participants > 0 {
			add(fmt.Sprintf(" %d of %d submitted", len(in), r.Participants))
			if line := partyLine(m, in, r.Participants); line != "" {
				add(line)
			} else if len(in) > 0 {
				add(" " + who)
			}
		} else {
			add(" Plans in: " + who)
		}
		if c := m.Self.Cast; c != nil && c.LandsInRound > 0 {
			add(fmt.Sprintf(" %s lands in round %d", m.AbilityName(c.Ability), c.LandsInRound))
		}
	} else {
		if r.ResolvedParticipants > 0 {
			add(fmt.Sprintf(" %d of %d submitted", len(r.ResolvedSubmitted), r.ResolvedParticipants))
		}
		switch {
		case len(r.Plays) > 0:
			for _, p := range r.Plays {
				add(fmt.Sprintf(" %s: %s", nameList(m, []string{p.Character}), playWords(p.Source)))
			}
		case contains(r.ResolvedSubmitted, you):
			add(" your own plan was played")
		default:
			add(" no plan from you: a stand-in")
			add(" or the safe default ran")
		}
		add(" next round opening…")
	}
	if r.Open {
		in := r.HasSubmitted(you) || v.Plan.Sent
		state := "not sent yet"
		if in {
			state = "accepted"
		}
		head := fmt.Sprintf(" Plan (%d/%d): %s", len(v.Plan.Steps), MaxPlanSteps, state)
		if len(v.Plan.Steps) == 0 && !in {
			head = " Plan: empty, not sent"
		}
		add(head)
		if h >= 7 {
			if in {
				add("  counts even if you disconnect")
			} else {
				add("  Enter submits; else safe default")
			}
		}
		room := h - len(rows)
		steps := v.Plan.Steps
		switch {
		case len(steps) <= room || room <= 0:
		case room == 1: // no room to say how many: the last step is the one that matters
			steps = steps[len(steps)-1:]
		default:
			skip := len(steps) - room + 1
			add(fmt.Sprintf("  …%d earlier", skip))
			steps = steps[skip:]
		}
		for i, s := range steps {
			if len(rows) >= h {
				break
			}
			add(fmt.Sprintf(" %2d %s", len(v.Plan.Steps)-len(steps)+i+1, s))
		}
	}
	return pad(rows, w, h)
}

// partyLine is the party and who of it has submitted: "[x] you  [ ] Bram". It
// is empty when the characters the model sees in the party are not the ones the
// round waits for (then only the names that submitted are known).
func partyLine(m *model.Model, in []string, participants int64) string {
	ids := []string{m.Self.ID}
	ids = append(ids, m.GroupIDs()...)
	// The others in the instance are in view; one that has just arrived is
	// among the entities until the next snapshot lists it with the group.
	for _, id := range m.EntityIDs() {
		if m.Entities[id].GetKind() == pb.EntityKind_ENTITY_KIND_CHARACTER {
			ids = append(ids, id)
		}
	}
	if participants < 2 || int64(len(ids)) != participants {
		return "" // alone, "1 of 1 submitted" says it all
	}
	var parts []string
	for _, id := range ids {
		mark := "[ ]"
		if contains(in, id) {
			mark = "[x]"
		}
		parts = append(parts, mark+" "+nameList(m, []string{id}))
	}
	return " " + strings.Join(parts, "  ")
}

// playWords is what a round's character played, in few words (D49).
func playWords(source string) string {
	switch source {
	case "own_plan":
		return "own plan"
	case "stand_in":
		return "stand-in plan"
	case "safe_default":
		return "safe default"
	}
	return source
}

func ranBy(reason string) string {
	switch reason {
	case "all_submitted":
		return "all submitted"
	case "deadline":
		return "deadline"
	case "instance_ended":
		return "instance ended"
	}
	return reason
}

func contains(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func nameList(m *model.Model, ids []string) string {
	var out []string
	for _, id := range ids {
		switch {
		case id == m.Self.ID:
			out = append(out, "you")
		case m.Group[id] != nil:
			out = append(out, m.Group[id].Entity.GetName())
		case m.Entities[id] != nil:
			out = append(out, m.Entities[id].GetName())
		default:
			out = append(out, id)
		}
	}
	return strings.Join(out, ", ")
}
