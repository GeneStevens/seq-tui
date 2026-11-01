package scenario_test

import (
	"context"
	"math"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	"github.com/GeneStevens/seq/core"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
	"github.com/GeneStevens/seq/testserver"
)

// The scenarios of M9's polish (SEQ-057, D59, TUI-019): the second character
// that is directed is online and heard, and the O key logs out without a hang.

// A duo is two characters of one account (ann1, ann2) beside another account's
// character (cy1), all in the Testgrounds.
func duoAndAFriend(t *testing.T) (*scenario.World, *scenario.Player, *scenario.Player) {
	t.Helper()
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: []testserver.Account{
		{Name: "ann", Characters: []string{"ann1", "ann2"}}, {Name: "cy", Characters: []string{"cy1"}}}})
	w.Place("ann", "ann1", 100000, 89000)
	w.Place("ann", "ann2", 100000, 89500)
	w.Place("cy", "cy1", 100000, 90000)
	ann := w.LoginWithDuo("ann", "ann1", "ann2")
	cy := w.Login("cy", "cy1")
	return w, ann, cy
}

func whoKnows(p *scenario.Player, name string) bool {
	command(p, "/who")
	defer p.Press("esc")
	known := false
	p.Model(func(m *model.Model) { _, known = m.Social.Known[model.Key(name)] })
	return known
}

// The directed second character is online to the others (who, tell) and what is
// said to it, near or by name, is on the screen of the one who directs it.
func TestTheDirectedSecondCharacterIsOnlineAndHeard(t *testing.T) {
	w, ann, cy := duoAndAFriend(t)
	ann.Press("p")
	w.Advance(time.Second)
	ann.Press("x")
	ann.Press("up") // a command for ann2: its live input
	w.Advance(time.Second)

	if !whoKnows(cy, "ann2") {
		t.Fatal("ann2, directed, is not in cy's who list")
	}
	saysLine(cy, "hello both")
	command(cy, "/tell ann2 psst, a word")
	w.Settle()
	notice(t, cy, "", false)
	ann.Duo(func(m *model.Model) {
		if !hasChat(m, "[say] cy1: hello both") || !hasChat(m, "[tell] cy1: psst, a word") {
			t.Fatalf("ann2 did not hear: %v", m.Social.Chat)
		}
	})
	if s := ann.Screen(80, 24); !strings.Contains(s, "[tell] cy1: psst, a word") {
		t.Fatalf("the directed character's chat is not on the screen:\n%s", s)
	}
	ann.Golden("directed_online")
}

func hasChat(m *model.Model, want string) bool {
	for _, l := range m.Social.Chat {
		if l.String() == want {
			return true
		}
	}
	return false
}

// O logs the character out: the program ends, the others see it gone, and the
// server does not stall on the way (SEQ-057).
func TestLogoutWithTheOKey(t *testing.T) {
	w, ann, cy := duoAndAFriend(t)
	if !whoKnows(cy, "ann1") {
		t.Fatal("ann1 is not online before the logout")
	}
	ann.LogOut()
	start := time.Now()
	w.Advance(time.Second)
	w.Settle()
	if d := time.Since(start); d > time.Second {
		t.Fatalf("the server took %v to settle after the logout", d)
	}
	command(cy, "/tell ann1 are you there")
	notice(t, cy, "not online", true)
}

// The Hollow Warren's camps, in the order the two walk them: the camp's id (its
// mob is "mob-<id>-1") and where its mob stands, in millimetres. Every camp but
// three is off the long gallery (y 70000) up a shaft, so the way to it is the
// gallery to the shaft's x, then along the shaft.
var warrenCamps = []struct {
	id   string
	x, y int64
}{
	{"diggers-c", 62000, 70000}, {"diggers-a", 30000, 105000}, {"diggers-b", 28000, 40000},
	{"scouts-a", 72000, 112000}, {"slingers-a", 72000, 124000}, {"scouts-b", 72000, 28000},
	{"slingers-b", 96000, 50000},
	{"cutthroats-a", 142000, 105000}, {"cutthroats-b", 142000, 35000},
	{"shamans-a", 172000, 90000}, {"shamans-b", 180000, 45000},
	{"brutes-a", 204000, 108000}, {"brutes-c", 224000, 70000}, {"brutes-b", 214000, 34000},
	{"stalkers-a", 262000, 102000}, {"stalkers-b", 262000, 38000},
	{"deep-shamans-a", 288000, 85000}, {"deep-shamans-b", 288000, 32000},
	{"wardens-a", 312000, 104000}, {"wardens-b", 312000, 36000},
	{"chief", 340000, 70000},
}

