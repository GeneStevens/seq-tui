package screen

import (
	"fmt"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// D75 and D76 on the screen: danger is never hidden, and what the terminal
// cannot show with pictures it says with words, marks and colour. Everything
// here reads the model; the server decides what the marks and the hints are.

// reachHintMm is how close a door, a drop or a person must be for the key hint
// to offer the key. It is only a hint: the server decides (its reach is 5 m for
// a door and a drop, and an NPC's own for a person).
const reachHintMm = 5000

// momentWindowMs is how long, in game time, the level-up banner and the
// hand-in summary stay up.
const momentWindowMs = 20_000

// anyScreen says a screen or menu is open over the map.
func (v View) anyScreen() bool {
	return v.Help || v.Sessions != nil || v.Quest != nil || v.Create != nil || v.Social != nil || v.Slots != nil ||
		v.Inventory != nil || v.Traders != nil || v.Trade != nil || v.Talk != nil || v.Board != nil || v.Role != nil ||
		v.Contracts != nil || v.Overview || v.Menu != nil
}

// serviceOpen says one of the services is open: trade, trainer, bank, inventory,
// talk, the board or contracts.
func serviceOpen(v View) bool {
	return v.Inventory != nil || v.Traders != nil || v.Trade != nil || v.Talk != nil || v.Board != nil || v.Contracts != nil
}

// servicePanel says the screen is one of the services, which take the map's
// area and keep the rest of the screen (D75), and gives its title and lines.
func servicePanel(m *model.Model, v View, w, h int, st Styler) (title string, lines []string, ok bool) {
	switch {
	case v.Inventory != nil:
		title, lines = "Inventory and gear", inventoryScreenLines(m, v.Inventory, w, h, st)
	case v.Traders != nil:
		title, lines = "Traders", tradersScreenLines(m, v.Traders, w, h, st)
	case v.Trade != nil:
		title, lines = "Trade", tradeScreenLines(m, v.Trade, w, h, st)
	case v.Talk != nil:
		title, lines = "Talking", talkLines(m, v.Talk, w, h, st)
	case v.Board != nil:
		title, lines = "Contract board", boardLines(m, v, v.Board, w, h, st)
	case v.Contracts != nil:
		title, lines = "Contracts", contractsScreenLines(m, v.Contracts, w, h, st)
	default:
		return "", nil, false
	}
	// A refusal is shown in the panel being looked at as well as on the status
	// line, in place of the panel's last line (its key help).
	if v.Notice != "" && len(lines) > 0 {
		n := " " + v.Notice
		if v.NoticeBad {
			n = " ! " + v.Notice
		}
		if v.NoticeBad {
			lines[len(lines)-1] = st.Red(st.Bold(fit(n, w)))
		} else {
			lines[len(lines)-1] = fit(n, w)
		}
	}
	return title, lines, true
}

// joinNames says a few names in a sentence: "A", "A and B", "A and 2 more".
func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	}
	return fmt.Sprintf("%s and %d more", names[0], len(names)-1)
}

// alarmText is the line that raises the alarm (D75), "" when there is none:
// something is attacking the character and a screen is open over the map, or the
// mob that began is a danger (hard or deadly).
func alarmText(m *model.Model, v View) string {
	if !m.UnderAttack() {
		return ""
	}
	names := m.AttackerNames()
	a := m.Alert
	danger := a != nil && a.Danger && (len(m.Self.Attackers) == 0 || containsID(m.Self.Attackers, a.Mob))
	if !danger && !v.anyScreen() {
		return ""
	}
	if len(names) == 0 && a != nil {
		names = []string{a.Name}
	}
	who := joinNames(names)
	verb := "is"
	if len(names) > 1 {
		verb = "are"
	}
	text := fmt.Sprintf("⚠ %s %s attacking you", who, verb)
	if danger {
		text = fmt.Sprintf("⚠ DANGER: %s (level %d, %s) is attacking you", a.Name, a.Level, a.Con)
		if len(names) > 1 {
			text = fmt.Sprintf("⚠ DANGER: %s (level %d, %s) and %s attacking you", a.Name, a.Level, a.Con, joinNames(without(names, a.Name)))
		}
	}
	text += fmt.Sprintf(" — HP %d/%d", m.Self.HP, m.Self.MaxHP)
	if v.anyScreen() {
		return text + " — Esc to fight"
	}
	return text + " — F to fight"
}

func without(names []string, n string) []string {
	var out []string
	for _, s := range names {
		if s != n {
			out = append(out, s)
		}
	}
	return out
}

func containsID(ids []string, id string) bool {
	for _, s := range ids {
		if s == id {
			return true
		}
	}
	return false
}

// alarmed says the boxes of a screen over the map are drawn with a red border.
func alarmed(m *model.Model) bool { return m.UnderAttack() }

