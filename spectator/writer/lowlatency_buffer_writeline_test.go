package writer

import (
	"runtime"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/Netflix/spectator-go/v2/spectator/logger"
)

// TestLowLatencyBuffer_WriteLineMatchesWrite verifies that WriteLine(prefix, value)
// flushes byte-identical output to Write(prefix + value), across commutative and
// order-sensitive meter types (which take different shard-selection paths).
func TestLowLatencyBuffer_WriteLineMatchesWrite(t *testing.T) {
	cases := []struct{ prefix, value string }{
		{"c:hot.counter:", "1"},                 // commutative counter
		{"c:hot.counter,app=foo,zone=z:", "42"}, // counter with tags
		{"g:hot.gauge:", "3.14"},                // order-sensitive gauge
		{"g,120:ttl.gauge:", "7"},               // order-sensitive gauge with TTL
		{"C:mono.counter:", "5"},                // order-sensitive monotonic counter
		{"A:age.gauge:", "0"},                   // order-sensitive age gauge
	}

	shards := runtime.NumCPU()
	bufferSize := 2 * 2 * chunkSize * shards

	collect := func(write func(b *LowLatencyBuffer)) []string {
		mem := &MemoryWriter{}
		buf := NewLowLatencyBuffer(mem, logger.NewDefaultLogger(), bufferSize, 5*time.Millisecond)
		write(buf)
		buf.Close() // flushes remaining data
		lines := splitAndFilterMetricLines(mem)
		slices.Sort(lines)
		return lines
	}

	viaWrite := collect(func(b *LowLatencyBuffer) {
		for _, c := range cases {
			b.Write(c.prefix + c.value)
		}
	})
	viaWriteLine := collect(func(b *LowLatencyBuffer) {
		for _, c := range cases {
			b.WriteLine(c.prefix, c.value)
		}
	})

	if !slices.Equal(viaWrite, viaWriteLine) {
		t.Fatalf("WriteLine output differs from Write output:\n Write:     %q\n WriteLine: %q", viaWrite, viaWriteLine)
	}
}

// TestLowLatencyBuffer_TypedWritesMatchWrite verifies that WriteInt / WriteUint /
// WriteFloat produce output identical to the equivalent Write(prefix + strconv.Format...).
func TestLowLatencyBuffer_TypedWritesMatchWrite(t *testing.T) {
	shards := runtime.NumCPU()
	bufferSize := 2 * 2 * chunkSize * shards

	collect := func(write func(b *LowLatencyBuffer)) []string {
		mem := &MemoryWriter{}
		buf := NewLowLatencyBuffer(mem, logger.NewDefaultLogger(), bufferSize, 5*time.Millisecond)
		write(buf)
		buf.Close()
		lines := splitAndFilterMetricLines(mem)
		slices.Sort(lines)
		return lines
	}

	viaWrite := collect(func(b *LowLatencyBuffer) {
		b.Write("c:my.counter:" + strconv.FormatInt(42, 10))
		b.Write("c:my.counter:" + strconv.FormatInt(-7, 10))
		b.Write("U:mono.uint:" + strconv.FormatUint(18446744073709551615, 10))
		b.Write("g:my.gauge:" + strconv.FormatFloat(3.14159, 'f', 6, 64))
		b.Write("t:my.timer:" + strconv.FormatFloat(0.001234, 'f', 6, 64))
	})
	viaTyped := collect(func(b *LowLatencyBuffer) {
		b.WriteInt("c:my.counter:", 42)
		b.WriteInt("c:my.counter:", -7)
		b.WriteUint("U:mono.uint:", 18446744073709551615)
		b.WriteFloat("g:my.gauge:", 3.14159)
		b.WriteFloat("t:my.timer:", 0.001234)
	})

	if !slices.Equal(viaWrite, viaTyped) {
		t.Fatalf("typed writes differ from Write output:\n Write: %q\n Typed: %q", viaWrite, viaTyped)
	}
}
