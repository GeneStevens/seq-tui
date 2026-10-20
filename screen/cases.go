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
	// PlanSlots are the slot changes the plan holds; Char (the sidebar state)
	// draws the character view, Traders and Trade the traders' screen and the
	// trade window.
	PlanSlots map[int]string
	Char      *Sidebar
	Traders   *Traders
	Trade     *TradeUI
	// TraderName puts the traders' screen's cursor on the trader with that name.
	TraderName string
	// Talk, Board and Contracts draw the conversation, the contract board and the
	// Contracts screen; Overview the whole-zone map.
	Talk      *Talk
	Board     *Board
	Contracts *ContractsUI
	Overview  bool
	// Role draws the screen for choosing the recruit's role; Duo the account's
	// second character on a stance (its own state is not in the recorded log).
	Role *Role
	Duo  *DuoView
	// Line draws the command line with its text; Social the Social screen, whose
	// who list is the model's Known characters when WhoKnown is set (the answer to
	// a who request is not in the recorded messages' model state).
	Line     *Line
	Social   *SocialUI
	WhoKnown bool
	HelpPage int
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
	// The class menu is the server's, from the welcome the log begins with.
	{Name: "create_name", Log: "wizard.jsonl", Mark: "create_name", Create: &Create{}},
	{Name: "create_class", Log: "wizard.jsonl", Mark: "create_name", Create: &Create{Name: "Zedd", Class: 3, Classes: true}},
	{Name: "create_refused", Log: "wizard.jsonl", Mark: "create_name", Create: &Create{Name: "zEDD", Class: 1, Err: "someone already has that name (name_taken)"}},
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
	{Name: "turn_cast_plan", Log: "turn_cast.jsonl", Mark: "turn_cast_plan", Left: 20 * time.Second, Plan: []string{"target Plain Scuttler", "use Frost Bolt (key 2)"}},
	{Name: "turn_cast_round", Log: "turn_cast.jsonl", Mark: "turn_cast_round", Left: 20 * time.Second, Notice: "plan accepted, counts even if you disconnect"},
	// SEQ-040 and SEQ-042 (TUI-010): the class menu the server sent, looting and
	// gear, the traders, a trade, the Peril menu, effects seen on login, and a
	// slot changed by the plan.
	{Name: "class_menu", Log: "class_menu.jsonl", Mark: "class_menu", Create: &Create{Name: "Mira", Class: 5, Classes: true}},
	{Name: "character_bags", Log: "loot_equip.jsonl", Mark: "inventory_looted", Char: &Sidebar{Tab: TabBags}},
	{Name: "character_gear", Log: "loot_equip.jsonl", Mark: "inventory_worn", Notice: "wear Worn Cap ok", Char: &Sidebar{}},
	{Name: "vendor", Log: "traders.jsonl", Mark: "vendor", Traders: &Traders{Open: true, Cur: [2]int{1, 0}}, TraderName: "Testgrounds Vendor"},
	{Name: "trainer", Log: "traders.jsonl", Mark: "trainer", Traders: &Traders{Open: true, Cur: [2]int{2, 0}}, TraderName: "Wizard Trainer"},
	{Name: "bank", Log: "traders.jsonl", Mark: "bank", Notice: "bank Buckler ok", Traders: &Traders{Open: true}, TraderName: "Testgrounds Banker"},
	{Name: "trade_ask", Log: "trade.jsonl", Mark: "trade_ask", Trade: &TradeUI{}},
	{Name: "trade_window", Log: "trade.jsonl", Mark: "trade_window", Notice: "offer Scuttler Blade denied: that item is bound and cannot be traded (bound)", NoticeBad: true, Trade: &TradeUI{Item: 1}},
	{Name: "peril_menu", Log: "peril.jsonl", Mark: "peril_menu", Menu: true, MenuSel: 3},
	{Name: "peril_paid", Log: "peril.jsonl", Mark: "peril_paid", Notice: "instance requested; the log says when it is open"},
	{Name: "login_rooted", Log: "mid_fight.jsonl", Mark: "login_rooted"},
	{Name: "slots_plan", Log: "slots_round.jsonl", Mark: "slots_plan", Slots: &Slots{Slot: 0}, PlanSlots: map[int]string{1: "wizard-root"}, Left: 20 * time.Second,
		Plan: []string{"key 1: Root"}, Notice: "slot change added to the plan: Esc closes this, Enter submits the plan"},
	{Name: "slots_plan_round", Log: "slots_round.jsonl", Mark: "slots_plan_round", Left: 20 * time.Second, Plan: []string{"key 1: Root"}},
	{Name: "slots_changed", Log: "slots_round.jsonl", Mark: "slots_changed", Left: 20 * time.Second, Notice: "plan accepted, counts even if you disconnect"},
	// SEQ-044 and SEQ-045 (TUI-011): Thornwick's square, talking, the contract
	// board, the Contracts screen, the scout's marker, the zone overview, the
	// Harvest Raid's banner and a rare in view.
	{Name: "thornwick_square", Log: "thornwick_board.jsonl", Mark: "thornwick_square"},
	{Name: "talk_board", Log: "thornwick_board.jsonl", Mark: "talk_board", Notice: "talk to Thornwick Contract Board ok", Talk: &Talk{NPC: "board"}},
	{Name: "board_offers", Log: "thornwick_board.jsonl", Mark: "board_offers", Board: &Board{NPC: "board"}},
	{Name: "board_taken", Log: "thornwick_board.jsonl", Mark: "board_taken", Notice: "take Thin the Meadow Wolves ok", Board: &Board{NPC: "board"}},
	{Name: "contracts_held", Log: "thornwick_board.jsonl", Mark: "contracts_held", Contracts: &ContractsUI{}},
	{Name: "town_overview", Log: "thornwick_board.jsonl", Mark: "town_overview", Overview: true},
	{Name: "downs_camp", Log: "thornwick_hunt.jsonl", Mark: "downs_camp", Notice: "move ok"},
	{Name: "hunt_ready", Log: "thornwick_hunt.jsonl", Mark: "hunt_ready", Notice: "attack ok"},
	{Name: "board_turn_in", Log: "thornwick_hunt.jsonl", Mark: "board_turn_in", Board: &Board{NPC: "board", Pane: 1}},
	{Name: "hunt_turned_in", Log: "thornwick_hunt.jsonl", Mark: "hunt_turned_in", Notice: "hand in Thin the Meadow Wolves ok", Board: &Board{NPC: "board"}},
	{Name: "scout_marker_in_view", Log: "thornwick_scout.jsonl", Mark: "scout_marker_in_view", Notice: "use door ok"},
	{Name: "scout_reached", Log: "thornwick_scout.jsonl", Mark: "scout_reached", Notice: "move ok"},
	{Name: "zone_overview", Log: "thornwick_scout.jsonl", Mark: "zone_overview", Overview: true},
	{Name: "talk_vendor", Log: "thornwick_shops.jsonl", Mark: "talk_vendor", Notice: "talk to Hilda Brasswick ok", Talk: &Talk{NPC: "vendor-armour"}},
	{Name: "talk_vendor_window", Log: "thornwick_shops.jsonl", Mark: "talk_vendor_window", Traders: &Traders{Open: true}, TraderName: "Hilda Brasswick"},
	{Name: "trainer_window", Log: "thornwick_shops.jsonl", Mark: "trainer_window", Traders: &Traders{Open: true, Cur: [2]int{2, 0}}, TraderName: "Gorran Stoutarm"},
	{Name: "event_banner", Log: "thornwick_raid.jsonl", Mark: "event_banner"},
	{Name: "event_over", Log: "thornwick_raid.jsonl", Mark: "event_over", Notice: "attack ok"},
	{Name: "rare_in_view", Log: "thornwick_rare.jsonl", Mark: "rare_in_view", Notice: "target ok"},
	// SEQ-049 (TUI-013): the Watch Captain's quest, choosing a role, the house
	// companion and its panel, a turn-based plan with companion steps, and a
	// second character on a stance.
	{Name: "companion_talk", Log: "companion_quest.jsonl", Mark: "companion_talk", Notice: "talk to Captain Rhosyn Hartley ok", Talk: &Talk{NPC: "captain"}},
	{Name: "quest_step_taken", Log: "companion_quest.jsonl", Mark: "quest_step_taken"},
	{Name: "quest_lookout_far", Log: "companion_lookout.jsonl", Mark: "quest_lookout_far"},
	{Name: "quest_lookout_reached", Log: "companion_lookout.jsonl", Mark: "quest_lookout_reached", Notice: "move ok"},
	{Name: "quest_ready", Log: "companion_role.jsonl", Mark: "quest_ready", Board: &Board{NPC: "captain", Pane: 1}},
	{Name: "quest_done", Log: "companion_role.jsonl", Mark: "quest_done"},
	{Name: "choose_role", Log: "companion_role.jsonl", Mark: "choose_role", Role: &Role{NPC: "captain"}},
	{Name: "companion_away", Log: "companion_role.jsonl", Mark: "companion_away"},
	{Name: "companion_called", Log: "companion_role.jsonl", Mark: "companion_called", Notice: "call the companion ok"},
	{Name: "companion_follow", Log: "companion_fight.jsonl", Mark: "companion_follow"},
	{Name: "companion_assist", Log: "companion_fight.jsonl", Mark: "companion_assist", Notice: "attack ok"},
	{Name: "companion_override", Log: "companion_fight.jsonl", Mark: "companion_override", Notice: "companion order: stop ok"},
	{Name: "companion_fresh_login", Log: "companion_reconnect.jsonl", Mark: "companion_fresh_login"},
	{Name: "companion_dismissed", Log: "companion_reconnect.jsonl", Mark: "companion_dismissed"},
	{Name: "companion_plan", Log: "companion_turns.jsonl", Mark: "companion_plan", Left: 20 * time.Second,
		Plan: []string{"companion: stance assist", "target Plain Scuttler", "companion: attack Plain Scuttler"}},
	{Name: "duo_out_of_play", Log: "duo_guard.jsonl", Mark: "duo_out_of_play", Duo: &DuoView{ID: "bram", Stance: "follow"}},
	{Name: "duo_party", Log: "duo_guard.jsonl", Mark: "duo_party", Notice: "second character in play", Duo: &DuoView{ID: "bram", Active: true, Stance: "follow"}},
	{Name: "duo_guard_fight", Log: "duo_guard.jsonl", Mark: "duo_guard_fight", Notice: "attack ok", Duo: &DuoView{ID: "bram", Active: true, Stance: "guard", Target: "mob-bramble-tuskers-a-1"}},
	// SEQ-055 and WLD-004 (TUI-017): no companion panel with the default setting,
	// owed pay listed at login, and a duo in the Hollow Warren.
	{Name: "shelved_captain", Log: "shelved.jsonl", Mark: "shelved_captain", Notice: "talk to Captain Rhosyn Hartley ok", Talk: &Talk{NPC: "captain"}},
	{Name: "shelved_help", Log: "shelved.jsonl", Mark: "shelved_help", Help: true, HelpPage: 4},
	{Name: "owed_pay_login", Log: "owed_login.jsonl", Mark: "owed_pay_login"},
	{Name: "warren_duo", Log: "duo_warren.jsonl", Mark: "warren_duo", Notice: "second character: stance assist", Duo: &DuoView{ID: "bram", Active: true, Stance: "assist"}},
	{Name: "warren_entered", Log: "duo_warren.jsonl", Mark: "warren_entered", Duo: &DuoView{ID: "bram", Active: true, Stance: "assist"}},
	{Name: "warren_digger", Log: "duo_warren.jsonl", Mark: "warren_digger", Notice: "target ok", Duo: &DuoView{ID: "bram", Active: true, Stance: "assist"}},
	{Name: "warren_room_cleared", Log: "duo_warren.jsonl", Mark: "warren_room_cleared", Notice: "attack ok", Duo: &DuoView{ID: "bram", Active: true, Stance: "assist"}},
	{Name: "help", Log: "fight_loot.jsonl", Upto: 10, Help: true},
	// TUI-024: the quest chain, from a recorded session on seq at SEQ-067.
	{Name: "first_steps_offered", Log: "first_steps.jsonl", Mark: "first_steps_offered"},
	{Name: "first_steps_trainer", Log: "first_steps.jsonl", Mark: "first_steps_trainer"},
	{Name: "first_steps_gate", Log: "first_steps.jsonl", Mark: "first_steps_gate"},
	{Name: "first_steps_done", Log: "first_steps.jsonl", Mark: "first_steps_done"},
	{Name: "help_chains", Log: "fight_loot.jsonl", Upto: 10, Help: true, HelpPage: 1},
	{Name: "help_mobs", Log: "fight_loot.jsonl", Upto: 10, Help: true, HelpPage: 2},
	{Name: "help_logs", Log: "fight_loot.jsonl", Upto: 10, Help: true, HelpPage: 3},
	{Name: "help_keys", Log: "fight_loot.jsonl", Upto: 10, Help: true, HelpPage: 4},
	// M9: chat, the party, who and friends (the logs are the first player's side
	// of the two- and three-account scenarios).
	{Name: "chat_refused", Log: "chat_social.jsonl", Mark: "chat_refused", Notice: "say: you are talking too fast: wait a moment (rate_limited)", NoticeBad: true},
	{Name: "chat_line", Log: "chat_social.jsonl", Mark: "chat_refused", Line: &Line{Text: "/tell cy1 are you the"}},
	{Name: "chat_scrolled", Log: "chat_social.jsonl", Mark: "chat_scrolled", Social: &SocialUI{Scroll: 2}},
	{Name: "party_chat", Log: "party_accounts.jsonl", Mark: "party_chat"},
	{Name: "who_list", Log: "who_friends.jsonl", Mark: "who_list", Social: &SocialUI{Tab: TabWho}, WhoKnown: true},
	{Name: "friends_list", Log: "who_friends.jsonl", Mark: "friends_list", Social: &SocialUI{Tab: TabFriends}},
	{Name: "help_social", Log: "chat_social.jsonl", Mark: "chat_refused", Help: true, HelpPage: 5},
	{Name: "party_warren", Log: "party_warren.jsonl", Mark: "warren_party_opened"},
	{Name: "party_round", Log: "party_turns.jsonl", Mark: "party_round_bob_sent"},
}

