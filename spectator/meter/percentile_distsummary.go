package meter

import (
	"fmt"
	"github.com/Netflix/spectator-go/spectator/writer"
)

// PercentileDistributionSummary is a distribution summary used to track the
// distribution of events, while also presenting the results as percentiles.
type PercentileDistributionSummary struct {
	id              *Id
	writer          writer.Writer
	meterTypeSymbol rune
}

func (p *PercentileDistributionSummary) MeterId() *Id {
	return p.id
}

// NewPercentileDistributionSummary creates a new *PercentileDistributionSummary using the meter identifier.
func NewPercentileDistributionSummary(id *Id, writer writer.Writer) *PercentileDistributionSummary {
	return &PercentileDistributionSummary{id, writer, 'D'}
}

// Record records a new value to track within the distribution.
func (p *PercentileDistributionSummary) Record(amount int64) {
	if amount >= 0 {
		var line = fmt.Sprintf("%c:%s:%d", p.meterTypeSymbol, p.id.spectatordId, amount)
		p.writer.Write(line)
	}
}
