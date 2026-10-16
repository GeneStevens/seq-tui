package app_test

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/app"
	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// program is the Bubble Tea model on a recorded log, with no server behind it:
// a command that needs one answers "not connected".
func program(t *testing.T, name string, upto int) (*app.App, *client.Session) {
	t.Helper()
	s := client.NewSession(client.Config{Model: load(t, name, upto)})
	return app.New(s), s
}

func press(a *app.App, k string) tea.Cmd {
	key := tea.Key{Text: k}
	switch k {
	case "up":
		key = tea.Key{Code: tea.KeyUp}
	case "down":
		key = tea.Key{Code: tea.KeyDown}
	case "enter":
		key = tea.Key{Code: tea.KeyEnter}
	case "esc":
		key = tea.Key{Code: tea.KeyEscape}
	case "backspace":
		key = tea.Key{Code: tea.KeyBackspace}
	case "tab":
		key = tea.Key{Code: tea.KeyTab}
	default:
		key.Code = rune(k[0])
	}
	_, cmd := a.Update(tea.KeyPressMsg(key))
	return cmd
}

var sgr = regexp.MustCompile("\x1b\\[[0-9;]*m")

// screenOf is the screen without its styling.
func screenOf(a *app.App) string { return sgr.ReplaceAllString(a.View().Content, "") }

func TestInstanceMenuOpensChoosesAndCloses(t *testing.T) {
	a, _ := program(t, "solo_trial.jsonl@instance_menu", 0)
	press(a, "i")
	if s := screenOf(a); !strings.Contains(s, "Instances") || !strings.Contains(s, "Quick Turns") || !strings.Contains(s, "Duo Turns") {
		t.Fatalf("no menu:\n%s", s)
	}
	press(a, "4")
	if s := strings.Split(screenOf(a), "\n"); !strings.Contains(strings.Join(s, "\n"), "> 4") {
		t.Errorf("the number did not choose:\n%s", strings.Join(s, "\n"))
	}
	press(a, "esc")
	if strings.Contains(screenOf(a), "Instances") {
		t.Error("esc did not close the menu")
	}
	press(a, "i")
	press(a, "down")
	press(a, "down")
	cmd := press(a, "enter") // chooses Long Game, the third
	if cmd == nil || strings.Contains(screenOf(a), "Instances") {
		t.Fatal("enter should close the menu and send the request")
	}
	a.Update(cmd()) // no server: the answer is the notice
	if s := screenOf(a); !strings.Contains(s, "open instance not sent") {
		t.Errorf("no notice:\n%s", s)
	}
}

func TestTurnBasedKeysBuildAPlanInsteadOfActing(t *testing.T) {
	a, sess := program(t, "quick_turns.jsonl@round_plan", 0)
	a.Update(tea.WindowSizeMsg{Width: 140, Height: 45}) // room for the whole plan
	for _, k := range []string{"down", "tab", "f"} {
		if cmd := press(a, k); cmd != nil {
			t.Fatalf("%s sent something: a bare intent is refused in a turn-based instance", k)
		}
	}
	s := screenOf(a)
	for _, want := range []string{"Plan (3/16): not sent yet", "move (100000,87000)", "target Plain Scuttler", "attack on", "0 of 1 submitted", "none yet", "+"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q:\n%s", want, s)
		}
	}
	press(a, "backspace")
	if !strings.Contains(screenOf(a), "Plan (2/16)") {
		t.Error("backspace did not remove the last step")
	}
	press(a, "esc")
	if s := screenOf(a); !strings.Contains(s, "Plan: empty, not sent") {
		t.Errorf("esc did not clear the plan:\n%s", s)
	}
	// A second move is refused, with its reason; the plan keeps its one.
	press(a, "down")
	press(a, "down")
	if s := screenOf(a); !strings.Contains(s, "Plan (1/16)") || !strings.Contains(s, "a plan holds one move or door") {
		t.Errorf("a second move was taken or not explained:\n%s", s)
	}
	press(a, "esc")
	// Enter sends the plan (here: nowhere), and the plan stays so it can be sent again.
	press(a, "down")
	cmd := press(a, "enter")
	if cmd == nil {
		t.Fatal("enter did not submit")
	}
	a.Update(cmd())
	if s := screenOf(a); !strings.Contains(s, "plan not sent") || !strings.Contains(s, "Plan (1/16)") {
		t.Errorf("after a failed send:\n%s", s)
	}
	// A new round starts an empty plan: the old one was for a round that ran.
	sess.View(func(m *model.Model) {
		m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: m.Last() + 1, Body: &pb.Event_RoundStarted{RoundStarted: &pb.RoundStarted{Round: m.Round.Number + 1, DeadlineMs: 1}}}}})
	})
	if s := screenOf(a); !strings.Contains(s, "Round 2") || !strings.Contains(s, "Plan: empty, not sent") {
		t.Errorf("the plan survived a new round:\n%s", s)
	}
}

