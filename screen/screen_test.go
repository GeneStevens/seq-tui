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
			if !strings.Contains(out, "! ") || !strings.Contains(out, "else's fight") {
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
	}{{20 * time.Second, "Round 1, 20 s left"}, {1500 * time.Millisecond, "Round 1, 2 s left"}, {0, "Round 1, deadline now"}, {-time.Second, "Round 1, deadline now"}} {
		if out := draw(tc.left, false, 80, 24); !strings.Contains(out, tc.want) {
			t.Errorf("left %v: want %q\n%s", tc.left, tc.want, out)
		}
	}
	if out := draw(5*time.Second, false, 140, 50); !strings.Contains(out, "Plan (2/16): not sent yet") || !strings.Contains(out, "Enter submits") {
		t.Errorf("unsent:\n%s", out)
	}
	if out := draw(5*time.Second, true, 140, 50); !strings.Contains(out, "Plan (2/16): accepted") || !strings.Contains(out, "counts even if you disconnect") {
		t.Errorf("sent:\n%s", out)
	}
	// Told again after a reconnect with our plan among the submitted: accepted,
	// though this screen never held the plan.
	m.Round.Submitted = []string{m.Self.ID}
	out := screen.Render(m, screen.View{Conn: screen.ConnOnline, Now: deadline.Add(-time.Second)}, 140, 45, screen.Plain())
	if !strings.Contains(out, "1 of 1 submitted") || !strings.Contains(out, "accepted") {
		t.Errorf("submitted:\n%s", out)
	}
}

// Only a turn-based instance has a Round panel, only an instance (open, entered
// or ended) has an Instance panel, and the world keeps its Nearby panel.
func TestPanelsFollowTheInstance(t *testing.T) {
	for _, tc := range []struct {
		name            string
		instance, round bool
	}{{"play_login", false, false}, {"instance_menu", false, false}, {"instance_opened", true, false}, {"instance_inside", true, false}, {"instance_cleared", true, false}, {"round_plan", true, true}, {"round_missed", true, true}} {
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
	out := c.Draw(replay(t, c), 140, 45)
	for _, want := range []string{"Scuttler Skirmish", "real-time", "party 2; normal", "Solo Trial", "party 1; normal", "Venture Den", "Peril Den", "party 2; heroic", "Long Game", "100-tick rounds (10 s), 24 h to plan", "Quick Turns", "20-tick rounds (2 s), 20 s to plan", "Duo Turns", "6 s to plan", "> 2"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

// D49: the menu shows each turn-based preset's sleep limit, and not a
// real-time preset's, which has none.
func TestMenuShowsTheSleepLimit(t *testing.T) {
	c := caseNamed(t, "instance_menu")
	out := c.Draw(replay(t, c), 140, 45)
	if n := strings.Count(out, "sleeps up to 7 d"); n != 3 {
		t.Errorf("%d presets show a sleep limit, want the 3 turn-based ones:\n%s", n, out)
	}
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "party 2; heroic") && strings.Contains(l, "sleeps") {
			t.Errorf("a real-time preset has a sleep limit: %q", l)
		}
	}
}

// D49: the Instance panel says what the server says the instance is, and the
// Round panel says "N of M submitted", counting the character's own receipt.
func TestInstanceAndRoundPanelsShowTheServersFields(t *testing.T) {
	c := caseNamed(t, "duo_one_of_two")
	out := c.Draw(replay(t, c), 80, 24)
	for _, want := range []string{"Duo Turns", "turn-based | inside", "party up to 2 | normal", "1 of 2 submitted", "Plan (1/16): accepted"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if out := c.Draw(replay(t, c), 140, 45); !strings.Contains(out, "sleeps up to 7 d") {
		t.Errorf("no sleep limit at 140x45:\n%s", out)
	}
	c = caseNamed(t, "duo_gathered")
	if out := c.Draw(replay(t, c), 80, 24); !strings.Contains(out, "0 of 2 submitted") {
		t.Errorf("gathered:\n%s", out)
	}
	c = caseNamed(t, "instance_inside")
	if out := c.Draw(replay(t, c), 140, 45); !strings.Contains(out, "real-time | inside") || !strings.Contains(out, "party up to 1 | normal") || strings.Contains(out, "sleeps") {
		t.Errorf("solo-trial:\n%s", out)
	}
	// The panel does not read the space's name.
	m := replay(t, c)
	m.Space = "Somewhere (Else)"
	if out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain()); !strings.Contains(out, "party up to 1 | normal") {
		t.Errorf("the panel changed with the space's name:\n%s", out)
	}
	// A resolved round shows who played what.
	m = replay(t, caseNamed(t, "duo_two_of_two"))
	m.Round.Open = false
	if out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain()); !strings.Contains(out, "2 of 2 submitted") || !strings.Contains(out, "own plan") {
		t.Errorf("resolved:\n%s", out)
	}
}

