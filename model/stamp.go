package model

import (
	"fmt"
	"time"
)

// Clock is the wall clock lines are stamped with when the client receives them
// (D121). A test sets a fixed one.
var Clock = time.Now

// replayWindowMs is how far behind the game time an event may be and still count
// as live: an event older than that was kept for the character (restored or
// replayed at login), and its local time is worked out from the game time.
const replayWindowMs = 3000

// StampWidth is the width of Stamp, in cells.
const StampWidth = 10

// localAt is the local wall-clock time of something said at gameMs of game time,
// when the client is at m.GameTimeMs now. A line with no game time, or one
// within the replay window, is stamped with the time it was received; an older
// one is placed by how long ago the game says it was, and is approximate. The
// zero time says the client has no clock.
func (m *Model) localAt(gameMs int64) (t time.Time, approx bool) {
	if Clock == nil {
		return time.Time{}, false
	}
	t = Clock()
	if behind := m.GameTimeMs - gameMs; gameMs > 0 && behind > replayWindowMs {
		return t.Add(-time.Duration(behind) * time.Millisecond), true
	}
	return t, false
}

// Stamp is the local wall-clock time of a line as the panels print it, 24-hour
// with tenths and no date: "16:04:03.2". A time worked out from the game time
// is marked "~16:04:03 " (not to the tenth), and a line with no time at all
// "--:--:--.-". It is StampWidth cells either way.
func Stamp(t time.Time, approx bool) string {
	switch {
	case t.IsZero():
		return "--:--:--.-"
	case approx:
		return "~" + t.Local().Format("15:04:05") + " "
	}
	t = t.Local()
	return fmt.Sprintf("%s.%d", t.Format("15:04:05"), t.Nanosecond()/100_000_000)
}

// StampGame is a line's local time and its game time together, for exports and
// reports: "16:04:03.2 (game 127445.6s)".
func StampGame(t time.Time, approx bool, gameMs int64) string {
	return fmt.Sprintf("%s (game %.1fs)", Stamp(t, approx), float64(gameMs)/1000)
}

// FixedTime is the instant FixClock holds the clock at, in the local zone.
func FixedTime() time.Time { return time.Date(2026, 11, 1, 16, 4, 3, 200_000_000, time.Local) }

// FixClock holds the clock at FixedTime, for the tests and the golden screens:
// every line is then stamped 16:04:03.2, whatever the day and the zone.
func FixClock() { Clock = FixedTime }

// NowStamp is the local time and the game time now, "16:04:03.2 (game 127445.6s)":
// what a /bug report carries, so a game master can find the moment in either.
func (m *Model) NowStamp() string {
	t, _ := m.localAt(m.GameTimeMs)
	return StampGame(t, false, m.GameTimeMs)
}
