package scenario_test

import (
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/core"
	"github.com/GeneStevens/seq/testserver"
)

// The scenarios of M9, playing together (SEQ-056, D58): each presses its own keys,
// two or three TUIs on as many accounts against the one test server.

// command types a slash command on the command line and sends it: "/" opens
// the line, with the slash in it, and the rest is typed after it.
func command(p *scenario.Player, line string) {
	p.Type("/" + strings.TrimPrefix(line, "/") + "\r")
}

// saysLine types text that is no command: it is said. (The line opens with a
// slash in it; Backspace takes it away.)
func saysLine(p *scenario.Player, text string) {
	p.Type("/\x7f" + text + "\r")
}

func heard(p *scenario.Player, want string) bool {
	found := false
	p.Model(func(m *model.Model) {
		for _, l := range m.Social.Chat {
			if l.String() == want {
				found = true
			}
		}
	})
	return found
}

func mustHear(t *testing.T, p *scenario.Player, want string) {
	t.Helper()
	if !heard(p, want) {
		var all []string
		p.Model(func(m *model.Model) {
			for _, l := range m.Social.Chat {
				all = append(all, l.String())
			}
		})
		t.Fatalf("%s did not hear %q; chat: %q", p.Character, want, all)
	}
}

func mustNotHear(t *testing.T, p *scenario.Player, want string) {
	t.Helper()
	if heard(p, want) {
		t.Fatalf("%s heard %q", p.Character, want)
	}
}

func inLog(p *scenario.Player, kind, contains string) bool {
	ok := false
	p.Model(func(m *model.Model) { ok = hasLog(m, kind, contains) })
	return ok
}

// threeFriends is a server of three accounts, ann1 and bob1 side by side and cy1
// out of reach of a say, and a fourth account, dan1, that never logs in.
func threeFriends(t *testing.T, fixture string) (*scenario.World, *scenario.Player, *scenario.Player, *scenario.Player) {
	t.Helper()
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{
		{Name: "ann", Characters: []string{"ann1"}}, {Name: "bob", Characters: []string{"bob1"}},
		{Name: "cy", Characters: []string{"cy1"}}, {Name: "dan", Characters: []string{"dan1"}}}})
	w.Fixture(fixture)
	w.Place("ann", "ann1", 100000, 89000)
	w.Place("bob", "bob1", 100000, 89500)
	w.Place("cy", "cy1", 100000+2*core.DefaultViewRadiusMM, 89000)
	ann := w.Login("ann", "ann1")
	bob := w.Login("bob", "bob1")
	cy := w.Login("cy", "cy1")
	return w, ann, bob, cy
}