// travel sends each player toward the point in one move and gives the game time
// to get there, until both are within a metre and a half.
// The keys walk three metres a press; the delve is 340 m long.
func travel(t *testing.T, w *scenario.World, to [2]int64, ps ...*scenario.Player) {
	t.Helper()
	for range 30 {
		var far int64
		for i, p := range ps {
			p.Model(func(m *model.Model) {
				dx, dy := m.Self.Pos.GetX()-to[0], m.Self.Pos.GetY()-to[1]-int64(i)*1000
				far = max(far, int64(math.Hypot(float64(dx), float64(dy))))
			})
		}
		if far <= 1500 {
			return
		}
		// The order is sent again each time round: a mob that engages a walker
		// stops its walk (D66), and a later order is obeyed.
		for i, p := range ps {
			if _, err := p.Sess.Move(context.Background(), to[0], to[1]+int64(i)*1000); err != nil {
				t.Fatalf("%s: move: %v", p.Character, err)
			}
		}
		w.Settle() // the server has the moves before the clock moves, not by luck
		w.Srv.Advance(time.Duration(far/delverSpeed+1) * time.Second)
		w.Settle()
	}
	for _, p := range ps {
		p.Model(func(m *model.Model) {
			t.Logf("%s at %v hp %d/%d dead %v", p.Character, m.Self.Pos, m.Self.HP, m.Self.MaxHP, m.Self.Dead)
		})
	}
	t.Fatalf("the players did not reach %v", to)
}

// delverSpeed is how fast the two walk the delve, in millimetres a second (a
// player's own is a third of it), so that the walks do not fill the clock with
// moves: the cost of the scenario is the server's events.
const delverSpeed = 12000

// hardy gives a character the hit points and armour a delver has gathered by
// level 12 (the scenario does not play the gathering).
func hardy(cs *core.CharacterSave) {
	cs.Stats.Stamina += 60
	cs.Stats.Armor += 250
	cs.Stats.SpeedMMPerSec = delverSpeed
	cs.Stats.WeaponMin += 150
	cs.Stats.WeaponMax += 170
	cs.HP = core.MaxHP(cs.Stats)
}

// killed says the log tells of the mob's death: a mob that is out of view is not
// dead.
func killed(m *model.Model, mob string) bool {
	return slices.ContainsFunc(m.Log, func(l model.LogEntry) bool { return l.Kind == "combat_death" && l.Target == mob })
}

// campMobs are the living mobs of a camp in view, the nearest first. A camp of
// one is "mob-ID-1"; a camp of two (D73) is "mob-ID-1-1" and "mob-ID-2-1".
func campMobs(m *model.Model, id string) []string {
	var out []string
	for mid, e := range m.Entities {
		if strings.HasPrefix(mid, "mob-"+id+"-") && !e.GetDead() {
			out = append(out, mid)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := m.Entities[out[i]].GetPos(), m.Entities[out[j]].GetPos()
		da := math.Hypot(float64(a.GetX()-m.Self.Pos.GetX()), float64(a.GetY()-m.Self.Pos.GetY()))
		db := math.Hypot(float64(b.GetX()-m.Self.Pos.GetX()), float64(b.GetY()-m.Self.Pos.GetY()))
		if da != db {
			return da < db
		}
		return out[i] < out[j]
	})
	return out
}

