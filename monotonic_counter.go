package spectator

import (
	"sync"
	"sync/atomic"
)

// MonotonicCounter is used track a monotonically increasing counter.
//
// You can find more about this type by viewing the relevant Java Spectator documentation here:
//
// https://netflix.github.io/spectator/en/latest/intro/gauge/#monotonic-counters
type MonotonicCounter struct {
	value int64
	// Pointers need to be after counters to ensure 64-bit alignment. See
	// note in atomicnum.go
	registry    *Registry
	id          *Id
	counter     *Counter
	counterOnce sync.Once
}

// NewMonotonicCounter generates a new monotonic counter, taking the registry so
// that it can lazy-load the underlying counter once `Set` is called the first
// time. It generates a new meter identifier from the name and tags.
func NewMonotonicCounter(registry *Registry, name string, tags map[string]string) *MonotonicCounter {
	return NewMonotonicCounterWithId(registry, NewId(name, tags))
}

// NewMonotonicCounterWithId generates a new monotonic counter, using the
// provided meter identifier.
func NewMonotonicCounterWithId(registry *Registry, id *Id) *MonotonicCounter {
	return &MonotonicCounter{
		registry: registry,
		id:       id,
	}
}

// Set adds amount to the current counter.
func (c *MonotonicCounter) Set(amount int64) {
	var uninitialized bool
	c.counterOnce.Do(func() {
		c.counter = c.registry.CounterWithId(c.id)
		uninitialized = true
	})

	if !uninitialized {
		prev := atomic.LoadInt64(&c.value)
		delta := amount - prev
		if delta >= 0 {
			c.counter.Add(delta)
		}
	}

	atomic.StoreInt64(&c.value, amount)
}

// Count returns the current counter value.
func (c *MonotonicCounter) Count() int64 {
	return atomic.LoadInt64(&c.value)
}
