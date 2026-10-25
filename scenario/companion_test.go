package scenario_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/content"
	"github.com/GeneStevens/seq/core"
	"github.com/GeneStevens/seq/testserver"
)

// The scenarios of the house companion and a second character on a stance
// (SEQ-049, D35, D42, D46): each presses its own keys.

// captainAt is a metre west of Captain Rhosyn Hartley, who stands at (104, 50) m.
const captainX, captainY = 103000, 50000

// levelUp makes the character a warrior of the level, knowing what the class
// knows at level 1 (as the other scenarios do), with a full set of hit points.
func levelUp(level int, edit func(*core.CharacterSave)) func(*core.CharacterSave) {
	return levelUpAs("warrior", level, edit)
}

// levelUpAs is levelUp for a character of any class.
func levelUpAs(class string, level int, edit func(*core.CharacterSave)) func(*core.CharacterSave) {
	return func(cs *core.CharacterSave) {
		z, _ := content.Zone(scenario.Thornwick)
		cl := z.Class(class)
		cs.Stats, cs.Known, cs.Slots, cs.XP = core.ClassCharacter(cl, *z.CharacterTemplate(class), level)
		cs.Known, cs.Slots = cl.KnownAt(1), core.DefaultSlots(cl.KnownAt(1))
		cs.HP = core.MaxHP(cs.Stats)
		if edit != nil {
			edit(cs)
		}
	}
}

func warriorAccount(chars ...string) []testserver.Account {
	classed := map[string]string{}
	for _, c := range chars {
		classed[c] = "warrior"
	}
	return []testserver.Account{{Name: "ann", Classed: classed}}
}

// The Watch Captain offers the first step of her quest and no other: talking
// to her opens her board, the Quest panel lists the step, and taking it starts
// a scout.
func TestCompanionQuestFirstStepFromTheCaptain(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: warriorAccount("hale")})
	w.Fixture("companion_quest.jsonl")
	w.Place("ann", "hale", captainX, captainY)
	w.Srv.Give("hale", levelUp(6, nil))
	p := w.Login("ann", "hale")
	p.Press("g")
	notice(t, p, "talk to Captain Rhosyn Hartley ok", false)
	p.Model(func(m *model.Model) {
		d := m.Said("captain")
		if d == nil || len(d.Options) != 3 || d.Options[0].GetAction() != "companion_role" || d.Options[1].GetAction() != "contract_accept" {
			t.Fatalf("the Captain said %+v", d)
		}
		steps := m.QuestSteps()
		if len(steps) != 1 || steps[0].Name != "Eyes on the Goblins" || steps[0].Quest != "companion" || steps[0].Step != 1 || steps[0].State != model.StepOffered || !m.QuestShown() {
			t.Fatalf("the quest as seen: %+v", steps)
		}
	})
	p.Golden("companion_talk")
	// The role cannot be chosen yet: the server says why.
	p.Press("1", "1", "enter")
	notice(t, p, "You haven't earned a companion yet", true)
	p.Press("2", "enter")
	notice(t, p, "take Eyes on the Goblins ok", false)
	p.Press("esc", "esc")
	p.Model(func(m *model.Model) {
		c := m.Contract("companion-goblin-lookout")
		steps := m.QuestSteps()
		if c == nil || c.GetKind() != "scout" || c.GetQuest() != "companion" || c.GetStep() != 1 || len(steps) != 1 || steps[0].State != model.StepHeld {
			t.Fatalf("after taking the step: %v, %+v", c, steps)
		}
	})
	p.Golden("quest_step_taken")
}

