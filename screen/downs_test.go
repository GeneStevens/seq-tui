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
// (Highfold, where y is small... see the note in the README) and a southern one
// (Eel Mere). The pinned seq-world still has the old Downs, so the map is built
// here; the goldens are testdata/downs, rewritten by `make golden`.

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
			{Id: "millrace", Name: "The Millrace", Kind: "water", Poly: poly(173000, 16000, 170000, 75000, 166000, 125000, 172000, 175000, 168000, 225000, 164000, 275000, 160000, 330000, 150000, 392000, 160000, 392000, 180000, 330000, 182000, 275000, 180000, 225000, 184000, 175000, 180000, 125000, 184000, 75000, 185000, 16000)},
			{Id: "far-road", Name: "Far Road", Kind: "road", Poly: poly(182000, 220000, 250000, 216000, 330000, 220000, 370000, 224000, 370000, 232000, 330000, 228000, 250000, 224000, 182000, 230000)},
			{Id: "drove-road", Name: "Drove Road", Kind: "road", Poly: poly(67000, 203000, 66000, 175000, 83000, 140000, 88000, 105000, 92000, 70000, 94000, 60000, 100000, 60000, 98000, 70000, 94000, 105000, 89000, 140000, 72000, 175000, 73000, 203000)},
			{Id: "hill-track", Name: "Hill Track", Kind: "road", Poly: poly(130000, 40000, 232000, 40000, 232000, 48000, 130000, 48000)},
			{Id: "highfold-footbridge", Name: "Highfold Footbridge", Kind: "bridge", Poly: rect(164000, 42000, 184000, 48000)},
			{Id: "highfold-village", Name: "Highfold", Kind: "town", Poly: rect(80000, 20000, 130000, 56000)},
			{Id: "highfold-fold-1", Name: "Highfold Sheepfold", Kind: "building", Poly: rect(84000, 24000, 92000, 30000)},
			{Id: "highfold-shed-1", Name: "Highfold Wool Shed", Kind: "building", Poly: rect(96000, 22000, 106000, 28000)},
			{Id: "hoarstone-1", Name: "The Hoarstones", Kind: "rock", Poly: rect(206000, 30000, 209000, 33000)},
			{Id: "hoarstone-2", Name: "The Hoarstones", Kind: "rock", Poly: rect(222000, 26000, 225000, 29000)},
			{Id: "highlow", Name: "Highlow", Kind: "rock", Poly: poly(244000, 52000, 262000, 50000, 268000, 62000, 256000, 70000, 242000, 64000)},
			{Id: "eel-mere", Name: "Eel Mere", Kind: "water", Poly: poly(150000, 392000, 200000, 386000, 255000, 394000, 278000, 412000, 252000, 430000, 196000, 434000, 150000, 424000)},
			{Id: "eelmere", Name: "Eelmere", Kind: "town", Poly: rect(102000, 396000, 134000, 426000)},
			{Id: "eelmere-house-1", Name: "Eelmere Stilt House", Kind: "building", Poly: rect(108000, 400000, 116000, 408000)},
			{Id: "eel-mere-reeds-w", Name: "Eel Mere Reeds", Kind: "forest", Poly: poly(134000, 396000, 148000, 396000, 150000, 424000, 136000, 430000)},
			{Id: "eel-mere-shore", Name: "Eel Mere Shore", Kind: "road", Poly: poly(134000, 440000, 200000, 436000, 260000, 436000, 300000, 420000, 300000, 428000, 260000, 444000, 134000, 446000)},
			{Id: "marlbridge-road", Name: "Marlbridge Road", Kind: "road", Poly: poly(153000, 276000, 151000, 300000, 146000, 340000, 134000, 380000, 120000, 396000, 128000, 396000, 142000, 380000, 154000, 340000, 159000, 300000, 161000, 276000)},
			{Id: "fordwatch", Name: "Fordwatch", Kind: "town", Poly: rect(142000, 250000, 160000, 276000)},
			{Id: "greywood-north", Name: "North Greywood", Kind: "forest", Poly: poly(122000, 95000, 144000, 87000, 164000, 105000, 160000, 145000, 164000, 185000, 150000, 190000, 126000, 160000)},
			{Id: "warren-rock-north", Name: "Hollow Warren Rocks", Kind: "rock", Poly: poly(380000, 195000, 420000, 195000, 420000, 219000, 396000, 219000, 384000, 209000)},
		},
		Doors: []*pb.Door{
			{Id: "town-gate", Kind: pb.DoorKind_DOOR_KIND_ZONE_LINE, Name: "Town Gate", Pos: &pb.Pos{X: 6000, Y: 225000}, LeadsTo: "thornwick", Label: "Town Gate — to Thornwick"},
			{Id: "hollow-warren", Kind: pb.DoorKind_DOOR_KIND_ZONE_LINE, Name: "Hollow Warren", Pos: &pb.Pos{X: 400000, Y: 225000}, LeadsTo: "hollow-warren", Label: "Hollow Warren — to the Hollow Warren"},
		},
		Markers: []*pb.Marker{
			{Id: "highfold", Kind: "scout", Name: "Highfold", Pos: &pb.Pos{X: 100000, Y: 50000}},
			{Id: "hoarstones", Kind: "scout", Name: "The Hoarstones", Pos: &pb.Pos{X: 246000, Y: 38000}},
			{Id: "highlow", Kind: "scout", Name: "Highlow", Pos: &pb.Pos{X: 255000, Y: 76000}},
			{Id: "eelmere", Kind: "scout", Name: "Eelmere", Pos: &pb.Pos{X: 118000, Y: 410000}},
			{Id: "eel-mere-shore", Kind: "scout", Name: "Eel Mere Shore", Pos: &pb.Pos{X: 212000, Y: 440000}},
			{Id: "fordwatch", Kind: "scout", Name: "Fordwatch", Pos: &pb.Pos{X: 158000, Y: 268000}},
		},
		Camps: []*pb.Camp{
			{Id: "hill-wolves-a", Mob: "Hill Wolf", Level: 6, Count: 2, Pos: &pb.Pos{X: 140000, Y: 70000}},
			{Id: "reed-eels-a", Mob: "Mere Eel", Level: 8, Count: 2, Pos: &pb.Pos{X: 180000, Y: 410000}},
		},
		Npcs: []*pb.NpcPlace{
			{Id: "highfold-reeve", Name: "Maud Fenner", Role: "quest_giver", Title: "Highfold Reeve", Pos: &pb.Pos{X: 100000, Y: 40000}},
			{Id: "eelmere-ferryman", Name: "Tobias Reedy", Role: "quest_giver", Title: "Eelmere Ferryman", Pos: &pb.Pos{X: 120000, Y: 412000}},
		},
	}
}

