package meter

import (
	"encoding/binary"
	"strconv"

	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"github.com/cespare/xxhash/v2"
)

// DistinctCountSketch estimates the number of distinct values seen during a step interval, such as
// the number of unique users, device ids, or source IPs, using a HyperLogLog sketch. The result is
// an estimate (~13% standard error), not an exact count.
//
// This client computes the xxHash64 of the recorded value locally and sends the precomputed hash
// to SpectatorD (the 's' line type), which derives the HyperLogLog registers. The hashing matches
// the other Spectator clients so that sketches recorded by different clients merge correctly.
// Integers are hashed as their 8-byte little-endian representation and strings as their UTF-8
// bytes.
//
// Recording into a sketch results in a fixed number of gauges (64) being published by SpectatorD,
// so treat any additional dimensions with the same diligence as percentile timers and ensure they
// have a small bounded cardinality. This type is safe for concurrent use.
type DistinctCountSketch struct {
	id         *Id
	writer     writer.Writer
	linePrefix string
}

// NewDistinctCountSketch generates a new distinct count sketch, using the provided meter
// identifier.
func NewDistinctCountSketch(id *Id, writer writer.Writer) *DistinctCountSketch {
	return &DistinctCountSketch{id, writer, "s:" + id.spectatordId + ":"}
}

// MeterId returns the meter identifier.
func (d *DistinctCountSketch) MeterId() *Id {
	return d.id
}

// RecordString records a distinct string value, hashed as its UTF-8 bytes.
func (d *DistinctCountSketch) RecordString(value string) {
	d.recordHash(xxhash.Sum64String(value))
}

// RecordBytes records a distinct value from its raw bytes.
func (d *DistinctCountSketch) RecordBytes(value []byte) {
	d.recordHash(xxhash.Sum64(value))
}

// RecordInt64 records a distinct integer value, hashed as its 8-byte little-endian representation.
func (d *DistinctCountSketch) RecordInt64(value int64) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(value))
	d.recordHash(xxhash.Sum64(buf[:]))
}

func (d *DistinctCountSketch) recordHash(hash uint64) {
	d.writer.Write(d.linePrefix + strconv.FormatUint(hash, 10))
}