// Back from the lookout the step is ready; the Quest panel says to hand it in.
// The Downs are another zone, so this one begins with the contract held.
func TestCompanionQuestScoutTheLookout(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: []testserver.Account{{Name: "ann", Home: scenario.Downs, Classed: map[string]string{"hale": "warrior"}}}})
	w.Fixture("companion_lookout.jsonl")
	w.Place("ann", "hale", 216000, 140000)
	w.Srv.Give("hale", levelUp(6, func(cs *core.CharacterSave) {
		cs.Contracts = []core.ContractState{{ID: "companion-goblin-lookout", Kind: "scout", Zone: "thornwick-downs", Match: "goblin-lookout", Need: 1, Name: "Eyes on the Goblins"}}
	}))
	p := w.Login("ann", "hale")
	p.Model(func(m *model.Model) {
		if c := m.Contract("companion-goblin-lookout"); c == nil || c.GetReady() || m.Marker(c) == nil {
			t.Fatalf("the scout contract: %v", c)
		}
	})
	p.Golden("quest_lookout_far")
	walk(w, p, "right", 4)
	p.Model(func(m *model.Model) {
		if c := m.Contract("companion-goblin-lookout"); c == nil || !c.GetReady() || !hasLog(m, "reward", "Eyes on the Goblins: 1 of 1") {
			t.Fatalf("at the lookout: %v", c)
		}
	})
	p.Golden("quest_lookout_reached")
}

// The end of the quest: the last step is handed in, the Quest panel says to
// choose a role, the Captain's second option opens the choice, and the recruit
// is called to the character's side.
func TestCompanionQuestEndsWithTheChoiceOfARole(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: warriorAccount("hale")})
	w.Fixture("companion_role.jsonl")
	w.Place("ann", "hale", captainX, captainY)
	w.Srv.Give("hale", levelUp(6, func(cs *core.CharacterSave) {
		cs.Companion = &core.CompanionRecord{Stage: 1}
		cs.Contracts = []core.ContractState{{ID: "companion-raiding-party", Kind: "hunt", Zone: "thornwick-downs", Match: "goblin", Need: 5, Have: 5, Name: "Break the Raiding Party"}}
	}))
	p := w.Login("ann", "hale")
	p.Press("g", "3") // the Captain's board, on the contracts held
	p.Golden("quest_ready")
	p.Press("enter")
	notice(t, p, "hand in Break the Raiding Party ok", false)
	w.Advance(time.Second)
	p.Model(func(m *model.Model) {
		c := m.Companion()
		if !c.GetUnlocked() || c.GetRole() != "" || len(m.Self.Contracts) != 0 || !hasLog(m, "companion", "You have earned a house companion") {
			t.Fatalf("after the last step: %v, %v", c, m.Self.Contracts)
		}
	})
	p.Press("esc", "esc")
	p.Golden("quest_done")

	p.Press("g", "1")
	p.Golden("choose_role")
	p.Press("3", "enter") // a blade
	notice(t, p, "recruit a melee ok", false)
	w.Advance(time.Second)
	p.Model(func(m *model.Model) {
		if c := m.Companion(); c.GetRole() != "melee" || c.GetPresent() || !hasLog(m, "companion", "takes the role of melee") {
			t.Fatalf("after choosing: %v", c)
		}
	})
	p.Press("esc", "esc")
	p.Golden("companion_away")

	p.Press("p")
	notice(t, p, "call the companion ok", false)
	w.Advance(time.Second)
	p.Model(func(m *model.Model) {
		g := m.CompanionMember()
		if c := m.Companion(); !c.GetPresent() || !c.GetSummoned() || g == nil || g.Entity.GetName() != "Watch Skirmisher" || g.Entity.GetLevel() != 6 {
			t.Fatalf("after calling it: %v, %v", c, g)
		}
	})
	p.Golden("companion_called")
	if s := p.Screen(80, 24); strings.Contains(s, "┌─ Quest") {
		t.Fatalf("the Quest panel stays after the quest is done:\n%s", s)
	}
}

// downsAccount is an account whose warrior wakes in the Downs.
func downsAccount() []testserver.Account {
	return []testserver.Account{{Name: "ann", Home: scenario.Downs, Classed: map[string]string{"hale": "warrior"}}}
}

