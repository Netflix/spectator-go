package histogram

import (
	"fmt"
	"time"

	"github.com/Netflix/spectator-go"
	"github.com/pkg/errors"
)

var timerTagValues []string

func init() {
	length := PercentileBucketsLength()
	timerTagValues = make([]string, length)

	for i := 0; i < length; i++ {
		timerTagValues[i] = fmt.Sprintf("T%04X", i)
	}
}

// PercentileTimer represents timing events, while capturing the histogram
// (percentiles) of those values.
type PercentileTimer struct {
	registry *spectator.Registry
	id       *spectator.Id
	min      time.Duration
	max      time.Duration
	timer    *spectator.Timer
	counters []*spectator.Counter
}

// default min and max durations we track
const (
	defaultMinTime = 10 * time.Millisecond
	defaultMaxTime = 60 * time.Second
)

// NewPercentileTimer creates a new *PercentileTimer using the registry to create the meter identifier.
func NewPercentileTimer(registry *spectator.Registry, name string, tags map[string]string) *PercentileTimer {
	return NewPercentileTimerWithIdRange(registry, registry.NewId(name, tags), defaultMinTime, defaultMaxTime)
}

// NewPercentileTimerWithId creates a new *PercentileTimer using the meter identifier.
func NewPercentileTimerWithId(registry *spectator.Registry, id *spectator.Id) *PercentileTimer {
	return NewPercentileTimerWithIdRange(registry, id, defaultMinTime, defaultMaxTime)
}

type percTimerBuilder struct {
	registry *spectator.Registry
	id       *spectator.Id
	name     string
	tags     map[string]string
	min      time.Duration
	max      time.Duration
}

func (b *percTimerBuilder) Build() (*PercentileTimer, error) {
	if b.registry == nil {
		return nil, errors.New("Need a registry in order to construct a PercentileTimer")
	}
	if b.id == nil && len(b.name) == 0 {
		return nil, errors.New("Need a name or id in order to construct a PercentileTimer")
	}

	id := b.id
	if id == nil {
		id = b.registry.NewId(b.name, b.tags)
	} else {
		id = id.WithTags(b.tags)
	}

	return NewPercentileTimerWithIdRange(b.registry, id, b.min, b.max), nil
}

// PercentileTimerBuilder returns a builder of the *PercentileTimer, which has
// some default values. You do this by calling the Build method on the returned
// value.
func PercentileTimerBuilder() *percTimerBuilder {
	return &percTimerBuilder{min: defaultMinTime, max: defaultMaxTime}
}

func (b *percTimerBuilder) Using(registry *spectator.Registry) *percTimerBuilder {
	b.registry = registry
	return b
}

func (b *percTimerBuilder) WithId(id *spectator.Id) *percTimerBuilder {
	b.id = id
	return b
}

func (b *percTimerBuilder) WithName(name string) *percTimerBuilder {
	b.name = name
	return b
}

func (b *percTimerBuilder) WithTags(tags map[string]string) *percTimerBuilder {
	b.tags = tags
	return b
}

func (b *percTimerBuilder) WithRange(minDuration time.Duration, maxDuration time.Duration) *percTimerBuilder {
	b.min = minDuration
	b.max = maxDuration
	return b
}

// NewPercentileTimerWithIdRange creates a new *PercentileTimer, while
// specifying the minimum / maximum range.
func NewPercentileTimerWithIdRange(registry *spectator.Registry, id *spectator.Id,
	minDuration time.Duration, maxDuration time.Duration) *PercentileTimer {
	timer := registry.TimerWithId(id)
	counters := make([]*spectator.Counter, PercentileBucketsLength())
	for i := 0; i < PercentileBucketsLength(); i++ {
		counters[i] = counterFor(registry, id, i, timerTagValues)
	}
	return &PercentileTimer{registry: registry, id: id, min: minDuration, max: maxDuration, timer: timer, counters: counters}
}

func restrict(amount time.Duration, min time.Duration, max time.Duration) time.Duration {
	r := amount
	if r > max {
		r = max
	} else if r < min {
		r = min
	}
	return r
}

// Record records the value for a single event.
func (t *PercentileTimer) Record(amount time.Duration) {
	t.timer.Record(amount)
	restricted := restrict(amount, t.min, t.max)
	t.counters[PercentileBucketsIndex(restricted.Nanoseconds())].Increment()
}

// Count returns the count of timed events.
func (t *PercentileTimer) Count() int64 {
	return t.timer.Count()
}

// TotalTime returns the total duration.
func (t *PercentileTimer) TotalTime() time.Duration {
	return t.timer.TotalTime()
}

// Percentile returns the latency for a specific percentile.
func (t *PercentileTimer) Percentile(p float64) float64 {
	counts := make([]int64, PercentileBucketsLength())
	for i, c := range t.counters {
		counts[i] = int64(c.Count())
	}
	return PercentileBucketsPercentile(counts, p) / 1e9
}
