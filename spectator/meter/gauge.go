package meter

import (
	"fmt"
	"strconv"

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

// MeterId returns the meter identifier.
func (g *Gauge) MeterId() *Id {
	return g.id
}

// Set records the current value.
func (g *Gauge) Set(value float64) {
	g.writer.Write(g.linePrefix + strconv.FormatFloat(value, 'f', 6, 64))
}
