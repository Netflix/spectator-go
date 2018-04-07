package spectator

import (
	"runtime"
	"time"
)

type memStatsCollector struct {
	registry         *Registry
	bytesAlloc       *Gauge
	allocationRate   *MonotonicCounter
	totalBytesSystem *Gauge
	numLiveObjects   *Gauge
	objectsAllocated *MonotonicCounter
	objectsFreed     *MonotonicCounter

	gcLastPauseTimeValue uint64
	gcPauseTime          *Timer
	gcAge                *Gauge
	gcCount              *MonotonicCounter
	forcedGcCount        *MonotonicCounter
	gcPercCpu            *Gauge
}

func memStats(m *memStatsCollector) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	updateMemStats(m, &mem)
}

func updateMemStats(m *memStatsCollector, mem *runtime.MemStats) {
	m.bytesAlloc.Set(float64(mem.Alloc))
	m.allocationRate.Set(int64(mem.TotalAlloc))
	m.totalBytesSystem.Set(float64(mem.Sys))
	m.numLiveObjects.Set(float64(mem.Mallocs - mem.Frees))
	m.objectsAllocated.Set(int64(mem.Mallocs))
	m.objectsFreed.Set(int64(mem.Frees))

	if mem.LastGC != m.gcLastPauseTimeValue {
		m.gcPauseTime.Record(time.Duration(mem.LastGC - m.gcLastPauseTimeValue))
		m.gcLastPauseTimeValue = mem.LastGC
	}

	nanos := m.registry.clock.Nanos()
	timeSinceLastGC := nanos - int64(mem.LastGC)
	secondsSinceLastGC := float64(timeSinceLastGC) / 1e9
	m.gcAge.Set(secondsSinceLastGC)

	m.gcCount.Set(int64(mem.NumGC))
	m.forcedGcCount.Set(int64(mem.NumForcedGC))
	m.gcPercCpu.Set(mem.GCCPUFraction * 100)
}

// Collect memory stats
// https://golang.org/pkg/runtime/#MemStats
func CollectMemStats(registry *Registry) {
	var mem memStatsCollector
	tags := map[string]string{
		"id": "memstats",
	}
	mem.registry = registry
	mem.bytesAlloc = registry.Gauge("mem.heapBytesAllocated", tags)
	mem.allocationRate = NewMonotonicCounter(registry, "mem.allocationRate", tags)
	mem.totalBytesSystem = registry.Gauge("mem.maxHeapBytes", tags)
	mem.numLiveObjects = registry.Gauge("mem.numLiveObjects", tags)
	mem.objectsAllocated = NewMonotonicCounter(registry, "mem.objectsAllocated", tags)
	mem.objectsFreed = NewMonotonicCounter(registry, "mem.objectsFreed", tags)
	mem.gcPauseTime = registry.Timer("gc.pauseTime", tags)
	mem.gcAge = registry.Gauge("gc.timeSinceLastGC", tags)
	mem.gcCount = NewMonotonicCounter(registry, "gc.count", tags)
	mem.forcedGcCount = NewMonotonicCounter(registry, "gc.forcedCount", tags)
	mem.gcPercCpu = registry.Gauge("gc.cpuPercentage", tags)

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		log := registry.log
		for {
			select {
			case <-ticker.C:
				log.Debugf("Collecting memory stats")
				memStats(&mem)
			}
		}
	}()
}
