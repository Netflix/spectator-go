package writer

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Netflix/spectator-go/v2/spectator/logger"
)

// chunkSize is set to 60KB, to ensure each message fits in the socket buffer (64KB), with some room
// to accommodate the last spectatord protocol line appended. The maximum length of a well-formed
// protocol line is 3,927 characters (3.8KB).
const chunkSize = 60 * 1024

// separator is the character used to indicate the end of a spectatord protocol line, when combining
// lines into a larger socket payload
const separator = "\n"

// bufferShard is the atomic unit of buffering, used to store one or more chunks of spectatord
// protocol lines. Order-sensitive meter types are selected deterministically by meter id, so
// that sequential updates for the same meter land in the same shard and are flushed in the
// order they were written. Other lines are distributed round-robin across shards. The number
// of shards is scaled to the number of CPUs on the system.
// The purpose of this design is to spread buffer access across a reasonable number of mutexes,
// in order to reduce overall latency when writing to the buffer. The impact of the shard design
// is less than the impact of the front and back buffer design, but it is still important for
// throughput reasons.
type bufferShard struct {
	data          [][]byte // Array of chunkSize chunks of spectatord protocol lines, stored as bytes
	chunkIndex    int      // Index of the chunk available for writes
	overflows     int      // Count the buffer overflows, which correspond to data drops, for reporting metrics
	overflowBytes int64    // Count the bytes that were dropped, for reporting metrics
	mu            sync.Mutex
}

// getChunkIndexForLine returns the chunkIndex that should be used for storing the line, or -1, if there is
// an overflow and the line cannot be stored in the bufferShard. It reports
// whether space was reserved (true) or the data must be dropped (false); on
// success b.chunkIndex identifies the chunk to append into (advancing it if the
// line does not fit in the current chunk).
func (b *bufferShard) reserveChunkForLine(lineLen int) bool {
	totalWriteLength := lineLen

	// All chunks are full for the shard, drop the data
	if b.chunkIndex >= len(b.data) {
		b.overflows++
		b.overflowBytes += int64(totalWriteLength)
		return false
	}

	// This should not happen, drop the data. The maximum length of a well-formed protocol line is 3.8KB.
	if lineLen > chunkSize {
		b.overflows++
		b.overflowBytes += int64(totalWriteLength)
		return false
	}

	if len(b.data[b.chunkIndex]) > 0 {
		// Chunk has data, so account for the separator character
		totalWriteLength++
	}

	if len(b.data[b.chunkIndex])+totalWriteLength > chunkSize {
		// Line does not fit in the current chunk, go to the next chunk
		b.chunkIndex++
	}

	// Out of space in the shard, drop the data
	if b.chunkIndex == len(b.data) {
		b.overflows++
		b.overflowBytes += int64(totalWriteLength)
		return false
	}

	return true
}

