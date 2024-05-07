package meter

import (
	"github.com/Netflix/spectator-go/spectator/writer"
	"testing"
)

func TestAgeGauge_Set(t *testing.T) {
	id := NewId("set", nil)
	w := writer.MemoryWriter{}
	g := NewAgeGauge(id, &w)
	g.Set(100)

	expected := "A:set:100"
	if w.Lines[0] != expected {
		t.Errorf("Expected line to be %s, got %s", expected, w.Lines[0])
	}
}

func TestAgeGauge_SetZero(t *testing.T) {
	id := NewId("setZero", nil)
	w := writer.MemoryWriter{}
	g := NewAgeGauge(id, &w)
	g.Set(0)

	expected := "A:setZero:0"
	if w.Lines[0] != expected {
		t.Errorf("Expected line to be %s, got %s", expected, w.Lines[0])
	}
}

func TestAgeGauge_Now(t *testing.T) {
	id := NewId("now", nil)
	w := writer.MemoryWriter{}
	g := NewAgeGauge(id, &w)
	g.Now()

	expected := "A:now:0"
	if w.Lines[0] != expected {
		t.Errorf("Expected line to be %s, got %s", expected, w.Lines[0])
	}
}

func TestAgeGauge_SetNegative(t *testing.T) {
	id := NewId("setNegative", nil)
	w := writer.MemoryWriter{}
	g := NewAgeGauge(id, &w)
	g.Set(-100)

	if len(w.Lines) != 0 {
		t.Error("Negative values should be ignored")
	}
}

func TestAgeGauge_SetMultipleValues(t *testing.T) {
	id := NewId("setMultiple", nil)
	w := writer.MemoryWriter{}
	g := NewAgeGauge(id, &w)
	g.Set(100)
	g.Set(200)
	g.Set(300)

	expectedLines := []string{"A:setMultiple:100", "A:setMultiple:200", "A:setMultiple:300"}
	for i, line := range w.Lines {
		if line != expectedLines[i] {
			t.Errorf("Expected line to be %s, got %s", expectedLines[i], line)
		}
	}
}
