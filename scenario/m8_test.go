package scenario_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/content"
	"github.com/GeneStevens/seq/core"
	"github.com/GeneStevens/seq/testserver"
)

// The scenarios of the M8 cleanup (SEQ-055, WLD-004, D56, D57): each presses its
// own keys.

// shelved is the server setting that turns the house companion off (D56), which
// is the default of seqd.
func shelved(_ string, z *core.Zone) { z.NoCompanions = true }

// With the house companion shelved the snapshot carries none, the Watch Captain
// offers no quest, and nothing of the companion shows: no panel, no help lines,
// no quest.
func TestNoCompanionUIWithTheDefaultSetting(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: warriorAccount("hale"), Zone: shelved})
	w.Fixture("shelved.jsonl")
	w.Place("ann", "hale", captainX, captainY)
	w.Srv.Give("hale", levelUp(6, nil))
	p := w.Login("ann", "hale")
	p.Model(func(m *model.Model) {
		if m.Companion() != nil {
			t.Fatalf("the snapshot carries a companion: %v", m.Companion())
		}
	})
	p.Press("g")
	notice(t, p, "talk to Captain Rhosyn Hartley ok", false)
	p.Model(func(m *model.Model) {
		if m.QuestShown() || len(m.QuestSteps()) > 0 {
			t.Fatalf("the Captain offers a quest step: %v", m.QuestSteps())
		}
	})
	p.Model(func(m *model.Model) {
		for _, o := range m.Said("captain").Options {
			if o.GetAction() == "companion_role" {
				t.Fatalf("the Captain's talk offers the recruit's role: %v", o)
			}
		}
	})
	p.Golden("shelved_captain")
	p.Press("esc")
	for _, sz := range scenario.Sizes {
		if s := p.Screen(sz[0], sz[1]); strings.Contains(s, "─ Companion") || strings.Contains(s, "─ Quest") {
			t.Fatalf("a companion panel at %dx%d:\n%s", sz[0], sz[1], s)
		}
	}
	// The help lists no companion keys, and the keys say there is no helper.
	p.Press("?")
	for _, sz := range scenario.Sizes {
		if s := p.Screen(sz[0], sz[1]); strings.Contains(s, "Companion") || strings.Contains(s, "Second character") {
			t.Fatalf("companion keys in the help at %dx%d:\n%s", sz[0], sz[1], s)
		}
	}
	p.Golden("shelved_help")
	p.Press("?", "h")
	notice(t, p, "there is no helper to command", false)
	p.Press("x")
	notice(t, p, "there is no second character to direct", false)
}

// duoAccount is an account with a warrior and a cleric, who wake in the Downs.
func duoAccount() []testserver.Account {
	return []testserver.Account{{Name: "ann", Home: scenario.Downs, Classed: map[string]string{"hale": "warrior", "bram": "cleric"}}}
}

// posOf is where the character stands, from its model.
func posOf(m *model.Model) (x, y int64) { return m.Self.Pos.GetX(), m.Self.Pos.GetY() }