// The ability bar names each slot's ability and says in a word whether it can
// be used now; the You panel has the mana, the XP and the cast.
func TestAbilityBarAndYouPanelShowTheKit(t *testing.T) {
	c := caseNamed(t, "cast_root")
	out := c.Draw(replay(t, c), 140, 45)
	for _, want := range []string{"Abilities (keys 1-8", "1 Root", "10mp busy", "2 Frost Bolt", "3 Flame Dart", "8 -", "Zedd  Lv 1 Wizard", "MP 150/150", "XP ", "40 to Lv 2", "Cast Root"} {
		if !strings.Contains(out, want) {
			t.Errorf("cast in progress: missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "empty") {
		t.Errorf("an empty slot says \"empty\":\n%s", out)
	}
	c = caseNamed(t, "rooted")
	out = c.Draw(replay(t, c), 80, 24)
	for _, want := range []string{"MP 140/150", "ready", "Plain Scuttler: Rooted 18s", "You: none"} {
		if !strings.Contains(out, want) {
			t.Errorf("rooted: missing %q\n%s", want, out)
		}
	}
	// A character with no abilities has no bar.
	c = caseNamed(t, "play_login")
	if out := c.Draw(replay(t, c), 80, 24); strings.Contains(out, "Abilities") || strings.Contains(out, "MP ") {
		t.Errorf("a character with no kit has a bar:\n%s", out)
	}
	// A Warrior has no mana line.
	c = caseNamed(t, "warrior_new")
	if out := c.Draw(replay(t, c), 80, 24); strings.Contains(out, "MP ") || !strings.Contains(out, "Lv 1 Warrior") {
		t.Errorf("warrior:\n%s", out)
	}
}

// Each slot says what stops it: a cooldown, mana, the target or the range.
func TestSlotsSayWhatStopsThem(t *testing.T) {
	m := replay(t, caseNamed(t, "kit_slotted"))
	draw := func() string { return screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain()) }
	if out := draw(); strings.Count(out, "tgt?") != 3 {
		t.Errorf("no target:\n%s", out)
	}
	m.Self.Target = "mob-near-1"
	for _, id := range m.EntityIDs() {
		m.Self.Target = id
	}
	if out := draw(); strings.Contains(out, "tgt?") || strings.Count(out, "ready") != 3 {
		t.Errorf("a target in range:\n%s", out)
	}
	m.Self.Mana = 8
	if out := draw(); strings.Count(out, "mana") != 2 { // Flame Dart costs 6 and still can be cast; Root and Frost Bolt cannot
		t.Errorf("not enough mana:\n%s", out)
	}
}

func TestCastEndedLineShowsWhy(t *testing.T) {
	for name, want := range map[string]string{"cast_moved": "Cancelled: you moved", "cast_interrupted": "Interrupted: damage"} {
		c := caseNamed(t, name)
		out := c.Draw(replay(t, c), 80, 24)
		if !strings.Contains(out, "! "+want) {
			t.Errorf("%s: missing %q\n%s", name, want, out)
		}
		if out := c.Draw(replay(t, c), 140, 45); !strings.Contains(out, "Frost Bolt") || !strings.Contains(out, "! Frost Bolt") {
			t.Errorf("%s at 140x45:\n%s", name, out)
		}
	}
}

func TestSlotScreenAndCreationScreen(t *testing.T) {
	c := caseNamed(t, "slots_pick")
	out := c.Draw(replay(t, c), 80, 24)
	for _, want := range []string{"Ability slots", "> 1  Flame Dart", "Choose the ability for slot 1", "> Root", "(empty)"} {
		if !strings.Contains(out, want) {
			t.Errorf("slot screen: missing %q\n%s", want, out)
		}
	}
	c = caseNamed(t, "create_class")
	m := replay(t, c)
	out = c.Draw(m, 80, 24)
	if len(m.Classes()) != 6 {
		t.Fatalf("the welcome's class menu has %d classes", len(m.Classes()))
	}
	for _, cl := range m.Classes() {
		if !strings.Contains(out, cl.GetName()) {
			t.Errorf("creation: no %s\n%s", cl.GetName(), out)
		}
	}
	if !strings.Contains(out, "Name   Zedd") || !strings.Contains(out, "> 4  Wizard") {
		t.Errorf("creation:\n%s", out)
	}
}

func TestLevelUpIsInTheLog(t *testing.T) {
	c := caseNamed(t, "levelup")
	for _, sz := range screen.Sizes {
		out := c.Draw(replay(t, c), sz[0], sz[1])
		if !strings.Contains(out, "You reached level 2") || !strings.Contains(out, "Lv 2 Wizard") {
			t.Errorf("at %v:\n%s", sz, out)
		}
	}
}

// A turn-based instance on a small screen keeps the plan: the Round panel comes
// before the Instance panel when the You panel is tall.
func TestPlanSurvivesASmallScreenWithTheAbilityBar(t *testing.T) {
	c := caseNamed(t, "turn_cast_plan")
	out := c.Draw(replay(t, c), 80, 24)
	for _, want := range []string{"Round 1", "Plan (2/16)"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
	out = c.Draw(replay(t, c), 80, 26) // the line of key hints costs the 80x24 screen a row
	for _, want := range []string{"use Frost Bolt (key 2)"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
	if out := c.Draw(replay(t, c), 140, 45); !strings.Contains(out, "use Frost Bolt (key 2)") || !strings.Contains(out, "─ Instance ─") {
		t.Errorf("140x45:\n%s", out)
	}
}

// The new screens say what they are for, at both sizes.
func TestEconomyScreens(t *testing.T) {
	for _, tc := range []struct {
		name  string
		wants []string
	}{
		{"class_menu", []string{"Class (the server's menu)", "> 6  Druid", "healer + spell", "Starts with, free:", "Thorn Bolt"}},
		{"character_bags", []string{"Your character", "Items carried", "Scuttler Blade", "Worn Cap", "Crowns 5"}},
		{"character_gear", []string{"Worn Cap Common", "Main hand", "Scuttler Blade", "tradable once"}},
		{"vendor", []string{"Testgrounds Vendor", "For sale (Enter buys one)", "Worn Cap", "Padded Vest", "You carry (Enter sells one)"}},
		{"trainer", []string{"Wizard Trainer", "Glyphshield", "45 crowns", "known"}},
		{"bank", []string{"In the bank, 1", "Buckler"}},
		{"trade_ask", []string{"Trade with another character in reach", "> c2"}},
		{"trade_window", []string{"Your offer", "c2's offer", "crowns 7", "Worn Cap", "[bound]", "not confirmed"}},
		{"peril_menu", []string{"Peril Den", "party 2; heroic", "entry fee"}},
		{"login_rooted", []string{"Plain Scuttler: Rooted"}},
		{"slots_plan", []string{"Ability slots", "(in the plan)", "joins the plan"}},
		{"turn_cast_round", []string{"Frost Bolt lands in round 2"}},
		{"duo_one_of_two", []string{"[x] you  [ ] bram"}},
	} {
		c := caseNamed(t, tc.name)
		m := replay(t, c)
		for _, sz := range screen.Sizes {
			out := c.Draw(m, sz[0], sz[1])
			for _, want := range tc.wants {
				if sz[0] == 80 && tc.name == "turn_cast_round" && want == "Frost Bolt lands in round 2" {
					want = "lands in" // the panel is narrow
				}
				if !strings.Contains(out, want) {
					t.Errorf("%s at %v: missing %q\n%s", tc.name, sz, want, out)
				}
			}
		}
	}
}

// A drop the character may not take is told apart on the map and in Nearby.
func TestDropsYouMayNotTakeAreMarked(t *testing.T) {
	c := caseNamed(t, "play_looted")
	m := replay(t, c)
	m.Items["scuttler-blade"] = &pb.ItemInfo{Id: "scuttler-blade", Name: "Scuttler Blade", Tier: "rare", BindOnPickup: true}
	m.Drops = map[string]*pb.LootDropState{
		"d1": {Id: "d1", Item: "scuttler-blade", Pos: &pb.Pos{X: m.Self.Pos.GetX() + 4000, Y: m.Self.Pos.GetY()}},
		"d2": {Id: "d2", Crowns: 5, CanLoot: true, Pos: &pb.Pos{X: m.Self.Pos.GetX() - 4000, Y: m.Self.Pos.GetY()}},
	}
	out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain())
	for _, want := range []string{"~ Scuttler Blade", "not lootable", "* 5 crowns", "~ not yours"} {
		if want == "~ not yours" {
			out = screen.Render(m, screen.View{Conn: screen.ConnOnline, Help: true, HelpPage: 1}, 140, 45, screen.Plain())
		}
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

// The Round panel's title fits the panel at 80 columns, whatever the round ran
// by: it is not cut with an ellipsis.
func TestRoundTitleFitsAt80Columns(t *testing.T) {
	seen := 0
	for _, c := range screen.Cases {
		for _, l := range strings.Split(c.Draw(replay(t, c), 80, 24), "\n") {
			if strings.Contains(l, "┌─ Round") {
				seen++
				if strings.Contains(l, "…") {
					t.Errorf("%s: the Round title is cut: %q", c.Name, l)
				}
			}
		}
	}
	if seen == 0 {
		t.Fatal("no case shows a Round panel")
	}
}

// Under attack, every screen keeps its size: the red border, the vitals strip
// and the status line's alarm change no row or column count (D75).
func TestScreensUnderAttackFitTheirSize(t *testing.T) {
	c := caseNamed(t, "vendor")
	m := replay(t, c)
	m.Self.HP, m.Self.MaxHP = 90, 100
	m.Self.InCombat, m.Self.Attackers = true, []string{"mob-1"}
	m.Entities["mob-1"] = &pb.EntityState{Id: "mob-1", Name: "Hedge Brigand", Kind: pb.EntityKind_ENTITY_KIND_MOB, Level: 4, Hp: 30, MaxHp: 40, Con: "hard", Pos: &pb.Pos{X: m.Self.Pos.GetX() + 4000, Y: m.Self.Pos.GetY()}}
	m.Self.Target = "mob-1"
	m.Alert = &model.Alert{Mob: "mob-1", Name: "Hedge Brigand", Con: "hard", Level: 4, Danger: true}
	views := map[string]screen.View{
		"map":       {},
		"inventory": {Char: true},
		"traders":   {Traders: &screen.Traders{}},
		"trade":     {Trade: &screen.TradeUI{}},
		"talk":      {Talk: &screen.Talk{NPC: m.NPCs[0].GetId()}},
		"board":     {Board: &screen.Board{NPC: m.NPCs[0].GetId()}},
		"contracts": {Contracts: &screen.ContractsUI{}},
		"slots":     {Slots: &screen.Slots{}},
		"menu":      {Menu: &screen.Menu{}},
		"help":      {Help: true},
		"overview":  {Overview: true},
		"quest":     {Quest: &screen.QuestUI{}},
		"notice":    {Char: true, Notice: "You can't do that while you're fighting", NoticeBad: true},
	}
	for name, v := range views {
		v.Conn = screen.ConnOnline
		for _, sz := range [][2]int{{80, 24}, {81, 25}, {100, 30}, {120, 40}, {140, 45}, {200, 60}, {80, 60}, {250, 24}} {
			out := screen.Render(m, v, sz[0], sz[1], screen.Plain())
			lines := strings.Split(out, "\n")
			if len(lines) != sz[1] {
				t.Errorf("%s at %v: %d lines", name, sz, len(lines))
			}
			for i, l := range lines {
				if n := utf8.RuneCountInString(l); n != sz[0] {
					t.Errorf("%s at %v: line %d is %d cells: %q", name, sz, i, n, l)
				}
			}
			last := lines[len(lines)-1]
			if !strings.Contains(last, "⚠ DANGER: Hedge Brigand (level 4, hard)") {
				t.Errorf("%s at %v: no danger alarm on the status line: %q", name, sz, last)
			}
			if name != "map" && name != "overview" && name != "help" && sz[0] >= 80 && !strings.Contains(out, "UNDER ATTACK") && name != "slots" && name != "menu" && name != "quest" {
				t.Errorf("%s at %v: the border does not say it is under attack:\n%s", name, sz, out)
			}
		}
	}
}

// A hard or deadly mob that begins an attack is a danger on the map screen too;
// an ordinary one is not an alarm there, though it is on any menu.
func TestTheAlarmOnTheMapIsForDanger(t *testing.T) {
	c := caseNamed(t, "vendor")
	m := replay(t, c)
	m.Self.InCombat, m.Self.Attackers = true, []string{"mob-1"}
	m.Alert = &model.Alert{Mob: "mob-1", Name: "Plain Scuttler", Con: "even", Level: 1}
	m.Entities["mob-1"] = &pb.EntityState{Id: "mob-1", Name: "Plain Scuttler", Kind: pb.EntityKind_ENTITY_KIND_MOB, Pos: m.Self.Pos}
	status := func(v screen.View) string {
		rows := strings.Split(screen.Render(m, v, 140, 45, screen.Plain()), "\n")
		return rows[len(rows)-1]
	}
	if s := status(screen.View{Conn: screen.ConnOnline}); strings.Contains(s, "⚠") {
		t.Errorf("an even mob raises the alarm on the map: %q", s)
	}
	if s := status(screen.View{Conn: screen.ConnOnline, Char: true}); !strings.Contains(s, "⚠ Plain Scuttler is attacking you") || !strings.Contains(s, "Esc to fight") {
		t.Errorf("a menu under attack: %q", s)
	}
	m.Alert.Danger, m.Alert.Con = true, "deadly"
	if s := status(screen.View{Conn: screen.ConnOnline}); !strings.Contains(s, "⚠ DANGER: Plain Scuttler (level 1, deadly) is attacking you") || !strings.Contains(s, "F to fight") {
		t.Errorf("a deadly mob on the map: %q", s)
	}
}

// The place is in the map's title and on the status line; the objective names
// itself, its direction and its distance.
func TestPlaceAndObjectiveAreNamed(t *testing.T) {
	c := caseNamed(t, "vendor")
	m := replay(t, c)
	m.Self.Place = "Trainers' Lane"
	sp := m.Self.Pos
	m.Self.Hints = []*pb.Hint{
		{Kind: "objective", Id: "x", Label: "Talk to Gorran Stoutarm", Pos: &pb.Pos{X: sp.GetX() - 60000, Y: sp.GetY()}},
		{Kind: "waypoint", Id: "x", Label: "Talk to Gorran Stoutarm", Pos: &pb.Pos{X: sp.GetX() - 60000, Y: sp.GetY()}},
	}
	out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain())
	rows := strings.Split(out, "\n")
	if !strings.Contains(rows[0], "— Trainers' Lane") {
		t.Errorf("the place is not in the map's title: %q", rows[0])
	}
	if last := rows[len(rows)-1]; !strings.Contains(last, "Trainers' Lane") || !strings.Contains(last, "→ Talk to Gorran Stoutarm, 60m W") {
		t.Errorf("status line: %q", last)
	}
	if !regexp.MustCompile(`◄ Gorran 60m`).MatchString(out) {
		t.Errorf("no pointer to the objective at the map's edge:\n%s", out)
	}
}

// Every role has a glyph of its own, the marks come from the hints, and the
// legend names them all.
func TestRoleGlyphsMarksAndLegend(t *testing.T) {
	m := model.New()
	pos := func(x int64) *pb.Pos { return &pb.Pos{X: 10000 + x, Y: 10000} }
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "Z", Seq: 1, GameTimeMs: 1000,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Pos: pos(0), Hp: 10, MaxHp: 10},
			Hints: []*pb.Hint{
				{Kind: "offer", Id: "trainer", Label: "Talk to Gorran", Pos: pos(2000)},
				{Kind: "ready", Id: "vendor", Label: "Hand in: X", Pos: pos(4000)},
				{Kind: "in_progress", Id: "banker", Label: "In progress: Y", Note: "2 of 3", Pos: pos(6000)},
				{Kind: "offer", Id: "board", Label: "New contracts", Note: "2 to take", Pos: pos(8000)},
			}},
		Npcs: []*pb.NpcInfo{
			{Id: "trainer", Name: "Gorran Stoutarm", Title: "Warrior Trainer", Role: "trainer", ReachMm: 5000, Pos: pos(2000)},
			{Id: "vendor", Name: "Hilda", Title: "Vendor", Role: "vendor", ReachMm: 5000, Pos: pos(4000)},
			{Id: "banker", Name: "Bertram", Title: "Banker", Role: "banker", ReachMm: 5000, Pos: pos(6000)},
			{Id: "board", Name: "Notice Board", Title: "Contract Board", Role: "contract_board", ReachMm: 5000, Pos: pos(8000)},
			{Id: "binder", Name: "Isolde", Title: "Binder", Role: "binder", ReachMm: 5000, Pos: pos(-2000)},
			{Id: "guard", Name: "Watchman", Title: "Guard", Role: "guard", ReachMm: 5000, Pos: pos(-4000)},
			{Id: "quester", Name: "Captain", Title: "Quest Giver", Role: "quest_giver", ReachMm: 5000, Pos: pos(-6000)},
		}}}})
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: "Z", Name: "Z", ViewRadiusMm: 20000,
		Bounds: &pb.Rect{MinX: -20000, MinY: -20000, MaxX: 40000, MaxY: 40000}}}})
	out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain())
	mapRows := strings.Join(strings.Split(out, "\n")[1:20], "\n")
	var row string
	for _, l := range strings.Split(mapRows, "\n") {
		if strings.Contains(l, "@") {
			row = l
		}
	}
	// From the west: quest person, guard, binder, you, then the marked trainer (!), vendor (?), banker (?), board (&).
	order := regexp.MustCompile(`Q\s+g\s+H\s+@\s+!\s+\?\s+\?\s+&`)
	if !order.MatchString(row) {
		t.Errorf("the people are not drawn by role and mark on the map row %q", row)
	}
	nearby := out[strings.Index(out, "Nearby"):]
	for _, want := range []string{"[hand in]", "[2/3]", "[new]"} {
		if !strings.Contains(nearby, want) {
			t.Errorf("Nearby lacks the tag %q:\n%s", want, nearby)
		}
	}
	// Quest-relevant entries come first, then the rest by distance.
	i1, i2 := strings.Index(nearby, "Gorran Stoutarm"), strings.Index(nearby, "Isolde")
	if i1 < 0 || i2 < 0 || i1 > i2 {
		t.Errorf("Nearby does not list the marked trainer before the nearer binder:\n%s", nearby)
	}
	help := screen.Render(m, screen.View{Conn: screen.ConnOnline, Help: true, HelpPage: 1}, 140, 45, screen.Plain())
	for _, want := range []string{"T trainer", "$ vendor", "B banker", "H binder", "g guard", "Q quest person", "& board", "! new", "? hand in", "? in progress", "c camp", "░ zone edge"} {
		if !strings.Contains(help, want) {
			t.Errorf("the legend lacks %q:\n%s", want, help)
		}
	}
	over := screen.Render(m, screen.View{Conn: screen.ConnOnline, Overview: true}, 140, 45, screen.Plain())
	for _, want := range []string{"T trainer", "░ zone edge", "D next door"} {
		if !strings.Contains(over, want) {
			t.Errorf("the zone map's legend lacks %q:\n%s", want, over)
		}
	}
}

