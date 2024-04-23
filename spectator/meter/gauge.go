package meter

import (
	"fmt"
	"github.com/Netflix/spectator-go/spectator/writer"
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
	id              *Id
	writer          writer.Writer
	meterTypeSymbol rune
}

// NewGauge generates a new gauge, using the provided meter identifier.
func NewGauge(id *Id, writer writer.Writer) *Gauge {
	return &Gauge{id, writer, 'g'}
}

// MeterId returns the meter identifier.
func (g *Gauge) MeterId() *Id {
	return g.id
}

// Set records the current value.
func (g *Gauge) Set(value float64) {
	var line = fmt.Sprintf("%c:%s:%f", g.meterTypeSymbol, g.id.spectatordId, value)
	g.writer.Write(line)
}
