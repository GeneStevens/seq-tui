package model

import "testing"

func TestSteadyStateAppendAllocatesNothing(t *testing.T) {
	m := fullLog()
	e := LogEntry{Kind: "combat_hit", Text: "x hit y for 3"}
	if n := testing.AllocsPerRun(2000, func() { m.addLog(e) }); n != 0 {
		t.Errorf("%v allocations per append, want 0", n)
	}
}

// The bound holds however many lines come, per kind, oldest cut first, in order.
func TestLogBoundKeepsTheNewestOfEachKindInOrder(t *testing.T) {
	m := NewBounded(5, 2)
	for i := range 1000 {
		kind := "combat_hit"
		if i%3 == 0 {
			kind = "entity_moved"
		}
		m.addLog(LogEntry{Seq: uint64(i + 1), Kind: kind, Text: "t"})
		var shown, quiet int
		var prev uint64
		for _, e := range m.Log {
			if e.Seq <= prev {
				t.Fatalf("after %d lines the log is out of order: %+v", i+1, m.Log)
			}
			prev = e.Seq
			if e.Hidden() {
				quiet++
			} else {
				shown++
			}
		}
		if shown >= 3*5 || quiet >= 3*5 {
			t.Fatalf("after %d lines: %d shown, %d routine", i+1, shown, quiet)
		}
		if m.Log[len(m.Log)-1].Seq != uint64(i+1) {
			t.Fatalf("the newest line is not last")
		}
	}
	var shown []uint64
	for _, e := range m.Log {
		if !e.Hidden() {
			shown = append(shown, e.Seq)
		}
	}
	if len(shown) < 5 || shown[len(shown)-1] != 999 {
		t.Errorf("shown %v", shown)
	}
}

func TestFaultsAreBounded(t *testing.T) {
	m := New()
	for i := range 100 {
		m.Faults = append(m.Faults, itoa(uint64(i)))
		m.Faults = capTail(m.Faults, faultsCap)
	}
	if len(m.Faults) != faultsCap || m.Faults[faultsCap-1] != "99" || m.Faults[0] != "84" {
		t.Errorf("faults %v", m.Faults)
	}
}

func TestNoticeIsALogLine(t *testing.T) {
	m := New()
	m.Notice("the recording stopped")
	if e := m.Log[len(m.Log)-1]; e.Kind != "notice" || e.Text != "the recording stopped" || e.Hidden() {
		t.Errorf("%+v", e)
	}
}
