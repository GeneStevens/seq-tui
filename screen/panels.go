package screen

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

const (
	targetRows = 5
)

// rightColumn stacks the panels that fit, in order: You, Instance (inside one,
// or once one is open or has ended), Round (turn-based), Target, Nearby. What a
// small screen cannot hold is left out, never squeezed; spare rows go to the
// Instance panel, or to the Round panel when Nearby does not fit.
func rightColumn(m *model.Model, v View, rw, th int, st Styler) []string {
	iw := rw - 2
	v.roomy = th >= roomyRows
	you := youLines(m, v, iw, st)
	rows := box(youTitle(m), rw, len(you)+2, you, st)
	left := th - len(rows)
	inst, round := 0, 0
	if instanceShown(m) {
		inst = max(instanceRows(m, iw-1), 2) + 2
	}
	turn := m.TurnBased()
	if turn {
		round = 7
	}
	if left < inst+round {
		// Too little room for both: the plan comes first, the Instance panel
		// takes what is left of the column or is left out.
		switch {
		case turn:
			round = min(round, max(min(5, left), left-inst))
			inst = left - round
		default:
			inst = left
		}
		if inst < 3 {
			inst = 0
		}
		if round < 3 {
			round = 0
		}
		left = 0
	} else {
		left -= inst + round
	}
	// The Companion, Party and Quest panels, when the character has them: they
	// come before the Target and Nearby panels, which give way to them.
	var helpers []panel
	for _, hp := range helperPanels(m, v, iw, st) {
		h := min(len(hp.lines)+2, left)
		if h < 3 {
			break
		}
		helpers, left = append(helpers, panel{hp.title, hp.lines[:min(len(hp.lines), h-2)]}), left-h
	}
	// The Target panel gives up its last line (the fight) when one row is short.
	target := left >= targetRows-1
	targetH := min(targetRows, left)
	if target {
		left -= targetH
	}
	if turn && target && targetH == targetRows && left > 5 {
		more := min(left-5, 4) // a longer plan list, while Nearby keeps 5 rows
		round += more
		left -= more
	}
	// The Contracts panel, when the character holds any and Nearby keeps its rows.
	consH, consLines := contractsBox(m, iw, st)
	if consH > 0 && left-consH >= 5 {
		left -= consH
	} else {
		consH = 0
	}
	nearby := left >= 5
	switch {
	case nearby:
	case turn:
		round += left
	case inst > 0:
		inst += left
	case target && targetH == targetRows:
		targetH += left
	}
	if inst > 0 {
		rows = append(rows, box("Instance", rw, inst, instanceLines(m, iw, inst-2, st), st)...)
	}
	if turn && round > 0 {
		rows = append(rows, box(roundTitle(m, v), rw, round, roundLines(m, v, iw, round-2, st), st)...)
	}
	for _, hp := range helpers {
		rows = append(rows, box(hp.title, rw, len(hp.lines)+2, hp.lines, st)...)
	}
	if target {
		tl := targetLines(m, iw, st)
		rows = append(rows, box("Target", rw, targetH, pad(tl[:min(len(tl), targetH-2)], iw, targetH-2), st)...)
	}
	if consH > 0 {
		rows = append(rows, box("Contracts (K)", rw, consH, consLines, st)...)
	}
	if nearby {
		h := th - len(rows)
		rows = append(rows, box("Nearby", rw, h, nearbyLines(m, iw, h-2, st), st)...)
	}
	for len(rows) < th {
		rows = append(rows, strings.Repeat(" ", rw))
	}
	return rows
}

// roomyRows is how tall the right column must be for the You panel to give
// auto-retaliate (D85) a row of its own; a shorter one folds it into the attack
// row, so the panels below keep their lines on an 80 x 24 screen.
const roomyRows = 30

// retaliateLine is the auto-retaliate setting (D85) and the key that turns it.
func retaliateLine(s model.Self, w int) string {
	return fit(" Auto-retaliate: "+onOff(s.AutoRetaliate)+"  (Ctrl+R)", w)
}

func onOff(b bool) string {
	if b {
		return "ON"
	}
	return "off"
}

