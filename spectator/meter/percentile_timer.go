package meter

import (
	"strconv"

	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"time"
)

// PercentileTimer represents timing events, while capturing the histogram
// (percentiles) of those values.
type PercentileTimer struct {
	id              *Id
	writer          writer.Writer
	meterTypeSymbol string
}

func NewPercentileTimer(
	id *Id,
	writer writer.Writer,
) *PercentileTimer {
	return &PercentileTimer{id, writer, "T"}
}

func (t *PercentileTimer) MeterId() *Id {
	return t.id
}

// Record records the value for a single event.
func (t *PercentileTimer) Record(amount time.Duration) {
	if amount >= 0 {
		var line = t.meterTypeSymbol + ":" + t.id.spectatordId + ":" + strconv.FormatFloat(amount.Seconds(), 'f', 6, 64)
		t.writer.Write(line)
	}
}
