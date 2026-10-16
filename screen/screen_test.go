package screen_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

const logs = "../testdata/logs"

func replay(t *testing.T, c screen.Case) *model.Model {
	t.Helper()
	m, err := c.Replay(logs)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// The goldens are compared, never written: `make golden` rewrites them.
func TestGoldenScreens(t *testing.T) {
	for _, c := range screen.Cases {
		m := replay(t, c)
		for _, sz := range screen.Sizes {
			name := c.File(sz[0], sz[1])
			want, err := os.ReadFile(filepath.Join("../testdata/screens", name))
			if err != nil {
				t.Errorf("%s: %v (run make golden)", name, err)
				continue
			}
			if got := c.Draw(m, sz[0], sz[1]) + "\n"; got != string(want) {
				t.Errorf("%s differs from its golden screen (make golden rewrites it after review)\n--- got\n%s--- want\n%s", name, got, want)
			}
		}
	}
}

func TestEveryScreenFitsItsSize(t *testing.T) {
	sizes := [][2]int{{80, 24}, {81, 25}, {100, 30}, {120, 40}, {140, 45}, {200, 60}, {80, 60}, {250, 24}}
	for _, c := range screen.Cases {
		m := replay(t, c)
		for _, sz := range sizes {
			out := c.Draw(m, sz[0], sz[1])
			lines := strings.Split(out, "\n")
			if len(lines) != sz[1] {
				t.Errorf("%s at %v: %d lines", c.Name, sz, len(lines))
			}
			for i, l := range lines {
				if n := utf8.RuneCountInString(l); n != sz[0] {
					t.Errorf("%s at %v: line %d is %d cells: %q", c.Name, sz, i, n, l)
				}
			}
		}
	}
}

func TestTooSmallSaysSoAndDoesNotOverflow(t *testing.T) {
	m := replay(t, screen.Cases[0])
	out := screen.Render(m, screen.View{}, 40, 10, screen.Plain())
	lines := strings.Split(out, "\n")
	if len(lines) != 10 || !strings.Contains(lines[0], "too small") {
		t.Fatalf("got %q", out)
	}
	for _, l := range lines {
		if utf8.RuneCountInString(l) != 40 {
			t.Fatalf("line %q is not 40 cells", l)
		}
	}
}

func TestEmptyModelDraws(t *testing.T) {
	out := screen.Render(model.New(), screen.View{Conn: screen.ConnOffline}, 80, 24, screen.Plain())
	if !strings.Contains(out, "OFFLINE") || len(strings.Split(out, "\n")) != 24 {
		t.Fatalf("got\n%s", out)
	}
}

var sgr = regexp.MustCompile("\x1b\\[[0-9;]*m")

// Nothing depends on colour: the styled screen is the plain screen with escape
// sequences added, so every distinction is in the text itself.
func TestStylingOnlyAddsEscapeSequences(t *testing.T) {
	for _, c := range screen.Cases {
		m := replay(t, c)
		v := screen.View{Conn: screen.ConnOnline, Notice: "x denied: y", NoticeBad: true, HasTick: true}
		plain := screen.Render(m, v, 100, 30, screen.Plain())
		styled := screen.Render(m, v, 100, 30, screen.ANSI())
		if strings.Contains(plain, "\x1b") {
			t.Errorf("%s: plain screen has escape sequences", c.Name)
		}
		if got := sgr.ReplaceAllString(styled, ""); got != plain {
			t.Errorf("%s: styled screen differs from plain once styling is removed", c.Name)
		}
	}
}

// D17: the screen never says what drives a character.
func TestNoControllerTypeShown(t *testing.T) {
	for _, c := range screen.Cases {
		m := replay(t, c)
		for _, sz := range screen.Sizes {
			low := strings.ToLower(c.Draw(m, sz[0], sz[1]))
			for _, w := range []string{"agent", "human", "bot", "npc", "ai-"} {
				if strings.Contains(low, w) {
					t.Errorf("%s at %v mentions %q", c.Name, sz, w)
				}
			}
		}
	}
}

// mapOf returns the interior rows of the map panel, the first box on screen.
func mapOf(out string) []string {
	var rows []string
	for _, l := range strings.Split(out, "\n")[1:] {
		if strings.HasPrefix(l, "└") {
			break
		}
		rows = append(rows, strings.Split(l, "│")[1])
	}
	return rows
}

func withPos(m *model.Model, x, y int64) *model.Model {
	m.Self.Pos = &pb.Pos{X: x, Y: y}
	return m
}

// The viewport centre is snapped to a cell: a step smaller than a cell does not
// move the picture, and the player stays at its centre.
func TestMapDoesNotJitter(t *testing.T) {
	c := screen.Cases[0]
	draw := func(x int64) string {
		m := withPos(replay(t, c), x, 0)
		return strings.Join(mapOf(screen.Render(m, screen.View{}, 80, 24, screen.Plain())), "\n")
	}
	base := draw(0)
	if draw(300) != base {
		t.Error("a 0.3 m step moved the map")
	}
	rows := strings.Split(base, "\n")
	if !strings.Contains(rows[len(rows)/2], "@") {
		t.Error("the player is not on the middle row")
	}
	if draw(5000) == base {
		t.Error("a 5 m step did not move the map")
	}
}

func TestTargetIsUpperCaseOfItsGlyph(t *testing.T) {
	m := replay(t, screen.Cases[0])
	out := screen.Render(m, screen.View{}, 80, 24, screen.Plain())
	if !strings.Contains(strings.Join(mapOf(out), "\n"), "M") || !strings.Contains(out, "Target: Shell Rat") {
		t.Errorf("target not marked:\n%s", out)
	}
	m.Self.Target = ""
	out = screen.Render(m, screen.View{}, 80, 24, screen.Plain())
	if strings.Contains(strings.Join(mapOf(out), "\n"), "M") {
		t.Error("an untargeted mob is drawn as a target")
	}
}

// A denial's reason is always on the screen (Pillar 8), even in a long line.
func TestDenialReasonIsShown(t *testing.T) {
	for _, c := range screen.Cases {
		if c.Name != "kill_steal" {
			continue
		}
		m := replay(t, c)
		for _, sz := range screen.Sizes {
			out := c.Draw(m, sz[0], sz[1])
			if !strings.Contains(out, "! ") || strings.Count(out, "encounter_owned") < 2 {
				t.Errorf("at %v the reason is missing:\n%s", sz, out)
			}
		}
	}
}

func caseNamed(t *testing.T, name string) screen.Case {
	t.Helper()
	for _, c := range screen.Cases {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no case %s", name)
	return screen.Case{}
}

// The Round panel counts down to the server's deadline and shows who has
// submitted and whether the plan was accepted.
func TestRoundPanel(t *testing.T) {
	c := caseNamed(t, "round_plan")
	m := replay(t, c)
	deadline := time.UnixMilli(m.Round.DeadlineMs)
	draw := func(left time.Duration, sent bool, w, h int) string {
		v := screen.View{Conn: screen.ConnOnline, Now: deadline.Add(-left), Plan: screen.Plan{Steps: c.Plan, Sent: sent}}
		return screen.Render(m, v, w, h, screen.Plain())
	}
	for _, tc := range []struct {
		left time.Duration
		want string
	}{{20 * time.Second, "Round 2, 20 s left"}, {1500 * time.Millisecond, "Round 2, 2 s left"}, {0, "Round 2, deadline now"}, {-time.Second, "Round 2, deadline now"}} {
		if out := draw(tc.left, false, 80, 24); !strings.Contains(out, tc.want) {
			t.Errorf("left %v: want %q\n%s", tc.left, tc.want, out)
		}
	}
	if out := draw(5*time.Second, false, 140, 45); !strings.Contains(out, "Plan (4/16): not sent yet") || !strings.Contains(out, "Enter submits") {
		t.Errorf("unsent:\n%s", out)
	}
	if out := draw(5*time.Second, true, 140, 45); !strings.Contains(out, "Plan (4/16): accepted") || !strings.Contains(out, "counts even if you disconnect") {
		t.Errorf("sent:\n%s", out)
	}
	// Told again after a reconnect with our plan among the submitted: accepted,
	// though this screen never held the plan.
	m.Round.Submitted = []string{m.Self.ID}
	out := screen.Render(m, screen.View{Conn: screen.ConnOnline, Now: deadline.Add(-time.Second)}, 140, 45, screen.Plain())
	if !strings.Contains(out, "Plans in: you") || !strings.Contains(out, "accepted") {
		t.Errorf("submitted:\n%s", out)
	}
}

// Only a turn-based instance has a Round panel, only an instance (open, entered
// or ended) has an Instance panel, and the world keeps its Nearby panel.
func TestPanelsFollowTheInstance(t *testing.T) {
	for _, tc := range []struct {
		name            string
		instance, round bool
	}{{"live_login", false, false}, {"instance_menu", false, false}, {"instance_opened", true, false}, {"instance_inside", true, false}, {"instance_cleared", true, false}, {"round_plan", true, true}, {"round_missed", true, true}} {
		c := caseNamed(t, tc.name)
		out := c.Draw(replay(t, c), 140, 45)
		if has := strings.Contains(out, "─ Instance ─"); has != tc.instance && !c.Menu {
			t.Errorf("%s: instance panel %v", tc.name, has)
		}
		if has := strings.Contains(out, "─ Round "); has != tc.round {
			t.Errorf("%s: round panel %v", tc.name, has)
		}
		if !c.Menu && !strings.Contains(out, "─ Nearby ─") {
			t.Errorf("%s: no Nearby panel at 140x45", tc.name)
		}
	}
}

func TestMenuListsEveryPresetWithItsTimeMode(t *testing.T) {
	c := caseNamed(t, "instance_menu")
	out := c.Draw(replay(t, c), 80, 24)
	for _, want := range []string{"Scuttler Skirmish", "real-time; party 2", "Solo Trial", "party 1; heroic", "Long Game", "100-tick rounds (10 s), 24 h to plan", "Quick Turns", "20-tick rounds (2 s), 20 s to plan", "Duo Turns", "6 s to plan", "> 4"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}
