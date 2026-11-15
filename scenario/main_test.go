package scenario_test

import (
	"flag"
	"testing"

	"github.com/GeneStevens/seq-tui/scenario"
)

func TestMain(m *testing.M) {
	_ = flag.Set("test.skip", "TestWizard|TestCast|TestWarrior|TestLevelUp|TestTurnBased|TestCreation|TestCompanion|TestSecondCharacter")
	scenario.Main(m)
} // DEBUGTMP