func youLines(m *model.Model, v View, w int, st Styler) []string {
	if m.HasKit() {
		return kitLines(m, v, w, st)
	}
	s := m.Self
	name := s.Name
	if name == "" {
		name = "(no character)"
	}
	head := fmt.Sprintf(" %s  %s  XP %d", name, levelWord(s), s.XP)
	hp := fmt.Sprintf(" HP %d/%d ", s.HP, s.MaxHP)
	hpLine := fit(hp+bar(s.HP, s.MaxHP, max(w-width(hp)-1, 0)), w)
	if s.Dead {
		hpLine = fit(" DEAD  HP 0/"+fmt.Sprint(s.MaxHP), w)
		hpLine = st.Red(st.Bold(hpLine))
	} else if s.MaxHP > 0 && s.HP*4 <= s.MaxHP {
		hpLine = st.Red(hpLine)
	}
	last := fit(" Auto-attack: "+onOff(s.AutoAttack), w)
	if !v.roomy {
		last = fit(" Atk "+onOff(s.AutoAttack)+" | Retaliate "+onOff(s.AutoRetaliate), w)
	}
	if s.Dead {
		// Nothing else can be done but release (D41); say so, and what is
		// happening once it is done.
		last = st.Bold(fit(" You are dead. R: release", w))
		if s.Released {
			last = st.Bold(fit(fmt.Sprintf(" Released: back in %d s", (s.RespawnInMs+999)/1000), w))
		}
	}
	rows := []string{
		st.Bold(withCrowns(head, s.Crowns, w)),
		hpLine,
		fit(" Target: "+targetName(m), w),
		last,
	}
	if !s.Dead && v.roomy {
		rows = append(rows, retaliateLine(s, w))
	}
	return withAttackers(m, withStatus(m, rows, w, st), w, st)
}

// withAttackers adds the line that names what is attacking the character (D75).
func withAttackers(m *model.Model, rows []string, w int, st Styler) []string {
	if names := m.AttackerNames(); len(names) > 0 {
		return append(rows, st.Red(fit(" Attacked by "+joinNames(names), w)))
	}
	return rows
}

type nearby struct {
	glyph  rune
	name   string
	d      int64
	dx, dy int64
	hp     string
	target bool
	id     string
	note   string // after the distance: "not lootable"
	tag    string // what the objective hints say of it: "[step 2]", "[2/3]", "[new]"
	quest  bool   // it matters to the quest or a held contract: Nearby lists it first
	title  string // an NPC's title, shown beside the name
	shut   bool   // a door the server says cannot be used: dim, "(shut)"
}

