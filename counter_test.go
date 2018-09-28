package spectator

import (
	"reflect"
	"testing"
)

func getCounter(name string) *Counter {
	id := newId(name, nil)
	return NewCounter(id)
}

func TestCounter_Increment(t *testing.T) {
	c := getCounter("inc")
	if c.Count() != 0 {
		t.Error("Count should start at 0, got ", c.Count())
	}

	c.Increment()
	if c.Count() != 1 {
		t.Error("Count should be 1, got ", c.Count())
	}

	c.Increment()
	if c.Count() != 2 {
		t.Error("Count should be 2, got ", c.Count())
	}
}

func TestCounter_AddFloat(t *testing.T) {
	c := getCounter("addFloat")
	if c.Count() != 0 {
		t.Error("Count should start at 0, got ", c.Count())
	}

	c.AddFloat(4.2)
	if c.Count() != 4.2 {
		t.Error("Count should be 4.2, got ", c.Count())
	}

	c.AddFloat(-0.1)
	if c.Count() != 4.2 {
		t.Error("Negative deltas should be ignored, got ", c.Count())
	}

	c.AddFloat(4.2)
	if c.Count() != 8.4 {
		t.Error("Expected 84, got ", c.Count())
	}
}

func TestCounter_Add(t *testing.T) {
	c := getCounter("add")
	if c.Count() != 0 {
		t.Error("Count should start at 0, got ", c.Count())
	}

	c.Add(42)
	if c.Count() != 42 {
		t.Error("Count should be 42, got ", c.Count())
	}

	c.Add(-1)
	if c.Count() != 42 {
		t.Error("Negative deltas should be ignored, got ", c.Count())
	}

	c.Add(42)
	if c.Count() != 84 {
		t.Error("Expected 84, got ", c.Count())
	}
}

func TestCounter_Measure(t *testing.T) {
	c := getCounter("measure")
	c.Increment()
	ms := c.Measure()
	if len(ms) != 1 {
		t.Error("Expected a single measurement from a counter, got ", len(ms))
	}

	counterMeasure := ms[0]
	if counterMeasure.value != 1 {
		t.Error("Expected 1, got ", counterMeasure.value)
	}

	expectedTags := make(map[string]string)
	expectedTags["statistic"] = "count"
	if !reflect.DeepEqual(expectedTags, counterMeasure.id.tags) {
		t.Error("Expecting a statistic=count tag")
	}

	if c.Count() != 0 {
		t.Error("Value should be reset after being measured. Got ", c.Count())
	}
}
