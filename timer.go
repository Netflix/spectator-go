package spectator

import (
	"sync/atomic"
	"time"
)

// Timer is used to measure how long (in seconds) some event is taking. This
// type is safe for concurrent use.
type Timer struct {
	count          int64
	totalTime      int64
	totalOfSquares uint64
	max            int64
	// Pointers need to be after counters to ensure 64-bit alignment. See
	// note in atomicnum.go
	id             *Id
}

// NewTimer generates a new timer, using the provided meter identifier.
func NewTimer(id *Id) *Timer {
	return &Timer{0, 0, 0, 0, id}
}

// MeterId returns the meter identifier.
func (t *Timer) MeterId() *Id {
	return t.id
}

// Record records the duration this specific event took.
func (t *Timer) Record(amount time.Duration) {
	if amount >= 0 {
		atomic.AddInt64(&t.count, 1)
		atomic.AddInt64(&t.totalTime, int64(amount))
		addFloat64(&t.totalOfSquares, float64(amount)*float64(amount))
		updateMax(&t.max, int64(amount))
	}
}

// Count returns the number of unique times that have been recorded.
func (t *Timer) Count() int64 {
	return atomic.LoadInt64(&t.count)
}

// TotalTime returns the total duration of all recorded events.
func (t *Timer) TotalTime() time.Duration {
	return time.Duration(atomic.LoadInt64(&t.totalTime))
}

// Measure returns the list of measurements known by the counter. This should
// return 4 measurements in the slice:
//
// count
// totalTime
// totalOfSquares
// max
func (t *Timer) Measure() []Measurement {
	cnt := Measurement{t.id.WithStat("count"), float64(atomic.SwapInt64(&t.count, 0))}
	totalNanos := atomic.SwapInt64(&t.totalTime, 0)
	tTime := Measurement{t.id.WithStat("totalTime"), float64(totalNanos) / 1e9}
	totalSqNanos := swapFloat64(&t.totalOfSquares, 0.0)
	tSq := Measurement{t.id.WithStat("totalOfSquares"), totalSqNanos / 1e18}
	maxNanos := atomic.SwapInt64(&t.max, 0)
	mx := Measurement{t.id.WithStat("max"), float64(maxNanos) / 1e9}

	return []Measurement{cnt, tTime, tSq, mx}
}
