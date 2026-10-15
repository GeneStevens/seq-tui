package record_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/GeneStevens/seq-tui/record"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
	"google.golang.org/protobuf/proto"
)

func TestRoundTripOneLineEach(t *testing.T) {
	in := []*pb.ServerMessage{
		{Body: &pb.ServerMessage_Welcome{Welcome: &pb.Welcome{ProtocolVersion: 1, TickMs: 100}}},
		{Body: &pb.ServerMessage_Event{Event: &pb.Event{Seq: 5, Body: &pb.Event_Denied{Denied: &pb.Denied{Reason: "encounter_owned"}}}}},
		{Body: &pb.ServerMessage_IntentResult{IntentResult: &pb.IntentResult{IntentId: "i-1", Reason: "unknown_target"}}},
	}
	var buf bytes.Buffer
	w := record.NewWriter(&buf)
	for _, m := range in {
		if err := w.Write(m); err != nil {
			t.Fatal(err)
		}
	}
	if n := strings.Count(buf.String(), "\n"); n != len(in) {
		t.Fatalf("%d lines for %d messages", n, len(in))
	}
	out, err := record.Read(strings.NewReader("# a comment\n\n" + buf.String()))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(in) {
		t.Fatalf("read %d", len(out))
	}
	for i := range in {
		if !proto.Equal(in[i], out[i]) {
			t.Errorf("message %d: %v != %v", i, in[i], out[i])
		}
	}
}

func TestReadReportsTheLine(t *testing.T) {
	_, err := record.Read(strings.NewReader("{\"pong\":{}}\nnot json\n"))
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("err %v", err)
	}
}