type LowLatencyBuffer struct {
	writer Writer
	logger logger.Logger

	// Two sets of bufferShard, each scaled to the number of CPUs on the system. There are two sets,
	// so that one can be drained for writes to the spectatord socket, while the other can be filled
	// with writes from the application without contending for the same mutexes. They are swapped back
	// and forth during periodic flushes, according to the flushInterval.
	frontBuffers  []*bufferShard
	backBuffers   []*bufferShard
	bufferSetSize int
	flushInterval time.Duration

	// activeBufferState is incremented on every buffer swap. Even values mean the front buffers
	// are active for application writes, odd values mean the back buffers are active. Writers
	// verify the state after locking a shard so delayed writes cannot append to a buffer set
	// that has already been swapped out and flushed.
	activeBufferState atomic.Uint64

	// Counter used to round-robin shards for commutative meter types and malformed lines.
	counter uint64

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewLowLatencyBuffer(writer Writer, logger logger.Logger, bufferSize int, flushInterval time.Duration) *LowLatencyBuffer {
	numCPUs := runtime.NumCPU()
	frontBuffers := make([]*bufferShard, numCPUs)
	backBuffers := make([]*bufferShard, numCPUs)
	maxChunks := bufferSize / (2 * numCPUs * chunkSize)
	if maxChunks < 1 {
		maxChunks = 1
		bufferSize = maxChunks * 2 * numCPUs * chunkSize
	}

	logger.Infof("Initialize LowLatencyBuffer with size %d bytes (%d shards of %d chunks), and flushInterval of %.2f seconds", bufferSize, numCPUs, maxChunks, flushInterval.Seconds())

	for i := 0; i < numCPUs; i++ {
		frontBuffers[i] = &bufferShard{
			data:       make([][]byte, maxChunks),
			chunkIndex: 0,
			overflows:  0,
		}
		backBuffers[i] = &bufferShard{
			data:       make([][]byte, maxChunks),
			chunkIndex: 0,
			overflows:  0,
		}
		// Allocate buffer memory up-front
		for j := 0; j < maxChunks; j++ {
			frontBuffers[i].data[j] = make([]byte, 0, chunkSize)
			backBuffers[i].data[j] = make([]byte, 0, chunkSize)
		}
	}

	llb := &LowLatencyBuffer{
		writer:        writer,
		logger:        logger,
		frontBuffers:  frontBuffers,
		backBuffers:   backBuffers,
		bufferSetSize: bufferSize / 2,
		flushInterval: flushInterval,
		stopCh:        make(chan struct{}),
	}

	// Start the flush goroutine
	llb.wg.Add(1)
	go llb.flushLoop()

	return llb
}

func (llb *LowLatencyBuffer) Write(line string) {
	// A whole line shards on itself (the meter id lies between its first and last
	// ':'); appending it with an empty value reuses the shared string append path.
	llb.writeSegments(line, "")
}

// WriteLine appends prefix + value as one protocol line without concatenating
// them first. Sharding uses the prefix alone: the meter id (which determines
// the shard for order-sensitive types) lies between the first and last ':' of
// the line, both of which are contained in the prefix, so the value never
// affects shard selection.
func (llb *LowLatencyBuffer) WriteLine(prefix, value string) {
	llb.writeSegments(prefix, value)
}

// WriteInt appends prefix + the base-10 value with no allocation: the value is
// formatted into a stack buffer and copied straight into the shard chunk.
func (llb *LowLatencyBuffer) WriteInt(prefix string, value int64) {
	var tmp [intFmtBufLen]byte
	llb.writeValueSegments(prefix, strconv.AppendInt(tmp[:0], value, 10))
}

// WriteUint appends prefix + the base-10 value with no allocation.
func (llb *LowLatencyBuffer) WriteUint(prefix string, value uint64) {
	var tmp [intFmtBufLen]byte
	llb.writeValueSegments(prefix, strconv.AppendUint(tmp[:0], value, 10))
}

// WriteFloat appends prefix + the value formatted as 'f' with 6 decimals, no allocation.
func (llb *LowLatencyBuffer) WriteFloat(prefix string, value float64) {
	var tmp [floatFmtBufLen]byte
	llb.writeValueSegments(prefix, strconv.AppendFloat(tmp[:0], value, 'f', 6, 64))
}

// writeSegments appends prefix followed by value (both strings) as one protocol
// line, retrying if the active buffer set is swapped out mid-write. Sharding is
// by prefix (see WriteLine).
func (llb *LowLatencyBuffer) writeSegments(prefix, value string) {
	shardIndex := llb.shardIndexFor(prefix)
	for {
		state := llb.activeBufferState.Load()
		buffer := llb.bufferShardForState(state, shardIndex)
		if buffer.appendLine(&llb.activeBufferState, state, prefix, value) {
			return
		}
	}
}

// writeValueSegments is writeSegments for a value that arrives as bytes (a
// number formatted into a caller stack buffer), so numeric emission avoids the
// intermediate value-string allocation. value must not be retained past the
// call; it is copied into the shard chunk under lock.
func (llb *LowLatencyBuffer) writeValueSegments(prefix string, value []byte) {
	shardIndex := llb.shardIndexFor(prefix)
	for {
		state := llb.activeBufferState.Load()
		buffer := llb.bufferShardForState(state, shardIndex)
		if buffer.appendLineBytes(&llb.activeBufferState, state, prefix, value) {
			return
		}
	}
}

// appendLine and appendLineBytes append "prefix + value" into the shard's active
// chunk under lock, returning false to signal a retry when the buffer set was
// swapped after the shard was selected. They are identical except for the value
// type (string vs []byte); Go's append accepts both, but a single generic body
// cannot cover both core types without an allocation, so the small locked tail
// is written twice. Keep them in sync.
func (b *bufferShard) appendLine(state *atomic.Uint64, expected uint64, prefix, value string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if state.Load() != expected {
		return false
	}
	if !b.reserveChunkForLine(len(prefix) + len(value)) {
		// overflows (drops) are counted in reserveChunkForLine, for metric reporting
		return true
	}
	if len(b.data[b.chunkIndex]) > 0 {
		// chunk has data, so add the separator to end the previous line
		b.data[b.chunkIndex] = append(b.data[b.chunkIndex], separator...)
	}
	b.data[b.chunkIndex] = append(b.data[b.chunkIndex], prefix...)
	b.data[b.chunkIndex] = append(b.data[b.chunkIndex], value...)
	return true
}

func (b *bufferShard) appendLineBytes(state *atomic.Uint64, expected uint64, prefix string, value []byte) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if state.Load() != expected {
		return false
	}
	if !b.reserveChunkForLine(len(prefix) + len(value)) {
		return true
	}
	if len(b.data[b.chunkIndex]) > 0 {
		b.data[b.chunkIndex] = append(b.data[b.chunkIndex], separator...)
	}
	b.data[b.chunkIndex] = append(b.data[b.chunkIndex], prefix...)
	b.data[b.chunkIndex] = append(b.data[b.chunkIndex], value...)
	return true
}

