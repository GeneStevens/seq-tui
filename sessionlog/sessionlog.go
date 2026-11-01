// Package sessionlog is the TUI's own record of every session (seq-design D64):
// one directory a session under the player's data dir, with record.jsonl (every
// message the server sent, in the format of `seq-tui -record` and of seq-mcp's
// log) and session.json (the play-session id, the server, the character, the
// versions). It lists them again for the sessions view, and exports them.
//
// Never a credential or a token: record.jsonl holds what the
// server sent, with the session's credential scrubbed from every line anyway,
// and session.json holds the server's address without user info or query.
package sessionlog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/GeneStevens/seq-tui/record"
)

// File names inside a session's directory.
const (
	RecordFile = "record.jsonl"
	MetaFile   = "session.json"
)

// Default caps: one session's recording, and the whole directory.
const (
	DefaultSessionBytes = 32 << 20
	DefaultDirBytes     = 256 << 20
)

const scrubbed = "[scrubbed]"

// Meta is session.json.
type Meta struct {
	Session   string            `json:"session"`
	Server    string            `json:"server"`
	Character string            `json:"character,omitempty"`
	Zone      string            `json:"zone,omitempty"`
	Client    string            `json:"client"`
	Versions  map[string]string `json:"versions,omitempty"`
	Start     time.Time         `json:"start"`
	End       time.Time         `json:"end,omitzero"`
}

// Options say how a log is opened.
type Options struct {
	Meta Meta
	// Credential is scrubbed from everything written.
	Credential string
	// SessionBytes and DirBytes cap one recording and the directory; 0 is the
	// default, negative no cap.
	SessionBytes, DirBytes int64
	// Now is the clock for the directory name and the times; nil is the wall clock.
	Now func() time.Time
}

// Log is one session's directory, open for writing.
type Log struct {
	dir string
	f   *os.File

	mu   sync.Mutex
	meta Meta
	now  func() time.Time

	// Rec receives the session's messages (client.Config.Log). The file is
	// buffered: Flush it (Close does) to be sure it holds everything.
	Rec *record.Writer
	cap *capWriter // nil when the recording has no size cap
}

// Stopped reports whether the recording has stopped at its size cap: the file
// then ends with a line that says so, and nothing more is written to it.
func (l *Log) Stopped() bool { return l.cap != nil && l.cap.stopped() }

// DefaultDir is where sessions are kept: XDG's data home on Linux, Application
// Support on macOS.
func DefaultDir() (string, error) {
	if x := os.Getenv("XDG_DATA_HOME"); x != "" && runtime.GOOS != "darwin" {
		return filepath.Join(x, "seq-tui", "sessions"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "seq-tui", "sessions"), nil
	}
	return filepath.Join(home, ".local", "share", "seq-tui", "sessions"), nil
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// DirName is a session's directory name: its start and its id, cleaned.
func DirName(start time.Time, session string) string {
	s := unsafeName.ReplaceAllString(session, "_")
	if len(s) > 64 {
		s = s[:64]
	}
	return start.UTC().Format("20060102-150405") + "-" + s
}

// SafeServer is a server address with no user info and no query.
func SafeServer(addr string) string {
	u, err := url.Parse(addr)
	if err != nil {
		return ""
	}
	u.User, u.RawQuery, u.Fragment = nil, "", ""
	return u.String()
}

// Versions are the versions session.json names.
func Versions(protocol int) map[string]string {
	v := map[string]string{"protocol": fmt.Sprint(protocol), "go": runtime.Version()}
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, d := range bi.Deps {
			if d.Path == "github.com/GeneStevens/seq" {
				v["seq"] = d.Version
			}
		}
		if bi.Main.Version != "" {
			v["seq-tui"] = bi.Main.Version
		}
	}
	return v
}