// killCamp fights the camp's mobs with both until they are down, a few seconds of
// game time at a time, and takes what they dropped.
func killCamp(t *testing.T, w *scenario.World, id string, ann, bob *scenario.Player) {
	t.Helper()
	seen, dead, seenN := false, false, 0
	step := 8 * time.Second
	for i := 0; i < 80 && !dead; i++ {
		for _, p := range []*scenario.Player{ann, bob} {
			key := ""
			var closeIn *[2]int64
			p.Model(func(m *model.Model) {
				live := campMobs(m, id)
				step = time.Second // a mob nobody is fighting (a patrol, a goblin that fled) is chased in short steps
				if len(m.Encounters) > 0 {
					step = 8 * time.Second
				}
				inView, felled := map[string]bool{}, map[string]bool{}
				for mid := range m.Entities {
					if strings.HasPrefix(mid, "mob-"+id+"-") {
						inView[mid] = true
					}
				}
				for _, l := range m.Log {
					if l.Kind == "combat_death" && strings.HasPrefix(l.Target, "mob-"+id+"-") {
						felled[l.Target] = true
					}
				}
				seen = seen || len(inView) > 0 || len(felled) > 0
				seenN = max(seenN, len(inView), len(felled))
				dead = dead || (seen && len(live) == 0 && len(felled) >= seenN)
				if len(live) == 0 {
					return
				}
				mob := live[0]
				e := m.Entities[mob]
				switch {
				case m.Self.Target != mob && !(m.Self.Target != "" && m.Entities[m.Self.Target] != nil && !m.Entities[m.Self.Target].GetDead() && strings.HasPrefix(m.Self.Target, "mob-"+id+"-")):
					key = "tab"
				case !m.Self.AutoAttack:
					key = "f"
				}
				if i > 0 { // a goblin that fled is chased home
					// out of reach of its mob, on the side we came from: a mob that keeps its
					// distance (a slinger) backs away from whoever is nearer than it likes
					y := e.GetPos().GetY() - 1500
					if m.Self.Pos.GetY() < e.GetPos().GetY() {
						y = e.GetPos().GetY() + 1500
					}
					closeIn = &[2]int64{e.GetPos().GetX(), y}
				}
			})
			if closeIn != nil {
				if _, err := p.Sess.Move(context.Background(), closeIn[0], closeIn[1]); err != nil {
					t.Fatalf("%s: move: %v", p.Character, err)
				}
				p.Sync()
			}
			if key != "" {
				p.Press(key)
			}
		}
		if !dead {
			w.Srv.Advance(step)
			w.Settle()
		}
	}
	if !dead {
		ann.Model(func(m *model.Model) {
			for _, l := range m.Log {
				if !l.Quiet {
					t.Logf("%d %s", l.GameTimeMs, l.Text)
				}
			}
		})
		t.Fatalf("camp %s did not fall (seen %v)", id, seen)
	}
	// What it dropped appears a tick after the kill; hale takes it all.
	w.Srv.Advance(time.Second)
	w.Settle()
	// Hale takes what falls; when his bags are full (a camp of two drops twice as
	// much) bram does, and what neither can carry crumbles away in two minutes.
	looter, full := ann, map[*scenario.Player]bool{}
	for range 24 {
		left := 0
		var to *pb.Pos
		looter.Model(func(m *model.Model) {
			left = len(m.Drops)
			for _, did := range m.DropIDs() { // a pair falls a little apart: stand by the drop
				if d := m.Drops[did].GetPos(); math.Hypot(float64(d.GetX()-m.Self.Pos.GetX()), float64(d.GetY()-m.Self.Pos.GetY())) > 3000 {
					to = d
					break
				}
			}
		})
		if left == 0 || (full[ann] && full[bob]) {
			break
		}
		if to != nil {
			if _, err := looter.Sess.Move(context.Background(), to.GetX(), to.GetY()); err != nil {
				t.Fatalf("move to the drop: %v", err)
			}
			looter.Sync()
			w.Srv.Advance(4 * time.Second)
			w.Settle()
		}
		looter.Press("l")
		w.Srv.Advance(time.Second)
		w.Settle()
		if n, bad := looter.Notice(); bad && strings.Contains(n, "bags are full") {
			full[looter] = true
			looter = bob
			if full[bob] {
				looter = ann
			}
		}
	}
}

// walkCamp takes both to the camp: along the gallery to the shaft that leads to
// it, then along the shaft to six metres short of the mob.
func walkCamp(t *testing.T, w *scenario.World, from *[2]int64, camp struct {
	id   string
	x, y int64
}, ann, bob *scenario.Player) {
	t.Helper()
	if from[1] != 70000 {
		travel(t, w, [2]int64{from[0], 70000}, ann, bob) // back to the gallery
	}
	stop := [2]int64{camp.x - 6000, 70000}
	if camp.y != 70000 {
		travel(t, w, [2]int64{camp.x, 70000}, ann, bob)
		stop = [2]int64{camp.x, camp.y + 6000*sign(70000-camp.y)}
	}
	travel(t, w, stop, ann, bob)
	*from = stop
}