func nearbyList(m *model.Model) []nearby {
	sp := m.Self.Pos
	mk := func(g rune, id, name string, p *pb.Pos, hp string) nearby {
		return nearby{glyph: g, id: id, name: name, d: dist(p.GetX(), p.GetY(), sp.GetX(), sp.GetY()),
			dx: p.GetX() - sp.GetX(), dy: p.GetY() - sp.GetY(), hp: hp, target: id == m.Self.Target}
	}
	var out []nearby
	ent := func(e *pb.EntityState) {
		hp := fmt.Sprintf("%d/%d", e.GetHp(), e.GetMaxHp())
		if e.GetDead() {
			hp = "dead"
		}
		n := mk(entityGlyph(e), e.GetId(), e.GetName(), e.GetPos(), hp)
		if h := m.QuestMob(e.GetName()); h != nil && e.GetKind() == pb.EntityKind_ENTITY_KIND_MOB && !e.GetDead() {
			n.quest, n.tag = true, m.HintTag(h)
			if n.tag == "" {
				n.tag = "[quest]"
			}
		}
		if band := e.GetCon(); band != "" && !e.GetDead() {
			n.note = band // the server's difficulty band for a mob, in words (D70)
		}
		if model.IsRare(e) && !e.GetDead() {
			n.note = strings.TrimSpace(n.note + " RARE")
		}
		out = append(out, n)
	}
	for _, id := range m.GroupIDs() {
		ent(m.Group[id].Entity)
	}
	for _, id := range m.EntityIDs() {
		ent(m.Entities[id])
	}
	for _, id := range m.DropIDs() {
		d := m.Drops[id]
		name := m.DropName(d)
		if mk := m.Mark(d.GetItem()); mk != "" {
			name += " " + mk // against what is worn (D78)
		}
		n := mk(glyphDrop, id, name, d.GetPos(), "")
		if !d.GetCanLoot() {
			n.glyph, n.note = glyphLocked, "not lootable"
		}
		out = append(out, n)
	}
	for _, n := range m.KnownNPCs() {
		gl, _ := npcMark(m, n)
		x := mk(gl, n.GetId(), n.GetName(), n.GetPos(), "")
		if !strings.Contains(n.GetName(), n.GetTitle()) {
			x.title = n.GetTitle()
		}
		if n.GetTitle() == "" {
			x.hp = n.GetRole() // an NPC from an older server has a role and no title
		}
		if mark := m.MarkOf(n.GetId()); mark != "" {
			x.quest = true
			x.tag = m.HintTag(m.HintFor(mark, n.GetId()))
		}
		out = append(out, x)
	}
	for _, d := range m.Map.GetDoors() {
		name := d.GetName()
		if name == "" {
			name = m.MarkerName(d.GetId())
		}
		x := mk(glyphDoor, d.GetId(), name, d.GetPos(), "")
		if m.IsNextDoor(d.GetId()) {
			x.quest, x.tag = true, "[next door]"
		}
		if usable, _ := m.DoorUsable(d.GetId()); !usable {
			x.shut = true
		}
		out = append(out, x)
	}
	held := map[string]bool{}
	for _, c := range m.Self.Contracts {
		if p := m.Marker(c); p != nil {
			held[c.GetMarker()] = true
			out = append(out, mk(glyphMarker, "marker-"+c.GetMarker(), m.MarkerName(c.GetMarker()), p, "scout marker"))
		}
	}
	// The zone's other named spots, by the names its pack gives them (D56).
	for _, k := range m.Map.GetMarkers() {
		if k.GetName() == "" || held[k.GetId()] {
			continue
		}
		switch k.GetKind() {
		case "scout":
			out = append(out, mk(glyphScout, "marker-"+k.GetId(), k.GetName(), k.GetPos(), "scout marker"))
		case "event_spawn":
			out = append(out, mk(glyphSpawn, "marker-"+k.GetId(), k.GetName(), k.GetPos(), "raiders gather"))
		case "event_goal":
			out = append(out, mk(glyphGoal, "marker-"+k.GetId(), k.GetName(), k.GetPos(), "raiders head here"))
		}
	}
	// What matters to the quest and the held contracts comes first (D76).
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].quest != out[j].quest {
			return out[i].quest
		}
		if out[i].d != out[j].d {
			return out[i].d < out[j].d
		}
		return out[i].id < out[j].id
	})
	return out
}

func nearbyLines(m *model.Model, w, h int, st Styler) []string {
	rows := make([]string, 0, h)
	if m.Self.Pos == nil {
		return pad(rows, w, h)
	}
	list := nearbyList(m)
	for i, n := range list {
		if len(rows) == h {
			break
		}
		right := metres(n.d) + " " + bearing(n.dx, n.dy)
		switch {
		case n.shut:
			right += " (shut)"
		case n.tag != "":
			right += " " + n.tag
		default:
			if n.hp != "" {
				right += " " + n.hp
			}
			if n.note != "" {
				right += " " + n.note
			}
		}
		right += " "
		g := n.glyph
		if n.target {
			g = upper(g)
		}
		if i < len(list)-1 && len(rows)+1 >= h {
			rows = append(rows, fit(fmt.Sprintf(" +%d more", len(list)-i), w))
			break
		}
		rows = append(rows, nearbyRow(n, g, right, w, st))
	}
	if len(list) == 0 {
		rows = append(rows, fit(" nothing in view", w))
	}
	return pad(rows, w, h)
}

// nearbyRow is one entry of Nearby on one line: the glyph, the name, an NPC's
// title beside it in dim, then the distance and what is known of it. The title
// is cut before the name is, and goes when it would leave the name too little.
func nearbyRow(n nearby, g rune, right string, w int, st Styler) string {
	if width(right) > w-8 {
		right = strings.TrimRight(fit(strings.TrimRight(right, " "), max(w-8, 1)), " ") + " " // a name keeps some room
	}
	room := max(w-width(right)-4, 1) // the cells for the name and the title
	name, title := n.name, ""
	if n.title != "" {
		full := "<" + n.title + ">"
		switch left := room - width(name) - 1; {
		case left >= width(full):
			title = full
		case left >= 5:
			title = strings.TrimRight(fit(full, left), " ")
		}
	}
	name = strings.TrimRight(fit(name, min(room, width(name))), " ")
	used := width(name)
	if title != "" {
		used += 1 + width(title)
	}
	gap := strings.Repeat(" ", max(w-3-used-width(right)-1, 0)+1)
	line := " " + string(g) + " " + name
	if title != "" {
		if n.target {
			line += " " + title
		} else {
			line += " " + st.Dim(title)
		}
	}
	line += gap + right
	if n.target {
		return st.Reverse(line)
	}
	if n.shut {
		return st.Dim(line)
	}
	return line
}

