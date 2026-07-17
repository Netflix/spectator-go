package meter

import (
	"github.com/Netflix/spectator-go/v2/spectator/writer"
)

// MaxGauge represents a value that is sampled at a specific point in time. One
// example might be the pending messages in a queue. This type is safe for
// concurrent use.
//
// You can find more about this type by viewing the relevant Java Spectator
// documentation here:
//
// https://netflix.github.io/spectator/en/latest/intro/gauge/
type MaxGauge struct {
	id         *Id
	writer     writer.Writer
	linePrefix string
}

// NewMaxGauge generates a new gauge, using the provided meter identifier.
func NewMaxGauge(id *Id, writer writer.Writer) *MaxGauge {
	return &MaxGauge{id, writer, "m:" + id.spectatordId + ":"}
}

// NewMaxGaugeDirect generates a new max gauge directly from a name and tags,
// without allocating an *Id or copying the tags map. commonTags carries the
// registry's extraCommonTags.
func NewMaxGaugeDirect(name string, tags, commonTags map[string]string, writer writer.Writer) *MaxGauge {
	return &MaxGauge{nil, writer, buildLinePrefix("m", name, tags, commonTags)}
}

// MeterId returns the meter identifier, reconstructing it from the line prefix
// if the gauge was created directly from a name and tags.
func (g *MaxGauge) MeterId() *Id {
	return resolveMeterId(g.id, g.linePrefix)
}

// Set records the current value.
func (g *MaxGauge) Set(value float64) {
	g.writer.WriteFloat(g.linePrefix, value)
}
