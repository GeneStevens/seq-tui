package screen

import (
	"fmt"

	"github.com/GeneStevens/seq-tui/model"
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
	for _, l := range wrapIndent("  "+q.GetText(), w-2, "  ") {
		add(l)
	}
	if n := q.GetTargetName(); n != "" {
		add(fmt.Sprintf("  (%s: %s)", q.GetTargetKind(), n))
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
