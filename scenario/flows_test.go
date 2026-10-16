package scenario_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/core"
	"github.com/GeneStevens/seq/testserver"
)

// The gate of Testgrounds is at (100000, 90000); a character that stands a
// metre south of it can use it.
const gateX, gateY = 100000, 89000

func one(account, character string) []testserver.Account {
	return []testserver.Account{{Name: account, Characters: []string{character}}}
}

func inside(m *model.Model) bool {
	return m.InInstance() && m.Ready() && m.Map != nil && m.Map.GetSpace() == m.Space
}

func back(m *model.Model) bool {
	return m.Ended != nil && !m.InInstance() && m.Map != nil && m.Map.GetSpace() == m.Space
}

func lastLog(m *model.Model) model.LogEntry { return m.Log[len(m.Log)-1] }

func notice(t *testing.T, p *scenario.Player, want string, bad bool) {
	t.Helper()
	if got, b := p.Notice(); !strings.Contains(got, want) || b != bad {
		t.Fatalf("the status line says %q (denied %v); want %q (denied %v)", got, b, want, bad)
	}
}

// Log in, walk, fight a scuttler with Tab and F, and take its drop with L.
func TestLoginMoveFightLoot(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria")})
	w.Fixture("play.jsonl")
	p := w.Login("ann", "aria")
	p.Golden("play_login")
	p.Model(func(m *model.Model) {
		if !m.Ready() || m.Self.ID != "aria" || m.Space != "Testgrounds" || m.Self.Pos.GetX() != 10000 {
			t.Fatalf("after login: %+v in %q", m.Self, m.Space)
		}
	})

	// Two steps east: 3 m a press, 4 m/s, so a second each.
	p.Press("right")
	w.Advance(time.Second)
	p.Press("right")
	w.Advance(time.Second)
	notice(t, p, "move ok", false)
	p.Model(func(m *model.Model) {
		if m.Self.Pos.GetX() != 16000 || m.Self.Pos.GetY() != 50000 {
			t.Fatalf("after two steps east: at %v", m.Self.Pos)
		}
	})

	// The scuttler at (25000, 50000) woke at 12 m. Tab picks it, F attacks.
	p.Press("tab")
	notice(t, p, "target ok", false)
	p.Press("f")
	p.Model(func(m *model.Model) {
		if m.Self.Target == "" || !m.Self.AutoAttack {
			t.Fatalf("target %q, auto-attack %v", m.Self.Target, m.Self.AutoAttack)
		}
	})
	w.Advance(8 * time.Second)
	p.Golden("play_fight")
	w.Advance(30 * time.Second)
	p.Model(func(m *model.Model) {
		if m.Self.XP != 20 || m.Self.Target != "" || len(m.Drops) != 1 {
			t.Fatalf("after the kill: xp %d, target %q, drops %d", m.Self.XP, m.Self.Target, len(m.Drops))
		}
	})
	p.Press("l")
	notice(t, p, "loot ok", false)
	p.Model(func(m *model.Model) {
		if !slices.Contains(m.Self.Inventory, "shell-fragment") || len(m.Drops) != 0 {
			t.Fatalf("after loot: inventory %v, drops %d", m.Self.Inventory, len(m.Drops))
		}
	})
	p.Golden("play_looted")
}

// Stand beside a scuttler until it kills the character, press R, and respawn.
func TestDieReleaseRespawn(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria"),
		Zone: func(_ string, z *core.Zone) { // a frail character, so a scuttler can kill it in seconds
			for i := range z.Characters {
				z.Characters[i].Stats.MaxHP = 6
			}
		}})
	w.Fixture("release.jsonl")
	w.Place("ann", "aria", 20000, 50000)
	p := w.Login("ann", "aria")
	w.Advance(30 * time.Second)
	p.Model(func(m *model.Model) {
		if !m.Self.Dead || m.Self.Released {
			t.Fatalf("dead %v released %v", m.Self.Dead, m.Self.Released)
		}
	})
	p.Golden("release_dead")
	p.Press("r")
	notice(t, p, "release ok", false)
	p.Model(func(m *model.Model) {
		if !m.Self.Released || lastLog(m).Kind != "released" {
			t.Fatalf("released %v, last log %+v", m.Self.Released, lastLog(m))
		}
	})
	p.Golden("release_released")
	w.Advance(6 * time.Second)
	p.Model(func(m *model.Model) {
		if m.Self.Dead || m.Self.Released || m.Self.Pos.GetX() != 10000 {
			t.Fatalf("after the respawn: %+v", m.Self)
		}
	})
	p.Golden("release_respawned")
}

