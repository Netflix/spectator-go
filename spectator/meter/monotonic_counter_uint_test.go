package meter

import (
	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"testing"
)

func TestMonotonicCounterUint_Set(t *testing.T) {
	w := writer.MemoryWriter{}
	id := NewId("set", nil)
	c := NewMonotonicCounterUint(id, &w)

	c.Set(4)

	expected := "U:set:4"
	if w.Lines()[0] != expected {
		t.Error("Expected ", expected, " got ", w.Lines()[0])
	}
}
