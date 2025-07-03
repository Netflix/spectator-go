package writer

import (
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"strings"
	"sync"
	"time"
)

type LineBuffer struct {
	writer       Writer
	bufferSize   int
	flushTimeout time.Duration
	logger       logger.Logger
	
	mu         sync.Mutex
	buffer     strings.Builder
	lineCount  int
	lastFlush  time.Time
	flushTimer *time.Timer
	closed     bool
}

func NewLineBuffer(writer Writer, bufferSize int, logger logger.Logger, timeoutOptional ...time.Duration) *LineBuffer {
	timeout := 5 * time.Second
	if len(timeoutOptional) > 0 {
		timeout = timeoutOptional[0]
	}

	lb := &LineBuffer{
		writer:       writer,
		bufferSize:   bufferSize,
		flushTimeout: timeout,
		logger:       logger,
		lastFlush:    time.Now(),
	}

	lb.startFlushTimer()

	return lb
}

func (lb *LineBuffer) Write(line string) {
	if lb.bufferSize <= 0 {
		lb.writer.Write(line)
		return
	}
	
	lb.mu.Lock()
	defer lb.mu.Unlock()
	
	if lb.closed {
		return
	}
	
	if lb.buffer.Len() > 0 {
		lb.buffer.WriteString("\n")
	}
	lb.buffer.WriteString(line)
	lb.lineCount++
	
	if lb.buffer.Len() >= lb.bufferSize {
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
	
	lb.flushLocked()
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
		lb.flushLocked()
	}
	
	lb.startFlushTimer()
}

func (lb *LineBuffer) flushLocked() {
	if lb.buffer.Len() == 0 {
		return
	}
	
	lb.logger.Debugf("Flushing buffer with %d lines (%d bytes)", lb.lineCount, lb.buffer.Len())
	lb.writer.Write(lb.buffer.String())
	lb.buffer.Reset()
	lb.lineCount = 0
	lb.lastFlush = time.Now()
}
