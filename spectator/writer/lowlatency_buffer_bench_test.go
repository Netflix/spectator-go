package writer

import (
	"runtime"
	"strconv"
	"testing"
	"time"
)

type discardLogger struct{}

func (discardLogger) Debugf(_ string, _ ...interface{}) {}
func (discardLogger) Infof(_ string, _ ...interface{})  {}
func (discardLogger) Errorf(_ string, _ ...interface{}) {}

func newBenchmarkLowLatencyBuffer(b *testing.B) *LowLatencyBuffer {
	b.Helper()

	shards := runtime.NumCPU()
	bufferSize := 2 * 16 * chunkSize * shards
	buffer := NewLowLatencyBuffer(&NoopWriter{}, discardLogger{}, bufferSize, time.Millisecond)
	b.Cleanup(buffer.Close)
	return buffer
}

func benchmarkProtocolLines(prefix string, count int) []string {
	lines := make([]string, count)
	for i := range lines {
		lines[i] = prefix + strconv.Itoa(i) + ":1"
	}
	return lines
}

func BenchmarkLowLatencyBuffer_WriteParallel(b *testing.B) {
	manyCounters := benchmarkProtocolLines("c:many.counter.", 1024)
	manyGauges := benchmarkProtocolLines("g:many.gauge.", 1024)

	tests := []struct {
		name  string
		lines []string
	}{
		{name: "hot_counter_same_id", lines: []string{"c:hot.counter:1"}},
		{name: "hot_gauge_same_id", lines: []string{"g:hot.gauge:1"}},
		{name: "many_counters", lines: manyCounters},
		{name: "many_gauges", lines: manyGauges},
	}

	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			buffer := newBenchmarkLowLatencyBuffer(b)
			mask := len(test.lines) - 1

			b.ReportAllocs()
			b.SetBytes(int64(len(test.lines[0])))
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					buffer.Write(test.lines[i&mask])
					i++
				}
			})
		})
	}
}
