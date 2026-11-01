package screen

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The small map shows 20 m each way of the character unless the zone says
// otherwise, and the zoom steps halve and double it (D121 ruling 6: no change).
func TestSmallMapDefaultIsTwentyMetres(t *testing.T) {
	if defaultViewRadiusMm != 20000 {
		t.Fatalf("the default view radius is %d mm", defaultViewRadiusMm)
	}
	for _, tc := range []struct {
		radius int64
		zoom   int
		want   int64
	}{{0, 0, 20000}, {-5, 0, 20000}, {20000, 0, 20000}, {0, 1, 10000}, {0, -1, 40000}, {40000, 0, 40000}} {
		if got := zoomed(tc.radius, tc.zoom); got != tc.want {
			t.Errorf("zoomed(%d, %d) = %d, want %d", tc.radius, tc.zoom, got, tc.want)
		}
	}
	// 40 rows span 2 x 20 m of height, a row being two millimetre-columns tall.
	if sc := newScale(0, 40); sc.row != 1000 || sc.col != 500 {
		t.Errorf("40 rows of the default map are %+v mm per cell", sc)
	}
}

// A gather contract says what its count is, shortened as the row needs (D121).
func TestGatherStateWords(t *testing.T) {
	c := &pb.ContractInfo{Name: "Pelts for the Tanners", Kind: "gather", Have: 1, Need: 3}
	for _, tc := range []struct {
		w    int
		want string
	}{
		{70, "1/3 · looted since you took it"},
		{40, "1/3 · looted since taken"},
		{36, "1/3 · looted"},
		{22, "1/3"},
	} {
		if got := stateWords(model.New(), c, tc.w); got != tc.want {
			t.Errorf("w=%d: %q, want %q", tc.w, got, tc.want)
		}
	}
	done := &pb.ContractInfo{Name: "Pelts", Kind: "gather", Have: 3, Need: 3, Ready: true}
	if got := stateWords(model.New(), done, 70); got != "READY" {
		t.Errorf("a finished gather says %q", got)
	}
	if got := stateWords(model.New(), &pb.ContractInfo{Name: "Wolves", Kind: "hunt", Have: 1, Need: 3}, 70); got != "1/3" {
		t.Errorf("a hunt says %q", got)
	}
}

func gatherWorld() *model.Model {
	m := model.New()
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Snapshot{Snapshot: &pb.Snapshot{Space: "z", Seq: 1, GameTimeMs: 1000,
		Self: &pb.SelfState{Entity: &pb.EntityState{Id: "me", Name: "Me", Pos: &pb.Pos{}, Hp: 10, MaxHp: 10},
			Contracts: []*pb.ContractInfo{{Id: "pelts", Name: "Pelts for the Tanners", Kind: "gather", Have: 1, Need: 3, Description: "Bring back three wolf pelts."}}}}}})
	return m
}

func TestGatherWordingInThePanelAndTheDetails(t *testing.T) {
	m := gatherWorld()
	_, box := contractsBox(m, 60, Plain())
	if got := strings.Join(strings.Fields(strings.Join(box, "\n")), " "); !strings.Contains(got, "Pelts for the Tanners 1/3 · looted since you took it") {
		t.Errorf("the Contracts panel says %q", got)
	}
	full := strings.Join(contractRows(m, 60, true, 0, Plain()), "\n")
	if !strings.Contains(full, "1 of 3 looted since you took it") {
		t.Errorf("the contract details say %q", full)
	}
	if strings.Contains(full, "carried") {
		t.Errorf("a gather contract still counts what is carried: %q", full)
	}
}

// The Events and Chat panels stamp each line with the local time (D121).
func TestPanelsStampLinesWithLocalTime(t *testing.T) {
	m := gatherWorld()
	m.Notice("something happened")
	m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Chat{Chat: &pb.ChatMessage{Channel: "say", From: "Mira", Text: "well met"}}})
	events := strings.Join(func() []string { l, _ := logLines(m, View{}, 70, 6, Plain()); return l }(), "\n")
	if !strings.Contains(events, "16:04:03.2 something happened") || strings.Contains(events, "1.0s") {
		t.Errorf("Events:\n%s", events)
	}
	chat := strings.Join(chatRowsOf(m, 70, Plain()), "\n")
	if !strings.Contains(chat, " 16:04:03.2 [say] Mira: well met") {
		t.Errorf("Chat:\n%s", chat)
	}
}
