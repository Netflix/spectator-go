// +build freebsd

// FreeBSD does not mount procfs by default, so omit fd stats on FreeBSD

package spectator

func getNumFiles(dir string) (n int, err error) {
	return 0, nil
}

func updateFdStats(s *sysStatsCollector, cur int, max uint64) {
	// do nothing on FreeBSD
}

func fdStats(s *sysStatsCollector) {
	// do nothing on FreeBSD
}