// Replay reads the log named by c from dir (testdata/logs) into a new model.
func (c Case) Replay(dir string) (*model.Model, error) {
	m := model.New()
	if c.Log == "" {
		return m, nil
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
	v.Slots, v.Create, v.PlanSlots = c.Slots, c.Create, c.PlanSlots
	v.Char, v.Side, v.Traders, v.Trade = c.Char != nil, c.Char, c.Traders, c.Trade
	v.Talk, v.Board, v.Contracts, v.Overview = c.Talk, c.Board, c.Contracts, c.Overview
	v.Role, v.Duo = c.Role, c.Duo
	v.Line, v.Social, v.HelpPage = c.Line, c.Social, c.HelpPage
	if c.Social != nil && c.WhoKnown {
		ui := *c.Social
		for _, k := range m.KnownCharacters() {
			ui.Who = append(ui.Who, &pb.WhoEntry{Name: k.Name, ClassId: k.Class, Level: k.Level, Zone: k.Zone})
		}
		ui.WhoAsked = true
		v.Social = &ui
	}
	if c.Traders != nil && c.TraderName != "" {
		t := *c.Traders
		for i, n := range TradersInView(m) {
			if n.GetName() == c.TraderName {
				t.Sel, t.NPC = i, n.GetId()
			}
		}
		v.Traders = &t
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
