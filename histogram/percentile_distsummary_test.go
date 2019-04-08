package histogram

import (
	"github.com/Netflix/spectator-go"
	"math"
	"reflect"
	"testing"
)

func TestPercentileDistributionSummary_Basic(t *testing.T) {
	r := spectator.NewRegistry(config)
	ds := NewPercentileDistributionSummary(r, "ds", map[string]string{})

	ds.Record(1000) // 0x29
	ds.Record(2000) // 0x2c
	ds.Record(3000) // 0x2F
	ds.Record(3001) // 0x2F

	ms := r.Measurements()
	measurementMap := measurementsToMap(ms)
	var expected = make(map[string]float64)
	expected["ds|count"] = 4
	expected["ds|totalAmount"] = 1000 + 2000 + 3000 + 3001
	expected["ds|max"] = 3001
	expected["ds|totalOfSquares"] = 1000*1000 + 2000*2000 + 3000*3000 + 3001*3001
	expected["ds|percentile|D0029"] = 1
	expected["ds|percentile|D002C"] = 1
	expected["ds|percentile|D002F"] = 2

	if !reflect.DeepEqual(expected, measurementMap) {
		t.Errorf("Expected %v, got %v", expected, measurementMap)
	}
}

func TestPercentileDistributionSummary_CountAmount(t *testing.T) {
	r := spectator.NewRegistry(config)
	ds := NewPercentileDistributionSummary(r, "ds", map[string]string{})

	ds.Record(1000) // 0x29
	ds.Record(2000) // 0x2c
	ds.Record(3000) // 0x2F
	ds.Record(3001) // 0x2F

	if ds.Count() != 4 {
		t.Errorf("Expected count = 4, got %d", ds.Count())
	}

	var expectedTotal int64 = 1000 + 2000 + 3000 + 3001
	if ds.TotalAmount() != expectedTotal {
		t.Errorf("Expected totalAmount=%d, got %d", expectedTotal, ds.TotalAmount())
	}
}

func checkPercentilesDs(t *testing.T, ds *PercentileDistributionSummary) {
	const N = 100000
	for i := 0; i < N; i++ {
		ds.Record(int64(i))
	}
	if ds.Count() != N {
		t.Errorf("Expected count = %d, got %d", N, ds.Count())
	}

	const expectedTotal = N * (N - 1) / 2

	if ds.TotalAmount() != expectedTotal {
		t.Errorf("Expected totalAmount = %d ns, got %d", expectedTotal, ds.TotalAmount())
	}

	for i := 0; i <= 100; i++ {
		expected := float64(i) * 1000
		threshold := .10 * expected
		actual := ds.Percentile(float64(i))
		if math.Abs(actual-expected) > threshold {
			t.Errorf("Expected %dth percentile to be near %.1f but got %.1f instead", i, expected, actual)
		}
	}
}

func TestPercentileDistributionSummary_Percentile(t *testing.T) {
	r := spectator.NewRegistry(config)
	ds := NewPercentileDistributionSummary(r, "ds", map[string]string{})

	checkPercentilesDs(t, ds)
}
