package screen

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/GeneStevens/seq-tui/model"
)

// Chat for a small screen (D101): the tabbed panel at the bottom with unread
// badges and the ticker, the speech bubbles over the map, the toast on the status
// line and the pull-up drawer. The screen reads the model and what the program
// keeps in the View, and decides nothing: which line is new, and for how long a
// bubble or a toast stays, are the program's.

// The tabs of the bottom panel.
const (
	PanelEvents = iota
	PanelChat
	PanelParty
	PanelTabs
)

var panelNames = [PanelTabs]string{"Events", "Chat", "Party"}

// PanelName is the name of a tab of the bottom panel.
func PanelName(tab int) string { return panelNames[min(max(tab, 0), PanelTabs-1)] }

// Marks are how far the player has read each tab: the log lines the short view
// showed (Events) and the chat lines (Chat, and Party for the party's) up to those
// numbers. What is past a mark is unread. Render moves them for the tab it draws.
type Marks struct{ Events, Chat, Party int }

// Unread counts what is past the marks. A nil Marks has read nothing.
func (mk *Marks) Unread(m *model.Model) (events, chat, party int) {
	var z Marks
	if mk == nil {
		mk = &z
	}
	events = max(m.LogN-mk.Events, 0)
	for i := len(m.Social.Chat) - 1; i >= 0; i-- {
		l := m.Social.Chat[i]
		if l.Seq <= min(mk.Chat, mk.Party) {
			break
		}
		if l.Mine {
			continue
		}
		if l.Seq > mk.Chat {
			chat++
		}
		if l.Channel == "party" && l.Seq > mk.Party {
			party++
		}
	}
	return events, chat, party
}

// Viewed marks tab as read: everything the model holds now. The Chat tab shows the
// party's lines too, so it clears both.
func (mk *Marks) Viewed(m *model.Model, tab int) {
	if mk == nil {
		return
	}
	switch tab {
	case PanelEvents:
		mk.Events = m.LogN
	case PanelChat:
		mk.Chat, mk.Party = m.Social.ChatN, m.Social.ChatN
	case PanelParty:
		mk.Party = m.Social.ChatN
	}
}

// Mail counts the tells and the mentions past the Chat mark: the lines that raise
// a toast, and stay as a count on the status line until the Chat tab is viewed.
func (mk *Marks) Mail(m *model.Model) int {
	from := 0
	if mk != nil {
		from = mk.Chat
	}
	n := 0
	for i := len(m.Social.Chat) - 1; i >= 0 && m.Social.Chat[i].Seq > from; i-- {
		if IsMail(m.Social.Chat[i]) {
			n++
		}
	}
	return n
}

// IsMail says a line, said by someone else, is a tell or names the character.
func IsMail(l model.ChatLine) bool {
	return !l.Mine && (l.Channel == "tell" || l.Mention)
}

// panelTab is the tab the bottom panel shows: the one the player chose, but
// Events while something fights the character (D101 §5; D75, danger is never
// hidden), when chat shrinks to its badge and the ticker.
func panelTab(m *model.Model, v View) int {
	if m.UnderAttack() {
		return PanelEvents
	}
	return min(max(v.Panel, 0), PanelTabs-1)
}

// badge is an unread count beside a tab's name: "•3", "" for none.
func badge(n int) string {
	switch {
	case n <= 0:
		return ""
	case n > 99:
		return "•99+"
	}
	return fmt.Sprintf("•%d", n)
}

// The channel filters of the drawer.
const (
	FilterAll = iota
	FilterSay
	FilterParty
	FilterTell
	Filters
)

var filterNames = [Filters]string{"all", "say", "party", "tell"}

// FilterName is the name of a drawer filter.
func FilterName(f int) string { return filterNames[min(max(f, 0), Filters-1)] }

// FilterMatch says the filter shows the line.
func FilterMatch(f int, l model.ChatLine) bool {
	switch f {
	case FilterSay:
		return l.Channel == "say"
	case FilterParty:
		return l.Channel == "party"
	case FilterTell:
		return l.Channel == "tell"
	}
	return true
}

// DrawerUI is the chat drawer, nil when closed. Scroll is how many rows back from
// the newest the chat is scrolled.
type DrawerUI struct{ Filter, Scroll int }

// Toast is a line on the status line for a few seconds: a tell or a mention.
type Toast struct{ Body, Hint string }

