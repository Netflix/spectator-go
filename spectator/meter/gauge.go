package meter

import (
	"fmt"

	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"time"
)

// Gauge represents a value that is sampled at a specific point in time. One
// example might be the pending messages in a queue. This type is safe for
// concurrent use.
//
// You can find more about this type by viewing the relevant Java Spectator
// documentation here:
//
// https://netflix.github.io/spectator/en/latest/intro/gauge/
type Gauge struct {
	id         *Id
	writer     writer.Writer
	linePrefix string
}

// NewGauge generates a new gauge, using the provided meter identifier.
func NewGauge(id *Id, writer writer.Writer) *Gauge {
	return &Gauge{id, writer, "g:" + id.spectatordId + ":"}
}

// NewGaugeWithTTL generates a new gauge, using the provided meter identifier and ttl.
func NewGaugeWithTTL(id *Id, writer writer.Writer, ttl time.Duration) *Gauge {
	return &Gauge{id, writer, fmt.Sprintf("g,%d", int(ttl.Seconds())) + ":" + id.spectatordId + ":"}
}

// NewGaugeDirect generates a new gauge directly from a name and tags, without
// allocating an *Id or copying the tags map. commonTags carries the registry's
// extraCommonTags.
func NewGaugeDirect(name string, tags, commonTags map[string]string, writer writer.Writer) *Gauge {
	return &Gauge{nil, writer, buildLinePrefix("g", name, tags, commonTags)}
}

// NewGaugeDirectWithTTL generates a new gauge with a ttl directly from a name and tags.
func NewGaugeDirectWithTTL(name string, tags, commonTags map[string]string, writer writer.Writer, ttl time.Duration) *Gauge {
	return &Gauge{nil, writer, buildLinePrefix(fmt.Sprintf("g,%d", int(ttl.Seconds())), name, tags, commonTags)}
}

// MeterId returns the meter identifier, reconstructing it from the line prefix
// if the gauge was created directly from a name and tags.
func (g *Gauge) MeterId() *Id {
	return resolveMeterId(g.id, g.linePrefix)
}

// Set records the current value.
func (g *Gauge) Set(value float64) {
	g.writer.WriteFloat(g.linePrefix, value)
}

// SetInt records the current value as an integer, avoiding the overhead of
// float64 conversion and formatting.
func (g *Gauge) SetInt(value int64) {
	g.writer.WriteInt(g.linePrefix, value)
}
