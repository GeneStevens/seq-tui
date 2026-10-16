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
		"fight_loot.jsonl":  fightLoot(),
		"kill_steal.jsonl":  killSteal(),
		"crossing.jsonl":    crossing(),
		"drop_resume.jsonl": dropResume(),
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
			e.Body = &pb.Event_LootDropped{LootDropped: &pb.LootDropped{Drop: "drop-enc-1-shell-fragment", Mob: mob, Item: "shell-fragment", Pos: pos(2000, 0), Encounter: "enc-1", CanLoot: true}}
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
			e.Body = &pb.Event_Reward{Reward: &pb.Reward{Recipient: me, Source: "drop-enc-1-shell-fragment", Kind: "loot", Item: "shell-fragment"}}
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
