package screen

import (
	"fmt"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
)

// Item links on the screen (D206): a linked item is its name in brackets in the
// item's rarity colour, in the chat (chatpanel.go) and on the talk screen, and a
// key shows its details. Everything shown is what the server said.

// ItemUI is the item screen, nil when closed: the links to choose among when
// several were in view, or the item chosen (ID) and its details.
type ItemUI struct {
	Pick []model.ChatPart
	Sel  int
	ID   string
}

// itemWord is a link as it is written: the name in brackets.
func itemWord(name string) string { return "[" + clean(name) + "]" }

// flowWords wraps words into rows of w cells, each led by indent and padded.
func flowWords(words []styledWord, w int, indent string) []string {
	var rows []string
	var b strings.Builder
	used := 0
	flush := func() {
		b.WriteString(strings.Repeat(" ", max(w-used, 0)))
		rows = append(rows, b.String())
		b.Reset()
		used = 0
	}
	start := func() {
		b.WriteString(indent)
		used = width(indent)
	}
	start()
	led := true // nothing but the indent on the row
	for _, word := range words {
		ww := word.width()
		if !led && used+1+ww > w {
			flush()
			start()
			led = true
		}
		if !led {
			b.WriteString(" ")
			used++
		}
		for _, s := range word {
			r := []rune(s.s)
			if room := w - used; len(r) > room { // a word longer than a row is cut
				r = r[:max(room, 0)]
			}
			b.WriteString(s.fn(string(r)))
			used += len(r)
		}
		led = false
	}
	flush()
	return rows
}

// speechRows is a line an NPC said, quoted and wrapped to w cells, its item
// links in their rarity colours; the line as plain text where it has none.
func speechRows(l string, parts []model.ChatPart, w int, st Styler) []string {
	if len(parts) == 0 {
		return wrapped("“"+l+"”", w, " ")
	}
	var b wordBuilder
	b.text("“", ident)
	for _, p := range parts {
		if p.Kind == model.KindItem {
			b.add(itemWord(p.Text), st.Chat.ItemStyle(p.Tier), "")
			continue
		}
		b.text(p.Text, ident)
	}
	b.text("”", ident)
	b.end()
	return flowWords(b.words, w, " ")
}

// itemLines is the item screen: the choice when several links were in view,
// else the item's card (D78) as the bags show it, with its comparison with what
// the character wears.
func itemLines(m *model.Model, ui *ItemUI, w, h int, st Styler) []string {
	var rows []string
	add := func(l string) { rows = append(rows, fit(l, w)) }
	if len(ui.Pick) > 0 {
		rows = append(rows, st.Bold(fit(" Items linked here: choose one", w)), fit("", w))
		for i, p := range ui.Pick {
			mark := "   "
			if i == ui.Sel {
				mark = " > "
			}
			num, name := fmt.Sprintf("%s%d  ", mark, i+1), itemWord(p.Text)
			tail := fit(" "+tierWord(p.Tier), max(w-width(num)-width(name), 0))
			if i == ui.Sel {
				rows = append(rows, st.Reverse(fit(num+name+tail, w)))
			} else {
				rows = append(rows, num+st.Chat.ItemStyle(p.Tier)(name)+tail)
			}
		}
		for len(rows) < h-1 {
			add("")
		}
		rows = append(rows, help(" {1}-{9} or {Up} / {Down} choose · {Enter} shows it · {Esc} closes", w, st))
		return pad(rows[:min(len(rows), h)], w, h)
	}
	it := m.Item(ui.ID)
	if it == nil {
		add(" …")
	} else {
		head := " " + st.Chat.ItemStyle(it.GetTier())(itemWord(it.GetName()))
		rows = append(rows, head+strings.Repeat(" ", max(w-width(" "+itemWord(it.GetName())), 0)))
		rows = append(rows, fit("", w))
		rows = append(rows, itemDetail(m, ui.ID, w, sellWords(m, ui.ID))...)
		if l := restrictWords(it.GetClasses(), it.GetMinLevel()); l != "" {
			add(" " + l)
		}
	}
	for len(rows) < h-1 {
		add("")
	}
	rows = append(rows, help(" {Esc} closes", w, st))
	return pad(rows[:min(len(rows), h)], w, h)
}

// restrictWords is who may use an item: its classes and the level it needs.
func restrictWords(classes []string, minLevel int64) string {
	var parts []string
	if len(classes) > 0 {
		parts = append(parts, "for "+strings.Join(classes, ", "))
	}
	if minLevel > 1 {
		parts = append(parts, fmt.Sprintf("needs level %d", minLevel))
	}
	return strings.Join(parts, "; ")
}
