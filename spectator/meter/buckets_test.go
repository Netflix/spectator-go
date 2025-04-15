package meter

import (
	"math"
	"math/rand"
	"testing"
)

func TestPercentileBucketsLength(t *testing.T) {
	if PercentileBucketsLength() != 276 {
		t.Errorf("Expecting 276 percentile buckets. Got %d", PercentileBucketsLength())
	}
}

func assertIndex(t *testing.T, v int64, expected int) {
	idx := PercentileBucketsIndex(v)
	if idx != expected {
		t.Errorf("PercentileBucketsIndex(%d) = %d, expected %d", v, idx, expected)
	}
}

func TestPercentileBucketsIndex(t *testing.T) {
	assertIndex(t, -1, 0)
	for i := 0; i < 5; i++ {
		assertIndex(t, int64(i), i)
	}

	assertIndex(t, 21, 16)
	assertIndex(t, 31, 18)
	assertIndex(t, 87, 25)
	assertIndex(t, 1020, 41)
	assertIndex(t, 10000, 55)
	assertIndex(t, 100000, 70)
	assertIndex(t, 1000*1000, 86)
	assertIndex(t, 10*1000*1000, 100)
	assertIndex(t, 100*1000*1000, 115)
	assertIndex(t, 1000*1000*1000, 131)
	assertIndex(t, 10*1000*1000*1000, 144)
	assertIndex(t, 100*1000*1000*1000, 160)
	assertIndex(t, 1000*1000*1000*1000, 175)
}

func getNumber() int64 {
	return rand.Int63()
}

func TestPercentileBucketsBucket(t *testing.T) {
	for i := 0; i < 10000; i++ {
		n := getNumber()
		b := PercentileBucketsBucket(n)
		if n > b {
			t.Errorf("Expecting %d <= %d", n, b)
		}
	}
}

func assertPercentiles(t *testing.T, results []float64, expected []float64) {
	if len(results) != len(expected) {
		t.Errorf("Expected %d results, got %d instead",
			len(expected), len(results))
	}

	for i, e := range expected {
		threshold := e * 0.1
		if math.Abs(e-results[i]) > threshold {
			t.Errorf("Percentile at index %d: Got %.1f - Expected %.1f",
				i, results[i], e)
		}
	}
}

func TestPercentileBucketsPercentiles(t *testing.T) {
	counts := make([]int64, PercentileBucketsLength())
	for i := 0; i < 100000; i++ {
		counts[PercentileBucketsIndex(int64(i))]++
	}
	var pcts []float64
	pcts = append(pcts,
		0.0, 25.0, 50.0, 75.0, 90.0, 95.0, 98.0, 99.0, 99.5, 100.0)

	results := PercentileBucketsPercentiles(counts, pcts)
	var expected []float64
	expected = append(expected,
		0.0, 25e3, 50e3, 75e3, 90e3, 95e3, 98e3, 99e3, 99.5e3, 100e3)
	assertPercentiles(t, results, expected)
}

func TestPercentileBucketsPercentile(t *testing.T) {
	counts := make([]int64, PercentileBucketsLength())
	for i := 0; i < 100000; i++ {
		counts[PercentileBucketsIndex(int64(i))]++
	}

	var pcts []float64
	pcts = append(pcts,
		0.0, 25.0, 50.0, 75.0, 90.0, 95.0, 98.0, 99.0, 99.5, 100.0)
	for _, pct := range pcts {
		expected := pct * 1e3
		threshold := 0.1*expected + 1e-12
		p := PercentileBucketsPercentile(counts, pct)
		if math.Abs(expected-p) > threshold {
			t.Errorf("Failed to compute %.1f percentile: %.1f != %.1f",
				pct, p, expected)
		}
	}
}