func (llb *LowLatencyBuffer) bufferShardForState(state uint64, shardIndex int) *bufferShard {
	if frontBuffersActive(state) {
		return llb.frontBuffers[shardIndex]
	}
	return llb.backBuffers[shardIndex]
}

func frontBuffersActive(state uint64) bool {
	return state%2 == 0
}

// shardIndexFor returns the shard a protocol line should be written to. Lines for the same
// order-sensitive meter id always hash to the same shard, so sequential writes preserve order
// through flush. Other lines use the global counter for round-robin distribution.
func (llb *LowLatencyBuffer) shardIndexFor(line string) int {
	numShards := uint32(len(llb.frontBuffers))
	if key, ok := orderSensitiveMeterIdKey(line); ok {
		return int(fnv1aHash(key) % numShards)
	}
	return int(atomic.AddUint64(&llb.counter, 1) % uint64(numShards))
}

// orderSensitiveMeterIdKey returns the meter id portion of a spectatord protocol line for
// meter types where arrival order affects interpretation by spectatord. The protocol line
// format is "<type>:<id>:<value>" or "<type>,<extra>:<id>:<value>" (e.g. "g,120:..." for
// gauges with a TTL). The id is the substring between the first and last ':' in the line.
// Returns ok=false for commutative meter types and malformed lines.
func orderSensitiveMeterIdKey(line string) (string, bool) {
	first := strings.IndexByte(line, ':')
	if first <= 0 {
		return "", false
	}
	if !isOrderSensitiveMeterType(line[:first]) {
		return "", false
	}
	last := strings.LastIndexByte(line, ':')
	if last <= first {
		return "", false
	}
	return line[first+1 : last], true
}

// isOrderSensitiveMeterType identifies protocol types that carry absolute samples rather than
// commutative observations. Reordering these samples can change what spectatord reports.
func isOrderSensitiveMeterType(symbol string) bool {
	switch symbol {
	case "A", "C", "U", "g":
		return true
	default:
		return strings.HasPrefix(symbol, "g,")
	}
}