// The line of key hints changes with what the character stands next to (D76).
func TestKeyHintsFollowWhatIsNear(t *testing.T) {
	hint := func(m *model.Model) string {
		rows := strings.Split(screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain()), "\n")
		return strings.TrimSpace(rows[len(rows)-2])
	}
	base := func(mut func(s *pb.Snapshot)) *model.Model {
		m := model.New()
		s := &pb.Snapshot{Space: "Z", Seq: 1, GameTimeMs: 1000, Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Pos: &pb.Pos{X: 10000, Y: 10000}, Hp: 10, MaxHp: 10}}}
		mut(s)
		m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: s}})
		m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: &pb.ZoneMap{Space: "Z", Name: "Z", ViewRadiusMm: 20000,
			Doors: []*pb.Door{{Id: "east-gate", Kind: pb.DoorKind_DOOR_KIND_ZONE_LINE, Pos: &pb.Pos{X: 12000, Y: 10000}, Name: "East Gate", Label: "East Gate — to the Thornwick Downs"}}}}})
		return m
	}
	mob := &pb.EntityState{Id: "mob-1", Name: "Meadow Wolf", Kind: pb.EntityKind_ENTITY_KIND_MOB, Hp: 20, MaxHp: 20, Pos: &pb.Pos{X: 18000, Y: 10000}}
	for _, tc := range []struct {
		name string
		m    *model.Model
		want string
	}{
		{"door", base(func(*pb.Snapshot) {}), "E: go through the East Gate to the Thornwick Downs"},
		{"person", base(func(s *pb.Snapshot) {
			s.Npcs = []*pb.NpcInfo{{Id: "t", Name: "Gorran Stoutarm", Title: "Warrior Trainer", Role: "trainer", ReachMm: 5000, Pos: &pb.Pos{X: 9000, Y: 10000}}}
		}), "G: talk to Gorran Stoutarm <Warrior Trainer>"},
		{"drop", base(func(s *pb.Snapshot) {
			s.Drops = []*pb.LootDropState{{Id: "d", Item: "worn-cap", CanLoot: true, Pos: &pb.Pos{X: 11000, Y: 10000}}}
			s.Items = []*pb.ItemInfo{{Id: "worn-cap", Name: "Worn Cap"}}
		}), "L: loot Worn Cap"},
		{"a mob in view", base(func(s *pb.Snapshot) { s.Entities = []*pb.EntityState{mob} }), "Tab: target · F: attack"},
		{"dead", base(func(s *pb.Snapshot) { s.Self.Entity.Dead = true }), "R: release"},
		{"nothing", base(func(s *pb.Snapshot) {
			s.Self.Entity.Pos = &pb.Pos{X: 10000, Y: 40000}
		}), "Walk: click or arrows"},
	} {
		if got := hint(tc.m); !strings.Contains(got, tc.want) {
			t.Errorf("%s: the hint line says %q, want %q", tc.name, got, tc.want)
		}
	}
	// The hint for a door and a person together, and nothing for a drop the character may not take.
	both := base(func(s *pb.Snapshot) {
		s.Npcs = []*pb.NpcInfo{{Id: "t", Name: "Gorran Stoutarm", Title: "Warrior Trainer", Role: "trainer", ReachMm: 5000, Pos: &pb.Pos{X: 9000, Y: 10000}}}
		s.Drops = []*pb.LootDropState{{Id: "d", Item: "worn-cap", CanLoot: false, Pos: &pb.Pos{X: 11000, Y: 10000}}}
	})
	if got := hint(both); !strings.Contains(got, "E: go through") || !strings.Contains(got, "G: talk to") || strings.Contains(got, "L: loot") {
		t.Errorf("door and person: %q", got)
	}
}

