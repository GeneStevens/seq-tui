package app_test

import (
	"testing"

	"github.com/GeneStevens/seq-tui/app"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
)

// load replays a recorded log into a model: a file name in testdata/logs, or
// "file@mark" for the messages before a mark the scenario wrote.
func load(t *testing.T, spec string, upto int) *model.Model {
	t.Helper()
	msgs, err := record.Load("../testdata/logs", spec)
	if err != nil {
		t.Fatal(err)
	}
	if upto > 0 {
		msgs = msgs[:upto]
	}
	m := model.New()
	for _, msg := range msgs {
		m.Apply(msg)
	}
	return m
}

func TestMoveKeysAimAStepFromWhereYouStand(t *testing.T) {
	m := load(t, "play.jsonl@play_login", 0) // at (10000, 50000)
	for key, want := range map[string][2]int64{
		"up": {10000, 50000 + app.StepMm}, "w": {10000, 50000 + app.StepMm},
		"down": {10000, 50000 - app.StepMm}, "s": {10000, 50000 - app.StepMm},
		"left": {10000 - app.StepMm, 50000}, "a": {10000 - app.StepMm, 50000},
		"right": {10000 + app.StepMm, 50000}, "d": {10000 + app.StepMm, 50000},
	} {
		a := app.Plan(key, m)
		if a.Kind != app.Move || a.X != want[0] || a.Y != want[1] {
			t.Errorf("%s: %+v, want move to %v", key, a, want)
		}
	}
}

func TestTabCyclesLivingMobsNearestFirst(t *testing.T) {
	m := load(t, "kill_steal.jsonl", 0)
	a := app.Plan("tab", m)
	if a.Kind != app.Target || a.ID == "" {
		t.Fatalf("%+v", a)
	}
	// With the only living mob already targeted, Tab picks it again; it never
	// picks a dead one or a character.
	m.Self.Target = a.ID
	if b := app.Plan("tab", m); b.Kind != app.Target || b.ID != a.ID {
		t.Errorf("%+v", b)
	}
	m.Entities[a.ID].Dead = true
	if b := app.Plan("tab", m); b.Kind != app.None || b.Note == "" {
		t.Errorf("a dead mob was targeted: %+v", b)
	}
}

func TestAttackToggles(t *testing.T) {
	m := load(t, "fight_loot.jsonl", 8) // auto-attack on after event 12
	if a := app.Plan("f", m); a.Kind != app.Attack || a.On {
		t.Errorf("with auto-attack on, f should turn it off: %+v", a)
	}
	m.Self.AutoAttack = false
	if a := app.Plan("space", m); a.Kind != app.Attack || !a.On {
		t.Errorf("%+v", a)
	}
}

func TestLootTakesEverythingInReach(t *testing.T) {
	m := load(t, "fight_loot.jsonl", 14) // the drop has just appeared
	if a := app.Plan("l", m); a.Kind != app.Loot || a.ID != "" { // loot-all names no drop (D78)
		t.Errorf("%+v", a)
	}
	if a := app.Plan("l", load(t, "play.jsonl@play_login", 0)); a.Kind != app.None || a.Note == "" {
		t.Errorf("no drop in view: %+v", a)
	}
}

func TestDoorNamesTheNearestDoor(t *testing.T) {
	m := load(t, "play.jsonl@play_login", 0)
	if a := app.Plan("e", m); a.Kind != app.UseDoor || a.ID != "gate-north" {
		t.Errorf("%+v", a)
	}
	if a := app.Plan("e", model.New()); a.Kind != app.None {
		t.Errorf("no map: %+v", a)
	}
}

func TestOtherKeys(t *testing.T) {
	m := load(t, "fight_loot.jsonl", 8)
	for key, want := range map[string]app.Kind{
		"o": app.Logout, "q": app.Quit, "ctrl+c": app.Quit, "?": app.ToggleHelp, "v": app.ToggleVerbose, "esc": app.Target, "z": app.None,
	} {
		if a := app.Plan(key, m); a.Kind != want {
			t.Errorf("%s: %+v, want kind %v", key, a, want)
		}
	}
}

func TestRKeyReleases(t *testing.T) {
	m := load(t, "release.jsonl@release_dead", 0) // dead
	if a := app.Plan("r", m); a.Kind != app.Release {
		t.Errorf("%+v", a)
	}
}
