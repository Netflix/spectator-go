package spectator

import (
	"os"
	"runtime"
	"syscall"
	"time"
)

func getNumFiles(dir string) int {
	f, err := os.Open(dir)
	if err != nil {
		return 0
	}
	defer f.Close()
	entries, err := f.Readdirnames(-1)
	if err != nil {
		return 0
	}
	return len(entries)
}

type sysStatsCollector struct {
	registry      *Registry
	curOpen       *Gauge
	maxOpen       *Gauge
	numGoroutines *Gauge
}

func updateFdStats(s *sysStatsCollector, cur int, max uint64) {
	s.curOpen.Set(float64(cur))
	s.maxOpen.Set(float64(max))
}

func fdStats(s *sysStatsCollector) {
	// do not include /proc/self/fd in the count, since it will be opened
	// when we get the number of files under self/fd
	currentFdCount := getNumFiles("/proc/self/fd") - 1
	var rl syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rl); err != nil {
		s.registry.log.Errorf("Unable to get max open files")
	}
	maxFdCount := rl.Cur
	updateFdStats(s, currentFdCount, maxFdCount)
}

func goRuntimeStats(s *sysStatsCollector) {
	s.numGoroutines.Set(float64(runtime.NumGoroutine()))
}

// Collects system stats: current/max file handles, number of goroutines
func CollectSysStats(registry *Registry) {
	var s sysStatsCollector
	s.registry = registry
	s.maxOpen = registry.Gauge("fh.allocated", nil)
	s.curOpen = registry.Gauge("fh.max", nil)
	s.numGoroutines = registry.Gauge("go.numGoroutines", nil)

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		log := registry.log
		for {
			select {
			case <-ticker.C:
				log.Debugf("Collecting system stats")
				fdStats(&s)
				goRuntimeStats(&s)
			}
		}
	}()
}

// Starts the collection of memory and file handle metrics
func CollectRuntimeMetrics(registry *Registry) {
	CollectMemStats(registry)
	CollectSysStats(registry)
}
