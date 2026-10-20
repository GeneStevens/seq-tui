package app

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/GeneStevens/seq-tui/client"
	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

// The program's side of hold-to-walk: keys into the Walker, the Walker's intents
// onto the wire in order, and the Predictor's mark onto the screen.

type frameMsg struct{}

// walkReq is a movement intent waiting to be written.
type walkReq struct {
	out  WalkOut
	for_ string // the character it is for when it is not the player's own (D56)
}

// walkState is what App keeps for walking.
type walkState struct {
	w       *Walker
	pred    Predictor
	framing bool // a frame tick is on its way
	queue   chan walkReq
	once    sync.Once
	queued  atomic.Int32 // requests not yet written
	denied  atomic.Value // string: the last refusal of a walk, for the status line
	sends   []WalkSend   // every movement intent asked for, for a scenario to count
	done    chan struct{}
	close   sync.Once
	still   bool // this draw does not move the mark (a scenario's own program draws)
}

func newWalkState() *walkState {
	return &walkState{w: NewWalker(), queue: make(chan walkReq, 64), done: make(chan struct{})}
}

// walkKey handles a movement key press.
func (a *App) walkKey(k string) tea.Cmd {
	a.walk.w.Press(k, a.now())
	return a.walkPoll()
}

// walkRelease handles a movement key coming up.
func (a *App) walkRelease(k string) tea.Cmd {
	a.walk.w.Release(k)
	return a.walkPoll()
}

// walkPoll sends what the walker asks and keeps the frame clock going while the
// character walks or its mark is still settling.
func (a *App) walkPoll() tea.Cmd {
	ws := a.walk
	if why, _ := ws.denied.Swap("").(string); why != "" {
		a.view.Notice, a.view.NoticeBad = why, true
	}
	if out, ok := ws.w.Poll(a.now()); ok {
		req := walkReq{out: out}
		if a.Clock != nil { // a scenario counts them
			ws.sends = append(ws.sends, WalkSend{At: a.now(), WalkOut: out})
		}
		if a.focus {
			req.for_ = a.duo.Character()
			a.duo.SetDirected(true)
		}
		ws.start(a.sess)
		ws.queued.Add(1)
		select {
		case ws.queue <- req:
		default:
			ws.queued.Add(-1)
			ws.w.Failed()
		}
	}
	if ws.framing || !(ws.w.Active() || !ws.pred.Settled()) {
		return nil
	}
	ws.framing = true
	return tea.Tick(frameEvery, func(_ time.Time) tea.Msg { return frameMsg{} })
}

// start runs the goroutine that writes the walk's intents, in the order they were
// asked for: a Stop must not overtake the Walk before it.
func (ws *walkState) start(sess *client.Session) {
	ws.once.Do(func() {
		go func() {
			for {
				select {
				case req := <-ws.queue:
					ws.write(sess, req)
					ws.queued.Add(-1)
				case <-ws.done:
					return
				}
			}
		}()
	})
}

func (ws *walkState) write(sess *client.Session, req walkReq) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	if req.for_ != "" {
		ctx = client.Directing(ctx, req.for_)
	}
	var (
		id  string
		err error
	)
	if req.out.Stop {
		id, err = sess.Stop(ctx)
	} else {
		id, err = sess.Walk(ctx, uint32(req.out.Heading))
	}
	if err != nil {
		ws.denied.Store("walk not sent: " + err.Error())
		return
	}
	// The answer is read later, off the queue: only a refusal is shown.
	go func() {
		rctx, rcancel := context.WithTimeout(context.Background(), commandTimeout)
		defer rcancel()
		if st, err := sess.Result(rctx, id); err == nil && st.State == model.IntentDenied {
			ws.denied.Store("walk denied: " + deniedWords(st))
		}
	}()
}

// stopWalking lets every key go and says so: another screen took the keys, or
// the character the keys were for changed.
func (a *App) stopWalking() tea.Cmd {
	a.walk.w.ReleaseAll()
	return a.walkPoll()
}

// walkFrame is a frame tick.
func (a *App) walkFrame() tea.Cmd {
	a.walk.framing = false
	return a.walkTick()
}

// walkTick is walkPoll for the program's own clocks (a frame, a redraw, a message
// from the server). A scenario's clock is the server's, which moves between one
// look and the next, so it polls at the times it chooses (Frame) and not when a
// timer fires.
func (a *App) walkTick() tea.Cmd {
	if a.Clock != nil {
		return nil
	}
	return a.walkPoll()
}

// closeWalk ends the goroutine that writes the walk.
func (a *App) closeWalk() { a.walk.close.Do(func() { close(a.walk.done) }) }

// drawAhead puts the predicted mark in place of the character's position in the
// model for the length of a draw, and returns the call that puts it back. It does
// so only for the player's own character on foot.
func (a *App) drawAhead(m *model.Model) func() {
	ws, now := a.walk, a.now()
	heading := noHeading
	if !a.focus && !m.Self.Dead && !m.TurnBased() {
		heading = ws.w.Heading(now)
	}
	if a.focus || m.Self.Pos == nil || ws.still {
		return func() {}
	}
	ws.pred.Direct = a.Clock != nil
	x, y := ws.pred.Step(m, heading, now)
	old := m.Self.Pos
	m.Self.Pos = &pb.Pos{X: x, Y: y, Z: old.GetZ()}
	return func() { m.Self.Pos = old }
}

// WalkSend is a movement intent the walker asked for, and when.
type WalkSend struct {
	At time.Time
	WalkOut
}
