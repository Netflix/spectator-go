package writer

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"runtime"
	"strings"
	"testing"
	"time"
)

func splitAndFilterMetricLines(memWriter *MemoryWriter) []string {
	var lines []string
	for _, mwLine := range memWriter.Lines() {
		for _, protocolLine := range strings.Split(mwLine, separator) {
			if strings.Contains(protocolLine, "spectator-go.lowLatencyBuffer") {
				// The number of buffer stats metric lines depends on flush activity - discard
				continue
			}
			lines = append(lines, protocolLine)
		}
	}
	return lines
}

func TestLowLatencyBuffer_ShardMessageDistribution(t *testing.T) {
	memWriter := &MemoryWriter{}
	shards := runtime.NumCPU()
	bufferSize := 2 * 2 * chunkSize * shards // two buffer sets, two 60 KB chunks/shard
	buffer := NewLowLatencyBuffer(memWriter, logger.NewDefaultLogger(), bufferSize, 5*time.Millisecond)
	defer buffer.Close()

	// Verify that the buffer sets have the expected shard count
	if len(buffer.frontBuffers) != shards {
		t.Errorf("Expected %d shards in the front buffer set, got %d", shards, len(buffer.frontBuffers))
	}
	if len(buffer.backBuffers) != shards {
		t.Errorf("Expected %d shards in the back buffer set, got %d", shards, len(buffer.backBuffers))
	}

	// Write messages and verify they're distributed across buffer shards
	numMessages := shards * 10
	for i := 0; i < numMessages; i++ {
		buffer.Write(fmt.Sprintf("message=%v,", i))
	}

	// Check that multiple buffer shards have data
	shardsWithData := 0
	for _, shard := range buffer.frontBuffers {
		shard.mu.Lock()
		if len(shard.data) > 0 {
			shardsWithData++
		}
		shard.mu.Unlock()
	}

	if shardsWithData == 0 {
		t.Error("No buffer shards have data, data distribution may not be working")
	}

	// Wait for flush and verify all messages are received, filtering out statistics metrics
	time.Sleep(15 * time.Millisecond)

	lines := splitAndFilterMetricLines(memWriter)
	if len(lines) != numMessages {
		t.Errorf("Expected %d protocol lines in MemoryWriter, got %d", numMessages, len(lines))
	}
}

func TestLowLatencyBuffer_FrontBuffersFlushFirst(t *testing.T) {
	// Create a buffer instance with a long flush interval timer, to allow for manual flush trigger
	memWriter := &MemoryWriter{}
	shards := runtime.NumCPU()
	bufferSize := 2 * 2 * chunkSize * shards // two buffer sets, two 60 KB chunks/shard
	buffer := NewLowLatencyBuffer(memWriter, logger.NewDefaultLogger(), bufferSize, 3*time.Minute)
	defer buffer.Close()

	if buffer.useFrontBuffers.Load() != true {
		t.Errorf("Expected useFrontBuffers to be true")
	}

	// Ensure that buffer swapping and flushing logic is correct
	buffer.Write("message1")
	buffer.swapAndFlush()

	if buffer.useFrontBuffers.Load() != false {
		t.Errorf("Expected useFrontBuffers to be false")
	}

	lines := splitAndFilterMetricLines(memWriter)
	if len(lines) != 1 {
		t.Errorf("Expected %d lines in the front buffer set, got %d", 1, len(lines))
	}
	if lines[0] != "message1" {
		t.Errorf("Expected first message to be message1, got %s", lines[0])
	}
}

func TestLowLatencyBuffer_ChunkBoundaries_HalfSize(t *testing.T) {
	// Create a buffer instance with a long flush interval timer, to allow for manual flush trigger
	memWriter := &MemoryWriter{}
	shards := runtime.NumCPU()
	bufferSize := 2 * 2 * chunkSize * shards // two buffer sets, two 60 KB chunks/shard
	buffer := NewLowLatencyBuffer(memWriter, logger.NewDefaultLogger(), bufferSize, 3*time.Minute)
	defer buffer.Close()

	var totalMessages int

	// Fill all chunks with half the max size message
	for i := 0; i < 2; i++ {
		for j := 0; j < shards; j++ {
			msg := strings.Repeat("x", chunkSize/2)
			buffer.Write(msg)
			totalMessages++
		}
	}

	buffer.swapAndFlush()

	// Verify the total number of lines received
	lines := splitAndFilterMetricLines(memWriter)
	if len(lines) != totalMessages {
		t.Errorf("Expected %d lines, got %d", totalMessages, len(lines))
	}

	// Verify the messages match the chunk size
	for _, line := range lines {
		if len(line) != chunkSize/2 {
			t.Errorf("Expected %d message size, got %d", chunkSize/2, len(line))
		}
	}
}

func TestLowLatencyBuffer_ChunkBoundaries_MaxSize(t *testing.T) {
	// Create a buffer instance with a long flush interval timer, to allow for manual flush trigger
	memWriter := &MemoryWriter{}
	shards := runtime.NumCPU()
	bufferSize := 2 * 2 * chunkSize * shards // two buffer sets, two 60 KB chunks/shard
	buffer := NewLowLatencyBuffer(memWriter, logger.NewDefaultLogger(), bufferSize, 3*time.Minute)
	defer buffer.Close()

	var totalMessages int

	// Fill all chunks with the max size message
	for i := 0; i < 2; i++ {
		for j := 0; j < shards; j++ {
			msg := strings.Repeat("x", chunkSize)
			buffer.Write(msg)
			totalMessages++
		}
	}

	buffer.swapAndFlush()

	// Verify the total number of lines received
	lines := splitAndFilterMetricLines(memWriter)
	if len(lines) != totalMessages {
		t.Errorf("Expected %d lines, got %d", totalMessages, len(lines))
	}

	// Verify the messages match the chunk size
	for _, line := range lines {
		if len(line) != chunkSize {
			t.Errorf("Expected %d message size, got %d", chunkSize, len(line))
		}
	}
}

func TestLowLatencyBuffer_ChunkBoundaries_OverMaxSizeIsDropped(t *testing.T) {
	// Create a buffer instance with a long flush interval timer, to allow for manual flush trigger
	memWriter := &MemoryWriter{}
	shards := runtime.NumCPU()
	bufferSize := 2 * 2 * chunkSize * shards // two buffer sets, two 60 KB chunks/shard
	buffer := NewLowLatencyBuffer(memWriter, logger.NewDefaultLogger(), bufferSize, 3*time.Minute)
	defer buffer.Close()

	// Fill all chunks with an over max size message
	for i := 0; i < 2; i++ {
		for j := 0; j < shards; j++ {
			msg := strings.Repeat("x", chunkSize + 1)
			buffer.Write(msg)
		}
	}

	buffer.swapAndFlush()

	// Verify the total number of lines received
	lines := splitAndFilterMetricLines(memWriter)
	if len(lines) != 0 {
		t.Errorf("Expected %d lines, got %d", 0, len(lines))
	}
}