// Open makes the session's directory under root, prunes the oldest sessions past
// the directory cap, and opens record.jsonl headed by the session id.
func Open(root string, o Options) (*Log, error) {
	now := o.Now
	if now == nil {
		now = time.Now
	}
	meta := o.Meta
	if meta.Start.IsZero() {
		meta.Start = now()
	}
	meta.Server = SafeServer(meta.Server)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	if o.DirBytes >= 0 {
		Prune(root, cmpCap(o.DirBytes, DefaultDirBytes))
	}
	dir := filepath.Join(root, DirName(meta.Start, meta.Session))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, RecordFile), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	// Lines go cap, then scrub, then a buffer, then the file: the cap and the
	// scrub see each line whole, and the file is written in blocks.
	bw := bufio.NewWriterSize(f, 64<<10)
	w := &scrubWriter{w: bw, secret: o.Credential}
	capBytes := cmpCap(o.SessionBytes, DefaultSessionBytes)
	var out io.Writer = w
	var cw *capWriter
	if o.SessionBytes >= 0 {
		cw = &capWriter{w: w, left: capBytes}
		out = cw
	}
	if _, err := fmt.Fprintf(out, "# seq session %s\n", unsafeName.ReplaceAllString(meta.Session, "_")); err != nil {
		f.Close()
		return nil, err
	}
	l := &Log{dir: dir, f: f, meta: meta, now: now, Rec: record.NewFlushingWriter(out, bw.Flush), cap: cw}
	if err := l.write(); err != nil {
		f.Close()
		return nil, err
	}
	return l, nil
}

// Start is Open unless off (-no-record): then nothing is made or written and the
// log is nil, which the app and the session take as no recording.
func Start(root string, off bool, o Options) (*Log, error) {
	if off {
		return nil, nil
	}
	return Open(root, o)
}

func cmpCap(v, def int64) int64 {
	if v == 0 {
		return def
	}
	return v
}

// Dir is the session's directory.
func (l *Log) Dir() string { return l.dir }

// SetCharacter notes the character the session plays (a new character is only
// known once it is made).
func (l *Log) SetCharacter(id string) {
	l.mu.Lock()
	changed := l.meta.Character != id
	l.meta.Character = id
	l.mu.Unlock()
	if changed {
		_ = l.write()
	}
}

// SetZone notes the zone the session is in, as it moves.
func (l *Log) SetZone(zone string) {
	l.mu.Lock()
	changed := l.meta.Zone != zone
	l.meta.Zone = zone
	l.mu.Unlock()
	if changed {
		_ = l.write()
	}
}

// Close ends the log: the end time goes into session.json.
func (l *Log) Close() error {
	l.mu.Lock()
	l.meta.End = l.now()
	l.mu.Unlock()
	err := l.Rec.Flush()
	if werr := l.write(); err == nil {
		err = werr
	}
	if cerr := l.f.Close(); err == nil {
		err = cerr
	}
	return err
}

func (l *Log) write() error {
	l.mu.Lock()
	b, err := json.MarshalIndent(l.meta, "", "  ")
	l.mu.Unlock()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(l.dir, MetaFile), append(b, '\n'), 0o600)
}

// scrubWriter removes a secret from what passes through. A record line is
// written whole, so a secret is never split across two writes.
type scrubWriter struct {
	w      io.Writer
	secret string
}

func (s *scrubWriter) Write(p []byte) (int, error) {
	if s.secret == "" || !bytes.Contains(p, []byte(s.secret)) {
		return s.w.Write(p)
	}
	if _, err := s.w.Write(bytes.ReplaceAll(p, []byte(s.secret), []byte(scrubbed))); err != nil {
		return 0, err
	}
	return len(p), nil
}

// capWriter stops writing past a size and says so, once.
type capWriter struct {
	mu   sync.Mutex
	w    io.Writer
	left int64
	done bool
}

func (c *capWriter) stopped() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.done
}

