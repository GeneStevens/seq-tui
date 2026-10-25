package model

import (
	"strings"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// ServerPrefix leads the log line of a server announcement (D100).
const ServerPrefix = "Server: "

// Banner is a server announcement the screen shows at the top until it is
// dismissed or runs out. N counts the announcements seen, so a screen can tell a
// new one from the one it is already showing.
type Banner struct {
	Text string
	N    int
}

// announce takes a server-wide announcement: a distinct line in the log, which
// keeps it, and the banner.
func (m *Model) announce(a *pb.Announcement) {
	text := strings.TrimSpace(a.GetText())
	if text == "" {
		return
	}
	m.announcements++
	m.Banner = &Banner{Text: text, N: m.announcements}
	m.Log = append(m.Log, LogEntry{Seq: m.last, GameTimeMs: m.GameTimeMs, Kind: "announcement", Text: ServerPrefix + text})
	m.trimLog()
}

// ClearBanner drops the announcement banner; the log keeps the line.
func (m *Model) ClearBanner() { m.Banner = nil }

// cutWords are the codes the server gives when it ends a login or a connection
// without a text of its own, in words.
var cutWords = map[string]string{
	"credential_revoked": "Your sign-in was revoked, so the server ended your connection.",
	"invite_revoked":     "Your invitation was revoked, so the server ended your connection.",
}

// noteCut remembers, for an error that ends or refuses a connection, what the
// server said, in plain words: a suspension, a ban or a game master's kick (D100)
// arrives as ERROR_CODE_UNAUTHENTICATED, with the text as the message.
func (m *Model) noteCut(e *pb.Error) {
	if e.GetCode() != pb.ErrorCode_ERROR_CODE_UNAUTHENTICATED || e.GetMessage() == "" {
		return
	}
	m.Cut = firstText(cutWords[e.GetMessage()], e.GetMessage())
}

// ClearCut forgets the text of a refused or ended connection: the connection is
// back.
func (m *Model) ClearCut() { m.Cut = "" }
