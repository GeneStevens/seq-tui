package screen

import (
	"fmt"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Playing together (M9, D58): the chat panel, the Party panel, the invitation
// prompt, the command line, and the Social screen with the chat scrollback, the
// who list and the friends list. The screen shows what the server said and
// decides nothing.

// Line is the command line, open while the player types a chat line or a
// command; nil when closed.
type Line struct{ Text string }

// The tabs of the Social screen.
const (
	TabChat = iota
	TabWho
	TabFriends
	tabs
)

var tabNames = [tabs]string{"Chat", "Who", "Friends"}

// SocialUI is the Social screen, nil when closed. Scroll is how many lines the
// chat is scrolled back; Sel the row chosen on the who and friends lists. Who is
// the last answer to a who request, and WhoFilter says what was asked.
type SocialUI struct {
	Tab, Scroll, Sel int
	Who              []*pb.WhoEntry
	WhoAsked         bool
	WhoFilter        string
}

// NextTab moves to the next tab.
func (s *SocialUI) NextTab() { s.Tab, s.Sel = (s.Tab+1)%tabs, 0 }

// chatShown says the chat panel is on the main screen: once anything has been
// said, and while a line is being typed.
func chatShown(m *model.Model, v View) bool {
	return len(m.Social.Chat) > 0 || v.Line != nil
}

func channelPaint(st Styler, channel string) func(string) string {
	switch channel {
	case "party":
		return st.Cyan
	case "tell":
		return st.Yellow
	}
	return func(s string) string { return s }
}

// chatTag is the channel in words, so colour only repeats it.
func chatTag(l model.ChatLine) string {
	switch l.Channel {
	case "party":
		return "[party] " + l.From + ":"
	case "tell":
		if l.Mine {
			return "[tell] to " + l.To + ":"
		}
		return "[tell] " + l.From + ":"
	}
	return "[say] " + l.From + ":"
}

// chatRowsOf wraps the chat into rows of w cells, oldest first, each with the
// style of its channel.
func chatRowsOf(m *model.Model, w int, st Styler) []string {
	var rows []string
	for _, l := range m.Social.Chat {
		text := chatTag(l) + " " + clean(l.Text)
		for i, ln := range wrap(text, max(w-1, 8)) {
			p := " "
			if i > 0 {
				p = "   "
			}
			rows = append(rows, channelPaint(st, l.Channel)(fit(p+ln, w)))
		}
	}
	return rows
}

// chatLines are the rows of the Chat panel: the newest lines that fit.
func chatLines(m *model.Model, w, h int, st Styler) []string {
	rows := chatRowsOf(m, w, st)
	if len(rows) > h {
		rows = rows[len(rows)-h:]
	}
	if len(rows) == 0 {
		rows = append(rows, st.Dim(fit(" nobody has said anything", w)))
	}
	return pad(rows, w, h)
}

// partyPanel is the Party panel: the party's other members with their health,
// mana, level and zone. It is nil outside a party.
func partyPanel(m *model.Model, iw int, st Styler) *panel {
	slots := slotRows(m, iw, st)
	if !m.InParty() && len(slots) == 0 {
		return nil
	}
	rows := slots
	for _, mt := range m.Mates() {
		if mt.Me {
			continue
		}
		mark := " "
		if mt.Leader {
			mark = "*"
		}
		head := fmt.Sprintf("%s%s", mark, mt.Name) + statusTag(mt.Status)
		if mt.Level > 0 {
			head += fmt.Sprintf("  Lv %d", mt.Level)
		}
		if !mt.Online {
			rows = append(rows, st.Dim(fit(" "+head+"  away", iw)))
			continue
		}
		rows = append(rows, fit(" "+head, iw))
		if mt.Zone != "" && (iw >= 30 || mt.Dead) {
			rows[len(rows)-1] = fit(" "+head+"  "+mt.Zone, iw)
		}
		if mt.Dead {
			rows = append(rows, st.Red(fit("   DEAD", iw)))
			continue
		}
		hp := fmt.Sprintf("   HP %d/%d ", mt.HP, mt.MaxHP)
		line := hp + bar(mt.HP, mt.MaxHP, max(min(iw-width(hp)-8, 8), 0))
		if mt.MaxMana > 0 {
			line += fmt.Sprintf(" MP %d", mt.Mana)
		}
		row := fit(line, iw)
		if mt.MaxHP > 0 && mt.HP*4 <= mt.MaxHP {
			row = st.Red(row)
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		rows = append(rows, st.Dim(fit(" just you so far", iw)))
	}
	title := "Party, led by " + m.Social.Party.GetLeader()
	switch {
	case !m.InParty():
		title = "Party slots"
	case m.IsLeader():
		title = "Party, you lead"
	}
	return &panel{title, rows}
}

// inviteBanner is the one-row prompt for a party invitation waiting for an
// answer, "" when there is none.
func inviteBanner(m *model.Model, w int, st Styler) string {
	inv := m.Social.Invite
	if inv == nil {
		return ""
	}
	return st.Reverse(st.Bold(fit(fmt.Sprintf(" PARTY INVITE  %s invites you to a party.  Y accept  N decline", inv.From), w)))
}

// bannerRows is the most rows a server message takes at the top of the screen.
const bannerRows = 3

// serverBanner is a server message as banner rows, led by tag: wrapped to w and
// cut at bannerRows (the log keeps an announcement whole), in bold reverse so it
// cannot be missed.
func serverBanner(tag, text string, w int, st Styler) []string {
	lines := wrap(clean(tag+"  "+text), max(w-2, 8))
	if len(lines) > bannerRows {
		lines = lines[:bannerRows]
		lines[bannerRows-1] = strings.TrimRight(lines[bannerRows-1], " ") + "…"
	}
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, st.Reverse(st.Bold(fit(" "+l, w))))
	}
	return out
}

