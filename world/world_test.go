package world_test

import (
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/world"
)

// seq-world's zones come with their ground: Thornwick's streets, the Downs'
// river and its ford, the Warren's floor, and the patrol paths of each.
func TestTheWorldsZonesHaveGroundAndPatrols(t *testing.T) {
	dir, err := world.PinnedPacks()
	if err != nil {
		t.Fatal(err)
	}
	src, err := world.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	kinds := func(tr *model.Terrain) map[model.AreaKind]bool {
		k := map[model.AreaKind]bool{}
		for _, a := range tr.Areas {
			k[a.Kind] = true
		}
		return k
	}
	town, downs, warren := src.Terrain("Thornwick"), src.Terrain("Thornwick Downs"), src.Terrain("Hollow Warren")
	if town == nil || downs == nil || warren == nil {
		t.Fatalf("zones with ground: %v", len(src))
	}
	if k := kinds(town); !k[model.Building] || !k[model.Road] || len(town.Patrols) == 0 {
		t.Errorf("Thornwick: %v, %d patrols", k, len(town.Patrols))
	}
	if k := kinds(downs); !k[model.Water] || !k[model.Ford] || !k[model.Bridge] || !k[model.Forest] || !k[model.Road] {
		t.Errorf("the Downs: %v", k)
	}
	if !warren.HasFloor() {
		t.Error("the Warren has no floor")
	}
	for _, tr := range []*model.Terrain{town, downs, warren} {
		for _, p := range tr.Patrols {
			if len(p.Points) < 2 {
				t.Errorf("patrol %s has %d points", p.Camp, len(p.Points))
			}
		}
	}
	if src.Terrain("Nowhere") != nil {
		t.Error("an unknown zone has ground")
	}
}
