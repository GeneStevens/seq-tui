// Command mkcapture plays a scripted session against a live development server
// and records every message the server sends, to make a fixture under
// testdata/logs. It drives the same client.Session the program uses.
//
//	SEQ_SESSION_TOKEN=... go run ./cmd/mkcapture -scenario release -character me-1 -out testdata/logs/live_release.jsonl
//	SEQ_SESSION_TOKEN=... go run ./cmd/mkcapture -scenario door    -character me-2 -out testdata/logs/live_door.jsonl
//	SEQ_SESSION_TOKEN=... go run ./cmd/mkcapture -scenario solo    -character me-1 -out testdata/logs/live_solo_trial.jsonl
//	SEQ_SESSION_TOKEN=... go run ./cmd/mkcapture -scenario turns   -character me-1 -out testdata/logs/live_quick_turns.jsonl
//
// release: stand beside a mob until it kills the character, release, wait for
// the respawn. door: walk to the zone line, cross it, and cross back.
// solo: open solo-trial from the menu, walk in, kill and loot the scuttler
// until the instance is cleared, and be handed back. turns: the same in
// quick-turns, a plan a round, one missed deadline, a dropped connection that
// resumes and a fresh login into the open round.
// The server is the one `make play` starts (seqd, -insecure's loopback rules
// apply). An account's characters are one group, and an account has one
// instance at a time, so run each of solo and turns on a fresh server with
// the character standing in the world. A fixture is reviewed before it is
// committed; tests only read it.
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "mkcapture:", err)
		os.Exit(1)
	}
}

func run() error {
	addr := flag.String("url", "wss://127.0.0.1:8443/seq/v1", "the development server's gateway (loopback only)")
	char := flag.String("character", "me-1", "the character to play")
	scenario := flag.String("scenario", "release", "release, door, solo or turns")
	out := flag.String("out", "", "the log to write")
	flag.Parse()
	token := os.Getenv("SEQ_SESSION_TOKEN")
	if token == "" || *out == "" {
		return fmt.Errorf("set SEQ_SESSION_TOKEN and -out")
	}
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer f.Close()
	cfg := client.Config{URL: *addr, TLS: &tls.Config{InsecureSkipVerify: true}, Credential: token, Character: *char, Record: record.NewWriter(f)}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	c := &capture{cfg: cfg, ctx: ctx}
	if err := c.login(); err != nil {
		return err
	}
	defer func() { c.s.Close() }()
	switch *scenario {
	case "release":
		return c.release()
	case "door":
		return c.door()
	case "turns":
		return c.turns()
	case "solo":
		return c.solo()
	}
	return fmt.Errorf("unknown scenario %q", *scenario)
}

type capture struct {
	cfg client.Config
	s   *client.Session
	ctx context.Context
}

// login connects a new session, with a new model, to the same recording.
func (c *capture) login() error {
	cfg := c.cfg
	cfg.Session, cfg.Model = "", nil
	c.s = client.NewSession(cfg)
	return c.s.Login(c.ctx)
}

// until polls the model until pred holds.
func (c *capture) until(what string, pred func(*model.Model) bool) error {
	for {
		ok := false
		c.s.View(func(m *model.Model) { ok = pred(m) })
		if ok {
			return nil
		}
		select {
		case <-c.ctx.Done():
			return fmt.Errorf("waiting for %s: %w", what, c.ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// do sends an intent and requires the server to accept it.
func (c *capture) do(what string, send func() (string, error)) error {
	id, err := send()
	if err != nil {
		return err
	}
	st, err := c.s.Result(c.ctx, id)
	if err != nil {
		return err
	}
	fmt.Printf("%s: %s %s\n", what, st.State, st.Reason)
	if st.State != model.IntentAccepted {
		return fmt.Errorf("%s was denied: %s", what, st.Reason)
	}
	return nil
}

func (c *capture) release() error {
	// The nearest scuttler of the test zone wakes at 12 m; stand 12 m from it.
	if err := c.do("walk to the mob", func() (string, error) { return c.s.Move(c.ctx, 22000, 50000) }); err != nil {
		return err
	}
	if err := c.until("death", func(m *model.Model) bool { return m.Self.Dead }); err != nil {
		return err
	}
	time.Sleep(time.Second)
	if err := c.do("release", func() (string, error) { return c.s.Release(c.ctx) }); err != nil {
		return err
	}
	if err := c.until("respawn", func(m *model.Model) bool { return !m.Self.Dead && !m.Self.Released }); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return nil
}

func (c *capture) at(x, y int64) func(*model.Model) bool {
	return func(m *model.Model) bool {
		dx, dy := m.Self.Pos.GetX()-x, m.Self.Pos.GetY()-y
		return dx*dx+dy*dy < 1500*1500
	}
}

func (c *capture) walk(x, y int64) error {
	if err := c.do(fmt.Sprintf("walk to (%d, %d)", x, y), func() (string, error) { return c.s.Move(c.ctx, x, y) }); err != nil {
		return err
	}
	return c.until("arrival", c.at(x, y))
}

func (c *capture) cross(from string) error {
	var door string
	c.s.View(func(m *model.Model) {
		best := int64(-1)
		for _, d := range m.Map.GetDoors() {
			dx, dy := d.GetPos().GetX()-m.Self.Pos.GetX(), d.GetPos().GetY()-m.Self.Pos.GetY()
			if dd := dx*dx + dy*dy; best < 0 || dd < best {
				best, door = dd, d.GetId()
			}
		}
	})
	if door == "" {
		return fmt.Errorf("no door on the map of %s", from)
	}
	if err := c.do("use "+door, func() (string, error) { return c.s.UseDoor(c.ctx, door) }); err != nil {
		return err
	}
	return c.until("the other side", func(m *model.Model) bool {
		return m.Space != from && m.Ready() && m.Map != nil && m.Map.GetSpace() == m.Space
	})
}

func (c *capture) door() error {
	var first string
	c.s.View(func(m *model.Model) { first = m.Space })
	// Round the wall at x = 40 m and the mobs near it, to the zone line.
	for _, p := range [][2]int64{{30000, 85000}, {100000, 80000}, {194000, 10000}} {
		if err := c.walk(p[0], p[1]); err != nil {
			return err
		}
	}
	if err := c.cross(first); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	var second string
	c.s.View(func(m *model.Model) { second = m.Space })
	if err := c.cross(second); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return nil
}
