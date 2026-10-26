package app

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// Every mouse action has a key (D77): the game is fully playable without a mouse.
// Each row names what the mouse does, and the key whose Action is the same.
func TestEveryMouseActionHasAKey(t *testing.T) {
	m := bindingModel(t)
	for _, tc := range []struct {
		mouse, key string
		want       Kind
	}{
		{"Option-click the ground: walk", "up", Move},
		{"Option-click a mob or the last spot: walk to it", "W", WalkTo},
		{"click a mob: select it", "tab", Target},
		{"click it again: attack", "f", Attack},
		{"click a person again: talk", "g", Talk},
		{"click loot: loot all", "l", Loot},
		{"scroll up: zoom in", "+", ZoomMap},
		{"scroll down: zoom out", "-", ZoomMap},
		{"hold Alt: label everything", "ctrl+a", ToggleLabels},
		{"click the large map: walk (arrows and Enter)", "m", OpenOverview},
	} {
		a := Plan(tc.key, m)
		switch {
		case zero(a):
			t.Errorf("%s: key %q does nothing", tc.mouse, tc.key)
		case a.Kind != None && a.Kind != tc.want:
			t.Errorf("%s: key %q is %+v, want kind %d", tc.mouse, tc.key, a, tc.want)
		}
		// A key that needs something in view (a target, a person, a drop) says so in a
		// note when there is nothing: it is bound all the same.
	}
	// The keys are in the help.
	listed := screen.HelpBindingKeys()
	for _, k := range []string{"+", "-", "ctrl+a", "tab", "f", "g", "l", "m", "esc"} {
		found := false
		for _, l := range listed {
			found = found || l == k
		}
		if !found {
			t.Errorf("key %q is not in the help's table", k)
		}
	}
}

// clickWorld is a character at the origin with a far mob, a near one, two
// people (one in reach) and two drops (one in reach).
func clickWorld() (*App, *model.Model) {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "Z", Seq: 1, GameTimeMs: 1000,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Pos: &pb.Pos{}, Hp: 50, MaxHp: 50}, Stats: &pb.Stats{MeleeRangeMm: 2500}},
		Entities: []*pb.EntityState{
			{Id: "far", Name: "Far Rat", Kind: pb.EntityKind_ENTITY_KIND_MOB, Pos: &pb.Pos{X: 10000}, Hp: 5, MaxHp: 5},
			{Id: "near", Name: "Near Rat", Kind: pb.EntityKind_ENTITY_KIND_MOB, Pos: &pb.Pos{X: 2000}, Hp: 5, MaxHp: 5},
		},
		Npcs: []*pb.NpcInfo{
			{Id: "here", Name: "Hilda", Role: "vendor", ReachMm: 3000, Pos: &pb.Pos{X: 1000}},
			{Id: "away", Name: "Gorran", Role: "guard", ReachMm: 3000, Pos: &pb.Pos{X: 20000}},
		},
		Drops: []*pb.LootDropState{
			{Id: "d-here", Item: "cap", CanLoot: true, Pos: &pb.Pos{X: 3000}},
			{Id: "d-away", Item: "cap", CanLoot: true, Pos: &pb.Pos{X: 20000}},
		},
	}}})
	return New(client.NewSession(client.Config{Model: m})), m
}

func pickOf(m *model.Model, kind screen.PickKind, id string) screen.Pick {
	p := thingPos(m, kind, id)
	return screen.Pick{Kind: kind, ID: id, Pos: p, Name: id}
}

