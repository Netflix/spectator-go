package writer

import (
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"testing"
	"time"
)

func TestLineBuffer_DisabledWhenBufferSizeZero(t *testing.T) {
	memWriter := &MemoryWriter{}
	buffer := NewLineBuffer(memWriter, 0, logger.NewDefaultLogger())
	
	buffer.Write("line1")
	buffer.Write("line2")
	
	lines := memWriter.Lines()
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "line1" {
		t.Errorf("Expected 'line1', got '%s'", lines[0])
	}
	if lines[1] != "line2" {
		t.Errorf("Expected 'line2', got '%s'", lines[1])
	}
}

func TestLineBuffer_FlushesOnSizeExceeded(t *testing.T) {
	memWriter := &MemoryWriter{}
	buffer := NewLineBuffer(memWriter, 20, logger.NewDefaultLogger())
	
	buffer.Write("short")
	lines := memWriter.Lines()
	if len(lines) != 0 {
		t.Errorf("Expected 0 lines before buffer full, got %d", len(lines))
	}
	
	buffer.Write("this_is_a_longer_line")
	lines = memWriter.Lines()
	if len(lines) != 1 {
		t.Errorf("Expected 1 line after buffer full, got %d", len(lines))
	}
	
	expected := "short\nthis_is_a_longer_line"
	if lines[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, lines[0])
	}
}

func TestLineBuffer_FlushesOnTimeout(t *testing.T) {
	memWriter := &MemoryWriter{}
	buffer := NewLineBuffer(memWriter, 1000, logger.NewDefaultLogger(), 1 * time.Millisecond)
	
	buffer.Write("line1")
	buffer.Write("line2")
	
	lines := memWriter.Lines()
	if len(lines) != 0 {
		t.Errorf("Expected 0 lines before timeout, got %d", len(lines))
	}
	
	time.Sleep(2 * time.Millisecond)
	
	lines = memWriter.Lines()
	if len(lines) != 1 {
		t.Errorf("Expected 1 line after timeout, got %d", len(lines))
	}
	
	expected := "line1\nline2"
	if lines[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, lines[0])
	}
}

func TestLineBuffer_FlushesOnClose(t *testing.T) {
	memWriter := &MemoryWriter{}
	buffer := NewLineBuffer(memWriter, 1000, logger.NewDefaultLogger())
	
	buffer.Write("line1")
	buffer.Write("line2")
	
	lines := memWriter.Lines()
	if len(lines) != 0 {
		t.Errorf("Expected 0 lines before close, got %d", len(lines))
	}
	
	buffer.Close()
	
	lines = memWriter.Lines()
	if len(lines) != 1 {
		t.Errorf("Expected 1 line after close, got %d", len(lines))
	}
	
	expected := "line1\nline2"
	if lines[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, lines[0])
	}
}

func TestLineBuffer_IgnoresWritesAfterClose(t *testing.T) {
	memWriter := &MemoryWriter{}
	buffer := NewLineBuffer(memWriter, 1000, logger.NewDefaultLogger())
	
	buffer.Write("line1")
	buffer.Close()
	buffer.Write("line2")
	
	lines := memWriter.Lines()
	if len(lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(lines))
	}
	if lines[0] != "line1" {
		t.Errorf("Expected 'line1', got '%s'", lines[0])
	}
}
