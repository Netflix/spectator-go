package histogram

import (
	"fmt"

	"github.com/Netflix/spectator-go"
)

var distTagValues []string

func init() {
	length := PercentileBucketsLength()
	distTagValues = make([]string, length)

	for i := 0; i < length; i++ {
		distTagValues[i] = fmt.Sprintf("D%04X", i)
	}
}

// PercentileDistributionSummary is a distribution summary used to track the
// distribution of events, while also presenting the results as percentiles.
type PercentileDistributionSummary struct {
	registry *spectator.Registry
	id       *spectator.Id
	summary  *spectator.DistributionSummary
	counters []*spectator.Counter
}

// NewPercentileDistributionSummary creates a new *PercentileDistributionSummary using the registry to create the meter identifier.
func NewPercentileDistributionSummary(registry *spectator.Registry, name string, tags map[string]string) *PercentileDistributionSummary {
	return NewPercentileDistributionSummaryWithId(registry, registry.NewId(name, tags))
}

// NewPercentileDistributionSummaryWithId creates a new *PercentileDistributionSummary using the meter identifier.
func NewPercentileDistributionSummaryWithId(registry *spectator.Registry, id *spectator.Id) *PercentileDistributionSummary {
	ds := registry.DistributionSummaryWithId(id)
	counters := make([]*spectator.Counter, PercentileBucketsLength())
	for i := 0; i < PercentileBucketsLength(); i++ {
		counters[i] = counterFor(registry, id, i, distTagValues)
	}
	return &PercentileDistributionSummary{registry: registry, id: id, summary: ds, counters: counters}
}

// Record records a new value to track within the distribution.
func (t *PercentileDistributionSummary) Record(amount int64) {
	t.summary.Record(amount)
	t.counters[PercentileBucketsIndex(amount)].Increment()
}

// Count returns the number of unique events that have been tracked.
func (t *PercentileDistributionSummary) Count() int64 {
	return t.summary.Count()
}

// TotalAmount is the total of all records summed together.
func (t *PercentileDistributionSummary) TotalAmount() int64 {
	return t.summary.TotalAmount()
}

// Percentile returns the distribution for a specific percentile.
func (t *PercentileDistributionSummary) Percentile(p float64) float64 {
	counts := make([]int64, PercentileBucketsLength())
	for i, c := range t.counters {
		counts[i] = int64(c.Count())
	}
	return PercentileBucketsPercentile(counts, p)
}
