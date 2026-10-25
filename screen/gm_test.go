package screen_test

import (
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/screen"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func heard(m *model.Model, body *pb.ServerMessage) *model.Model {
	m.Apply(body)
	return m
}

func TestServerMessagesShowAtTheTopAndInTheLog(t *testing.T) {
	long := strings.Repeat("The server restarts soon, please find a safe place. ", 8)
	for _, c := range []struct {
		name string
		m    func() *model.Model
		v    screen.View
		want []string
		not  []string
	}{
		{"announcement", func() *model.Model {
			return heard(model.New(), &pb.ServerMessage{Body: &pb.ServerMessage_Announcement{Announcement: &pb.Announcement{Text: "Restart in 5 minutes."}}})
		}, screen.View{Conn: screen.ConnOnline}, []string{"SERVER Restart in 5 minutes.", "# "}, nil},
		{"a long one is cut to three rows, the log keeps it whole", func() *model.Model {
			return heard(model.New(), &pb.ServerMessage{Body: &pb.ServerMessage_Announcement{Announcement: &pb.Announcement{Text: long}}})
		}, screen.View{Conn: screen.ConnOnline}, []string{"SERVER The server restarts", "…"}, nil},
		{"a ban while the client is off the server", model.New, screen.View{Conn: screen.ConnReconnecting, Cut: "This account is banned."},
			[]string{"DISCONNECTED This account is banned.", "Q quits"}, nil},
		{"a cut text is not shown once the connection is back", model.New, screen.View{Conn: screen.ConnOnline, Cut: "This account is banned."},
			nil, []string{"DISCONNECTED"}},
		{"nothing to say, nothing shown", model.New, screen.View{Conn: screen.ConnOnline}, nil, []string{"SERVER", "DISCONNECTED"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			out := screen.Render(c.m(), c.v, 120, 40, screen.Plain())
			for _, w := range c.want {
				if !strings.Contains(out, w) {
					t.Errorf("missing %q:\n%s", w, out)
				}
			}
			for _, w := range c.not {
				if strings.Contains(out, w) {
					t.Errorf("unexpected %q:\n%s", w, out)
				}
			}
		})
	}
}

func TestHelpMentionsServerMessages(t *testing.T) {
	m := model.New()
	for p := range screen.HelpPages() {
		out := screen.Render(m, screen.View{Help: true, HelpPage: p}, 120, 60, screen.Plain())
		if strings.Contains(out, "Messages from the server") && strings.Contains(out, "game master") {
			return
		}
	}
	t.Error("no help page speaks of server announcements and game-master messages")
}
