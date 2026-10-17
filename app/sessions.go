package app

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/record"
	"github.com/GeneStevens/seq-tui/screen"
	"github.com/GeneStevens/seq-tui/sessionlog"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The sessions view (D64): the player's own logs. Local recordings come from the
// log directory; the server's list and logs are asked of the server, which
// answers for the account of the credential only. Nothing here decides
// anything: a replay draws a recording, an export writes a file.

// zonesLooked is how many of the server's most recent sessions have their log
// read for the zone they were in (the list itself has none).
const zonesLooked = 8

// replay is a recording being drawn in place of the live screen.
type replay struct {
	id   string
	msgs []*pb.ServerMessage
	pos  int
	m    *model.Model
}

type (
	// sessionsLoaded answers the request for the server's list.
	sessionsLoaded struct {
		rows []screen.SessionRow
		err  error
	}
	// logLoaded answers the request for one session's log; export says the player
	// asked to write it to a file.
	logLoaded struct {
		id     string
		lines  []sessionlog.ServerLine
		total  int
		err    error
		export bool
	}
)

// SetLogs gives the app the session log it writes (nil if recording is off) and
// the directory it lists, for the sessions view.
func (a *App) SetLogs(root string, l *sessionlog.Log) { a.logRoot, a.log = root, l }

// noteLog tells the session's log which character and zone it is in. The model
// is locked by the caller.
func (a *App) noteLog(m *model.Model) {
	if a.log == nil {
		return
	}
	if m.Self.ID != "" {
		a.log.SetCharacter(m.Self.ID)
	}
	if m.Space != "" {
		a.log.SetZone(m.Space)
	}
}

func (a *App) openSessions() tea.Cmd {
	dir := a.logRoot
	if a.LogLabel != "" {
		dir = a.LogLabel
	}
	a.ses = &screen.SessionsUI{Dir: dir}
	return a.loadSessions()
}

// localRows are the recordings on this machine, newest first.
func (a *App) localRows() []screen.SessionRow {
	var rows []screen.SessionRow
	for _, e := range sessionlog.List(a.logRoot) {
		r := screen.SessionRow{Local: true, ID: e.Meta.Session, Character: e.Meta.Character, Zone: e.Meta.Zone, Start: e.Meta.Start, Ended: !e.Meta.End.IsZero()}
		switch {
		case !e.Meta.End.IsZero():
			r.Length = e.Meta.End.Sub(e.Meta.Start)
		case a.log != nil && e.Dir == a.log.Dir():
			r.Length = a.now().Sub(e.Meta.Start)
		}
		rows = append(rows, r)
	}
	return rows
}

// loadSessions reads the local list now and asks the server for its own.
func (a *App) loadSessions() tea.Cmd {
	ui := a.ses
	ui.Rows = a.localRows()
	ui.Loading, ui.Err = true, ""
	a.inflight++
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 4*commandTimeout)
		defer cancel()
		list, err := a.sess.MySessions(ctx)
		if err != nil {
			return sessionsLoaded{err: err}
		}
		var rows []screen.SessionRow
		for i, s := range list {
			r := screen.SessionRow{ID: s.GetSession(), Character: s.GetCharacter(), Lines: int(s.GetLines()), Ended: s.GetEnded(),
				Start:  time.UnixMilli(s.GetFirstMs()),
				Length: time.Duration(s.GetLastMs()-s.GetFirstMs()) * time.Millisecond}
			if i < zonesLooked {
				if rep, err := a.sess.SessionLog(ctx, s.GetSession()); err == nil {
					r.Zone = zoneOf(rep.GetLines())
				}
			}
			rows = append(rows, r)
		}
		return sessionsLoaded{rows: rows}
	}
}

var (
	zoneConnect = regexp.MustCompile(` in (\S+)`)
	zoneMoved   = regexp.MustCompile(`moved from \S+ to (\S+)`)
)

// zoneOf is the zone a session ended in, from the words of its log: where it
// connected, then where it was moved.
func zoneOf(lines []*pb.SessionLogLine) string {
	zone := ""
	for _, l := range lines {
		switch l.GetEvent() {
		case "connect", "resume":
			if m := zoneConnect.FindStringSubmatch(l.GetText()); m != nil {
				zone = m[1]
			}
		case "handoff":
			if m := zoneMoved.FindStringSubmatch(l.GetText()); m != nil {
				zone = m[1]
			}
		}
	}
	return zone
}

// fetchLog asks the server for one session's log.
func (a *App) fetchLog(id string, export bool) tea.Cmd {
	a.inflight++
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 4*commandTimeout)
		defer cancel()
		rep, err := a.sess.SessionLog(ctx, id)
		if err != nil {
			return logLoaded{id: id, err: err, export: export}
		}
		var lines []sessionlog.ServerLine
		for _, l := range rep.GetLines() {
			lines = append(lines, sessionlog.ServerLine{TMs: l.GetTMs(), Text: l.GetText()})
		}
		return logLoaded{id: id, lines: lines, total: int(rep.GetTotalLines()), export: export}
	}
}

