package spectator

import (
	"reflect"
	"testing"
)

func getDistributionSummary(name string) *DistributionSummary {
	id := NewId(name, nil)
	return NewDistributionSummary(id)
}

func TestDistributionSummary_Record(t *testing.T) {
	c := getDistributionSummary("inc")
	if c.Count() != 0 {
		t.Error("Count should start at 0, got ", c.Count())
	}

	c.Record(100)
	if c.Count() != 1 {
		t.Error("Count should be 1, got ", c.Count())
	}
	if c.TotalAmount() != 100 {
		t.Error("TotalAmount should be 100, got ", c.TotalAmount())
	}

	c.Record(0)
	if c.Count() != 2 {
		t.Error("Count should be 2, got ", c.Count())
	}

	if c.TotalAmount() != 100 {
		t.Error("TotalAmount should be 100, got ", c.TotalAmount())
	}

	c.Record(100)
	if c.Count() != 3 {
		t.Error("Count should be 3, got ", c.Count())
	}
	if c.TotalAmount() != 200 {
		t.Error("TotalAmount should be 200, got ", c.TotalAmount())
	}

	c.Record(-1)
	if c.Count() != 3 && c.TotalAmount() != 200 {
		t.Error("Negative values should be ignored")
	}
}

func assertDistributionSummary(t *testing.T, d *DistributionSummary, count int64,
	total int64, totalSq float64, max int64) {
	ms := d.Measure()
	if len(ms) != 4 {
		t.Error("Expected 4 measurements from a DistributionSummary, got ", len(ms))
	}

	expected := make(map[string]float64)
	expected[d.id.WithStat("count").mapKey()] = float64(count)
	expected[d.id.WithStat("totalAmount").mapKey()] = float64(total)
	expected[d.id.WithStat("totalOfSquares").mapKey()] = totalSq
	expected[d.id.WithStat("max").mapKey()] = float64(max)

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

	// ensure DistributionSummary is reset after being measured
	if d.Count() != 0 || d.TotalAmount() != 0 {
		t.Error("DistributionSummary should be reset after being measured")
	}
}

func TestDistributionSummary_Measure(t *testing.T) {
	c := getDistributionSummary("measure")
	c.Record(100)
	c.Record(200)
	assertDistributionSummary(t, c, 2, 300, 100*100+200*200, 200)
}
