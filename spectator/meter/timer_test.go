package meter

import (
	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"testing"
	"time"
)

func TestTimer_Record(t *testing.T) {
	id := NewId("recordTimer", nil)
	w := writer.MemoryWriter{}
	timer := NewTimer(id, &w)
	timer.Record(1000 * time.Millisecond)
	timer.Record(2000 * time.Millisecond)
	timer.Record(3000 * time.Millisecond)
	timer.Record(3001 * time.Millisecond)

	expectedLines := []string{"t:recordTimer:1.000000", "t:recordTimer:2.000000", "t:recordTimer:3.000000", "t:recordTimer:3.001000"}
	for i, line := range w.Lines {
		if line != expectedLines[i] {
			t.Errorf("Expected line to be %s, got %s", expectedLines[i], line)
		}
	}
}

func TestTimer_RecordZero(t *testing.T) {
	id := NewId("recordTimerZero", nil)
	w := writer.MemoryWriter{}
	timer := NewTimer(id, &w)
	timer.Record(0)

	expected := "t:recordTimerZero:0.000000"
	if w.Lines[0] != expected {
		t.Errorf("Expected line to be %s, got %s", expected, w.Lines[0])
	}
}

func TestTimer_RecordNegative(t *testing.T) {
	id := NewId("recordTimerNegative", nil)
	w := writer.MemoryWriter{}
	timer := NewTimer(id, &w)
	timer.Record(-100 * time.Millisecond)

	if len(w.Lines) != 0 {
		t.Error("Negative durations should be ignored")
	}
}

func TestTimer_RecordMultipleValues(t *testing.T) {
	id := NewId("recordTimerMultiple", nil)
	w := writer.MemoryWriter{}
	timer := NewTimer(id, &w)
	timer.Record(100 * time.Millisecond)
	timer.Record(200 * time.Millisecond)
	timer.Record(300 * time.Millisecond)

	expectedLines := []string{
		"t:recordTimerMultiple:0.100000",
		"t:recordTimerMultiple:0.200000",
		"t:recordTimerMultiple:0.300000",
	}
	for i, line := range w.Lines {
		if line != expectedLines[i] {
			t.Errorf("Expected line to be %s, got %s", expectedLines[i], line)
		}
	}
}
