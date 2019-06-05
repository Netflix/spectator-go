package spectator

import (
	"sync"
	"testing"
)

func TestUpdateMaxRace(t *testing.T) {
	// The previous race condition in updateMax seemed to get caught at least 1/100 times with the below test, so we
	// just run it multiple times to try and catch any race issue.
	for testIteration := 0; testIteration < 100; testIteration++ {
		m := int64(100)
		wg := sync.WaitGroup{}
		wg.Add(100)
		for i := 0; i < 100; i++ {
			go func(n int64) {
				updateMax(&m, n)
				wg.Done()
			}(int64(100 + i))
		}
		wg.Wait()

		if m != 199 {
			t.Errorf("Got incorrect max value: %d", m)
		}
	}
}
