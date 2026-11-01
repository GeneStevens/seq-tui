package model_test

import (
	"os"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
)

// Lines carry the time they came (D121); tests that compare models need it fixed.
func TestMain(m *testing.M) {
	model.FixClock()
	os.Exit(m.Run())
}
