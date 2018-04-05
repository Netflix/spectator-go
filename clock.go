package spectator

import "time"

type Clock interface {
	WallTime() int64
	MonotonicTime() time.Duration
}

type SystemClock struct{}

func (c *SystemClock) WallTime() int64 {
	return int64(c.MonotonicTime()) / 1000000
}

func (c *SystemClock) MonotonicTime() time.Duration {
	now := time.Now()
	return time.Duration(now.UnixNano())
}

type ManualClock struct {
	wall      int64
	monotonic time.Duration
}

func (c *ManualClock) WallTime() int64 {
	return c.wall
}

func (c *ManualClock) MonotonicTime() time.Duration {
	return c.monotonic
}
