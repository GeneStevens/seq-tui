package model_test

import (
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func mobAt(id, name string, x int64, hp int64) *pb.EntityState {
	return &pb.EntityState{Id: id, Name: name, Kind: pb.EntityKind_ENTITY_KIND_MOB, Level: 3, Hp: hp, MaxHp: 100, Pos: &pb.Pos{X: x, Y: 1000}, Con: "even"}
}

func logTexts(m *model.Model, kind string) []string {
	var out []string
	for _, l := range m.Log {
		if l.Kind == kind {
			out = append(out, l.Text)
		}
	}
	return out
}

func want(t *testing.T, m *model.Model, kind string, texts ...string) {
	t.Helper()
	got := logTexts(m, kind)
	if len(got) != len(texts) {
		t.Fatalf("%s lines: %q; want %q", kind, got, texts)
	}
	for i := range got {
		if got[i] != texts[i] {
			t.Errorf("%s line %d: %q; want %q", kind, i, got[i], texts[i])
		}
	}
}

// What mobs do in a fight is said in plain words, from the events the server
// sends (D70): a heal between mobs, a shot from range, a joining partner or
// guard, a flight and a turn back, once each.
func TestMobBehaviourIsSaidInPlainWords(t *testing.T) {
	m := model.New()
	snap(m, 1, "z", &pb.SelfState{}, nil,
		mobAt("m-stalker", "Cave Stalker", 5000, 100), mobAt("m-shaman", "Deep Shaman", 9000, 100),
		mobAt("m-slinger", "Goblin Slinger", 15000, 100), mobAt("m-chief", "Warren Chief", 20000, 100), mobAt("m-guard", "Warden", 22000, 100))

	// A shaman heals the stalker; a mob healing itself is not "healing" another.
	event(m, 100, &pb.Event{Body: &pb.Event_Combat{Combat: &pb.Combat{Kind: pb.CombatKind_COMBAT_KIND_HEAL, Attacker: "m-shaman", Target: "m-stalker", Amount: 30, Ability: "shaman-mend-kin"}}})
	want(t, m, "combat_heal", "Deep Shaman heals Cave Stalker for 30")

	// The slinger shoots from 14 m: said once, the next shot is an ordinary hit.
	for i := range 2 {
		event(m, 200+int64(i)*100, &pb.Event{Body: &pb.Event_Combat{Combat: &pb.Combat{Kind: pb.CombatKind_COMBAT_KIND_HIT, Attacker: "m-slinger", Target: "me", Amount: 4}}})
	}
	want(t, m, "mob_shoots", "Goblin Slinger shoots from range")

	// The Chief's cry, and a guard answering it; and a partner that joins a pull.
	event(m, 1000, &pb.Event{Body: &pb.Event_AbilityUsed{AbilityUsed: &pb.AbilityUsed{Actor: "m-chief", Ability: "chief-pack-cry-rally", Target: "m-chief"}}})
	event(m, 1000, &pb.Event{Body: &pb.Event_EncounterOpened{EncounterOpened: &pb.EncounterOpened{Id: "e1", Mob: "m-guard", Owner: "me"}}})
	want(t, m, "mob_joins", "Warden answers Warren Chief's Pack Cry")
	event(m, 2000, &pb.Event{Body: &pb.Event_EncounterOpened{EncounterOpened: &pb.EncounterOpened{Id: "e2", Mob: "m-slinger", Owner: "me"}}})
	event(m, 2000, &pb.Event{Body: &pb.Event_EncounterOpened{EncounterOpened: &pb.EncounterOpened{Id: "e3", Mob: "m-stalker", Owner: "me"}}})
	want(t, m, "mob_joins", "Warden answers Warren Chief's Pack Cry", "Cave Stalker, partner of Goblin Slinger, joins the fight")

	// A hurt mob that moves away from the fight flees, once; its next swing is a turn back.
	hurt := mobAt("m-cutter", "Goblin Cutter", 3000, 15)
	event(m, 3000, &pb.Event{Body: &pb.Event_EntityAppeared{EntityAppeared: &pb.EntityAppeared{Entity: hurt}}})
	event(m, 3000, &pb.Event{Body: &pb.Event_EncounterOpened{EncounterOpened: &pb.EncounterOpened{Id: "e4", Mob: "m-cutter", Owner: "me"}}})
	for i, x := range []int64{3300, 3700, 4100} {
		event(m, 3100+int64(i)*100, &pb.Event{Body: &pb.Event_EntityMoved{EntityMoved: &pb.EntityMoved{Id: "m-cutter", Pos: &pb.Pos{X: x, Y: 1000}}}})
	}
	want(t, m, "mob_flees", "Goblin Cutter flees")
	event(m, 6000, &pb.Event{Body: &pb.Event_Combat{Combat: &pb.Combat{Kind: pb.CombatKind_COMBAT_KIND_MISS, Attacker: "m-cutter", Target: "me"}}})
	event(m, 6100, &pb.Event{Body: &pb.Event_Combat{Combat: &pb.Combat{Kind: pb.CombatKind_COMBAT_KIND_MISS, Attacker: "m-cutter", Target: "me"}}})
	want(t, m, "mob_returns", "Goblin Cutter turns back to fight")
}

// The ability of a mob is named from its id, since the server describes only a character's.
func TestAMobAbilityIsNamedFromItsId(t *testing.T) {
	for id, name := range map[string]string{"hexer-cinder-bolt": "Cinder Bolt", "shaman-mend-kin": "Mend Kin", "chief-pack-cry-rally": "Pack Cry", "brew": "Brew"} {
		if got := model.MobAbilityName(id); got != name {
			t.Errorf("%s is %q; want %q", id, got, name)
		}
	}
}
