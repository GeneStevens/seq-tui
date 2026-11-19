package model

import (
	"strings"
	"time"
	"unicode"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// News (D211): the release notes players are given, one release at a time, newest
// first. /news shows the latest and /news more the next older one; the model keeps
// which one the log last showed, so repeating /news more goes back one more.

// NewsOffset is the entry /news asks for: the latest, or with more the one after
// the one last shown.
func (m *Model) NewsOffset(more bool) uint32 {
	if !more {
		return 0
	}
	return uint32(m.NewsAt) + 1
}

// NoteNews puts one page of the release log (the entry at offset) in the log, as
// plain lines: the date and one line for each bullet. It returns the words when
// there is nothing to show, "" when the entry came.
func (m *Model) NoteNews(offset uint32, r *pb.ReleaseLogReply) string {
	if r.GetRefused() != "" {
		return "the release notes are not available now"
	}
	if len(r.GetEntries()) == 0 {
		if offset == 0 || r.GetTotal() == 0 {
			return "Nothing yet."
		}
		return "No older release notes."
	}
	e := r.GetEntries()[0]
	m.NewsAt = int(offset)
	date := time.UnixMilli(e.GetDeployedMs()).UTC().Format("2006-01-02")
	m.newsLine("What's new, " + date + ":")
	for _, n := range e.GetNotes() {
		m.newsLine("  - " + n)
	}
	return ""
}

func (m *Model) newsLine(text string) {
	m.Log = append(m.Log, LogEntry{Kind: "news", Text: PlainText(text)})
	m.trimLog()
}

// PlainText is text from the server with its control characters taken out (a
// newline or tab becomes a space), so it cannot move the cursor or recolour the
// screen.
func PlainText(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\t' || r == '\r':
			return ' '
		case unicode.IsControl(r):
			return -1
		}
		return r
	}, s)
}
