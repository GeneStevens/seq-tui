package app_test

import (
	"strings"
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
	press := func(key string) (app.Action, string) {
		a := app.TurnKey(key, m, &d)
		why := ""
		if a.Kind == app.PlanAdd {
			why = d.Add(*a.Step)
		}
		return a, why
	}
	// A move step starts where the character stands now; a plan holds one move
	// (doc 05 §4.6), so a second is refused and says why.
	press("down")
	if s := d.Steps[0]; s.Kind != app.Move || s.X != 100000 || s.Y != 90000-app.StepMm {
		t.Errorf("move: %+v", s)
	}
	if _, why := press("left"); !strings.Contains(why, "one move or door") || len(d.Steps) != 1 {
		t.Errorf("a second move: %q, %d steps", why, len(d.Steps))
	}
	if _, why := press("e"); !strings.Contains(why, "one move or door") {
		t.Errorf("a door after a move: %q", why)
	}
	if p := d.Dest(); p.GetX() != 100000 || p.GetY() != 90000-app.StepMm {
		t.Errorf("dest %v", p)
	}
	press("tab")
	if s := d.Steps[1]; s.Kind != app.Target || s.ID == "" {
		t.Errorf("tab: %+v", s)
	}
	// Auto-attack is off now; the plan turns it on. That is the plan's one
	// action: neither a second toggle nor an ability is taken after it.
	press("f")
	if !d.Steps[2].On {
		t.Errorf("attack step %+v", d.Steps[2])
	}
	for _, key := range []string{"f", "1", "8"} {
		if _, why := press(key); !strings.Contains(why, "one ability or attack toggle") || len(d.Steps) != 3 {
			t.Errorf("%s after an attack: %q, %d steps", key, why, len(d.Steps))
		}
	}
	press("x")
	press("r")
	if d.Steps[3].Kind != app.Stop || d.Steps[4].Kind != app.Release {
		t.Errorf("%+v", d.Steps[3:])
	}
	if got := len(d.Agent()); got != 5 {
		t.Errorf("%d toolkit steps", got)
	}
	lines := d.Lines(m)
	if lines[1] != "target Plain Scuttler" || lines[2] != "attack on" || lines[3] != "stop" {
		t.Errorf("lines %q", lines)
	}
	// Removing the attack frees the action for an ability.
	d.Undo()
	d.Undo()
	d.Undo()
	if _, why := press("2"); why != "" || d.Steps[len(d.Steps)-1].Kind != app.UseAbility || d.Steps[len(d.Steps)-1].Slot != 2 {
		t.Errorf("an ability after removing the attack: %q %+v", why, d.Steps)
	}
	if got := d.Lines(m); got[len(got)-1] != "use key 2 (empty)" {
		t.Errorf("lines %q", got)
	}
	// Backspace removes the last step, Esc all of them; Enter submits.
	if a, _ := press("backspace"); a.Kind != app.PlanUndo {
		t.Errorf("%+v", a)
	}
	if a, _ := press("esc"); a.Kind != app.PlanClear {
		t.Errorf("%+v", a)
	}
	if a, _ := press("enter"); a.Kind != app.PlanSubmit {
		t.Errorf("%+v", a)
	}
}

func TestPlanDraft(t *testing.T) {
	var d app.Draft
	if d.Undo() || d.Clear() {
		t.Error("an empty plan has nothing to remove")
	}
	for i := 0; i < screen.MaxPlanSteps; i++ {
		if why := d.Add(app.Action{Kind: app.Stop}); why != "" {
			t.Fatalf("step %d refused: %s", i+1, why)
		}
	}
	if why := d.Add(app.Action{Kind: app.Stop}); !strings.Contains(why, "plan is full") {
		t.Errorf("a 17th step was taken: the server takes 16 (%q)", why)
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
	if a := app.TurnKey("3", m, &d); a.Kind != app.PlanAdd || a.Step.Kind != app.UseAbility || a.Step.Slot != 3 {
		t.Errorf("a number key adds an ability step: %+v", a)
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

// Between rounds a slot changes by a plan step (D50): any number of them, none
// counted as the plan's move or action, and the plan sends them to the server.
func TestSlotChangesAreStepsOfThePlan(t *testing.T) {
	m := load(t, "turn_cast.jsonl@turn_cast_plan", 0)
	var d app.Draft
	for _, s := range []app.Action{
		{Kind: app.SetSlot, Slot: 1, ID: "wizard-frost-bolt"},
		{Kind: app.SetSlot, Slot: 2, ID: "wizard-flame-dart"},
		{Kind: app.SetSlot, Slot: 1},
		{Kind: app.UseAbility, Slot: 2},
		{Kind: app.Move, X: 1, Y: 2},
	} {
		if why := d.Add(s); why != "" {
			t.Fatalf("%+v refused: %s", s, why)
		}
	}
	got := d.Slots()
	if len(got) != 2 || got[1] != "" || got[2] != "wizard-flame-dart" {
		t.Errorf("the plan's slots: %v", got)
	}
	lines := d.Lines(m)
	if lines[0] != "key 1: Frost Bolt" || lines[2] != "empty key 1" {
		t.Errorf("lines %q", lines)
	}
	if n := len(d.Agent()); n != 5 {
		t.Errorf("%d steps for the server", n)
	}
}
