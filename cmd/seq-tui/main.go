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
	"os"
	"os/signal"
	"strconv"
	"strings"

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
	args := os.Args[1:]
	sub := ""
	if len(args) > 0 && (args[0] == "login" || args[0] == "logout") {
		sub, args = args[0], args[1:]
	}
	flag.CommandLine.Parse(args)
	if flag.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q (the commands are login and logout)", flag.Arg(0))
	}

	root := *logDir
	if root == "" {
		var err error
		if root, err = sessionlog.DefaultDir(); err != nil && !*noRec && sub == "" {
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
	tlsCfg, err := tlsFor(*addr, *insecure)
	if err != nil {
		return err
	}
	cfgDir, err := client.ConfigDir()
	if err != nil {
		return fmt.Errorf("no place for the saved sign-in (set SEQ_SESSION_TOKEN): %w", err)
	}
	si := signIn{Gateway: *addr, Dir: cfgDir, Out: os.Stdout, HTTP: client.HTTPClient(tlsCfg), Open: openBrowser}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	switch sub {
	case "logout":
		return si.Logout()
	case "login":
		_, err := si.Login(ctx)
		return err
	}
	token, saved, err := si.Token(ctx, os.Getenv("SEQ_SESSION_TOKEN"))
	if err != nil {
		return err
	}
	stop() // Ctrl+C belongs to the screen from here
	p := play{flags: playFlags{addr: *addr, char: *char, create: *create, duo: *duo, rec: *rec, noRec: *noRec, noMouse: *noMouse, bell: *bell}, root: root, tls: tlsCfg}
	err = p.run(token)
	if saved && client.IsCredentialRefused(err) {
		fmt.Fprintln(os.Stderr, "Your saved sign-in is no longer valid; signing in again.")
		if path, perr := si.path(); perr == nil {
			client.RemoveSession(path)
		}
		ctx, stop = signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		sv, lerr := si.Login(ctx)
		if lerr != nil {
			return lerr
		}
		stop()
		err = p.run(sv.Token)
	}
	return err
}

type playFlags struct {
	addr, char, duo, rec         string
	create, noRec, noMouse, bell bool
}

// play is one start of the screen on a server.
type play struct {
	flags playFlags
	root  string
	tls   *tls.Config
}

func (p play) run(token string) error {
	f := p.flags
	cfg := client.Config{URL: f.addr, TLS: p.tls, Credential: token, Character: f.char}
	if f.rec != "" {
		rf, err := os.Create(f.rec)
		if err != nil {
			return err
		}
		defer rf.Close()
		cfg.Record = record.NewWriter(rf)
	}
	cfg.Session = client.NewSessionID()
	who := f.char
	if f.create || f.char == "" {
		who = "" // not known until it is made or picked
	}
	slog, err := sessionlog.Start(p.root, f.noRec, sessionlog.Options{Credential: token, Meta: sessionlog.Meta{
		Session: cfg.Session, Server: f.addr, Character: who, Client: "seq-tui", Versions: sessionlog.Versions(client.ProtocolVersion)}})
	if err != nil {
		return fmt.Errorf("opening the session log (-no-record turns it off): %w", err)
	}
	if slog != nil {
		defer slog.Close()
		cfg.Log = slog.Rec
	}
	return app.Launch{Cfg: cfg, Create: f.create, LogRoot: p.root, Log: slog, Bell: f.bell, NoMouse: f.noMouse,
		Duo: f.duo, DuoAgent: os.Getenv("SEQ_DUO_AGENT_SECRET")}.Run()
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