// ToastFor is the toast a line raises: "✉ Mira: group for the Warren?" with the
// key that answers it, "" when the line raises none.
func ToastFor(l model.ChatLine) *Toast {
	if !IsMail(l) {
		return nil
	}
	hint := "C chat"
	if l.Channel == "tell" {
		hint = "r reply"
	}
	return &Toast{Body: "✉ " + l.From + ": " + clean(l.Text), Hint: hint}
}

// toastRow is the toast drawn on the status line's row: the body cut to leave the
// hint whole.
func toastRow(t *Toast, w int, st Styler) string {
	hint := "  " + t.Hint
	body := fit(" "+t.Body, max(w-width(hint)-1, 1))
	return st.Reverse(st.Chat.Tell(st.Bold(fit(body+hint, w))))
}

// Bubble is a line a character said, shown over the map until Until.
type Bubble struct {
	Name, Text string
	Until      time.Time
}

// Bubble measures (D101 §1).
const (
	BubbleLasts = 4 * time.Second
	BubbleChars = 24
	bubbleMax   = 4 // at most this many at once, the newest
)

// BubbleText is what a bubble says of a line: its first 24 characters, then …
func BubbleText(text string) string {
	r := []rune(strings.Join(strings.Fields(clean(text)), " "))
	if len(r) > BubbleChars {
		return string(r[:BubbleChars]) + "…"
	}
	return string(r)
}

// LiveBubbles are the bubbles still up at now, oldest first: a speaker has one
// (its newest), and no more than a few show at once.
func LiveBubbles(bs []Bubble, now time.Time) []Bubble {
	var out []Bubble
	for i := len(bs) - 1; i >= 0 && len(out) < bubbleMax; i-- {
		b := bs[i]
		if !b.Until.After(now) {
			continue
		}
		dup := false
		for _, o := range out {
			dup = dup || model.Key(o.Name) == model.Key(b.Name)
		}
		if !dup {
			out = append(out, b)
		}
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// paintBubbles draws each bubble by its speaker, above first, then beside, in
// open ground only (never over a glyph) and clear of the labels. A speaker out of
// the window, or with no room around it, gets none: the log has the line.
func paintBubbles(g *grid, m *model.Model, bs []Bubble, used map[[2]int64]bool) {
	if used == nil {
		used = map[[2]int64]bool{}
	}
	for i := len(bs) - 1; i >= 0; i-- { // the newest first: it gets the best place
		b := bs[i]
		pos := m.PosOfName(b.Name)
		if pos == nil {
			continue
		}
		c, r := g.cellOf(pos)
		text := "“" + BubbleText(b.Text) + "”"
		w := int64(width(text))
		for _, at := range bubbleSpots(c, r, w) {
			if g.roomFor(at[0], at[1], int(w)) && g.labelRoom(used, at[0], at[1], int(w)) {
				g.text(at[0], at[1], text, kBubble)
				for k := int64(0); k < w; k++ {
					used[[2]int64{at[0] + k, at[1]}] = true
				}
				break
			}
		}
	}
}

// bubbleSpots are the places a bubble of w cells may take by a speaker at (c, r),
// best first: above, above to either side, beside it, below, two rows off.
func bubbleSpots(c, r, w int64) [][2]int64 {
	return [][2]int64{{c - w/2, r - 1}, {c, r - 1}, {c - w + 1, r - 1}, {c + 1, r}, {c - w, r}, {c - w/2, r + 1}, {c, r + 1}, {c - w/2, r - 2}, {c - w/2, r + 2}}
}

// seg is a run of text and the style it is drawn in.
type seg struct {
	s  string
	fn func(string) string
}

func drawSegs(segs []seg) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(s.fn(s.s))
	}
	return b.String()
}

// styledWord is a word made of runs of different styles.
type styledWord []seg

func (w styledWord) width() int {
	n := 0
	for _, s := range w {
		n += width(s.s)
	}
	return n
}

// channelStyle is the style of a channel's lines.
func channelStyle(st Styler, channel string) func(string) string {
	switch channel {
	case "party":
		return st.Chat.Party
	case "tell":
		return st.Chat.Tell
	}
	return st.Chat.Say
}

// nameStyle is the style of a name by what it is (D101 §8).
func nameStyle(m *model.Model, st Styler, name string) func(string) string {
	switch m.KindOfName(name) {
	case model.NameSelf:
		return st.Chat.Me
	case model.NamePlayer:
		return st.Chat.Player
	case model.NameNPC:
		return st.Chat.NPC
	}
	return st.Chat.Neutral
}

