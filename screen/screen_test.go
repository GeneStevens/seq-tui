package screen_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
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
