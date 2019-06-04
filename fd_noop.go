// +build !linux

package spectator

func getNumFiles(dir string) (n int, err error) {
	return 0, nil
}

func updateFdStats(s *sysStatsCollector, cur int, max uint64) {
	// do nothing
}

func fdStats(s *sysStatsCollector) {
	// do nothing
}
