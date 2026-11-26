// Command mkfixtures writes the recorded-log fixtures under testdata/logs. The
// logs are built from the shapes of seq's golden fixtures (protocol/testdata),
// in the order a real session delivers them; tests only read them. The logs
// that come from a played session (play, release, door, solo_trial,
// quick_turns, duo_turns) are the scenarios' (`make fixtures` runs both). Run
// it by hand, and review the diff, if a fixture must change:
//
//	go run ./cmd/mkfixtures testdata/logs
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GeneStevens/seq-tui/record"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

type msg = *pb.ServerMessage

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: mkfixtures <dir>")
		os.Exit(2)
	}
	for name, msgs := range map[string][]msg{
		"fight_loot.jsonl":     fightLoot(),
		"kill_steal.jsonl":     killSteal(),
		"crossing.jsonl":       crossing(),
		"drop_resume.jsonl":    dropResume(),
		"castable_chief.jsonl": castableChief(),
	} {
		f, err := os.Create(filepath.Join(os.Args[1], name))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		w := record.NewWriter(f)
		for _, m := range msgs {
			if err := w.Write(m); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		_ = f.Close()
	}
}

func pos(x, y int64) *pb.Pos { return &pb.Pos{X: x, Y: y} }

func welcome() msg {
	return &pb.ServerMessage{Body: &pb.ServerMessage_Welcome{Welcome: &pb.Welcome{ProtocolVersion: 1, MinProtocolVersion: 1, TickMs: 100,
		Limits: &pb.Limits{MaxMessageBytes: 65536, MaxIntentsPerSecond: 20}}}}
}

func char(id, name string, p *pb.Pos, hp int64) *pb.EntityState {
	return &pb.EntityState{Id: id, Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, Name: name, Pos: p, Hp: hp, MaxHp: 120, Level: 3}
}

func rat(id string, p *pb.Pos) *pb.EntityState {
	return &pb.EntityState{Id: id, Kind: pb.EntityKind_ENTITY_KIND_MOB, Name: "Shell Rat", Pos: p, Hp: 20, MaxHp: 20, Level: 1}
}

func snapshot(seq uint64, space string, self *pb.EntityState, xp int64, others ...*pb.EntityState) msg {
	return &pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Seq: seq, GameTimeMs: int64(seq) * 100, Tick: seq, Space: space,
		Self:     &pb.SelfState{Entity: self, Xp: xp, Stats: &pb.Stats{Level: 3, MaxHp: 120, SpeedMmS: 4500, MeleeRangeMm: 2500}},
		Entities: others}}}
}

func zoneMap(space, name string, doors ...*pb.Door) msg {
	return &pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: space, Name: name,
		Bounds: &pb.Rect{MinX: -50000, MinY: -50000, MaxX: 50000, MaxY: 50000}, Doors: doors, ViewRadiusMm: 20000,
		Walls: []*pb.Wall{{A: pos(-10000, 5000), B: pos(10000, 5000)}}}}}
}

func result(id string, reason string) msg {
	return &pb.ServerMessage{Body: &pb.ServerMessage_IntentResult{IntentResult: &pb.IntentResult{IntentId: id, Accepted: reason == "", Reason: reason}}}
}

func ev(seq uint64, f func(*pb.Event)) msg {
	e := &pb.Event{Seq: seq, GameTimeMs: int64(seq) * 100}
	f(e)
	return &pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: e}}
}

func hit(seq uint64, by, target string, n int64) msg {
	return ev(seq, func(e *pb.Event) {
		e.Body = &pb.Event_Combat{Combat: &pb.Combat{Kind: pb.CombatKind_COMBAT_KIND_HIT, Attacker: by, Target: target, Amount: n}}
	})
}

func targetChanged(seq uint64, actor, target string) msg {
	return ev(seq, func(e *pb.Event) {
		e.Body = &pb.Event_TargetChanged{TargetChanged: &pb.TargetChanged{Actor: actor, Target: target}}
	})
}

func autoAttack(seq uint64, actor, target string, on bool) msg {
	return ev(seq, func(e *pb.Event) {
		e.Body = &pb.Event_AutoAttackChanged{AutoAttackChanged: &pb.AutoAttackChanged{Actor: actor, Target: target, On: on}}
	})
}

func opened(seq uint64, id, owner, mob string) msg {
	return ev(seq, func(e *pb.Event) {
		e.Body = &pb.Event_EncounterOpened{EncounterOpened: &pb.EncounterOpened{Id: id, Owner: owner, Mob: mob}}
	})
}