// Two accounts clear the whole Hollow Warren at Venture: they party up, one opens
// the delve and both go in, and they walk it camp by camp, the warrior holding
// each mob and the cleric beside him, until the last, the Warren Chief, is down
// and the loot taken; the server then ends the instance as cleared and hands
// them back. (The keys walk three metres a press; the walks of the delve are
// moves to the point, and the fights run game time fast.) It runs in three steps,
// each within the limit.
func TestTwoAccountsClearTheWholeWarren(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{scenario.Warren}, Zone: shelved, Accounts: []testserver.Account{
		{Name: "ann", Home: scenario.Warren, Classed: map[string]string{"hale": "warrior"}},
		{Name: "bob", Home: scenario.Warren, Classed: map[string]string{"bram": "cleric"}}}})
	w.Place("ann", "hale", 11000, 70000)
	w.Place("bob", "bram", 11000, 69000)
	w.Srv.Give("hale", levelUp(12, hardy))
	w.Srv.Give("bram", levelUpAs("cleric", 12, hardy))
	ann := w.Login("ann", "hale")
	bob := w.Login("bob", "bram")
	from := [2]int64{11000, 70000}
	var space string
	camps := warrenCamps

	w.Step("into the delve and the diggings", func() {
		command(ann, "/invite bram")
		w.Settle()
		bob.Press("y")
		w.Settle()
		for _, p := range []*scenario.Player{ann, bob} {
			p.Eventually("the party of two", func(m *model.Model) bool { return len(m.PartyMembers()) == 2 })
		}
		w.Advance(time.Second)
		openPreset(t, bob, "venture-warren", true)
		w.Advance(time.Second)
		bob.Press("e")
		bob.Eventually("bram in the delve", inside)
		w.Advance(500 * time.Millisecond)
		ann.Press("e")
		ann.Eventually("hale in the delve", inside)
		ann.Model(func(m *model.Model) { space = m.Space })
		w.Advance(time.Second)
		ann.Golden("warren_full_entered")
		for _, c := range camps[:7] {
			walkCamp(t, w, &from, c, ann, bob)
			killCamp(t, w, c.id, ann, bob)
		}
		w.Advance(time.Second)
		notYetCleared(t, ann, bob)
		ann.Golden("warren_full_diggings")
	})
	w.Step("the den", func() {
		for _, c := range camps[7:14] {
			walkCamp(t, w, &from, c, ann, bob)
			killCamp(t, w, c.id, ann, bob)
		}
		w.Advance(time.Second)
		notYetCleared(t, ann, bob)
		ann.Golden("warren_full_den")
	})
	w.Step("the deep and the Warren Chief", func() {
		for _, c := range camps[14:20] {
			walkCamp(t, w, &from, c, ann, bob)
			killCamp(t, w, c.id, ann, bob)
		}
		notYetCleared(t, ann, bob)
		walkCamp(t, w, &from, camps[20], ann, bob)
		ann.Golden("warren_full_chief")
		killCamp(t, w, "chief", ann, bob)
		// The delve is cleared: the server ends it and hands both back.
		for _, p := range []*scenario.Player{ann, bob} {
			w.Advance(time.Second)
			p.Eventually("the hand-back", back)
			p.Model(func(m *model.Model) {
				if m.Ended.Reason != "cleared" || m.Instance != nil || m.Space == space {
					t.Fatalf("%s: ended %+v, instance %v, in %q", p.Character, m.Ended, m.Instance, m.Space)
				}
			})
		}
		overviewGolden(ann, "warren_full_cleared")
	})
}

func sign(n int64) int64 {
	if n < 0 {
		return -1
	}
	return 1
}

// notYetCleared fails the scenario if the delve has ended: the server ends it
// when the last mob is down and the loot is taken, not before.
func notYetCleared(t *testing.T, ps ...*scenario.Player) {
	t.Helper()
	for _, p := range ps {
		p.Model(func(m *model.Model) {
			if m.Ended != nil || !m.InInstance() {
				t.Fatalf("%s: the delve ended early: %+v", p.Character, m.Ended)
			}
		})
	}
}

// overviewGolden takes the golden screen of the whole-zone map, and closes it.
// The map shows where the two stand and what the delve is. It is for the golden
// after the hand-back: the normal screen's event log there tells whether bram's
// arrival in the zone has been heard, and the server hands the two back on its
// own, outside the clock. (The golden screens inside the delve are the normal
// ones: the run is the same every time, TUI-021.)
func overviewGolden(p *scenario.Player, name string) {
	p.Press("m")
	p.Golden(name)
	p.Press("esc")
}
