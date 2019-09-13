package spectator

import (
	"math"
	"reflect"
	"testing"
)

func getGauge(name string) *Gauge {
	return NewGauge(NewId(name, nil))
}

func TestGauge_Init(t *testing.T) {
	g := getGauge("g")
	v := g.Get()
	if !math.IsNaN(v) {
		t.Error("Gauges should not have an initial value: ", v)
	}
}

func TestGauge_Set(t *testing.T) {
	g := getGauge("g")
	g.Set(1.0)
	if v := g.Get(); v != 1.0 {
		t.Error("Expected 1.0, got ", v)
	}
}

func TestGauge_Measure(t *testing.T) {
	g := getGauge("g")
	g.Set(42.0)
	ms := g.Measure()

	expectedId := NewId("g", map[string]string{"statistic": "gauge"})
	expected := []Measurement{{expectedId, 42.0}}
	if !reflect.DeepEqual(expected, ms) {
		t.Error("Unexpected measurements: ", ms)
	}

	if v := g.Get(); !math.IsNaN(v) {
		t.Error("Gauge values should be reset after being measured, got ", v)
	}
}