// One session plays both characters of the account (D56): X switches the keys
// and the screen to the other one, which is moved, targeted and made to cast
// by the same keys, with the stance waiting; the status line says whom it is
// directing, Z hands it back to its stance, X gives the session back to the first.
func TestDirectYourSecondCharacter(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: duoAccount(), Zone: shelved})
	w.Place("ann", "hale", 146000, 95000) // beside the Bramble Tuskers' camp, out of their notice
	w.Place("ann", "bram", 144000, 95000)
	w.Srv.Give("hale", levelUp(5, nil))
	w.Srv.Give("bram", levelUpAs("cleric", 5, nil))
	p := w.LoginWithDuo("ann", "hale", "bram")

	// Out of play, there is nobody to direct.
	p.Press("x")
	notice(t, p, "not in play", false)
	p.Press("p")
	notice(t, p, "second character in play", false)
	w.Advance(time.Second)

	p.Press("x")
	notice(t, p, "directing bram", false)
	if s := status(p); !strings.Contains(s, "directing bram") {
		t.Fatalf("the status line says %q", s)
	}
	p.Model(func(m *model.Model) {
		if m.Self.ID != "hale" {
			t.Fatalf("hale's own model changed to %q", m.Self.ID)
		}
	})
	if s := p.Screen(80, 24); !strings.Contains(s, "bram  Lv 5 Cleric") {
		t.Fatalf("the screen is not bram's:\n%s", s)
	}
	p.Golden("directing_bram")

	// Move: the arrow keys move bram and not hale. The stance, which would follow
	// hale, waits.
	var hx, hy, bx, by int64
	p.Model(func(m *model.Model) { hx, hy = posOf(m) })
	p.Duo(func(m *model.Model) { bx, by = posOf(m) })
	walk(w, p, "up", 2)
	p.Model(func(m *model.Model) {
		if x, y := posOf(m); x != hx || y != hy {
			t.Fatalf("hale moved from (%d,%d) to (%d,%d) while bram was directed", hx, hy, x, y)
		}
	})
	p.Duo(func(m *model.Model) {
		if x, y := posOf(m); x != bx || y <= by+3000 {
			t.Fatalf("bram went from (%d,%d) to (%d,%d)", bx, by, x, y)
		}
	})
	w.Advance(3 * time.Second) // a stance would have walked it back by now
	p.Duo(func(m *model.Model) {
		if _, y := posOf(m); y <= by+3000 {
			t.Fatalf("bram's stance walked it back to y=%d", y)
		}
	})

	// Target and cast: Tab picks a tusker in bram's view; key 1 is bram's first
	// ability, not hale's Taunt.
	p.Press("tab")
	notice(t, p, "target ok", false)
	var target string
	p.Duo(func(m *model.Model) { target = m.Self.Target })
	if target == "" || !strings.Contains(target, "tusker") {
		t.Fatalf("bram's target is %q", target)
	}
	p.Model(func(m *model.Model) {
		if m.Self.Target != "" {
			t.Fatalf("hale was targeted too: %q", m.Self.Target)
		}
	})
	p.Press("1")
	notice(t, p, "key 1", false)
	w.Advance(2 * time.Second)
	p.Duo(func(m *model.Model) {
		if !hasLog(m, "ability_used", "") {
			t.Fatalf("bram cast nothing: %v", m.Log)
		}
	})
	p.Golden("directed_cast")

	// Follow: Z hands it back to its stance, which brings it back to hale.
	p.Press("esc")
	p.Press("z")
	notice(t, p, "back to its stance", false)
	for range 10 {
		w.Advance(time.Second)
	}
	p.Duo(func(m *model.Model) {
		if x, y := posOf(m); abs(x-hx) > 6000 || abs(y-hy) > 6000 {
			t.Fatalf("bram, on follow, is at (%d,%d); hale is at (%d,%d)", x, y, hx, hy)
		}
	})

	// X gives the session back to hale: the keys are hale's again.
	p.Press("x")
	notice(t, p, "you are back on your own character", false)
	if s := status(p); strings.Contains(s, "directing") {
		t.Fatalf("the status line still says %q", s)
	}
	p.Press("up")
	w.Advance(time.Second)
	p.Model(func(m *model.Model) {
		if _, y := posOf(m); y <= hy {
			t.Fatalf("hale did not move: y=%d", y)
		}
	})
	p.Golden("directing_over")
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// In a turn-based round the other character has a plan of its own (D56): the
// Round panel shows the plan being built for the one that is directed, one
// session submits it, and it counts as that character's own, so the round runs
// as soon as both plans are in.
func TestDirectedPlanInATurnBasedRound(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds},
		Accounts: []testserver.Account{{Name: "ann", Characters: []string{"aria", "bram"}}}, Zone: shelved})
	w.Place("ann", "aria", gateX, gateY)
	w.Place("ann", "bram", gateX, gateY)
	p := w.LoginWithDuo("ann", "aria", "bram")
	p.Press("p")
	notice(t, p, "second character in play", false)
	openPreset(t, p, "duo-turns", true)
	w.Advance(time.Second)

	// Bram is sent through the gate by a direct order, then Aria goes in.
	p.Press("x")
	p.Press("e")
	notice(t, p, "use door ok", false)
	p.Press("x")
	p.Press("e")
	p.Eventually("round 1", openRound)
	p.Eventually("two in round 1", func(m *model.Model) bool { return m.Round != nil && m.Round.Participants == 2 })
	p.DuoEventually("bram in the instance", func(m *model.Model) bool { return m.InInstance() && m.Round != nil && m.Round.Open })

	// Bram's plan, built from his own screen: he is the one who moves and targets.
	p.Press("x")
	p.Press("down", "tab", "f")
	want := []string{"move (10000,87000)", "target Plain Scuttler", "attack on"}
	var bx, by int64
	p.Duo(func(m *model.Model) { bx, by = posOf(m) })
	want[0] = "move (" + itoa(bx) + "," + itoa(by-3000) + ")"
	if got := p.Plan(); !slices.Equal(got, want) {
		t.Fatalf("bram's plan %q; want %q", got, want)
	}
	if s := p.Screen(140, 45); !strings.Contains(s, "bram's plan (3/16): not sent yet") {
		t.Fatalf("the Round panel does not show bram's plan:\n%s", s)
	}
	p.Golden("directed_plan")
	p.Press("enter")
	notice(t, p, "plan accepted", false)
	p.Model(func(m *model.Model) {
		if !m.Round.HasSubmitted("bram") || m.Round.HasSubmitted("aria") {
			t.Fatalf("submitted: %v", m.Round.Submitted)
		}
	})

	// Back on Aria: her plan is her own, empty until she builds it, and with it
	// both plans are in.
	p.Press("x")
	if got := p.Plan(); len(got) != 0 {
		t.Fatalf("aria's plan holds bram's steps: %q", got)
	}
	p.Press("down", "tab", "f", "enter")
	notice(t, p, "plan accepted", false)
	w.Advance(100 * time.Millisecond)
	p.Eventually("the round to run", func(m *model.Model) bool { return m.Round != nil && m.Round.Resolved == 1 })
	p.Model(func(m *model.Model) {
		if r := m.Round; r.ResolvedBy != "all_submitted" || r.PlayedBy("bram") != "own_plan" || r.PlayedBy("aria") != "own_plan" {
			t.Fatalf("round %+v: bram's directed plan did not count as his own", r)
		}
	})
	p.Golden("directed_plan_ran")
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// A character that has owed pay when it logs in sees it listed in the log, by
// the names of the event and of the instance preset it is owed for, and what
// belongs to another character is not shown (D55, D56).
func TestOwedPayIsListedAtLogin(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{{Name: "ann", Characters: []string{"aria", "bram"}}}})
	w.Fixture("owed_login.jsonl")
	z, err := content.Testgrounds()
	if err != nil {
		t.Fatal(err)
	}
	def, preset := z.WorldEvents[0], z.InstancePresets[0]
	p := w.Login("ann", "aria") // (a first login is a join, which pays what is owed)
	w.Srv.Owe(core.OwedPay{Key: core.OweKey("", def.ID, 1, "aria"), ID: "aria", Event: def.ID, Cycle: 1, Reward: core.ContractReward{XP: 30, Crowns: 15}})
	w.Srv.Owe(core.OwedPay{Key: core.OweKey(core.OwedRefund, preset.ID, 7, "aria"), ID: "aria", Event: preset.ID, Cycle: 7, Kind: core.OwedRefund, Reward: core.ContractReward{Crowns: 50}})
	w.Srv.Owe(core.OwedPay{Key: core.OweKey("", def.ID, 1, "bram"), ID: "bram", Event: def.ID, Cycle: 1, Reward: core.ContractReward{XP: 1}})
	p.Fresh() // and in again: the snapshot is made afresh
	p.Model(func(m *model.Model) {
		if len(m.Self.Owed) != 2 {
			t.Fatalf("owed pay in the snapshot: %v", m.Self.Owed)
		}
		if !hasLog(m, "reward", "owed pay waiting from the "+def.Name+": 30 XP and 15 crowns, paid at your next join") ||
			!hasLog(m, "reward", "owed pay waiting from a refund for "+preset.Name+": 50 crowns, paid at your next join") {
			t.Fatalf("the log does not list the owed pay: %v", m.Log)
		}
		if n := strings.Count(logText(m), "owed pay waiting"); n != 2 {
			t.Fatalf("%d owed pay lines: %v", n, m.Log)
		}
	})
	p.Golden("owed_pay_login")
}

