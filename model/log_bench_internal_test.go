package model

import "testing"

// fullLog is a model whose log has reached its bound and stays there.
func fullLog() *Model {
	m := New()
	for i := range 4 * DefaultLogCap {
		m.addLog(LogEntry{Seq: uint64(i), Kind: "combat_hit", Text: "x hit y for 3"})
		m.addLog(LogEntry{Seq: uint64(i), Kind: "entity_moved", Text: "y moved to (1, 2)"})
	}
	return m
}

// BenchmarkLogAppend is one line added to a full log (T1): the steady state
// allocates and copies nothing.
func BenchmarkLogAppend(b *testing.B) {
	m := fullLog()
	e := LogEntry{Kind: "combat_hit", Text: "x hit y for 3"}
	b.ReportAllocs()
	for b.Loop() {
		m.addLog(e)
	}
}
