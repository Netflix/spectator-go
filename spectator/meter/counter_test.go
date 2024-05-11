package meter

import (
	"github.com/Netflix/spectator-go/spectator/writer"
	"testing"
)

func TestCounter_Increment(t *testing.T) {
	w := writer.MemoryWriter{}
	id := NewId("inc", nil)
	c := NewCounter(id, &w)

	c.Increment()

	expected := "c:inc:1"
	if w.Lines[0] != expected {
		t.Error("Expected ", expected, " got ", w.Lines[0])
	}
}

func TestCounter_Add(t *testing.T) {
	w := writer.MemoryWriter{}
	id := NewId("add", nil)
	c := NewCounter(id, &w)

	c.Add(4)

	expected := "c:add:4"
	if w.Lines[0] != expected {
		t.Error("Expected ", expected, " got ", w.Lines[0])
	}

	c.Add(-1)
	if len(w.Lines) != 1 {
		t.Error("Negative deltas should be ignored")
	}
}

func TestCounter_AddFloat(t *testing.T) {
	w := writer.MemoryWriter{}
	id := NewId("addFloat", nil)
	c := NewCounter(id, &w)

	c.AddFloat(4.2)

	expected := "c:addFloat:4.200000"
	if w.Lines[0] != expected {
		t.Error("Expected ", expected, " got ", w.Lines[0])
	}

	c.AddFloat(-0.1)
	if len(w.Lines) != 1 {
		t.Error("Negative deltas should be ignored")
	}
}