// announceBanner is the server announcement on show (D100), nil when there is none.
func announceBanner(m *model.Model, w int, st Styler) []string {
	if m.Banner == nil {
		return nil
	}
	return serverBanner("SERVER", m.Banner.Text, w, st)
}

// cutBanner is the server's own words for a connection it ended or a login it
// refused, while the client is off the server; nil otherwise.
func cutBanner(v View, w int, st Styler) []string {
	if v.Cut == "" || v.Conn != ConnReconnecting {
		return nil
	}
	return serverBanner("DISCONNECTED", v.Cut+" The client keeps trying to reconnect; Q quits.", w, st)
}

// lineRow is the command line, drawn in the place of the status line.
func lineRow(l *Line, w int, st Styler) string {
	hint := "Enter send  Esc cancel"
	text := " > " + l.Text + "_"
	r := []rune(clean(text))
	if room := w - width(hint) - 2; len(r) > room && room > 4 {
		text = "…" + string(r[len(r)-room+1:])
	}
	if width(text)+1+width(hint) <= w {
		return st.Reverse(fit(fit(text, w-width(hint)-1)+" "+hint, w))
	}
	return st.Reverse(fit(text, w))
}

// socialScreenLines is the Social screen: a row of tabs and the tab's list.
func socialScreenLines(m *model.Model, ui *SocialUI, w, h int, st Styler) []string {
	var tabsRow []string
	for i, n := range tabNames {
		if i == ui.Tab {
			n = "[" + n + "]"
		} else {
			n = " " + n + " "
		}
		tabsRow = append(tabsRow, n)
	}
	rows := []string{fit(" "+strings.Join(tabsRow, " ")+"   Tab switches, Esc closes, / types", w), fit("", w)}
	body := h - len(rows) - 1
	switch ui.Tab {
	case TabChat:
		all := chatRowsOf(m, w, st)
		ui.Scroll = max(min(ui.Scroll, len(all)-body), 0)
		end := len(all) - ui.Scroll
		start := max(end-body, 0)
		rows = append(rows, pad(all[start:end], w, body)...)
		hint := " Up / Down, PgUp / PgDn scroll"
		if ui.Scroll > 0 {
			hint = fmt.Sprintf(" scrolled back %d lines: End goes to the newest", ui.Scroll)
		}
		if len(all) == 0 {
			hint = " nobody has said anything yet: / then a line says it; /p party, /tell name text"
		}
		rows = append(rows, st.Dim(fit(hint, w)))
	case TabWho:
		rows = append(rows, whoRows(ui, w, body, st)...)
		rows = append(rows, st.Dim(fit(" I invite  F add friend  R ask again | /who zone=NAME class=ID 3-7", w)))
	case TabFriends:
		rows = append(rows, friendRows(m, ui, w, body, st)...)
		rows = append(rows, st.Dim(fit(" I invite  D remove  R ask again | /friend add NAME", w)))
	}
	return rows
}

