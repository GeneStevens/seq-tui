package scenario_test

import (
	"testing"
	"time"

	"github.com/GeneStevens/seq-tui/app"
	"github.com/GeneStevens/seq-tui/model"
	"github.com/GeneStevens/seq-tui/scenario"
	pb "github.com/GeneStevens/seq/protocol/seqv1"
	"github.com/GeneStevens/seq/testserver"
)

// TUI-031: hold-to-walk. A held movement key sends Walk and keeps it alive,
// releasing it sends Stop, a throttle keeps the intents under the rate limit,
// and the character's own mark runs ahead of the server's word. The zone is
// Testgrounds with no spawns: the character walks east along y=50 m from 20 m
// at 4 m/s (400 mm a tick) towards the wall at x=40 m (from y=30 to y=70 m).

func walker(t *testing.T) (*scenario.World, *scenario.Player) {
	t.Helper()
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria"), Zone: noSpawns})
	w.Place("ann", "aria", 20000, 50000)
	return w, w.Login("ann", "aria")
}

const tick = scenario.Tick

// stops and walks count the movement intents the program sent.
func counts(p *scenario.Player) (walks, stops int) {
	for _, s := range p.WalkSends() {
		if s.Stop {
			stops++
		} else {
			walks++
		}
	}
	return
}

