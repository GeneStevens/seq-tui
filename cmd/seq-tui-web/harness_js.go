//go:build js && webtest

package main

import (
	"strings"
	"syscall/js"
	"time"

	tea "charm.land/bubbletea/v2"
	pb "github.com/GeneStevens/seq/protocol/seqv1"

	"github.com/GeneStevens/seq-tui/app"
	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
)

// rec records the content of every view the program draws in seqLastView.
type rec struct{ a *app.App }

func (r rec) Init() tea.Cmd { return r.a.Init() }
func (r rec) Update(m tea.Msg) (tea.Model, tea.Cmd) {
	_, c := r.a.Update(m)
	return r, c
}
func (r rec) View() tea.View {
	v := r.a.View()
	js.Global().Set("seqLastView", v.Content)
	return v
}

func harness(l app.Launch) (bool, error) {
	mode := js.Global().Get("seqHarness").String()
	text := js.Global().Get("seqReplay").String()
	msgs, err := record.Read(strings.NewReader(text))
	if err != nil {
		return true, err
	}
	m := model.New()
	for _, msg := range msgs {
		m.Apply(msg)
	}
	sess := client.NewSession(client.Config{Model: m})
	var a *app.App
	if mode == "picker" {
		now := time.Now().UnixMilli()
		var chars []*pb.CharacterEntry
		for i, n := range []string{"me-1", "me-2", "me-3", "Brannoc"} {
			chars = append(chars, &pb.CharacterEntry{Character: n, Name: n, Class: "warrior", Level: uint32(3 + i), Zone: "The Long Meadow", LastPlayedMs: now - int64(i)*3600000})
		}
		a = app.NewPicker(sess, chars)
	} else {
		a = app.New(sess)
	}
	p := tea.NewProgram(rec{a}, l.Program...)
	l.Started(p)
	_, err = p.Run()
	return true, err
}
