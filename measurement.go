package spectator

import "fmt"

type Measurement struct {
	id    *Id
	value float64
}

func (m Measurement) String() string {
	return fmt.Sprintf("M{id=%v, value=%f}", m.id, m.value)
}