// vitalsStrip is the one line every screen that takes the whole window keeps
// (D75): your health, your target and what is attacking you.
func vitalsStrip(m *model.Model, w int, st Styler) string {
	s := m.Self
	parts := []string{fmt.Sprintf("HP %d/%d", s.HP, s.MaxHP)}
	if s.Dead {
		parts[0] = "DEAD"
	}
	names := m.AttackerNames()
	other := true // what attacks is not (only) the target
	if t := targetName(m); t != "none" {
		if e := m.Entities[s.Target]; e != nil {
			t = fmt.Sprintf("%s %d/%d", t, e.GetHp(), e.GetMaxHp())
		}
		if len(names) == 1 && containsID(s.Attackers, s.Target) {
			t += ", attacking you"
			other = false
		}
		parts = append(parts, "Target: "+t)
	} else {
		parts = append(parts, "Target: none")
	}
	switch {
	case len(names) > 0 && other:
		parts = append(parts, "In combat: attacked by "+joinNames(names))
	case s.InCombat:
		parts = append(parts, "In combat")
	}
	line := fit(" "+strings.Join(parts, " | "), w)
	if m.UnderAttack() || (s.MaxHP > 0 && s.HP*4 <= s.MaxHP) {
		return st.Red(st.Bold(line))
	}
	return st.Bold(line)
}

// youTitle is the title of the You panel.
func youTitle(m *model.Model) string {
	if m.Self.InCombat {
		return "You: in combat"
	}
	return "You"
}

// hintLine is the line of keys that change with what the character is next to
// (D76): the door, the person, the drop, a mob in view, the body to release.
func hintLine(m *model.Model, v View, w int, st Styler) string {
	var parts []string
	switch {
	case m.Self.Dead:
		parts = append(parts, "R: release your body and wake at your bind point")
	case m.Self.Pos == nil:
	default:
		if d := nearestDoorHint(m); d != nil {
			parts = append(parts, "E: "+doorWords(d))
		}
		if n := nearestNPCHint(m); n != nil {
			parts = append(parts, "G: talk to "+model.NPCLabel(n))
		}
		if d := nearestDropHint(m); d != nil {
			parts = append(parts, "L: loot "+m.DropName(d))
		}
		if mobsInView(m) > 0 {
			switch {
			case m.Self.Target != "" && m.Self.AutoAttack:
				parts = append(parts, "F: stop attacking · Esc: clear the target")
			case m.Self.Target != "":
				parts = append(parts, "F: attack · Tab: next target")
			default:
				parts = append(parts, "Tab: target · F: attack")
			}
		}
	}
	if len(parts) == 0 {
		parts = []string{"Walk: arrows or WASD · M: map · ?: help"}
	}
	return st.Dim(fit(" "+strings.Join(parts, "  ·  "), w))
}

// doorWords is what a door does, from the server's label ("East Gate — to the Thornwick Downs").
func doorWords(d *pb.Door) string {
	label := d.GetLabel()
	if label == "" {
		label = d.GetName()
	}
	label = strings.Replace(label, " — to ", " to ", 1)
	if d.GetKind() == pb.DoorKind_DOOR_KIND_INSTANCE_ENTRANCE {
		return "enter " + label
	}
	return "go through the " + label
}

func nearestDoorHint(m *model.Model) *pb.Door {
	var best *pb.Door
	var bd int64
	for _, d := range m.Map.GetDoors() {
		if dd := dist(d.GetPos().GetX(), d.GetPos().GetY(), m.Self.Pos.GetX(), m.Self.Pos.GetY()); dd <= reachHintMm && (best == nil || dd < bd) {
			best, bd = d, dd
		}
	}
	return best
}

func nearestNPCHint(m *model.Model) *pb.NpcInfo {
	var best *pb.NpcInfo
	var bd int64
	for _, n := range m.KnownNPCs() {
		if !m.InReach(n) {
			continue
		}
		if dd := dist(n.GetPos().GetX(), n.GetPos().GetY(), m.Self.Pos.GetX(), m.Self.Pos.GetY()); best == nil || dd < bd {
			best, bd = n, dd
		}
	}
	return best
}

func nearestDropHint(m *model.Model) *pb.LootDropState {
	var best *pb.LootDropState
	var bd int64
	for _, id := range m.DropIDs() {
		d := m.Drops[id]
		if !d.GetCanLoot() {
			continue
		}
		if dd := dist(d.GetPos().GetX(), d.GetPos().GetY(), m.Self.Pos.GetX(), m.Self.Pos.GetY()); dd <= reachHintMm && (best == nil || dd < bd) {
			best, bd = d, dd
		}
	}
	return best
}

func mobsInView(m *model.Model) int {
	n := 0
	for _, e := range m.Entities {
		if e.GetKind() == pb.EntityKind_ENTITY_KIND_MOB && !e.GetDead() {
			n++
		}
	}
	return n
}

