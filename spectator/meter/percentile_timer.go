package meter

import (
	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"time"
)

// PercentileTimer represents timing events, while capturing the histogram
// (percentiles) of those values.
type PercentileTimer struct {
	id         *Id
	writer     writer.Writer
	linePrefix string
}

func NewPercentileTimer(
	id *Id,
	writer writer.Writer,
) *PercentileTimer {
	return &PercentileTimer{id, writer, "T:" + id.spectatordId + ":"}
}

// NewPercentileTimerDirect generates a new percentile timer directly from a name
// and tags, without allocating an *Id or copying the tags map. commonTags
// carries the registry's extraCommonTags.
func NewPercentileTimerDirect(name string, tags, commonTags map[string]string, writer writer.Writer) *PercentileTimer {
	return &PercentileTimer{nil, writer, buildLinePrefix("T", name, tags, commonTags)}
}

func (t *PercentileTimer) MeterId() *Id {
	return resolveMeterId(t.id, t.linePrefix)
}

// Record records the value for a single event.
func (t *PercentileTimer) Record(amount time.Duration) {
	if amount >= 0 {
		t.writer.WriteFloat(t.linePrefix, amount.Seconds())
	}
}