// Cross the zone line with E, and come back with E.
func TestDoorAcrossAndBack(t *testing.T) {
	w := scenario.New(t, testserver.Options{Accounts: one("ann", "aria")})
	w.Fixture("door.jsonl")
	w.Place("ann", "aria", 194000, 10000)
	p := w.Login("ann", "aria")
	p.Golden("door_before")
	p.Press("e")
	p.Eventually("the other zone", func(m *model.Model) bool {
		return m.Space == "Outskirts" && m.Ready() && m.Map != nil && m.Map.GetSpace() == m.Space
	})
	if c, ok := w.Srv.Character("aria"); !ok || c.Space != testserver.Outskirts {
		t.Fatalf("the server has aria in %+v", c)
	}
	p.Golden("door_across")
	p.Press("e")
	p.Eventually("Testgrounds", func(m *model.Model) bool {
		return m.Space == "Testgrounds" && m.Ready() && m.Map != nil && m.Map.GetSpace() == m.Space
	})
	p.Golden("door_back")
}

func openRound(m *model.Model) bool { return inside(m) && m.Round != nil && m.Round.Open }

// Open solo-trial from the menu, enter it with E, clear it, and be handed back.
func TestSoloTrialClear(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria")})
	w.Fixture("solo_trial.jsonl")
	w.Place("ann", "aria", gateX, gateY)
	p := w.Login("ann", "aria")
	p.Press("i", "2")
	p.Golden("instance_menu")
	p.Press("enter")
	w.Advance(time.Second)
	notice(t, p, "instance requested", false)
	p.Model(func(m *model.Model) {
		if m.Opened == nil || m.Opened.Preset != "solo-trial" || m.InInstance() {
			t.Fatalf("opened %+v, inside %v", m.Opened, m.InInstance())
		}
	})
	p.Golden("instance_opened")
	p.Press("e")
	p.Eventually("the instance", inside)
	p.Model(func(m *model.Model) {
		in := m.Instance
		if in.GetPreset() != "solo-trial" || in.GetPartyCap() != 1 || in.GetTimeMode() != "realtime" || in.GetDifficulty() != "heroic" || m.TurnBased() {
			t.Fatalf("the server says the instance is %v", in)
		}
	})
	p.Golden("instance_inside")

	// The scuttler stands 14 m south and wakes at 12 m: one step closer, and it
	// comes. Then Tab, F, and the fight.
	p.Press("down")
	w.Advance(40 * time.Second)
	p.Press("tab", "f")
	notice(t, p, "attack ok", false)
	w.Advance(40 * time.Second)
	p.Model(func(m *model.Model) {
		if len(m.Drops) != 1 {
			t.Fatalf("drops %d", len(m.Drops))
		}
	})
	p.Press("l")
	notice(t, p, "loot ok", false)
	p.Eventually("the hand-back", back)
	p.Model(func(m *model.Model) {
		if m.Ended.Reason != "cleared" || m.Instance != nil || m.Round != nil || m.Self.Pos.GetY() < 85000 {
			t.Fatalf("ended %+v, instance %v, at %v", m.Ended, m.Instance, m.Self.Pos)
		}
	})
	if c, _ := w.Srv.Character("aria"); c.Space != testserver.Testgrounds {
		t.Fatalf("the server has aria in %q", c.Space)
	}
	p.Golden("instance_cleared")
}