// Chat in each channel: a say reaches those near, a tell reaches one character
// anywhere, the party's line reaches the party and nobody else. The server's
// refusals (not online, no such character, too fast) are on the status line in
// words, and /mute hides a character from the one who muted it.
func TestChatInEveryChannel(t *testing.T) {
	w, ann, bob, cy := threeFriends(t, "chat_social.jsonl")

	// Say: ann and bob are side by side, cy is far off. Plain text is a say.
	saysLine(ann, "hello there")
	w.Settle()
	mustHear(t, ann, "[say] ann1: hello there") // a speaker sees its own line
	mustHear(t, bob, "[say] ann1: hello there")
	mustNotHear(t, cy, "[say] ann1: hello there")
	bob.Golden("chat_say")

	// Tell: cy hears it from far away; /r answers it.
	command(bob, "/tell cy1 psst, a word")
	w.Settle()
	mustHear(t, cy, "[tell] bob1: psst, a word")
	mustHear(t, bob, "[tell] to cy1: psst, a word")
	mustNotHear(t, ann, "[tell] bob1: psst, a word")
	command(cy, "/r on my way")
	w.Settle()
	mustHear(t, bob, "[tell] cy1: on my way")
	cy.Golden("chat_tell")

	// Refusals, plainly.
	command(ann, "/tell dan1 are you there")
	notice(t, ann, "not online", true)
	command(ann, "/tell nobody hello")
	notice(t, ann, "no character by that name", true)
	command(ann, "/tell cy1")
	notice(t, ann, "usage: /tell", true)
	command(ann, "/dance")
	notice(t, ann, "not a command", true)
	command(cy, "/p anyone?")
	notice(t, cy, "not in a party", true)
	command(cy, "/r")
	notice(t, cy, "usage: /tell", true)
	for i := range 5 { // a burst of five, and the sixth is too fast
		saysLine(ann, "spam "+string(rune('a'+i)))
	}
	saysLine(ann, "spam f")
	notice(t, ann, "talking too fast", true)
	ann.Golden("chat_refused")

	// Mute: cy mutes bob; his tell no longer reaches it, and cy is not told. The
	// list says who is muted.
	command(cy, "/mute bob1")
	notice(t, cy, "muted bob1", false)
	command(cy, "/mute")
	notice(t, cy, "muted: bob1", false)
	command(bob, "/tell cy1 are you there")
	w.Settle()
	mustNotHear(t, cy, "[tell] bob1: are you there")
	command(cy, "/unmute bob1")
	notice(t, cy, "unmuted bob1", false)
	w.Advance(time.Second)
	command(bob, "/tell cy1 are you back")
	w.Settle()
	mustHear(t, cy, "[tell] bob1: are you back")

	// The scrollback: more has been said than the screen holds. /chat opens it, Up
	// and PgUp scroll back, End goes to the newest, Esc closes it.
	for i := range 14 {
		w.Advance(time.Second)
		saysLine(ann, "line number "+string(rune('a'+i)))
	}
	w.Settle()
	ann.Mark("chat_scrolled")
	command(bob, "/chat")
	bob.Press("up", "up", "pgup")
	bob.Model(func(m *model.Model) {
		if len(m.Social.Chat) < 20 {
			t.Fatalf("only %d lines said", len(m.Social.Chat))
		}
	})
	bob.Golden("chat_scrollback")
	bob.Press("end", "tab", "tab", "tab")
	bob.Press("esc")
	if s := bob.Screen(80, 24); !strings.Contains(s, "line number n") {
		t.Fatalf("the chat panel does not show the newest line:\n%s", s)
	}
}

