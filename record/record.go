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
	mu    sync.Mutex
	w     io.Writer
	flush func() error
}

// NewWriter returns a Writer on w.
func NewWriter(w io.Writer) *Writer { return &Writer{w: w} }

// NewFlushingWriter returns a Writer on w, which buffers; flush writes out what
// it holds, and Flush calls it. Whoever writes through it flushes it, at the
// latest when the log ends.
func NewFlushingWriter(w io.Writer, flush func() error) *Writer {
	return &Writer{w: w, flush: flush}
}

// Flush writes out what the Writer's destination holds; a plain Writer has
// nothing.
func (r *Writer) Flush() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.flush == nil {
		return nil
	}
	return r.flush()
}

// Encode is a message as one log line, newline included. It is what a log holds,
// so a message going to several logs is encoded once and written with WriteLine.
func Encode(m *pb.ServerMessage) ([]byte, error) {
	b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(m)
	if err != nil {
		return nil, err
	}
	// protojson's whitespace is deliberately unstable; the line is compacted so
	// a log is stable to review.
	var buf bytes.Buffer
	if err := compact(&buf, b); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// Write appends one message as one line.
func (r *Writer) Write(m *pb.ServerMessage) error {
	line, err := Encode(m)
	if err != nil {
		return err
	}
	return r.WriteLine(line)
}

// WriteLine appends a line made by Encode.
func (r *Writer) WriteLine(line []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.w.Write(line)
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
