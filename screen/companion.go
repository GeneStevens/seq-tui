package screen

import (
	"fmt"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq/agentkit"
)

// The Quest panel, the Companion panel, the Party panel for a second character
// on a stance, and the screen where the recruit's role is chosen (SEQ-049, D35,
// D42, D46). Everything shown is what the server said; the protocol tells no
// mana for another character, so none is drawn.

// Role is the screen for choosing the recruit's role at the quest giver, when
// it is open.
type Role struct {
	NPC string
	Sel int
}

// DuoView is the account's second character, played by a stance from this
// client (D42): its id, whether it is in play, its stance and standing order
// ("attack", "move", "stop" or ""). Its level and hit points are the group's.
type DuoView struct {
	ID     string
	Active bool
	Stance string
	Order  string
	// Target is what the character itself is fighting (its own state, which only
	// its own connection knows), "" for nothing.
	Target string
}

// panel is a boxed block of the right column.
type panel struct {
	title string
	lines []string
}

// helperPanels are the Companion, Party and Quest panels the character has now,
// in the order they claim room.
func helperPanels(m *model.Model, v View, iw int, st Styler) []panel {
	var out []panel
	both := companionShown(m) && v.Duo != nil
	if companionShown(m) {
		title := "Companion, " + m.Companion().GetRole()
		if both && !v.KeysToDuo {
			title += " [keys]"
		}
		out = append(out, panel{title, companionLines(m, iw, st)})
	}
	if v.Duo != nil {
		title := "Party, 2nd character"
		if both && v.KeysToDuo {
			title += " [keys]"
		}
		out = append(out, panel{title, duoLines(m, v.Duo, iw, st)})
	}
	if m.QuestShown() {
		out = append(out, panel{"Quest", questLines(m, iw, st)})
	}
	return out
}

func companionShown(m *model.Model) bool {
	c := m.Companion()
	return c.GetUnlocked() && c.GetRole() != ""
}

// controls are the keys of a helper, in the words of the panel.
func controls(w int) []string {
	if w >= 36 {
		return []string{
			fit(" P call or send away   H stance", w),
			fit(" Y attack my target    U come here", w),
			fit(" J stop    Z drop the order", w),
		}
	}
	return nil // a narrow panel has no room for them: ? lists them
}

func companionLines(m *model.Model, w int, st Styler) []string {
	c := m.Companion()
	g := m.CompanionMember()
	name := "your recruit"
	level := m.Self.Level
	if g != nil {
		name, level = g.Entity.GetName(), g.Entity.GetLevel()
	}
	rows := []string{st.Bold(fit(fmt.Sprintf(" %s  Lv %d", name, level), w))}
	switch {
	case g != nil && g.Entity.GetDead():
		rows = append(rows, st.Red(fit(" DEAD (it stands up again soon)", w)))
	case g != nil:
		hp := fmt.Sprintf(" HP %d/%d ", g.Entity.GetHp(), g.Entity.GetMaxHp())
		line := fit(hp+bar(g.Entity.GetHp(), g.Entity.GetMaxHp(), max(w-width(hp)-1, 0)), w)
		if g.Entity.GetMaxHp() > 0 && g.Entity.GetHp()*4 <= g.Entity.GetMaxHp() {
			line = st.Red(line)
		}
		rows = append(rows, line)
	case c.GetSummoned():
		rows = append(rows, fit(" on its way: it comes when you join a space", w))
	default:
		rows = append(rows, fit(" not with you: P calls it", w))
	}
	action := model.HelperAction(m, g, c.GetOrder(), c.GetStance())
	if g == nil {
		action = "away"
	}
	rows = append(rows, fit(" stance "+c.GetStance(), w), fit(" "+action, w))
	return append(rows, controls(w)...)
}

