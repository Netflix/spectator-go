package meter

import (
	"runtime"
	"testing"

	"github.com/Netflix/spectator-go/v2/spectator/writer"
)

// BenchmarkCounterAddCombined measures a combined meter creation and write path:
// NewId (tag copy + toSpectatorIdFromPairs) → NewCounter → Add (writeLineInt + writer.Write).
func BenchmarkCounterAddCombined(b *testing.B) {
	w := &writer.NoopWriter{}
	tags := map[string]string{
		"nf.app":     "meshcontrol-xds",
		"nf.cluster": "meshcontrol-xds-main",
		"nf.asg":     "meshcontrol-xds-main-v042",
		"nf.region":  "us-east-1",
		"id":         "envoy.cluster.upstream_cx_active",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := NewCounter(NewId("spectator.meter.count", tags), w)
		c.Add(1)
	}
}

// BenchmarkCounterAddCombinedPostGC measures the combined path with periodic GC to validate
// pool repopulation cost. With a single shared byteBufPool, only one buffer
// allocation is needed per P after GC (vs two with separate pools).
func BenchmarkCounterAddCombinedPostGC(b *testing.B) {
	w := &writer.NoopWriter{}
	tags := map[string]string{
		"nf.app":     "meshcontrol-xds",
		"nf.cluster": "meshcontrol-xds-main",
		"nf.asg":     "meshcontrol-xds-main-v042",
		"nf.region":  "us-east-1",
		"id":         "envoy.cluster.upstream_cx_active",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%1024 == 0 {
			runtime.GC()
		}
		c := NewCounter(NewId("spectator.meter.count", tags), w)
		c.Add(1)
	}
}