// fnv1aHash is an inline FNV-1a 32-bit hash over a string, written to avoid the allocation
// that hash/fnv incurs by exposing an io.Writer-style interface.
func fnv1aHash(s string) uint32 {
	const (
		offset32 uint32 = 2166136261
		prime32  uint32 = 16777619
	)
	h := offset32
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime32
	}
	return h
}

// flushLoop runs in a separate goroutine and handles buffer swapping and flushing
func (llb *LowLatencyBuffer) flushLoop() {
	defer llb.wg.Done()

	ticker := time.NewTicker(llb.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			llb.swapAndFlush()
		case <-llb.stopCh:
			// Final flush before shutdown
			llb.swapAndFlush()
			return
		}
	}
}

// swapAndFlush swaps the front and back buffers and flushes the deactivated buffers
func (llb *LowLatencyBuffer) swapAndFlush() {
	start := time.Now()

	// Swap the buffer sets, so one can be drained, while the other accepts application writes
	state := llb.activeBufferState.Add(1)

	var bufferSet string
	var buffersToFlush []*bufferShard
	if frontBuffersActive(state) {
		// Front buffers are now in use for application writes, so flush the back buffers
		bufferSet = "back"
		buffersToFlush = llb.backBuffers
	} else {
		// Back buffers are now in use application writes, so flush the front buffers
		bufferSet = "front"
		buffersToFlush = llb.frontBuffers
	}

	// Flush each buffer shard, from the deactivated set
	var bytesWritten int
	for _, buffer := range buffersToFlush {
		bytesWritten += llb.flushBufferShard(buffer, bufferSet)
	}

	pctUsage := float64(bytesWritten) / float64(llb.bufferSetSize)
	if bytesWritten > 0 {
		llb.writer.WriteString(fmt.Sprintf("c:spectator-go.lowLatencyBuffer.bytesWritten,bufferSet=%s:%d", bufferSet, bytesWritten))
	}
	if pctUsage > 0 {
		llb.writer.WriteString(fmt.Sprintf("g,1:spectator-go.lowLatencyBuffer.pctUsage,bufferSet=%s:%f", bufferSet, pctUsage))
	}

	llb.writer.WriteString(fmt.Sprintf("t:spectator-go.lowLatencyBuffer.flushTime,bufferSet=%s:%f", bufferSet, time.Since(start).Seconds()))
}

// flushBufferShard flushes a single bufferShard to the socket, iterating through all chunks
func (llb *LowLatencyBuffer) flushBufferShard(buffer *bufferShard, bufferSet string) int {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	// If there is no data to flush from the shard, then skip socket writes
	if buffer.chunkIndex == 0 && len(buffer.data[0]) == 0 {
		return 0
	}

	// Write each chunk to the socket, and reset the chunk
	var bytesWritten int
	for i := 0; i <= buffer.chunkIndex && i < len(buffer.data); i++ {
		if len(buffer.data[i]) > 0 {
			bytesWritten += len(buffer.data[i])
			llb.writer.WriteBytes(buffer.data[i])
			buffer.data[i] = buffer.data[i][:0]
		}
	}

	// record status metrics and reset shard statistics
	if buffer.overflows > 0 {
		llb.writer.WriteString(fmt.Sprintf("c:spectator-go.lowLatencyBuffer.overflows,bufferSet=%s:%d", bufferSet, buffer.overflows))
		llb.writer.WriteString(fmt.Sprintf("d:spectator-go.lowLatencyBuffer.overflowBytes,bufferSet=%s:%d", bufferSet, buffer.overflowBytes))
		buffer.overflows = 0
		buffer.overflowBytes = 0
	}
	buffer.chunkIndex = 0
	return bytesWritten
}

func (llb *LowLatencyBuffer) Close() {
	// Signal the flush goroutine to stop
	close(llb.stopCh)

	// Wait for the goroutine to finish
	llb.wg.Wait()
}
