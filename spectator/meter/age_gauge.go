package meter

import (
	"fmt"
	"github.com/Netflix/spectator-go/spectator/writer"
)

// AgeGauge represents a value that is the time in seconds since the epoch at which an event
// has successfully occurred, or 0 to use the current time in epoch seconds. After an Age Gauge
// has been set, it will continue reporting the number of seconds since the last time recorded,
// for as long as the spectatord process runs. The purpose of this metric type is to enable users
// to more easily implement the Time Since Last Success alerting pattern.
//
// To set `now()` as the last success, set a value of 0.
type AgeGauge struct {
	id              *Id
	writer          writer.Writer
	meterTypeSymbol string
}

// NewAgeGauge generates a new gauge, using the provided meter identifier.
func NewAgeGauge(id *Id, writer writer.Writer) *AgeGauge {
	return &AgeGauge{id, writer, "A"}
}

// MeterId returns the meter identifier.
func (g *AgeGauge) MeterId() *Id {
	return g.id
}

// Set records the current time in seconds since the epoch.
func (g *AgeGauge) Set(seconds int64) {
	if seconds >= 0 {
		var line = fmt.Sprintf("%s:%s:%d", g.meterTypeSymbol, g.id.spectatordId, seconds)
		g.writer.Write(line)
	}
}

// Now records the current time in epoch seconds, using a spectatord feature.
func (g *AgeGauge) Now() {
	var line = fmt.Sprintf("%s:%s:0", g.meterTypeSymbol, g.id.spectatordId)
	g.writer.Write(line)
}