// What a click means (D109): a plain click never moves; Option-click is the only
// one that does. Each row is a fresh program, clicked in order.
func TestWhatAClickMeans(t *testing.T) {
	ground := screen.Pick{Kind: screen.PickGround, Pos: &pb.Pos{X: -5000}}
	type click struct {
		pick screen.Pick
		alt  bool
	}
	pick := func(kind screen.PickKind, id string) screen.Pick { _, m := clickWorld(); return pickOf(m, kind, id) }
	for _, tc := range []struct {
		name   string
		clicks []click
		want   Kind   // the last click's action
		note   string // what its note holds
	}{
		{"plain ground: a hint, no move", []click{{ground, false}}, None, "Option-click to walk here"},
		{"plain ground twice: the hint once", []click{{ground, false}, {ground, false}}, None, ""},
		{"Option ground: walk", []click{{ground, true}}, Move, ""},
		{"plain mob: select and target", []click{{pick(screen.PickMob, "far"), false}}, Target, ""},
		{"plain mob twice: attack, no move", []click{{pick(screen.PickMob, "far"), false}, {pick(screen.PickMob, "far"), false}}, Attack, ""},
		{"Option mob: close in", []click{{pick(screen.PickMob, "far"), true}}, Approach, ""},
		{"plain person in reach: card, then talk", []click{{pick(screen.PickNPC, "here"), false}, {pick(screen.PickNPC, "here"), false}}, Talk, ""},
		{"plain person out of reach: how far, no walk", []click{{pick(screen.PickNPC, "away"), false}}, None, "Too far (20 m): Option-click to walk there"},
		{"plain person out of reach, again: still no walk", []click{{pick(screen.PickNPC, "away"), false}, {pick(screen.PickNPC, "away"), false}}, None, "Too far (20 m)"},
		{"Option person out of reach: walk up", []click{{pick(screen.PickNPC, "away"), true}}, Approach, ""},
		{"Option person in reach: talk", []click{{pick(screen.PickNPC, "here"), true}}, Talk, ""},
		{"plain loot in reach: loot", []click{{pick(screen.PickDrop, "d-here"), false}}, Loot, ""},
		{"plain loot out of reach: how far, no walk", []click{{pick(screen.PickDrop, "d-away"), false}}, None, "Too far (20 m): Option-click to walk there"},
		{"Option loot out of reach: walk up", []click{{pick(screen.PickDrop, "d-away"), true}}, Approach, ""},
	} {
		a, m := clickWorld()
		var got Action
		for _, c := range tc.clicks {
			got = a.pickAction(m, c.pick, c.alt)
		}
		if got.Kind != tc.want || !strings.Contains(got.Note, tc.note) {
			t.Errorf("%s: got %+v, want kind %d with note %q", tc.name, got, tc.want, tc.note)
		}
	}
}

// Attacking a target already attacked sends nothing, and says how far it is when
// it is out of reach; the key W walks to what was clicked.
func TestNoDuplicateAttackAndTheWalkKey(t *testing.T) {
	a, m := clickWorld()
	m.Self.Target, m.Self.AutoAttack = "far", true
	p := pickOf(m, screen.PickMob, "far")
	a.pickAction(m, p, false)
	got := a.pickAction(m, p, false)
	if got.Kind != None || !strings.Contains(got.Note, "Too far: 10 m. Option-click the target to close in") {
		t.Errorf("a second click while attacking from afar: %+v", got)
	}
	m.Self.Pos = &pb.Pos{X: 9000}
	if got := a.pickAction(m, p, false); got.Kind != None || !strings.Contains(got.Note, "Already attacking") {
		t.Errorf("a second click while attacking in reach: %+v", got)
	}
	// W: the clicked thing, else the target, else the ground clicked, else a note.
	if w := a.walkToAction(m); w.Kind != Approach || w.ID != "far" {
		t.Errorf("W after a click on the mob: %+v", w)
	}
	b, m2 := clickWorld()
	if w := b.walkToAction(m2); w.Kind != None || w.Note == "" {
		t.Errorf("W with nothing clicked: %+v", w)
	}
	b.pickAction(m2, screen.Pick{Kind: screen.PickGround, Pos: &pb.Pos{X: 7000}}, false)
	if w := b.walkToAction(m2); w.Kind != Move || w.X != 7000 {
		t.Errorf("W after a click on the ground: %+v", w)
	}
}

// The approach closes in short of the thing, follows it, and acts in reach.
func TestApproachClosesInAndActs(t *testing.T) {
	a, m := clickWorld()
	ap := &approach{id: "far", kind: screen.PickMob, name: "Far Rat", since: a.now()}
	acts, done, _ := a.approachSteps(m, ap)
	if done || len(acts) != 1 || acts[0].Kind != Move || acts[0].X <= 0 || acts[0].X >= 10000 || !acts[0].Quiet {
		t.Fatalf("first step %+v done %v: want a quiet Move short of the mob", acts, done)
	}
	if acts, _, _ := a.approachSteps(m, ap); len(acts) != 0 {
		t.Errorf("the same place sent again at once: %+v", acts)
	}
	m.Entities["far"].Pos = &pb.Pos{X: 14000} // it moved
	if acts, _, _ := a.approachSteps(m, ap); len(acts) != 1 || acts[0].Kind != Move {
		t.Errorf("a mob that moved is not followed: %+v", acts)
	}
	m.Entities["far"].Pos = &pb.Pos{X: 1000}
	acts, done, _ = a.approachSteps(m, ap)
	if !done || len(acts) != 2 || acts[0].Kind != Target || acts[1].Kind != Attack || !acts[1].On {
		t.Errorf("in reach: %+v done %v, want Target then Attack", acts, done)
	}
	m.Self.Target, m.Self.AutoAttack = "far", true
	if acts, done, _ = a.approachSteps(m, ap); !done || len(acts) != 0 {
		t.Errorf("already attacking: %+v, want nothing sent", acts)
	}
	m.Entities["far"].Dead = true
	if _, done, note := a.approachSteps(m, ap); !done || note == "" {
		t.Errorf("a dead mob ends the approach: done %v note %q", done, note)
	}
}
