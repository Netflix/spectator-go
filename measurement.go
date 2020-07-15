package spectator

import "fmt"

// Measurement represents a single meter's measurement at a given point in time.
type Measurement struct {
	id    *Id
	value float64
}

func (m Measurement) String() string {
	return fmt.Sprintf("M{id=%v, value=%f}", m.id, m.value)
}

// Id returns the measurement's identifier.
func (m Measurement) Id() *Id {
	return m.id
}

// Value returns the measurement's value.
func (m Measurement) Value() float64 {
	return m.value
}

// NewMeasurement generates a new measurement, using the provided identifier and
// value.
func NewMeasurement(id *Id, Value float64) Measurement {
	return Measurement{id: id, value: Value}
}