// companionRecord is a finished quest with the role chosen and the recruit called.
func companionRecord(role, stance string) core.CompanionRecord {
	return core.CompanionRecord{Stage: 2, Unlocked: true, Role: role, Stance: stance, Summoned: true}
}

// companionActs counts the log's lines of the companion fighting.
func companionActs(m *model.Model) int {
	n := 0
	for _, e := range m.Log {
		if e.Actor == "comp-hale" && (e.Kind == "combat_hit" || e.Kind == "combat_miss" || e.Kind == "ability_used") {
			n++
		}
	}
	return n
}

// advanceUntil runs the game clock in steps of step, up to limit of them, until
// done holds.
func advanceUntil(t *testing.T, w *scenario.World, p *scenario.Player, what string, step time.Duration, limit int, done func(*model.Model) bool) {
	t.Helper()
	for range limit {
		ok := false
		p.Model(func(m *model.Model) { ok = done(m) })
		if ok {
			return
		}
		w.Advance(step)
	}
	t.Fatalf("gave up waiting for %s", what)
}

// The companion on assist fights what its owner fights; the owner then
// overrides it (J: stand and fight nothing) and orders it back to the fight (Y).
func TestCompanionAssistsThenIsOverridden(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: downsAccount()})
	w.Fixture("companion_fight.jsonl")
	w.Place("ann", "hale", 146000, 95000) // beside the Bramble Tuskers' camp, out of their notice
	w.Srv.Give("hale", levelUp(5, nil))
	w.Srv.GiveCompanion("hale", companionRecord("tank", ""))
	p := w.Login("ann", "hale")
	p.Model(func(m *model.Model) {
		g := m.CompanionMember()
		if c := m.Companion(); !c.GetPresent() || c.GetStance() != "follow" || g == nil || g.Entity.GetName() != "Watch Shieldman" {
			t.Fatalf("at login: %v, %v", c, g)
		}
	})
	p.Golden("companion_follow")

	p.Press("h")
	notice(t, p, "companion stance assist ok", false)
	w.Advance(time.Second)
	p.Model(func(m *model.Model) {
		if m.Companion().GetStance() != "assist" || !hasLog(m, "companion", "takes the assist stance") {
			t.Fatalf("after H: %v", m.Companion())
		}
	})
	p.Press("tab")
	walk(w, p, "right", 2)
	p.Press("f")
	advanceUntil(t, w, p, "the companion to join the fight", 2*time.Second, 40, func(m *model.Model) bool { return companionActs(m) > 0 })
	p.Golden("companion_assist")

	// The override: it stands, and what it did stays what it did.
	p.Press("j")
	notice(t, p, "companion order: stop ok", false)
	w.Advance(time.Second)
	var acts int
	p.Model(func(m *model.Model) {
		acts = companionActs(m)
		if m.Companion().GetOrder() != "stop" {
			t.Fatalf("after J: %v", m.Companion())
		}
	})
	p.Golden("companion_override")
	w.Advance(6 * time.Second)
	p.Model(func(m *model.Model) {
		if n := companionActs(m); n != acts {
			t.Fatalf("the companion fought on after the order to stand: %d acts, then %d", acts, n)
		}
	})

	// Y sends it after the target again, and Z gives the stance back.
	p.Press("y")
	notice(t, p, "companion order: attack ok", false)
	advanceUntil(t, w, p, "the companion to fight again", 2*time.Second, 40, func(m *model.Model) bool { return companionActs(m) > acts })
	p.Press("z")
	notice(t, p, "companion order: clear ok", false)
	w.Advance(time.Second)
	p.Model(func(m *model.Model) {
		if m.Companion().GetOrder() != "" || m.Companion().GetStance() != "assist" {
			t.Fatalf("after Z: %v", m.Companion())
		}
	})
}

