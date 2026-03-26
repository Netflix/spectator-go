package writer

import (
	"testing"
	"time"

	"github.com/Netflix/spectator-go/v2/spectator/logger"
)

func benchmarkLineBufferWriteString(b *testing.B, line string) {
	buf := NewLineBuffer(&NoopWriter{}, logger.NewDefaultLogger(), 1<<30, time.Hour)
	defer buf.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Write(line)
	}
}

func benchmarkLineBufferWriteBytes(b *testing.B, line []byte) {
	buf := NewLineBuffer(&NoopWriter{}, logger.NewDefaultLogger(), 1<<30, time.Hour)
	defer buf.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.WriteBytes(line)
	}
}

func BenchmarkLineBufferWriteString(b *testing.B) {
	benchmarkLineBufferWriteString(b, "spectator.meter.count,nf.app=meshcontrol-xds:1")
}

func BenchmarkLineBufferWriteBytes(b *testing.B) {
	benchmarkLineBufferWriteBytes(b, []byte("spectator.meter.count,nf.app=meshcontrol-xds:1"))
}