func TestRealTimeKeysStillActInTheWorld(t *testing.T) {
	a, _ := program(t, "solo_trial.jsonl@instance_menu", 0)
	if cmd := press(a, "d"); cmd == nil {
		t.Error("a move key in a real-time zone sends the move")
	}
}

func TestNumberKeysUseTheAbilityInThatSlot(t *testing.T) {
	a, _ := program(t, "wizard.jsonl@kit_slotted", 0)
	cmd := press(a, "2")
	if cmd == nil {
		t.Fatal("2 sent nothing")
	}
	a.Update(cmd()) // no server: the answer is the notice, named by the slot's ability
	if s := screenOf(a); !strings.Contains(s, "key 2 Frost Bolt not sent") {
		t.Errorf("no notice:\n%s", s)
	}
	for _, k := range []string{"1", "3", "8"} {
		if cmd := press(a, k); cmd == nil {
			t.Errorf("%s sent nothing: the server answers an empty slot", k)
		}
	}
	if cmd := press(a, "9"); cmd != nil {
		t.Error("9 is not a slot")
	}
}

func TestSlotScreenChoosesASlotAndAnAbility(t *testing.T) {
	a, _ := program(t, "wizard.jsonl@kit_new", 0) // Flame Dart, Frost Bolt, Root in slots 1 to 3
	press(a, "b")
	if s := screenOf(a); !strings.Contains(s, "Ability slots") || !strings.Contains(s, "> 1  Flame Dart") {
		t.Fatalf("no slot screen:\n%s", s)
	}
	press(a, "5")
	if s := screenOf(a); !strings.Contains(s, "> 5  (empty)") {
		t.Errorf("the number did not choose slot 5:\n%s", s)
	}
	if cmd := press(a, "enter"); cmd != nil {
		t.Error("choosing a slot sent something")
	}
	press(a, "down")
	press(a, "down")
	if s := screenOf(a); !strings.Contains(s, "Choose the ability for slot 5") || !strings.Contains(s, "> Root") {
		t.Errorf("not choosing an ability:\n%s", s)
	}
	cmd := press(a, "enter")
	if cmd == nil {
		t.Fatal("enter did not set the slot")
	}
	a.Update(cmd())
	if s := screenOf(a); !strings.Contains(s, "slot 5 not sent") {
		t.Errorf("no notice:\n%s", s)
	}
	// Backspace empties the slot under the cursor; Esc closes the screen.
	if cmd := press(a, "backspace"); cmd == nil {
		t.Error("backspace did not clear the slot")
	}
	press(a, "esc")
	if strings.Contains(screenOf(a), "Ability slots") {
		t.Error("esc did not close the slot screen")
	}
}

func TestCreationScreenKeys(t *testing.T) {
	a := app.NewCreator(client.NewSession(client.Config{}))
	typeKeys := func(keys ...string) {
		for _, k := range keys {
			press(a, k)
		}
	}
	s := screenOf(a)
	if !strings.Contains(s, "New character") || !strings.Contains(s, "> 1  Warrior") {
		t.Fatalf("no creation screen:\n%s", s)
	}
	typeKeys("Z", "e", "d", "d", "1", "!", " ")
	if !strings.Contains(screenOf(a), "Name   Zedd_") {
		t.Errorf("only letters make a name:\n%s", screenOf(a))
	}
	typeKeys("backspace")
	if !strings.Contains(screenOf(a), "Name   Zed_") {
		t.Errorf("backspace:\n%s", screenOf(a))
	}
	// Enter with a short name says so and sends nothing.
	typeKeys("backspace")
	if cmd := press(a, "enter"); cmd != nil || !strings.Contains(screenOf(a), "! a name is 3 to 16 letters") {
		t.Errorf("a short name was sent or not explained:\n%s", screenOf(a))
	}
	typeKeys("d", "tab", "down", "down", "down")
	if !strings.Contains(screenOf(a), "> 4  Wizard") {
		t.Errorf("down in the class list:\n%s", screenOf(a))
	}
	typeKeys("6", "up")
	if !strings.Contains(screenOf(a), "> 5  Paladin") {
		t.Errorf("up after 6:\n%s", screenOf(a))
	}
	// Letters in the class list do not change the name.
	typeKeys("x")
	if !strings.Contains(screenOf(a), "Name   Zed ") || strings.Contains(screenOf(a), "Zedx") {
		t.Errorf("a letter in the class list:\n%s", screenOf(a))
	}
	// Enter makes the character (here: nowhere to make it) and the screen stays with the reason.
	cmd := press(a, "enter")
	if cmd == nil || !strings.Contains(screenOf(a), "Making the character") {
		t.Fatalf("enter did not send:\n%s", screenOf(a))
	}
	a.Update(cmd())
	if s := screenOf(a); !strings.Contains(s, "! not connected") || strings.Contains(s, "Making") {
		t.Errorf("after a failed try:\n%s", s)
	}
	if cmd := press(a, "ctrl+c"); cmd == nil {
		t.Error("ctrl+c did not quit")
	}
}