func logText(m *model.Model) string {
	var b strings.Builder
	for _, e := range m.Log {
		b.WriteString(e.Text + "\n")
	}
	return b.String()
}

// The whole duo by hand (D56, D57, Pillar 10): two characters of one account,
// one session. They go into the Hollow Warren's Venture delve together (bram, too
// far from the gate to be sent through it, follows his leader by his stance),
// clear the first room (the digger of the diggings), and come out by direct
// orders, bram first.
func TestADuoByHandClearsAWarrenRoom(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{scenario.Warren},
		Accounts: []testserver.Account{{Name: "ann", Home: scenario.Warren, Classed: map[string]string{"hale": "warrior", "bram": "cleric"}}}, Zone: shelved})
	w.Fixture("duo_warren.jsonl")
	w.Place("ann", "hale", 11000, 70000)
	w.Place("ann", "bram", 24000, 70000)
	w.Srv.Give("hale", levelUp(7, nil))
	w.Srv.Give("bram", levelUpAs("cleric", 7, nil))
	p := w.LoginWithDuo("ann", "hale", "bram")
	p.Press("p")
	notice(t, p, "second character in play", false)
	w.Advance(time.Second)
	p.Model(func(m *model.Model) {
		if m.InView("bram") == nil {
			t.Fatalf("bram is not in hale's party: %v", m.Entities)
		}
	})
	p.Press("h") // assist: bram fights what hale targets
	notice(t, p, "stance assist", false)
	p.Golden("warren_duo")

	// Into the delve: hale takes the gate, and bram, out of reach of it, is left
	// to his stance, which walks him to the gate his leader left by.
	openPreset(t, p, "venture-warren", true)
	w.Advance(time.Second)
	p.Press("e")
	notice(t, p, "out_of_reach", false)
	p.Eventually("hale in the delve", inside)
	var space string
	p.Model(func(m *model.Model) { space = m.Space })
	for range 5 {
		w.Advance(time.Second)
	}
	p.DuoEventually("bram in the delve with hale", func(m *model.Model) bool { return m.Space == space && m.Ready() })
	p.Golden("warren_entered")

	// Up to the diggings, where the room's digger patrols. Bram, on assist, follows
	// and fights what hale targets.
	walkVia(t, w, p, [2]int64{52000, 70000})
	advanceUntil(t, w, p, "the digger in view", time.Second, 20, func(m *model.Model) bool { return m.Entities["mob-diggers-c-1"] != nil })
	p.Press("tab")
	p.Model(func(m *model.Model) {
		if m.Self.Target != "mob-diggers-c-1" {
			t.Fatalf("hale's target is %q", m.Self.Target)
		}
	})
	p.Golden("warren_digger")
	closeInAndFight(t, w, p, "mob-diggers-c-1", func(m *model.Model) bool { return hasLog(m, "combat_death", "Warren Digger") })
	p.Model(func(m *model.Model) {
		if !hasLog(m, "combat_hit", "bram") && !hasLog(m, "ability_used", "bram") {
			t.Fatalf("bram took no part in the fight: %v", m.Log)
		}
		if e := m.Entities["mob-diggers-c-1"]; e != nil && !e.GetDead() {
			t.Fatalf("the digger lives: %v", e)
		}
	})
	p.Golden("warren_room_cleared")

	// Back to the gate. Out by direct orders: X, and bram is walked up to the gate
	// with the arrow keys and sent through it with E, alone.
	p.Press("f") // auto-attack off: the way out is not a fight
	walkVia(t, w, p, [2]int64{13000, 70000})
	p.Press("x")
	notice(t, p, "directing bram", false)
	for range 20 {
		var bx int64
		p.Duo(func(m *model.Model) { bx, _ = posOf(m) })
		if bx <= 14000 {
			break
		}
		p.Press("left")
		w.Advance(time.Second)
	}
	p.Golden("warren_bram_at_gate")
	p.Press("e")
	notice(t, p, "use door ok", false)
	p.DuoEventually("bram out of the delve", func(m *model.Model) bool {
		return m.Ready() && m.Space != space && m.Map != nil && m.Map.GetSpace() == m.Space
	})
	p.Model(func(m *model.Model) {
		if m.Space != space {
			t.Fatalf("hale left with him: %q", m.Space)
		}
	})

	// Hale follows. The explicit door for the second character is not sent: it is
	// in another space, where the same door opens an instance.
	p.Press("x")
	p.Press("e")
	notice(t, p, "use door ok", false)
	if got, _ := p.Notice(); strings.Contains(got, "second character") {
		t.Fatalf("hale's door was also tried for bram: %q", got)
	}
	p.Eventually("hale out of the delve", func(m *model.Model) bool {
		return m.Ready() && m.Space != space && m.Map != nil && m.Map.GetSpace() == m.Space
	})
	var out string
	p.Model(func(m *model.Model) { out = m.Space })
	p.DuoEventually("both in the Warren", func(m *model.Model) bool { return m.Space == out })
	p.Golden("warren_out")
}