// objectiveWords is the objective on the status line: its name, direction and
// distance, "→ Talk to Gorran Stoutarm, 23m W".
func objectiveWords(m *model.Model) string {
	h := m.Objective()
	if h == nil || m.Self.Pos == nil {
		return ""
	}
	label := h.GetLabel()
	if h.GetZone() != "" {
		label += " (" + h.GetZone() + ")"
	}
	at := h
	if w := m.Waypoint(); w != nil && w.GetPos() != nil {
		at = w
	}
	if at.GetPos() == nil {
		return "→ " + label
	}
	sp := m.Self.Pos
	return fmt.Sprintf("→ %s, %s %s", label, metres(dist(at.GetPos().GetX(), at.GetPos().GetY(), sp.GetX(), sp.GetY())), bearing(at.GetPos().GetX()-sp.GetX(), at.GetPos().GetY()-sp.GetY()))
}

// momentBanner is the banner of a level just gained (D76), as rows: the new
// level with the health and mana gained, then what can be trained and where.
func momentBanner(m *model.Model, v View, w int, st Styler) []string {
	mo := m.LevelUp
	if mo == nil || mo.Level == nil {
		return nil
	}
	now := m.GameTimeMs
	if v.GameNow > now {
		now = v.GameNow
	}
	if now-mo.AtMs > momentWindowMs {
		return nil
	}
	return levelBannerRows(m, mo.Level, w, st)
}

func levelBannerRows(m *model.Model, l *pb.LevelUp, w int, st Styler) []string {
	head := fmt.Sprintf(" LEVEL UP! You reached level %d. Health %d → %d", l.GetLevel(), l.GetHpBefore(), l.GetHpAfter())
	if l.GetManaAfter() > 0 || l.GetManaBefore() > 0 {
		head += fmt.Sprintf(", mana %d → %d", l.GetManaBefore(), l.GetManaAfter())
	}
	for _, g := range l.GetGains() {
		head += fmt.Sprintf(", %s %+d", strings.ToLower(g.GetName()), g.GetAmount())
	}
	rows := []string{st.Reverse(st.Bold(fit(head, w)))}
	if tr := l.GetTrainable(); len(tr) > 0 {
		var parts []string
		for _, t := range tr {
			p := fmt.Sprintf("%s from %s", t.GetName(), t.GetTrainerName())
			if t.GetTrainerAt() != "" {
				p += " in " + t.GetTrainerAt()
			}
			parts = append(parts, fmt.Sprintf("%s (%s)", p, crownsWord(t.GetCost())))
		}
		rows = append(rows, st.Reverse(fit(" You can now train: "+strings.Join(parts, "; "), w)))
	}
	return rows
}

// handInLines is the summary of a contract or quest step just handed in, for the
// panel that did it (D76), nil when there is none within the window.
func handInLines(m *model.Model, v View, w int, st Styler) []string {
	mo := m.HandIn
	if mo == nil || mo.Hand == nil {
		return nil
	}
	now := m.GameTimeMs
	if v.GameNow > now {
		now = v.GameNow
	}
	if now-mo.AtMs > momentWindowMs {
		return nil
	}
	h := mo.Hand
	var pay []string
	if h.GetXp() > 0 {
		pay = append(pay, fmt.Sprintf("%d XP", h.GetXp()))
	}
	if h.GetCrowns() > 0 {
		pay = append(pay, crownsWord(h.GetCrowns()))
	}
	if h.GetItem() != "" {
		name := h.GetItemName()
		if name == "" {
			name = m.ItemName(h.GetItem())
		}
		pay = append(pay, name)
	}
	what := "contract"
	if h.GetKind() == "quest_step" {
		what = "quest step"
	}
	line := fmt.Sprintf(" Handed in the %s %s: %s", what, h.GetName(), strings.Join(pay, ", "))
	if len(pay) == 0 {
		line = fmt.Sprintf(" Handed in the %s %s", what, h.GetName())
	}
	if l := h.GetLevelUp(); l != nil {
		line += fmt.Sprintf(". You reached level %d!", l.GetLevel())
	}
	return []string{st.Green(st.Bold(fit(line, w)))}
}

// legendLines is the key to the map's glyphs and marks (D76), wrapped to w.
func legendLines(m *model.Model, w int) []string {
	rows := [][]string{
		{"@ you", "m mob", "M target", "x dead", "p player", "r rare", "* loot", "~ not yours"},
		{"T trainer", "$ vendor", "B banker", "H binder", "g guard", "Q quest person", "& board", "D door"},
		{"! new for you (gold)", "? hand in ready (gold)", "? in progress (grey)", "gold m quest mob", "gold D next door"},
		{"c camp: where mobs are found (dim)", "░ zone edge: walking ends", "◄ ► ▲ ▼ the objective, off the map"},
		{"% your marker", "; scout marker", "v event spawn", "G event goal", "+ plan end"},
		{"Ground  # building  ^ rock  ≈ water  : ford  = bridge  \" forest  · road  . floor"},
	}
	var out []string
	for _, r := range rows {
		line := ""
		for _, item := range r {
			switch {
			case line == "":
				line = item
			case width(line)+2+width(item) <= w:
				line += "  " + item
			default:
				out = append(out, line)
				line = item
			}
		}
		out = append(out, line)
	}
	return out
}