// chatHead is the tag of a line, its speaker's name (or the one told) and what
// follows the name: "[tell] to ", "Mira", ":". The tag says the channel in words,
// so colour only repeats it.
func chatHead(l model.ChatLine) (tag, name, colon string) {
	switch l.Channel {
	case "party":
		return "[party] ", l.From, ":"
	case "tell":
		if l.Mine {
			return "[tell] to ", l.To, ":"
		}
		return "[tell] ", l.From, ":"
	}
	return "[say] ", l.From, ":"
}

// chatTag is the channel in words, so colour only repeats it.
func chatTag(l model.ChatLine) string {
	tag, name, colon := chatHead(l)
	return tag + name + colon
}

// chatWords are the words of a line of chat, styled: the tag in the channel's
// colour, the name by what it is, the character's own name in its own colour
// wherever the text has it.
func chatWords(m *model.Model, l model.ChatLine, st Styler) []styledWord {
	base := channelStyle(st, l.Channel)
	tag, name, colon := chatHead(l)
	var words []styledWord
	for _, t := range strings.Fields(tag) {
		words = append(words, styledWord{{t, base}})
	}
	words = append(words, styledWord{{clean(name), nameStyle(m, st, name)}, {colon, base}})
	for _, t := range strings.Fields(clean(l.Text)) {
		words = append(words, textWord(t, m.Self.Name, base, st))
	}
	return words
}

// textWord is a word of the text; its letters in the character's own colour if
// they are the character's name.
func textWord(t, self string, base func(string) string, st Styler) styledWord {
	r := []rune(t)
	lo, hi := 0, len(r)
	word := func(c rune) bool { return unicode.IsLetter(c) || unicode.IsDigit(c) }
	for lo < hi && !word(r[lo]) {
		lo++
	}
	for hi > lo && !word(r[hi-1]) {
		hi--
	}
	if self == "" || lo == hi || model.Key(string(r[lo:hi])) != model.Key(self) {
		return styledWord{{t, base}}
	}
	var w styledWord
	for _, p := range []seg{{string(r[:lo]), base}, {string(r[lo:hi]), st.Chat.Me}, {string(r[hi:]), base}} {
		if p.s != "" {
			w = append(w, p)
		}
	}
	return w
}

// chatLineRows is one line of chat as rows w cells wide, each with its styles. A
// line that names the character starts its first row with », in bold, so it is
// found without colour.
func chatLineRows(m *model.Model, l model.ChatLine, w int, st Styler) []string {
	avail := max(w-1, 8)
	var rows []string
	var cur []seg
	curW := 0
	flush := func() {
		lead := " "
		fn := func(s string) string { return s }
		if len(rows) == 0 && l.Mention {
			lead, fn = "»", st.Chat.Mention
		} else if len(rows) > 0 {
			lead = "    "
		}
		segs := append([]seg{{lead, fn}}, cur...)
		if l.Mention {
			for i := range segs {
				inner := segs[i].fn
				segs[i].fn = func(s string) string { return st.Chat.Mention(inner(s)) }
			}
		}
		used := width(lead) + curW
		segs = append(segs, seg{strings.Repeat(" ", max(w-used, 0)), func(s string) string { return s }})
		rows = append(rows, drawSegs(segs))
		cur, curW = nil, 0
	}
	room := func() int {
		if len(rows) == 0 {
			return avail
		}
		return avail - 3
	}
	for _, word := range chatWords(m, l, st) {
		ww := word.width()
		for ww > room() { // a word longer than a row is cut
			if curW > 0 {
				flush()
				continue
			}
			var head, tail styledWord
			left := room()
			for _, s := range word {
				rs := []rune(s.s)
				take := min(left, len(rs))
				if take > 0 {
					head = append(head, seg{string(rs[:take]), s.fn})
				}
				if take < len(rs) {
					tail = append(tail, seg{string(rs[take:]), s.fn})
				}
				left -= take
			}
			cur, curW = append(cur, head...), head.width()
			flush()
			word, ww = tail, tail.width()
		}
		if curW > 0 && curW+1+ww > room() {
			flush()
		}
		if curW > 0 {
			cur, curW = append(cur, seg{" ", func(s string) string { return s }}), curW+1
		}
		cur, curW = append(cur, word...), curW+ww
	}
	if curW > 0 || len(rows) == 0 {
		flush()
	}
	return rows
}

