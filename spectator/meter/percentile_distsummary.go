package meter

import (
	"github.com/Netflix/spectator-go/v2/spectator/writer"
)

// PercentileDistributionSummary is a distribution summary used to track the
// distribution of events, while also presenting the results as percentiles.
type PercentileDistributionSummary struct {
	id         *Id
	writer     writer.Writer
	linePrefix string
}

func (p *PercentileDistributionSummary) MeterId() *Id {
	return resolveMeterId(p.id, p.linePrefix)
}

// NewPercentileDistributionSummary creates a new *PercentileDistributionSummary using the meter identifier.
func NewPercentileDistributionSummary(id *Id, writer writer.Writer) *PercentileDistributionSummary {
	return &PercentileDistributionSummary{id, writer, "D:" + id.spectatordId + ":"}
}

// NewPercentileDistributionSummaryDirect creates a new *PercentileDistributionSummary
// directly from a name and tags, without allocating an *Id or copying the tags map.
// commonTags carries the registry's extraCommonTags.
func NewPercentileDistributionSummaryDirect(name string, tags, commonTags map[string]string, writer writer.Writer) *PercentileDistributionSummary {
	return &PercentileDistributionSummary{nil, writer, buildLinePrefix("D", name, tags, commonTags)}
}

// Record records an amount to track within the distribution.
func (p *PercentileDistributionSummary) Record(amount int64) {
	if amount >= 0 {
		p.writer.WriteInt(p.linePrefix, amount)
	}
}
