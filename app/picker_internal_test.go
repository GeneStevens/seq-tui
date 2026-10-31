package app

import (
	"errors"
	"testing"

	pb "github.com/GeneStevens/seq/protocol/seqv1"

	"github.com/GeneStevens/seq-tui/screen"
)

// A refusal for being in use asks for a fresh list at once; any other answer
// does not.
func TestPickedMsgRefreshesTheListOnlyAfterCharacterInUse(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		refresh bool
	}{
		{"in use", errors.New(`select refused: code:ERROR_CODE_SELECT_REFUSED message:"character_in_use"`), true},
		{"other refusal", errors.New(`select refused: message:"not_permitted"`), false},
	} {
		a := &App{picker: &screen.Picker{Rows: []screen.PickerRow{{ID: "hale", Online: true}}, Busy: true}, inflight: 1}
		cmd := a.pickedMsg(picked{"hale", tc.err})
		if (cmd != nil) != tc.refresh || a.refreshing != tc.refresh {
			t.Errorf("%s: refresh cmd %v, refreshing %v, want %v", tc.name, cmd != nil, a.refreshing, tc.refresh)
		}
	}
}

// A fresh list replaces the rows, keeps the chosen character chosen, and a
// failed or empty answer changes nothing.
func TestListedMsgReplacesRowsAndKeepsTheChoice(t *testing.T) {
	entry := func(id string, online bool) *pb.CharacterEntry {
		return &pb.CharacterEntry{Character: id, Name: id, Online: online}
	}
	a := &App{picker: NewPicker(nil, []*pb.CharacterEntry{entry("hale", true), entry("brin", false)}).picker, refreshing: true, inflight: 1}
	a.picker.Sel = 1
	a.listedMsg(listed{chars: []*pb.CharacterEntry{entry("hale", false), entry("zed", false), entry("brin", false)}})
	if a.picker.Rows[0].Online || a.picker.Chosen().ID != "brin" || a.refreshing || a.inflight != 0 {
		t.Errorf("rows %+v, chosen %v, refreshing %v, inflight %d", a.picker.Rows, a.picker.Chosen(), a.refreshing, a.inflight)
	}
	a.inflight = 1
	a.listedMsg(listed{err: errors.New("down")})
	a.inflight = 1
	a.listedMsg(listed{})
	if len(a.picker.Rows) != 3 {
		t.Errorf("a failed or empty answer changed the rows: %+v", a.picker.Rows)
	}
}
