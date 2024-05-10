package meter

import (
	"github.com/Netflix/spectator-go/spectator/writer"
	"testing"
)

func TestPercentileDistributionSummary_Record(t *testing.T) {
	id := NewId("recordPercentile", nil)
	w := writer.MemoryWriter{}
	ds := NewPercentileDistributionSummary(id, &w)
	ds.Record(1000)
	ds.Record(2000)
	ds.Record(3000)
	ds.Record(3001)

	expectedLines := []string{"D:recordPercentile:1000", "D:recordPercentile:2000", "D:recordPercentile:3000", "D:recordPercentile:3001"}
	for i, line := range w.Lines {
		if line != expectedLines[i] {
			t.Errorf("Expected line to be %s, got %s", expectedLines[i], line)
		}
	}
}

func TestPercentileDistributionSummary_RecordZero(t *testing.T) {
	id := NewId("recordPercentileZero", nil)
	w := writer.MemoryWriter{}
	ds := NewPercentileDistributionSummary(id, &w)
	ds.Record(0)

	expected := "D:recordPercentileZero:0"
	if w.Lines[0] != expected {
		t.Errorf("Expected line to be %s, got %s", expected, w.Lines[0])
	}
}

func TestPercentileDistributionSummary_RecordMultipleValues(t *testing.T) {
	id := NewId("recordPercentileMultiple", nil)
	w := writer.MemoryWriter{}
	ds := NewPercentileDistributionSummary(id, &w)
	ds.Record(100)
	ds.Record(200)
	ds.Record(300)

	expectedLines := []string{"D:recordPercentileMultiple:100", "D:recordPercentileMultiple:200", "D:recordPercentileMultiple:300"}
	for i, line := range w.Lines {
		if line != expectedLines[i] {
			t.Errorf("Expected line to be %s, got %s", expectedLines[i], line)
		}
	}
}
