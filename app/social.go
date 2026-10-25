package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Playing together (M9, D58): the command line, the Social screen, the answer to
// a party invitation. Nothing here decides anything: every request goes to the
// server, which answers or refuses with its reason, and the status line says it.

// inviteLasts is how long an invitation waits for an answer on the screen; the
// server's own lasts a minute, and refuses a late accept with its reason.
const inviteLasts = time.Minute

// maxLine is the longest line the command line takes: a chat line is 200
// characters and a command a little more.
const maxLine = 240

// lineKey handles a key press while the command line is open.
func (a *App) lineKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	l := a.line
	switch k := msg.String(); k {
	case "esc", "ctrl+c":
		a.line = nil
	case "f1":
		a.view.Help, a.view.HelpPage, a.view.HelpScroll = true, 0, 0 // ? is a letter here
	case "enter":
		text := lineWire(l)
		a.line = nil
		if text == "" || text == "/" {
			return a, nil
		}
		a.view.Notice, a.view.NoticeBad = "", false
		a.inflight++
		return a, a.send(Action{Kind: SocialDo, ID: "line", Note: text})
	case "backspace":
		if r := []rune(l.Text); len(r) > 0 {
			l.Text = string(r[:len(r)-1])
		}
	case "ctrl+u":
		l.Text = ""
	case "tab":
		a.lineTab()
	default:
		if t := msg.Key().Text; t != "" && len([]rune(l.Text)) < maxLine {
			l.Text += t
		}
	}
	return a, nil
}

// socialKey handles a key on the Social screen.
func (a *App) socialKey(k string, m *model.Model) Action {
	ui := a.soc
	rows := 0
	switch ui.Tab {
	case screen.TabWho:
		rows = len(ui.Who)
	case screen.TabFriends:
		rows = len(m.Social.Friends)
	}
	selected := func() string {
		switch {
		case ui.Tab == screen.TabWho && ui.Sel < len(ui.Who):
			return ui.Who[ui.Sel].GetName()
		case ui.Tab == screen.TabFriends && ui.Sel < len(m.Social.Friends):
			return m.Social.Friends[ui.Sel].GetName()
		}
		return ""
	}
	switch k {
	case "esc", "q":
		a.soc = nil
	case "ctrl+c":
		return Action{Kind: Quit}
	case "tab":
		ui.NextTab()
	case "/":
		return Action{Kind: OpenLine}
	case "up", "k":
		if ui.Tab == screen.TabChat {
			ui.Scroll++
		} else {
			ui.Sel = max(ui.Sel-1, 0)
		}
	case "down", "j":
		if ui.Tab == screen.TabChat {
			ui.Scroll = max(ui.Scroll-1, 0)
		} else {
			ui.Sel = min(ui.Sel+1, max(rows-1, 0))
		}
	case "pgup":
		ui.Scroll += 5
	case "pgdown":
		ui.Scroll = max(ui.Scroll-5, 0)
	case "end":
		ui.Scroll = 0
	case "r":
		switch ui.Tab {
		case screen.TabWho:
			return Action{Kind: SocialDo, ID: "who", Note: ui.WhoFilter}
		case screen.TabFriends:
			return Action{Kind: SocialDo, ID: "friends"}
		}
	case "i":
		if n := selected(); n != "" {
			return Action{Kind: SocialDo, ID: "invite", Note: n}
		}
	case "f":
		if n := selected(); n != "" && ui.Tab == screen.TabWho {
			return Action{Kind: SocialDo, ID: "friend add", Note: n}
		}
	case "d":
		if n := selected(); n != "" && ui.Tab == screen.TabFriends {
			return Action{Kind: SocialDo, ID: "friend remove", Note: n}
		}
	}
	return Action{}
}

// openSocial opens the Social screen on a tab.
func (a *App) openSocial(tab int) {
	if a.soc == nil {
		a.soc = &screen.SocialUI{}
	}
	if a.soc.Tab != tab {
		a.soc.Tab, a.soc.Sel = tab, 0
	}
	a.soc.Scroll = 0
}

// bannerLasts is how long a server announcement stays at the top of the screen
// if no key is pressed.
const bannerLasts = 10 * time.Second

