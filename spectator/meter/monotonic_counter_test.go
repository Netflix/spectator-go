package meter

import (
	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"testing"
)

func TestMonotonicCounter_Set(t *testing.T) {
	w := writer.MemoryWriter{}
	id := NewId("set", nil)
	c := NewMonotonicCounter(id, &w)

	c.Set(4)

	expected := "C:set:4"
	if w.Lines()[0] != expected {
		t.Error("Expected ", expected, " got ", w.Lines()[0])
	}
}
