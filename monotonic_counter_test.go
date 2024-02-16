package spectator

import "testing"

func TestNewMonotonicCounter(t *testing.T) {
	r := NewRegistry(makeConfig("http://example.org"))
	c := NewMonotonicCounter(r, "mono", nil)

	if v := c.Count(); v != 0 {
		t.Errorf("Counters should be initialized to 0, got %d", v)
	}

	c.Set(42)
	if v := c.Count(); v != 42 {
		t.Errorf("Expected 42, got %d", v)
	}

	// now we have a delta
	c.Set(52)
	if c.counter == nil {
		t.Errorf("Underlying counter should not be nil")
	}

	if v := c.counter.Count(); v != 10 {
		t.Errorf("Delta should be 10, got %f", v)
	}
}

func TestMonotonicCounterStartingAt0(t *testing.T) {
	r := NewRegistry(makeConfig("http://example.org"))
	c := NewMonotonicCounter(r, "mono", nil)

	c.Set(0)
	if c.counter == nil {
		t.Fatalf("Underlying counter should not be nil")
	}

	if c.counter.Count() != 0.0 {
		t.Errorf("Count should be 0, got %f", c.counter.Count())
	}

	c.Set(1)
	if c.counter.Count() != 1.0 {
		t.Errorf("Count should be 1, got %f", c.counter.Count())
	}
}

func TestMonotonicCounterThreadSafety(t *testing.T) {
	r := NewRegistry(makeConfig("http://example.org"))
	c := NewMonotonicCounter(r, "mono", nil)

	go func() {
		c.Set(0)
	}()
	go func() {
		c.Set(0)
	}()
}
