package model_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func at(model *model.Model, ms int64, body any) {
	e := &pb.Event{Seq: model.Last() + 1, GameTimeMs: ms}
	switch b := body.(type) {
	case *pb.Event_CastStarted:
		e.Body = b
	case *pb.Event_CastCancelled:
		e.Body = b
	case *pb.Event_CastInterrupted:
		e.Body = b
	case *pb.Event_AbilityUsed:
		e.Body = b
	case *pb.Event_ResourceChanged:
		e.Body = b
	case *pb.Event_EffectApplied:
		e.Body = b
	case *pb.Event_EffectExpired:
		e.Body = b
	case *pb.Event_SlotSet:
		e.Body = b
	case *pb.Event_Reward:
		e.Body = b
	}
	model.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: e}})
}

func snapshot(level int64, abilities ...string) *pb.ServerMessage {
	infos := []*pb.AbilityInfo{}
	slots := []*pb.AbilitySlot{}
	for i, id := range abilities {
		infos = append(infos, &pb.AbilityInfo{Id: id, Name: strings.ToUpper(id[:1]) + id[1:], CastMs: 1000, CooldownMs: 5000, ManaCost: 10, RangeMm: 20000, Target: "enemy"})
		slots = append(slots, &pb.AbilitySlot{Slot: uint32(i + 1), Ability: id})
	}
	return &pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{
		Space: "z", GameTimeMs: 1000, Seq: 1,
		Self: &pb.SelfState{
			Entity: &pb.EntityState{Id: "me", Name: "Me", Level: level, Hp: 50, MaxHp: 50, Pos: &pb.Pos{X: 0, Y: 0}},
			Class:  "wizard", Mana: 30, MaxMana: 30, GcdMs: 1500, Abilities: infos, Slots: slots,
			XpLevelStart: 0, XpNextLevel: 40,
			Effects: []*pb.EffectState{{Ability: "ward", Kind: "buff", Stat: "armor", TicksLeft: 2}, {Ability: "x", Kind: "stun", ExpiresAtMs: 4000}},
		},
		Entities: []*pb.EntityState{{Id: "mob", Kind: pb.EntityKind_ENTITY_KIND_MOB, Name: "Mob", Pos: &pb.Pos{X: 10000}, Hp: 20, MaxHp: 20}},
	}}}
}

func TestSnapshotCarriesTheKit(t *testing.T) {
	m := model.New()
	m.Apply(snapshot(1, "bolt", "dart"))
	s := m.Self
	if !m.HasKit() || s.Class != "wizard" || s.Mana != 30 || s.MaxMana != 30 || s.XPNext != 40 || m.SlotOf("dart") != 2 || m.SlotOf("nope") != 0 || m.AbilityName("bolt") != "Bolt" {
		t.Fatalf("self %+v", s)
	}
	if len(s.Effects) != 2 || !s.Effects[0].Approx || s.Effects[0].EndsAtMs != 1000+2*model.RulesTickMs || s.Effects[1].Approx || s.Effects[1].EndsAtMs != 4000 {
		t.Fatalf("effects %+v", s.Effects)
	}
	if g, _ := m.Slot(3, 1000); g != model.SlotEmpty {
		t.Errorf("an empty slot is %v", g)
	}
	// A character with no abilities has no kit and no bar.
	if e := model.New(); e.HasKit() {
		t.Error("an empty model has a kit")
	}
}

func TestSlotGaugeSaysWhyAnAbilityCannotBeUsed(t *testing.T) {
	m := model.New()
	m.Apply(snapshot(1, "bolt"))
	if g, _ := m.Slot(1, 1000); g != model.SlotNoTarget {
		t.Errorf("no target: %v", g)
	}
	m.Self.Target = "mob"
	if g, _ := m.Slot(1, 1000); g != model.SlotReady {
		t.Errorf("ready: %v", g)
	}
	m.Entities["mob"].Pos = &pb.Pos{X: 21000}
	if g, _ := m.Slot(1, 1000); g != model.SlotRange {
		t.Errorf("range: %v", g)
	}
	m.Entities["mob"].Pos = &pb.Pos{X: 5000}
	m.Self.Mana = 9
	if g, _ := m.Slot(1, 1000); g != model.SlotNoMana {
		t.Errorf("mana: %v", g)
	}
	m.Self.Mana = 30
	m.Self.GCDUntilMs = 2500
	if g, left := m.Slot(1, 1000); g != model.SlotGCD || left != 1500 {
		t.Errorf("gcd: %v %d", g, left)
	}
	m.Self.ReadyAtMs["bolt"] = 9000
	if g, left := m.Slot(1, 1000); g != model.SlotCooldown || left != 8000 {
		t.Errorf("cooldown: %v %d", g, left)
	}
	m.Self.ReadyAtMs["bolt"] = 0
	m.Self.GCDUntilMs = 0
	m.Self.Cast = &model.Cast{Ability: "bolt", EndsAtMs: 1500}
	if g, left := m.Slot(1, 1000); g != model.SlotCasting || left != 500 {
		t.Errorf("casting: %v %d", g, left)
	}
}

