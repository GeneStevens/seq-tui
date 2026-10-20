package app

import (
	"math"
	"time"

	"github.com/GeneStevens/seq-tui/model"
)

// Hold-to-walk (SEQ-088, TUI-031). A held movement key walks the character along
// a heading and releasing it stops: the Walker turns key events into the few
// intents that say so, and the Predictor moves the character's own mark ahead of
// the server's word so that a key shows at once. Both are pure: the program
// feeds them events and the clock.

const (
	// walkGap is the least time between two movement intents: one for each server
	// tick (100 ms). Repeats are coalesced into it, so a fast key repeat can never
	// reach the server's intent limit (20 a second).
	walkGap = 100 * time.Millisecond
	// walkKeepAlive is how often a held key repeats its heading to keep the walk
	// alive; the server lets one go for 500 ms.
	walkKeepAlive = 200 * time.Millisecond
	// repeatIdle is how long a terminal that reports no key release may be quiet
	// once its key is repeating before the key counts as let go; the repeat comes
	// every 30 to 90 ms.
	repeatIdle = 150 * time.Millisecond
	// firstDelay is the guess at the pause between a key going down and its first
	// repeat, until one has been seen: a key held for the first time must not stop
	// in that pause.
	firstDelay = 450 * time.Millisecond
	// keyLost is how long a key whose release is reported may be silent (no
	// repeat) before it counts as let go anyway: a release that never came, on a
	// terminal that lost the focus say.
	keyLost = 3 * time.Second

	// frameEvery is the redraw rate while the character walks: about 30 a second.
	frameEvery = 33 * time.Millisecond
)

// heading of each direction key in degrees: 0 east, 90 north, counter-clockwise,
// as the Walk intent has it.
var dirOf = map[string][2]int{
	"up": {0, 1}, "w": {0, 1},
	"down": {0, -1}, "s": {0, -1},
	"left": {-1, 0}, "a": {-1, 0},
	"right": {1, 0}, "d": {1, 0},
}

// dirName is the one name of the direction a key stands for, so W and Up held
// together are one key.
var dirName = map[string]string{
	"up": "up", "w": "up", "down": "down", "s": "down", "left": "left", "a": "left", "right": "right", "d": "right",
}

// MoveKey says k is a movement key (arrow or WASD).
func MoveKey(k string) bool { _, ok := dirOf[k]; return ok }

// headings by the signs of the two held directions.
var degrees = map[[2]int]int{
	{1, 0}: 0, {1, 1}: 45, {0, 1}: 90, {-1, 1}: 135, {-1, 0}: 180, {-1, -1}: 225, {0, -1}: 270, {1, -1}: 315,
}

type heldKey struct {
	since, last time.Time
	repeated    bool
	interval    time.Duration
	dir         [2]int
}

// WalkOut is an intent the walker asks to send: Stop, or Walk along Heading.
type WalkOut struct {
	Stop    bool
	Heading int
}

// Walker follows which movement keys are down and says what to tell the server.
type Walker struct {
	// Releases says the terminal reports key releases (the kitty keyboard
	// protocol's event types): a key is down from its press to its release. Without
	// it a key is down while its repeats keep coming.
	Releases bool

	held  map[string]*heldKey
	first time.Duration // the pause before the first repeat, as far as it has been seen

	sent   int // the heading last told, noHeading for a stop, unknown for nothing yet
	sentAt time.Time
}

const (
	noHeading = -1
	unknown   = -2
)

// NewWalker returns a walker with no key down.
func NewWalker() *Walker {
	return &Walker{held: map[string]*heldKey{}, first: firstDelay, sent: noHeading}
}

