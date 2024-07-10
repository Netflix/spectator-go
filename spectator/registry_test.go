package spectator

import (
	"fmt"
	"github.com/Netflix/spectator-go/v2/spectator/logger"
	"github.com/Netflix/spectator-go/v2/spectator/writer"
	"testing"
	"time"
)

func TestRegistryWithMemoryWriter_AgeGauge(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistry(mw)

	ageGauge := r.AgeGauge("test_age_gauge", nil)
	ageGauge.Set(100)
	expected := "A:test_age_gauge:100"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_AgeGaugeWithId(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistryWithCommonTags(mw)

	ageGauge := r.AgeGaugeWithId(r.NewId("test_age_gauge", nil))
	ageGauge.Set(100)
	expected := "A:test_age_gauge,extra-tag=foo:100"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_Counter(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistry(mw)

	counter := r.Counter("test_counter", nil)
	counter.Increment()
	expected := "c:test_counter:1"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_CounterWithId(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistryWithCommonTags(mw)

	counter := r.CounterWithId(r.NewId("test_counter", nil))
	counter.Increment()
	expected := "c:test_counter,extra-tag=foo:1"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_DistributionSummary(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistry(mw)

	distSummary := r.DistributionSummary("test_distributionsummary", nil)
	distSummary.Record(300)
	expected := "d:test_distributionsummary:300"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_DistributionSummaryWithId(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistryWithCommonTags(mw)

	distSummary := r.DistributionSummaryWithId(r.NewId("test_distributionsummary", nil))
	distSummary.Record(300)
	expected := "d:test_distributionsummary,extra-tag=foo:300"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_Gauge(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistry(mw)

	gauge := r.Gauge("test_gauge", nil)
	gauge.Set(100)
	expected := "g:test_gauge:100.000000"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_GaugeWithId(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistryWithCommonTags(mw)

	gauge := r.GaugeWithId(r.NewId("test_gauge", nil))
	gauge.Set(100)
	expected := "g:test_gauge,extra-tag=foo:100.000000"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_GaugeWithTTL(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistry(mw)

	ttl := 60 * time.Second
	gauge := r.GaugeWithTTL("test_gauge_ttl", nil, ttl)
	gauge.Set(100.1)

	expected := fmt.Sprintf("g,%d:test_gauge_ttl:100.100000", int(ttl.Seconds()))
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_GaugeWithIdWithTTL(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistryWithCommonTags(mw)

	ttl := 60 * time.Second
	gauge := r.GaugeWithIdWithTTL(r.NewId("test_gauge_ttl", nil), ttl)
	gauge.Set(100.1)

	expected := fmt.Sprintf("g,%d:test_gauge_ttl,extra-tag=foo:100.100000", int(ttl.Seconds()))
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_MaxGauge(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistry(mw)

	maxGauge := r.MaxGauge("test_maxgauge", nil)
	maxGauge.Set(200)
	expected := "m:test_maxgauge:200.000000"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_MaxGaugeWithId(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistryWithCommonTags(mw)

	maxGauge := r.MaxGaugeWithId(r.NewId("test_maxgauge", nil))
	maxGauge.Set(200)
	expected := "m:test_maxgauge,extra-tag=foo:200.000000"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_MonotonicCounter(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistry(mw)

	counter := r.MonotonicCounter("test_monotonic_counter", nil)
	counter.Set(1)
	expected := "C:test_monotonic_counter:1.000000"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_MonotonicCounterWithId(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistryWithCommonTags(mw)

	counter := r.MonotonicCounterWithId(r.NewId("test_monotonic_counter", nil))
	counter.Set(1)
	expected := "C:test_monotonic_counter,extra-tag=foo:1.000000"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_MonotonicCounterUint(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistry(mw)

	counter := r.MonotonicCounterUint("test_monotonic_counter_uint", nil)
	counter.Set(1)
	expected := "U:test_monotonic_counter_uint:1"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_MonotonicCounterUintWithId(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistryWithCommonTags(mw)

	counter := r.MonotonicCounterUintWithId(r.NewId("test_monotonic_counter_uint", nil))
	counter.Set(1)
	expected := "U:test_monotonic_counter_uint,extra-tag=foo:1"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_PercentileDistributionSummary(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistry(mw)

	percentileDistSummary := r.PercentileDistributionSummary("test_percentiledistributionsummary", nil)
	percentileDistSummary.Record(400)
	expected := "D:test_percentiledistributionsummary:400"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_PercentileDistributionSummaryWithId(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistryWithCommonTags(mw)

	percentileDistSummary := r.PercentileDistributionSummaryWithId(r.NewId("test_percentiledistributionsummary", nil))
	percentileDistSummary.Record(400)
	expected := "D:test_percentiledistributionsummary,extra-tag=foo:400"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_PercentileTimer(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistry(mw)

	percentileTimer := r.PercentileTimer("test_percentiletimer", nil)
	percentileTimer.Record(500 * time.Millisecond)
	expected := "T:test_percentiletimer:0.500000"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_PercentileTimerWithId(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistryWithCommonTags(mw)

	percentileTimer := r.PercentileTimerWithId(r.NewId("test_percentiletimer", nil))
	percentileTimer.Record(500 * time.Millisecond)
	expected := "T:test_percentiletimer,extra-tag=foo:0.500000"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_Timer(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistry(mw)

	timer := r.Timer("test_timer", nil)
	timer.Record(100 * time.Millisecond)
	expected := "t:test_timer:0.100000"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestRegistryWithMemoryWriter_TimerWithId(t *testing.T) {
	mw := &writer.MemoryWriter{}
	r := NewTestRegistryWithCommonTags(mw)

	timer := r.TimerWithId(r.NewId("test_timer", nil))
	timer.Record(100 * time.Millisecond)
	expected := "t:test_timer,extra-tag=foo:0.100000"
	if len(mw.Lines()) != 1 || mw.Lines()[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, mw.Lines()[0])
	}
}

func TestNewRegistryWithEmptyConfig(t *testing.T) {
	_, err := NewRegistry(&Config{})

	if err != nil {
		t.Errorf("Registry should not return an error for empty config, got '%v'", err)
	}
}

func TestNewRegistryWithNilConfig(t *testing.T) {
	_, err := NewRegistry(nil)

	if err == nil {
		t.Errorf("Registry should return an error for nil config, got nil")
	}
}

func NewTestRegistry(mw *writer.MemoryWriter) Registry {
	return &spectatordRegistry{
		config: &Config{},
		writer: mw,
		logger: logger.NewDefaultLogger(),
	}
}

func NewTestRegistryWithCommonTags(mw *writer.MemoryWriter) Registry {
	return &spectatordRegistry{
		config: &Config{"", map[string]string{"extra-tag": "foo"}, nil},
		writer: mw,
		logger: logger.NewDefaultLogger(),
	}
}
