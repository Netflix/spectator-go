package writer

import (
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"strconv"
	"strings"
	"sync"
	"time"
)

type LineBuffer struct {
	writer Writer
	logger logger.Logger

	bufferSize    int
	buffer        strings.Builder
	lineCount     int
	flushInterval time.Duration
	lastFlush     time.Time
	flushTimer    *time.Timer

	mu sync.Mutex
}

func NewLineBuffer(writer Writer, logger logger.Logger, bufferSize int, flushInterval time.Duration) *LineBuffer {
	logger.Infof("Initialize LineBuffer with size %d bytes, and flushInterval of %.2f seconds", bufferSize, flushInterval.Seconds())

	lb := &LineBuffer{
		writer:        writer,
		logger:        logger,
		bufferSize:    bufferSize,
		lineCount:     0,
		flushInterval: flushInterval,
		lastFlush:     time.Now(),
	}

	lb.startFlushTimer()

	return lb
}

func (lb *LineBuffer) Write(line string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.beginLineLocked()
	lb.buffer.WriteString(line)
	lb.endLineLocked()
}

func (lb *LineBuffer) WriteLine(prefix, value string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.beginLineLocked()
	lb.buffer.WriteString(prefix)
	lb.buffer.WriteString(value)
	lb.endLineLocked()
}

// WriteInt appends prefix + the base-10 value, formatting the value into a stack
// buffer to avoid a value-string allocation. Formatting is done before taking
// the lock so the mutex only covers the buffer appends.
func (lb *LineBuffer) WriteInt(prefix string, value int64) {
	var tmp [intFmtBufLen]byte
	v := strconv.AppendInt(tmp[:0], value, 10)

	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.beginLineLocked()
	lb.buffer.WriteString(prefix)
	lb.buffer.Write(v)
	lb.endLineLocked()
}

// WriteUint appends prefix + the base-10 value.
func (lb *LineBuffer) WriteUint(prefix string, value uint64) {
	var tmp [intFmtBufLen]byte
	v := strconv.AppendUint(tmp[:0], value, 10)

	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.beginLineLocked()
	lb.buffer.WriteString(prefix)
	lb.buffer.Write(v)
	lb.endLineLocked()
}

// WriteFloat appends prefix + the value formatted as 'f' with 6 decimals.
func (lb *LineBuffer) WriteFloat(prefix string, value float64) {
	var tmp [floatFmtBufLen]byte
	v := strconv.AppendFloat(tmp[:0], value, 'f', 6, 64)

	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.beginLineLocked()
	lb.buffer.WriteString(prefix)
	lb.buffer.Write(v)
	lb.endLineLocked()
}

// beginLineLocked writes the line separator if the buffer already holds data.
// Callers must hold lb.mu.
func (lb *LineBuffer) beginLineLocked() {
	if lb.buffer.Len() > 0 {
		// buffer has data, so add the separator to indicate the end of the previous line
		lb.buffer.WriteString(separator)
	}
}

// endLineLocked accounts for the written line and flushes on overflow.
// Callers must hold lb.mu.
func (lb *LineBuffer) endLineLocked() {
	lb.lineCount++

	if lb.buffer.Len() >= lb.bufferSize {
		lb.writer.WriteString("c:spectator-go.lineBuffer.overflows:1")
		lb.flush()
	}
}

func (lb *LineBuffer) startFlushTimer() {
	lb.flushTimer = time.AfterFunc(lb.flushInterval, lb.flushLocked)
}

func (lb *LineBuffer) flushLocked() {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if time.Since(lb.lastFlush) >= lb.flushInterval {
		lb.flush()
	}

	lb.startFlushTimer()
}

func (lb *LineBuffer) flush() {
	// If there is no data to flush from the buffer, then skip socket writes
	if lb.buffer.Len() == 0 {
		return
	}

	lb.logger.Debugf("Flushing buffer with %d lines (%d bytes)", lb.lineCount, lb.buffer.Len())
	lb.writer.WriteString(lb.buffer.String())
	lb.writer.WriteString("c:spectator-go.lineBuffer.bytesWritten:" + strconv.Itoa(lb.buffer.Len()))
	lb.buffer.Reset()
	lb.lineCount = 0
	lb.lastFlush = time.Now()
}

func (lb *LineBuffer) Close() {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if lb.flushTimer != nil {
		lb.flushTimer.Stop()
	}

	lb.flush()
}