// The level-up banner names what can be trained and where (D76), and the
// hand-in summary says what a contract or a quest step paid.
func TestLevelUpBannerAndHandInSummary(t *testing.T) {
	c := caseNamed(t, "vendor")
	m := replay(t, c)
	m.GameTimeMs = 50_000
	m.LevelUp = &model.Moment{AtMs: 49_000, Level: &pb.LevelUp{Level: 3, FromLevel: 2, HpBefore: 120, HpAfter: 148, ManaBefore: 0, ManaAfter: 0,
		Gains:     []*pb.AttributeGain{{Name: "Strength", Amount: 4}},
		Trainable: []*pb.Trainable{{Ability: "warrior-second-wind", Name: "Second Wind", Level: 3, Cost: 45, Trainer: "trainer-warrior", TrainerName: "Gorran Stoutarm", TrainerAt: "Trainers' Lane"}}}}
	for _, sz := range [][2]int{{80, 24}, {140, 45}} {
		out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, sz[0], sz[1], screen.Plain())
		rows := strings.Split(out, "\n")
		if !strings.Contains(rows[0], "LEVEL UP! You reached level 3. Health 120 → 148") || !strings.Contains(rows[1], "You can now train: Second Wind from Gorran Stoutarm") {
			t.Errorf("%v: the banner:\n%s", sz, strings.Join(rows[:3], "\n"))
		}
		if sz[0] == 140 && !strings.Contains(rows[1], "in Trainers' Lane (45 crowns)") {
			t.Errorf("%v: the banner does not say where: %q", sz, rows[1])
		}
		if len(rows) != sz[1] {
			t.Errorf("%v: %d rows", sz, len(rows))
		}
	}
	m.GameTimeMs = 100_000
	if out := screen.Render(m, screen.View{Conn: screen.ConnOnline}, 140, 45, screen.Plain()); strings.Contains(out, "LEVEL UP!") {
		t.Errorf("the banner stays up:\n%s", out)
	}
	// The hand-in summary in the board panel.
	m.GameTimeMs = 50_000
	m.HandIn = &model.Moment{AtMs: 49_500, Hand: &pb.HandIn{Kind: "quest_step", Name: "First Steps", Xp: 20, Crowns: 15, Item: "worn-cap", LevelUp: &pb.LevelUp{Level: 2}}}
	out := screen.Render(m, screen.View{Conn: screen.ConnOnline, Board: &screen.Board{NPC: m.NPCs[0].GetId()}}, 140, 45, screen.Plain())
	if !strings.Contains(out, "Handed in the quest step First Steps: 20 XP, 15 crowns, Worn Cap. You reached level 2!") {
		t.Errorf("no hand-in summary:\n%s", out)
	}
}

