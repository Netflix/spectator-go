package meter

import (
	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"testing"
	"time"
)

func TestPercentileTimer_Record(t *testing.T) {
	id := NewId("recordPercentileTimer", nil)
	w := writer.MemoryWriter{}
	pt := NewPercentileTimer(id, &w)
	pt.Record(1000 * time.Millisecond)
	pt.Record(2000 * time.Millisecond)
	pt.Record(3000 * time.Millisecond)
	pt.Record(3001 * time.Millisecond)

	expectedLines := []string{
		"T:recordPercentileTimer:1.000000",
		"T:recordPercentileTimer:2.000000",
		"T:recordPercentileTimer:3.000000",
		"T:recordPercentileTimer:3.001000",
	}
	for i, line := range w.Lines {
		if line != expectedLines[i] {
			t.Errorf("Expected line to be %s, got %s", expectedLines[i], line)
		}
	}
}

func TestPercentileTimer_RecordZero(t *testing.T) {
	id := NewId("recordPercentileTimerZero", nil)
	w := writer.MemoryWriter{}
	pt := NewPercentileTimer(id, &w)
	pt.Record(0)

	expected := "T:recordPercentileTimerZero:0.000000"
	if w.Lines[0] != expected {
		t.Errorf("Expected line to be %s, got %s", expected, w.Lines[0])
	}
}

func TestPercentileTimer_RecordNegative(t *testing.T) {
	id := NewId("recordPercentileTimerNegative", nil)
	w := writer.MemoryWriter{}
	pt := NewPercentileTimer(id, &w)
	pt.Record(-100 * time.Millisecond)

	if len(w.Lines) != 0 {
		t.Error("Negative durations should be ignored")
	}
}

func TestPercentileTimer_RecordMultipleValues(t *testing.T) {
	id := NewId("recordPercentileTimerMultiple", nil)
	w := writer.MemoryWriter{}
	pt := NewPercentileTimer(id, &w)
	pt.Record(100 * time.Millisecond)
	pt.Record(200 * time.Millisecond)
	pt.Record(300 * time.Millisecond)

	expectedLines := []string{
		"T:recordPercentileTimerMultiple:0.100000",
		"T:recordPercentileTimerMultiple:0.200000",
		"T:recordPercentileTimerMultiple:0.300000",
	}
	for i, line := range w.Lines {
		if line != expectedLines[i] {
			t.Errorf("Expected line to be %s, got %s", expectedLines[i], line)
		}
	}
}
