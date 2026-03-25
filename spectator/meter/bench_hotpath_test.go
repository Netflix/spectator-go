package meter

import (
	"testing"

	"github.com/Netflix/spectator-go/v2/spectator/writer"
)

// BenchmarkCounterAddHotPath measures the full hot path:
// NewId (tag copy + toSpectatorId) → NewCounter → Add (fmt.Sprintf + writer.Write).
// This is the dominant allocation pattern seen in production heap profiles.
func BenchmarkCounterAddHotPath(b *testing.B) {
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
