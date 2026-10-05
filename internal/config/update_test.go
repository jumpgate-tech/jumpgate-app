package config

import (
	"fmt"
	"sync"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// Two processes editing config.json must not lose each other's edits. Each
// goroutine here takes the lock through its own file descriptor, which is how
// a second process would, so this is the cross-process case in miniature.
func TestUpdateSerialisesConcurrentEdits(t *testing.T) {
	testutil.Home(t)
	const n = 20
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := Update(func(c *Config) error {
				c.Targets = append(c.Targets, Target{ID: fmt.Sprintf("t%d", i), Mode: "local"})
				return nil
			})
			if err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Targets) != n {
		t.Fatalf("%d targets saved, want %d: concurrent edits were lost", len(c.Targets), n)
	}
}

func TestUpdateDoesNotSaveWhenFnFails(t *testing.T) {
	testutil.Home(t)
	_, err := Update(func(c *Config) error {
		c.Targets = append(c.Targets, Target{ID: "x", Mode: "local"})
		return fmt.Errorf("nope")
	})
	if err == nil {
		t.Fatal("want fn's error")
	}
	c, _ := Load()
	if len(c.Targets) != 0 {
		t.Fatal("a failed Update was saved")
	}
}