func pad(rows []string, w, h int) []string {
	for len(rows) < h {
		rows = append(rows, strings.Repeat(" ", w))
	}
	return rows
}

func targetLines(m *model.Model, w int, st Styler) []string {
	id := m.Self.Target
	e := m.Entities[id]
	if id != "" && e == nil && m.Group[id] != nil {
		e = m.Group[id].Entity
	}
	if e == nil {
		return pad([]string{fit(" no target (Tab to choose)", w)}, w, targetRows-2)
	}
	g := upper(entityGlyph(e))
	state := fmt.Sprintf("HP %d/%d ", e.GetHp(), e.GetMaxHp())
	if e.GetDead() {
		state = "DEAD "
	}
	sp := m.Self.Pos
	right := metres(dist(e.GetPos().GetX(), e.GetPos().GetY(), sp.GetX(), sp.GetY())) + " " + bearing(e.GetPos().GetX()-sp.GetX(), e.GetPos().GetY()-sp.GetY())
	fight := "not in a fight"
	for _, eid := range m.EncounterIDs() {
		enc := m.Encounters[eid]
		if enc.GetMob() != id {
			continue
		}
		if enc.GetOwner() == m.Self.ID {
			fight = "fighting you"
		} else if o := m.Group[enc.GetOwner()]; o != nil || enc.GetOwner() != "" {
			who := enc.GetOwner()
			if o != nil {
				who = o.Entity.GetName()
			} else if n := m.Entities[who]; n != nil {
				who = n.GetName()
			}
			fight = "in a fight with " + who
		}
	}
	return []string{
		st.Bold(fit(fmt.Sprintf(" %c %s  Lv %d%s", g, e.GetName(), e.GetLevel(), rareTag(e)), w)),
		fit(" "+state+bar(e.GetHp(), e.GetMaxHp(), max(w-width(state)-width(right)-4, 0))+" "+right, w),
		fit(" "+bandWord(e)+fight, w),
	}
}

// bandWord is the mob's difficulty band for the Target panel, in words so that
// it never rests on colour alone: "hard, ".
func bandWord(e *pb.EntityState) string {
	if e.GetCon() == "" || e.GetDead() {
		return ""
	}
	return e.GetCon() + ", "
}

// rareTag marks a roaming rare in the Target panel.
func rareTag(e *pb.EntityState) string {
	if model.IsRare(e) && !e.GetDead() {
		return " RARE"
	}
	return ""
}

// logRow is one wrapped, styled line of the log panel.
type logRow struct {
	text string
	fn   func(string) string
}

// logLines is the log panel's rows, and how many lines are past the last one in
// view while it is scrolled back. Live, it walks the log from the newest entry
// back and stops once h rows are in hand, so only the entries that show are
// wrapped and styled; scrolled back, it wraps the whole log to find its place.
func logLines(m *model.Model, v View, w, h int, st Styler) ([]string, int) {
	pos := v.pos(PanelEvents)
	need := h
	if pos != nil && !pos.Live() {
		need = math.MaxInt
	}
	var blocks [][]logRow // newest entry first
	var ords []int
	have := 0
	for i := len(m.Log) - 1; i >= 0 && have < need; i-- {
		e := &m.Log[i]
		if !v.Verbose && e.Hidden() {
			continue
		}
		rows := logEntryRows(m, e, w, st)
		blocks = append(blocks, rows)
		ords = append(ords, int(e.Ord))
		have += len(rows)
	}
	all := make([]logRow, 0, have)
	seqs := make([]int, 0, have)
	for i := len(blocks) - 1; i >= 0; i-- {
		all = append(all, blocks[i]...)
		for range blocks[i] {
			seqs = append(seqs, ords[i])
		}
	}
	lo, hi, fresh := pos.window(seqs, h)
	all = all[lo:hi]
	rows := make([]string, 0, h)
	for _, s := range all {
		rows = append(rows, s.fn(fit(s.text, w)))
	}
	if len(all) == 0 {
		rows = append(rows, fit(" nothing has happened yet", w))
	}
	return pad(rows, w, h), fresh
}