// Parties across accounts: an invitation is a prompt (Y or N), the Party panel
// lists the others with health, mana, level and where they are, the party's
// chat reaches the party, a leaver and a kicked member are told.
func TestPartiesAcrossAccounts(t *testing.T) {
	w, ann, bob, cy := threeFriends(t, "party_accounts.jsonl")

	// A decline: bob says no to ann's first invitation.
	command(ann, "/invite bob1")
	notice(t, ann, "invited bob1", false)
	w.Settle()
	bob.Golden("party_invite")
	bob.Press("n")
	notice(t, bob, "declined", false)
	w.Settle()
	if !inLog(ann, "notice_party_invite_declined", "bob1 declined") {
		t.Fatalf("ann was not told bob declined")
	}
	bob.Press("y") // no prompt now: nothing happens
	bob.Model(func(m *model.Model) {
		if m.Social.Invite != nil || m.InParty() {
			t.Fatalf("bob: %+v %v", m.Social.Invite, m.Social.Party)
		}
	})

	// An accept: the party forms on both.
	w.Advance(time.Second)
	command(ann, "/invite bob1")
	w.Settle()
	bob.Press("y")
	notice(t, bob, "accepted", false)
	w.Settle()
	for _, p := range []*scenario.Player{ann, bob} {
		p.Eventually("the party of two", func(m *model.Model) bool { return len(m.PartyMembers()) == 2 })
	}
	ann.Model(func(m *model.Model) {
		if !m.IsLeader() || m.Social.Party.GetLeader() != "ann1" {
			t.Fatalf("ann leads? %v", m.Social.Party)
		}
	})
	w.Advance(100 * time.Millisecond) // the space tells the whole party state on first sight, in its next tick (D60)
	bob.Eventually("ann's vitals in the party state", func(m *model.Model) bool { return mate(m, "ann1") })
	bob.Model(func(m *model.Model) {
		if m.IsLeader() {
			t.Fatalf("bob leads")
		}
		mates := m.Mates()
		if len(mates) != 2 || !mates[0].Leader || mates[0].Name != "ann1" || mates[0].Zone == "" || mates[0].MaxHP == 0 {
			t.Fatalf("bob's mates: %+v", mates)
		}
	})
	bob.Golden("party_formed")

	// Party chat: cy, far off and in no party, hears nothing of it.
	command(bob, "/p ready when you are")
	w.Settle()
	mustHear(t, ann, "[party] bob1: ready when you are")
	mustNotHear(t, cy, "[party] bob1: ready when you are")
	ann.Golden("party_chat")

	// A third joins by invitation; the leader may invite, a member may not kick.
	command(ann, "/invite cy1")
	w.Settle()
	cy.Press("y")
	w.Settle()
	cy.Eventually("cy in the party", func(m *model.Model) bool { return len(m.PartyMembers()) == 3 })
	bob.Eventually("bob sees three", func(m *model.Model) bool { return len(m.PartyMembers()) == 3 })
	command(bob, "/kick cy1")
	notice(t, bob, "only the party's leader", true)
	cy.Golden("party_far_member")

	// The leader removes cy; ann leaves, and the lead passes to bob.
	command(ann, "/kick cy1")
	notice(t, ann, "removed cy1", false)
	w.Settle()
	if !inLog(cy, "notice_party_kicked", "you were removed") {
		cy.Model(func(m *model.Model) { t.Fatalf("cy: %v", m.Log) })
	}
	cy.Model(func(m *model.Model) {
		if m.InParty() {
			t.Fatalf("cy is still in %v", m.Social.Party)
		}
	})
	command(ann, "/leave")
	notice(t, ann, "left the party", false)
	w.Settle()
	if !inLog(bob, "notice_party_left", "ann1 left") && !inLog(bob, "notice_party_disbanded", "disbanded") {
		bob.Model(func(m *model.Model) { t.Fatalf("bob: %v", m.Log) })
	}
	bob.Model(func(m *model.Model) {
		if m.InParty() {
			t.Fatalf("a party of one lingers: %v", m.Social.Party)
		}
	})
}

// Who and friends: the who list with a filter, the friends list with each
// friend's status, and a line in the log when a friend logs in or out.
func TestWhoAndFriends(t *testing.T) {
	_, ann, _, cy := threeFriends(t, "who_friends.jsonl")

	command(ann, "/who")
	ann.Model(func(m *model.Model) {
		if len(m.Social.Known) != 3 {
			t.Fatalf("ann knows %v", m.Social.Known)
		}
	})
	ann.Golden("who_list")
	command(ann, "/who 90-99")
	ann.Golden("who_nobody")
	command(ann, "/who zone=nowhere")
	ann.Press("esc")

	command(ann, "/friend add cy1")
	notice(t, ann, "cy1 is a friend now", false)
	command(ann, "/friend add dan1")
	command(ann, "/friend add ann1")
	notice(t, ann, "not yourself", true)
	command(ann, "/friends")
	ann.Golden("friends_list")
	ann.Press("esc")

	// cy's connection drops and comes back; ann, who has cy as a friend, is told of
	// each, in the log.
	cy.Drop()
	ann.Eventually("cy offline", func(m *model.Model) bool { return hasLog(m, "notice_friend_offline", "friend cy1 logged out") })
	ann.Eventually("cy online", func(m *model.Model) bool { return hasLog(m, "notice_friend_online", "friend cy1 logged in") })
	ann.Golden("friend_logged_in")
	command(ann, "/friends")
	ann.Golden("friends_after")
	ann.Press("esc")
	command(ann, "/friend remove cy1")
	notice(t, ann, "not a friend any more", false)
	command(ann, "/anon on")
	notice(t, ann, "hidden", false)
	command(ann, "/who")
	ann.Press("esc")

	// The help has a page on all of it.
	ann.Press("?", "right", "right", "right", "right", "right")
	ann.Golden("help_social")
	ann.Press("esc")
}

