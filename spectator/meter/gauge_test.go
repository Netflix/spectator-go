package meter

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"testing"
	"time"
)

func TestGauge_Set(t *testing.T) {
	id := NewId("set", nil)
	w := writer.MemoryWriter{}
	g := NewGauge(id, &w)
	g.Set(100.1)

	expected := "g:set:100.100000"
	if w.Lines()[0] != expected {
		t.Errorf("Expected line to be %s, got %s", expected, w.Lines()[0])
	}
}

func TestGauge_SetZero(t *testing.T) {
	id := NewId("setZero", nil)
	w := writer.MemoryWriter{}
	g := NewGauge(id, &w)
	g.Set(0)

	expected := "g:setZero:0.000000"
	if w.Lines()[0] != expected {
		t.Errorf("Expected line to be %s, got %s", expected, w.Lines()[0])
	}
}

func TestGauge_SetNegative(t *testing.T) {
	id := NewId("setNegative", nil)
	w := writer.MemoryWriter{}
	g := NewGauge(id, &w)
	g.Set(-100.1)

	expected := "g:setNegative:-100.100000"
	if w.Lines()[0] != expected {
		t.Errorf("Expected line to be %s, got %s", expected, w.Lines()[0])
	}
}

func TestGauge_SetMultipleValues(t *testing.T) {
	id := NewId("setMultiple", nil)
	w := writer.MemoryWriter{}
	g := NewGauge(id, &w)
	g.Set(100.1)
	g.Set(200.2)
	g.Set(300.3)

	expectedLines := []string{"g:setMultiple:100.100000", "g:setMultiple:200.200000", "g:setMultiple:300.300000"}
	for i, line := range w.Lines() {
		if line != expectedLines[i] {
			t.Errorf("Expected line to be %s, got %s", expectedLines[i], line)
		}
	}
}

func TestGaugeWithTTL_Set(t *testing.T) {
	id := NewId("setWithTTL", nil)
	w := writer.MemoryWriter{}
	ttl := 60 * time.Second
	g := NewGaugeWithTTL(id, &w, ttl)
	g.Set(100.1)

	expected := fmt.Sprintf("g,%d:setWithTTL:100.100000", int(ttl.Seconds()))
	if w.Lines()[0] != expected {
		t.Errorf("Expected line to be %s, got %s", expected, w.Lines()[0])
	}
}
