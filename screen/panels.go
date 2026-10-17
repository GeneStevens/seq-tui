package screen

import (
	"fmt"
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
	you := youLines(m, v, iw, st)
	rows := box("You", rw, len(you)+2, you, st)
	left := th - len(rows)
	inst, round := 0, 0
	if instanceShown(m) {
		inst = max(instanceRows(m), 2) + 2
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
	head := fmt.Sprintf(" %s  Lv %d  XP %d", name, s.Level, s.XP)
	hp := fmt.Sprintf(" HP %d/%d ", s.HP, s.MaxHP)
	hpLine := fit(hp+bar(s.HP, s.MaxHP, max(w-width(hp)-1, 0)), w)
	if s.Dead {
		hpLine = fit(" DEAD  HP 0/"+fmt.Sprint(s.MaxHP), w)
		hpLine = st.Red(st.Bold(hpLine))
	} else if s.MaxHP > 0 && s.HP*4 <= s.MaxHP {
		hpLine = st.Red(hpLine)
	}
	last := fit(" Auto-attack: "+onOff(s.AutoAttack), w)
	if s.Dead {
		// Nothing else can be done but release (D41); say so, and what is
		// happening once it is done.
		last = st.Bold(fit(" You are dead. R: release", w))
		if s.Released {
			last = st.Bold(fit(fmt.Sprintf(" Released: back in %d s", (s.RespawnInMs+999)/1000), w))
		}
	}
	return []string{
		st.Bold(withCrowns(head, s.Crowns, w)),
		hpLine,
		fit(" Target: "+targetName(m), w),
		last,
	}
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
		if model.IsRare(e) && !e.GetDead() {
			n.note = "RARE"
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
		n := mk(glyphDrop, id, m.DropName(d), d.GetPos(), "")
		if !d.GetCanLoot() {
			n.glyph, n.note = glyphLocked, "not lootable"
		}
		out = append(out, n)
	}
	for _, n := range m.KnownNPCs() {
		out = append(out, mk(npcGlyph(n), n.GetId(), n.GetName(), n.GetPos(), n.GetRole()))
	}
	for _, d := range m.Map.GetDoors() {
		out = append(out, mk(glyphDoor, d.GetId(), m.MarkerName(d.GetId()), d.GetPos(), ""))
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
			out = append(out, mk(glyphSpawn, "marker-"+k.GetId(), k.GetName(), k.GetPos(), "event spawn"))
		case "event_goal":
			out = append(out, mk(glyphGoal, "marker-"+k.GetId(), k.GetName(), k.GetPos(), "event goal"))
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
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
		if n.hp != "" {
			right += " " + n.hp
		}
		if n.note != "" {
			right += " " + n.note
		}
		right += " "
		g := n.glyph
		if n.target {
			g = upper(g)
		}
		if len(rows) == h-1 && i < len(list)-1 {
			rows = append(rows, fit(fmt.Sprintf(" +%d more", len(list)-i), w))
			break
		}
		room := w - width(right) - 4
		line := fit(" "+string(g)+" "+fit(n.name, max(room, 1)), w-width(right)-1) + " " + right
		line = fit(line, w)
		if n.target {
			line = st.Reverse(line)
		}
		rows = append(rows, line)
	}
	if len(list) == 0 {
		rows = append(rows, fit(" nothing in view", w))
	}
	return pad(rows, w, h)
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
		fit(" "+fight, w),
	}
}

// rareTag marks a roaming rare in the Target panel.
func rareTag(e *pb.EntityState) string {
	if model.IsRare(e) && !e.GetDead() {
		return " RARE"
	}
	return ""
}

func logLines(m *model.Model, v View, w, h int, st Styler) []string {
	type styled struct {
		text string
		fn   func(string) string
	}
	var all []styled
	for _, e := range m.Log {
		if !v.Verbose && (e.Kind == "entity_moved" || e.Kind == "rules_tick" || e.Kind == "resource_changed") {
			continue
		}
		mark, fn := " ", func(s string) string { return s }
		switch {
		case e.Kind == "denied":
			mark, fn = "!", func(s string) string { return st.Red(st.Bold(s)) }
		case (e.Kind == "combat_hit" || e.Kind == "combat_miss") && e.Target == m.Self.ID:
			mark, fn = "<", st.Red
		case e.Kind == "combat_death":
			mark, fn = "x", st.Yellow
			if e.Actor == m.Self.ID {
				fn = func(s string) string { return st.Yellow(st.Bold(s)) }
			}
		case e.Kind == "world_event" || e.Kind == "rare_appeared":
			mark, fn = "*", func(s string) string { return st.Yellow(st.Bold(s)) }
		case e.Kind == "reward":
			mark, fn = "+", st.Green
		case e.Kind == "cast_cancelled" || e.Kind == "cast_interrupted":
			mark, fn = "!", func(s string) string { return st.Red(st.Bold(s)) }
			if e.Actor != m.Self.ID {
				mark, fn = " ", st.Dim
			}
		case e.Kind == "level_up":
			mark, fn = "+", func(s string) string { return st.Green(st.Bold(s)) }
		case e.Kind == "entity_moved" || e.Kind == "rules_tick" || e.Kind == "resource_changed":
			fn = st.Dim
		}
		prefix := fmt.Sprintf(" %s %5.1fs ", mark, float64(e.GameTimeMs)/1000)
		for i, ln := range wrap(clean(e.Text), max(w-width(prefix), 8)) {
			p := prefix
			if i > 0 {
				p = strings.Repeat(" ", width(prefix)+2)
			}
			all = append(all, styled{p + ln, fn})
		}
	}
	if len(all) > h {
		all = all[len(all)-h:]
	}
	rows := make([]string, 0, h)
	for _, s := range all {
		rows = append(rows, s.fn(fit(s.text, w)))
	}
	if len(all) == 0 {
		rows = append(rows, fit(" nothing has happened yet", w))
	}
	return pad(rows, w, h)
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
