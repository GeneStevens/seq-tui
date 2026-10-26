# Bubble Tea on js/wasm: the local patch and the upstream pull request

seq-tui runs in a browser as WebAssembly (docs/browser.md). Bubble Tea v2.1.0
(`charm.land/bubbletea/v2`) does not build for `GOOS=js`: its tty and signal
files exist for the unix systems and windows only, so four hooks are undefined:

```
tea.go:714:8: p.listenForResize undefined
tea.go:796:8: undefined: suspendSupported
tty.go:19:2: undefined: suspendProcess
tty.go:29:11: p.initInput undefined
```

Until upstream has a fix, `third_party/bubbletea` is a copy of v2.1.0 with the
one added file below (`replace charm.land/bubbletea/v2 => ./third_party/bubbletea`
in go.mod). Nothing else in it is changed. With the file, a program gives
`tea.WithInput`, `tea.WithOutput`, `tea.WithWindowSize` and
`tea.WithColorProfile` and sends its own `tea.WindowSizeMsg`, as seq-tui does
(`cmd/seq-tui-web`); an unmodified `tea.Program.Run` then works in a browser and in
Node.

## The patch

Against bubbletea v2.1.0 (`diff -u`, applies with `patch -p1` or `git apply`):

```diff
--- /dev/null
+++ b/tty_js.go
@@ -0,0 +1,25 @@
+//go:build js
+// +build js
+
+package tea
+
+// There is no tty, no signal and no process group in a js/wasm host (a
+// browser, Node): the embedder supplies input and output through
+// WithInput/WithOutput and sends a WindowSizeMsg itself when its terminal
+// resizes. These are the platform hooks the other tty files provide, as no-ops.
+
+// listenForResize has no SIGWINCH to wait for; like the windows build it only
+// reports that it is done.
+func (p *Program) listenForResize(done chan struct{}) {
+	close(done)
+}
+
+// initInput has no tty to put in raw mode.
+func (p *Program) initInput() error {
+	return nil
+}
+
+const suspendSupported = false
+
+// suspendProcess is never called, as suspendSupported is false.
+func suspendProcess() {}
```

## Upstream pull request, ready to paste

**Title:** `feat: build for GOOS=js (browser/wasm) with no-op tty hooks`

**Description:**

> Bubble Tea has no js platform files, so `GOOS=js GOARCH=wasm go build` fails
> with `undefined: suspendSupported`, `suspendProcess`, `(*Program).initInput`
> and `(*Program).listenForResize`.
>
> This adds `tty_js.go` (`//go:build js`) providing those four hooks:
>
> - `listenForResize` closes `done` and returns, as the windows build does:
>   there is no SIGWINCH in a js host, so the embedder sends `WindowSizeMsg`
>   itself with `Program.Send` (an xterm.js `onResize`, for example).
> - `initInput` returns nil: there is no tty to put into raw mode. The input is
>   whatever `WithInput` gave (an `io.Pipe` fed from xterm.js's `onData`).
> - `suspendSupported` is false and `suspendProcess` is empty: there is no
>   process group to stop.
>
> Nothing changes for any other platform.
>
> Used with `WithInput`, `WithOutput`, `WithWindowSize`, `WithColorProfile` and
> `WithoutSignalHandler`, a full-screen program runs in a browser on xterm.js
> and in Node. Tested with
> `GOOS=js GOARCH=wasm go build ./...` and `go vet`, and by running the program
> under Node with piped input and output: first frame, a `WindowSizeMsg`, a key
> press and a quit.
>
> Possible follow-ups, not in this PR: `OpenTTY` and the `os.Stdin` fallback in
> `Run` for js (a program that gives `WithInput` never reaches them), and a
> test that builds for js in CI.
