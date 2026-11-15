package model_test

import (
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// A figure that appears keeps its class, appearance and look (an absent look
// stays absent); a change of what it wears updates it and goes to no log.
func TestAFiguresLookIsKeptAndItsChangesAreNotLogged(t *testing.T) {
	m := model.New()
	jerkin := &pb.VisibleGear{Chest: &pb.ItemLook{Family: pb.LookFamily_LOOK_FAMILY_JERKIN}}
	event(m, 1000, &pb.Event{Body: &pb.Event_EntityAppeared{EntityAppeared: &pb.EntityAppeared{Entity: &pb.EntityState{
		Id: "p1", Name: "Aria", Kind: pb.EntityKind_ENTITY_KIND_CHARACTER, Class: "ranger", VisibleGear: jerkin}}}})
	event(m, 1100, &pb.Event{Body: &pb.Event_EntityAppeared{EntityAppeared: &pb.EntityAppeared{Entity: &pb.EntityState{
		Id: "w1", Name: "Wolf", Kind: pb.EntityKind_ENTITY_KIND_MOB, Appearance: "a grey wolf"}}}})
	if e := m.Entities["p1"]; e.GetClass() != "ranger" || e.GetVisibleGear().GetChest() == nil {
		t.Errorf("the player lost its class or look: %v", e)
	}
	if e := m.Entities["w1"]; e.GetAppearance() != "a grey wolf" || e.GetVisibleGear() != nil {
		t.Errorf("the mob's appearance, or an absent look: %v", e)
	}
	logged := len(m.Log)
	event(m, 1200, &pb.Event{Body: &pb.Event_EntityHeld{EntityHeld: &pb.EntityHeld{Id: "p1", VisibleGear: &pb.VisibleGear{}}}})
	if g := m.Entities["p1"].GetVisibleGear(); g == nil || g.GetChest() != nil {
		t.Errorf("the change was not kept: %v", g)
	}
	if len(m.Log) != logged {
		t.Errorf("a change of gear is in the log: %+v", m.Log[logged:])
	}
}