// walkBoth walks the players leg by leg to the point, a second of game time to
// a step, along x first and then y, each by its own arrows.
func walkBoth(t *testing.T, w *scenario.World, to [2]int64, ps ...*scenario.Player) {
	t.Helper()
	const step = 3000
	for range 400 {
		keys := map[*scenario.Player]string{}
		for _, p := range ps {
			var x, y int64
			p.Model(func(m *model.Model) { x, y = posOf(m) })
			dx, dy := to[0]-x, to[1]-y
			switch {
			case dx >= step:
				keys[p] = "right"
			case dx <= -step:
				keys[p] = "left"
			case dy >= step:
				keys[p] = "up"
			case dy <= -step:
				keys[p] = "down"
			}
		}
		if len(keys) == 0 {
			return
		}
		// Both walk together: the keys go down, ten ticks, three of them to hold past a tap, of walking (a step of
		// about 3 m), and come up.
		for _, p := range ps {
			if k, ok := keys[p]; ok {
				p.Hold(k)
			}
		}
		w.Advance(10 * scenario.Tick)
		for _, p := range ps {
			if k, ok := keys[p]; ok {
				p.Let(k)
			}
		}
		w.Advance(scenario.Tick)
	}
	t.Fatalf("the players did not reach %v", to)
}

// fightBoth closes both players in on the mob, targets it and swings, a second
// of game time to a step, until it is dead.
func fightBoth(t *testing.T, w *scenario.World, mob string, ps ...*scenario.Player) {
	t.Helper()
	for range 80 {
		dead := false
		for _, p := range ps {
			key := ""
			p.Model(func(m *model.Model) {
				e := m.Entities[mob]
				dead = dead || hasLog(m, "combat_death", "Warren Digger")
				if e == nil || e.GetDead() {
					return
				}
				dx, dy := e.GetPos().GetX()-m.Self.Pos.GetX(), e.GetPos().GetY()-m.Self.Pos.GetY()
				switch {
				case abs(dx) >= abs(dy) && abs(dx) > 2500:
					key = map[bool]string{true: "right", false: "left"}[dx > 0]
				case abs(dy) > 2500:
					key = map[bool]string{true: "up", false: "down"}[dy > 0]
				case m.Self.Target != mob:
					key = "tab"
				case !m.Self.AutoAttack:
					key = "f"
				}
			})
			if key != "" {
				p.Press(key)
			}
		}
		if dead {
			return
		}
		w.Advance(time.Second)
	}
	t.Fatalf("the fight with %s did not finish", mob)
}