// watchBanner notes when a server announcement first came and drops one that has
// been up long enough. The model is locked by the caller.
func (a *App) watchBanner(m *model.Model) {
	b := m.Banner
	switch {
	case b == nil:
		a.bannerN = 0
	case b.N != a.bannerN:
		a.bannerN, a.bannerAt = b.N, a.now()
	case a.now().Sub(a.bannerAt) > bannerLasts:
		m.ClearBanner()
	}
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// watchInvite notes when an invitation first came and drops one that has waited
// out its time. The model is locked by the caller.
func (a *App) watchInvite(m *model.Model) {
	inv := m.Social.Invite
	switch {
	case inv == nil:
		a.inviteN = 0
	case inv.N != a.inviteN:
		a.inviteN, a.inviteAt = inv.N, a.now()
	case a.now().Sub(a.inviteAt) > inviteLasts:
		m.ClearInvite()
	}
}

// words splits a command line: the command (lower case, without its slash) and
// the rest of the line.
func splitLine(text string) (cmd, rest string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", text
	}
	text = text[1:]
	cmd, rest, _ = strings.Cut(text, " ")
	return strings.ToLower(cmd), strings.TrimSpace(rest)
}

// socialSend runs one request about playing together. act.ID is the verb; for
// "line" the verb is read from the text the player typed.
func (a *App) socialSend(ctx context.Context, act Action) resultMsg {
	verb, arg := act.ID, act.Note
	if verb == "line" {
		cmd, rest := splitLine(arg)
		verb, arg = lineVerb(cmd), rest
		if cmd == "" {
			verb = "say"
		}
		if verb == "party?" {
			verb = map[bool]string{true: "partylist", false: "party"}[arg == ""]
		}
		if verb == "?" {
			return resultMsg{note: "/" + cmd + " is not a command: ? lists them", bad: true}
		}
	}
	bad := func(note string) resultMsg { return resultMsg{note: note, bad: true} }
	refused := func(what string, res *pb.SocialResult, err error) (resultMsg, bool) {
		switch {
		case err != nil:
			return bad(what + ": " + err.Error()), true
		case !res.GetOk() && res.GetReason() == "muted":
			// The server's own words for the mute came as a notice with the refusal.
			why := model.SocialReason("muted")
			a.sess.View(func(m *model.Model) { why = firstNonEmpty(m.Social.MutedText, why) })
			return bad(what + ": " + why), true
		case !res.GetOk():
			if res.GetReason() == "ooc_off" {
				a.sess.View(func(m *model.Model) { m.SetOOCOff(true) })
			}
			return bad(what + ": " + model.SocialReason(res.GetReason())), true
		}
		return resultMsg{}, false
	}
	name, rest := firstWord(arg)
	switch verb {
	case "say":
		res, err := a.sess.Say(ctx, arg)
		if r, no := refused("say", res, err); no {
			return r
		}
		return resultMsg{}
	case "party":
		res, err := a.sess.PartyChat(ctx, arg)
		if r, no := refused("party chat", res, err); no {
			return r
		}
		return resultMsg{}
	case "ooc", "oocsay":
		on := strings.ToLower(arg)
		switch {
		case arg == "":
			return bad("usage: /ooc TEXT says it to everyone online; /ooc off stops you hearing and speaking on it, /ooc on turns it back on")
		case verb == "ooc" && (on == "on" || on == "off"):
			res, err := a.sess.SetOOC(ctx, on == "on")
			if r, no := refused("ooc "+on, res, err); no {
				return r
			}
			a.sess.View(func(m *model.Model) { m.SetOOCOff(on == "off") })
			if on == "off" {
				return resultMsg{note: "ooc is off: you will not hear it and cannot speak on it (/ooc on turns it back on)"}
			}
			return resultMsg{note: "ooc is on: you hear it, and what you say on it goes to everyone who has it on"}
		}
		res, err := a.sess.OOC(ctx, arg)
		if r, no := refused("ooc", res, err); no {
			return r
		}
		return resultMsg{}
	case "report":
		if name == "" || rest == "" {
			return bad("usage: /report NAME what happened (it goes to the game masters with the latest chat)")
		}
		var before int
		a.sess.View(func(m *model.Model) { before = m.Social.ReportAck })
		res, err := a.sess.Report(ctx, name, rest)
		if r, no := refused("report "+name, res, err); no {
			return r
		}
		return resultMsg{note: a.reportAck(ctx, before)}
	case "whois", "whoisask":
		if name == "" {
			return bad("usage: /whois NAME")
		}
		card := a.whoisCard(ctx, name)
		if verb == "whoisask" {
			return resultMsg{after: func(a *App) { a.setCard(name, card) }}
		}
		return resultMsg{after: func(a *App) {
			a.chat.sel = nil
			a.chat.card = card
		}}
	case "tell", "reply":
		to, text := name, rest
		if verb == "reply" {
			a.sess.View(func(m *model.Model) { to = m.Social.LastTell })
			text = arg
			if to == "" {
				return bad("nobody has told you anything to reply to")
			}
		}
		if to == "" || text == "" {
			return bad("usage: /tell NAME text (/r text answers the last tell)")
		}
		res, err := a.sess.Tell(ctx, to, text)
		if r, no := refused("tell "+to, res, err); no {
			return r
		}
		return resultMsg{}
	case "mute", "unmute":
		if name == "" {
			res, err := a.sess.Mute(ctx, "list", "")
			if r, no := refused("mute list", res, err); no {
				return r
			}
			if len(res.GetNames()) == 0 {
				return resultMsg{note: "you have muted nobody"}
			}
			return resultMsg{note: "muted: " + strings.Join(res.GetNames(), ", ")}
		}
		res, err := a.sess.Mute(ctx, verb, name)
		if r, no := refused(verb+" "+name, res, err); no {
			return r
		}
		if verb == "mute" {
			return resultMsg{note: "muted " + name + ": their say, party, tell and ooc lines are hidden from you"}
		}
		return resultMsg{note: "unmuted " + name}
	case "invite", "kick", "accept", "decline":
		if name == "" && (verb == "invite" || verb == "kick") {
			return bad("usage: /" + verb + " NAME")
		}
		res, err := a.sess.Party(ctx, verb, name)
		if r, no := refused(verb, res, err); no {
			return r
		}
		return resultMsg{note: map[string]string{"invite": "invited " + name, "kick": "removed " + name + " from the party", "accept": "accepted the invitation", "decline": "declined the invitation"}[verb], clearInvite: verb == "accept" || verb == "decline"}
	case "leave":
		res, err := a.sess.Party(ctx, "leave", "")
		if r, no := refused("leave", res, err); no {
			return r
		}
		return resultMsg{note: "you left the party"}
	case "partylist":
		res, err := a.sess.Party(ctx, "list", "")
		if r, no := refused("party", res, err); no {
			return r
		}
		return resultMsg{}
	case "who":
		zone, class, lo, hi, err := whoFilter(arg)
		if err != nil {
			return bad(err.Error())
		}
		list, res, err := a.sess.Who(ctx, zone, class, lo, hi)
		if r, no := refused("who", res, err); no {
			return r
		}
		entries := list.GetEntries()
		return resultMsg{after: func(a *App) {
			a.openSocial(screen.TabWho)
			a.soc.Who, a.soc.WhoAsked, a.soc.WhoFilter = entries, true, arg
		}}
	case "friends":
		_, res, err := a.sess.Friends(ctx, "list", "")
		if r, no := refused("friends", res, err); no {
			return r
		}
		return resultMsg{after: func(a *App) { a.openSocial(screen.TabFriends) }}
	case "friend add", "friend remove", "friend":
		action := strings.TrimPrefix(verb, "friend ")
		if verb == "friend" {
			action, arg = name, strings.TrimSpace(rest)
			name, _ = firstWord(arg)
		} else {
			name = arg
		}
		if (action != "add" && action != "remove") || name == "" {
			return bad("usage: /friend add NAME or /friend remove NAME")
		}
		_, res, err := a.sess.Friends(ctx, action, name)
		if r, no := refused("friend "+action, res, err); no {
			return r
		}
		_, _, _ = a.sess.Friends(ctx, "list", "") // keeps the list the screen shows right
		if action == "add" {
			return resultMsg{note: name + " is a friend now"}
		}
		return resultMsg{note: name + " is not a friend any more"}
	case "anon":
		on := strings.EqualFold(name, "on")
		if !on && !strings.EqualFold(name, "off") {
			return bad("usage: /anon on (hide your zone from who and friends) or /anon off")
		}
		res, err := a.sess.Anonymous(ctx, on)
		if r, no := refused("anon", res, err); no {
			return r
		}
		if on {
			return resultMsg{note: "your zone is hidden from who and friends lists"}
		}
		return resultMsg{note: "your zone is shown in who and friends lists"}
	case "status":
		status := strings.ToLower(name)
		switch status {
		case "clear", "none", "off", "":
			status = ""
		case "away", "autopilot":
		default:
			return bad("usage: /status away, /status autopilot or /status clear (no effect on play)")
		}
		res, err := a.sess.Status(ctx, status)
		if r, no := refused("status", res, err); no {
			return r
		}
		if status == "" {
			return resultMsg{note: "status cleared"}
		}
		return resultMsg{note: "status: " + model.StatusWords(status) + " (no effect on rewards or play)"}
	case "chat":
		return resultMsg{after: func(a *App) { a.openSocial(screen.TabChat) }}
	case "help":
		return resultMsg{after: func(a *App) { a.view.Help, a.view.HelpPage, a.view.HelpScroll = true, 0, 0 }}
	case "quest", "dismiss":
		return resultMsg{after: func(a *App) {
			a.quest = &screen.QuestUI{}
			a.sess.View(func(m *model.Model) { a.quest.Confirm = verb == "dismiss" && m.Quest() != nil })
		}}
	}
	return bad("not a command: " + verb)
}

