package meter

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/writer"
)

// MonotonicCounter is used to measure the rate at which some event is occurring. This
// type is safe for concurrent use.
//
// The value is a monotonically increasing number. A minimum of two samples must be received
// in order for spectatord to calculate a delta value and report it to the backend.
//
// A variety of networking metrics may be reported monotonically and this metric type provides a
// convenient means of recording these values, at the expense of a slower time-to-first metric.
type MonotonicCounter struct {
	id              *Id
	writer          writer.Writer
	meterTypeSymbol string
}

// NewMonotonicCounter generates a new counter, using the provided meter identifier.
func NewMonotonicCounter(id *Id, writer writer.Writer) *MonotonicCounter {
	return &MonotonicCounter{id, writer, "C"}
}

// MeterId returns the meter identifier.
func (c *MonotonicCounter) MeterId() *Id {
	return c.id
}

// Set sets a value as the current measurement; spectatord calculates the delta.
func (c *MonotonicCounter) Set(value uint64) {
	var line = fmt.Sprintf("%s:%s:%d", c.meterTypeSymbol, c.id.spectatordId, value)
	c.writer.Write(line)
}