// quick-turns, a round at a time: a plan built with the keys, edited, submitted
// and received; a deadline missed; a dropped connection; a fresh login into the
// open round; then the mob dead, the loot taken and the hand-back.
func TestQuickTurns(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria")})
	w.Fixture("quick_turns.jsonl")
	w.Place("ann", "aria", gateX, gateY)
	p := w.Login("ann", "aria")
	p.Press("i", "4", "enter")
	w.Advance(time.Second)
	p.Press("e")
	p.Eventually("round 1", openRound)
	p.Model(func(m *model.Model) {
		if !m.TurnBased() || m.Round.Number != 1 || m.Round.Participants != 1 || m.Instance.GetSleepLimitMs() == 0 {
			t.Fatalf("round %+v, instance %v", m.Round, m.Instance)
		}
	})

	// Build a plan, take a step back with Backspace, throw it away with Esc, build it again.
	p.Press("down", "down", "tab", "f")
	if got := p.Plan(); len(got) != 4 || got[3] != "attack on" {
		t.Fatalf("plan %q", got)
	}
	p.Press("backspace")
	if got := p.Plan(); len(got) != 3 {
		t.Fatalf("after Backspace: plan %q", got)
	}
	p.Press("esc")
	if got := p.Plan(); len(got) != 0 {
		t.Fatalf("after Esc: plan %q", got)
	}
	p.Press("down", "down", "tab")
	if got := p.Plan(); len(got) != 3 {
		t.Fatalf("plan %q", got)
	}
	p.Golden("round_plan")

	// Enter sends it; the answer is the receipt.
	p.Press("enter")
	notice(t, p, "plan accepted, counts even if you disconnect", false)
	p.Golden("round_receipt")
	w.Advance(100 * time.Millisecond)
	p.Model(func(m *model.Model) {
		r := m.Round
		if r.Number != 2 || !r.Open || r.Resolved != 1 || r.ResolvedBy != "all_submitted" || r.PlayedBy("aria") != "own_plan" {
			t.Fatalf("round %+v", r)
		}
		if e := lastLog(m); e.Kind != "round_started" {
			t.Fatalf("last log %+v", e)
		}
		if !strings.Contains(m.Log[len(m.Log)-2].Text, "you played your own plan") {
			t.Fatalf("the round log says %q", m.Log[len(m.Log)-2].Text)
		}
	})

	p.Mark("round_ran")

	// Round 2 gets no plan: the deadline passes with the clock.
	w.Advance(21 * time.Second)
	p.Model(func(m *model.Model) {
		r := m.Round
		if r.Number != 3 || !r.Open || r.Resolved != 2 || r.ResolvedBy != "deadline" || r.PlayedBy("aria") != "safe_default" {
			t.Fatalf("round %+v", r)
		}
		if !strings.Contains(m.Log[len(m.Log)-2].Text, "you played the safe default") {
			t.Fatalf("the round log says %q", m.Log[len(m.Log)-2].Text)
		}
	})
	p.Golden("round_missed")

	// The connection drops in round 3 and the program resumes it: nothing
	// repeated, nothing missed.
	p.Drop()
	p.Model(func(m *model.Model) {
		if m.Connections != 2 || m.Duplicates != 0 || len(m.Faults) != 0 || m.Round.Number != 3 || !m.Round.Open {
			t.Fatalf("after the resume: connections %d, duplicates %d, faults %v, round %+v", m.Connections, m.Duplicates, m.Faults, m.Round)
		}
	})
	p.Golden("round_resumed")

	// A fresh login into the open round: the snapshot, and the round told again.
	p.Fresh()
	p.Eventually("round 3 again", openRound)
	p.Model(func(m *model.Model) {
		if m.Connections != 1 || m.Round.Number != 3 || !m.TurnBased() || m.Instance.GetPreset() != "quick-turns" {
			t.Fatalf("round %+v, instance %v", m.Round, m.Instance)
		}
	})
	p.Golden("round_fresh")

	// Play on with empty plans (the fight goes on by standing orders), taking
	// the loot when it drops, until the instance is cleared.
	for range 40 {
		var drops int
		var attacking bool
		p.Model(func(m *model.Model) { drops, attacking = len(m.Drops), m.Self.AutoAttack })
		if drops > 0 {
			p.Mark("round_drop")
			p.Press("l")
		}
		if !attacking && drops == 0 {
			p.Press("f")
		}
		p.Press("enter")
		w.Advance(100 * time.Millisecond)
		var done bool
		p.Model(func(m *model.Model) { done = m.Ended != nil })
		if done {
			break
		}
	}
	p.Eventually("the hand-back", back)
	p.Model(func(m *model.Model) {
		if m.Ended.Reason != "cleared" || m.Round != nil || m.TurnBased() {
			t.Fatalf("ended %+v, round %+v", m.Ended, m.Round)
		}
	})
	p.Golden("round_cleared")
}