// Press notes a key going down, or repeating when it is down already.
func (w *Walker) Press(key string, now time.Time) {
	name, ok := dirName[key]
	if !ok {
		return
	}
	if !w.Releases {
		// A terminal repeats only the last key pressed: an earlier one, still down
		// or not, no longer speaks. A turn is the new key alone.
		for n := range w.held {
			if n != name {
				delete(w.held, n)
			}
		}
	}
	h := w.held[name]
	if h == nil {
		w.held[name] = &heldKey{since: now, last: now, dir: dirOf[key]}
		return
	}
	switch {
	case !h.repeated:
		h.repeated = true // the first repeat comes after the terminal's pause, which says nothing of the rate
		if !w.Releases {
			w.first = min(max(now.Sub(h.since)+60*time.Millisecond, 200*time.Millisecond), time.Second)
		}
	case h.interval == 0:
		h.interval = now.Sub(h.last)
	default:
		h.interval = (3*h.interval + now.Sub(h.last)) / 4
	}
	h.last = now
}

// Release notes a key coming up.
func (w *Walker) Release(key string) { delete(w.held, dirName[key]) }

// ReleaseAll lets every key go: the window lost the focus, say.
func (w *Walker) ReleaseAll() { clear(w.held) }

// expire drops the keys that have been quiet too long to be down.
func (w *Walker) expire(now time.Time) {
	for n, h := range w.held {
		limit := keyLost
		if !w.Releases {
			limit = w.first
			if h.repeated {
				limit = min(max(2*h.interval, repeatIdle), 2*repeatIdle)
			}
		}
		if now.Sub(h.last) > limit {
			delete(w.held, n)
		}
	}
}

// Heading is the heading the keys down give, noHeading for none (two opposite
// keys cancel).
func (w *Walker) Heading(now time.Time) int {
	w.expire(now)
	var s [2]int
	for _, h := range w.held {
		s[0] += h.dir[0]
		s[1] += h.dir[1]
	}
	s[0], s[1] = sign(s[0]), sign(s[1])
	if d, ok := degrees[s]; ok {
		return d
	}
	return noHeading
}

func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	}
	return 0
}

// Active says the walker has work: a key is down, or a walk it started is not
// yet stopped.
func (w *Walker) Active() bool { return len(w.held) > 0 || w.sent >= 0 || w.sent == unknown }

// Walking says the character is, as far as the walker has told, walking.
func (w *Walker) Walking() bool { return w.sent >= 0 }

// Poll says what to send at now, if anything: a change of heading or a stop,
// held back to one intent in walkGap; else the same heading again every
// walkKeepAlive so the server's walk does not time out.
func (w *Walker) Poll(now time.Time) (WalkOut, bool) {
	want := w.Heading(now)
	since := now.Sub(w.sentAt)
	switch {
	case want == w.sent:
		if want >= 0 && since >= walkKeepAlive {
			break
		}
		return WalkOut{}, false
	case since < walkGap && !w.sentAt.IsZero():
		return WalkOut{}, false // a tick has not passed: coalesced into the next
	}
	w.sent, w.sentAt = want, now
	if want == noHeading {
		return WalkOut{Stop: true}, true
	}
	return WalkOut{Heading: want}, true
}

// Failed says the intent Poll returned was not sent: the next Poll says it again.
func (w *Walker) Failed() { w.sent, w.sentAt = unknown, time.Time{} }

// predictLead is the farthest the mark runs ahead of the server's last word: a
// tick and a half. A server silent for longer than predictSilent is not moving
// the character (a wall, a snare, a stun), and the mark goes back to its word.
const (
	predictLead   = 150 * time.Millisecond
	predictSilent = 250 * time.Millisecond
	// snapMm is how far the mark may be from where it should be before it jumps
	// there; less than that it blends.
	snapMm = 500
	// blendWalking and blendStopped are how quickly the mark closes on where it
	// should be: the time in which it covers about two thirds of the way.
	blendWalking = 40 * time.Millisecond
	blendStopped = 60 * time.Millisecond
)