func duoLines(m *model.Model, d *DuoView, w int, st Styler) []string {
	g := m.InView(d.ID)
	name, level := d.ID, int64(0)
	if g != nil {
		name, level = g.Entity.GetName(), g.Entity.GetLevel()
	} else if n := m.NameOf(d.ID); n != "" {
		name = n
	}
	head := " " + name
	if level > 0 {
		head = fmt.Sprintf(" %s  Lv %d", name, level)
	}
	rows := []string{st.Bold(fit(head, w))}
	switch {
	case !d.Active:
		rows = append(rows, fit(" out of play: P brings it in", w))
	case g != nil && g.Entity.GetDead():
		rows = append(rows, st.Red(fit(" DEAD", w)))
	case g != nil:
		hp := fmt.Sprintf(" HP %d/%d ", g.Entity.GetHp(), g.Entity.GetMaxHp())
		line := fit(hp+bar(g.Entity.GetHp(), g.Entity.GetMaxHp(), max(w-width(hp)-1, 0)), w)
		if g.Entity.GetMaxHp() > 0 && g.Entity.GetHp()*4 <= g.Entity.GetMaxHp() {
			line = st.Red(line)
		}
		rows = append(rows, line)
	default:
		rows = append(rows, fit(" not in your group (a stance follows)", w))
	}
	action := "out of play"
	if d.Active {
		own := &model.Member{Target: d.Target}
		if g != nil {
			own.Entity = g.Entity
		}
		action = model.HelperAction(m, own, d.Order, d.Stance)
	}
	rows = append(rows, fit(" stance "+d.Stance, w), fit(" "+action, w))
	return append(rows, controls(w)...)
}

// questLines are the steps of the Watch Captain's quest as the character has
// seen them, and what to do next.
func questLines(m *model.Model, w int, st Styler) []string {
	c := m.Companion()
	giver := "the quest giver"
	if id := m.QuestGiver(); id != "" {
		giver = m.NPCName(id)
	}
	var rows []string
	ready, held, offered := false, "", false
	for _, s := range m.QuestSteps() {
		mark, right := "[ ]", ""
		switch s.State {
		case model.StepOffered:
			offered = true
		case model.StepHeld:
			mark, right = "[>]", fmt.Sprintf("%d/%d", s.Have, s.Need)
			if s.Kind == "scout" {
				right = "scout"
			}
			if held == "" {
				held = s.Description
			}
		case model.StepReady:
			mark, right, ready = "[!]", "READY", true
		case model.StepDone:
			mark, right = "[x]", "done"
		}
		room := max(w-width(right)-6, 4)
		rows = append(rows, fit(padTo(" "+mark+" "+fit(s.Name, room), w-width(right)-1)+right, w))
	}
	var next string
	switch {
	case c.GetUnlocked() && c.GetRole() == "":
		next = "Quest done! Choose your recruit's role: G at " + giver + "."
	case ready:
		next = "Hand it in: G at " + giver + "."
	case held != "":
		next = held
	case offered:
		next = giver + " has work for you: G."
	default:
		next = "Talk to " + giver + " (G)."
	}
	rows = append(rows, wrapped(next, w, " ")...)
	return rows
}

// padTo pads s with spaces to n cells.
func padTo(s string, n int) string {
	if width(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-width(s))
}

// roleLines is the screen for choosing the recruit's role.
func roleLines(m *model.Model, r *Role, w, h int, st Styler) []string {
	var rows []string
	add := func(l string) { rows = append(rows, fit(l, w)) }
	add(st.Bold(" Your recruit of the Watch, from " + m.NPCName(r.NPC)))
	add("")
	if d := m.Said(r.NPC); d != nil {
		for _, l := range d.Lines {
			rows = append(rows, wrapped("“"+l+"”", w, " ")...)
		}
		add("")
	}
	cur := m.Companion().GetRole()
	for i, role := range agentkit.CompanionRoles {
		mark, note := "  ", ""
		if role == cur {
			note = "  (your recruit now)"
		}
		if i == r.Sel {
			mark = "> "
		}
		line := fit(fmt.Sprintf(" %s%d  %s%s", mark, i+1, role, note), w)
		if i == r.Sel {
			line = st.Reverse(line)
		}
		rows = append(rows, line)
	}
	add("")
	add(" Number keys or Up / Down choose, Enter sends it. Esc goes back.")
	add(" The server answers on the status line; every refusal says why.")
	if len(rows) > h {
		rows = rows[:h]
	}
	return pad(rows, w, h)
}
