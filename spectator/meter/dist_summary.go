package meter

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/writer"
)

// DistributionSummary is used to track the distribution of events. This is safe
// for concurrent use.
//
// You can find more about this type by viewing the relevant Java Spectator
// documentation here:
//
// https://netflix.github.io/spectator/en/latest/intro/dist-summary/
type DistributionSummary struct {
	id              *Id
	writer          writer.Writer
	meterTypeSymbol string
}

// NewDistributionSummary generates a new distribution summary, using the
// provided meter identifier.
func NewDistributionSummary(id *Id, writer writer.Writer) *DistributionSummary {
	return &DistributionSummary{id, writer, "d"}
}

// MeterId returns the meter identifier.
func (d *DistributionSummary) MeterId() *Id {
	return d.id
}

// Record records a new value to track within the distribution.
func (d *DistributionSummary) Record(amount int64) {
	if amount >= 0 {
		var line = fmt.Sprintf("%s:%s:%d", d.meterTypeSymbol, d.id.spectatordId, amount)
		d.writer.Write(line)
	}
}
