package meter

import (
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
	id         *Id
	writer     writer.Writer
	linePrefix string
}

// NewDistributionSummary generates a new distribution summary, using the
// provided meter identifier.
func NewDistributionSummary(id *Id, writer writer.Writer) *DistributionSummary {
	return &DistributionSummary{id, writer, "d:" + id.spectatordId + ":"}
}

// NewDistributionSummaryDirect generates a new distribution summary directly from
// a name and tags, without allocating an *Id or copying the tags map. commonTags
// carries the registry's extraCommonTags.
func NewDistributionSummaryDirect(name string, tags, commonTags map[string]string, writer writer.Writer) *DistributionSummary {
	return &DistributionSummary{nil, writer, buildLinePrefix("d", name, tags, commonTags)}
}

// MeterId returns the meter identifier, reconstructing it from the line prefix
// if the summary was created directly from a name and tags.
func (d *DistributionSummary) MeterId() *Id {
	return resolveMeterId(d.id, d.linePrefix)
}

// Record records a value to track within the distribution.
func (d *DistributionSummary) Record(amount int64) {
	if amount >= 0 {
		d.writer.WriteInt(d.linePrefix, amount)
	}
}
