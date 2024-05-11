package meter

import (
	"github.com/Netflix/spectator-go/spectator/writer"
	"testing"
)

func TestMonotonicCounter_Set(t *testing.T) {
	w := writer.MemoryWriter{}
	id := NewId("add", nil)
	c := NewMonotonicCounter(id, &w)

	c.Set(4)

	expected := "C:add:4"
	if w.Lines[0] != expected {
		t.Error("Expected ", expected, " got ", w.Lines[0])
	}

	c.Set(-1)
	if len(w.Lines) != 1 {
		t.Error("Negative deltas should be ignored")
	}
}
