package app

import (
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/model"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
)

var t0 = time.Unix(1_000_000, 0)

// polls runs the walker's frame clock (every 33 ms) from from to to, feeding
// events at their times, and returns what it asked to send.
type driven struct {
	w     *Walker
	sends []WalkSend
}

func (d *driven) poll(now time.Time) {
	if out, ok := d.w.Poll(now); ok {
		d.sends = append(d.sends, WalkSend{At: now, WalkOut: out})
	}
}

// A key repeating every 30 ms for ten seconds, with and without releases
// reported, never asks for more than one movement intent in 100 ms, and ends
// with a single stop.
func TestAThirtyMillisecondRepeatIsCoalescedToOneIntentForATick(t *testing.T) {
	for _, releases := range []bool{false, true} {
		d := &driven{w: NewWalker()}
		d.w.Releases = releases
		end := t0.Add(10 * time.Second)
		next := t0
		for now := t0; now.Before(end); now = now.Add(frameEvery) {
			for !next.After(now) { // the repeats due by this frame
				d.w.Press("right", next)
				next = next.Add(30 * time.Millisecond)
			}
			d.poll(now)
		}
		for now := end; now.Before(end.Add(time.Second)); now = now.Add(frameEvery) {
			if releases && now == end {
				d.w.Release("right")
			}
			d.poll(now)
		}
		for i := 1; i < len(d.sends); i++ {
			if gap := d.sends[i].At.Sub(d.sends[i-1].At); gap < walkGap {
				t.Fatalf("releases=%v: intents %d and %d are %v apart", releases, i-1, i, gap)
			}
		}
		walks, stops := 0, 0
		for _, s := range d.sends {
			if s.Stop {
				stops++
			} else {
				walks++
			}
		}
		if walks > 52 || stops != 1 || !d.sends[len(d.sends)-1].Stop {
			t.Fatalf("releases=%v: %d walks and %d stops for ten seconds of hold", releases, walks, stops)
		}
	}
}

// Without releases the first repeat comes after the terminal's pause, which does
// not stop the walk; the pause is learned, and later holds stop sooner after
// the last repeat.
func TestTheFirstRepeatPauseDoesNotStopTheWalk(t *testing.T) {
	d := &driven{w: NewWalker()}
	d.w.Press("up", t0)
	for now := t0; now.Before(t0.Add(400 * time.Millisecond)); now = now.Add(frameEvery) {
		d.poll(now)
	}
	if len(d.sends) == 0 || d.sends[len(d.sends)-1].Stop {
		t.Fatalf("the walk stopped in the pause: %+v", d.sends)
	}
	d.w.Press("up", t0.Add(400*time.Millisecond)) // the first repeat
	if d.w.first != 460*time.Millisecond {
		t.Fatalf("the pause learned is %v", d.w.first)
	}
	// Quiet after the repeat: stopped inside repeatIdle plus a frame.
	last := t0.Add(430 * time.Millisecond)
	d.w.Press("up", last)
	var stopAt time.Time
	for now := last; now.Before(last.Add(time.Second)); now = now.Add(5 * time.Millisecond) {
		d.poll(now)
		if n := len(d.sends); n > 0 && d.sends[n-1].Stop && stopAt.IsZero() {
			stopAt = now
		}
	}
	if after := stopAt.Sub(last); after < 150*time.Millisecond || after > 200*time.Millisecond {
		t.Fatalf("stopped %v after the last repeat", after)
	}
}

// A key reported released stops at once (after the tick's gap), a tap is a
// walk and a stop a tick apart, and a key whose release was lost is let go
// after keyLost.
func TestReleasesAndLostKeys(t *testing.T) {
	d := &driven{w: NewWalker()}
	d.w.Releases = true
	d.w.Press("a", t0)
	d.poll(t0)
	d.w.Release("a")
	d.poll(t0.Add(10 * time.Millisecond)) // inside the gap: held back
	if len(d.sends) != 1 {
		t.Fatalf("the stop was not coalesced into the tick: %+v", d.sends)
	}
	d.poll(t0.Add(100 * time.Millisecond))
	if len(d.sends) != 2 || !d.sends[1].Stop || d.sends[0].Heading != 180 {
		t.Fatalf("a tap sent %+v", d.sends)
	}
	d.w.Press("w", t0.Add(time.Second))
	d.poll(t0.Add(time.Second))
	d.poll(t0.Add(time.Second + keyLost + frameEvery))
	if n := len(d.sends); !d.sends[n-1].Stop {
		t.Fatalf("a key with no release stayed down: %+v", d.sends)
	}
}

