// Package world reads the ground of the zones (areas and patrol paths, D53)
// from seq-world's packs, the world's content repository (D54).
//
// This is a stopgap and the only place the client reads world content: the
// protocol's zone map does not carry areas or patrols yet, so a client that
// draws them reads them where the server does. Nothing here is copied into this
// repository, and nothing but `model.Terrain` leaves the package. When the
// protocol carries the ground, the model is fed from it and this package goes.
package world

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq/content"
	"github.com/GeneStevens/seq/core"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// EnvPacks names a directory of packs to read instead of the pinned
// seq-world's.
const EnvPacks = "SEQ_WORLD_PACKS"

// PinnedPacks is the packs directory of the seq-world module go.mod pins, found
// by asking the go tool where it is (or EnvPacks, when set).
func PinnedPacks() (string, error) {
	if d := os.Getenv(EnvPacks); d != "" {
		return d, nil
	}
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/GeneStevens/seq-world").Output()
	dir := strings.TrimSpace(string(out))
	if err != nil || dir == "" {
		return "", fmt.Errorf("seq-world is not in the module cache (go mod download github.com/GeneStevens/seq-world): %v", err)
	}
	return filepath.Join(dir, "packs"), nil
}

// Source is the terrain of each zone of the packs, by the zone's name (the
// name Snapshot.space carries).
type Source map[string]*model.Terrain

// Terrain implements model.TerrainSource.
func (s Source) Terrain(space string) *model.Terrain { return s[space] }

// Load reads the packs under dir and returns the terrain of every zone that has
// areas or patrols.
func Load(dir string) (Source, error) {
	if _, err := content.UsePacks(dir); err != nil {
		return nil, err
	}
	return FromLoaded()
}

// FromLoaded is Load for packs the engine has already loaded (a test server
// that was given them).
func FromLoaded() (Source, error) {
	ids, err := content.ZoneIDs()
	if err != nil {
		return nil, err
	}
	src := Source{}
	for _, id := range ids {
		z, err := content.Zone(id)
		if err != nil {
			return nil, err
		}
		if t := terrainOf(z); t != nil {
			src[z.Name] = t
		}
	}
	return src, nil
}

func terrainOf(z *core.Zone) *model.Terrain {
	t := &model.Terrain{}
	for _, a := range z.Areas {
		t.Areas = append(t.Areas, model.NewArea(a.ID, model.AreaKind(a.Kind), poss(a.Poly)))
	}
	for _, s := range z.Spawns {
		if s.Patrol != nil && len(s.Patrol.Points) > 1 {
			t.Patrols = append(t.Patrols, model.Patrol{Camp: s.ID, Loop: s.Patrol.Loop, Points: poss(s.Patrol.Points)})
		}
	}
	if len(t.Areas) == 0 && len(t.Patrols) == 0 {
		return nil
	}
	return t
}

func poss(ps []core.Pos) []*pb.Pos {
	out := make([]*pb.Pos, len(ps))
	for i, p := range ps {
		out[i] = &pb.Pos{X: int64(p.X), Y: int64(p.Y), Z: int64(p.Z)}
	}
	return out
}
