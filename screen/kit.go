package screen

import (
	"fmt"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// abilityRows is the height of the ability bar's box: two rows of four slots
// and the row of effects.
const abilityRows = 5

// gameNow is the game time the countdowns read: the view's, else the last the
// server said.
func gameNow(m *model.Model, v View) int64 {
	if v.GameNow > 0 {
		return v.GameNow
	}
	return m.GameTimeMs
}

// secs says a span of game time in seconds, rounded up ("1.2s" under ten).
func secs(ms int64) string {
	if ms < 10_000 {
		return fmt.Sprintf("%.1fs", float64(max(ms, 0)+99)/1000)
	}
	if ms < 120_000 {
		return fmt.Sprintf("%ds", (ms+999)/1000)
	}
	return fmt.Sprintf("%dm", (ms+59_999)/60_000)
}

// wholeSecs is a span in whole seconds, rounded up; a long one in minutes.
func wholeSecs(ms int64) string {
	if ms >= 120_000 {
		return fmt.Sprintf("%dm", (ms+59_999)/60_000)
	}
	return fmt.Sprintf("%ds", (max(ms, 0)+999)/1000)
}

// className is a class id as a player reads it: the name the class menu gives
// it, else the id with a capital.
func className(m *model.Model, id string) string {
	if c := m.Class(id); c != nil && c.GetName() != "" {
		return c.GetName()
	}
	if id == "" {
		return ""
	}
	return strings.ToUpper(id[:1]) + id[1:]
}

// xpLines is the XP bar of the You panel: how far through the level, and how
// much is left.
func xpLine(m *model.Model, w int) string {
	s := m.Self
	if s.XPNext <= 0 {
		return fit(fmt.Sprintf(" XP %d (top level)", s.XP), w)
	}
	left := " XP "
	right := fmt.Sprintf(" %d to Lv %d", max(s.XPNext-s.XP, 0), s.Level+1)
	return fit(left+bar(s.XP-s.XPStart, s.XPNext-s.XPStart, max(w-width(left)-width(right), 0))+right, w)
}

// castLine is the cast bar of the You panel: the ability and the time left; when
// the last cast did not land, the line says so until another begins.
func castLine(m *model.Model, v View, w int, st Styler) string {
	s := m.Self
	if c := s.Cast; c != nil && c.LandsInRound > 0 {
		// A cast that waits for a later round has no countdown: game time stands
		// still between rounds.
		return fit(fmt.Sprintf(" Cast %s: lands in round %d", m.AbilityName(c.Ability), c.LandsInRound), w)
	}
	if c := s.Cast; c != nil {
		left := " Cast " + fit(m.AbilityName(c.Ability), max(min(w-18, 14), 6))
		left = strings.TrimRight(left, " ")
		now := gameNow(m, v)
		right := " " + secs(c.EndsAtMs-now)
		total := int64(0)
		if a := m.AbilityByID(c.Ability); a != nil {
			total = a.GetCastMs()
		}
		total = max(total, c.EndsAtMs-now)
		return fit(left+" "+bar(total-(c.EndsAtMs-now), total, max(w-width(left)-width(right)-1, 0))+right, w)
	}
	if e := s.LastCastEnd; e != nil {
		line := fmt.Sprintf(" ! %s %s: %s", m.AbilityName(e.Ability), e.Broken, castShort(e.Reason))
		if width(line) > w { // the room is short: drop the ability's name, the log has it
			line = fmt.Sprintf(" ! %s: %s", strings.ToUpper(e.Broken[:1])+e.Broken[1:], castShort(e.Reason))
		}
		return st.Red(st.Bold(fit(line, w)))
	}
	return ""
}

// castShort is why a cast did not land, in two or three words.
func castShort(reason string) string {
	switch reason {
	case "moved":
		return "you moved"
	case "damage":
		return "damage"
	case "interrupt":
		return "interrupted"
	}
	return model.CastWhy(reason)
}

// kitLines are the rows of the You panel for a character with abilities: name
// and class, hit points, mana (for those that have it), XP, and the cast.
func kitLines(m *model.Model, v View, w int, st Styler) []string {
	s := m.Self
	head := fmt.Sprintf(" %s  Lv %d %s", s.Name, s.Level, className(m, s.Class))
	hp := fmt.Sprintf(" HP %d/%d ", s.HP, s.MaxHP)
	hpLine := fit(hp+bar(s.HP, s.MaxHP, max(w-width(hp)-1, 0)), w)
	if s.Dead {
		hpLine = st.Red(st.Bold(fit(" DEAD  HP 0/"+fmt.Sprint(s.MaxHP), w)))
	} else if s.MaxHP > 0 && s.HP*4 <= s.MaxHP {
		hpLine = st.Red(hpLine)
	}
	rows := []string{st.Bold(withCrowns(head, s.Crowns, w)), hpLine}
	if s.MaxMana > 0 {
		mp := fmt.Sprintf(" MP %d/%d ", s.Mana, s.MaxMana)
		rows = append(rows, fit(mp+bar(s.Mana, s.MaxMana, max(w-width(mp)-1, 0)), w))
	}
	rows = append(rows, xpLine(m, w))
	switch {
	case s.Dead:
		last := st.Bold(fit(" You are dead. R: release", w))
		if s.Released {
			last = st.Bold(fit(fmt.Sprintf(" Released: back in %d s", (s.RespawnInMs+999)/1000), w))
		}
		rows = append(rows, last)
	case castLine(m, v, w, st) != "":
		rows = append(rows, castLine(m, v, w, st))
	case v.roomy:
		rows = append(rows, fit(" Atk "+onOff(s.AutoAttack)+" | "+targetName(m), w))
	default:
		rows = append(rows, fit(" Atk "+onOff(s.AutoAttack)+" Ret "+onOff(s.AutoRetaliate)+" | "+targetName(m), w))
	}
	if !s.Dead && v.roomy {
		rows = append(rows, retaliateLine(s, w))
	}
	return withAttackers(m, withStatus(m, rows, w, st), w, st)
}

// slotStatus says in a word whether the slot can be used now.
func slotStatus(g model.Gauge, left int64) string {
	switch g {
	case model.SlotReady:
		return "ready"
	case model.SlotCasting:
		return "busy"
	case model.SlotCooldown:
		return secs(left)
	case model.SlotGCD:
		return "gcd"
	case model.SlotNoMana:
		return "mana"
	case model.SlotNoTarget:
		return "tgt?"
	case model.SlotRange:
		return "range"
	}
	return "empty"
}

func slotCell(m *model.Model, v View, n, cw int, st Styler) string {
	g, left := m.Slot(n, gameNow(m, v))
	name, cost := "-", ""
	if a := m.AbilityByID(m.Self.Slots[n-1].Ability); a != nil {
		name = a.GetName()
		if cw >= 28 && a.GetManaCost() > 0 {
			cost = fmt.Sprintf("%dmp ", a.GetManaCost())
		}
	}
	if g == model.SlotEmpty {
		return st.Dim(fit(fmt.Sprintf("%d -", n), cw)) // an empty slot says nothing more
	}
	// The status follows its own slot's name, closer to it than to the next
	// slot, and the cells are told apart by the blank that ends each.
	right := cost + slotStatus(g, left)
	cell := fit(fmt.Sprintf("%d %s %s", n, fit(strings.TrimRight(name, " "), max(min(width(name), cw-3-width(right)-2), 1)), right), cw-1) + " "
	if g != model.SlotReady {
		return st.Dim(cell)
	}
	return st.Bold(cell)
}

func effectLabel(m *model.Model, e model.Effect) string {
	switch e.Kind {
	case "stun":
		return "Stunned"
	case "root":
		return "Rooted"
	case "snare":
		return "Snared"
	}
	return m.AbilityName(e.Ability)
}

func effectList(m *model.Model, v View, fx []model.Effect) string {
	if len(fx) == 0 {
		return "none"
	}
	now := gameNow(m, v)
	out := make([]string, len(fx))
	for i, e := range fx {
		out[i] = effectLabel(m, e)
		left := wholeSecs(e.EndsAtMs - now)
		switch {
		case e.EndsAtMs <= 0:
		case e.Approx:
			out[i] += " ~" + left
		default:
			out[i] += " " + left
		}
	}
	return strings.Join(out, ", ")
}

// abilityLines is the ability bar: the eight slots in two rows of four, then the
// effects on the character and on its target with the time left.
func abilityLines(m *model.Model, v View, w int, st Styler) []string {
	cw, extra := (w-1)/4, (w-1)%4
	var rows []string
	for r := 0; r < 2; r++ {
		line := " "
		for c := 0; c < 4; c++ {
			line += slotCell(m, v, r*4+c+1, cw+boolInt(c < extra), st)
		}
		rows = append(rows, line)
	}
	fx := " You: " + effectList(m, v, m.Self.Effects)
	if t := m.Self.Target; t != "" {
		fx += "   |   " + targetName(m) + ": " + effectList(m, v, m.TargetEffects[t])
	}
	return append(rows, fit(" "+strings.TrimSpace(fx), w))
}

// Create is the character creation screen, when it is open: the name typed so
// far, which class of the server's class menu the cursor is on, and which of
// the two the keys go to.
type Create struct {
	Name  string
	Class int
	// Classes says the cursor is on the class list; else it is on the name.
	Classes bool
	// Busy: the character is being made. Err is why the last try failed.
	Busy bool
	Err  string
}

// MaxName is the longest name the server takes.
const MaxName = 16

// rolesWords names a class's roles: one base role, or two for a hybrid.
func rolesWords(roles []string) string {
	return strings.Join(roles, " + ")
}

func createLines(m *model.Model, c *Create, w, h int, st Styler) []string {
	var rows []string
	add := func(s string) { rows = append(rows, fit(s, w)) }
	menu := m.Classes()
	add(" Make a character: a name and a class.")
	add("")
	cursor := "_"
	if c.Classes {
		cursor = ""
	}
	name := fmt.Sprintf(" Name   %s%s", c.Name, cursor)
	if !c.Classes {
		rows = append(rows, st.Reverse(fit(name+strings.Repeat(" ", max(MaxName-width(c.Name), 0))+"   3 to 16 letters", w)))
	} else {
		add(name + "   3 to 16 letters")
	}
	add("")
	add(" Class (the server's menu)")
	if len(menu) == 0 {
		add("  The server has sent no class menu yet.")
	}
	for i, cl := range menu {
		mark := "  "
		if i == c.Class {
			mark = "> "
		}
		line := fit(fmt.Sprintf(" %s%d  %-9s %s", mark, i+1, cl.GetName(), rolesWords(cl.GetRoles())), w)
		if i == c.Class && c.Classes {
			line = st.Reverse(line)
		} else if i == c.Class {
			line = st.Bold(line)
		}
		rows = append(rows, line)
	}
	add("")
	// The class under the cursor: what it is, and the kit it starts with, free.
	var tail []string
	switch {
	case c.Busy:
		tail = append(tail, fit(" Making the character…", w))
	case c.Err != "":
		tail = append(tail, st.Red(st.Bold(fit(" ! "+c.Err, w))))
	default:
		tail = append(tail, fit("", w))
	}
	tail = append(tail, fit(" Type the name. Tab or Down moves to the classes: Up / Down or 1-"+fmt.Sprint(max(len(menu), 1))+" choose.", w),
		fit(" Enter makes the character and plays it. Ctrl+C quits.", w))
	if c.Class >= 0 && c.Class < len(menu) {
		cl := menu[c.Class]
		for _, l := range wrap(cl.GetDescription(), w-2) {
			add(" " + l)
		}
		var kit []string
		for _, a := range cl.GetKit() {
			kit = append(kit, fmt.Sprintf("%s (%s)", a.GetName(), abilityDetail(a)))
		}
		if len(kit) > 0 {
			add(" Starts with, free:")
			for _, k := range kit {
				if len(rows)+len(tail) >= h {
					break
				}
				add("   " + k)
			}
		}
	}
	for len(rows)+len(tail) < h {
		rows = append(rows, strings.Repeat(" ", w))
	}
	rows = rows[:min(len(rows), h-len(tail))]
	return append(rows, tail...)
}

// Slots is the slot screen, when it is open: the slot the cursor is on and,
// while an ability is being chosen for it, the cursor in the list of what the
// character knows.
type Slots struct {
	Slot int // 0 to 7
	Pick bool
	Cur  int // the ability under the cursor while Pick
}

func abilityDetail(a *pb.AbilityInfo) string {
	cast := "instant"
	if a.GetCastMs() > 0 {
		cast = fmt.Sprintf("%.1f s cast", float64(a.GetCastMs())/1000)
	}
	d := cast
	if a.GetCooldownMs() > 0 {
		d += fmt.Sprintf(", cd %ds", a.GetCooldownMs()/1000)
	}
	if a.GetManaCost() > 0 {
		d += fmt.Sprintf(", %d mana", a.GetManaCost())
	}
	if a.GetRangeMm() > 0 {
		d += fmt.Sprintf(", range %dm", a.GetRangeMm()/1000)
	}
	return d + ", " + a.GetTarget()
}

func slotScreenLines(m *model.Model, v View, w, h int, st Styler) []string {
	sl := v.Slots
	var rows []string
	add := func(s string) { rows = append(rows, fit(s, w)) }
	add(" Keys 1-8 use the slots in play. Up / Down or 1-8 choose; Enter picks an ability.")
	if m.TurnBased() {
		add(" Backspace empties a slot, Esc closes. Here a change joins the plan (Enter submits).")
	} else {
		add(" Backspace empties a slot, Esc closes. Slots change out of combat only.")
	}
	for i := 0; i < model.Slots; i++ {
		id := m.Self.Slots[i].Ability
		planned := ""
		if p, ok := v.PlanSlots[i+1]; ok {
			id, planned = p, "  (in the plan)"
		}
		text := "(empty)"
		if a := m.AbilityByID(id); a != nil {
			text = fmt.Sprintf("%-16s %s", a.GetName(), abilityDetail(a))
		}
		text += planned
		mark := "  "
		if i == sl.Slot {
			mark = "> "
		}
		line := fit(fmt.Sprintf(" %s%d  %s", mark, i+1, text), w)
		if i == sl.Slot && !sl.Pick {
			line = st.Reverse(line)
		}
		rows = append(rows, line)
	}
	add("")
	if sl.Pick {
		add(fmt.Sprintf(" Choose the ability for slot %d: Up / Down, Enter sets it, Esc goes back.", sl.Slot+1))
	} else {
		add(" Known abilities")
	}
	room := h - len(rows)
	known := m.Self.Abilities
	from := 0
	if sl.Pick && sl.Cur >= room {
		from = sl.Cur - room + 1
	}
	for i := from; i < len(known) && len(rows) < h; i++ {
		a := known[i]
		where := "     "
		if s := m.SlotOf(a.GetId()); s > 0 {
			where = fmt.Sprintf("key %d", s)
		}
		mark := "  "
		if sl.Pick && i == sl.Cur {
			mark = "> "
		}
		line := fit(fmt.Sprintf(" %s%-16s %s  %s", mark, a.GetName(), where, abilityDetail(a)), w)
		if sl.Pick && i == sl.Cur {
			line = st.Reverse(line)
		}
		rows = append(rows, line)
	}
	return pad(rows[:min(len(rows), h)], w, h)
}

// withCrowns is the You panel's first line: the name and level, and the
// character's crowns at the right end ("| 84 crowns", or "| 84cr" when the room is
// short); the name gives way first.
func withCrowns(head string, crowns int64, w int) string {
	for _, c := range []string{fmt.Sprintf("| %d crowns", crowns), fmt.Sprintf("| %dcr", crowns)} {
		if width(head)+1+width(c) <= w {
			return fit(head, w-width(c)) + c
		}
	}
	c := fmt.Sprintf("| %dcr", crowns)
	return fit(head, max(w-width(c)-1, 1)) + " " + c
}