// closeInAndFight walks the character up to the mob and swings at it, a second
// of game time to a step, until done.
func closeInAndFight(t *testing.T, w *scenario.World, p *scenario.Player, mob string, done func(*model.Model) bool) {
	t.Helper()
	for range 60 {
		var key string
		finished := false
		p.Model(func(m *model.Model) {
			finished = done(m)
			e := m.Entities[mob]
			if e == nil || e.GetDead() || finished {
				return
			}
			dx, dy := e.GetPos().GetX()-m.Self.Pos.GetX(), e.GetPos().GetY()-m.Self.Pos.GetY()
			switch {
			case abs(dx) >= abs(dy) && abs(dx) > 2500:
				key = map[bool]string{true: "right", false: "left"}[dx > 0]
			case abs(dy) > 2500:
				key = map[bool]string{true: "up", false: "down"}[dy > 0]
			case !m.Self.AutoAttack:
				key = "f"
			}
		})
		if finished {
			return
		}
		if key != "" {
			p.Press(key)
		}
		w.Advance(time.Second)
	}
	t.Fatalf("the fight with %s did not finish", mob)
}

// A town guard on its patrol is streamed as it walks (D56): the screen follows
// it, and the overview draws no patrol path, since none is told.
func TestAGuardWalksOnTheScreenAndNoPatrolPathIsDrawn(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: scenario.Town, Accounts: townAccount()})
	p := w.NewCharacter("ann")
	makeWarrior(p, "Hale")
	walkVia(t, w, p, [2]int64{100000, 45000})
	p.Eventually("the gate guard in view", func(m *model.Model) bool { return m.NPC("guard-gate") != nil })
	var at int64
	p.Model(func(m *model.Model) { at = m.NPC("guard-gate").GetPos().GetX() })
	for range 6 {
		w.Advance(time.Second)
	}
	p.Model(func(m *model.Model) {
		if now := m.NPC("guard-gate").GetPos().GetX(); now == at {
			t.Fatalf("the guard stands at %d after six seconds", now)
		}
	})
	p.Press("m")
	if s := p.Screen(140, 45); strings.Contains(s, "mobs patrol") {
		t.Fatalf("the overview still draws patrol paths:\n%s", s)
	}
	p.Press("esc")
}
