# seq-tui's copy of bubbletea

This is `charm.land/bubbletea/v2` **v2.1.0**, unchanged except for one added
file, so that it builds for `GOOS=js GOARCH=wasm` (the browser build, see
docs/browser.md). The root `go.mod` points to it with a `replace`.

The change, and the text for an upstream pull request, are in
docs/upstream-bubbletea-js.md. To update: copy the new release over this
directory, keep `tty_js.go`, and check that `make web-smoke` still passes. When
upstream has a js build, delete this directory and the `replace`.
