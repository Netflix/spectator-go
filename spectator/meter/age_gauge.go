package meter

import (
	"github.com/Netflix/spectator-go/v2/spectator/writer"
)

// AgeGauge represents a value that is the time in seconds since the epoch at which an event
// has successfully occurred, or 0 to use the current time in epoch seconds. After an Age Gauge
// has been set, it will continue reporting the number of seconds since the last time recorded,
// for as long as the spectatord process runs. The purpose of this metric type is to enable users
// to more easily implement the Time Since Last Success alerting pattern.
//
// To set `now()` as the last success, set a value of 0.
type AgeGauge struct {
	id         *Id
	writer     writer.Writer
	linePrefix string
}

// NewAgeGauge generates a new gauge, using the provided meter identifier.
func NewAgeGauge(id *Id, writer writer.Writer) *AgeGauge {
	return &AgeGauge{id, writer, "A:" + id.spectatordId + ":"}
}

// NewAgeGaugeDirect generates a new age gauge directly from a name and tags,
// without allocating an *Id or copying the tags map. commonTags carries the
// registry's extraCommonTags.
func NewAgeGaugeDirect(name string, tags, commonTags map[string]string, writer writer.Writer) *AgeGauge {
	return &AgeGauge{nil, writer, buildLinePrefix("A", name, tags, commonTags)}
}

// MeterId returns the meter identifier, reconstructing it from the line prefix
// if the gauge was created directly from a name and tags.
func (g *AgeGauge) MeterId() *Id {
	return resolveMeterId(g.id, g.linePrefix)
}

// Set records the current time in seconds since the epoch.
func (g *AgeGauge) Set(seconds int64) {
	if seconds >= 0 {
		g.writer.WriteInt(g.linePrefix, seconds)
	}
}

// Now records the current time in epoch seconds, using a spectatord feature.
func (g *AgeGauge) Now() {
	g.writer.WriteLine(g.linePrefix, "0")
}
