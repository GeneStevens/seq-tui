package screen_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The bigger Downs (420 x 450 m, seq-world's WLD-031): a trimmed copy of its map
// with the places the quests send a character to, drawn from a northern view
// (Highfold) and a southern one (Eel Mere). The map is built here in the layout
// seq-world gave the Downs in WLD-053 (D125): north is up, so Highfold has the
// large y. The goldens are testdata/downs, rewritten by `make golden`.

func rect(x0, y0, x1, y1 int64) []*pb.Pos {
	return []*pb.Pos{{X: x0, Y: y0}, {X: x1, Y: y0}, {X: x1, Y: y1}, {X: x0, Y: y1}}
}

func poly(xy ...int64) []*pb.Pos {
	var out []*pb.Pos
	for i := 0; i+1 < len(xy); i += 2 {
		out = append(out, &pb.Pos{X: xy[i], Y: xy[i+1]})
	}
	return out
}

func downsMap() *pb.ZoneMap {
	return &pb.ZoneMap{Space: "thornwick-downs", Name: "Thornwick Downs", ViewRadiusMm: 40000,
		Bounds: &pb.Rect{MinX: 0, MinY: 0, MaxX: 420000, MaxY: 450000},
		Areas: []*pb.Area{
			{Id: "millrace", Name: "The Millrace", Kind: "water", Poly: poly(173000, 434000, 170000, 375000, 166000, 325000, 172000, 275000, 168000, 225000, 164000, 175000, 160000, 120000, 150000, 58000, 160000, 58000, 180000, 120000, 182000, 175000, 180000, 225000, 184000, 275000, 180000, 325000, 184000, 375000, 185000, 434000)},
			{Id: "far-road", Name: "Far Road", Kind: "road", Poly: poly(182000, 230000, 250000, 234000, 330000, 230000, 370000, 226000, 370000, 218000, 330000, 222000, 250000, 226000, 182000, 220000)},
			{Id: "drove-road", Name: "Drove Road", Kind: "road", Poly: poly(67000, 247000, 66000, 275000, 83000, 310000, 88000, 345000, 92000, 380000, 94000, 390000, 100000, 390000, 98000, 380000, 94000, 345000, 89000, 310000, 72000, 275000, 73000, 247000)},
			{Id: "hill-track", Name: "Hill Track", Kind: "road", Poly: poly(130000, 410000, 232000, 410000, 232000, 402000, 130000, 402000)},
			{Id: "highfold-footbridge", Name: "Highfold Footbridge", Kind: "bridge", Poly: rect(164000, 408000, 184000, 402000)},
			{Id: "highfold-village", Name: "Highfold", Kind: "town", Poly: rect(80000, 430000, 130000, 394000)},
			{Id: "highfold-fold-1", Name: "Highfold Sheepfold", Kind: "building", Poly: rect(84000, 426000, 92000, 420000)},
			{Id: "highfold-shed-1", Name: "Highfold Wool Shed", Kind: "building", Poly: rect(96000, 428000, 106000, 422000)},
			{Id: "hoarstone-1", Name: "The Hoarstones", Kind: "rock", Poly: rect(206000, 420000, 209000, 417000)},
			{Id: "hoarstone-2", Name: "The Hoarstones", Kind: "rock", Poly: rect(222000, 424000, 225000, 421000)},
			{Id: "highlow", Name: "Highlow", Kind: "rock", Poly: poly(244000, 398000, 262000, 400000, 268000, 388000, 256000, 380000, 242000, 386000)},
			{Id: "eel-mere", Name: "Eel Mere", Kind: "water", Poly: poly(150000, 58000, 200000, 64000, 255000, 56000, 278000, 38000, 252000, 20000, 196000, 16000, 150000, 26000)},
			{Id: "eelmere", Name: "Eelmere", Kind: "town", Poly: rect(102000, 54000, 134000, 24000)},
			{Id: "eelmere-house-1", Name: "Eelmere Stilt House", Kind: "building", Poly: rect(108000, 50000, 116000, 42000)},
			{Id: "eel-mere-reeds-w", Name: "Eel Mere Reeds", Kind: "forest", Poly: poly(134000, 54000, 148000, 54000, 150000, 26000, 136000, 20000)},
			{Id: "eel-mere-shore", Name: "Eel Mere Shore", Kind: "road", Poly: poly(134000, 10000, 200000, 14000, 260000, 14000, 300000, 30000, 300000, 22000, 260000, 6000, 134000, 4000)},
			{Id: "marlbridge-road", Name: "Marlbridge Road", Kind: "road", Poly: poly(153000, 174000, 151000, 150000, 146000, 110000, 134000, 70000, 120000, 54000, 128000, 54000, 142000, 70000, 154000, 110000, 159000, 150000, 161000, 174000)},
			{Id: "fordwatch", Name: "Fordwatch", Kind: "town", Poly: rect(142000, 200000, 160000, 174000)},
			{Id: "greywood-north", Name: "North Greywood", Kind: "forest", Poly: poly(122000, 355000, 144000, 363000, 164000, 345000, 160000, 305000, 164000, 265000, 150000, 260000, 126000, 290000)},
			{Id: "warren-rock-north", Name: "Hollow Warren Rocks", Kind: "rock", Poly: poly(380000, 255000, 420000, 255000, 420000, 231000, 396000, 231000, 384000, 241000)},
		},
		Doors: []*pb.Door{
			{Id: "town-gate", Kind: pb.DoorKind_DOOR_KIND_ZONE_LINE, Name: "Town Gate", Pos: &pb.Pos{X: 6000, Y: 225000}, LeadsTo: "thornwick", Label: "Town Gate — to Thornwick"},
			{Id: "hollow-warren", Kind: pb.DoorKind_DOOR_KIND_ZONE_LINE, Name: "Hollow Warren", Pos: &pb.Pos{X: 400000, Y: 225000}, LeadsTo: "hollow-warren", Label: "Hollow Warren — to the Hollow Warren"},
		},
		Markers: []*pb.Marker{
			{Id: "highfold", Kind: "scout", Name: "Highfold", Pos: &pb.Pos{X: 100000, Y: 400000}},
			{Id: "hoarstones", Kind: "scout", Name: "The Hoarstones", Pos: &pb.Pos{X: 246000, Y: 412000}},
			{Id: "highlow", Kind: "scout", Name: "Highlow", Pos: &pb.Pos{X: 255000, Y: 374000}},
			{Id: "eelmere", Kind: "scout", Name: "Eelmere", Pos: &pb.Pos{X: 118000, Y: 40000}},
			{Id: "eel-mere-shore", Kind: "scout", Name: "Eel Mere Shore", Pos: &pb.Pos{X: 212000, Y: 10000}},
			{Id: "fordwatch", Kind: "scout", Name: "Fordwatch", Pos: &pb.Pos{X: 158000, Y: 182000}},
		},
		Camps: []*pb.Camp{
			{Id: "hill-wolves-a", Mob: "Hill Wolf", Level: 6, Count: 2, Pos: &pb.Pos{X: 140000, Y: 380000}},
			{Id: "reed-eels-a", Mob: "Mere Eel", Level: 8, Count: 2, Pos: &pb.Pos{X: 180000, Y: 40000}},
		},
		Npcs: []*pb.NpcPlace{
			{Id: "highfold-reeve", Name: "Maud Fenner", Role: "quest_giver", Title: "Highfold Reeve", Pos: &pb.Pos{X: 100000, Y: 410000}},
			{Id: "eelmere-ferryman", Name: "Tobias Reedy", Role: "quest_giver", Title: "Eelmere Ferryman", Pos: &pb.Pos{X: 120000, Y: 38000}},
		},
	}
}

// downsWide is the Downs with the places that stand in several pieces, as the
// real zone's do (D120): the same name on several areas, small features beside
// the large, and camps in the woods. It is the map of the large map's goldens.
func downsWide() *pb.ZoneMap {
	zm := downsMap()
	zm.Areas = append(zm.Areas,
		&pb.Area{Id: "highfold-fold-2", Name: "Highfold Sheepfold", Kind: "building", Poly: rect(84000, 417000, 92000, 412000)},
		&pb.Area{Id: "highfold-fold-3", Name: "Highfold Sheepfold", Kind: "building", Poly: rect(95000, 417000, 103000, 412000)},
		&pb.Area{Id: "hoarstone-3", Name: "The Hoarstones", Kind: "rock", Poly: rect(236000, 414000, 239000, 411000)},
		&pb.Area{Id: "goblin-copse-n", Name: "North Goblin Copse", Kind: "forest", Poly: poly(300000, 150000, 340000, 154000, 352000, 120000, 320000, 104000, 296000, 120000)},
		&pb.Area{Id: "goblin-copse-e", Name: "East Goblin Copse", Kind: "forest", Poly: poly(344000, 330000, 384000, 334000, 392000, 300000, 360000, 290000)},
		&pb.Area{Id: "goblin-hut-1", Name: "Goblin Hut", Kind: "building", Poly: rect(310000, 140000, 318000, 134000)},
		&pb.Area{Id: "goblin-hut-2", Name: "Goblin Hut", Kind: "building", Poly: rect(324000, 138000, 332000, 132000)},
		&pb.Area{Id: "goblin-hut-3", Name: "Goblin Hut", Kind: "building", Poly: rect(316000, 126000, 324000, 120000)},
		&pb.Area{Id: "eldbury-ring-1", Name: "Eldbury Ring", Kind: "rock", Poly: rect(40000, 150000, 43000, 147000)},
		&pb.Area{Id: "eldbury-ring-2", Name: "Eldbury Ring", Kind: "rock", Poly: rect(100000, 146000, 103000, 143000)},
		&pb.Area{Id: "eldbury-ring-3", Name: "Eldbury Ring", Kind: "rock", Poly: rect(76000, 118000, 79000, 115000)},
		&pb.Area{Id: "eldbury-ring-4", Name: "Eldbury Ring", Kind: "rock", Poly: rect(52000, 128000, 55000, 125000)},
		// D121: small places that matter are named at any zoom: a hamlet with a
		// quest person, a stockade (a building), a ford with its warden.
		&pb.Area{Id: "barleycross", Name: "Barleycross", Kind: "town", Poly: rect(262000, 330000, 284000, 312000)},
		&pb.Area{Id: "goblin-stockade", Name: "Goblin Stockade", Kind: "building", Poly: rect(356000, 150000, 372000, 138000)},
		&pb.Area{Id: "goblin-rocks", Name: "Goblin Rocks", Kind: "rock", Poly: rect(326000, 116000, 329000, 113000)})
	zm.Npcs = append(zm.Npcs,
		&pb.NpcPlace{Id: "eldbury-druid", Name: "Aldith Marrow", Role: "quest_giver", Title: "Ring Warden", Pos: &pb.Pos{X: 77000, Y: 120000}},
		&pb.NpcPlace{Id: "barleycross-miller", Name: "Edda Barley", Role: "quest_giver", Title: "Barleycross Miller", Pos: &pb.Pos{X: 272000, Y: 322000}},
		&pb.NpcPlace{Id: "fordwatch-warden", Name: "Hob Ferrier", Role: "vendor", Title: "Fordwatch Warden", Pos: &pb.Pos{X: 151000, Y: 188000}})
	zm.Camps = append(zm.Camps,
		&pb.Camp{Id: "goblin-camp-a", Mob: "Goblin Skulker", Level: 7, Count: 3, Pos: &pb.Pos{X: 320000, Y: 128000}},
		&pb.Camp{Id: "goblin-camp-b", Mob: "Goblin Skulker", Level: 7, Count: 3, Pos: &pb.Pos{X: 372000, Y: 310000}})
	return zm
}

// downsWorld is the Downs with the character at (x, y) and the quest step that
// brings it there.
func downsWorld(x, y int64, q *pb.QuestInfo, seen ...*pb.NpcInfo) *model.Model {
	return downsWorldOn(downsMap(), x, y, q, seen...)
}

func downsWorldOn(zm *pb.ZoneMap, x, y int64, q *pb.QuestInfo, seen ...*pb.NpcInfo) *model.Model {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_ZoneMap{ZoneMap: zm}})
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "thornwick-downs", Seq: 1, GameTimeMs: 1000,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Hale", Level: 8, Pos: &pb.Pos{X: x, Y: y}, Hp: 130, MaxHp: 130}, Quest: q},
		Npcs: seen}}})
	return m
}

