//go:build live

package main

import (
	"testing"
)

func TestLiveSmokeStartupJoin(t *testing.T) {
	target := defaultTarget()
	t.Logf("config: %s", smokeConfigSummary(target))
	t.Log("---")

	steps := runSmoke(target)
	for _, s := range steps {
		t.Logf("%s", s)
	}

	t.Log("---")
	if smokeHasFailed(steps) {
		t.Fatal("smoke failed — see step details above")
	}
	t.Log("all smoke steps passed")
}