// Two people on two accounts party up, chat, open a Warren instance together
// (one opens it, the other sees it and enters), clear the first room, and come
// out again. The cleric heals while the warrior holds the digger.
func TestTwoAccountsClearAWarrenRoomTogether(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{scenario.Warren}, Zone: shelved, Accounts: []testserver.Account{
		{Name: "ann", Home: scenario.Warren, Classed: map[string]string{"hale": "warrior"}},
		{Name: "bob", Home: scenario.Warren, Classed: map[string]string{"bram": "cleric"}}}})
	w.Fixture("party_warren.jsonl")
	w.Place("ann", "hale", 11000, 70000)
	w.Place("bob", "bram", 11000, 69000)
	w.Srv.Give("hale", levelUp(7, nil))
	w.Srv.Give("bram", levelUpAs("cleric", 7, nil))
	ann := w.Login("ann", "hale")
	bob := w.Login("bob", "bram")

	// Party up and talk.
	command(ann, "/invite bram")
	notice(t, ann, "invited bram", false)
	w.Settle()
	bob.Golden("warren_invited")
	bob.Press("y")
	w.Settle()
	for _, p := range []*scenario.Player{ann, bob} {
		p.Eventually("the party of two", func(m *model.Model) bool { return len(m.PartyMembers()) == 2 })
	}
	command(bob, "/p the delve gate is right here")
	w.Settle()
	mustHear(t, ann, "[party] bram: the delve gate is right here")
	saysLine(ann, "I will take the front")
	w.Settle()
	mustHear(t, bob, "[say] hale: I will take the front")
	ann.Golden("warren_party")

	// The member who is not the leader opens the delve; the leader sees it is
	// open for the party and has the door.
	w.Advance(time.Second)
	openPreset(t, bob, "venture-warren", true)
	w.Advance(time.Second)
	ann.Eventually("the party's delve open", func(m *model.Model) bool { return m.Opened != nil && m.Opened.Preset == "venture-warren" })
	if !inLog(ann, "instance_opened", "for your party") {
		ann.Model(func(m *model.Model) { t.Fatalf("hale's log: %v", m.Log) })
	}
	ann.Golden("warren_party_opened")
	// Bram, who opened it, goes in first, and hale follows through the same door.
	bob.Press("e")
	bob.Eventually("bram in the delve", inside)
	w.Advance(500 * time.Millisecond)
	ann.Press("e")
	ann.Eventually("hale in the delve", inside)
	var space string
	ann.Model(func(m *model.Model) { space = m.Space })
	bob.Model(func(m *model.Model) {
		if m.Space != space || len(m.Mates()) != 2 {
			t.Fatalf("bram is in %q (hale in %q), mates %+v", m.Space, space, m.Mates())
		}
	})
	w.Advance(time.Second)

	// Up to the first room, and the digger that patrols it.
	walkBoth(t, w, [2]int64{50000, 70000}, ann, bob)
	for range 20 {
		seen := false
		ann.Model(func(m *model.Model) { seen = m.Entities["mob-diggers-c-1"] != nil })
		if seen {
			break
		}
		w.Advance(time.Second)
	}
	fightBoth(t, w, "mob-diggers-c-1", ann, bob)
	for _, p := range []*scenario.Player{ann, bob} {
		p.Model(func(m *model.Model) {
			if !hasLog(m, "combat_death", "Warren Digger") {
				t.Fatalf("%s: the digger is not dead: %v", p.Character, m.Log)
			}
		})
	}
	bob.Model(func(m *model.Model) {
		if !hasLog(m, "combat_hit", "You hit") && !hasLog(m, "ability_used", "You use") && !hasLog(m, "combat_heal", "You heal") {
			t.Fatalf("bram took no part: %v", m.Log)
		}
	})
	// The digger's drop is taken (it appears a tick after the kill), so the
	// screens do not depend on when.
	advanceUntil(t, w, ann, "the digger's drop", time.Second, 10, func(m *model.Model) bool { return len(m.Drops) > 0 })
	advanceUntil(t, w, bob, "the digger's drop in bram's view", time.Second, 10, func(m *model.Model) bool { return len(m.Drops) > 0 })
	for range 6 {
		left := 0
		ann.Model(func(m *model.Model) { left = len(m.Drops) })
		if left == 0 {
			break
		}
		ann.Press("l")
		w.Advance(time.Second)
	}
	for _, p := range []*scenario.Player{ann, bob} {
		p.Eventually("the drops gone", func(m *model.Model) bool { return len(m.Drops) == 0 })
	}
	// Each has the other in view for the screens (the server tells a mover's
	// appearance and departure as it walks, and the walk is over).
	advanceUntil(t, w, ann, "bram in hale's view", 100*time.Millisecond, 12, func(m *model.Model) bool { return mate(m, "bram") })
	ann.Golden("warren_pair_room_cleared")

	// Out again, together; the party chat still reaches both.
	ann.Press("f")
	walkBoth(t, w, [2]int64{12000, 70000}, ann, bob)
	ann.Press("e")
	bob.Press("e")
	for _, p := range []*scenario.Player{ann, bob} {
		p.Eventually("out of the delve", func(m *model.Model) bool {
			return m.Space != space && m.Ready() && m.Map != nil && m.Map.GetSpace() == m.Space
		})
	}
	command(ann, "/p out safe")
	w.Settle()
	mustHear(t, bob, "[party] hale: out safe")
}

