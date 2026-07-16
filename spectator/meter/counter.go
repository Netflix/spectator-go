package meter

import (
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
	id         *Id
	writer     writer.Writer
	linePrefix string
}

// NewCounter generates a new counter, using the provided meter identifier.
func NewCounter(id *Id, writer writer.Writer) *Counter {
	return &Counter{id, writer, "c:" + id.spectatordId + ":"}
}

// NewCounterDirect generates a new counter directly from a name and tags,
// without allocating an *Id or copying the tags map. commonTags carries the
// registry's extraCommonTags. MeterId reconstructs an Id on demand.
func NewCounterDirect(name string, tags, commonTags map[string]string, writer writer.Writer) *Counter {
	return &Counter{nil, writer, buildLinePrefix("c", name, tags, commonTags)}
}

// MeterId returns the meter identifier, reconstructing it from the line prefix
// if the counter was created directly from a name and tags.
func (c *Counter) MeterId() *Id {
	return resolveMeterId(c.id, c.linePrefix)
}

// Increment increments the counter.
func (c *Counter) Increment() {
	c.writer.WriteLine(c.linePrefix, "1")
}

// Add adds an int64 delta to the current measurement.
func (c *Counter) Add(delta int64) {
	if delta > 0 {
		c.writer.WriteInt(c.linePrefix, delta)
	}
}

// AddFloat adds a float64 delta to the current measurement.
func (c *Counter) AddFloat(delta float64) {
	if delta > 0.0 {
		c.writer.WriteFloat(c.linePrefix, delta)
	}
}