// A terminal that reports releases: the key going down sends Walk and the
// character has moved by the end of the tick that holds it; held, the walk is
// kept alive well past the server's 500 ms timeout; the release sends Stop and
// the character stands still within one tick of it.
func TestKittyHoldWalksAndReleaseStops(t *testing.T) {
	w, p := walker(t)
	p.Kitty()
	start := p.Pos()

	p.KeyEvent("right", scenario.KeyDown)
	if got := p.Pos(); got != start {
		t.Fatalf("the character moved before a tick: %v", got)
	}
	w.Advance(tick)
	first := p.Pos()
	if first[0] <= start[0] || first[1] != start[1] {
		t.Fatalf("a tick after the key went down the character is at %v, from %v", first, start)
	}
	t.Logf("key down to first movement: %v of game time (one tick), %d mm", tick, first[0]-start[0])

	prev := first
	for i := range 29 { // three seconds held: far past the walk timeout
		w.Advance(tick)
		if now := p.Pos(); now[0] != prev[0]+400 {
			t.Fatalf("tick %d of the hold: at %v after %v", i+2, now, prev)
		} else {
			prev = now
		}
	}
	walks, stops := counts(p)
	if stops != 0 || walks < 3 || walks > 17 {
		t.Fatalf("holding sent %d walks and %d stops", walks, stops)
	}

	p.KeyEvent("right", scenario.KeyRelease)
	atRelease := p.Pos()
	w.Advance(tick)
	w.Advance(tick)
	stopped := p.Pos()
	if stopped[0]-atRelease[0] > 2*400 {
		t.Fatalf("the character coasted %d mm past the release", stopped[0]-atRelease[0])
	}
	t.Logf("release to stop: %d mm (%d ticks) of coasting", stopped[0]-atRelease[0], (stopped[0]-atRelease[0])/400)
	w.Advance(10 * tick)
	if got := p.Pos(); got != stopped {
		t.Fatalf("the character walked on after the stop: %v then %v", stopped, got)
	}
	if _, stops := counts(p); stops != 1 {
		t.Fatalf("one release sent %d stops", stops)
	}
	if p.Drawn0() != stopped {
		t.Fatalf("the mark %v is not where the server put the character, %v", p.Drawn0(), stopped)
	}
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

// A terminal that reports no releases: a key is down while its repeats come. The
// first repeat comes after the terminal's own delay, and that pause does not stop
// the walk; once the repeats stop the walk is stopped within about 150 ms.
func TestHoldWithoutReleasesStopsWhenTheRepeatStops(t *testing.T) {
	w, p := walker(t)
	start := p.Pos()

	p.Press("right")
	for range 4 { // the pause before the first repeat: 400 ms
		w.Advance(tick)
	}
	if got := p.Pos(); got[0] <= start[0]+1200 {
		t.Fatalf("the walk did not run through the pause before the first repeat: %v from %v", got, start)
	}
	for range 25 { // the repeat: a key every 40 ms for a second
		p.Press("right")
		w.Advance(40 * time.Millisecond)
	}
	if _, stops := counts(p); stops != 0 {
		t.Fatalf("the walk was stopped %d times while the key repeated", stops)
	}
	last := w.Now()
	for range 6 {
		w.Advance(50 * time.Millisecond)
	}
	var stop app.WalkSend
	for _, s := range p.WalkSends() {
		if s.Stop {
			stop = s
		}
	}
	if stop.At.IsZero() {
		t.Fatalf("no stop after the repeat ended: %v", p.WalkSends())
	}
	if after := stop.At.Sub(last); after < 140*time.Millisecond || after > 260*time.Millisecond {
		t.Fatalf("the stop came %v after the last repeat", after)
	}
	t.Logf("last repeat to Stop: %v", stop.At.Sub(last))
	at := p.Pos()
	w.Advance(10 * tick)
	if got := p.Pos(); got != at {
		t.Fatalf("the character walked on after the stop: %v then %v", at, got)
	}
	// A new hold learned the pause: the next walk is not stopped inside it.
	p.Press("right")
	w.Advance(500 * time.Millisecond)
}

// Two keys down give a diagonal where the terminal reports it; releasing one
// turns the walk; opposite keys cancel; nothing down stops.
func TestTurningAndDiagonals(t *testing.T) {
	w, p := walker(t)
	p.Kitty()
	p.KeyEvent("right", scenario.KeyDown)
	w.Advance(2 * tick)
	p.KeyEvent("up", scenario.KeyDown) // a diagonal: north-east
	w.Advance(tick)                    // the turn lands
	a := p.Pos()
	w.Advance(3 * tick)
	b := p.Pos()
	if b[0] <= a[0] || b[1] <= a[1] || abs64((b[0]-a[0])-(b[1]-a[1])) > 6 {
		t.Fatalf("holding right and up walked %v to %v, not along the diagonal", a, b)
	}
	p.KeyEvent("right", scenario.KeyRelease) // north
	w.Advance(tick)                          // the turn lands
	a = p.Pos()
	w.Advance(3 * tick)
	b = p.Pos()
	if b[0] != a[0] || b[1] <= a[1] {
		t.Fatalf("holding up walked %v to %v, not north", a, b)
	}
	p.KeyEvent("left", scenario.KeyDown) // north-west
	w.Advance(tick)
	a = p.Pos()
	w.Advance(3 * tick)
	b = p.Pos()
	if b[0] >= a[0] || b[1] <= a[1] {
		t.Fatalf("holding up and left walked %v to %v, not north-west", a, b)
	}
	p.KeyEvent("d", scenario.KeyDown) // d against left cancels: north
	w.Advance(tick)
	a = p.Pos()
	w.Advance(3 * tick)
	b = p.Pos()
	if b[0] != a[0] || b[1] <= a[1] {
		t.Fatalf("left against d walked %v to %v, not north", a, b)
	}
	for _, k := range []string{"left", "d", "up"} {
		p.KeyEvent(k, scenario.KeyRelease)
	}
	w.Advance(2 * tick)
	a = p.Pos()
	w.Advance(5 * tick)
	if b = p.Pos(); b != a {
		t.Fatalf("the walk went on after every key came up: %v to %v", a, b)
	}
	var heads []int
	for _, s := range p.WalkSends() {
		switch {
		case s.Stop:
			heads = append(heads, -1)
		case len(heads) == 0 || heads[len(heads)-1] != s.Heading:
			heads = append(heads, s.Heading)
		}
	}
	want := []int{0, 45, 90, 135, 90, -1}
	if len(heads) != len(want) {
		t.Fatalf("headings sent %v, want %v", heads, want)
	}
	for i := range want {
		if heads[i] != want[i] {
			t.Fatalf("headings sent %v, want %v", heads, want)
		}
	}
}

// A key repeating every 30 ms for ten seconds, on a terminal that reports no
// releases (the worst case: a press for every repeat), sends at most one
// movement intent for each server tick, and the server never limits the
// session.
func TestAThirtyMillisecondRepeatForTenSecondsIsNeverRateLimited(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria"), Zone: noSpawns})
	w.Place("ann", "aria", 10000, 80000)
	p := w.Login("ann", "aria")
	start := p.Pos()
	for range 100 { // three or four repeats in every tick
		p.Repeat("right", 3)
		w.Advance(tick)
	}
	sends := p.WalkSends()
	if len(sends) == 0 || len(sends) > 105 {
		t.Fatalf("ten seconds of repeat sent %d movement intents", len(sends))
	}
	for i := 1; i < len(sends); i++ {
		if gap := sends[i].At.Sub(sends[i-1].At); gap < 100*time.Millisecond {
			t.Fatalf("intents %d and %d were %v apart: more than one for a tick", i-1, i, gap)
		}
	}
	t.Logf("%d movement intents in 10 s: %.1f a second", len(sends), float64(len(sends))/10)
	if closed, why := p.Sess.Closed(); closed {
		t.Fatalf("the server ended the session: %v", why)
	}
	p.Model(func(m *model.Model) {
		for _, in := range m.Intents {
			if in.State == model.IntentDenied {
				t.Errorf("an intent was denied: %+v", in)
			}
		}
		for _, e := range m.Log {
			if e.Reason == "rate_limited" || e.Kind == "rate_limited" {
				t.Errorf("the server limited the session: %+v", e)
			}
		}
		if len(m.Errors) > 0 {
			t.Errorf("protocol errors: %v", m.Errors)
		}
	})
	if got := p.Pos(); got[0] < start[0]+30000 {
		t.Fatalf("ten seconds of hold walked only %d mm", got[0]-start[0])
	}
}

// The mark runs ahead of the server's word while a key is held: between two
// ticks it is a fraction of a step ahead, and it never strays from the server's
// position by more than half a metre; the server's word is the truth at rest.
func TestThePredictedMarkRunsAheadAndReconciles(t *testing.T) {
	w, p := walker(t)
	p.Kitty()
	p.KeyEvent("right", scenario.KeyDown)
	w.Advance(tick)
	w.Advance(tick)
	ahead := false
	for range 40 {
		w.Advance(25 * time.Millisecond)
		srv, drawn := p.Pos(), p.Drawn0()
		if drawn[0] > srv[0] {
			ahead = true
		}
		if d := abs64(drawn[0] - srv[0]); d > 500 || drawn[1] != srv[1] {
			t.Fatalf("the mark %v is %d mm from the server's position %v", drawn, d, srv)
		}
	}
	if !ahead {
		t.Fatal("the mark never ran ahead of the server's position")
	}
	p.KeyEvent("right", scenario.KeyRelease)
	w.Advance(3 * tick)
	if srv, drawn := p.Pos(), p.Drawn0(); srv != drawn {
		t.Fatalf("at rest the mark is %v and the server says %v", drawn, srv)
	}
}

// Prediction never runs through a wall: the mark stops at the edge, the server
// stops the character there, and the two agree.
func TestThePredictedMarkStopsAtAWall(t *testing.T) {
	w := scenario.New(t, testserver.Options{Zones: []string{testserver.Testgrounds}, Accounts: one("ann", "aria"), Zone: noSpawns})
	w.Place("ann", "aria", 37000, 50000) // 3 m from the wall at x=40 m
	p := w.Login("ann", "aria")
	p.Kitty()
	p.KeyEvent("right", scenario.KeyDown)
	for range 60 {
		w.Advance(50 * time.Millisecond)
		if drawn := p.Drawn0(); drawn[0] > 40000 {
			t.Fatalf("the mark %v is through the wall at x=40000", drawn)
		}
	}
	srv, drawn := p.Pos(), p.Drawn0()
	if srv[0] > 40000 || srv[0] < 39000 {
		t.Fatalf("the server put the character at %v against the wall", srv)
	}
	if abs64(drawn[0]-srv[0]) > 500 {
		t.Fatalf("at the wall the mark is %v and the server says %v", drawn, srv)
	}
	p.KeyEvent("right", scenario.KeyRelease)
	w.Advance(3 * tick)
	if srv, drawn := p.Pos(), p.Drawn0(); srv != drawn {
		t.Fatalf("at rest the mark is %v and the server says %v", drawn, srv)
	}
}

// A click-walk is a Move to a destination (D77), and Esc ends it with a Stop.
func TestEscEndsAClickWalk(t *testing.T) {
	w, p := walker(t)
	p.Resize(140, 45)
	from := p.Pos()
	x, y := at(t, p, &pb.Pos{X: from[0] - 10000, Y: from[1]})
	p.Click(x, y)
	w.Advance(5 * tick)
	mid := p.Pos()
	if mid[0] >= from[0] {
		t.Fatalf("the click did not walk the character west: %v to %v", from, mid)
	}
	p.Press("esc")
	notice(t, p, "stop ok", false)
	stopped := p.Pos()
	w.Advance(5 * tick)
	if got := p.Pos(); got != stopped {
		t.Fatalf("the character walked on after Esc: %v then %v", stopped, got)
	}
	if got := p.Pos(); got[0] <= from[0]-10000+500 {
		t.Fatalf("the walk was not cut short: %v", got)
	}
}

// How soon a key shows: from the key going down to the first movement of the
// character's mark in the world (about a frame), to the first scroll of the map by
// a cell, and to the server's first move (a tick); and from the release to the
// mark standing still. The timings are of game time, from the scenario's clock.
func TestTheKeyShowsWithinAFrameAndTheMapScrollsWithinACell(t *testing.T) {
	w, p := walker(t)
	p.Resize(140, 45)
	p.Kitty()
	wall := &pb.Pos{X: 40000, Y: 50000}
	cell := func() int { c, _, _ := p.Where(wall); return c }
	start, c0 := p.Drawn0(), cell()

	p.KeyEvent("right", scenario.KeyDown)
	var markAt, cellAt, serverAt time.Duration
	for at := time.Duration(0); at < time.Second && (markAt == 0 || cellAt == 0 || serverAt == 0); {
		w.Advance(33 * time.Millisecond)
		at += 33 * time.Millisecond
		if markAt == 0 && p.Drawn0() != start {
			markAt = at
		}
		if cellAt == 0 && cell() != c0 {
			cellAt = at
		}
		if serverAt == 0 && p.Pos() != start {
			serverAt = at
		}
	}
	if markAt == 0 || markAt > 66*time.Millisecond {
		t.Fatalf("the mark moved after %v", markAt)
	}
	if serverAt == 0 || serverAt > 132*time.Millisecond {
		t.Fatalf("the server moved the character after %v", serverAt)
	}
	if cellAt == 0 || cellAt > 400*time.Millisecond {
		t.Fatalf("the map scrolled a cell after %v", cellAt)
	}
	t.Logf("key down to the mark's first movement: %v; to the server's first move: %v; to the first cell the map scrolls: %v", markAt, serverAt, cellAt)

	for range 10 {
		w.Advance(33 * time.Millisecond)
	}
	p.KeyEvent("right", scenario.KeyRelease)
	var stoppedAt time.Duration
	last := p.Drawn0()
	for at := time.Duration(0); at < time.Second; {
		w.Advance(33 * time.Millisecond)
		at += 33 * time.Millisecond
		if now := p.Drawn0(); now == last && p.Pos() == last {
			stoppedAt = at
			break
		}
		last = p.Drawn0()
	}
	if stoppedAt == 0 || stoppedAt > 400*time.Millisecond {
		t.Fatalf("the mark stood still %v after the release", stoppedAt)
	}
	t.Logf("key up to the mark standing still on the server's position: %v", stoppedAt)
}
