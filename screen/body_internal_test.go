package screen

import (
	"strings"
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func bodiesModel(b ...pb.BodyType) *model.Model {
	m := model.New()
	m.Welcome = &pb.Welcome{Bodies: b, Classes: []*pb.ClassInfo{{Id: "warrior", Name: "Warrior"}}}
	return m
}

// The creation screen has a Body line only when the server lists both bodies,
// and shows none chosen to begin with (D202).
func TestCreateLinesBodyLine(t *testing.T) {
	both := bodiesModel(pb.BodyType_BODY_TYPE_MALE, pb.BodyType_BODY_TYPE_FEMALE)
	one := bodiesModel(pb.BodyType_BODY_TYPE_MALE)
	for _, c := range []struct {
		name string
		m    *model.Model
		c    *Create
		want string // "" for no Body line
	}{
		{"one body", one, &Create{}, ""},
		{"none listed", bodiesModel(), &Create{}, ""},
		{"two, none chosen", both, &Create{}, "Body   Female · Male"},
		{"two, male chosen", both, &Create{Body: pb.BodyType_BODY_TYPE_MALE}, "Body   Female · [Male]"},
	} {
		got := ""
		for _, l := range createLines(c.m, c.c, 70, 24, Plain()) {
			if strings.Contains(l, "Body ") {
				got = strings.TrimSpace(l)
			}
		}
		if got != c.want {
			t.Errorf("%s: body line %q, want %q", c.name, got, c.want)
		}
	}
}

// The picker names each body beside the class and offers the free change, only
// where the server allows it; with one body listed it looks as it did.
func TestPickerLinesBodyAndChange(t *testing.T) {
	now := time.Now()
	both := bodiesModel(pb.BodyType_BODY_TYPE_MALE, pb.BodyType_BODY_TYPE_FEMALE)
	one := bodiesModel(pb.BodyType_BODY_TYPE_MALE)
	rows := []PickerRow{{ID: "a", Name: "Aria", Class: "Ranger", Body: "female", CanChange: true, Level: 3}, {ID: "b", Name: "Brin", Class: "Warrior", Body: "male", Level: 2}}
	text := func(m *model.Model, p *Picker) string {
		return strings.Join(pickerLines(m, p, now, 100, 20, Plain()), "\n")
	}

	got := text(both, &Picker{Rows: rows})
	if !strings.Contains(got, "Body") || !strings.Contains(got, "female") || !strings.Contains(got, "Change body (once, free)") {
		t.Errorf("two bodies, change open:\n%s", got)
	}
	if got := text(both, &Picker{Rows: rows, Sel: 1}); strings.Contains(got, "Change body") {
		t.Errorf("a change offered on a character with none open:\n%s", got)
	}
	if got := text(one, &Picker{Rows: []PickerRow{{ID: "b", Name: "Brin", Class: "Warrior", Body: "male"}}}); strings.Contains(got, "Body") || strings.Contains(got, "male") {
		t.Errorf("one body listed: the picker shows a body:\n%s", got)
	}
	if got := text(both, &Picker{Rows: rows, Confirm: true}); !strings.Contains(got, "Change Aria to male?") {
		t.Errorf("confirmation:\n%s", got)
	}
}
