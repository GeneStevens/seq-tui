//go:build js
// +build js

package tea

// There is no tty, no signal and no process group in a js/wasm host (a
// browser, Node): the embedder supplies input and output through
// WithInput/WithOutput and sends a WindowSizeMsg itself when its terminal
// resizes. These are the platform hooks the other tty files provide, as no-ops.

// listenForResize has no SIGWINCH to wait for; like the windows build it only
// reports that it is done.
func (p *Program) listenForResize(done chan struct{}) {
	close(done)
}

// initInput has no tty to put in raw mode.
func (p *Program) initInput() error {
	return nil
}

const suspendSupported = false

// suspendProcess is never called, as suspendSupported is false.
func suspendProcess() {}