// H goes through the five stances and back; the Companion panel and the
// server's own state agree at each.
func TestCompanionStancesCycle(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: downsAccount()})
	w.Place("ann", "hale", 100000, 100000)
	w.Srv.Give("hale", levelUp(5, nil))
	w.Srv.GiveCompanion("hale", companionRecord("healer", ""))
	p := w.Login("ann", "hale")
	for _, want := range []string{"assist", "guard", "heal", "passive", "follow"} {
		p.Press("h")
		notice(t, p, "companion stance "+want+" ok", false)
		w.Advance(time.Second)
		p.Model(func(m *model.Model) {
			if m.Companion().GetStance() != want || !hasLog(m, "companion", "takes the "+want+" stance") {
				t.Fatalf("after H: %v", m.Companion())
			}
		})
		if s := p.Screen(80, 24); !strings.Contains(s, "stance "+want) {
			t.Fatalf("the panel does not say the stance is %s:\n%s", want, s)
		}
		if want == "heal" {
			p.Golden("companion_heal_stance")
		}
	}
}

// The companion goes with the connection and is back with it (D35): the stance
// it was on is kept through a drop and a fresh login.
func TestCompanionDisconnectsAndReturns(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: downsAccount()})
	w.Fixture("companion_reconnect.jsonl")
	w.Place("ann", "hale", 100000, 100000)
	w.Srv.Give("hale", levelUp(5, nil))
	w.Srv.GiveCompanion("hale", companionRecord("caster", "guard"))
	p := w.Login("ann", "hale")
	present := func(m *model.Model) bool {
		c, g := m.Companion(), m.CompanionMember()
		return c.GetPresent() && c.GetStance() == "guard" && g != nil && g.Entity.GetName() == "Watch Spellcaster"
	}
	p.Model(func(m *model.Model) {
		if !present(m) {
			t.Fatalf("at login: %v", m.Companion())
		}
	})
	p.Mark("companion_with_me")
	p.Drop()
	w.Advance(2 * time.Second)
	p.Eventually("the companion with me again", present)
	p.Golden("companion_after_drop")

	// Leaving by the program's own logout takes the companion along (it is not
	// independent); a fresh login brings it back by itself.
	p.Fresh()
	w.Advance(2 * time.Second)
	p.Eventually("the companion after a fresh login", present)
	p.Model(func(m *model.Model) {
		if c := m.Companion(); !c.GetSummoned() || c.GetRole() != "caster" {
			t.Fatalf("after a fresh login: %v", c)
		}
	})
	p.Golden("companion_fresh_login")

	// P sends it away, and it stays away across a fresh login too.
	p.Press("p")
	notice(t, p, "send the companion away ok", false)
	w.Advance(time.Second)
	p.Model(func(m *model.Model) {
		if c := m.Companion(); c.GetPresent() || c.GetSummoned() || m.CompanionMember() != nil {
			t.Fatalf("after P: %v", c)
		}
	})
	p.Fresh()
	w.Advance(time.Second)
	p.Model(func(m *model.Model) {
		if c := m.Companion(); c.GetPresent() || m.CompanionMember() != nil {
			t.Fatalf("a dismissed companion came back: %v", c)
		}
	})
	p.Golden("companion_dismissed")
}

