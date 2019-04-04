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

type PercentileDistributionSummary struct {
	registry *spectator.Registry
	id       *spectator.Id
	summary  *spectator.DistributionSummary
	counters []*spectator.Counter
}

func NewPercentileDistributionSummary(registry *spectator.Registry, name string, tags map[string]string) *PercentileDistributionSummary {
	return NewPercentileDistributionSummaryWithId(registry, registry.NewId(name, tags))
}

func NewPercentileDistributionSummaryWithId(registry *spectator.Registry, id *spectator.Id) *PercentileDistributionSummary {
	ds := registry.DistributionSummaryWithId(id)
	var counters = make([]*spectator.Counter, PercentileBucketsLength())
	for i := 0; i < PercentileBucketsLength(); i++ {
		counters[i] = counterFor(registry, id, i, distTagValues)
	}
	return &PercentileDistributionSummary{registry: registry, id: id, summary: ds, counters: counters}
}

func (t *PercentileDistributionSummary) Record(amount int64) {
	t.summary.Record(amount)
	t.counters[PercentileBucketsIndex(amount)].Increment()
}

func (t *PercentileDistributionSummary) Count() int64 {
	return t.summary.Count()
}

func (t *PercentileDistributionSummary) TotalAmount() int64 {
	return t.summary.TotalAmount()
}

func (t *PercentileDistributionSummary) Percentile(p float64) float64 {
	var counts = make([]int64, PercentileBucketsLength())
	for i, c := range t.counters {
		counts[i] = int64(c.Count())
	}
	return PercentileBucketsPercentile(counts, p)
}
