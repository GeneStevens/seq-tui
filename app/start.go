package app

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/sessionlog"
)

// Launch is what the entry points (the terminal's cmd/seq-tui and the browser's
// cmd/seq-tui-web) work out for a live session; Run then starts it the same way
// for both.
type Launch struct {
	Cfg       client.Config // the connection: URL, credential, character, session id, recordings
	Create    bool          // open on the character creation screen
	LogRoot   string        // where the sessions are recorded (the sessions view lists them)
	Log       *sessionlog.Log
	Bell      bool
	NoMouse   bool
	Duo       string // the account's second character, played by a stance; empty for none
	DuoAgent  string // an agent credential for it
	Program   []tea.ProgramOption
	Started   func(*tea.Program) // called with the program just before it runs (the browser sends resizes through it)
	Session   *client.Session    // a ready session (no login: a recorded log, in the browser's smoke test); Cfg is then unused
	LoginWait time.Duration      // how long to wait for the server (default 15 s)
}

// Run logs in, picks the first screen (the character, the picker, or the
// creator), and runs the program until it quits.
func (l Launch) Run() error {
	wait := l.LoginWait
	if wait == 0 {
		wait = 15 * time.Second
	}
	char := l.Cfg.Character
	sess := l.Session
	if sess == nil {
		sess = client.NewSession(l.Cfg)
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	prog := func(s *client.Session) *App { return New(s) }
	login := sess.Login
	if l.Create || char == "" { // no character named: connect, then pick or make one
		login = sess.Connect
	}
	if l.Session == nil {
		if err := login(ctx); err != nil {
			return err
		}
	}
	defer sess.Close()
	switch {
	case l.Create:
		prog = NewCreator
	case char == "":
		// The account's characters are the server's to say: the picker lists
		// them, and an account with none goes straight to making one.
		chars, err := sess.ListCharacters(ctx)
		if err != nil {
			return fmt.Errorf("listing the account's characters: %w", err)
		}
		if len(chars) > 0 {
			prog = func(s *client.Session) *App { return NewPicker(s, chars) }
		} else {
			prog = NewCreator
		}
	}
	a := prog(sess)
	a.SetLogs(l.LogRoot, l.Log)
	a.Bell = l.Bell
	a.NoMouse = l.NoMouse
	if l.Duo != "" && !l.Create && char != "" {
		dc := l.Cfg
		dc.Character, dc.Record, dc.Log = l.Duo, nil, nil
		dc.Agent = l.DuoAgent // an agent credential for it: its turn-based plans are then stand-ins
		d := client.NewDuo(dc, char)
		defer d.Close()
		a.SetDuo(d)
		dctx, stop := context.WithCancel(context.Background())
		defer stop()
		go d.Run(dctx)
	}
	p := tea.NewProgram(a, l.Program...)
	if l.Started != nil {
		l.Started(p)
	}
	_, err := p.Run()
	return err
}