func tableRows(head string, lines []string, sel, w, h int, empty string, st Styler) []string {
	rows := []string{st.Bold(fit(head, w))}
	if len(lines) == 0 {
		rows = append(rows, st.Dim(fit(" "+empty, w)))
		return pad(rows, w, h)
	}
	room := h - 1
	start := 0
	if sel >= room {
		start = sel - room + 1
	}
	for i := start; i < len(lines) && len(rows) < h; i++ {
		mark := "  "
		line := fit(mark+lines[i], w)
		if i == sel {
			line = st.Reverse(fit("> "+lines[i], w))
		}
		rows = append(rows, line)
	}
	return pad(rows, w, h)
}

func whoRows(ui *SocialUI, w, h int, st Styler) []string {
	var lines []string
	for _, e := range ui.Who {
		zone := e.GetZone()
		if zone == "" {
			zone = "(hidden)"
		}
		lines = append(lines, fmt.Sprintf("%-16s %-10s %3d  %s%s", e.GetName(), e.GetClassId(), e.GetLevel(), zone, statusTag(e.GetStatus())))
	}
	ui.Sel = max(min(ui.Sel, len(lines)-1), 0)
	head := fmt.Sprintf("  %-16s %-10s %3s  %s", "Name", "Class", "Lv", "Zone")
	if ui.WhoFilter != "" {
		head += "   (" + ui.WhoFilter + ")"
	}
	empty := "press R to ask who is online"
	if ui.WhoAsked {
		empty = "nobody matches"
	}
	return tableRows(head, lines, ui.Sel, w, h, empty, st)
}

func friendRows(m *model.Model, ui *SocialUI, w, h int, st Styler) []string {
	var lines []string
	for _, f := range m.Social.Friends {
		state := "offline"
		if f.GetOnline() {
			state = "online"
			if f.GetZone() != "" {
				state += ", " + f.GetZone()
			} else {
				state += " (zone hidden)"
			}
		}
		lines = append(lines, fmt.Sprintf("%-16s %s", f.GetName(), state))
	}
	ui.Sel = max(min(ui.Sel, len(lines)-1), 0)
	empty := "no friends yet: /friend add NAME"
	if !m.Social.HasFriends {
		empty = "press R to ask for the list"
	}
	return tableRows(fmt.Sprintf("  %-16s %s", "Name", "Status"), lines, ui.Sel, w, h, empty, st)
}

// socialHelp are the lines of the second page of the key help: playing together.
func socialHelp() []string {
	rows := []string{
		"Command line     /  opens it (the slash is in it). Enter sends, Esc cancels",
		"                 Plain text is said to the characters within 40 m of you",
		"Chat             /say TEXT   /p TEXT: the party   /tell NAME TEXT",
		"                 /r TEXT: answer the last tell   /mute NAME   /unmute NAME",
		"                 /mute alone lists who you muted. 200 characters a line;",
		"                 the server limits how fast you talk",
		"                 say is white, party cyan, tell yellow, and each is tagged",
		"Scrollback       /chat: Up / Down, PgUp / PgDn, End; Tab: Who, Friends",
		"Party            /invite NAME   /accept   /decline   /leave   /party",
		"                 /kick NAME (the leader). An invitation shows at the top:",
		"                 Y accepts it, N declines it",
		"                 The Party panel: * leads, health, mana, level, zone",
		"Instances        any member opens one with I; all walk to its door, E",
		"                 The Party panel lists its slots; ? Group rules says who counts",
		"                 turn-based: the Round panel marks each member's plan",
		"Status           /status away, /status autopilot, /status clear: a word beside",
		"                 your name in who, the party and You. It has no reward effect",
		"Who              /who zone=NAME class=ID 3-7 (every filter optional)",
		"                 Up / Down choose; I invites, F makes a friend, R asks again",
		"Friends          /friends   /friend add NAME   /friend remove NAME",
		"                 D removes, I invites; the log says who logs in or out",
		"Anonymous        /anon on hides your zone from who and friends; /anon off",
		"Logs             Ctrl+L: sessions. Every session is recorded under its id in",
		"                 seq-tui/sessions in your data dir (-log-dir, -no-record)",
	}
	// The topic is the first column of the table, drawn as a key.
	for i, r := range rows {
		if topic := strings.TrimSpace(r[:17]); topic != "" {
			rows[i] = "{" + topic + "}" + strings.Repeat(" ", 17-len(topic)) + r[17:]
		}
		rows[i] = "  " + rows[i]
	}
	return rows
}
