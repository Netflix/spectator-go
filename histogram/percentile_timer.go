package histogram

import (
	"fmt"
	"github.com/Netflix/spectator-go"
	"github.com/pkg/errors"
	"time"
)

var timerTagValues []string

func init() {
	length := PercentileBucketsLength()
	timerTagValues = make([]string, length)

	for i := 0; i < length; i++ {
		timerTagValues[i] = fmt.Sprintf("T%04X", i)
	}
}

type PercentileTimer struct {
	registry *spectator.Registry
	id       *spectator.Id
	min      time.Duration
	max      time.Duration
	timer    *spectator.Timer
	counters []*spectator.Counter
}

// default min and max durations we track
const defaultMinTime = 10 * time.Millisecond
const defaultMaxTime = 60 * time.Second

func NewPercentileTimer(registry *spectator.Registry, name string, tags map[string]string) *PercentileTimer {
	return NewPercentileTimerWithIdRange(registry, registry.NewId(name, tags), defaultMinTime, defaultMaxTime)
}

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

func NewPercentileTimerWithIdRange(registry *spectator.Registry, id *spectator.Id,
	minDuration time.Duration, maxDuration time.Duration) *PercentileTimer {
	timer := registry.TimerWithId(id)
	var counters = make([]*spectator.Counter, PercentileBucketsLength())
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

func (t *PercentileTimer) Record(amount time.Duration) {
	t.timer.Record(amount)
	restricted := restrict(amount, t.min, t.max)
	t.counters[PercentileBucketsIndex(restricted.Nanoseconds())].Increment()
}

func (t *PercentileTimer) Count() int64 {
	return t.timer.Count()
}

func (t *PercentileTimer) TotalTime() time.Duration {
	return t.timer.TotalTime()
}

func (t *PercentileTimer) Percentile(p float64) float64 {
	var counts = make([]int64, PercentileBucketsLength())
	for i, c := range t.counters {
		counts[i] = int64(c.Count())
	}
	return PercentileBucketsPercentile(counts, p) / 1e9
}
