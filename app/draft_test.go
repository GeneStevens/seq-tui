package app_test

import (
	"testing"

	"github.com/GeneStevens/seq-tui/app"
	"github.com/GeneStevens/seq-tui/screen"
)

func TestPlanKeysBuildASequenceOfSteps(t *testing.T) {
	m := load(t, "quick_turns.jsonl@round_plan", 0) // round 1 open, at (100000, 90000), a mob 14 m south
	if !m.TurnBased() {
		t.Fatal("not turn-based")
	}
	var d app.Draft
	press := func(key string) app.Action {
		a := app.TurnKey(key, m, &d)
		if a.Kind == app.PlanAdd {
			d.Add(*a.Step)
		}
		return a
	}
	// Arrow steps start where the previous move of the plan ends, not where the
	// character stands now.
	press("down")
	press("down")
	press("left")
	want := [][2]int64{{100000, 90000 - app.StepMm}, {100000, 90000 - 2*app.StepMm}, {100000 - app.StepMm, 90000 - 2*app.StepMm}}
	for i, w := range want {
		if s := d.Steps[i]; s.Kind != app.Move || s.X != w[0] || s.Y != w[1] {
			t.Errorf("step %d: %+v, want move to %v", i+1, s, w)
		}
	}
	if p := d.Dest(); p.GetX() != want[2][0] || p.GetY() != want[2][1] {
		t.Errorf("dest %v", p)
	}
	press("tab")
	if s := d.Steps[3]; s.Kind != app.Target || s.ID == "" {
		t.Errorf("tab: %+v", s)
	}
	// Auto-attack is off now; the plan turns it on, then off again.
	press("f")
	press("f")
	if !d.Steps[4].On || d.Steps[5].On {
		t.Errorf("attack steps %+v %+v", d.Steps[4], d.Steps[5])
	}
	press("x")
	press("r")
	if d.Steps[6].Kind != app.Stop || d.Steps[7].Kind != app.Release {
		t.Errorf("%+v", d.Steps[6:])
	}
	if got := len(d.Agent()); got != 8 {
		t.Errorf("%d toolkit steps", got)
	}
	lines := d.Lines(m)
	if lines[3] != "target Plain Scuttler" || lines[4] != "attack on" || lines[5] != "attack off" || lines[6] != "stop" {
		t.Errorf("lines %q", lines)
	}
	// Backspace removes the last step, Esc all of them; Enter submits.
	if a := press("backspace"); a.Kind != app.PlanUndo {
		t.Errorf("%+v", a)
	}
	if a := press("esc"); a.Kind != app.PlanClear {
		t.Errorf("%+v", a)
	}
	if a := press("enter"); a.Kind != app.PlanSubmit {
		t.Errorf("%+v", a)
	}
}

func TestPlanDraft(t *testing.T) {
	var d app.Draft
	if d.Undo() || d.Clear() {
		t.Error("an empty plan has nothing to remove")
	}
	for i := 0; i < screen.MaxPlanSteps; i++ {
		if !d.Add(app.Action{Kind: app.Stop}) {
			t.Fatalf("step %d refused", i+1)
		}
	}
	if d.Add(app.Action{Kind: app.Stop}) {
		t.Error("a 17th step was taken: the server takes 16")
	}
	d.Sent = true
	if !d.Undo() || d.Sent || len(d.Steps) != screen.MaxPlanSteps-1 {
		t.Errorf("sent %v steps %d: an edit makes the plan unsent", d.Sent, len(d.Steps))
	}
	ver := d.Ver
	d.Clear()
	if d.Ver == ver || len(d.Steps) != 0 {
		t.Error("clear")
	}
}

func TestPlanLootAndDoorAreChosenFromWhereThePlanEnds(t *testing.T) {
	m := load(t, "quick_turns.jsonl@round_drop", 0) // the drop has appeared
	var d app.Draft
	a := app.TurnKey("l", m, &d)
	if a.Kind != app.PlanAdd || a.Step.Kind != app.Loot || a.Step.ID != "drop-inst-1.enc-1-shell-fragment" {
		t.Errorf("%+v", a.Step)
	}
	if a := app.TurnKey("e", m, &d); a.Kind != app.PlanAdd || a.Step.Kind != app.UseDoor || a.Step.ID != "gate-north" {
		t.Errorf("%+v", a.Step)
	}
	// A bare key that would act in real time never does in a plan.
	for _, key := range []string{"up", "tab", "f", "l", "e", "r"} {
		if a := app.TurnKey(key, m, &d); a.Kind != app.PlanAdd && a.Kind != app.None {
			t.Errorf("%s: %+v", key, a)
		}
	}
}

func TestSharedKeysAndTheMenuKey(t *testing.T) {
	m := load(t, "quick_turns.jsonl@round_plan", 0)
	var d app.Draft
	for key, want := range map[string]app.Kind{"i": app.OpenMenu, "q": app.Quit, "?": app.ToggleHelp, "o": app.Logout, "v": app.ToggleVerbose, "z": app.None} {
		if a := app.TurnKey(key, m, &d); a.Kind != want {
			t.Errorf("turn %s: %+v", key, a)
		}
	}
	w := load(t, "solo_trial.jsonl@instance_menu", 0)
	if a := app.Plan("i", w); a.Kind != app.OpenMenu {
		t.Errorf("%+v", a)
	}
	if w.TurnBased() {
		t.Error("the world is real-time: bare intents are sent")
	}
}
