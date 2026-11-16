package app

import (
	"errors"
	"testing"

	pb "github.com/GeneStevens/seq/protocol/seqv1"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
)

func bodiesModel(b ...pb.BodyType) *model.Model {
	m := model.New()
	m.Welcome = &pb.Welcome{Bodies: b, Classes: []*pb.ClassInfo{{Id: "warrior", Name: "Warrior"}}}
	return m
}

// With two bodies listed creation waits for a choice and says so plainly; with
// one it sends no body (D202).
func TestCreateKeyBody(t *testing.T) {
	both := bodiesModel(pb.BodyType_BODY_TYPE_MALE, pb.BodyType_BODY_TYPE_FEMALE)
	a := &App{create: &screen.Create{Name: "Aria"}}
	if act := a.createKey("enter", both); act.Kind == CreateCharacter || a.create.Err != "choose a body: Female or Male" {
		t.Fatalf("no body chosen: action %+v, err %q", act, a.create.Err)
	}
	for _, k := range []string{"down", "right"} { // to the body line, no preselection until a key
		if a.createKey(k, both); k == "down" && (!a.create.Bodies || a.create.Body != 0) {
			t.Fatalf("down: bodies %v body %v", a.create.Bodies, a.create.Body)
		}
	}
	if a.create.Body != pb.BodyType_BODY_TYPE_FEMALE {
		t.Fatalf("right chose %v, want female", a.create.Body)
	}
	a.createKey("m", both)
	if act := a.createKey("enter", both); act.Kind != CreateCharacter || act.Body != "male" || act.ID != "warrior" {
		t.Errorf("action %+v", act)
	}
	a.createKey("tab", both) // body -> classes
	if !a.create.Classes || a.create.Bodies {
		t.Errorf("tab: classes %v bodies %v", a.create.Classes, a.create.Bodies)
	}
	one := bodiesModel(pb.BodyType_BODY_TYPE_MALE)
	b := &App{create: &screen.Create{Name: "Aria"}}
	b.createKey("down", one)
	if !b.create.Classes || b.create.Bodies {
		t.Errorf("one body: down went to bodies")
	}
	if act := b.createKey("enter", one); act.Kind != CreateCharacter || act.Body != "" {
		t.Errorf("one body: action %+v", act)
	}
}

// A character made whose choosing failed is offered on the picker, refreshed
// and chosen; nothing creates it again.
func TestMadeNotChosenOffersTheCharacter(t *testing.T) {
	a := &App{create: &screen.Create{Busy: true}, inflight: 1}
	cmd := a.madeNotChosen(created{err: errors.New("select failed"), id: "c9"})
	if cmd == nil || a.create != nil || a.picker == nil || a.want != "c9" || a.picker.Err == "" {
		t.Fatalf("cmd %v, create %v, picker %v, want %q", cmd != nil, a.create, a.picker, a.want)
	}
	a.listedMsg(listed{chars: []*pb.CharacterEntry{{Character: "c1", Name: "Old"}, {Character: "c9", Name: "New"}}})
	if c := a.picker.Chosen(); c == nil || c.ID != "c9" || a.want != "" {
		t.Errorf("chosen %v", c)
	}
	if act := a.pickerKey("enter"); act.Kind != PickCharacter || act.ID != "c9" {
		t.Errorf("enter: %+v", act)
	}
}

// The picker's B asks, Enter sends the owner's change to the other body, any
// other key does not; a refusal is said plainly.
func TestPickerChangeBody(t *testing.T) {
	rows := []*pb.CharacterEntry{{Character: "c1", Name: "Aria", BodyType: pb.BodyType_BODY_TYPE_FEMALE, BodyChangeAvailable: true}, {Character: "c2", Name: "Brin", BodyType: pb.BodyType_BODY_TYPE_MALE}}
	a := NewPicker(nil, rows)
	a.picker.Sel = 1
	if a.pickerKey("b"); a.picker.Confirm {
		t.Fatal("B asked about a character with no change open")
	}
	a.picker.Sel = 0
	a.pickerKey("b")
	if !a.picker.Confirm {
		t.Fatal("B did not ask")
	}
	if act := a.pickerKey("x"); act.Kind != None || a.picker.Confirm {
		t.Errorf("other key: %+v", act)
	}
	a.pickerKey("b")
	if act := a.pickerKey("enter"); act.Kind != ChangeBody || act.ID != "c1" || act.Body != "male" {
		t.Errorf("enter: %+v", act)
	}
	a.inflight, a.picker.Busy = 1, true
	a.bodyChangedMsg(bodyChanged{errors.New("body change refused: used: the one free change of body has been used")})
	if a.picker.Busy || a.picker.Err != "used: the one free change of body has been used" {
		t.Errorf("refusal: busy %v err %q", a.picker.Busy, a.picker.Err)
	}
}