// logEntryRows is one log entry, wrapped to w cells and styled.
func logEntryRows(m *model.Model, e *model.LogEntry, w int, st Styler) []logRow {
	mark, fn := " ", func(s string) string { return s }
	switch {
	case e.Refused():
		mark, fn = "!", func(s string) string { return st.Red(st.Bold(s)) }
	case e.Kind == "announcement":
		mark, fn = "#", func(s string) string { return st.Chat.Server(s) }
	case (e.Kind == "combat_hit" || e.Kind == "combat_miss") && e.Target == m.Self.ID:
		mark, fn = "<", st.Red
	case e.Kind == "combat_death":
		mark, fn = "x", st.Bold
		if e.Actor != m.Self.ID {
			fn = func(s string) string { return s }
		}
	case e.Kind == "world_event" || e.Kind == "rare_appeared":
		mark, fn = "*", st.Bold
	case e.Kind == "reward":
		mark, fn = "+", st.Green
	case e.Kind == "cast_cancelled" || e.Kind == "cast_interrupted":
		mark, fn = "!", func(s string) string { return st.Red(st.Bold(s)) }
		if e.Actor != m.Self.ID {
			mark, fn = " ", st.Dim
		}
	case e.Kind == "level_up":
		mark, fn = "+", func(s string) string { return st.Green(st.Bold(s)) }
	case e.Kind == "quest":
		mark, fn = "+", st.Green
	case e.Kind == "walk_stopped":
		mark, fn = "!", st.Bold
	case e.Kind == "entity_moved" || e.Kind == "rules_tick" || e.Kind == "resource_changed":
		fn = st.Dim
	}
	prefix := " " + mark + " " + model.Stamp(e.At, e.Approx) + " "
	muted := false // inside the capped part of an experience line (D86)
	var out []logRow
	for i, ln := range wrap(clean(e.Text), max(w-width(prefix), 8)) {
		p := prefix
		if i > 0 {
			p = strings.Repeat(" ", width(prefix)+2)
		}
		lfn := fn
		if e.Capped {
			if at := strings.Index(ln, "(capped"); at >= 0 && !muted {
				muted = true
				head, tail := fn, st.Dim
				lfn = func(s string) string { return mutedFrom(s, len([]rune(p))+len([]rune(ln[:at])), head, tail) }
			} else if muted {
				lfn = st.Dim
			}
		}
		out = append(out, logRow{p + ln, lfn})
	}
	return out
}

// wrap breaks s into lines of at most n cells at spaces; continuation lines
// are indented by the caller. A word longer than a line is cut.
func wrap(s string, n int) []string {
	var out []string
	cur := ""
	for _, word := range strings.Fields(s) {
		for width(word) > n {
			if cur != "" {
				out, cur = append(out, cur), ""
			}
			r := []rune(word)
			out, word = append(out, string(r[:n])), string(r[n:])
		}
		switch {
		case cur == "":
			cur = word
		case width(cur)+1+width(word) <= n-2*boolInt(len(out) > 0):
			cur += " " + word
		default:
			out, cur = append(out, cur), word
		}
	}
	if cur != "" || len(out) == 0 {
		out = append(out, cur)
	}
	return out
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// targetName is what the target is called, or "none".
func targetName(m *model.Model) string {
	id := m.Self.Target
	switch {
	case id == "":
		return "none"
	case m.Entities[id] != nil:
		return m.Entities[id].GetName()
	case m.Group[id] != nil:
		return m.Group[id].Entity.GetName()
	}
	return id
}

// mutedFrom draws s in head up to the cell at, and the rest in tail: the capped
// part of an experience line is information, so it is dim, not green.
func mutedFrom(s string, at int, head, tail func(string) string) string {
	r := []rune(s)
	if at >= len(r) {
		return head(s)
	}
	return head(string(r[:at])) + tail(string(r[at:]))
}
