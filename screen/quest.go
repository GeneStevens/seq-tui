package screen

import (
	"fmt"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// QuestUI is the quest chain screen (D62): the chain the character is on, and
// the one question that dismisses it for good.
type QuestUI struct {
	Confirm bool // the question "dismiss it for good?" is open
}

func questScreenLines(m *model.Model, ui *QuestUI, w, h int, st Styler) []string {
	var rows []string
	add := func(s string) { rows = append(rows, fit(" "+s, w)) }
	q := m.Quest()
	if q == nil {
		add("You are on no quest chain.")
		add("")
		add("A new character in Thornwick is offered First Steps. It is optional, and a chain")
		add("you dismiss is never offered again.")
		return padRows(append(rows, "", st.Dim(fit(" Esc closes", w))), w, h)
	}
	add(st.Bold(model.QuestProgress(q)))
	if d := q.GetDescription(); d != "" {
		for _, l := range wrapIndent(d, w-2, "") {
			add(l)
		}
	}
	add("")
	add("Now:")
	for _, l := range stepLines(m, q, w-2, "  ", true, st) {
		add(l)
	}
	if model.HasReward(q.GetStepReward()) {
		add("  This step pays " + m.RewardWords(q.GetStepReward()) + ".")
	}
	if model.HasReward(q.GetReward()) {
		add("The whole chain pays " + m.RewardWords(q.GetReward()) + " at its end.")
	}
	add("")
	add("It is optional: nothing waits for it and nothing reminds you.")
	for len(rows) < h-1 {
		rows = append(rows, "")
	}
	if ui.Confirm {
		rows = append(rows, st.Reverse(fit(" Dismiss "+q.GetName()+" for good? It is never offered again.  Y: yes   any other key: no", w)))
	} else {
		rows = append(rows, st.Dim(fit(" X dismisses it for good (after a question)   Esc closes", w)))
	}
	return padRows(rows, w, h)
}

// stepLines are the lines of the current step, at w cells: what to do, how far
// it has come, where it is done (the zone and the door to take when that is
// another zone, and with detail the target) and why a watch went back (quests v2), each starting with ind.
func stepLines(m *model.Model, q *pb.QuestInfo, w int, ind string, detail bool, st Styler) []string {
	var out []string
	for _, l := range wrapIndent(ind+m.QuestStepWords(q), w, ind) {
		out = append(out, l)
	}
	if p := m.QuestProgressLine(q); p != "" {
		out = append(out, st.Bold(ind+p))
	}
	if h := m.WatchHint(q); h != "" {
		for _, l := range wrapIndent(ind+h, w, ind) {
			out = append(out, st.Yellow(l))
		}
	}
	out = append(out, whereLines(m, q, w, ind, detail)...)
	return out
}

// whereLines say where the step is done when that is not where the character
// stands: the zone, and the door to take toward it.
func whereLines(m *model.Model, q *pb.QuestInfo, w int, ind string, detail bool) []string {
	var out []string
	switch {
	case model.QuestAway(q):
		out = append(out, ind+"This step is in another zone: "+q.GetZoneName()+".")
		if name, pos := m.QuestDoor(q); pos != nil {
			sp := m.Self.Pos
			out = append(out, fmt.Sprintf(ind+"Take the %s: %s %s of you.", name, metres(dist(pos.GetX(), pos.GetY(), sp.GetX(), sp.GetY())), bearing(pos.GetX()-sp.GetX(), pos.GetY()-sp.GetY())))
		} else {
			out = append(out, ind+"Find the way there from here; the map marks doors.")
		}
	case detail && q.GetTargetName() != "":
		out = append(out, fmt.Sprintf(ind+"(%s: %s)", q.GetTargetKind(), q.GetTargetName()))
	}
	var fit []string
	for _, l := range out {
		fit = append(fit, wrapIndent(l, w, ind+"  ")...)
	}
	return fit
}
