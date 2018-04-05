package spectator

import "sync/atomic"

type Counter struct {
	id    *Id
	count int64
}

func NewCounter(id *Id) *Counter {
	return &Counter{id, 0}
}

func (c *Counter) MeterId() *Id {
	return c.id
}

func (c *Counter) Measure() []Measurement {
	cnt := atomic.SwapInt64(&c.count, 0)
	return []Measurement{{c.id.WithStat("count"), float64(cnt)}}
}

func (c *Counter) Increment() {
	atomic.AddInt64(&c.count, 1)
}

func (c *Counter) Add(delta int64) {
	if delta > 0 {
		atomic.AddInt64(&c.count, delta)
	}
}

func (c *Counter) Count() int64 {
	return atomic.LoadInt64(&c.count)
}
