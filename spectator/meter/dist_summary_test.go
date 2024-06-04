package meter

import (
	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"testing"
)

func TestDistributionSummary_RecordPositiveValue(t *testing.T) {
	id := NewId("recordPositive", nil)
	w := writer.MemoryWriter{}
	ds := NewDistributionSummary(id, &w)
	ds.Record(100)

	expected := "d:recordPositive:100"
	if w.Lines()[0] != expected {
		t.Errorf("Expected line to be %s, got %s", expected, w.Lines()[0])
	}
}

func TestDistributionSummary_RecordZeroValue(t *testing.T) {
	id := NewId("recordZero", nil)
	w := writer.MemoryWriter{}
	ds := NewDistributionSummary(id, &w)
	ds.Record(0)

	expected := "d:recordZero:0"
	if w.Lines()[0] != expected {
		t.Errorf("Expected line to be %s, got %s", expected, w.Lines()[0])
	}
}

func TestDistributionSummary_RecordNegativeValue(t *testing.T) {
	id := NewId("recordNegative", nil)
	w := writer.MemoryWriter{}
	ds := NewDistributionSummary(id, &w)
	ds.Record(-100)

	if len(w.Lines()) != 0 {
		t.Errorf("Expected no lines, got %d", len(w.Lines()))
	}
}

func TestDistributionSummary_RecordMultipleValues(t *testing.T) {
	id := NewId("recordMultiple", nil)
	w := writer.MemoryWriter{}
	ds := NewDistributionSummary(id, &w)
	ds.Record(100)
	ds.Record(200)
	ds.Record(300)

	expectedLines := []string{"d:recordMultiple:100", "d:recordMultiple:200", "d:recordMultiple:300"}
	for i, line := range w.Lines() {
		if line != expectedLines[i] {
			t.Errorf("Expected line to be %s, got %s", expectedLines[i], line)
		}
	}
}
