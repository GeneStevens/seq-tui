// Command seq-tui is the terminal client for a SEQ server (seq-design doc 07).
//
//	SEQ_SESSION_TOKEN=seq_st_... seq-tui -insecure -character me-1
//
// connects to a local server and opens the screen. -replay draws a recorded
// log instead, with no server and no terminal; -record keeps the messages of a
// live session in such a log.
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
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seq-tui:", err)
		os.Exit(1)
	}
}

func run() error {
	addr := flag.String("url", "wss://127.0.0.1:8443/seq/v1", "the server's gateway")
	char := flag.String("character", "me-1", "the character to play")
	insecure := flag.Bool("insecure", false, "development only: accept a self-signed certificate (loopback addresses only)")
	rec := flag.String("record", "", "write every message the server sends to this file")
	replay := flag.String("replay", "", "draw this recorded log instead of connecting")
	text := flag.Bool("text", false, "with -replay: print the model as plain text instead of the screen")
	size := flag.String("size", "80x24", "with -replay: the screen size, WxH")
	flag.Parse()

	if *replay != "" {
		return runReplay(*replay, *text, *size)
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
	sess := client.NewSession(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := sess.Login(ctx); err != nil {
		return err
	}
	defer sess.Close()
	_, err := tea.NewProgram(app.New(sess)).Run()
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