// In a turn-based instance the keys for the companion add steps to the plan,
// and the companion does what the plan says: it fights once told to attack.
func TestTurnBasedRoundWithACompanionStep(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria")})
	w.Fixture("companion_turns.jsonl")
	w.Place("ann", "aria", gateX, gateY)
	w.Srv.GiveCompanion("aria", core.CompanionRecord{Stage: 2, Unlocked: true, Role: "tank", Summoned: true})
	p := w.Login("ann", "aria")
	openPreset(t, p, "duo-turns", true)
	w.Advance(time.Second)
	p.Press("e")
	p.Eventually("round 1", openRound)
	p.Model(func(m *model.Model) {
		if c := m.Companion(); !c.GetPresent() || m.CompanionMember() == nil {
			t.Fatalf("the companion is not in the instance: %v", c)
		}
	})
	// The companion is told by the plan: a stance, and an order to attack the
	// target the plan chooses.
	p.Press("h")
	p.Press("tab", "y")
	want := []string{"companion: stance assist", "target Plain Scuttler", "companion: attack Plain Scuttler"}
	if got := p.Plan(); !slices.Equal(got, want) {
		t.Fatalf("plan %q; want %q", got, want)
	}
	// One stance and one order for it: a second is not added.
	p.Press("h")
	notice(t, p, "one companion stance and one companion order", true)
	p.GoldenWithout("companion_plan", arrival)
	p.Press("enter")
	notice(t, p, "plan accepted, counts even if you disconnect", false)
	w.Advance(100 * time.Millisecond)
	advanceUntil(t, w, p, "the companion to fight", 500*time.Millisecond, 40, func(m *model.Model) bool {
		for _, e := range m.Log {
			if e.Actor == "comp-aria" && (e.Kind == "combat_hit" || e.Kind == "combat_miss") {
				return true
			}
		}
		return false
	})
	p.Model(func(m *model.Model) {
		if c := m.Companion(); c.GetStance() != "assist" || c.GetOrder() != "attack" {
			t.Fatalf("after the round: %v", c)
		}
	})
	p.GoldenWithout("companion_round", arrival)
}

// A second character of the account plays by a stance beside the first (D42):
// P brings it in, H and the orders command it, and it fights on guard. (Sending it away logs it out; the test server can stall five seconds when a session ends while it settles, so no scenario does that.)
func TestSecondCharacterOnGuard(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: []testserver.Account{{Name: "ann", Home: scenario.Downs,
		Classed: map[string]string{"hale": "warrior", "bram": "cleric"}}}})
	w.Fixture("duo_guard.jsonl")
	w.Place("ann", "hale", 146000, 95000)
	w.Place("ann", "bram", 144000, 95000)
	w.Srv.Give("hale", levelUp(5, nil))
	w.Srv.Give("bram", levelUpAs("cleric", 5, nil))
	p := w.LoginWithDuo("ann", "hale", "bram")
	p.Golden("duo_out_of_play")
	p.Press("p")
	notice(t, p, "second character in play", false)
	w.Advance(time.Second)
	p.Model(func(m *model.Model) {
		if g := m.InView("bram"); g == nil || g.Entity.GetName() != "bram" {
			t.Fatalf("the second character is not in view: %v", m.Entities)
		}
	})
	p.Golden("duo_party")
	p.Press("h", "h")
	notice(t, p, "second character: stance guard", false)

	// Hale walks up to a tusker and swings; Bram, on guard, fights what strikes Hale.
	p.Press("tab")
	walk(w, p, "right", 2)
	p.Press("f")
	advanceUntil(t, w, p, "the second character to fight", time.Second, 60, func(m *model.Model) bool {
		for _, e := range m.Log {
			if e.Actor == "bram" && (e.Kind == "combat_hit" || e.Kind == "combat_miss" || e.Kind == "ability_used") {
				return true
			}
		}
		return false
	})
	p.Golden("duo_guard_fight")
	p.Duo(func(m *model.Model) {
		if m.Self.ID != "bram" || m.Self.Class != "cleric" {
			t.Fatalf("the second character's own model: %q %q", m.Self.ID, m.Self.Class)
		}
	})

	// An order beats the stance: J has it stand and fight nothing.
	p.Press("j")
	notice(t, p, "second character ordered: stop", false)
	w.Advance(time.Second)
	w.Advance(time.Second)
	p.Duo(func(m *model.Model) {
		if m.Self.Target != "" || m.Self.AutoAttack || m.Self.Cast != nil {
			t.Fatalf("the second character, ordered to stand, is still at it: target %q", m.Self.Target)
		}
	})
	p.Golden("duo_ordered_to_stand")
}