func TestCastLifecycleMirrorsTheEvents(t *testing.T) {
	m := model.New()
	m.Apply(snapshot(1, "bolt", "dart"))
	at(m, 2000, &pb.Event_CastStarted{CastStarted: &pb.CastStarted{Actor: "me", Target: "mob", Ability: "bolt", CastMs: 1000}})
	if c := m.Self.Cast; c == nil || c.Ability != "bolt" || c.EndsAtMs != 3000 || c.Slot != 1 || m.Self.GCDUntilMs != 3500 {
		t.Fatalf("cast %+v gcd %d", m.Self.Cast, m.Self.GCDUntilMs)
	}
	// Moving cancels it: nothing spent, nothing on cooldown, and the line says why.
	at(m, 2500, &pb.Event_CastCancelled{CastCancelled: &pb.CastEnded{Actor: "me", Ability: "bolt", Reason: "moved"}})
	if e := m.Self.LastCastEnd; m.Self.Cast != nil || e == nil || e.Broken != "cancelled" || e.Reason != "moved" || m.Self.Mana != 30 || m.Self.ReadyAtMs["bolt"] != 0 {
		t.Fatalf("after moving: %+v %+v", m.Self.Cast, e)
	}
	if last := m.Log[len(m.Log)-1]; last.Kind != "cast_cancelled" || last.Text != "Me's Bolt was cancelled: you moved" {
		t.Fatalf("log %+v", last)
	}
	// A new cast clears the note; damage interrupts it; a cast that lands costs mana and starts the cooldown.
	at(m, 3000, &pb.Event_CastStarted{CastStarted: &pb.CastStarted{Actor: "me", Ability: "dart", CastMs: 1000}})
	if m.Self.LastCastEnd != nil {
		t.Error("the old note stayed")
	}
	at(m, 3400, &pb.Event_CastInterrupted{CastInterrupted: &pb.CastEnded{Actor: "me", Ability: "dart", Reason: "damage"}})
	if e := m.Self.LastCastEnd; m.Self.Cast != nil || e == nil || e.Broken != "interrupted" || e.Reason != "damage" {
		t.Fatalf("after damage: %+v", e)
	}
	if last := m.Log[len(m.Log)-1]; last.Text != "Me's Dart was interrupted: damage broke it" {
		t.Fatalf("log %q", last.Text)
	}
	at(m, 4000, &pb.Event_CastStarted{CastStarted: &pb.CastStarted{Actor: "me", Ability: "bolt", CastMs: 1000}})
	at(m, 5000, &pb.Event_ResourceChanged{ResourceChanged: &pb.ResourceChanged{Actor: "me", Resource: "mana", Delta: -10, Reason: "cost"}})
	at(m, 5000, &pb.Event_AbilityUsed{AbilityUsed: &pb.AbilityUsed{Actor: "me", Target: "mob", Ability: "bolt", Slot: 1}})
	if m.Self.Cast != nil || m.Self.Mana != 20 || m.Self.ReadyAtMs["bolt"] != 10000 || m.Self.LastCastEnd != nil {
		t.Fatalf("after landing: cast %+v mana %d ready %d", m.Self.Cast, m.Self.Mana, m.Self.ReadyAtMs["bolt"])
	}
	// Another character's casts are not ours.
	at(m, 5100, &pb.Event_CastStarted{CastStarted: &pb.CastStarted{Actor: "mob", Ability: "bolt", CastMs: 1000}})
	if m.Self.Cast != nil {
		t.Error("a mob's cast became ours")
	}
	// Mana regenerates by rules tick.
	at(m, 6000, &pb.Event_ResourceChanged{ResourceChanged: &pb.ResourceChanged{Actor: "me", Resource: "mana", Delta: 7, Reason: "regen"}})
	if m.Self.Mana != 27 {
		t.Errorf("mana %d", m.Self.Mana)
	}
}

