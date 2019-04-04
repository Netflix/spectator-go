package histogram

import (
	"github.com/Netflix/spectator-go"
	"math"
	"math/bits"
)

var bucketValues []int64
var powerOf4Index []int

func init() {
	const DIGITS uint8 = 2
	bucketValues = append(bucketValues, 1, 2, 3)
	powerOf4Index = append(powerOf4Index, 0)

	for exp := DIGITS; exp < 64; exp += DIGITS {
		var current int64 = 1 << exp
		delta := current / 3
		next := (current << DIGITS) - delta
		powerOf4Index = append(powerOf4Index, len(bucketValues))

		for ; current < next; current += delta {
			bucketValues = append(bucketValues, current)
		}
	}

	bucketValues = append(bucketValues, math.MaxInt64)
}

func counterFor(registry *spectator.Registry, id *spectator.Id, i int, tagValues []string) *spectator.Counter {
	tags := map[string]string{"statistic": "percentile"}
	tags["percentile"] = tagValues[i]
	return registry.CounterWithId(id.WithTags(tags))
}

func PercentileBucketsLength() int {
	return len(bucketValues)
}

func PercentileBucketsIndex(v int64) int {
	if v <= 0 {
		return 0
	} else if v <= 15 {
		return int(v)
	} else {
		lz := bits.LeadingZeros64(uint64(v))
		shift := uint(64 - lz - 1)
		prevPowerOf2 := (v >> shift) << shift
		prevPowerOf4 := prevPowerOf2
		if shift%2 != 0 {
			shift--
			prevPowerOf4 = prevPowerOf2 >> 1
		}

		base := prevPowerOf4
		delta := base / 3
		offset := int((v - base) / delta)
		pos := offset + powerOf4Index[shift/2]
		if pos >= len(bucketValues)-1 {
			return len(bucketValues) - 1
		}
		return pos + 1
	}
}

func PercentileBucketsBucket(v int64) int64 {
	return bucketValues[PercentileBucketsIndex(v)]
}

func PercentileBucketsPercentiles(counts []int64, pcts []float64) []float64 {
	results := make([]float64, len(pcts))
	total := float64(0)
	for _, c := range counts {
		total += float64(c)
	}
	pctIndex := 0
	prev := int64(0)
	prevP := 0.0
	prevB := int64(0)
	for i, nextB := range bucketValues {
		next := prev + counts[i]
		nextP := 100 * float64(next) / total
		for ; pctIndex < len(pcts) && nextP >= pcts[pctIndex]; pctIndex++ {
			f := (pcts[pctIndex] - prevP) / (nextP - prevP)
			results[pctIndex] = f*float64(nextB-prevB) + float64(prevB)
		}
		if pctIndex >= len(pcts) {
			break
		}
		prev = next
		prevP = nextP
		prevB = nextB
	}

	return results
}

func PercentileBucketsPercentile(counts []int64, pct float64) float64 {
	pcts := make([]float64, 1)
	pcts[0] = pct
	results := PercentileBucketsPercentiles(counts, pcts)
	return results[0]
}