func wallModel() *model.Model {
	m := model.New()
	m.Map = &pb.ZoneMap{
		Bounds: &pb.Rect{MinX: 0, MinY: 0, MaxX: 100000, MaxY: 100000},
		Walls:  []*pb.Wall{{A: &pb.Pos{X: 40000, Y: 30000}, B: &pb.Pos{X: 40000, Y: 70000}}},
	}
	m.Self.Pos = &pb.Pos{X: 38000, Y: 50000}
	m.SelfSpeedMmS = 4000
	m.SelfMoves = 1
	return m
}

// The mark stops at a wall and at the bounds, sliding along a wall by the longer
// axis, as the server does.
func TestPredictionNeverRunsThroughAWall(t *testing.T) {
	m := wallModel()
	if x, y := walkAlong(m, 38000, 50000, 0, 5000); x > 40000 || x < 39000 || y != 50000 {
		t.Fatalf("east into the wall ended at %v,%v", x, y)
	}
	if x, y := walkAlong(m, 38000, 50000, 45, 5000); x > 40000 || y <= 50000 {
		t.Fatalf("north-east at the wall ended at %v,%v: it should slide north", x, y)
	}
	if x, _ := walkAlong(m, 99000, 50000, 0, 5000); x > 100000 {
		t.Fatalf("east ended at %v, past the bounds", x)
	}
}

// Ahead of the server's word by the speed over the time since it arrived, no
// more than a tick and a half; a silent server pulls the mark back; an arriving
// position farther than half a metre from the mark snaps it; a nearer one blends.
func TestPredictionBlendsAndSnaps(t *testing.T) {
	m := wallModel()
	m.Self.Pos = &pb.Pos{X: 10000, Y: 50000}
	var p Predictor
	x, _ := p.Step(m, noHeading, t0)
	if x != 10000 {
		t.Fatalf("at rest the mark is at %d", x)
	}
	// Walking: after 100 ms the mark is a step ahead (4000 mm/s is 400 mm).
	var now time.Time
	for ms := 0; ms <= 100; ms += 10 {
		now = t0.Add(time.Duration(ms) * time.Millisecond)
		x, _ = p.Step(m, 0, now)
	}
	if x < 10200 || x > 10400 {
		t.Fatalf("a tick into the walk the mark is at %d", x)
	}
	// The server's word arrives, 0.3 m from the mark: blended, not snapped.
	m.Self.Pos, m.SelfMoves = &pb.Pos{X: 10100, Y: 50000}, m.SelfMoves+1
	now = now.Add(10 * time.Millisecond)
	if x2, _ := p.Step(m, 0, now); x2 < x-100 {
		t.Fatalf("a near word snapped the mark from %d to %d", x, x2)
	}
	// 2 m away: snapped.
	m.Self.Pos, m.SelfMoves = &pb.Pos{X: 13000, Y: 50000}, m.SelfMoves+1
	now = now.Add(10 * time.Millisecond)
	if x2, _ := p.Step(m, noHeading, now); x2 != 13000 {
		t.Fatalf("a far word left the mark at %d", x2)
	}
	// A silent server: the mark never runs more than a tick and a half ahead,
	// and goes back to the word after 250 ms.
	m.Self.Pos, m.SelfMoves = &pb.Pos{X: 20000, Y: 50000}, m.SelfMoves+1
	now = now.Add(time.Second)
	p.Step(m, 0, now)
	for ms := 10; ms <= 600; ms += 10 {
		x, _ = p.Step(m, 0, now.Add(time.Duration(ms)*time.Millisecond))
		if x > 20000+600+5 {
			t.Fatalf("%d ms after the word the mark is at %d", ms, x)
		}
	}
	if x > 20100 {
		t.Fatalf("a silent server left the mark at %d", x)
	}
}
