// Package record writes and reads a session's incoming messages as a log: one
// protobuf-JSON ServerMessage per line, in arrival order (seq-design doc 03
// layer 5). A log can be replayed into the model with no backend.
package record

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"sync"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
	"google.golang.org/protobuf/encoding/protojson"
)

// Writer appends messages to a log.
type Writer struct {
	mu sync.Mutex
	w  io.Writer
}

// NewWriter returns a Writer on w.
func NewWriter(w io.Writer) *Writer { return &Writer{w: w} }

// Write appends one message as one line.
func (r *Writer) Write(m *pb.ServerMessage) error {
	b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(m)
	if err != nil {
		return err
	}
	// protojson's whitespace is deliberately unstable; the line is compacted so
	// a log is stable to review.
	var buf bytes.Buffer
	if err := compact(&buf, b); err != nil {
		return err
	}
	buf.WriteByte('\n')
	r.mu.Lock()
	defer r.mu.Unlock()
	_, err = r.w.Write(buf.Bytes())
	return err
}

// Read reads a whole log. Blank lines and lines starting with # are skipped.
func Read(r io.Reader) ([]*pb.ServerMessage, error) {
	var out []*pb.ServerMessage
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	for n := 1; sc.Scan(); n++ {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		var m pb.ServerMessage
		if err := protojson.Unmarshal(line, &m); err != nil {
			return nil, fmt.Errorf("line %d: %w", n, err)
		}
		out = append(out, &m)
	}
	return out, sc.Err()
}