func TestEffectsOnYouAndOnYourTarget(t *testing.T) {
	m := model.New()
	m.Apply(snapshot(1, "bolt"))
	at(m, 2000, &pb.Event_EffectApplied{EffectApplied: &pb.EffectApplied{Source: "me", Target: "mob", Ability: "root", Kind: "root", DurationMs: 8000}})
	at(m, 2000, &pb.Event_EffectApplied{EffectApplied: &pb.EffectApplied{Source: "mob", Target: "me", Ability: "bite", Kind: "snare", DurationMs: 3000}})
	if fx := m.EffectsOn("mob"); len(fx) != 1 || fx[0].Kind != "root" || fx[0].EndsAtMs != 10000 {
		t.Fatalf("on the mob: %+v", fx)
	}
	if fx := m.EffectsOn("me"); len(fx) != 3 || fx[2].Kind != "snare" || fx[2].EndsAtMs != 5000 {
		t.Fatalf("on me: %+v", fx)
	}
	// Applied again, it replaces itself rather than stacking in the list.
	at(m, 3000, &pb.Event_EffectApplied{EffectApplied: &pb.EffectApplied{Source: "me", Target: "mob", Ability: "root", Kind: "root", DurationMs: 8000}})
	if fx := m.EffectsOn("mob"); len(fx) != 1 || fx[0].EndsAtMs != 11000 {
		t.Fatalf("refreshed: %+v", fx)
	}
	at(m, 4000, &pb.Event_EffectExpired{EffectExpired: &pb.EffectExpired{Source: "me", Target: "mob", Ability: "root", Kind: "root"}})
	at(m, 4000, &pb.Event_EffectExpired{EffectExpired: &pb.EffectExpired{Source: "mob", Target: "me", Ability: "bite", Kind: "snare"}})
	if len(m.EffectsOn("mob")) != 0 || len(m.EffectsOn("me")) != 2 {
		t.Fatalf("after expiring: %+v %+v", m.EffectsOn("mob"), m.EffectsOn("me"))
	}
	// A death clears what lay on the dead.
	at(m, 4100, &pb.Event_EffectApplied{EffectApplied: &pb.EffectApplied{Source: "me", Target: "mob", Ability: "root", Kind: "root", DurationMs: 8000}})
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: m.Last() + 1, Body: &pb.Event_Combat{Combat: &pb.Combat{Kind: pb.CombatKind_COMBAT_KIND_DEATH, Attacker: "me", Target: "mob"}}}}})
	if len(m.EffectsOn("mob")) != 0 {
		t.Errorf("effects on a dead mob: %+v", m.EffectsOn("mob"))
	}
}

func TestSlotSetUpdatesTheBar(t *testing.T) {
	m := model.New()
	m.Apply(snapshot(1, "bolt", "dart"))
	at(m, 2000, &pb.Event_SlotSet{SlotSet: &pb.SlotSet{Actor: "me", Slot: 1, Ability: "dart"}})
	at(m, 2000, &pb.Event_SlotSet{SlotSet: &pb.SlotSet{Actor: "me", Slot: 2}})
	if m.Self.Slots[0].Ability != "dart" || m.Self.Slots[1].Ability != "" || m.SlotOf("bolt") != 0 {
		t.Fatalf("slots %+v", m.Self.Slots)
	}
	if last := m.Log[len(m.Log)-1]; last.Text != "slot 2 cleared" {
		t.Errorf("log %q", last.Text)
	}
}

// A reward that raises the level is followed by a fresh snapshot with the new
// ability; the log says what was learned and where it went.
func TestLevelUpNamesTheNewAbility(t *testing.T) {
	m := model.New()
	m.Apply(snapshot(1, "bolt"))
	at(m, 2000, &pb.Event_Reward{Reward: &pb.Reward{Recipient: "me", Source: "mob", Kind: "xp", Amount: 60, Level: 2}})
	if m.Self.Level != 2 || m.Self.XP != 60 {
		t.Fatalf("level %d xp %d", m.Self.Level, m.Self.XP)
	}
	if last := m.Log[len(m.Log)-1]; last.Text != "Me received 60 xp from Mob and reached level 2" {
		t.Fatalf("reward line %q", last.Text)
	}
	m.Apply(snapshot(2, "bolt", "ward"))
	last := m.Log[len(m.Log)-1]
	if last.Kind != "level_up" || last.Text != "you reached level 2: new ability Ward (key 2)" {
		t.Fatalf("level-up line %+v", last)
	}
	// A snapshot that changes nothing (a respawn) says nothing more.
	n := len(m.Log)
	m.Apply(snapshot(2, "bolt", "ward"))
	if len(m.Log) != n {
		t.Errorf("a second snapshot logged %q", m.Log[len(m.Log)-1].Text)
	}
	// A level that teaches nothing still gets its line.
	at(m, 3000, &pb.Event_Reward{Reward: &pb.Reward{Recipient: "me", Kind: "xp", Amount: 10, Level: 3}})
	m.Apply(snapshot(3, "bolt", "ward"))
	if last := m.Log[len(m.Log)-1]; last.Text != "you reached level 3" {
		t.Errorf("level 3: %q", last.Text)
	}
}

func TestAbilityReasonsAreInWords(t *testing.T) {
	for _, code := range []string{"no_mana", "global_cooldown", "on_cooldown", "casting", "stunned", "empty_slot", "in_combat", "out_of_range", "name_taken", "bad_name", "unknown_class", "too_many_characters"} {
		if w := model.Words(code); w == code || !strings.HasSuffix(w, "("+code+")") {
			t.Errorf("%s: %q", code, w)
		}
	}
}
