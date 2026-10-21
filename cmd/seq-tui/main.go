// Command seq-tui is the terminal client for a SEQ server (seq-design doc 07).
//
//	SEQ_SESSION_TOKEN=seq_st_... seq-tui -insecure -character me-1
//	SEQ_SESSION_TOKEN=seq_st_... seq-tui -insecure -new
//
// connects to a local server and opens the screen; -new opens it on the
// character creation screen (a name and a class) instead of a character. -replay draws a recorded
// log instead, with no server and no terminal; -record keeps the messages of a
// live session in such a log.
//
// Every session is also recorded by default (D64), under its play-session id, in
// its own directory of the data dir (-log-dir changes it; -no-record turns it
// off): record.jsonl and session.json. Ctrl+L opens the sessions view, which lists
// them with the server's own log of the account's sessions. -replay takes such a
// directory, or a session id, as well as a file.
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/app"
	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
	"github.com/GeneStevens/seq-tui/screen"
	"github.com/GeneStevens/seq-tui/sessionlog"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seq-tui:", err)
		os.Exit(1)
	}
}

func run() error {
	addr := flag.String("url", "wss://127.0.0.1:8443/seq/v1", "the server's gateway")
	char := flag.String("character", "", "the character to play (its id); without it, and without -new, a picker lists the account's characters, as the server has them")
	create := flag.Bool("new", false, "make a new character (a name and one of the six classes) and play it")
	insecure := flag.Bool("insecure", false, "development only: accept a self-signed certificate (loopback addresses only)")
	duo := flag.String("duo", "", "the account's second character to play by a stance beside this one (P brings it in; H, Y, U, J, Z command it by its stance; X directs it from this session); SEQ_DUO_AGENT_SECRET, if set, is an agent credential for it, so that in a turn-based instance its plans go as stand-ins")
	rec := flag.String("record", "", "write every message the server sends to this file")
	noRec := flag.Bool("no-record", false, "do not record the session in the log directory (it is recorded by default)")
	logDir := flag.String("log-dir", "", "where the sessions are recorded (default: seq-tui/sessions in your data dir)")
	replay := flag.String("replay", "", "draw this recorded log (a file, a session directory, or a session id) instead of connecting")
	text := flag.Bool("text", false, "with -replay: print the model as plain text instead of the screen")
	size := flag.String("size", "80x24", "with -replay: the screen size, WxH")
	noMouse := flag.Bool("no-mouse", false, "do not ask the terminal for mouse reports (clicks, hover, the wheel); the keys do everything, and the terminal's own text selection works")
	bell := flag.Bool("bell", false, "ring the terminal's bell when a mob begins attacking you (off by default)")
	flag.Parse()

	root := *logDir
	if root == "" {
		var err error
		if root, err = sessionlog.DefaultDir(); err != nil && !*noRec {
			return fmt.Errorf("no place for the session log (use -log-dir or -no-record): %w", err)
		}
	}
	if *replay != "" {
		path, err := sessionlog.Find(root, *replay)
		if err != nil {
			return err
		}
		return runReplay(path, *text, *size)
	}
	token := os.Getenv("SEQ_SESSION_TOKEN")
	if token == "" {
		return fmt.Errorf("set SEQ_SESSION_TOKEN to your session token (it is never taken as a flag)")
	}
	var tlsCfg *tls.Config
	if *insecure {
		u, err := url.Parse(*addr)
		if err != nil {
			return err
		}
		switch u.Hostname() {
		case "127.0.0.1", "localhost", "::1":
			tlsCfg = &tls.Config{InsecureSkipVerify: true} // a development server's in-memory certificate
		default:
			return fmt.Errorf("-insecure is for a local server only, not %s", u.Hostname())
		}
	}
	cfg := client.Config{URL: *addr, TLS: tlsCfg, Credential: token, Character: *char}
	if *rec != "" {
		f, err := os.Create(*rec)
		if err != nil {
			return err
		}
		defer f.Close()
		cfg.Record = record.NewWriter(f)
	}
	cfg.Session = client.NewSessionID()
	who := *char
	if *create || *char == "" {
		who = "" // not known until it is made or picked
	}
	slog, err := sessionlog.Start(root, *noRec, sessionlog.Options{Credential: token, Meta: sessionlog.Meta{
		Session: cfg.Session, Server: *addr, Character: who, Client: "seq-tui", Versions: sessionlog.Versions(client.ProtocolVersion)}})
	if err != nil {
		return fmt.Errorf("opening the session log (-no-record turns it off): %w", err)
	}
	if slog != nil {
		defer slog.Close()
		cfg.Log = slog.Rec
	}
	sess := client.NewSession(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	prog := func(s *client.Session) *app.App { return app.New(s) }
	login := sess.Login
	if *create || *char == "" { // no character named: connect, then pick or make one
		login = sess.Connect
	}
	if err := login(ctx); err != nil {
		return err
	}
	defer sess.Close()
	switch {
	case *create:
		prog = app.NewCreator
	case *char == "":
		// The account's characters are the server's to say: the picker lists
		// them, and an account with none goes straight to making one.
		chars, err := sess.ListCharacters(ctx)
		if err != nil {
			return fmt.Errorf("listing the account's characters: %w", err)
		}
		if len(chars) > 0 {
			prog = func(s *client.Session) *app.App { return app.NewPicker(s, chars) }
		} else {
			prog = app.NewCreator
		}
	}
	a := prog(sess)
	a.SetLogs(root, slog)
	a.Bell = *bell
	a.NoMouse = *noMouse
	if *duo != "" && !*create && *char != "" {
		dc := cfg
		dc.Character, dc.Record, dc.Log = *duo, nil, nil
		dc.Agent = os.Getenv("SEQ_DUO_AGENT_SECRET") // an agent credential for it: its turn-based plans are then stand-ins
		d := client.NewDuo(dc, *char)
		defer d.Close()
		a.SetDuo(d)
		dctx, stop := context.WithCancel(context.Background())
		defer stop()
		go d.Run(dctx)
	}
	_, err = tea.NewProgram(a).Run()
	return err
}

func runReplay(path string, text bool, size string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	msgs, err := record.Read(f)
	if err != nil {
		return err
	}
	m := model.New()
	for _, msg := range msgs {
		m.Apply(msg)
	}
	if text {
		fmt.Print(m.Text())
		return nil
	}
	ws, hs, ok := strings.Cut(size, "x")
	w, err1 := strconv.Atoi(ws)
	h, err2 := strconv.Atoi(hs)
	if !ok || err1 != nil || err2 != nil {
		return fmt.Errorf("-size wants WxH, like 80x24")
	}
	fmt.Println(screen.Render(m, screen.View{Conn: screen.ConnOnline}, w, h, screen.Plain()))
	return nil
}
