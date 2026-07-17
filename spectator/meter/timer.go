package meter

import (
	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"time"
)

// Timer is used to measure how long (in seconds) some event is taking. This
// type is safe for concurrent use.
type Timer struct {
	id         *Id
	writer     writer.Writer
	linePrefix string
}

// NewTimer generates a new timer, using the provided meter identifier.
func NewTimer(id *Id, writer writer.Writer) *Timer {
	return &Timer{id, writer, "t:" + id.spectatordId + ":"}
}

// NewTimerDirect generates a new timer directly from a name and tags, without
// allocating an *Id or copying the tags map. commonTags carries the registry's
// extraCommonTags.
func NewTimerDirect(name string, tags, commonTags map[string]string, writer writer.Writer) *Timer {
	return &Timer{nil, writer, buildLinePrefix("t", name, tags, commonTags)}
}

// MeterId returns the meter identifier, reconstructing it from the line prefix
// if the timer was created directly from a name and tags.
func (t *Timer) MeterId() *Id {
	return resolveMeterId(t.id, t.linePrefix)
}

// Record records the duration this specific event took.
func (t *Timer) Record(amount time.Duration) {
	if amount >= 0 {
		t.writer.WriteFloat(t.linePrefix, amount.Seconds())
	}
}
