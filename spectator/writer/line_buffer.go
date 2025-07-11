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

	if lb.buffer.Len() > 0 {
		// buffer has data, so add the separator to indicate the end of the previous line
		lb.buffer.WriteString(separator)
	}

	lb.buffer.WriteString(line)
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
