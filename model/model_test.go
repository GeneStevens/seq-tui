package model_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// replay reads a recorded log from testdata/logs and applies it to a model:
// no backend, no network (doc 03 layer 5). The name is a file, or "file@mark"
// for the messages before a mark a scenario wrote into it.
func replay(t *testing.T, name string) *model.Model {
	t.Helper()
	msgs, err := record.Load(filepath.Join("..", "testdata", "logs"), name)
	if err != nil {
		t.Fatal(err)
	}
	m := model.New()
	for _, msg := range msgs {
		m.Apply(msg)
	}
	return m
}

func TestFightWithLoot(t *testing.T) {
	m := replay(t, "fight_loot.jsonl")
	if m.Last() != 23 || len(m.Faults) != 0 || m.Duplicates != 0 {
		t.Fatalf("stream: last %d faults %v dups %d", m.Last(), m.Faults, m.Duplicates)
	}
	s := m.Self
	if s.XP != 275 || s.HP != 115 || s.Target != "" || s.AutoAttack {
		t.Errorf("self: xp %d hp %d target %q auto %v", s.XP, s.HP, s.Target, s.AutoAttack)
	}
	if len(s.Inventory) != 1 || s.Inventory[0] != "shell-fragment" {
		t.Errorf("inventory %v", s.Inventory)
	}
	mob := m.Entities["mob-outskirts-rat-1-1"]
	if mob == nil || !mob.Dead || mob.Hp != 0 {
		t.Errorf("mob %v", mob)
	}
	if len(m.Drops) != 0 || len(m.Encounters) != 0 {
		t.Errorf("drops %v encounters %v: the loot was taken and the fight is over", m.Drops, m.Encounters)
	}
	if m.Map == nil || m.Map.GetName() != "Outskirts" {
		t.Errorf("map %v", m.Map)
	}
	for _, id := range []string{"tui-1", "tui-2", "tui-3"} {
		if i, ok := m.Intent(id); !ok || i.State != model.IntentAccepted || i.Kind != "?" {
			t.Errorf("intent %s: %+v (no intent was sent in a replay, so the kind is unknown)", id, i)
		}
	}
	var closed *model.LogEntry
	for i := range m.Log {
		if m.Log[i].Kind == "encounter_closed" {
			closed = &m.Log[i]
		}
	}
	if closed == nil || closed.Reason != "mob_died" || closed.Seq != 18 {
		t.Errorf("encounter_closed entry %+v", closed)
	}
	if len(m.Log) != 14 { // the 13 events and the one-time tip on "tradable once" (D78)
		t.Errorf("log has %d entries", len(m.Log))
	}
}

func TestKillStealDenial(t *testing.T) {
	m := replay(t, "kill_steal.jsonl")
	i, ok := m.Intent("tui-2")
	if !ok || i.State != model.IntentDenied || i.Reason != "encounter_owned" {
		t.Fatalf("intent %+v", i)
	}
	var denied *model.LogEntry
	for k := range m.Log {
		if m.Log[k].Kind == "denied" {
			denied = &m.Log[k]
		}
	}
	if denied == nil || denied.Reason != "encounter_owned" {
		t.Fatalf("denied entry %+v", denied)
	}
	if e := m.Encounters["enc-9"]; e == nil || e.GetOwner() != "char-bram" {
		t.Errorf("encounter %v: bram still owns it", e)
	}
	if m.Self.XP != 0 || m.Entities["mob-outskirts-rat-2-1"].Hp != 7 {
		t.Errorf("xp %d, mob hp %d", m.Self.XP, m.Entities["mob-outskirts-rat-2-1"].Hp)
	}
	if len(m.Group) != 0 {
		t.Errorf("bram is not in the group: %v", m.Group)
	}
}

func TestCrossing(t *testing.T) {
	m := replay(t, "crossing.jsonl")
	if m.Space != "testgrounds" || m.Map == nil || m.Map.GetSpace() != "testgrounds" {
		t.Fatalf("space %q map %v", m.Space, m.Map)
	}
	if len(m.Entities) != 1 || m.Entities["mob-testgrounds-rat-1-1"] == nil {
		t.Errorf("the old space's entities must be gone: %v", m.EntityIDs())
	}
	// the new space's stream is numbered afresh: the snapshot said 7, the event is 8
	if m.Last() != 8 || len(m.Faults) != 0 {
		t.Errorf("last %d faults %v", m.Last(), m.Faults)
	}
	var kinds []string
	for _, e := range m.Log {
		kinds = append(kinds, e.Kind)
	}
	if len(kinds) != 2 || kinds[0] != "space_changed" || m.Log[0].Seq != 0 {
		t.Errorf("log %v", kinds)
	}
}

func TestDropAndResume(t *testing.T) {
	m := replay(t, "drop_resume.jsonl")
	if m.Connections != 2 {
		t.Errorf("connections %d", m.Connections)
	}
	if m.Last() != 54 || len(m.Faults) != 0 {
		t.Errorf("last %d faults %v", m.Last(), m.Faults)
	}
	if m.Duplicates != 1 {
		t.Errorf("duplicates %d: the repeated 52 must be taken once", m.Duplicates)
	}
	// 5+5(dup, ignored)+6+4 = 15 taken from a 20 hp rat
	if hp := m.Entities["mob-outskirts-rat-1-1"].Hp; hp != 5 {
		t.Errorf("mob hp %d: every hit exactly once", hp)
	}
	if n := len(m.Log); n != 4 {
		t.Errorf("log %d entries, want 4 (51-54)", n)
	}
}

