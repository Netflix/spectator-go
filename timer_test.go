package spectator

import (
	"reflect"
	"testing"
)

func getTimer(name string) *Timer {
	id := newId(name, nil)
	return NewTimer(id)
}

func TestTimer_Record(t *testing.T) {
	c := getTimer("inc")
	if c.Count() != 0 {
		t.Error("Count should start at 0, got ", c.Count())
	}

	c.Record(100)
	if c.Count() != 1 {
		t.Error("Count should be 1, got ", c.Count())
	}
	if c.TotalTime() != 100 {
		t.Error("TotalTime should be 100, got ", c.TotalTime())
	}

	c.Record(0)
	if c.Count() != 2 {
		t.Error("Count should be 2, got ", c.Count())
	}
	if c.TotalTime() != 100 {
		t.Error("TotalTime should be 100, got ", c.TotalTime())
	}

	c.Record(100)
	if c.Count() != 3 {
		t.Error("Count should be 3, got ", c.Count())
	}
	if c.TotalTime() != 200 {
		t.Error("TotalTime should be 200, got ", c.TotalTime())
	}

	c.Record(-1)
	if c.Count() != 3 && c.TotalTime() != 200 {
		t.Error("Negative times should be ignored")
	}
}

func assertTimer(t *testing.T, timer *Timer, count int64, total int64, totalSq float64, max int64) {
	ms := timer.Measure()
	if len(ms) != 4 {
		t.Error("Expected 4 measurements from a Timer, got ", len(ms))
	}

	expected := make(map[string]float64)
	expected[timer.id.WithStat("count").mapKey()] = float64(count)
	expected[timer.id.WithStat("totalTime").mapKey()] = float64(total) / 1e9
	expected[timer.id.WithStat("totalOfSquares").mapKey()] = totalSq / 1e18
	expected[timer.id.WithStat("max").mapKey()] = float64(max) / 1e9

	got := make(map[string]float64)
	for _, v := range ms {
		got[v.id.mapKey()] = v.value
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Expected measurements (count=%d, total=%d, totalSq=%.0f, max=%d)", count, total, totalSq, max)
		for _, m := range ms {
			t.Errorf("Got %s %v = %f", m.id.name, m.id.tags, m.value)
		}
	}

	// ensure timer is reset after being measured
	if timer.Count() != 0 || timer.TotalTime() != 0 {
		t.Error("Timer should be reset after being measured")
	}
}

func TestTimer_Measure(t *testing.T) {
	c := getTimer("measure")
	c.Record(100)
	c.Record(200)
	assertTimer(t, c, 2, 300, 100*100+200*200, 200)
}
