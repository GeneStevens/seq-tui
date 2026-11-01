package screen_test

import (
	"os"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
)

// The panels stamp each line with the time it came (D121); the goldens need it fixed.
func TestMain(m *testing.M) {
	model.FixClock()
	os.Exit(m.Run())
}
