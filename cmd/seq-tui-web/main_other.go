//go:build !js

// Command seq-tui-web is the browser build of seq-tui (see main_js.go); it only
// runs as WebAssembly. This file keeps the package building on other systems, so
// that go build ./... and go vet ./... pass natively.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "seq-tui-web is built with GOOS=js GOARCH=wasm (make web); see docs/browser.md")
	os.Exit(2)
}
