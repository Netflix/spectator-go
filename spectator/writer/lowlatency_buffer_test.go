package writer

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"runtime"
	"slices"
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

	if !frontBuffersActive(buffer.activeBufferState.Load()) {
		t.Errorf("Expected front buffers to be active")
	}

	// Ensure that buffer swapping and flushing logic is correct
	buffer.Write("message1")
	buffer.swapAndFlush()

	if frontBuffersActive(buffer.activeBufferState.Load()) {
		t.Errorf("Expected back buffers to be active")
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

func TestLowLatencyBuffer_PreservesOrderForSameMeterId(t *testing.T) {
	memWriter := &MemoryWriter{}
	shards := runtime.NumCPU()
	bufferSize := 2 * 2 * chunkSize * shards
	buffer := NewLowLatencyBuffer(memWriter, logger.NewDefaultLogger(), bufferSize, 3*time.Minute)
	defer buffer.Close()

	buffer.Write("g:test.metric,worker=lineage:1.000000")
	buffer.Write("g:test.metric,worker=lineage:0.000000")

	buffer.swapAndFlush()

	lines := splitAndFilterMetricLines(memWriter)
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "g:test.metric,worker=lineage:1.000000" {
		t.Errorf("Expected first line to be g:test.metric,worker=lineage:1.000000, got %s", lines[0])
	}
	if lines[1] != "g:test.metric,worker=lineage:0.000000" {
		t.Errorf("Expected second line to be g:test.metric,worker=lineage:0.000000, got %s", lines[1])
	}
}

func TestLowLatencyBuffer_PreservesPerIdOrderInterleaved(t *testing.T) {
	memWriter := &MemoryWriter{}
	shards := runtime.NumCPU()
	bufferSize := 2 * 2 * chunkSize * shards
	buffer := NewLowLatencyBuffer(memWriter, logger.NewDefaultLogger(), bufferSize, 3*time.Minute)
	defer buffer.Close()

	writes := []string{"g:a:1", "g:b:1", "g:a:0", "g:b:0"}
	for _, w := range writes {
		buffer.Write(w)
	}

	buffer.swapAndFlush()

	lines := splitAndFilterMetricLines(memWriter)

	var aLines, bLines []string
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "g:a:"):
			aLines = append(aLines, line)
		case strings.HasPrefix(line, "g:b:"):
			bLines = append(bLines, line)
		}
	}

	wantA := []string{"g:a:1", "g:a:0"}
	wantB := []string{"g:b:1", "g:b:0"}
	if !slices.Equal(aLines, wantA) {
		t.Errorf("Expected per-id order for a to be %v, got %v", wantA, aLines)
	}
	if !slices.Equal(bLines, wantB) {
		t.Errorf("Expected per-id order for b to be %v, got %v", wantB, bLines)
	}
}

func TestLowLatencyBuffer_RejectsStaleShardSelectionAfterSwap(t *testing.T) {
	memWriter := &MemoryWriter{}
	shards := runtime.NumCPU()
	bufferSize := 2 * 2 * chunkSize * shards
	buffer := NewLowLatencyBuffer(memWriter, logger.NewDefaultLogger(), bufferSize, 3*time.Minute)
	defer buffer.Close()

	staleLine := "g:swap.race:1"
	newLine := "g:swap.race:2"
	shardIndex := buffer.shardIndexFor(staleLine)
	staleState := buffer.activeBufferState.Load()
	staleShard := buffer.bufferShardForState(staleState, shardIndex)

	buffer.swapAndFlush()
	buffer.Write(newLine)
	buffer.swapAndFlush()

	if !frontBuffersActive(buffer.activeBufferState.Load()) {
		t.Fatalf("Expected front buffers to be active after two swaps")
	}
	if buffer.writeToActiveShard(staleShard, staleState, []byte(staleLine)) {
		t.Fatal("Expected stale buffer selection to be rejected")
	}

	buffer.swapAndFlush()

	lines := splitAndFilterMetricLines(memWriter)
	if len(lines) != 1 {
		t.Fatalf("Expected only the newer line to be flushed, got %d lines: %v", len(lines), lines)
	}
	if lines[0] != newLine {
		t.Fatalf("Expected flushed line %q, got %q", newLine, lines[0])
	}
}

func TestLowLatencyBuffer_OrderSensitiveMeterIdKey(t *testing.T) {
	tests := []struct {
		line string
		key  string
		ok   bool
	}{
		{line: "g:test.metric:1", key: "test.metric", ok: true},
		{line: "g,60:test.metric:1", key: "test.metric", ok: true},
		{line: "A:test.metric:1", key: "test.metric", ok: true},
		{line: "C:test.metric:1", key: "test.metric", ok: true},
		{line: "U:test.metric:1", key: "test.metric", ok: true},
		{line: "c:test.metric:1", ok: false},
		{line: "d:test.metric:1", ok: false},
		{line: "D:test.metric:1", ok: false},
		{line: "t:test.metric:1", ok: false},
		{line: "T:test.metric:1", ok: false},
		{line: "m:test.metric:1", ok: false},
		{line: "message=1", ok: false},
		{line: "g:test.metric", ok: false},
		{line: "gauge:test.metric:1", ok: false},
	}

	for _, test := range tests {
		t.Run(test.line, func(t *testing.T) {
			key, ok := orderSensitiveMeterIdKey(test.line)
			if ok != test.ok {
				t.Fatalf("Expected ok=%t, got %t", test.ok, ok)
			}
			if key != test.key {
				t.Fatalf("Expected key %q, got %q", test.key, key)
			}
		})
	}
}

func TestLowLatencyBuffer_RoundRobinsSameCounterId(t *testing.T) {
	shards := runtime.NumCPU()
	if shards < 2 {
		t.Skip("Need at least 2 CPUs to test shard distribution")
	}

	memWriter := &MemoryWriter{}
	bufferSize := 2 * 2 * chunkSize * shards
	buffer := NewLowLatencyBuffer(memWriter, logger.NewDefaultLogger(), bufferSize, 3*time.Minute)
	defer buffer.Close()

	for i := 0; i < shards*2; i++ {
		buffer.Write("c:hot.counter:1")
	}

	shardsWithData := 0
	for _, shard := range buffer.frontBuffers {
		shard.mu.Lock()
		if len(shard.data[0]) > 0 {
			shardsWithData++
		}
		shard.mu.Unlock()
	}

	if shardsWithData < 2 {
		t.Errorf("Expected same-id counter writes to round-robin across shards, got %d shard(s) with data", shardsWithData)
	}
}

func TestLowLatencyBuffer_DistributesIdsAcrossShards(t *testing.T) {
	shards := runtime.NumCPU()
	if shards < 2 {
		t.Skip("Need at least 2 CPUs to test shard distribution")
	}

	memWriter := &MemoryWriter{}
	bufferSize := 2 * 2 * chunkSize * shards
	buffer := NewLowLatencyBuffer(memWriter, logger.NewDefaultLogger(), bufferSize, 3*time.Minute)
	defer buffer.Close()

	numIds := shards * 8
	for i := 0; i < numIds; i++ {
		buffer.Write(fmt.Sprintf("g:metric.%d:1", i))
	}

	shardsWithData := 0
	for _, shard := range buffer.frontBuffers {
		shard.mu.Lock()
		if len(shard.data[0]) > 0 {
			shardsWithData++
		}
		shard.mu.Unlock()
	}

	if shardsWithData < 2 {
		t.Errorf("Expected at least 2 shards to have data when writing %d distinct ids across %d shards, got %d", numIds, shards, shardsWithData)
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
			msg := strings.Repeat("x", chunkSize+1)
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