// downsWide is the Downs with the places that stand in several pieces, as the
// real zone's do (D120): the same name on several areas, small features beside
// the large, and camps in the woods. It is the map of the large map's goldens.
func downsWide() *pb.ZoneMap {
	zm := downsMap()
	zm.Areas = append(zm.Areas,
		&pb.Area{Id: "highfold-fold-2", Name: "Highfold Sheepfold", Kind: "building", Poly: rect(84000, 33000, 92000, 38000)},
		&pb.Area{Id: "highfold-fold-3", Name: "Highfold Sheepfold", Kind: "building", Poly: rect(95000, 33000, 103000, 38000)},
		&pb.Area{Id: "hoarstone-3", Name: "The Hoarstones", Kind: "rock", Poly: rect(236000, 36000, 239000, 39000)},
		&pb.Area{Id: "goblin-copse-n", Name: "North Goblin Copse", Kind: "forest", Poly: poly(300000, 300000, 340000, 296000, 352000, 330000, 320000, 346000, 296000, 330000)},
		&pb.Area{Id: "goblin-copse-e", Name: "East Goblin Copse", Kind: "forest", Poly: poly(344000, 120000, 384000, 116000, 392000, 150000, 360000, 160000)},
		&pb.Area{Id: "goblin-hut-1", Name: "Goblin Hut", Kind: "building", Poly: rect(310000, 310000, 318000, 316000)},
		&pb.Area{Id: "goblin-hut-2", Name: "Goblin Hut", Kind: "building", Poly: rect(324000, 312000, 332000, 318000)},
		&pb.Area{Id: "goblin-hut-3", Name: "Goblin Hut", Kind: "building", Poly: rect(316000, 324000, 324000, 330000)},
		&pb.Area{Id: "eldbury-ring-1", Name: "Eldbury Ring", Kind: "rock", Poly: rect(40000, 300000, 43000, 303000)},
		&pb.Area{Id: "eldbury-ring-2", Name: "Eldbury Ring", Kind: "rock", Poly: rect(100000, 304000, 103000, 307000)},
		&pb.Area{Id: "eldbury-ring-3", Name: "Eldbury Ring", Kind: "rock", Poly: rect(76000, 332000, 79000, 335000)},
		&pb.Area{Id: "eldbury-ring-4", Name: "Eldbury Ring", Kind: "rock", Poly: rect(52000, 322000, 55000, 325000)})
	zm.Camps = append(zm.Camps,
		&pb.Camp{Id: "goblin-camp-a", Mob: "Goblin Skulker", Level: 7, Count: 3, Pos: &pb.Pos{X: 320000, Y: 322000}},
		&pb.Camp{Id: "goblin-camp-b", Mob: "Goblin Skulker", Level: 7, Count: 3, Pos: &pb.Pos{X: 372000, Y: 140000}})
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
		Progress: 20, Need: 60, Unit: "seconds", TargetKind: "place", Target: "highfold", TargetName: "Highfold", TargetPos: &pb.Pos{X: 100000, Y: 50000}}
	south := &pb.QuestInfo{Id: "eelmere-catch", Name: "A Boat for the Ferryman", Step: 3, Steps: 5, Kind: "kill", Text: "Drive the mere eels off the ferry landing.",
		Progress: 2, Need: 5, Unit: "kills", TargetKind: "place", Target: "eel-mere-shore", TargetName: "Eel Mere Shore", TargetPos: &pb.Pos{X: 212000, Y: 440000}}
	cases := []struct {
		name string
		x, y int64
		q    *pb.QuestInfo
		big  bool
		must []string
	}{
		{"downs_highfold", 100000, 44000, north, false, []string{"Watching… 20 of 60 s", "100,44 m"}},
		{"downs_eel_mere", 190000, 412000, south, false, []string{"Kills: 2 of 5", "190,412 m"}},
		{"downs_highfold_overview", 100000, 44000, north, true, nil},
		{"downs_eel_mere_overview", 190000, 412000, south, true, nil},
		{"downs_wide_overview", 100000, 44000, north, true, nil},
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