func (c *capWriter) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.done {
		return len(p), nil
	}
	if int64(len(p)) > c.left {
		c.done = true
		_, err := io.WriteString(c.w, "# stopped: this session's size cap was reached; nothing more is written\n")
		return len(p), err
	}
	c.left -= int64(len(p))
	return c.w.Write(p)
}

// Entry is a session on disk.
type Entry struct {
	Dir  string
	Meta Meta
	Size int64
}

// RecordPath is the entry's recording.
func (e Entry) RecordPath() string { return filepath.Join(e.Dir, RecordFile) }

// List reads the sessions under root, newest first. A directory with no readable
// session.json is not a session.
func List(root string) []Entry {
	des, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []Entry
	for _, de := range des {
		if !de.IsDir() {
			continue
		}
		dir := filepath.Join(root, de.Name())
		b, err := os.ReadFile(filepath.Join(dir, MetaFile))
		if err != nil {
			continue
		}
		var m Meta
		if json.Unmarshal(b, &m) != nil {
			continue
		}
		e := Entry{Dir: dir, Meta: m}
		if st, err := os.Stat(e.RecordPath()); err == nil {
			e.Size = st.Size()
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Meta.Start.After(out[j].Meta.Start) })
	return out
}

// Prune removes the oldest sessions under root until the directory holds at
// most max bytes. Only directories with a session.json are touched.
func Prune(root string, max int64) {
	es := List(root)
	var total int64
	sizes := make([]int64, len(es))
	for i, e := range es {
		sizes[i] = dirSize(e.Dir)
		total += sizes[i]
	}
	for i := len(es) - 1; i >= 0 && total > max; i-- {
		if os.RemoveAll(es[i].Dir) == nil {
			total -= sizes[i]
		}
	}
}

func dirSize(dir string) int64 {
	var n int64
	_ = filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if fi, err := d.Info(); err == nil {
				n += fi.Size()
			}
		}
		return nil
	})
	return n
}

// Find resolves a replay argument: a session directory, a recording file or a
// session id under root. It returns the recording's path.
func Find(root, arg string) (string, error) {
	if st, err := os.Stat(arg); err == nil {
		if st.IsDir() {
			return filepath.Join(arg, RecordFile), nil
		}
		return arg, nil
	}
	for _, e := range List(root) {
		if e.Meta.Session == arg {
			return e.RecordPath(), nil
		}
	}
	return "", fmt.Errorf("no recording %q (a file, a session directory, or a session id in %s)", arg, root)
}

// ExportDir is where exports go: beside the sessions, not among them.
func ExportDir(root string) string { return filepath.Join(filepath.Dir(root), "exports") }

// ExportLocal copies a recording to the export directory and returns the path.
func ExportLocal(root string, e Entry) (string, error) {
	dst := filepath.Join(ExportDir(root), filepath.Base(e.Dir)+".jsonl")
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return "", err
	}
	b, err := os.ReadFile(e.RecordPath())
	if err != nil {
		return "", err
	}
	return dst, os.WriteFile(dst, b, 0o600)
}

// ServerLine is one line of the server's log of a session.
type ServerLine struct {
	TMs  int64
	Text string
}

// String is the line as the exports print it: the local time with its zone
// offset (D121) and the words, which keep the game time the server put in them.
func (l ServerLine) String() string {
	return time.UnixMilli(l.TMs).Local().Format("2006-01-02 15:04:05.000 -0700") + "  " + l.Text
}

// ExportServer writes the server's log of a session, readable, to the export
// directory and returns the path.
func ExportServer(root, session string, lines []ServerLine) (string, error) {
	dst := filepath.Join(ExportDir(root), "server-"+unsafeName.ReplaceAllString(session, "_")+".txt")
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return "", err
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "session %s (the server's log)\n", session)
	for _, l := range lines {
		sb.WriteString(l.String() + "\n")
	}
	return dst, os.WriteFile(dst, []byte(sb.String()), 0o600)
}
