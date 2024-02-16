package spectator

import (
	"runtime"
	"testing"
	"time"
)

func TestUpdateMemStats(t *testing.T) {
	var clock ManualClock
	registry := NewRegistryWithClock(makeConfig(""), &clock)
	var mem memStatsCollector

	initializeMemStatsCollector(registry, &mem)
	clock.SetFromDuration(1 * time.Minute)

	var memStats runtime.MemStats
	memStats.Alloc = 100
	memStats.TotalAlloc = 200
	memStats.Sys = 300
	memStats.Mallocs = 10
	memStats.Frees = 5
	memStats.LastGC = uint64(30 * time.Second)
	memStats.NumGC = 2
	memStats.NumForcedGC = 1
	memStats.GCCPUFraction = .5
	memStats.PauseTotalNs = uint64(5 * time.Millisecond)
	updateMemStats(&mem, &memStats)

	ms := myMeters(registry)
	if len(ms) != 11 {
		t.Error("Expected 11 meters registered, got", len(ms))
	}

	expectedValues := map[string]float64{
		"mem.numLiveObjects":     5,
		"mem.heapBytesAllocated": 100,
		"mem.maxHeapBytes":       300,
		"gc.timeSinceLastGC":     float64(30),
		"gc.cpuPercentage":       50,
	}
	for _, m := range ms {
		name := m.MeterId().name
		if name == "gc.pauseTime" {
			assertTimer(t, m.(*Timer), 1, 5*1e6, 25*1e12, 5*1e6)
		} else {
			expected := expectedValues[name]
			measures := m.Measure()
			if expected > 0 {
				if len(measures) != 1 {
					t.Fatalf("Expected one value from %v: got %d", m.MeterId(), len(measures))
				}
				if v := measures[0].value; v != expected {
					t.Errorf("%v: expected %f. got %f", m.MeterId(), expected, v)
				}
			} else {
				if len(measures) != 0 {
					t.Errorf("Unexpected measurements from %v: got %d measurements", m.MeterId(), len(measures))
				}
			}
		}
	}

	clock.SetFromDuration(2 * time.Minute)

	memStats.Alloc = 200
	memStats.TotalAlloc = 400
	memStats.Sys = 600
	memStats.Mallocs = 20
	memStats.Frees = 10
	memStats.LastGC = uint64(30 * time.Second)
	memStats.NumGC = 5
	memStats.NumForcedGC = 2
	memStats.GCCPUFraction = .4
	memStats.PauseTotalNs = uint64(15 * time.Millisecond)

	updateMemStats(&mem, &memStats)
	ms = registry.Meters()
	expectedValues = map[string]float64{
		"mem.numLiveObjects":     10,
		"mem.heapBytesAllocated": 200,
		"mem.maxHeapBytes":       600,
		"mem.objectsAllocated":   10,
		"mem.objectsFreed":       5,
		"mem.allocationRate":     200,
		"gc.timeSinceLastGC":     float64(90),
		"gc.cpuPercentage":       40,
		"gc.count":               3,
		"gc.forcedCount":         1,
	}
	for _, m := range ms {
		name := m.MeterId().name
		switch name {
		case "gc.pauseTime":
			assertTimer(t, m.(*Timer), 1, 10*1e6, 100*1e12, 10*1e6)
		case "spectator.registrySize":
		default:
			expected := expectedValues[name]
			measures := m.Measure()
			if expected > 0 {
				if len(measures) != 1 {
					t.Errorf("Expected one value from %v: got %d", m.MeterId(), len(measures))
				}
				if v := measures[0].value; v != expected {
					t.Errorf("%v: expected %f. got %f", m.MeterId(), expected, v)
				}
			} else if len(measures) != 0 {
				t.Errorf("Unexpected measurements from %v: got %d measurements", m.MeterId(), len(measures))
			}
		}
	}
}
