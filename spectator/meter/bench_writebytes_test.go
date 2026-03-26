package meter

import (
	"strconv"
	"testing"

	"github.com/Netflix/spectator-go/v2/spectator/writer"
)

var (
	benchStringSink string
	benchBytesSink  []byte
)

type stringOnlyWriter struct{}

func (w *stringOnlyWriter) Write(line string)      { benchStringSink = line }
func (w *stringOnlyWriter) WriteBytes(line []byte) { benchBytesSink = line }
func (w *stringOnlyWriter) WriteString(line string) {
	benchStringSink = line
}
func (w *stringOnlyWriter) Close() error { return nil }

type socketLikeWriter struct{}

func (w *socketLikeWriter) Write(line string) {
	benchBytesSink = []byte(line)
}
func (w *socketLikeWriter) WriteBytes(line []byte) { benchBytesSink = line }
func (w *socketLikeWriter) WriteString(line string) {
	benchBytesSink = []byte(line)
}
func (w *socketLikeWriter) Close() error { return nil }

func writeLineIntViaWrite(w writer.Writer, symbol string, spectatordId string, val int64) {
	bp := byteBufPool.Get().(*[]byte)
	b := appendLinePrefix((*bp)[:0], symbol, spectatordId)
	b = strconv.AppendInt(b, val, 10)
	w.Write(string(b))
	*bp = b
	byteBufPool.Put(bp)
}

func benchmarkWriteLineIntCurrent(b *testing.B, w writer.Writer) {
	const symbol = "c"
	const spectatordId = "spectator.meter.count,nf.app=meshcontrol-xds,nf.cluster=meshcontrol-xds-main,nf.asg=meshcontrol-xds-main-v042,nf.region=us-east-1,id=envoy.cluster.upstream_cx_active"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writeLineIntViaWrite(w, symbol, spectatordId, 1)
	}
}

func benchmarkWriteLineIntBytes(b *testing.B, w writer.Writer) {
	const symbol = "c"
	const spectatordId = "spectator.meter.count,nf.app=meshcontrol-xds,nf.cluster=meshcontrol-xds-main,nf.asg=meshcontrol-xds-main-v042,nf.region=us-east-1,id=envoy.cluster.upstream_cx_active"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writeLineInt(w, symbol, spectatordId, 1)
	}
}

func BenchmarkWriteLineIntCurrentStringOnlyWriter(b *testing.B) {
	benchmarkWriteLineIntCurrent(b, &stringOnlyWriter{})
}

func BenchmarkWriteLineIntBytesStringOnlyWriter(b *testing.B) {
	benchmarkWriteLineIntBytes(b, &stringOnlyWriter{})
}

func BenchmarkWriteLineIntCurrentSocketLikeWriter(b *testing.B) {
	benchmarkWriteLineIntCurrent(b, &socketLikeWriter{})
}

func BenchmarkWriteLineIntBytesSocketLikeWriter(b *testing.B) {
	benchmarkWriteLineIntBytes(b, &socketLikeWriter{})
}
