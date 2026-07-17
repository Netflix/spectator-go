package meter

// End-to-end emit benchmarks: meter -> real LowLatencyBuffer (backed by a
// NoopWriter so we exercise the buffer append path without socket I/O).
// Uses only the public Counter API, so the loop body is identical before and
// after the WriteLine interface extension.

import (
	"runtime"
	"testing"
	"time"

	"github.com/Netflix/spectator-go/v2/spectator/writer"
)

type emitQuietLogger struct{}

func (emitQuietLogger) Debugf(string, ...interface{}) {}
func (emitQuietLogger) Infof(string, ...interface{})  {}
func (emitQuietLogger) Errorf(string, ...interface{}) {}

// bufWriter adapts a LowLatencyBuffer to the Writer interface (its Close has no
// error return).
type bufWriter struct{ b *writer.LowLatencyBuffer }

func (w *bufWriter) Write(line string)                   { w.b.Write(line) }
func (w *bufWriter) WriteLine(prefix, value string)      { w.b.WriteLine(prefix, value) }
func (w *bufWriter) WriteInt(prefix string, v int64)     { w.b.WriteInt(prefix, v) }
func (w *bufWriter) WriteUint(prefix string, v uint64)   { w.b.WriteUint(prefix, v) }
func (w *bufWriter) WriteFloat(prefix string, v float64) { w.b.WriteFloat(prefix, v) }
func (w *bufWriter) WriteBytes(line []byte)              { w.b.Write(string(line)) }
func (w *bufWriter) WriteString(line string)             { w.b.Write(line) }
func (w *bufWriter) Close() error                        { w.b.Close(); return nil }

func newBenchBuffer(b *testing.B) *bufWriter {
	b.Helper()
	shards := runtime.NumCPU()
	bufferSize := 2 * 16 * 60 * 1024 * shards
	buf := writer.NewLowLatencyBuffer(&writer.NoopWriter{}, emitQuietLogger{}, bufferSize, time.Millisecond)
	w := &bufWriter{buf}
	b.Cleanup(func() { w.Close() })
	return w
}

var emitBenchTags = map[string]string{
	"app":    "gandalfAgent",
	"result": "allowed",
	"policy": "explicit",
}

// Cached counter (constructed once), Increment in the loop — isolates the emit path.
func BenchmarkEmitBuffer_Counter_Increment(b *testing.B) {
	c := NewCounter(NewId("gandalfAgent.authzEngine.appAuthzResult", emitBenchTags), newBenchBuffer(b))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Increment()
	}
}

// Cached counter, Add(delta) in the loop — value requires formatting.
func BenchmarkEmitBuffer_Counter_Add(b *testing.B) {
	c := NewCounter(NewId("gandalfAgent.authzEngine.appAuthzResult", emitBenchTags), newBenchBuffer(b))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Add(int64(i) + 1)
	}
}

// Cached gauge, Set(float) in the loop.
func BenchmarkEmitBuffer_Gauge_Set(b *testing.B) {
	g := NewGauge(NewId("gandalfAgent.gauge", emitBenchTags), newBenchBuffer(b))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Set(3.14159)
	}
}

// Cached timer, Record in the loop.
func BenchmarkEmitBuffer_Timer_Record(b *testing.B) {
	t := NewTimer(NewId("gandalfAgent.timer", emitBenchTags), newBenchBuffer(b))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t.Record(time.Duration(i) * time.Millisecond)
	}
}

// Cached distribution summary, Record in the loop.
func BenchmarkEmitBuffer_DistSummary_Record(b *testing.B) {
	d := NewDistributionSummary(NewId("gandalfAgent.distsummary", emitBenchTags), newBenchBuffer(b))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Record(int64(i))
	}
}

// Construct-per-call (gandalf's actual pattern) over a real buffer, via NewId
// (the WithId path) for comparison.
func BenchmarkEmitBuffer_Counter_ConstructPerCall(b *testing.B) {
	w := newBenchBuffer(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewCounter(NewId("gandalfAgent.authzEngine.appAuthzResult", emitBenchTags), w).Increment()
	}
}

// Construct-per-call via the direct (name, tags) path — what registry.Counter
// now uses. No Id, no tag-map copy.
func BenchmarkEmitBuffer_Counter_ConstructPerCall_Direct(b *testing.B) {
	w := newBenchBuffer(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewCounterDirect("gandalfAgent.authzEngine.appAuthzResult", emitBenchTags, nil, w).Increment()
	}
}
