package meter

import (
	"github.com/Netflix/spectator-go/v2/spectator/writer"
)

// MonotonicCounterUint is used to measure the rate at which some event is occurring. This
// type is safe for concurrent use.
//
// The value is a monotonically increasing number. A minimum of two samples must be received
// in order for spectatord to calculate a delta value and report it to the backend.
//
// This version of the monotonic counter is intended to support use cases where a data source value
// can be sampled as-is, because it is already in base units, such as bytes, and thus, the data type
// is uint64.
//
// A variety of networking metrics may be reported monotonically and this metric type provides a
// convenient means of recording these values, at the expense of a slower time-to-first metric.
type MonotonicCounterUint struct {
	id         *Id
	writer     writer.Writer
	linePrefix string
}

// NewMonotonicCounterUint generates a new counter, using the provided meter identifier.
func NewMonotonicCounterUint(id *Id, writer writer.Writer) *MonotonicCounterUint {
	return &MonotonicCounterUint{id, writer, "U:" + id.spectatordId + ":"}
}

// NewMonotonicCounterUintDirect generates a new uint monotonic counter directly
// from a name and tags, without allocating an *Id or copying the tags map.
// commonTags carries the registry's extraCommonTags.
func NewMonotonicCounterUintDirect(name string, tags, commonTags map[string]string, writer writer.Writer) *MonotonicCounterUint {
	return &MonotonicCounterUint{nil, writer, buildLinePrefix("U", name, tags, commonTags)}
}

// MeterId returns the meter identifier, reconstructing it from the line prefix
// if the counter was created directly from a name and tags.
func (c *MonotonicCounterUint) MeterId() *Id {
	return resolveMeterId(c.id, c.linePrefix)
}

// Set sets a value as the current measurement; spectatord calculates the delta.
func (c *MonotonicCounterUint) Set(value uint64) {
	c.writer.WriteUint(c.linePrefix, value)
}
