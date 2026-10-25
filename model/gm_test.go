package model

import (
	"strings"
	"testing"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func announcement(text string) *pb.ServerMessage {
	return &pb.ServerMessage{Body: &pb.ServerMessage_Announcement{Announcement: &pb.Announcement{Text: text}}}
}

func serverError(code pb.ErrorCode, text string) *pb.ServerMessage {
	return &pb.ServerMessage{Body: &pb.ServerMessage_Error{Error: &pb.Error{Code: code, Message: text}}}
}

func TestAnAnnouncementIsABannerAndAStyledLogLine(t *testing.T) {
	m := New()
	m.Apply(announcement("  Restart in 5 minutes.  "))
	if m.Banner == nil || m.Banner.Text != "Restart in 5 minutes." || m.Banner.N != 1 {
		t.Fatalf("banner: %+v", m.Banner)
	}
	if got := m.Log[len(m.Log)-1]; got.Kind != "announcement" || got.Text != "Server: Restart in 5 minutes." {
		t.Fatalf("log: %+v", got)
	}
	m.ClearBanner()
	if m.Banner != nil || len(m.Log) != 1 {
		t.Fatalf("a dismissed banner leaves the log: %+v %d", m.Banner, len(m.Log))
	}
	m.Apply(announcement("Back up."))
	if m.Banner.N != 2 {
		t.Fatalf("a second announcement is a new banner: %+v", m.Banner)
	}
	m.Apply(announcement("   "))
	if m.Banner.N != 2 || len(m.Log) != 2 {
		t.Fatalf("an empty announcement shows nothing: %+v", m.Banner)
	}
}

func TestTheServersWordsForAnEndedConnectionAreKept(t *testing.T) {
	for _, c := range []struct {
		name string
		code pb.ErrorCode
		text string
		want string
	}{
		{"ban", pb.ErrorCode_ERROR_CODE_UNAUTHENTICATED, "This account is banned.", "This account is banned."},
		{"kick", pb.ErrorCode_ERROR_CODE_UNAUTHENTICATED, "A game master removed you.", "A game master removed you."},
		{"revoked code in words", pb.ErrorCode_ERROR_CODE_UNAUTHENTICATED, "credential_revoked", "Your sign-in was revoked, so the server ended your connection."},
		{"other errors are not a cut", pb.ErrorCode_ERROR_CODE_BAD_MESSAGE, "nope", ""},
		{"empty text", pb.ErrorCode_ERROR_CODE_UNAUTHENTICATED, "", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := New()
			m.Apply(serverError(c.code, c.text))
			if m.Cut != c.want {
				t.Errorf("cut %q, want %q", m.Cut, c.want)
			}
			m.ClearCut()
			if m.Cut != "" {
				t.Errorf("not cleared: %q", m.Cut)
			}
		})
	}
}

func TestAMuteNoticeReadsAsTheServersTextAndIsRefused(t *testing.T) {
	const text = "A game master has muted you until 2026-10-25 09:00 UTC. Nobody can hear what you say until then."
	for _, c := range []struct{ name, text, want string }{
		{"the server's text", text, text},
		{"no text", "", "A game master has muted you. Nobody can hear what you say until the mute is lifted."},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := New()
			m.Apply(&pb.ServerMessage{Body: &pb.ServerMessage_Notice{Notice: &pb.SocialNotice{Kind: "muted", Text: c.text}}})
			e := m.Log[len(m.Log)-1]
			if e.Text != c.want || e.Kind != "notice_muted" || !e.Refused() || m.Social.MutedText != c.want {
				t.Errorf("log %+v, muted text %q", e, m.Social.MutedText)
			}
		})
	}
	if w := SocialReason("muted"); !strings.Contains(w, "muted") {
		t.Errorf("reason: %q", w)
	}
}
