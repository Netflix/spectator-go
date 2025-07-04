package writer

import (
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"strings"
	"sync"
	"time"
)

// 60KB, to account for 64KB socket buffer size
const chunkSize = 60 * 1024

type LineBuffer struct {
	writer       Writer
	bufferSize   int
	flushTimeout time.Duration
	logger       logger.Logger

	mu          sync.Mutex
	builder     strings.Builder
	buffers     []string
	currentSize int
	lineCount   int
	lastFlush   time.Time
	flushTimer  *time.Timer
	closed      bool
}

func NewLineBuffer(writer Writer, bufferSize int, logger logger.Logger, timeoutOptional ...time.Duration) *LineBuffer {
	timeout := 5 * time.Second
	if len(timeoutOptional) > 0 {
		timeout = timeoutOptional[0]
	}

	lb := &LineBuffer{
		writer:       writer,
		bufferSize:   bufferSize,
		currentSize:  0,
		flushTimeout: timeout,
		logger:       logger,
		lastFlush:    time.Now(),
	}

	lb.startFlushTimer()

	return lb
}

func (lb *LineBuffer) Write(line string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if lb.closed {
		return
	}

	if lb.builder.Len() > 0 {
		lb.builder.WriteString("\n")
	}
	lb.builder.WriteString(line)
	lb.lineCount++

	if lb.bufferSize <= chunkSize {
		if lb.builder.Len() >= lb.bufferSize {
			lb.flushBuilderLocked()
		}
		return
	}

	if lb.builder.Len() >= chunkSize {
		lb.buffers = append(lb.buffers, lb.builder.String())
		lb.currentSize += lb.builder.Len()
		lb.builder.Reset()
	}

	if lb.currentSize + lb.builder.Len() >= lb.bufferSize {
		lb.flushLocked()
	}
}

func (lb *LineBuffer) Close() error {
	if lb.bufferSize <= 0 {
		return lb.writer.Close()
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	if lb.closed {
		return nil
	}

	lb.closed = true

	if lb.flushTimer != nil {
		lb.flushTimer.Stop()
	}

	if lb.bufferSize <= chunkSize {
		lb.flushBuilderLocked()
	} else {
		lb.flushLocked()
	}

	return lb.writer.Close()
}

func (lb *LineBuffer) startFlushTimer() {
	lb.flushTimer = time.AfterFunc(lb.flushTimeout, lb.timerFlush)
}

func (lb *LineBuffer) timerFlush() {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if lb.closed {
		return
	}

	if time.Since(lb.lastFlush) >= lb.flushTimeout {
		if lb.bufferSize <= chunkSize {
			lb.flushBuilderLocked()
		} else {
			lb.flushLocked()
		}
	}

	lb.startFlushTimer()
}

func (lb *LineBuffer) flushLocked() {
	if lb.builder.Len() > 0 {
		lb.buffers = append(lb.buffers, lb.builder.String())
		lb.currentSize += lb.builder.Len()
		lb.builder.Reset()
	}

	if lb.currentSize == 0 {
		return
	}

	lb.logger.Debugf("Flushing buffer with %d lines (%d bytes)", lb.lineCount, lb.currentSize)
	for _, line := range lb.buffers {
		lb.writer.Write(line)
	}
	lb.buffers = nil
	lb.lineCount = 0
	lb.currentSize = 0
	lb.lastFlush = time.Now()
}

func (lb *LineBuffer) flushBuilderLocked() {
	if lb.builder.Len() == 0 {
		return
	}

	lb.logger.Debugf("Flushing buffer with %d lines (%d bytes)", lb.lineCount, lb.builder.Len())
	lb.writer.Write(lb.builder.String())
	lb.builder.Reset()
	lb.lineCount = 0
	lb.lastFlush = time.Now()
}
