package writer

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Netflix/spectator-go/v2/spectator/logger"
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
	if staleShard.appendLine(&buffer.activeBufferState, staleState, staleLine, "") {
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

func TestLowLatencyBuffer_MeterIdKey(t *testing.T) {
	tests := []struct {
		line string
		key  string
		ok   bool
	}{
		{line: "g:test.metric:1", key: "test.metric", ok: true},
		{line: "g,60:test.metric:1", key: "test.metric", ok: true},
		{line: "A:test.metric:1", key: "test.metric", ok: true},
		// Unlike orderSensitiveMeterIdKey, commutative types yield an id
		{line: "c:test.metric:1", key: "test.metric", ok: true},
		{line: "d:test.metric:1", key: "test.metric", ok: true},
		{line: "t:test.metric:1", key: "test.metric", ok: true},
		{line: "gauge:test.metric:1", key: "test.metric", ok: true},
		{line: "c:test.metric,app=foo:1", key: "test.metric,app=foo", ok: true},
		// WriteLine passes the prefix alone, which ends with the value separator
		{line: "c:test.metric:", key: "test.metric", ok: true},
		{line: ":test.metric:1", key: "test.metric", ok: true},
		{line: "message=1", ok: false},
		{line: "g:test.metric", ok: false},
		{line: "", ok: false},
	}

	for _, test := range tests {
		t.Run(test.line, func(t *testing.T) {
			key, ok := meterIdKey(test.line)
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

// captureLogger records Infof messages, so the once-per-flush-window overflow log line
// can be inspected.
type captureLogger struct {
	mu       sync.Mutex
	messages []string
}

var _ logger.Logger = (*captureLogger)(nil)

func (l *captureLogger) Debugf(_ string, _ ...interface{}) {}

func (l *captureLogger) Infof(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, fmt.Sprintf(format, v...))
}

func (l *captureLogger) Errorf(_ string, _ ...interface{}) {}

// overflowMessages returns the overflow log lines, discarding the initialization message.
func (l *captureLogger) overflowMessages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var messages []string
	for _, message := range l.messages {
		if strings.Contains(message, "LowLatencyBuffer overflow") {
			messages = append(messages, message)
		}
	}
	return messages
}

// newSingleChunkBuffer creates a buffer with one chunk per shard and a long flush interval
// timer, to allow for manual flush trigger. One chunk per shard means a single chunkSize
// line fills a shard, and every subsequent line written to it is dropped.
func newSingleChunkBuffer(t *testing.T, memWriter *MemoryWriter, log logger.Logger) *LowLatencyBuffer {
	t.Helper()
	shards := runtime.NumCPU()
	bufferSize := 2 * chunkSize * shards // two buffer sets, one 60 KB chunk/shard
	buffer := NewLowLatencyBuffer(memWriter, log, bufferSize, 3*time.Minute)
	t.Cleanup(buffer.Close)
	return buffer
}

// appendToShard writes a line to a specific shard, bypassing shard selection, so tests can
// drive overflows deterministically regardless of the number of CPUs.
func appendToShard(t *testing.T, buffer *LowLatencyBuffer, shard *bufferShard, line string) {
	t.Helper()
	state := buffer.activeBufferState.Load()
	if !shard.appendLine(&buffer.activeBufferState, state, line, "") {
		t.Fatalf("Expected line %q to be accepted by the active buffer set", line)
	}
}

// fillShard writes one chunkSize line to the shard, which leaves the shard with data to
// flush and no room for any further line.
func fillShard(t *testing.T, buffer *LowLatencyBuffer, shard *bufferShard) {
	t.Helper()
	const prefix = "g:overflow.fill:"
	appendToShard(t, buffer, shard, prefix+strings.Repeat("1", chunkSize-len(prefix)))

	shard.mu.Lock()
	defer shard.mu.Unlock()
	if shard.overflows != 0 {
		t.Fatalf("Expected the fill line to be stored, got %d overflow(s)", shard.overflows)
	}
}

func TestBufferShard_RecordOverflowSamplesFirstDroppedId(t *testing.T) {
	shard := &bufferShard{}

	shard.recordOverflow("g:first.dropped:1", 17)
	shard.recordOverflow("g:second.dropped:1", 18)

	if shard.overflows != 2 {
		t.Errorf("Expected %d overflows, got %d", 2, shard.overflows)
	}
	if shard.overflowBytes != 35 {
		t.Errorf("Expected %d overflow bytes, got %d", 35, shard.overflowBytes)
	}
	if shard.droppedSampleId != "first.dropped" {
		t.Errorf("Expected the first dropped meter id to be sampled, got %q", shard.droppedSampleId)
	}
}

func TestLowLatencyBuffer_OverflowLogsOncePerFlushWindow(t *testing.T) {
	memWriter := &MemoryWriter{}
	log := &captureLogger{}
	buffer := newSingleChunkBuffer(t, memWriter, log)

	shard := buffer.frontBuffers[0]
	fillShard(t, buffer, shard)

	dropped := []string{"g:overflow.dropped:1", "g:overflow.dropped:2", "c:other.dropped:3"}
	var droppedBytes int
	for _, line := range dropped {
		appendToShard(t, buffer, shard, line)
		droppedBytes += len(line)
	}

	buffer.swapAndFlush()

	messages := log.overflowMessages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 overflow log line for the flush window, got %d: %v", len(messages), messages)
	}
	message := messages[0]
	for _, want := range []string{
		"bufferSet=front",
		fmt.Sprintf("dropped %d lines (%d bytes)", len(dropped), droppedBytes),
		"sample dropped meter id: overflow.dropped",
	} {
		if !strings.Contains(message, want) {
			t.Errorf("Expected overflow log line to contain %q, got %q", want, message)
		}
	}

	// The per-shard overflow metrics are unchanged by the log line
	overflowMetrics := map[string]bool{
		fmt.Sprintf("c:spectator-go.lowLatencyBuffer.overflows,bufferSet=front:%d", len(dropped)):     false,
		fmt.Sprintf("d:spectator-go.lowLatencyBuffer.overflowBytes,bufferSet=front:%d", droppedBytes): false,
	}
	for _, line := range memWriter.Lines() {
		if _, ok := overflowMetrics[line]; ok {
			overflowMetrics[line] = true
		}
	}
	for line, found := range overflowMetrics {
		if !found {
			t.Errorf("Expected metric line %q in MemoryWriter, got %v", line, memWriter.Lines())
		}
	}
}