func TestLogIsBounded(t *testing.T) {
	m := model.NewBounded(3, 2)
	f, _ := os.Open(filepath.Join("..", "testdata", "logs", "fight_loot.jsonl"))
	defer f.Close()
	msgs, _ := record.Read(f)
	for _, msg := range msgs {
		m.Apply(msg)
	}
	// The bound is applied in batches (trimLog): up to three caps of lines
	// between passes, the newest always kept, in order.
	if n := len(m.Log); n < 3 || n > 8 || m.Log[n-3].Seq != 22 || m.Log[n-2].Seq != 23 || m.Log[n-1].Kind != "tip" {
		t.Errorf("log %+v", m.Log)
	}
	if len(m.Intents) != 2 {
		t.Errorf("intents %d", len(m.Intents))
	}
}

func TestGapIsReported(t *testing.T) {
	g := model.New()
	f, _ := os.Open(filepath.Join("..", "testdata", "logs", "kill_steal.jsonl"))
	defer f.Close()
	msgs, _ := record.Read(f)
	for i, msg := range msgs {
		if i == 4 { // drop event 32
			continue
		}
		g.Apply(msg)
	}
	if len(g.Faults) != 1 {
		t.Errorf("faults %v", g.Faults)
	}
}

func TestText(t *testing.T) {
	out := replay(t, "kill_steal.jsonl").Text()
	for _, want := range []string{"Aria", "encounter_owned", "enc-9 mob-outskirts-rat-2-1 owned by char-bram"} {
		if !strings.Contains(out, want) {
			t.Errorf("text lacks %q:\n%s", want, out)
		}
	}
}

// play.jsonl is the session of the login-move-fight-loot scenario.
func TestScenarioLogin(t *testing.T) {
	m := replay(t, "play.jsonl@play_login")
	if m.Self.ID != "aria" || m.Space != "Testgrounds" || m.Map == nil || len(m.Map.GetDoors()) != 2 || len(m.Entities) != 1 || m.Welcome == nil {
		t.Errorf("self %+v space %q map %v entities %v", m.Self, m.Space, m.Map, m.EntityIDs())
	}
	if m.Instance != nil || m.InInstance() {
		t.Errorf("a world zone carries no instance: %v", m.Instance)
	}
}

// The release scenario: the character dies, releases, and comes back; the
// fresh snapshot after the respawned event is the new view.
func TestReleaseAndRespawnFromAScenarioLog(t *testing.T) {
	m := replay(t, "release.jsonl")
	s := m.Self
	if s.Dead || s.Released || s.HP != s.MaxHP || s.Pos.GetX() != 10000 || s.Pos.GetY() != 50000 {
		t.Errorf("after the respawn: %+v", s)
	}
	if len(m.Faults) != 0 || m.Duplicates != 0 {
		t.Errorf("stream: last %d faults %v dups %d", m.Last(), m.Faults, m.Duplicates)
	}
	var kinds []string
	for _, e := range m.Log {
		if e.Kind == "released" || e.Kind == "respawned" {
			kinds = append(kinds, e.Kind)
		}
	}
	if strings.Join(kinds, " ") != "released respawned" {
		t.Errorf("log kinds %v", kinds)
	}
}

func TestDeathThenReleasedShowsTheWait(t *testing.T) {
	m := replay(t, "release.jsonl@release_dead")
	if !m.Self.Dead || m.Self.Released {
		t.Errorf("dead, not released: %+v", m.Self)
	}
	m = replay(t, "release.jsonl@release_released")
	if !m.Self.Dead || !m.Self.Released || m.Self.RespawnInMs != 5000 {
		t.Errorf("released and waiting: %+v", m.Self)
	}
}

func TestCrossingTheDoorAndBackFromAScenarioLog(t *testing.T) {
	m := replay(t, "door.jsonl")
	if m.Space != "Testgrounds" || m.Map == nil || m.Map.GetSpace() != "Testgrounds" || len(m.Faults) != 0 {
		t.Errorf("back in the first zone: space %q map %v last %d faults %v", m.Space, m.Map.GetSpace(), m.Last(), m.Faults)
	}
	var entered []string
	for _, e := range m.Log {
		if e.Kind == "space_changed" {
			entered = append(entered, e.Text)
		}
	}
	if strings.Join(entered, "; ") != "entered Outskirts; entered Testgrounds" {
		t.Errorf("crossings %v", entered)
	}
}

func TestLootGoneRemovesTheDropAtOnce(t *testing.T) {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Seq: 5, Space: "z", Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me"}},
		Drops: []*pb.LootDropState{{Id: "drop-1"}, {Id: "drop-2"}}}}})
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: 6, Body: &pb.Event_LootGone{LootGone: &pb.LootGone{Drop: "drop-1", Reason: "expired"}}}}})
	if len(m.Drops) != 1 || m.Drops["drop-2"] == nil {
		t.Errorf("drops %v: loot_gone removes the drop, not the next snapshot", m.Drops)
	}
	if e := m.Log[len(m.Log)-1]; e.Kind != "loot_gone" || e.Reason != "expired" {
		t.Errorf("log entry %+v", e)
	}
}

func TestForgottenResultIsFlagged(t *testing.T) {
	m := model.New()
	m.IntentSent("a", 1, "move")
	m.IntentSent("b", 2, "move")
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_IntentResult{IntentResult: &pb.IntentResult{IntentId: "a", Accepted: true, ResultUnknown: true}}})
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_IntentResult{IntentResult: &pb.IntentResult{IntentId: "b", Accepted: true}}})
	a, _ := m.Intent("a")
	b, _ := m.Intent("b")
	if a.State != model.IntentAccepted || !a.Unknown || b.Unknown {
		t.Errorf("a %+v b %+v", a, b)
	}
}
