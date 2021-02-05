package spectator

import "math"

// Gauge represents a value that is sampled at a specific point in time. One
// example might be the pending messages in a queue. This type is safe for
// concurrent use.
//
// You can find more about this type by viewing the relevant Java Spectator
// documentation here:
//
// https://netflix.github.io/spectator/en/latest/intro/gauge/
type Gauge struct {
	valueBits uint64
	// Pointers need to be after counters to ensure 64-bit alignment. See
	// note in atomicnum.go
	id        *Id
}

// NewGauge generates a new gauge, using the provided meter identifier.
func NewGauge(id *Id) *Gauge {
	return &Gauge{math.Float64bits(math.NaN()), id}
}

// MeterId returns the meter identifier.
func (g *Gauge) MeterId() *Id {
	return g.id
}

// Measure returns the list of measurements known by the gauge. This will either
// contain one item (the current value). This also resets the internal value to
// NaN.
func (g *Gauge) Measure() []Measurement {
	return []Measurement{{g.id.WithDefaultStat("gauge"), swapFloat64(&g.valueBits, math.NaN())}}
}

// Set records the current value.
func (g *Gauge) Set(value float64) {
	storeFloat64(&g.valueBits, value)
}

// Get retrieves the current value.
func (g *Gauge) Get() float64 {
	return loadFloat64(&g.valueBits)
}