// lineVerb maps a command word to the request it makes, "?" for one it does not
// know.
func lineVerb(cmd string) string {
	switch cmd {
	case "":
		return "say"
	case "say", "s":
		return "say"
	case "p", "pc", "partychat":
		return "party"
	case "party":
		return "party?" // chat with text, the party list without
	case "t", "tell", "w", "whisper":
		return "tell"
	case "r", "reply":
		return "reply"
	case "ooc":
		return "ooc"
	case "oocsay", "report", "whois":
		return cmd
	case "mute", "ignore":
		return "mute"
	case "unmute", "unignore":
		return "unmute"
	case "invite", "inv":
		return "invite"
	case "accept", "decline", "kick", "leave", "who", "status", "friends", "friend", "anon", "chat", "help", "quest", "dismiss":
		return cmd
	}
	return "?"
}

// firstWord splits off the first word.
func firstWord(s string) (word, rest string) {
	word, rest, _ = strings.Cut(strings.TrimSpace(s), " ")
	return word, strings.TrimSpace(rest)
}

// whoFilter reads the filters of /who: zone=NAME, class=ID and a level range
// (3-7, 5- or -9). A bare word is a zone.
func whoFilter(arg string) (zone, class string, lo, hi uint32, err error) {
	for _, f := range strings.Fields(arg) {
		switch {
		case strings.HasPrefix(f, "zone="):
			zone = strings.TrimPrefix(f, "zone=")
		case strings.HasPrefix(f, "class="):
			class = strings.TrimPrefix(f, "class=")
		case strings.Contains(f, "-") && strings.Trim(f, "0123456789-") == "":
			a, b, _ := strings.Cut(f, "-")
			for i, s := range []string{a, b} {
				if s == "" {
					continue
				}
				n, perr := strconv.ParseUint(s, 10, 32)
				if perr != nil {
					return "", "", 0, 0, fmt.Errorf("/who: %q is not a level range", f)
				}
				if i == 0 {
					lo = uint32(n)
				} else {
					hi = uint32(n)
				}
			}
		default:
			return "", "", 0, 0, fmt.Errorf("/who: %q is not a filter: zone=NAME class=ID 3-7", f)
		}
	}
	return zone, class, lo, hi, nil
}