// fightLoot: aria targets a rat, kills it, and takes the drop.
func fightLoot() []msg {
	const me, mob = "char-aria", "mob-outskirts-rat-1-1"
	return []msg{
		welcome(),
		snapshot(10, "outskirts", char(me, "Aria", pos(0, 0), 120), 250, rat(mob, pos(2000, 0))),
		zoneMap("outskirts", "Outskirts"),
		result("tui-1", ""),
		targetChanged(11, me, mob),
		result("tui-2", ""),
		autoAttack(12, me, mob, true),
		opened(13, "enc-1", me, mob),
		hit(14, me, mob, 8),
		hit(15, mob, me, 5),
		hit(16, me, mob, 12),
		ev(17, func(e *pb.Event) {
			e.Body = &pb.Event_Combat{Combat: &pb.Combat{Kind: pb.CombatKind_COMBAT_KIND_DEATH, Attacker: me, Target: mob}}
		}),
		ev(18, func(e *pb.Event) {
			e.Body = &pb.Event_EncounterClosed{EncounterClosed: &pb.EncounterClosed{Id: "enc-1", Owner: me, Mob: mob, Reason: "mob_died"}}
		}),
		ev(19, func(e *pb.Event) {
			e.Body = &pb.Event_LootDropped{LootDropped: &pb.LootDropped{Drop: "drop-enc-1-shell-fragment", Mob: mob, Item: "shell-fragment", Pos: pos(2000, 0), Encounter: "enc-1", CanLoot: true,
				ItemInfo: &pb.ItemInfo{Id: "shell-fragment", Name: "Shell Fragment", Tier: "common"}}}
		}),
		ev(20, func(e *pb.Event) {
			e.Body = &pb.Event_Reward{Reward: &pb.Reward{Recipient: me, Source: mob, Kind: "xp", Amount: 25}}
		}),
		ev(21, func(e *pb.Event) {
			e.Body = &pb.Event_TargetCleared{TargetCleared: &pb.TargetCleared{Actor: me, Target: mob, Reason: "target_dead"}}
		}),
		autoAttack(22, me, "", false),
		result("tui-3", ""),
		ev(23, func(e *pb.Event) {
			e.Body = &pb.Event_Reward{Reward: &pb.Reward{Recipient: me, Source: "drop-enc-1-shell-fragment", Kind: "loot", Item: "shell-fragment", ItemInfo: &pb.ItemInfo{Id: "shell-fragment", Name: "Shell Fragment", Tier: "common"}}}
		}),
	}
}

// killSteal: bram owns the fight with a rat; aria's attack on it is denied.
func killSteal() []msg {
	const me, other, mob = "char-aria", "char-bram", "mob-outskirts-rat-2-1"
	return []msg{
		welcome(),
		snapshot(30, "outskirts", char(me, "Aria", pos(0, 0), 120), 0, char(other, "Bram", pos(3000, 0), 120), rat(mob, pos(4000, 0))),
		zoneMap("outskirts", "Outskirts"),
		opened(31, "enc-9", other, mob),
		hit(32, other, mob, 6),
		result("tui-1", ""),
		targetChanged(33, me, mob),
		result("tui-2", "encounter_owned"),
		ev(34, func(e *pb.Event) {
			e.Body = &pb.Event_Denied{Denied: &pb.Denied{Intent: "auto_attack", Target: mob, Reason: "encounter_owned", Actor: me}}
		}),
		hit(35, other, mob, 7),
	}
}

// crossing: aria uses a door and arrives in another space.
func crossing() []msg {
	const me = "char-aria"
	door := &pb.Door{Id: "line-east", Kind: pb.DoorKind_DOOR_KIND_ZONE_LINE, Pos: pos(50000, 0)}
	return []msg{
		welcome(),
		snapshot(40, "outskirts", char(me, "Aria", pos(49000, 0), 120), 10, rat("mob-outskirts-rat-3-1", pos(48000, 500))),
		zoneMap("outskirts", "Outskirts", door),
		result("tui-1", ""),
		{Body: &pb.ServerMessage_Event{Event: &pb.Event{GameTimeMs: 4100, Body: &pb.Event_SpaceChanged{SpaceChanged: &pb.SpaceChanged{Space: "testgrounds"}}}}},
		snapshot(7, "testgrounds", char(me, "Aria", pos(-49000, 0), 120), 10),
		zoneMap("testgrounds", "Testgrounds"),
		ev(8, func(e *pb.Event) {
			e.Body = &pb.Event_EntityAppeared{EntityAppeared: &pb.EntityAppeared{Entity: rat("mob-testgrounds-rat-1-1", pos(-45000, 0))}}
		}),
	}
}

