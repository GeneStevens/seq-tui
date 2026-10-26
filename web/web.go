// Package web holds the browser page of seq-tui (index.html and its assets, see
// docs/browser.md) and the list of keys the page keeps from the browser.
package web

import (
	_ "embed"
	"encoding/json"
)

//go:embed keys.json
var keysJSON []byte

// Keys are the browser's own shortcuts that the page must care about, in the
// terminal's key names (as the help's binding table has them).
type Keys struct {
	// Intercept are the keys the page stops the browser acting on (preventDefault),
	// so that they reach the game. index.html reads the same file.
	Intercept []string `json:"intercept"`
	// Unreachable are the keys a page cannot take from the browser at all (they
	// close or open tabs and windows); no binding may use one.
	Unreachable []string `json:"unreachable"`
}

// InterceptedKeys reads keys.json.
func InterceptedKeys() (Keys, error) {
	var k Keys
	err := json.Unmarshal(keysJSON, &k)
	return k, err
}