// A duo on one account: two TUIs, one character each, open duo-turns together
// and enter in the gathering round.
func TestDuoGatheringRound(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds},
		Accounts: []testserver.Account{{Name: "ann", Characters: []string{"aria", "bram"}}}})
	w.Fixture("duo_turns.jsonl")
	w.Place("ann", "aria", gateX, gateY)
	w.Place("ann", "bram", gateX, gateY)
	a := w.Login("ann", "aria")
	b := w.Login("ann", "bram")
	a.Press("i", "5", "enter")
	w.Advance(time.Second)
	for _, p := range []*scenario.Player{a, b} {
		p.Model(func(m *model.Model) {
			if m.Opened == nil || m.Opened.Preset != "duo-turns" || m.Opened.PartyCap != 2 {
				t.Fatalf("%s: opened %+v", p.Character, m.Opened)
			}
		})
	}
	// Aria goes in; the round waits for her alone, until Bram comes.
	a.Press("e")
	a.Eventually("round 1", openRound)
	a.Model(func(m *model.Model) {
		if m.Round.Participants != 1 {
			t.Fatalf("aria alone: %+v", m.Round)
		}
	})
	b.Press("e")
	b.Eventually("round 1", openRound)
	a.Eventually("two in round 1", func(m *model.Model) bool { return m.Round != nil && m.Round.Participants == 2 })
	for _, p := range []*scenario.Player{a, b} {
		p.Model(func(m *model.Model) {
			if m.Round.Number != 1 || m.Round.Participants != 2 || len(m.Round.Submitted) != 0 || m.Instance.GetPartyCap() != 2 {
				t.Fatalf("%s: round %+v", p.Character, m.Round)
			}
		})
	}
	a.Golden("duo_gathered")

	// Aria sends a plan: one of two. She edits it and sends it again: the last wins.
	a.Press("up", "enter")
	notice(t, a, "plan accepted", false)
	a.Golden("duo_one_of_two")
	a.Press("backspace", "right", "enter")
	notice(t, a, "plan accepted", false)
	// Bram, logging in fresh, is told who has submitted: the round again, with Aria in.
	b.Fresh()
	b.Eventually("the round told again", func(m *model.Model) bool { return openRound(m) && len(m.Round.Submitted) == 1 })
	b.Golden("duo_bram_told")
	b.Press("enter")
	w.Advance(100 * time.Millisecond)
	for _, p := range []*scenario.Player{a, b} {
		p.Model(func(m *model.Model) {
			r := m.Round
			if r.Resolved != 1 || r.ResolvedBy != "all_submitted" || r.ResolvedParticipants != 2 || len(r.ResolvedSubmitted) != 2 ||
				r.PlayedBy("aria") != "own_plan" || r.PlayedBy("bram") != "own_plan" {
				t.Fatalf("%s: round %+v", p.Character, r)
			}
		})
	}
	a.Golden("duo_two_of_two")
	if c, _ := w.Srv.Character("aria"); c.Entity.Pos.X != gateX+3000 {
		t.Fatalf("aria's last plan was up then right: she stands at %v", c.Entity.Pos)
	}
}