// mate says the party state has the member online, with its state.
func mate(m *model.Model, name string) bool {
	for _, mt := range m.Mates() {
		if mt.Name == name {
			return mt.Online && mt.Zone != "" && mt.MaxHP > 0
		}
	}
	return false
}

// In a turn-based instance of a party of two accounts the round waits for both:
// the Round panel names each member and marks who has sent a plan, which the
// server tells everyone (D50), and the round runs when both have.
func TestAPartysRoundShowsEveryMembersPlan(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{
		{Name: "ann", Characters: []string{"ann1"}}, {Name: "bob", Characters: []string{"bob1"}}}})
	w.Fixture("party_turns.jsonl")
	w.Place("ann", "ann1", gateX, gateY)
	w.Place("bob", "bob1", gateX+1000, gateY)
	ann := w.Login("ann", "ann1")
	bob := w.Login("bob", "bob1")
	command(ann, "/invite bob1")
	w.Settle()
	bob.Press("y")
	w.Settle()
	for _, p := range []*scenario.Player{ann, bob} {
		p.Eventually("the party of two", func(m *model.Model) bool { return len(m.PartyMembers()) == 2 })
	}
	w.Advance(time.Second)

	// Bob opens it; both walk in, bob first.
	openPreset(t, bob, "duo-turns", true)
	w.Advance(time.Second)
	bob.Press("e")
	bob.Eventually("round 1", openRound)
	w.Advance(500 * time.Millisecond)
	ann.Press("e")
	ann.Eventually("round 1", openRound)
	for _, p := range []*scenario.Player{ann, bob} {
		p.Eventually("both in the round", func(m *model.Model) bool { return m.Round != nil && m.Round.Participants == 2 })
	}
	for _, p := range []*scenario.Player{ann, bob} {
		other := map[string]string{"ann1": "bob1", "bob1": "ann1"}[p.Character]
		if s := p.Screen(140, 45); !strings.Contains(s, "0 of 2 submitted") || !strings.Contains(s, "[ ] you  [ ] "+other) {
			t.Fatalf("%s: the Round panel does not list both members:\n%s", p.Character, s)
		}
	}

	// Bob sends his plan: ann is told, and her panel marks him.
	bob.Press("enter")
	notice(t, bob, "plan accepted", false)
	w.Settle()
	ann.Eventually("bob's plan told", func(m *model.Model) bool { return m.Round.HasSubmitted("bob1") })
	if s := ann.Screen(140, 45); !strings.Contains(s, "1 of 2 submitted") || !strings.Contains(s, "[ ] you  [x] bob1") {
		t.Fatalf("ann's Round panel does not mark bob's plan:\n%s", s)
	}
	ann.Mark("party_round_bob_sent") // drawn from the recording (screen/cases.go): the countdown reads a clock the scenario does not hold

	// Ann sends hers: the round runs, and the log says what each played.
	ann.Press("enter")
	w.Advance(100 * time.Millisecond)
	for _, p := range []*scenario.Player{ann, bob} {
		p.Eventually("round 1 resolved", func(m *model.Model) bool { return m.Round != nil && m.Round.Resolved >= 1 })
		p.Model(func(m *model.Model) {
			if m.Round.PlayedBy("ann1") != "own_plan" || m.Round.PlayedBy("bob1") != "own_plan" {
				t.Fatalf("%s: round plays %+v", p.Character, m.Round.Plays)
			}
		})
	}
}
