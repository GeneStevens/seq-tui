package screen

import (
	"fmt"
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

func menuLines(m *model.Model, v View, w, h int, st Styler) []string {
	var rows []string
	add := func(s string) { rows = append(rows, fit(s, w)) }
	add(" A private copy of this zone for your group; then press E at its door.")
	add("")
	if len(m.Presets) == 0 {
		add(" This zone offers no instances (or its map has not arrived).")
	}
	for i, p := range m.Presets {
		mark := "  "
		if i == v.Menu.Sel {
			mark = "> "
		}
		head := fmt.Sprintf(" %s%d  %s  (%s), door %s", mark, i+1, p.GetName(), p.GetId(), p.GetEntrance())
		detail := fmt.Sprintf("      %s; party %d; %s", timeMode(p, tickOf(m)), p.GetPartyCap(), p.GetDifficulty())
		if i == v.Menu.Sel {
			rows = append(rows, st.Reverse(fit(head, w)), st.Reverse(fit(detail, w)))
		} else {
			rows = append(rows, st.Bold(fit(head, w)), fit(detail, w))
		}
	}
	add("")
	add(" Up / Down or the number choose; Enter opens it; Esc closes the menu.")
	add(" Denied inside an instance, and when your group already has one open.")
	for len(rows) < h {
		rows = append(rows, strings.Repeat(" ", w))
	}
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

func instanceLines(m *model.Model, w, h int, st Styler) []string {
	var l1, l2, l3 string
	p := m.Current()
	mode := ""
	switch {
	case p != nil:
		mode = "real-time"
		if p.GetTimeMode() == "turn_based" {
			mode = "turn-based"
		}
	case m.TurnBased():
		mode = "turn-based"
	}
	switch {
	case m.InInstance():
		l1 = m.InstanceName()
		l2 = strings.Trim(mode+" | inside", " |")
		if p != nil {
			l3 = fmt.Sprintf("party up to %d | %s", p.GetPartyCap(), p.GetDifficulty())
		}
	case m.Opened != nil:
		p = m.Preset(m.Opened.Preset)
		l1 = presetName(m, m.Opened.Preset)
		mode = ""
		if p != nil {
			mode = "real-time"
			if p.GetTimeMode() == "turn_based" {
				mode = "turn-based"
			}
		}
		l2 = strings.Trim(mode+" | open", " |")
		if p != nil {
			l3 = "enter at " + p.GetEntrance() + " (E)"
		}
	default:
		l1 = presetName(m, m.Ended.Preset) + " ended"
		if m.Ended.Preset == "" {
			l1 = "Instance ended"
		}
		l2 = m.Ended.Reason
		l3 = "you were handed back"
	}
	rows := []string{st.Bold(fit(" "+l1, w)), fit(" "+l2, w), fit(" "+l3, w)}
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
		who := "none yet"
		if len(r.Submitted) > 0 {
			who = nameList(m, r.Submitted)
		}
		add(" Plans in: " + who)
	} else {
		if contains(r.ResolvedSubmitted, you) {
			add(" your own plan was played")
		} else {
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
		if h >= 8 {
			if in {
				add("  counts even if you disconnect")
			} else {
				add("  Enter submits; else safe default")
			}
		}
		room := h - len(rows)
		steps := v.Plan.Steps
		if len(steps) > room && room > 0 {
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