// dropResume: the connection drops after event 52; a resume replays 53 and
// 54 once each (a repeat of 52 is sent too, as a careless server might, and
// the model must take it once), then the stream goes on.
func dropResume() []msg {
	const me, mob = "char-aria", "mob-outskirts-rat-1-1"
	return []msg{
		welcome(),
		snapshot(50, "outskirts", char(me, "Aria", pos(0, 0), 120), 0, rat(mob, pos(2000, 0))),
		zoneMap("outskirts", "Outskirts"),
		result("tui-1", ""),
		targetChanged(51, me, mob),
		hit(52, me, mob, 5),
		// the connection drops here; the client dials again and resumes after 52
		welcome(),
		hit(52, me, mob, 5),
		hit(53, me, mob, 6),
		result("tui-2", ""),
		hit(54, me, mob, 4),
	}
}

// castableChief: rex faces a Warren Chief that casts Brow Breaker (2 s) four times,
// each ending its own way, as seq's castable-chief wire records have it: the
// cast that lands (rex steps out of reach and back in on the way), one that rex
// interrupts with a Kick, one a stun breaks, and one that rex is out of reach of
// at the deadline. The screens are drawn at the messages the cases name
// (screen.Cases chief_*), so the order of these is the order of the cases.
func castableChief() []msg {
	const me, chief = "char-rex", "mob-chief-1"
	at := func(ms int64, f func(*pb.Event)) msg {
		e := &pb.Event{GameTimeMs: ms}
		f(e)
		return &pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: e}}
	}
	seq := uint64(80)
	next := func(ms int64, f func(*pb.Event)) msg {
		seq++
		return at(ms, func(e *pb.Event) { e.Seq = seq; f(e) })
	}
	started := func(ms int64) msg {
		return next(ms, func(e *pb.Event) {
			e.Text = "Warren Chief begins casting Brow Breaker on you"
			e.Body = &pb.Event_CastStarted{CastStarted: &pb.CastStarted{Actor: chief, Target: me, Ability: "chief-brow-breaker", CastMs: 2000}}
		})
	}
	moved := func(ms, x int64) msg {
		return next(ms, func(e *pb.Event) {
			e.Body = &pb.Event_EntityMoved{EntityMoved: &pb.EntityMoved{Id: me, Pos: pos(x, 10000)}}
		})
	}
	used := func(ms int64, actor, target, ability string) msg {
		return next(ms, func(e *pb.Event) {
			e.Body = &pb.Event_AbilityUsed{AbilityUsed: &pb.AbilityUsed{Actor: actor, Target: target, Ability: ability}}
		})
	}
	stun := func(ms int64, source, target, ability string, dur int64) msg {
		return next(ms, func(e *pb.Event) {
			e.Body = &pb.Event_EffectApplied{EffectApplied: &pb.EffectApplied{Source: source, Target: target, Ability: ability, Kind: "stun", DurationMs: dur}}
		})
	}
	ended := func(ms int64, interrupted bool, reason string) msg {
		return next(ms, func(e *pb.Event) {
			o := &pb.CastEnded{Actor: chief, Target: me, Ability: "chief-brow-breaker", Reason: reason}
			if interrupted {
				e.Body = &pb.Event_CastInterrupted{CastInterrupted: o}
			} else {
				e.Body = &pb.Event_CastCancelled{CastCancelled: o}
			}
		})
	}
	self := char(me, "Rex", pos(10000, 10000), 120)
	boss := &pb.EntityState{Id: chief, Kind: pb.EntityKind_ENTITY_KIND_MOB, Name: "Warren Chief", Pos: pos(11800, 10000), Hp: 300, MaxHp: 300, Level: 7}
	snap := snapshot(80, "warren", self, 0, boss)
	return []msg{
		welcome(),
		snap,
		zoneMap("warren", "Hollow Warren"),
		next(8100, func(e *pb.Event) {
			e.Body = &pb.Event_TargetChanged{TargetChanged: &pb.TargetChanged{Actor: me, Target: chief}}
		}),
		// lands: cast_started at 8200, rex out of reach for a while, back in at the deadline
		started(8200),
		moved(8600, 6900),
		moved(9000, 10000),
		used(10200, chief, me, "chief-brow-breaker"),
		next(10200, func(e *pb.Event) {
			e.Body = &pb.Event_Combat{Combat: &pb.Combat{Kind: pb.CombatKind_COMBAT_KIND_HIT, Attacker: chief, Target: me, Amount: 19, Ability: "chief-brow-breaker", School: "physical"}}
		}),
		stun(10200, chief, me, "chief-brow-breaker", 2000),
		// kicked
		started(20800),
		used(21100, me, chief, "rogue-kick"),
		ended(21100, true, "interrupt"),
		// stunned
		started(30000),
		used(30100, me, chief, "paladin-judgment"),
		stun(30100, me, chief, "paladin-judgment", 1500),
		ended(30100, false, "stunned"),
		// stepped out
		started(40000),
		moved(40500, 6900),
		ended(42000, false, "out_of_range"),
	}
}
