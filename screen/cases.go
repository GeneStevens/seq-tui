package screen

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// A Case is one golden screen: a recorded log replayed into the model up to a
// point, drawn at each Size. The goldens live in testdata/screens and change
// only through `make golden` (cmd/mkscreens), never in a test.
type Case struct {
	Name string
	Log  string // file in testdata/logs
	Upto int    // messages replayed; 0 is all of them
	Help bool
	// Notice is the last command's answer shown on the status line.
	Notice    string
	NoticeBad bool
	// Menu draws the instance menu with the cursor on MenuSel.
	Menu    bool
	MenuSel int
	// Plan is the plan being built (as the Round panel lists it), ending at
	// Dest on the map; Left is how long the round's deadline is away.
	Plan []string
	Dest *[2]int64
	Left time.Duration
}

// Sizes the goldens are drawn at.
var Sizes = [][2]int{{80, 24}, {140, 45}}

// Cases are the golden screens.
var Cases = []Case{
	{Name: "fight_mid", Log: "fight_loot.jsonl", Upto: 10},
	{Name: "fight_loot_end", Log: "fight_loot.jsonl", Notice: "loot accepted"},
	{Name: "kill_steal", Log: "kill_steal.jsonl", Notice: "attack denied: encounter_owned", NoticeBad: true},
	{Name: "crossing_before", Log: "crossing.jsonl", Upto: 4},
	{Name: "crossing_after", Log: "crossing.jsonl"},
	{Name: "drop_resume", Log: "drop_resume.jsonl"},
	{Name: "live_login", Log: "live_login.jsonl"},
	// Recorded from a live development server (TUI-006): dying, releasing,
	// respawning, and crossing the zone line to the second zone and back.
	{Name: "release_dead", Log: "live_release.jsonl", Upto: 309},
	{Name: "release_released", Log: "live_release.jsonl", Upto: 321},
	{Name: "release_respawned", Log: "live_release.jsonl"},
	{Name: "door_before", Log: "live_door.jsonl", Upto: 681},
	{Name: "door_across", Log: "live_door.jsonl", Upto: 684},
	{Name: "door_back", Log: "live_door.jsonl"},
	// Recorded from a live development server (TUI-007): the instance menu,
	// solo-trial opened, entered and cleared with the hand-back, and
	// quick-turns round by round.
	{Name: "instance_menu", Log: "live_solo_trial.jsonl", Upto: 3, Menu: true, MenuSel: 3},
	{Name: "instance_opened", Log: "live_solo_trial.jsonl", Upto: 386, Notice: "instance requested; the log says when it is open"},
	{Name: "instance_inside", Log: "live_solo_trial.jsonl", Upto: 395},
	{Name: "instance_cleared", Log: "live_solo_trial.jsonl"},
	{Name: "round_plan", Log: "live_quick_turns.jsonl", Upto: 391, Left: 14 * time.Second,
		Plan: []string{"move (100000,87000)", "move (100000,84000)", "target Plain Scuttler", "attack on"}, Dest: &[2]int64{100000, 84000}},
	{Name: "round_resolved_own", Log: "live_quick_turns.jsonl", Upto: 457},
	{Name: "round_missed", Log: "live_quick_turns.jsonl", Upto: 478},
	{Name: "round_reconnected", Log: "live_quick_turns.jsonl", Upto: 462, Left: 15 * time.Second},
	{Name: "help", Log: "fight_loot.jsonl", Upto: 10, Help: true},
}

// Replay reads the log named by c from dir (testdata/logs) into a new model.
func (c Case) Replay(dir string) (*model.Model, error) {
	f, err := os.Open(filepath.Join(dir, c.Log))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	msgs, err := record.Read(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", c.Log, err)
	}
	if c.Upto > 0 && c.Upto < len(msgs) {
		msgs = msgs[:c.Upto]
	}
	m := model.New()
	for _, msg := range msgs {
		m.Apply(msg)
	}
	return m, nil
}

// Draw renders the case at w x h, plain, with the clock fixed.
func (c Case) Draw(m *model.Model, w, h int) string {
	v := View{Help: c.Help, Conn: ConnOnline, HasTick: true, Since: 400 * time.Millisecond, Notice: c.Notice, NoticeBad: c.NoticeBad, Plan: Plan{Steps: c.Plan}}
	if c.Menu {
		v.Menu = &Menu{Sel: c.MenuSel}
	}
	if c.Dest != nil {
		v.Plan.Dest = &pb.Pos{X: c.Dest[0], Y: c.Dest[1]}
	}
	if m.Round != nil { // the clock stands where the case says: Left before the deadline
		v.Now = time.UnixMilli(m.Round.DeadlineMs).Add(-c.Left)
	}
	return Render(m, v, w, h, Plain())
}

// File is the golden file name for the case at w x h.
func (c Case) File(w, h int) string { return fmt.Sprintf("%s_%dx%d.txt", c.Name, w, h) }