// chatRows wraps the chat the filter keeps into rows of w cells, oldest first. It
// walks from the newest line back and stops once need rows are in hand, so only
// the lines that show are wrapped and styled.
func chatRows(m *model.Model, w, need int, keep func(model.ChatLine) bool, st Styler) []string {
	var blocks [][]string // newest line first
	have := 0
	for i := len(m.Social.Chat) - 1; i >= 0 && have < need; i-- {
		l := m.Social.Chat[i]
		if !keep(l) {
			continue
		}
		rows := chatLineRows(m, l, w, st)
		blocks = append(blocks, rows)
		have += len(rows)
	}
	out := make([]string, 0, have)
	for i := len(blocks) - 1; i >= 0; i-- {
		out = append(out, blocks[i]...)
	}
	return out
}

// chatRowsOf is all the chat as rows, for the Social screen.
func chatRowsOf(m *model.Model, w int, st Styler) []string {
	return chatRows(m, w, 1<<30, func(model.ChatLine) bool { return true }, st)
}

func partyOnly(l model.ChatLine) bool { return l.Channel == "party" }

// chatTabLines are the rows of the Chat and Party tabs: the newest lines that fit.
func chatTabLines(m *model.Model, w, h int, keep func(model.ChatLine) bool, empty string, st Styler) []string {
	rows := chatRows(m, w, h, keep, st)
	if len(rows) > h {
		rows = rows[len(rows)-h:]
	}
	if len(rows) == 0 {
		rows = append(rows, st.Dim(fit(" "+empty, w)))
	}
	return pad(rows, w, h)
}

// newestChat is the newest line of chat, for the ticker.
func newestChat(m *model.Model) (model.ChatLine, bool) {
	if n := len(m.Social.Chat); n > 0 {
		return m.Social.Chat[n-1], true
	}
	return model.ChatLine{}, false
}

// panelTop is the top border of the bottom panel: the tabs with their badges, the
// active one in brackets, and, while chat is not what the panel shows, the newest
// line of chat running along the rest of the border. It notes where the tabs are
// for a click.
func panelTop(m *model.Model, v View, w, y int, tab int, st Styler) string {
	border := st.Dim
	ev, ch, pa := v.Read.Unread(m)
	counts := [PanelTabs]int{ev, ch, pa}
	segs := []seg{{"┌─ ", border}}
	x := 3
	for i := 0; i < PanelTabs; i++ {
		name := panelNames[i]
		if i == tab {
			label := "[" + name
			if i == PanelEvents {
				label += ": " + logMode(v)
			}
			label += "]"
			segs = append(segs, seg{label, st.Accent})
			v.Hits.notePanelTab(x, y, width(label), i)
			x += width(label)
		} else {
			segs = append(segs, seg{name, st.Bold})
			b := badge(counts[i])
			n := width(name) + width(b)
			if b != "" {
				segs = append(segs, seg{" " + b, st.Bold})
				n++
			}
			v.Hits.notePanelTab(x, y, n, i)
			x += n
		}
		if i < PanelTabs-1 {
			segs = append(segs, seg{"  ", border})
			x += 2
		}
	}
	segs = append(segs, seg{" ", border})
	x++
	// The ticker takes the right end of the border, if there is room for a line worth reading.
	if l, ok := newestChat(m); ok && tab != PanelChat {
		room := w - x - 5
		if room >= 14 {
			t := fit(chatTag(l)+" "+l.Text, room)
			t = strings.TrimRight(t, " ")
			segs = append(segs, seg{strings.Repeat("─", max(w-x-width(t)-4, 1)) + " ", border}, seg{t, channelStyle(st, l.Channel)}, seg{" ─┐", border})
			return drawSegs(segs)
		}
	}
	segs = append(segs, seg{strings.Repeat("─", max(w-x-1, 0)) + "┐", border})
	return drawSegs(segs)
}

// bottomPanel is the tabbed panel at the bottom of the screen: h rows of w cells.
// y is the row it starts on. Drawing it is what reads its tab, so Render moves
// the marks.
func bottomPanel(m *model.Model, v View, w, h, y int, st Styler) []string {
	tab := panelTab(m, v)
	v.Read.Viewed(m, tab)
	var body []string
	switch tab {
	case PanelChat:
		body = chatTabLines(m, w-2, h-2, func(model.ChatLine) bool { return true }, "nobody has said anything: / types, C lifts the chat over the map", st)
	case PanelParty:
		body = chatTabLines(m, w-2, h-2, partyOnly, "no party chat: /p types to the party", st)
	default:
		body = logLines(m, v, w-2, h-2, st)
	}
	side := st.Dim("│")
	rows := []string{panelTop(m, v, w, y, tab, st)}
	for i := 0; i < h-2; i++ {
		line := strings.Repeat(" ", w-2)
		if i < len(body) && body[i] != "" {
			line = body[i]
		}
		rows = append(rows, side+line+side)
	}
	return append(rows, st.Dim("└"+strings.Repeat("─", w-2)+"┘"))
}

