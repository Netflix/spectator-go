package meter

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/writer"
)

// Counter is used to measure the rate at which some event is occurring. This
// type is safe for concurrent use.
//
// You can find more about this type by viewing the relevant Java Spectator
// documentation here:
//
// https://netflix.github.io/spectator/en/latest/intro/counter/
type Counter struct {
	id           *Id
	writer       writer.Writer
	incByOneLine string
}

// NewCounter generates a new counter, using the provided meter identifier.
func NewCounter(id *Id, writer writer.Writer) *Counter {
	return &Counter{id, writer, fmt.Sprintf("c:%s:1", id.spectatordId)}
}

// MeterId returns the meter identifier.
func (c *Counter) MeterId() *Id {
	return c.id
}

// Increment increments the counter.
func (c *Counter) Increment() {
	c.writer.Write(c.incByOneLine)
}

// Add adds an int64 delta to the current measurement.
func (c *Counter) Add(delta int64) {
	if delta == 1 {
		c.writer.Write(c.incByOneLine)
		return
	}
	if delta > 0 {
		var line = fmt.Sprintf("c:%s:%d", c.id.spectatordId, delta)
		c.writer.Write(line)
	}
}

// AddFloat adds a float64 delta to the current measurement.
func (c *Counter) AddFloat(delta float64) {
	if delta > 0.0 {
		var line = fmt.Sprintf("c:%s:%f", c.id.spectatordId, delta)
		c.writer.Write(line)
	}
}
