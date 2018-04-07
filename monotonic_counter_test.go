package spectator

import "testing"

func TestNewMonotonicCounter(t *testing.T) {
	r := NewRegistry(makeConfig("http://example.org"))
	c := NewMonotonicCounter(r, "mono", nil)

	if v := c.Count(); v != 0 {
		t.Errorf("Counters should be initialized to 0, got %d", v)
	}

	if c.counter != nil {
		t.Errorf("Underlying counter should be nil until we get a delta")
	}

	c.Set(42)
	if v := c.Count(); v != 42 {
		t.Errorf("Expected 42, got %d", v)
	}

	if c.counter != nil {
		t.Errorf("Underlying counter should be nil until we get a delta")
	}

	// now we have a delta
	c.Set(52)
	if c.counter == nil {
		t.Errorf("Underlying counter should not be nil")
	}

	if v := c.counter.Count(); v != 10 {
		t.Errorf("Delta should be 10, got %d", v)
	}
}
