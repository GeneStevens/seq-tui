package app

import (
	"strings"

	pb "github.com/GeneStevens/seq/protocol/seqv1"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
)

// Item links (D206): making them in the chat line, and showing a linked item's
// details. The server checks every id and writes the item's own name, so the
// terminal sends ids and where they stand, never markup.

// maxLinks is how many items a chat line holds (the server's own limit).
const maxLinks = 5

// linkSpans are the links of a line as the server takes them: each item's id and
// the span of text (in code points) that holds its "[Name]". A name the player
// has edited away from the text is no longer a link.
func linkSpans(text string, links []screen.LineLink) []*pb.ItemLink {
	rs := []rune(text)
	var out []*pb.ItemLink
	at := 0
	for _, l := range links {
		needle := []rune("[" + l.Name + "]")
		i := indexRunes(rs[at:], needle)
		if i < 0 {
			continue
		}
		start := at + i
		at = start + len(needle)
		out = append(out, &pb.ItemLink{Item: l.Item, Start: uint32(start), End: uint32(at)})
	}
	return out
}

// indexRunes is the place of sub in rs, -1 for none.
func indexRunes(rs, sub []rune) int {
	for i := 0; i+len(sub) <= len(rs); i++ {
		if string(rs[i:i+len(sub)]) == string(sub) {
			return i
		}
	}
	return -1
}

// linkItem writes the item into the chat line as "[Name]" and notes its id,
// opening the line if it is shut. It says why when it cannot.
func (a *App) linkItem(m *model.Model, inst string) (note string, bad bool) {
	if inst == "" {
		return "no item to link", true
	}
	id, name := model.ItemID(inst), m.ItemName(inst)
	l := a.line
	if l == nil {
		l = &screen.Line{Channel: screen.ChanSay}
	}
	word := "[" + name + "]"
	switch {
	case len(l.Links) >= maxLinks:
		return "a chat line holds at most 5 item links", true
	case len([]rune(l.Text))+len([]rune(word))+1 > maxLine:
		return "the line is too long for another item", true
	}
	if l.Text != "" && !strings.HasSuffix(l.Text, " ") {
		l.Text += " "
	}
	l.Text += word + " "
	l.Links = append(l.Links, screen.LineLink{Item: id, Name: name})
	a.line = l
	return "", false
}

// itemsInView are the items linked in what the screen shows: what the person on
// the talk screen said, else the links drawn in the chat, each once, in the order
// they stand.
func (a *App) itemsInView(m *model.Model) []model.ChatPart {
	var out []model.ChatPart
	add := func(p model.ChatPart) {
		for _, o := range out {
			if o.ID == p.ID {
				return
			}
		}
		out = append(out, p)
	}
	if a.talk != nil {
		if d := m.Said(a.talk.NPC); d != nil {
			for _, p := range model.DialogueLinks(d) {
				add(p)
			}
		}
		return out
	}
	for _, h := range a.mouse.hits.Items { // as drawn: the caller has refreshed them
		add(m.LinkedItem(h.ID))
	}
	return out
}

// showItems opens the item screen on the one item in view and returns its id, to
// be asked of the server; with several it opens the choice and returns "". It
// says why when there is none.
func (a *App) showItems(m *model.Model) (id, note string) {
	items := a.itemsInView(m)
	switch len(items) {
	case 0:
		return "", "no item is linked in what is on the screen (open the Chat tab, [ ], or talk to someone)"
	case 1:
		a.items = &screen.ItemUI{ID: items[0].ID}
		return items[0].ID, ""
	}
	a.items = &screen.ItemUI{Pick: items[:min(len(items), 9)]}
	return "", ""
}

// itemKey handles a key on the item screen: a choice by number or by the cursor,
// and Esc.
func (a *App) itemKey(k string, m *model.Model) Action {
	ui := a.items
	switch {
	case k == "ctrl+c":
		return Action{Kind: Quit}
	case k == "esc" || k == "q":
		a.items = nil
	case len(ui.Pick) == 0:
	case up(k) || down(k):
		ui.Sel = step(ui.Sel, k, len(ui.Pick))
	case enter(k):
		return a.chooseItem(ui.Sel)
	case len(k) == 1 && k[0] >= '1' && k[0] <= '9':
		if i := int(k[0] - '1'); i < len(ui.Pick) {
			return a.chooseItem(i)
		}
	}
	return Action{}
}

// chooseItem shows the details of the i-th of the links to choose from.
func (a *App) chooseItem(i int) Action {
	id := a.items.Pick[i].ID
	*a.items = screen.ItemUI{ID: id}
	return Action{Kind: ShowItems, ID: id}
}
