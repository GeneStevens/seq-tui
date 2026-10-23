package client

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/GeneStevens/seq-tui/record"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The recorder's bounds: how many messages may wait for the disk, how often
// what has been written is flushed, and how long Close waits for the rest.
const (
	recorderQueue = 4096
	recorderFlush = 200 * time.Millisecond
	recorderWait  = 5 * time.Second
)

// recItem is one thing to record: a message, to be encoded by the writer, or
// its line when the message cannot wait to be (a tick frame, which the toolkit
// goes on to write the numbers of its events into once the hook has seen it).
type recItem struct {
	msg  *pb.ServerMessage
	line []byte
}

// recorder writes the session's messages to its recordings on a goroutine of its
// own, so that a slow disk never holds the session lock (and with it Apply and
// the screen). Each message is encoded once for every recording. When the queue
// is full the message is dropped, never waited for, and the drop is counted: the
// next line written says "# dropped N" (a comment, which a replay skips).
//
// submit and stop are called with the session locked; the rest is the goroutine's.
type recorder struct {
	ch      chan recItem
	done    chan struct{}
	sinks   []*record.Writer
	dropped atomic.Int64
	stopped bool
}

func (r *recorder) start(sinks ...*record.Writer) {
	for _, w := range sinks {
		if w != nil {
			r.sinks = append(r.sinks, w)
		}
	}
	r.ch, r.done = make(chan recItem, recorderQueue), make(chan struct{})
	go r.run()
}

// submit queues msg; with no recording, or once stopped, it does nothing.
func (r *recorder) submit(msg *pb.ServerMessage) { r.queue(recItem{msg: msg}) }

// submitLine queues a message already encoded (record.Encode).
func (r *recorder) submitLine(line []byte) { r.queue(recItem{line: line}) }

// recording says there is somewhere to write.
func (r *recorder) recording() bool { return r.ch != nil }

func (r *recorder) queue(it recItem) {
	if r.ch == nil || r.stopped {
		return
	}
	select {
	case r.ch <- it:
	default:
		r.dropped.Add(1)
	}
}

// stop ends the intake; the goroutine writes what is queued and exits.
func (r *recorder) stop() {
	if r.ch != nil && !r.stopped {
		r.stopped = true
		close(r.ch)
	}
}

// wait returns when everything queued is written and flushed, or after d.
func (r *recorder) wait(d time.Duration) {
	if r.done == nil {
		return
	}
	select {
	case <-r.done:
	case <-time.After(d):
	}
}

func (r *recorder) run() {
	defer close(r.done)
	tick := time.NewTicker(recorderFlush)
	defer tick.Stop()
	dirty := false
	for {
		select {
		case it, ok := <-r.ch:
			if !ok {
				r.flush()
				return
			}
			r.write(it)
			dirty = true
		case <-tick.C:
			if dirty {
				r.flush()
				dirty = false
			}
		}
	}
}

func (r *recorder) write(it recItem) {
	if n := r.dropped.Swap(0); n > 0 {
		r.line([]byte(fmt.Sprintf("# dropped %d messages: the recording could not keep up\n", n)))
	}
	line := it.line
	if line == nil {
		var err error
		if line, err = record.Encode(it.msg); err != nil {
			return
		}
	}
	r.line(line)
}

func (r *recorder) line(b []byte) {
	for _, w := range r.sinks {
		_ = w.WriteLine(b)
	}
}

func (r *recorder) flush() {
	for _, w := range r.sinks {
		_ = w.Flush()
	}
}
