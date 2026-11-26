package scenario_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
)

// The cast cue's reach is a figure the client keeps (the server sends none with
// a cast); this holds each one to the range_mm seq-world's pinned packs give the
// ability, so the figure cannot drift from the content unseen.
func TestClientReachMatchesTheWorldPack(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(scenario.WorldPacks(t), "*", "*abilities.yaml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no ability packs: %v", err)
	}
	var text string
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		text += string(b)
	}
	for _, id := range model.ClientReachAbilities() {
		found := regexp.MustCompile(`\{id: ` + regexp.QuoteMeta(id) + `,[^}\n]*?range_mm: (\d+)`).FindStringSubmatch(text)
		if found == nil {
			t.Errorf("%s: no range_mm in the world's packs", id)
			continue
		}
		if mm, _ := strconv.ParseInt(found[1], 10, 64); mm != model.ClientReachMm(id) {
			t.Errorf("%s: the client says a reach of %d mm, the pack %d mm", id, model.ClientReachMm(id), mm)
		}
	}
}
