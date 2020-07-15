package spectator

// Counter is used to measure the rate at which some event is occurring. This
// type is safe for concurrent use.
//
// You can find more about this type by viewing the relevant Java Spectator
// documentation here:
//
// https://netflix.github.io/spectator/en/latest/intro/counter/
type Counter struct {
	id    *Id
	count uint64
}

// NewCounter generates a new counter, using the provided meter identifier.
func NewCounter(id *Id) *Counter {
	return &Counter{id, 0}
}

// MeterId returns the meter identifier.
func (c *Counter) MeterId() *Id {
	return c.id
}

// Measure returns the list of measurements known by the counter. This will
// either contain one item, the last value, or no items. This also resets the
// internal counter to 0.0.
func (c *Counter) Measure() []Measurement {
	cnt := swapFloat64(&c.count, 0.0)
	if cnt > 0 {
		return []Measurement{{c.id.WithDefaultStat("count"), cnt}}
	} else {
		return []Measurement{}
	}
}

// Increment increments the counter.
func (c *Counter) Increment() {
	addFloat64(&c.count, 1)
}

// AddFloat adds a specific float64 delta to the current measurement.
func (c *Counter) AddFloat(delta float64) {
	if delta > 0.0 {
		addFloat64(&c.count, delta)
	}
}

// Add is to add a specific int64 delta to the current measurement.
func (c *Counter) Add(delta int64) {
	if delta > 0 {
		addFloat64(&c.count, float64(delta))
	}
}

// Count returns the current value for the counter.
func (c *Counter) Count() float64 {
	return loadFloat64(&c.count)
}