// setCard puts the server's answer on the card of name, if that card is still the
// one open.
func (a *App) setCard(name string, c *screen.CharCard) {
	if a.chat.card != nil && model.Key(a.chat.card.Name) == model.Key(name) {
		a.chat.card = c
	}
}

// reportAck is what the status line says of a report: the server's own
// acknowledgement, which follows its result, waited for a moment.
func (a *App) reportAck(ctx context.Context, before int) string {
	for range 20 {
		text := ""
		a.sess.View(func(m *model.Model) {
			if m.Social.ReportAck > before {
				for i := len(m.Log) - 1; i >= 0 && text == ""; i-- {
					if m.Log[i].Kind == "notice_report_filed" {
						text = m.Log[i].Text
					}
				}
			}
		})
		if text != "" {
			return text
		}
		select {
		case <-ctx.Done():
		case <-time.After(25 * time.Millisecond):
		}
	}
	return "report sent to the game masters"
}

// whoisCard asks the server who name is and makes the card of its answer.
func (a *App) whoisCard(ctx context.Context, name string) *screen.CharCard {
	reply, res, err := a.sess.Whois(ctx, name)
	switch {
	case err != nil:
		return &screen.CharCard{Name: name, State: screen.CardFailed, Err: err.Error()}
	case !res.GetOk() && res.GetReason() == "no_such_character", res.GetOk() && reply == nil:
		return &screen.CharCard{Name: name, State: screen.CardUnknown}
	case !res.GetOk():
		return &screen.CharCard{Name: name, State: screen.CardFailed, Err: model.SocialReason(res.GetReason())}
	}
	return &screen.CharCard{Name: firstNonEmpty(reply.GetName(), name), State: screen.CardFound, Kind: reply.GetKind(), Class: reply.GetClassId(),
		Level: reply.GetLevel(), Zone: reply.GetZone(), Online: reply.GetOnline(), Status: reply.GetStatus()}
}