func TestTheBiggerDownsGoldens(t *testing.T) {
	north := &pb.QuestInfo{Id: "highfold-watch", Name: "The Shepherds' Watch", Step: 2, Steps: 4, Kind: "watch", Text: "Watch the fold from the green until the wolves show themselves.",
		Progress: 20, Need: 60, Unit: "seconds", TargetKind: "place", Target: "highfold", TargetName: "Highfold", TargetPos: &pb.Pos{X: 100000, Y: 400000}}
	south := &pb.QuestInfo{Id: "eelmere-catch", Name: "A Boat for the Ferryman", Step: 3, Steps: 5, Kind: "kill", Text: "Drive the mere eels off the ferry landing.",
		Progress: 2, Need: 5, Unit: "kills", TargetKind: "place", Target: "eel-mere-shore", TargetName: "Eel Mere Shore", TargetPos: &pb.Pos{X: 212000, Y: 10000}}
	cases := []struct {
		name string
		x, y int64
		q    *pb.QuestInfo
		big  bool
		must []string
	}{
		{"downs_highfold", 100000, 406000, north, false, []string{"Watching… 20 of 60 s", "100,406 m"}},
		{"downs_eel_mere", 190000, 38000, south, false, []string{"Kills: 2 of 5", "190,38 m"}},
		{"downs_highfold_overview", 100000, 406000, north, true, nil},
		{"downs_eel_mere_overview", 190000, 38000, south, true, nil},
		{"downs_wide_overview", 100000, 406000, north, true, nil},
	}
	for _, c := range cases {
		for _, sz := range [][2]int{{80, 24}, {120, 40}, {200, 50}} {
			if sz[0] == 200 && c.name != "downs_wide_overview" {
				continue // the wide window is held by the Downs' own golden
			}
			m := downsWorld(c.x, c.y, c.q)
			if c.name == "downs_wide_overview" {
				m = downsWorldOn(downsWide(), c.x, c.y, c.q)
			}
			v := view()
			v.Overview = c.big
			got := screen.Render(m, v, sz[0], sz[1], screen.Plain()) + "\n"
			for _, want := range c.must {
				if !strings.Contains(got, want) {
					t.Errorf("%s at %v does not show %q:\n%s", c.name, sz, want, got)
				}
			}
			for i, l := range strings.Split(strings.TrimRight(got, "\n"), "\n") {
				if n := len([]rune(l)); n > sz[0] {
					t.Errorf("%s at %v: row %d is %d cells wide", c.name, sz, i, n)
				}
			}
			if n := strings.Count(got, "\n"); n != sz[1] {
				t.Errorf("%s at %v: %d rows", c.name, sz, n)
			}
			file := filepath.Join("../testdata/downs", c.name+"_"+strconv.Itoa(sz[0])+"x"+strconv.Itoa(sz[1])+".txt")
			if os.Getenv("SEQ_GOLDEN") == "1" {
				if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			want, err := os.ReadFile(file)
			if err != nil {
				t.Errorf("%s: %v (run make golden)", file, err)
				continue
			}
			if got != string(want) {
				t.Errorf("%s differs from its golden (make golden rewrites it after review)\n--- got\n%s--- want\n%s", file, got, want)
			}
		}
	}
}
