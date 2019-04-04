// +build windows

package spectator

func getNumFiles(dir string) (n int, err error) {
	return 0, nil
}

func updateFdStats(s *sysStatsCollector, cur int, max uint64) {
	// do nothing on Windows
}

func fdStats(s *sysStatsCollector) {
	// do nothing on Windows
}
