package spectator

import (
	"sync/atomic"
)

// DistributionSummary is used to track the distribution of events. This is safe
// for concurrent use.
//
// You can find more about this type by viewing the relevant Java Spectator
// documentation here:
//
// https://netflix.github.io/spectator/en/latest/intro/dist-summary/
type DistributionSummary struct {
	id          *Id
	count       int64
	totalAmount int64
	totalSqBits uint64
	max         int64
}

// NewDistributionSummary generates a new distribution summary, using the
// provided meter identifier.
func NewDistributionSummary(id *Id) *DistributionSummary {
	return &DistributionSummary{id, 0, 0, 0, 0}
}

// MeterId returns the meter identifier.
func (d *DistributionSummary) MeterId() *Id {
	return d.id
}

// Record records a new value to track within the distribution.
func (d *DistributionSummary) Record(amount int64) {
	if amount >= 0 {
		atomic.AddInt64(&d.count, 1)
		atomic.AddInt64(&d.totalAmount, amount)
		addFloat64(&d.totalSqBits, float64(amount)*float64(amount))
		updateMax(&d.max, amount)
	}
}

// Count returns the number of unique events that have been tracked.
func (d *DistributionSummary) Count() int64 {
	return atomic.LoadInt64(&d.count)
}

// TotalAmount is the total of all records summed together.
func (d *DistributionSummary) TotalAmount() int64 {
	return atomic.LoadInt64(&d.totalAmount)
}

// Measure returns the list of measurements known by the counter. This should
// return 4 measurements in the slice:
//
// count
// totalAmount
// totalOfSquares
// max
func (d *DistributionSummary) Measure() []Measurement {
	cnt := Measurement{d.id.WithStat("count"), float64(atomic.SwapInt64(&d.count, 0))}
	tTime := Measurement{d.id.WithStat("totalAmount"), float64(atomic.SwapInt64(&d.totalAmount, 0))}
	tSq := Measurement{d.id.WithStat("totalOfSquares"), swapFloat64(&d.totalSqBits, 0.0)}
	mx := Measurement{d.id.WithStat("max"), float64(atomic.SwapInt64(&d.max, 0))}

	return []Measurement{cnt, tTime, tSq, mx}
}
