package screen

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
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
	return Render(m, View{Help: c.Help, Conn: ConnOnline, HasTick: true, Since: 400 * time.Millisecond, Notice: c.Notice, NoticeBad: c.NoticeBad}, w, h, Plain())
}

// File is the golden file name for the case at w x h.
func (c Case) File(w, h int) string { return fmt.Sprintf("%s_%dx%d.txt", c.Name, w, h) }
