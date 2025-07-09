package writer

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// chunkSize is set to 60KB, to ensure each message fits in the socket buffer (64KB), with some room
// to accommodate the last spectatord protocol line appended. The maximum length of a well-formed
// protocol line is 3,927 characters (3.8KB).
const chunkSize = 60 * 1024

// separator is the character used to indicate the end of a spectatord protocol line, when combining
// lines into a larger socket payload
const separator = "\n"

// bufferShard is the atomic unit of buffering, used to store one or more chunks of spectatord
// protocol lines. Each shard is accessed in round-robin format within either the front buffer
// or the back buffer, and the number of shards is scaled to the number of CPUs on the system.
// The purpose of this design is to spread buffer access across a reasonable number of mutexes,
// in order to reduce overall latency when writing to the buffer. The impact of the shard design
// is less than the impact of the front and back buffer design, but it is still important for
// throughput reasons.
type bufferShard struct {
	data       [][]byte // Array of chunkSize chunks of spectatord protocol lines, stored as bytes
	chunkIndex int      // Index of the chunk available for writes
	overflows  int      // Count the buffer overflows, which correspond to data drops, for reporting metrics
	mu         sync.Mutex
}

// getChunkIndexForLine returns the chunkIndex that should be used for storing the line, or -1, if there is
// an overflow and the line cannot be stored in the bufferShard.
func (b *bufferShard) getChunkIndexForLine(line []byte) int {
	// All chunks are full for the shard, drop the data
	if b.chunkIndex >= len(b.data) {
		b.overflows++
		return -1
	}

	// This should not happen, drop the data. The maximum length of a well-formed protocol line is 3.8KB.
	if len(line) > chunkSize {
		b.overflows++
		return -1
	}

	totalWriteLength := len(line)
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
		return -1
	}

	return b.chunkIndex
}

type LowLatencyBuffer struct {
	writer Writer
	logger logger.Logger

	// Two sets of bufferShard, each scaled to the number of CPUs on the system. There are two sets,
	// so that one can be drained for writes to the spectatord socket, while the other can be filled
	// with writes from the application without contending for the same mutexes. They are swapped back
	// and forth during periodic flushes, according to the flushInterval.
	frontBuffers    []*bufferShard
	backBuffers     []*bufferShard
	bufferSetSize   int
	useFrontBuffers atomic.Bool
	flushInterval   time.Duration

	// Distribute writes across the shards in the active buffer, with a round-robin scheme.
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

	llb.useFrontBuffers.Store(true)

	// Start the flush goroutine
	llb.wg.Add(1)
	go llb.flushLoop()

	return llb
}

func (llb *LowLatencyBuffer) Write(line string) {
	// Pick a shard index across all shards in the active buffer, with a round-robin distribution
	shardIndex := int(atomic.AddUint64(&llb.counter, 1)) % len(llb.frontBuffers)

	// Acquire read lock, to check which buffers are active
	var buffer *bufferShard
	if llb.useFrontBuffers.Load() {
		buffer = llb.frontBuffers[shardIndex]
	} else {
		buffer = llb.backBuffers[shardIndex]
	}

	// Add the line to the appropriate chunk in the buffer shard, or drop, if it overflows
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	lineBytes := []byte(line)

	// Check if current chunk can fit the new data
	idx := buffer.getChunkIndexForLine(lineBytes)
	if idx == -1 {
		// overflows (drops) are counted in getChunkIndexForLine, for metric reporting
		return
	}

	// We can write to the selected chunk
	if len(buffer.data[buffer.chunkIndex]) > 0 {
		// buffer has data, so add the separator, to indicate the end of the previous line
		buffer.data[buffer.chunkIndex] = append(buffer.data[buffer.chunkIndex], []byte(separator)...)
	}
	buffer.data[buffer.chunkIndex] = append(buffer.data[buffer.chunkIndex], lineBytes...)
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
	// Swap the buffer sets, so one can be drained, while the other accepts application writes
	old := llb.useFrontBuffers.Load()
	llb.useFrontBuffers.CompareAndSwap(old, !old)

	var bufferSet string
	var buffersToFlush []*bufferShard
	if llb.useFrontBuffers.Load() {
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
		buffer.overflows = 0
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
