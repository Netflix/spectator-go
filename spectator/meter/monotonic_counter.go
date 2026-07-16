package meter

import (
	"github.com/Netflix/spectator-go/v2/spectator/writer"
)

// MonotonicCounter is used to measure the rate at which some event is occurring. This
// type is safe for concurrent use.
//
// The value is a monotonically increasing number. A minimum of two samples must be received
// in order for spectatord to calculate a delta value and report it to the backend.
//
// This version of the monotonic counter is intended to support use cases where a data source value
// needs to be transformed into base units through division (e.g. nanoseconds into seconds), and
// thus, the data type is float64.
//
// A variety of networking metrics may be reported monotonically and this metric type provides a
// convenient means of recording these values, at the expense of a slower time-to-first metric.
type MonotonicCounter struct {
	id         *Id
	writer     writer.Writer
	linePrefix string
}

// NewMonotonicCounter generates a new counter, using the provided meter identifier.
func NewMonotonicCounter(id *Id, writer writer.Writer) *MonotonicCounter {
	return &MonotonicCounter{id, writer, "C:" + id.spectatordId + ":"}
}

// NewMonotonicCounterDirect generates a new monotonic counter directly from a
// name and tags, without allocating an *Id or copying the tags map. commonTags
// carries the registry's extraCommonTags.
func NewMonotonicCounterDirect(name string, tags, commonTags map[string]string, writer writer.Writer) *MonotonicCounter {
	return &MonotonicCounter{nil, writer, buildLinePrefix("C", name, tags, commonTags)}
}

// MeterId returns the meter identifier, reconstructing it from the line prefix
// if the counter was created directly from a name and tags.
func (c *MonotonicCounter) MeterId() *Id {
	return resolveMeterId(c.id, c.linePrefix)
}

// Set sets a value as the current measurement; spectatord calculates the delta.
func (c *MonotonicCounter) Set(value float64) {
	c.writer.WriteFloat(c.linePrefix, value)
}
