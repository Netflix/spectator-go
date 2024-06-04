package meter

import (
	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"testing"
)

func TestMaxGauge_Set(t *testing.T) {
	id := NewId("setMaxGauge", nil)
	w := writer.MemoryWriter{}
	g := NewMaxGauge(id, &w)
	g.Set(100.1)

	expected := "m:setMaxGauge:100.100000"
	if w.Lines()[0] != expected {
		t.Errorf("Expected line to be %s, got %s", expected, w.Lines()[0])
	}
}

func TestMaxGauge_SetZero(t *testing.T) {
	id := NewId("setMaxGaugeZero", nil)
	w := writer.MemoryWriter{}
	g := NewMaxGauge(id, &w)
	g.Set(0)

	expected := "m:setMaxGaugeZero:0.000000"
	if w.Lines()[0] != expected {
		t.Errorf("Expected line to be %s, got %s", expected, w.Lines()[0])
	}
}

func TestMaxGauge_SetNegative(t *testing.T) {
	id := NewId("setMaxGaugeNegative", nil)
	w := writer.MemoryWriter{}
	g := NewMaxGauge(id, &w)
	g.Set(-100.1)

	expected := "m:setMaxGaugeNegative:-100.100000"
	if w.Lines()[0] != expected {
		t.Errorf("Expected line to be %s, got %s", expected, w.Lines()[0])
	}
}

func TestMaxGauge_SetMultipleValues(t *testing.T) {
	id := NewId("setMaxGaugeMultiple", nil)
	w := writer.MemoryWriter{}
	g := NewMaxGauge(id, &w)
	g.Set(100.1)
	g.Set(200.2)
	g.Set(300.3)

	expectedLines := []string{"m:setMaxGaugeMultiple:100.100000", "m:setMaxGaugeMultiple:200.200000", "m:setMaxGaugeMultiple:300.300000"}
	for i, line := range w.Lines() {
		if line != expectedLines[i] {
			t.Errorf("Expected line to be %s, got %s", expectedLines[i], line)
		}
	}
}
