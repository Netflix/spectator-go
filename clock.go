package spectator

import (
	"sync/atomic"
	"time"
)

// Clock is an interface to provide functionality to create timing/latency
// metrics.
type Clock interface {
	Now() time.Time
	Nanos() int64
}

// SystemClock satisfies the Clock interface, using the system's time as its
// source.
type SystemClock struct{}

// Now returns time.Now(). Satisfies Clock interface.
func (c *SystemClock) Now() time.Time {
	return time.Now()
}

// Nanos returns time.Now().UnixNano(). Satisfies Clock interface.
func (c *SystemClock) Nanos() int64 {
	now := time.Now()
	return now.UnixNano()
}

// ManualClock satisfies the Clock interface, using provided values as the time
// source. You need to seed the time with either .SetFromDuration() or
// .SetNanos().
type ManualClock struct {
	nanos int64
}

// Now effectively returns time.Unix(0, c.Nanos()). Satisfies Clock interface.
func (c *ManualClock) Now() time.Time {
	return time.Unix(0, c.nanos)
}

// Nanos returns the internal nanoseconds. Satisfies Clock interface.
func (c *ManualClock) Nanos() int64 {
	return atomic.LoadInt64(&c.nanos)
}

// SetFromDuration takes a duration, and sets that to the internal nanosecond
// value. The .Now() probably has no value when this is used to populate the
// data.
func (c *ManualClock) SetFromDuration(duration time.Duration) {
	atomic.StoreInt64(&c.nanos, int64(duration))
}

// SetNanos sets the internal nanoseconds value directly.
func (c *ManualClock) SetNanos(nanos int64) {
	atomic.StoreInt64(&c.nanos, nanos)
}
