//go:build js && webtest

package main

import (
	"strings"
	"syscall/js"

	"github.com/GeneStevens/seq-tui/app"
	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
)

// The Node smoke test feeds a recorded log through the page global seqReplay.
// This hook is only in the build made with -tags webtest (make web-smoke): a
// page that set seqReplay could otherwise show any log without a server.
func init() { replay = playReplay }

func playReplay(l app.Launch) (bool, error) {
	if js.Global().Get("seqHarness").Type() == js.TypeString {
		return harness(l) // the display check (harness_js.go)
	}
	text := js.Global().Get("seqReplay")
	if text.Type() != js.TypeString {
		return false, nil
	}
	msgs, err := record.Read(strings.NewReader(text.String()))
	if err != nil {
		return true, err
	}
	m := model.New()
	for _, msg := range msgs {
		m.Apply(msg)
	}
	l.Cfg.Character = "replay" // not picked from a server
	l.Session = client.NewSession(client.Config{Model: m})
	return true, l.Run()
}
