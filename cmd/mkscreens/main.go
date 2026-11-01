// Command mkscreens rewrites the golden screens in testdata/screens from the
// recorded logs in testdata/logs. It is run by `make golden` and nowhere else:
// tests only compare. Review the diff before committing; a golden changing
// means the screen changed.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GeneStevens/seq-tui/screen"
)

func main() {
	out := filepath.Join("testdata", "screens")
	if err := os.MkdirAll(out, 0o755); err != nil {
		fail(err)
	}
	for _, c := range screen.Cases {
		m, err := c.Replay(filepath.Join("testdata", "logs"))
		if err != nil {
			fail(err)
		}
		for _, s := range screen.Sizes {
			if err := os.WriteFile(filepath.Join(out, c.File(s[0], s[1])), []byte(c.Draw(m, s[0], s[1])+"\n"), 0o644); err != nil {
				fail(err)
			}
		}
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "mkscreens:", err)
	os.Exit(1)
}
