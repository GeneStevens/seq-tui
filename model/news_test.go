package model

import (
	"strings"
	"testing"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

func TestNoteNewsRows(t *testing.T) {
	entry := &pb.ReleaseLogEntry{Release: "m11-b", DeployedMs: 1795000000000, Notes: []string{"Boats no longer sink.", "Gates\x1b[31m open\nat dawn."}}
	tests := []struct {
		name   string
		offset uint32
		reply  *pb.ReleaseLogReply
		why    string
		lines  []string
		at     int
	}{
		{"latest", 0, &pb.ReleaseLogReply{Total: 2, Entries: []*pb.ReleaseLogEntry{entry}}, "",
			[]string{"What's new, 2026-11-17:", "  - Boats no longer sink.", "  - Gates[31m open at dawn."}, 0},
		{"older", 1, &pb.ReleaseLogReply{Total: 2, Entries: []*pb.ReleaseLogEntry{entry}}, "",
			[]string{"What's new, 2026-11-17:", "  - Boats no longer sink.", "  - Gates[31m open at dawn."}, 1},
		{"none yet", 0, &pb.ReleaseLogReply{}, "Nothing yet.", nil, 0},
		{"past the end", 2, &pb.ReleaseLogReply{Total: 2}, "No older release notes.", nil, 0},
		{"refused", 0, &pb.ReleaseLogReply{Refused: "unavailable"}, "the release notes are not available now", nil, 0},
	}
	for _, tc := range tests {
		m := New()
		if got := m.NoteNews(tc.offset, tc.reply); got != tc.why {
			t.Errorf("%s: why %q, want %q", tc.name, got, tc.why)
		}
		var lines []string
		for _, e := range m.Log {
			lines = append(lines, e.Text)
		}
		if strings.Join(lines, "|") != strings.Join(tc.lines, "|") || m.NewsAt != tc.at {
			t.Errorf("%s: log %q at %d, want %q at %d", tc.name, lines, m.NewsAt, tc.lines, tc.at)
		}
	}
	m := New()
	if m.NewsOffset(false) != 0 || m.NewsOffset(true) != 1 {
		t.Errorf("offsets of a fresh model")
	}
	m.NewsAt = 3
	if m.NewsOffset(true) != 4 || m.NewsOffset(false) != 0 {
		t.Errorf("offsets after the third")
	}
}
