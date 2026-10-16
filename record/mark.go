package record

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	pb "github.com/GeneStevens/seq/protocol/seqv1"
	"google.golang.org/protobuf/encoding/protojson"
)

const markPrefix = "# mark "

// Mark writes a named point into the log, between the messages before it and
// those after. A scenario marks the moments a golden screen is drawn at, so a
// regenerated log does not move them (Read skips the line; ReadUntil stops at it).
func (r *Writer) Mark(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := io.WriteString(r.w, markPrefix+name+"\n")
	return err
}

// ReadUntil reads the messages of a log that come before the mark of that name.
func ReadUntil(r io.Reader, mark string) ([]*pb.ServerMessage, error) {
	var out []*pb.ServerMessage
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	for n := 1; sc.Scan(); n++ {
		line := bytes.TrimSpace(sc.Bytes())
		switch {
		case len(line) == 0:
		case strings.HasPrefix(string(line), markPrefix):
			if strings.TrimSpace(string(line[len(markPrefix):])) == mark {
				return out, nil
			}
		case line[0] == '#':
		default:
			var m pb.ServerMessage
			if err := protojson.Unmarshal(line, &m); err != nil {
				return nil, fmt.Errorf("line %d: %w", n, err)
			}
			out = append(out, &m)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("no mark %q in the log", mark)
}

// Load reads a log from dir by spec: a file name, or "file@mark" for the
// messages before that mark.
func Load(dir, spec string) ([]*pb.ServerMessage, error) {
	name, mark, marked := strings.Cut(spec, "@")
	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if marked {
		return ReadUntil(f, mark)
	}
	return Read(f)
}