// drawerLines are the rows inside the chat drawer: the filters, a warning while
// something attacks, the chat from the newest back by the scroll, and a hint.
func drawerLines(m *model.Model, v View, d *DrawerUI, w, h int, st Styler) []string {
	var tabs []string
	for i, n := range filterNames {
		if i == d.Filter {
			n = "[" + n + "]"
		}
		tabs = append(tabs, n)
	}
	rows := []string{fit(" "+strings.Join(tabs, " ")+"   Tab filters", w)}
	if m.UnderAttack() {
		rows = append(rows, st.Red(st.Bold(fit(" ⚠ UNDER ATTACK: F fights back. This closes when a new attack starts", w))))
	}
	body := h - len(rows) - 1
	keep := func(l model.ChatLine) bool { return FilterMatch(d.Filter, l) }
	all := chatRows(m, w, body+d.Scroll, keep, st)
	d.Scroll = max(min(d.Scroll, len(all)-body), 0)
	end := len(all) - d.Scroll
	start := max(end-body, 0)
	shown := all[start:end]
	if len(shown) == 0 {
		shown = []string{st.Dim(fit(" nothing said here", w))}
	}
	rows = append(rows, pad(shown, w, body)...)
	hint := " Up / Down, PgUp / PgDn, wheel scroll · End: newest · / types · Esc closes"
	if d.Scroll > 0 {
		hint = fmt.Sprintf(" scrolled back %d rows: End goes to the newest", d.Scroll)
	}
	return append(rows, st.Dim(fit(hint, w)))
}

// drawerHeight is how many rows the drawer takes of a map panel mh rows high:
// about half the screen, leaving the map its top.
func drawerHeight(mh, h int) int { return max(min(h/2, mh-3), 6) }

// withDrawer puts the chat drawer over the lower part of the map panel mp, which
// is w cells wide. The panels beside the map stay, with the vitals and the target:
// the drawer hides the map and nothing that is danger. top is the screen row mp
// starts on.
func withDrawer(mp []string, m *model.Model, v View, w, h, top int, st Styler) []string {
	dh := drawerHeight(len(mp), h)
	if dh > len(mp) {
		return mp
	}
	d := v.Drawer
	v.Read.Viewed(m, PanelChat) // the drawer shows the chat: it is read
	box := boxA("Chat: "+FilterName(d.Filter), w, dh, drawerLines(m, v, d, w-2, dh-2, st), st, m.UnderAttack())
	if v.Hits != nil {
		v.Hits.Drawer = Rect{0, top + len(mp) - dh, w, dh}
	}
	return append(append([]string(nil), mp[:len(mp)-dh]...), box...)
}

// A Line's channel: what plain text typed in the command line is sent as.
const (
	ChanSay   = "say"
	ChanParty = "party"
	ChanTell  = "tell"
)

// NextChannel is the channel after cur for Tab on the command line (D101 §6): say,
// party, the character told last, and round again. A channel that cannot be used
// (no party, nobody told) is passed over; say is always there.
func NextChannel(cur string, inParty bool, peer string) string {
	order := []string{ChanSay, ChanParty, ChanTell}
	ok := map[string]bool{ChanSay: true, ChanParty: inParty, ChanTell: peer != ""}
	at := 0
	for i, c := range order {
		if c == cur {
			at = i
		}
	}
	for i := 1; i <= len(order); i++ {
		if c := order[(at+i)%len(order)]; ok[c] {
			return c
		}
	}
	return ChanSay
}

// Prompt is what the command line says before the text: the channel it will send
// on, or [cmd] once the text is a command.
func (l *Line) Prompt() string {
	switch {
	case strings.HasPrefix(l.Text, "/"):
		return "[cmd] > "
	case l.Channel == ChanParty:
		return "[party] > "
	case l.Channel == ChanTell && l.Target != "":
		return "[tell " + clean(l.Target) + "] > "
	}
	return "[say] > "
}

// notePanelTab notes where a tab of the bottom panel was drawn, for a click.
func (h *Hits) notePanelTab(x, y, w, tab int) {
	if h != nil {
		h.Panel = append(h.Panel, Tab{Rect{x, y, w, 1}, tab})
	}
}