// Predictor draws the character ahead of the server while it walks. The server
// moves it every tick and its word is always behind; from the last position it
// told, the mark advances at the speed the server's own steps showed, so a key
// shows at once and the picture runs at the frame rate, not the tick's. When the
// server's word arrives the mark blends toward it, or jumps if it is off by more
// than snapMm. It never runs through what the zone map says blocks. Other
// entities are not predicted.
type Predictor struct {
	// Direct draws the mark where it should be, with no blending: a scenario's
	// clock moves between looks, and what it sees must not depend on how often
	// the program drew in between.
	Direct bool

	seq    uint64
	bx, by float64   // the server's last word
	baseAt time.Time // when it arrived, or when the walk began
	x, y   float64   // the mark
	at     time.Time // when the mark was last moved
	valid  bool
	moving bool
}

// Settled says the mark is at the server's word and nothing moves it.
func (p *Predictor) Settled() bool {
	return !p.valid || (!p.moving && math.Hypot(p.x-p.bx, p.y-p.by) < 5)
}

// Step moves the mark to now and returns where to draw the character. heading
// is the heading it walks along, noHeading when it does not.
func (p *Predictor) Step(m *model.Model, heading int, now time.Time) (x, y int64) {
	sp := m.Self.Pos
	if sp == nil {
		return 0, 0
	}
	if !p.valid || m.SelfMoves != p.seq {
		first := !p.valid
		p.seq, p.bx, p.by, p.baseAt = m.SelfMoves, float64(sp.GetX()), float64(sp.GetY()), now
		if first {
			p.x, p.y, p.at, p.valid = p.bx, p.by, now, true
		}
	}
	if heading != noHeading && !p.moving {
		p.baseAt = now // the walk begins: the lead is measured from here
	}
	p.moving = heading != noHeading
	tx, ty := p.bx, p.by
	if v := m.WalkSpeedMmS(); p.moving && v > 0 {
		age := now.Sub(p.baseAt)
		if age > predictSilent {
			age = 0
		}
		age = min(age, predictLead)
		tx, ty = walkAlong(m, p.bx, p.by, heading, float64(v)*age.Seconds())
	}
	dt := min(now.Sub(p.at), 100*time.Millisecond)
	p.at = now
	if d := math.Hypot(tx-p.x, ty-p.y); d > snapMm || (!p.moving && d < 5) || p.Direct {
		p.x, p.y = tx, ty
	} else {
		tau := blendStopped
		if p.moving {
			tau = blendWalking
		}
		k := 1 - math.Exp(-float64(dt)/float64(tau))
		p.x += (tx - p.x) * k
		p.y += (ty - p.y) * k
	}
	return int64(math.Round(p.x)), int64(math.Round(p.y))
}

// walkAlong is where a walk of dist millimetres along the heading from (x, y)
// ends, stopping at what the zone map says blocks and sliding along it by the
// longer axis first, as the server does.
func walkAlong(m *model.Model, x, y float64, heading int, dist float64) (float64, float64) {
	rad := float64(heading) * math.Pi / 180
	dx, dy := math.Cos(rad), math.Sin(rad)
	const stride = 100.0 // mm checked at a time: a wall is never skipped
	for dist > 0 {
		s := min(stride, dist)
		dist -= s
		nx, ny := x+dx*s, y+dy*s
		if passable(m, x, y, nx, ny) {
			x, y = nx, ny
			continue
		}
		ax, ay := x+dx*s, y // along x only
		bx, by := x, y+dy*s // along y only
		okx, oky := passable(m, x, y, ax, ay), passable(m, x, y, bx, by)
		switch {
		case okx && oky && math.Abs(dx) >= math.Abs(dy), okx && !oky:
			x = ax
		case oky:
			y = by
		default:
			return x, y
		}
	}
	return x, y
}

func passable(m *model.Model, x, y, nx, ny float64) bool {
	ix, iy := int64(math.Round(nx)), int64(math.Round(ny))
	return m.CanStandAt(ix, iy) && !m.StepBlocked(int64(math.Round(x)), int64(math.Round(y)), ix, iy)
}
