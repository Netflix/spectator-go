package writer

import (
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"testing"
	"time"
)

func TestLineBuffer_FlushesOnSizeExceeded(t *testing.T) {
	memWriter := &MemoryWriter{}
	buffer := NewLineBuffer(memWriter, logger.NewDefaultLogger(), 20, 5*time.Second)

	buffer.Write("short")
	lines := memWriter.Lines()
	if len(lines) != 0 {
		t.Errorf("Expected 0 lines before size exceeded, got %d: %s", len(lines), lines)
	}

	buffer.Write("this_is_a_longer_line")
	lines = memWriter.Lines()
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines after buffer size exceeded, got %d: %s", len(lines), lines)
	}

	expected := [...]string{
		"c:spectator-go.lineBuffer.overflows:1",
		"short\nthis_is_a_longer_line",
		"c:spectator-go.lineBuffer.bytesWritten:27",
	}
	for idx, line := range lines {
		if line != expected[idx] {
			t.Errorf("Expected '%s', got '%s'", expected[idx], line)
		}
	}
}

func TestLineBuffer_FlushesOnTimeout(t *testing.T) {
	memWriter := &MemoryWriter{}
	buffer := NewLineBuffer(memWriter, logger.NewDefaultLogger(), 1000, 1*time.Millisecond)

	buffer.Write("line1")
	buffer.Write("line2")

	lines := memWriter.Lines()
	if len(lines) != 0 {
		t.Errorf("Expected 0 lines before timeout, got %d", len(lines))
	}

	time.Sleep(2 * time.Millisecond)

	lines = memWriter.Lines()
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines after timeout, got %d: %s", len(lines), lines)
	}

	expected := [...]string{
		"line1\nline2",
		"c:spectator-go.lineBuffer.bytesWritten:11",
	}
	for idx, expect := range expected {
		if expect != lines[idx] {
			t.Errorf("Expected '%s', got '%s'", expect, lines[idx])
		}
	}
}

func TestLineBuffer_FlushesOnClose(t *testing.T) {
	memWriter := &MemoryWriter{}
	buffer := NewLineBuffer(memWriter, logger.NewDefaultLogger(), 1000, 5*time.Second)

	buffer.Write("line1")
	buffer.Write("line2")

	lines := memWriter.Lines()
	if len(lines) != 0 {
		t.Errorf("Expected 0 lines before close, got %d", len(lines))
	}

	buffer.Close()

	lines = memWriter.Lines()
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines after close, got %d: %s", len(lines), lines)
	}

	expected := [...]string{
		"line1\nline2",
		"c:spectator-go.lineBuffer.bytesWritten:11",
	}
	for idx, expect := range expected {
		if expect != lines[idx] {
			t.Errorf("Expected '%s', got '%s'", expect, lines[idx])
		}
	}
}