// The hierarchy (TUI-029): a panel's title is bold in the accent colour, its keys
// are highlighted over a dim help line, and NO_COLOR keeps bold and dim but no
// colour at all.
func TestHierarchyStylesAndNoColor(t *testing.T) {
	c := caseNamed(t, "vendor")
	m := replay(t, c)
	v := screen.View{Conn: screen.ConnOnline, Traders: &screen.Traders{}}
	ansi := screen.Render(m, v, 100, 30, screen.ANSI())
	if !strings.Contains(ansi, "\x1b[1;36mTraders\x1b[0m") {
		t.Errorf("the panel title is not bold in the accent colour:\n%q", ansi)
	}
	v.Traders = &screen.Traders{Open: true, NPC: m.NPCs[0].GetId()}
	ansi = screen.Render(m, v, 100, 30, screen.ANSI())
	if !strings.Contains(ansi, "\x1b[1m For sale") {
		t.Errorf("the section header is not bold:\n%q", ansi)
	}
	if !strings.Contains(ansi, "\x1b[36mEnter\x1b[0m") || !strings.Contains(ansi, "\x1b[2m") {
		t.Errorf("a help line does not have its keys highlighted over dim text:\n%q", ansi)
	}
	mono := screen.Render(m, v, 100, 30, screen.Mono())
	if !strings.Contains(mono, "\x1b[1mTraders") && !strings.Contains(mono, "\x1b[1m") {
		t.Error("NO_COLOR dropped bold too")
	}
	for _, code := range sgr.FindAllString(mono, -1) {
		if code != "\x1b[0m" && code != "\x1b[1m" && code != "\x1b[2m" && code != "\x1b[7m" {
			t.Errorf("NO_COLOR output has the colour %q", code)
			break
		}
	}
	if got := sgr.ReplaceAllString(mono, ""); got != screen.Render(m, v, 100, 30, screen.Plain()) {
		t.Error("the NO_COLOR screen differs from the plain one once styling is removed")
	}
}

