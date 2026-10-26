//go:build js

// Command seq-tui-web is seq-tui in a web browser (seq-design D106): the same
// program, compiled to WebAssembly, drawn by xterm.js in web/index.html. It
// talks to the server over the WebSocket the terminal client uses, on the page's
// own origin.
//
// The page and this program meet at a few names, all in the page's global scope:
//
//   - seqTerm: the xterm.js Terminal (write, cols, rows);
//   - seqInput(fn): this program gives the page a function to call with the text
//     xterm.js reports typed (its onData); it never blocks;
//   - seqResize(cols, rows): called by the page when the terminal's size changes;
//   - seqExited(message): called when the program ends, with the error or "";
//
// A test build (-tags webtest, make web-smoke) also reads seqReplay, the text of
// a recorded log, and plays it with no server (replay_js.go); the build that is
// served has no such hook.
//
// The settings come in go.env, as the page sets them: SEQ_SESSION_TOKEN (the
// pasted token), SEQ_URL (default wss://<the page's host>/seq/v1), SEQ_CHARACTER
// and SEQ_NEW (to open on character creation). Sessions are not recorded here,
// and duo mode is not available.
package main

import (
	"fmt"
	"io"
	"os"
	"syscall/js"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/GeneStevens/seq-tui/app"
	"github.com/GeneStevens/seq-tui/client"
)

// replay is set only by the webtest build (replay_js.go): it plays a recorded
// log when the page gives one, and says whether it did.
var replay func(l app.Launch) (bool, error)

func main() {
	err := run()
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	if f := js.Global().Get("seqExited"); f.Type() == js.TypeFunction {
		f.Invoke(msg)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "seq-tui:", err)
		os.Exit(1)
	}
}

func run() error {
	term := js.Global().Get("seqTerm")
	if term.IsUndefined() {
		return fmt.Errorf("the page has no seqTerm")
	}
	in, out := io.Pipe()
	defer in.Close()
	feedInput(out)
	opts := []tea.ProgramOption{
		tea.WithInput(in),
		tea.WithOutput(termWriter{term}),
		tea.WithEnvironment([]string{"TERM=xterm-256color", "COLORTERM=truecolor"}),
		tea.WithWindowSize(term.Get("cols").Int(), term.Get("rows").Int()),
		tea.WithColorProfile(colorprofile.TrueColor),
		tea.WithoutSignalHandler(),
	}
	l := app.Launch{Program: opts, Started: watchResize}
	if replay != nil {
		if done, err := replay(l); done {
			return err
		}
	}
	token := os.Getenv("SEQ_SESSION_TOKEN")
	if token == "" {
		return fmt.Errorf("no session token")
	}
	url := os.Getenv("SEQ_URL")
	if url == "" {
		loc := js.Global().Get("location")
		if loc.IsUndefined() {
			return fmt.Errorf("no server: set SEQ_URL")
		}
		scheme := "wss"
		if loc.Get("protocol").String() == "http:" {
			scheme = "ws"
		}
		url = scheme + "://" + loc.Get("host").String() + "/seq/v1"
	}
	char := os.Getenv("SEQ_CHARACTER")
	l.Cfg = client.Config{URL: url, Credential: token, Character: char, Session: client.NewSessionID()}
	l.Create = os.Getenv("SEQ_NEW") != ""
	return l.Run()
}

// feedInput gives the page the function that hands it what xterm.js reports
// typed. A call from the page must never block, so the text goes onto a channel
// (dropped, in the unlikely event it is full) and a goroutine writes it to the
// pipe the program reads.
func feedInput(w *io.PipeWriter) {
	ch := make(chan string, 4096)
	js.Global().Set("seqInput", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			select {
			case ch <- args[0].String():
			default:
			}
		}
		return nil
	}))
	go func() {
		for s := range ch {
			if _, err := w.Write([]byte(s)); err != nil {
				return
			}
		}
	}()
}

// watchResize turns the page's size reports into the window size the program
// is told of. The send is made off the page's call, which must not block.
func watchResize(p *tea.Program) {
	js.Global().Set("seqResize", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) >= 2 {
			w, h := args[0].Int(), args[1].Int()
			go p.Send(tea.WindowSizeMsg{Width: w, Height: h})
		}
		return nil
	}))
}

// termWriter draws on the terminal by xterm.js's write, as bytes so that a
// multi-byte character split across two writes still joins.
type termWriter struct{ term js.Value }

func (t termWriter) Write(b []byte) (int, error) {
	u := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(u, b)
	t.term.Call("write", u)
	return len(b), nil
}
