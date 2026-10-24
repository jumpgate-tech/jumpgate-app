package catalog

import (
	"sync"
	"testing"
)

// A swap must not race with readers still running from an earlier test.
func TestSwapModulesForTestIsRaceFree(t *testing.T) {
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				Modules()
				ModuleByID("msgboard")
			}
		}
	}()
	for i := 0; i < 200; i++ {
		SwapModulesForTest(nil)()
	}
	close(stop)
	wg.Wait()
}
