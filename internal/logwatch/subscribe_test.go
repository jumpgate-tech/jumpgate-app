package logwatch

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

// A line published while a subscriber takes its backlog lands in the backlog
// or on the channel, never both and never neither (spec A4: a reconnect
// neither duplicates nor skips).
func TestSubscribeRecentNeverDuplicatesOrSkips(t *testing.T) {
	const lines = 30 // under the channel's buffer, so nothing is dropped
	for round := 0; round < 1000; round++ {
		w := New(newFakeExecutor(), nil)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < lines; i++ {
				w.handleLine("u", fmt.Sprintf("ERROR line %d", i))
				runtime.Gosched()
			}
		}()
		for len(w.Recent(0)) < lines/3 { // subscribe mid-burst
			runtime.Gosched()
		}
		backlog, ch, unsub := w.SubscribeRecent(ringSize)
		wg.Wait()
		seen := map[string]int{}
		for _, h := range backlog {
			seen[h.Line]++
		}
	drain:
		for {
			select {
			case h := <-ch:
				seen[h.Line]++
			default:
				break drain
			}
		}
		unsub()
		for i := 0; i < lines; i++ {
			if n := seen[fmt.Sprintf("ERROR line %d", i)]; n != 1 {
				t.Fatalf("round %d: line %d seen %d times (backlog %d lines)", round, i, n, len(backlog))
			}
		}
	}
}

func TestSubscribeRecentBacklogIsTheNewestN(t *testing.T) {
	w := New(newFakeExecutor(), nil)
	for i := 0; i < 5; i++ {
		w.handleLine("u", fmt.Sprintf("ERROR line %d", i))
	}
	for n, want := range map[int]int{2: 2, 99: 5, 0: 0, -1: 0} {
		backlog, _, unsub := w.SubscribeRecent(n)
		unsub()
		if len(backlog) != want || backlog == nil {
			t.Errorf("SubscribeRecent(%d) = %d lines (nil %v), want %d", n, len(backlog), backlog == nil, want)
		}
	}
}

// A subscriber that fell behind and lost lines is told so: Resync hands it a
// fresh backlog and empties its channel, so it can replace what it holds
// instead of silently skipping. One that lost nothing gets false.
func TestResyncAfterDrops(t *testing.T) {
	w := New(newFakeExecutor(), nil)
	_, ch, unsub := w.SubscribeRecent(10)
	defer unsub()
	w.handleLine("u", "ERROR first")
	if _, ok := w.Resync(ch, 10); ok {
		t.Fatal("Resync without a drop")
	}
	<-ch
	for i := 0; i < 40; i++ { // the channel holds 32
		w.handleLine("u", fmt.Sprintf("ERROR line %d", i))
	}
	backlog, ok := w.Resync(ch, 10)
	if !ok || len(backlog) != 10 || backlog[9].Line != "ERROR line 39" {
		t.Fatalf("Resync = %d lines, %v", len(backlog), ok)
	}
	if len(ch) != 0 {
		t.Fatalf("%d stale lines left on the channel", len(ch))
	}
	if _, ok := w.Resync(ch, 10); ok {
		t.Fatal("a Resync clears the drop count")
	}
	w.handleLine("u", "ERROR after")
	if h := <-ch; h.Line != "ERROR after" {
		t.Fatalf("next line %q", h.Line)
	}
}
