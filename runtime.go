package spectator

import (
	"os"
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

type fdStatsCollector struct {
	registry *Registry
	curOpen  *Gauge
	maxOpen  *Gauge
}

func updateFdStats(f *fdStatsCollector, cur int, max uint64) {
	f.curOpen.Set(float64(cur))
	f.maxOpen.Set(float64(max))
}

func fdStats(f *fdStatsCollector) {
	// do not include /proc/self/fd in the count, since it will be opened
	// when we get the number of files under self/fd
	currentFdCount := getNumFiles("/proc/self/fd") - 1
	var rl syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rl); err != nil {
		f.registry.log.Errorf("Unable to get max open files")
	}
	maxFdCount := rl.Cur
	updateFdStats(f, currentFdCount, maxFdCount)
}

// Collects file handle stats
func CollectFileHandleStats(registry *Registry) {
	var f fdStatsCollector
	f.registry = registry
	tags := map[string]string{
		"id": "fdstats",
	}
	f.maxOpen = registry.Gauge("fh.maxOpen", tags)
	f.curOpen = registry.Gauge("fh.curOpen", tags)

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		log := registry.log
		for {
			select {
			case <-ticker.C:
				log.Debugf("Collecting file handle stats")
				fdStats(&f)
			}
		}
	}()
}

// Starts the collection of memory and file handle metrics
func CollectRuntimeMetrics(registry *Registry) {
	CollectMemStats(registry)
	CollectFileHandleStats(registry)
}