// sessionsKey handles a key while the sessions view is open.
func (a *App) sessionsKey(k string) Action {
	ui := a.ses
	if ui.Log != nil {
		switch k {
		case "esc", "q":
			ui.Log, ui.Scroll = nil, 0
		case "ctrl+c":
			return Action{Kind: Quit}
		case "up", "k":
			ui.Scroll = min(ui.Scroll+1, len(ui.Log))
		case "down", "j":
			ui.Scroll = max(ui.Scroll-1, 0)
		case "pgup":
			ui.Scroll = min(ui.Scroll+10, len(ui.Log))
		case "pgdown":
			ui.Scroll = max(ui.Scroll-10, 0)
		case "e":
			if r, ok := ui.Selected(); ok {
				return Action{Kind: SessionsDo, ID: "export", Note: r.ID}
			}
		}
		return Action{}
	}
	switch k {
	case "esc", "q":
		a.ses = nil
	case "ctrl+c":
		return Action{Kind: Quit}
	case "up", "k":
		ui.Sel = max(ui.Sel-1, 0)
	case "down", "j":
		ui.Sel = min(ui.Sel+1, max(len(ui.Rows)-1, 0))
	case "r":
		return Action{Kind: SessionsDo, ID: "refresh"}
	case "enter", "space", " ":
		if r, ok := ui.Selected(); ok {
			if r.Local {
				return Action{Kind: SessionsDo, ID: "replay", Note: r.ID}
			}
			return Action{Kind: SessionsDo, ID: "log", Note: r.ID}
		}
	case "e":
		if r, ok := ui.Selected(); ok {
			return Action{Kind: SessionsDo, ID: "export", Note: r.ID}
		}
	}
	return Action{}
}

// sessionsDo carries out a request of the sessions view.
func (a *App) sessionsDo(act Action) tea.Cmd {
	ui := a.ses
	if ui == nil {
		return nil
	}
	r, _ := ui.Selected()
	switch act.ID {
	case "refresh":
		return a.loadSessions()
	case "replay":
		a.startReplay(act.Note)
	case "log":
		return a.fetchLog(act.Note, false)
	case "export":
		switch {
		case r.Local:
			for _, e := range sessionlog.List(a.logRoot) {
				if e.Meta.Session == act.Note {
					path, err := sessionlog.ExportLocal(a.logRoot, e)
					a.exported(path, err)
					break
				}
			}
		case ui.Log != nil && ui.LogID == act.Note:
			a.exportLog(act.Note, ui.Lines)
		default:
			return a.fetchLog(act.Note, true)
		}
	}
	return nil
}

func (a *App) exportLog(id string, lines []sessionlog.ServerLine) {
	path, err := sessionlog.ExportServer(a.logRoot, id, lines)
	a.exported(path, err)
}

func (a *App) exported(path string, err error) {
	if err != nil {
		a.view.Notice, a.view.NoticeBad = "export failed: "+err.Error(), true
		return
	}
	a.view.Notice, a.view.NoticeBad = "exported to "+path, false
}

// sessionsMsg folds the answers to the sessions view's requests into it.
func (a *App) sessionsMsg(msg tea.Msg) {
	ui := a.ses
	switch msg := msg.(type) {
	case sessionsLoaded:
		a.inflight--
		if ui == nil {
			return
		}
		ui.Loading = false
		ui.Rows = slices.DeleteFunc(ui.Rows, func(r screen.SessionRow) bool { return !r.Local })
		if msg.err != nil {
			ui.Err = msg.err.Error()
			return
		}
		ui.Rows = append(ui.Rows, msg.rows...)
	case logLoaded:
		a.inflight--
		switch {
		case msg.err != nil:
			a.view.Notice, a.view.NoticeBad = "the server's log: "+msg.err.Error(), true
		case msg.export:
			a.exportLog(msg.id, msg.lines)
		case ui != nil:
			ui.LogID, ui.Lines, ui.Scroll = msg.id, msg.lines, 0
			ui.LogTitle = fmt.Sprintf("%s: the server's log, %d lines (UTC)", msg.id, msg.total)
			ui.Log = make([]string, len(msg.lines))
			for i, l := range msg.lines {
				ui.Log[i] = l.String()
			}
		}
	}
}

// startReplay draws a local recording in place, at its end.
func (a *App) startReplay(session string) {
	for _, e := range sessionlog.List(a.logRoot) {
		if e.Meta.Session != session {
			continue
		}
		f, err := os.Open(e.RecordPath())
		if err != nil {
			a.view.Notice, a.view.NoticeBad = "replay: "+err.Error(), true
			return
		}
		defer f.Close()
		msgs, err := record.Read(f)
		if err != nil {
			a.view.Notice, a.view.NoticeBad = "replay: "+err.Error(), true
			return
		}
		a.replay = &replay{id: session, msgs: msgs}
		a.replay.seek(len(msgs))
		return
	}
	a.view.Notice, a.view.NoticeBad = "replay: that recording is gone", true
}

// seek draws the recording up to its n-th message.
func (r *replay) seek(n int) {
	n = max(0, min(n, len(r.msgs)))
	if r.m == nil || n < r.pos {
		r.m, r.pos = model.New(), 0
	}
	for ; r.pos < n; r.pos++ {
		r.m.Apply(r.msgs[r.pos])
	}
}

// replayKey handles a key while a recording is drawn.
func (a *App) replayKey(k string) Action {
	r := a.replay
	step := max(len(r.msgs)/10, 1)
	switch k {
	case "esc", "q":
		a.replay = nil
	case "ctrl+c":
		return Action{Kind: Quit}
	case "right", "d":
		r.seek(r.pos + step)
	case "left", "a":
		r.seek(r.pos - step)
	case "end":
		r.seek(len(r.msgs))
	case "home":
		r.seek(0)
	case "v":
		return Action{Kind: ToggleVerbose}
	}
	return Action{}
}

// replayView is the screen of the recording being drawn.
func (a *App) replayView(w, h int, st screen.Styler) string {
	r := a.replay
	v := screen.View{Conn: screen.ConnReplay, Verbose: a.view.Verbose, Now: a.now(), HasTick: false,
		Notice: fmt.Sprintf("%s: message %d of %d  Left, Right step  Home, End  Esc back", r.id, r.pos, len(r.msgs))}
	return screen.Render(r.m, v, w, h, st)
}
