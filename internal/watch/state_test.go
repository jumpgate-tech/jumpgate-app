package watch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestStateWriteIsAtomic(t *testing.T) {
	// The watch runs only on Linux boxes. Windows refuses to rename over a
	// file another goroutine has open, and has no 0640 modes to check.
	if runtime.GOOS == "windows" {
		t.Skip("the watch state file is a Linux box file")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "state.json")
	big := State{At: t0}
	for i := range 2000 {
		big.Open = append(big.Open, Alert{Key: "endpoint:x", Message: "padding padding padding", Since: t0.Add(time.Duration(i))})
	}
	if err := WriteState(p, big); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			b, err := os.ReadFile(p)
			if err != nil {
				t.Error(err)
				return
			}
			var s State
			if err := json.Unmarshal(b, &s); err != nil {
				t.Errorf("partial read: %v", err)
				return
			}
		}
	}()
	for range 50 {
		if err := WriteState(p, big); err != nil {
			t.Fatal(err)
		}
	}
	close(stop)
	wg.Wait()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o640 {
		t.Errorf("mode %v", fi.Mode().Perm())
	}
	ents, _ := os.ReadDir(dir)
	if len(ents) != 1 {
		t.Errorf("left temp files: %v", ents)
	}
}