// What cannot be used is marked x with its reason, and a row greyed for the price
// keeps its mark bold inside the dim row.
func TestMarksSurviveGreyingAndNotUsableSaysWhy(t *testing.T) {
	c := caseNamed(t, "vendor")
	m := replay(t, c)
	n := m.NPCs[0]
	if len(n.GetStock()) < 2 {
		t.Fatal("the vendor sells too little")
	}
	item := n.GetStock()[1].GetItem() // the cursor's row: its card says why
	m.Self.Crowns = 0
	other := n.GetStock()[0].GetItem() // greyed, and not under the cursor
	m.Self.Facts = map[string]*pb.ItemFact{
		item:  {Item: item, Verdict: "not_usable", Reason: "class", Need: "mage,cleric"},
		other: {Item: other, Verdict: "upgrade", Slot: "head"},
	}
	v := screen.View{Conn: screen.ConnOnline, Traders: &screen.Traders{Open: true, NPC: n.GetId(), Cur: [2]int{1, 0}}}
	plain := screen.Render(m, v, 140, 45, screen.Plain())
	for _, want := range []string{"x " + m.ItemName(item), "can't use: mage, cleric only"} {
		if !strings.Contains(plain, want) {
			t.Errorf("the screen lacks %q:\n%s", want, plain)
		}
	}
	if ansi := screen.Render(m, v, 140, 45, screen.ANSI()); !strings.Contains(ansi, "\x1b[1m▲\x1b[0m\x1b[2m") {
		t.Errorf("the mark of a greyed row is dimmed with the rest")
	}
}
