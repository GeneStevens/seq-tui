package screen

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The look and class words of a figure: the worn phrase is agentkit's; "plain
// clothes" only when the server sent the block empty, nothing when it did not.
func TestLookWords(t *testing.T) {
	jerkin := &pb.VisibleGear{Chest: &pb.ItemLook{Family: pb.LookFamily_LOOK_FAMILY_JERKIN, Material: pb.LookMaterial_LOOK_MATERIAL_HIDE, Colour: pb.LookColour_LOOK_COLOUR_BROWN}}
	mob := pb.EntityKind_ENTITY_KIND_MOB
	player := pb.EntityKind_ENTITY_KIND_CHARACTER
	m := model.New()
	for _, c := range []struct {
		name  string
		e     *pb.EntityState
		wears string
		label string // the hover label of a player
	}{
		{"worn", &pb.EntityState{Name: "Aria", Kind: player, Level: 3, Class: "ranger", VisibleGear: jerkin}, "Wears: brown hide jerkin; empty hands", "Aria  Lv 3 Ranger"},
		{"plain clothes", &pb.EntityState{Name: "Aria", Kind: player, Level: 3, VisibleGear: &pb.VisibleGear{}}, "Wears: plain clothes; empty hands", "Aria  Lv 3"},
		{"old server", &pb.EntityState{Name: "Aria", Kind: player, Level: 3, Class: "ranger"}, "", "Aria  Lv 3 Ranger"},
		{"mob appearance", &pb.EntityState{Name: "Wolf", Kind: mob, Appearance: "a grey wolf, ribs showing"}, "Wears: a grey wolf, ribs showing", ""},
		{"mob without", &pb.EntityState{Name: "Wolf", Kind: mob}, "", ""},
	} {
		if got := WearsLine(c.e); got != c.wears {
			t.Errorf("%s: wears %q, want %q", c.name, got, c.wears)
		}
		if c.label != "" {
			if got := playerLabel(m, c.e); got != c.label {
				t.Errorf("%s: label %q, want %q", c.name, got, c.label)
			}
		}
	}
}

// The Target panel carries the look as its fourth line, cut with an ellipsis to
// the panel's width; the look card wraps all of it.
func TestTargetWearsLine(t *testing.T) {
	m := model.New()
	m.Self.Target = "p1"
	m.Entities["p1"] = &pb.EntityState{Id: "p1", Name: "Aria", Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, Level: 3, Pos: &pb.Pos{}, VisibleGear: &pb.VisibleGear{}}
	lines := targetLines(m, 20, Plain())
	if len(lines) != 4 || !strings.HasPrefix(lines[3], " Wears: plain cloth") || !strings.Contains(lines[3], "…") {
		t.Errorf("target lines %q", lines)
	}
	if card := LookLines(m, "p1", 40); len(card) != 2 || card[0] != "Aria  Lv 3" || !strings.Contains(card[1], "plain clothes; empty hands") {
		t.Errorf("look card %q", card)
	}
	m.Entities["p1"].VisibleGear = nil
	if lines := targetLines(m, 20, Plain()); len(lines) != 3 {
		t.Errorf("an old server's target has no Wears line: %q", lines)
	}
}

// A figure's light is said after its gear, in the server's words (D200.5); a
// figure with none reads as it did.
func TestLookCarriesLight(t *testing.T) {
	light := func(strength, colour, origin string) *pb.VisibleGear {
		return &pb.VisibleGear{Light: &pb.CarriedLight{Strength: strength, Colour: colour, Origin: origin}}
	}
	held := func(family string) *pb.HeldForms { return &pb.HeldForms{Off: &pb.ItemForm{Family: family}} }
	for _, c := range []struct {
		name string
		g    *pb.VisibleGear
		h    *pb.HeldForms
		want string
	}{
		{"none", &pb.VisibleGear{}, nil, "Wears: plain clothes; empty hands"},
		{"a torch", light("medium", "warm", "off_hand"), held("torch"), "Wears: plain clothes; holds a torch; carries a lit torch"},
		{"a lantern", light("wide", "warm", "off_hand"), held("lantern"), "Wears: plain clothes; holds a lantern; carries a lit lantern"},
		{"a cold body light", light("faint", "cold", "body"), nil, "Wears: plain clothes; empty hands; a pale glow at the throat"},
		{"a faint head light", light("small", "warm", "head"), nil, "Wears: plain clothes; empty hands; a small warm light at the brow"},
		{"an off-hand light that is no torch", light("small", "cold", "off_hand"), nil, "Wears: plain clothes; empty hands; carries a small cold light"},
	} {
		e := &pb.EntityState{Name: "Aria", Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, VisibleGear: c.g, Held: c.h}
		if got := WearsLine(e); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
