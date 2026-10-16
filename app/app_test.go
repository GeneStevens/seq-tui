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
	a, _ := program(t, "live_solo_trial.jsonl", 3)
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
	a, sess := program(t, "live_quick_turns.jsonl", 391)
	a.Update(tea.WindowSizeMsg{Width: 140, Height: 45}) // room for the whole plan
	for _, k := range []string{"down", "down", "tab", "f"} {
		if cmd := press(a, k); cmd != nil {
			t.Fatalf("%s sent something: a bare intent is refused in a turn-based instance", k)
		}
	}
	s := screenOf(a)
	for _, want := range []string{"Plan (4/16): not sent yet", "move (100000,84000)", "target Plain Scuttler", "attack on", "Plans in: none yet", "+"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q:\n%s", want, s)
		}
	}
	press(a, "backspace")
	if !strings.Contains(screenOf(a), "Plan (3/16)") {
		t.Error("backspace did not remove the last step")
	}
	press(a, "esc")
	if s := screenOf(a); !strings.Contains(s, "Plan: empty, not sent") {
		t.Errorf("esc did not clear the plan:\n%s", s)
	}
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
	if s := screenOf(a); !strings.Contains(s, "Round 3") || !strings.Contains(s, "Plan: empty, not sent") {
		t.Errorf("the plan survived a new round:\n%s", s)
	}
}

func TestRealTimeKeysStillActInTheWorld(t *testing.T) {
	a, _ := program(t, "live_solo_trial.jsonl", 3)
	if cmd := press(a, "d"); cmd == nil {
		t.Error("a move key in a real-time zone sends the move")
	}
}
