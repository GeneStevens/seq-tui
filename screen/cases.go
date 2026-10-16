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
	// Mark replays the messages before the mark of that name instead (a
	// scenario writes the marks when it records the log).
	Mark string
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
	// PlanSent: the server has the plan (the receipt came).
	PlanSent bool
	// Slots draws the slot screen, Create the character creation screen (and an
	// empty log draws it before any character is chosen).
	Slots  *Slots
	Create *Create
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
	// Recorded by the client scenarios (`make fixtures`, scenario/): each log
	// is one scenario's session, and a mark is a point the scenario named.
	{Name: "play_login", Log: "play.jsonl", Mark: "play_login"},
	{Name: "play_fight", Log: "play.jsonl", Mark: "play_fight"},
	{Name: "play_looted", Log: "play.jsonl", Mark: "play_looted", Notice: "loot ok"},
	// Dying, releasing, respawning, and crossing the zone line and back.
	{Name: "release_dead", Log: "release.jsonl", Mark: "release_dead"},
	{Name: "release_released", Log: "release.jsonl", Mark: "release_released", Notice: "release ok"},
	{Name: "release_respawned", Log: "release.jsonl", Mark: "release_respawned"},
	{Name: "door_before", Log: "door.jsonl", Mark: "door_before"},
	{Name: "door_across", Log: "door.jsonl", Mark: "door_across", Notice: "use door ok"},
	{Name: "door_back", Log: "door.jsonl", Mark: "door_back", Notice: "use door ok"},
	// The instance menu, solo-trial opened, entered and cleared with the
	// hand-back, and quick-turns and duo-turns round by round.
	{Name: "instance_menu", Log: "solo_trial.jsonl", Mark: "instance_menu", Menu: true, MenuSel: 1},
	{Name: "instance_opened", Log: "solo_trial.jsonl", Mark: "instance_opened", Notice: "instance requested; the log says when it is open"},
	{Name: "instance_inside", Log: "solo_trial.jsonl", Mark: "instance_inside", Notice: "use door ok"},
	{Name: "instance_cleared", Log: "solo_trial.jsonl", Mark: "instance_cleared", Notice: "loot ok"},
	{Name: "round_plan", Log: "quick_turns.jsonl", Mark: "round_plan", Left: 20 * time.Second,
		Plan: []string{"move (100000,87000)", "target Plain Scuttler"}, Dest: &[2]int64{100000, 87000}},
	{Name: "round_receipt", Log: "quick_turns.jsonl", Mark: "round_receipt", Left: 20 * time.Second, PlanSent: true, Notice: "plan accepted, counts even if you disconnect",
		Plan: []string{"move (100000,87000)", "target Plain Scuttler"}, Dest: &[2]int64{100000, 87000}},
	{Name: "round_missed", Log: "quick_turns.jsonl", Mark: "round_missed", Left: 19 * time.Second},
	{Name: "round_resumed", Log: "quick_turns.jsonl", Mark: "round_resumed", Left: 19 * time.Second},
	{Name: "round_fresh", Log: "quick_turns.jsonl", Mark: "round_fresh", Left: 19 * time.Second},
	{Name: "round_cleared", Log: "quick_turns.jsonl", Mark: "round_cleared", Notice: "plan accepted, counts even if you disconnect"},
	{Name: "duo_gathered", Log: "duo_turns.jsonl", Mark: "duo_gathered", Left: 6 * time.Second},
	{Name: "duo_one_of_two", Log: "duo_turns.jsonl", Mark: "duo_one_of_two", Left: 6 * time.Second, PlanSent: true, Notice: "plan accepted, counts even if you disconnect", Plan: []string{"move (100000,92000)"}},
	{Name: "duo_two_of_two", Log: "duo_turns.jsonl", Mark: "duo_two_of_two", Left: 6 * time.Second},
	// Classes and combat (SEQ-038, SEQ-039): the creation screen, a new Wizard,
	// the slot screen, a cast and the effect it leaves, a cancelled and an
	// interrupted cast, a Warrior, a level-up, and a cast in a turn-based plan.
	{Name: "create_name", Create: &Create{}},
	{Name: "create_class", Create: &Create{Name: "Zedd", Class: 3, Classes: true}},
	{Name: "create_refused", Create: &Create{Name: "zEDD", Class: 1, Err: "someone already has that name (name_taken)"}},
	{Name: "kit_new", Log: "wizard.jsonl", Mark: "kit_new"},
	{Name: "slots_pick", Log: "wizard.jsonl", Mark: "slots_pick", Slots: &Slots{Slot: 0, Pick: true, Cur: 2}},
	{Name: "slots_set", Log: "wizard.jsonl", Mark: "slots_set", Slots: &Slots{Slot: 2}},
	{Name: "kit_slotted", Log: "wizard.jsonl", Mark: "kit_slotted"},
	{Name: "cast_root", Log: "wizard.jsonl", Mark: "cast_root", Notice: "key 1 Root ok"},
	{Name: "rooted", Log: "wizard.jsonl", Mark: "rooted", Notice: "key 1 Root ok"},
	{Name: "nuked", Log: "wizard.jsonl", Mark: "nuked", Notice: "key 2 Frost Bolt ok"},
	{Name: "mana_back", Log: "wizard.jsonl", Mark: "mana_back", Notice: "key 2 Frost Bolt ok"},
	{Name: "cast_moved", Log: "casts.jsonl", Mark: "cast_moved", Notice: "move ok"},
	{Name: "cast_interrupted", Log: "casts.jsonl", Mark: "cast_interrupted", Notice: "key 2 Frost Bolt ok"},
	{Name: "warrior_new", Log: "warrior.jsonl", Mark: "warrior_new"},
	{Name: "warrior_taunt", Log: "warrior.jsonl", Mark: "warrior_taunt", Notice: "key 1 Taunt ok"},
	{Name: "warrior_second_wind", Log: "warrior.jsonl", Mark: "warrior_second_wind", Notice: "key 3 Second Wind ok"},
	{Name: "warrior_killed", Log: "warrior.jsonl", Mark: "warrior_killed", Notice: "key 2 Bash ok"},
	{Name: "levelup", Log: "levelup.jsonl", Mark: "levelup", Notice: "key 2 Frost Bolt ok"},
	{Name: "levelup_ward", Log: "levelup.jsonl", Mark: "levelup_ward", Notice: "key 3 Arcane Ward ok"},
	{Name: "turn_cast_plan", Log: "turn_cast.jsonl", Mark: "turn_cast_plan", Left: 20 * time.Second, Plan: []string{"target Plain Scuttler", "use Frost Bolt (key 2)"}},
	{Name: "turn_cast_round", Log: "turn_cast.jsonl", Mark: "turn_cast_round", Left: 20 * time.Second, Notice: "plan accepted, counts even if you disconnect"},
	{Name: "help", Log: "fight_loot.jsonl", Upto: 10, Help: true},
}

// Replay reads the log named by c from dir (testdata/logs) into a new model.
func (c Case) Replay(dir string) (*model.Model, error) {
	if c.Log == "" {
		return model.New(), nil
	}
	f, err := os.Open(filepath.Join(dir, c.Log))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var msgs []*pb.ServerMessage
	if c.Mark != "" {
		msgs, err = record.ReadUntil(f, c.Mark)
	} else {
		msgs, err = record.Read(f)
	}
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
	v := View{Help: c.Help, Conn: ConnOnline, HasTick: true, Since: 400 * time.Millisecond, Notice: c.Notice, NoticeBad: c.NoticeBad, Plan: Plan{Steps: c.Plan, Sent: c.PlanSent}}
	if c.Menu {
		v.Menu = &Menu{Sel: c.MenuSel}
	}
	v.Slots, v.Create = c.Slots, c.Create
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
