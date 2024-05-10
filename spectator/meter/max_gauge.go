package meter

import (
	"fmt"
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
	id              *Id
	writer          writer.Writer
	meterTypeSymbol string
}

// NewMaxGauge generates a new gauge, using the provided meter identifier.
func NewMaxGauge(id *Id, writer writer.Writer) *MaxGauge {
	return &MaxGauge{id, writer, "m"}
}

// MeterId returns the meter identifier.
func (g *MaxGauge) MeterId() *Id {
	return g.id
}

// Set records the current value.
func (g *MaxGauge) Set(value float64) {
	var line = fmt.Sprintf("%s:%s:%f", g.meterTypeSymbol, g.id.spectatordId, value)
	g.writer.Write(line)
}
