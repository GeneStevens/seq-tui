package screen

import (
	"fmt"

	"github.com/GeneStevens/seq-tui/model"
)

// castCueLines is the cast cue of the Target panel (presentation A3): the move
// the target is casting, on whom, and the time left, anchored to the server's game
// time; under it, for a cast the client knows the reach of, that reach and where
// the cast's target stands to it. When the cast is over, one line says how it
// ended, for a few seconds. Nothing when the target is not casting. The
// countdown reads gameNow, which the app advances with its own clock between the
// server's messages and puts right on each newer game time.
func castCueLines(m *model.Model, v View, id string, w int, st Styler) []string {
	if id == "" {
		return nil
	}
	now := gameNow(m, v)
	if c := m.CastOf(id); c != nil && c.Live(now) {
		move := m.CastName(c.Ability)
		on := m.CastTargetLabel(c.Target)
		var first string
		if c.LandsInRound > 0 {
			first = fmt.Sprintf(" Casting %s on %s: lands in round %d", move, on, c.LandsInRound)
		} else {
			right := " " + secs(c.EndsAtMs-now)
			head := fit(" Casting "+move+" on "+on, max(w-width(right), 0))
			if short := " " + move + " on " + on; width(head) < width(" Casting "+move+" on "+on) {
				head = fit(short, max(w-width(right), 0)) // a narrow panel drops the verb
			}
			if room := w - width(head) - width(right) - 1; room >= 6 {
				head += " " + bar(c.CastMs-c.Left(now), c.CastMs, room)
			}
			first = head + right
		}
		out := []string{st.Bold(fit(first, w))}
		return append(out, reachLines(m, c, w, st)...)
	}
	if e := m.CastEndOf(id); e != nil && e.Shown(now) {
		move := m.CastName(e.Ability)
		line := " " + model.CastEndWords(e, move)
		if width(line) > w {
			line = " " + model.CastEndShort(e, move)
		}
		fn := st.Green // the cast did not land: good news, said in words as well
		if e.Kind == model.EndLanded || e.Kind == model.EndMissed {
			fn = func(s string) string { return st.Red(st.Bold(s)) }
		}
		return []string{fn(fit(line, w))}
	}
	return nil
}

// reachLines say how far the cast reaches and where its target stands to it.
// The reach is the client's own figure (model.ClientReachMm), and is labelled so.
// Being out of reach now is a prediction: a cast whose target is out of reach
// when it ends is cancelled, but the target may step back in before then, so the
// cue stays and the line warns.
func reachLines(m *model.Model, c *model.OtherCast, w int, st Styler) []string {
	reach, d, known := m.CastReach(c)
	if reach == 0 {
		return nil
	}
	who := capFirst(m.CastTargetLabel(c.Target))
	rel := ""
	fn := func(s string) string { return s }
	switch {
	case !known:
		rel = "Distance unknown."
	case d <= reach:
		rel = fmt.Sprintf("%s: %s, in reach.", who, mtr(d))
	default:
		rel = fmt.Sprintf("%s: %s, OUT of reach (predicted).", who, mtr(d))
		fn = func(s string) string { return st.Red(st.Bold(s)) }
	}
	// The reach and the relation are sentences of their own, so that each fits a
	// row of the panel.
	var out []string
	for _, row := range wrap(fmt.Sprintf("Reach %s (client-side figure).", mtr(reach)), max(w-1, 8)) {
		out = append(out, fit(" "+row, w))
	}
	for _, row := range wrap(rel, max(w-1, 8)) {
		out = append(out, fn(fit(" "+row, w)))
	}
	return out
}

// mtr is a distance in metres to a tenth: "2.5 m".
func mtr(mm int64) string { return fmt.Sprintf("%.1f m", float64(mm)/1000) }

func capFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 'a' - 'A'
	}
	return string(r)
}
