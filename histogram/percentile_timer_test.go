package histogram

import (
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/Netflix/spectator-go"
)

func makeConfig(uri string) *spectator.Config {
	return &spectator.Config{Frequency: 10 * time.Millisecond, Timeout: 1 * time.Second, Uri: uri, BatchSize: 10000,
		CommonTags: map[string]string{
			"nf.app":     "test",
			"nf.cluster": "test-main",
			"nf.asg":     "test-main-v001",
			"nf.region":  "us-west-1",
		},
	}
}

var config = makeConfig("")

func checkPercentilesTimer(t *testing.T, timer *PercentileTimer, start int) {
	const N = 100000
	for i := 0; i < N; i++ {
		timer.Record(time.Duration(i) * time.Millisecond)
	}
	if timer.Count() != N {
		t.Errorf("Expected count = %d, got %d", N, timer.Count())
	}

	const expectedTotal = time.Duration(N*(N-1)/2) * time.Millisecond

	if timer.TotalTime() != expectedTotal {
		t.Errorf("Expected totalTime = %d ns, got %d ns", expectedTotal, timer.TotalTime())
	}

	for i := start; i <= 100; i++ {
		expected := float64(i)
		threshold := .15 * expected
		actual := timer.Percentile(float64(i))
		if math.Abs(actual-expected) > threshold {
			t.Errorf("Expected %dth percentile to be near %.1f but got %.1f instead", i, expected, actual)
		}
	}
}

func TestPercentileTimer_Basic(t *testing.T) {
	r := spectator.NewRegistry(config)
	timer, _ := PercentileTimerBuilder().Using(r).WithName("p").WithRange(0, 100*time.Second).Build()
	checkPercentilesTimer(t, timer, 0)
}

func checkValue(t *testing.T, t1 *PercentileTimer, t2 *PercentileTimer, expected float64) {
	threshold := expected / 5
	if math.Abs(expected-t1.Percentile(99)) > threshold {
		t.Errorf("Timer 1 - expected 99th percentile around %.1f, got %.1f", expected, t1.Percentile(99))
	}

	if math.Abs(expected-t2.Percentile(99)) > threshold {
		t.Errorf("Timer 2 - expected 99th percentile around %.1f, got %.1f", expected, t2.Percentile(99))
	}
}

func TestPercentileTimer_DifferentRanges(t *testing.T) {
	r := spectator.NewRegistry(config)
	t1, _ := PercentileTimerBuilder().Using(r).WithName("test").WithRange(10*time.Second, 50*time.Second).Build()
	t2, _ := PercentileTimerBuilder().Using(r).WithName("test").WithRange(100*time.Second, 200*time.Second).Build()

	t1.Record(5 * time.Second)
	checkValue(t, t1, t2, 10.0)

	t1.Record(500 * time.Second)
	checkValue(t, t1, t2, 50.0)

	t2.Record(5 * time.Second)
	checkValue(t, t1, t2, 100.0)

	t2.Record(500 * time.Second)
	checkValue(t, t1, t2, 200.0)
}

func measurementsToMap(ms []spectator.Measurement) map[string]float64 {
	var result = make(map[string]float64)
	for _, m := range ms {
		idStr := fmt.Sprintf("%s|%s", m.Id().Name(), m.Id().Tags()["statistic"])
		p := m.Id().Tags()["percentile"]
		if p != "" {
			idStr += "|" + p
		}
		result[idStr] = m.Value()
	}
	return result
}

func filterRegistrySize(ms []spectator.Measurement) []spectator.Measurement {
	var filtered []spectator.Measurement
	for _, m := range ms {
		if m.Id().Name() == "spectator.registrySize" {
			continue
		}

		filtered = append(filtered, m)
	}
	return filtered
}

func TestPercentileTimer_Measurements(t *testing.T) {
	r := spectator.NewRegistry(config)
	t1 := NewPercentileTimer(r, "test", map[string]string{})

	t1.Record(2 * time.Second)
	t1.Record(40 * time.Second)

	ms := filterRegistrySize(r.Measurements())
	ms_map := measurementsToMap(ms)
	var expected = make(map[string]float64)
	expected["test|max"] = 40
	expected["test|count"] = 2
	expected["test|totalTime"] = 40 + 2
	expected["test|totalOfSquares"] = 40*40 + 2*2
	expected["test|percentile|T0086"] = 1
	expected["test|percentile|T0099"] = 1

	if !reflect.DeepEqual(ms_map, expected) {
		t.Errorf("Expected: %v\nGot %v", expected, ms_map)
	}

	t1.Record(2 * time.Second)
	t1.Record(40 * time.Second)
	t1.Record(140 * time.Second)
	t1.Record(240 * time.Second)
	ms = filterRegistrySize(r.Measurements())
	ms_map = measurementsToMap(ms)
	expected["test|max"] = 240
	expected["test|count"] = 4
	expected["test|totalTime"] = 40 + 2 + 140 + 240
	expected["test|totalOfSquares"] = 40*40 + 2*2 + 140*140 + 240*240
	expected["test|percentile|T0086"] = 1
	expected["test|percentile|T0099"] = 1
	expected["test|percentile|T009D"] = 2

	if !reflect.DeepEqual(ms_map, expected) {
		t.Errorf("Expected: %v\nGot %v", expected, ms_map)
	}
}

func TestPercentileTimerBuilder_Preconditions(t *testing.T) {
	r := spectator.NewRegistry(config)

	_, err := PercentileTimerBuilder().Build()
	if err == nil {
		t.Errorf("should fail if registry or name/id are missing")
	}

	_, err = PercentileTimerBuilder().Using(r).Build()
	if err == nil {
		t.Errorf("should fail if name/id are missing")
	}

	_, err = PercentileTimerBuilder().WithName("foo").Build()
	if err == nil {
		t.Errorf("should fail if registry is missing")
	}

	_, err = PercentileTimerBuilder().WithId(r.NewId("foo", map[string]string{})).Build()
	if err == nil {
		t.Errorf("should fail if registry is missing")
	}
}

func TestPercentileTimerBuilder_IdTags(t *testing.T) {
	r := spectator.NewRegistry(config)

	id := r.NewId("foo", map[string]string{})
	p, err := PercentileTimerBuilder().Using(r).WithId(id).Build()
	if p == nil || err != nil {
		t.Fatalf("should succeed")
	}

	if p.id.Name() != "foo" {
		t.Errorf("Expected name 'foo' got %s", p.id.Name())
	}

	if len(p.id.Tags()) != 0 {
		t.Errorf("Expected no tags got %v", p.id.Tags())
	}

	_, err = PercentileTimerBuilder().Using(r).Build()
	if err == nil {
		t.Errorf("should fail if name/id are missing")
	}

	_, err = PercentileTimerBuilder().WithName("foo").Build()
	if err == nil {
		t.Errorf("should fail if registry is missing")
	}

	tags := map[string]string{"foo": "bar", "k": "v"}
	p, err = PercentileTimerBuilder().WithId(id).WithTags(tags).Using(r).Build()
	if err != nil {
		t.Fatalf("should succeed")
	}

	if !reflect.DeepEqual(p.id.Tags(), tags) {
		t.Errorf("Expected extra tags %v, got %v", tags, p.id.Tags())
	}
}
