package logwatch

import (
	"fmt"
	"testing"
)

// TestHandleLine_FullRingDoesNotAllocatePerLine is the regression for the
// ring copying itself on every line once full: each new hit appended and
// then re-copied all ringSize entries into a fresh slice, roughly 150 KB of
// allocation per line on a noisy node. A full ring must absorb a new hit
// in place.
func TestHandleLine_FullRingDoesNotAllocatePerLine(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector's instrumentation allocates; run without -race")
	}
	w := New(newFakeExecutor(), nil)
	for i := 0; i < ringSize+10; i++ {
		w.handleLine("u", fmt.Sprintf("ERROR line %d", i))
	}

	allocs := testing.AllocsPerRun(200, func() {
		w.handleLine("u", "ERROR steady state")
	})
	if allocs > 0 {
		t.Fatalf("handleLine on a full ring made %.1f allocations per line, want 0", allocs)
	}
}

// TestRecent_WrapsInOrder drives the ring through several wraps and checks
// Recent keeps its contract: at most ringSize hits, oldest first, newest
// last, and n selects the newest n.
func TestRecent_WrapsInOrder(t *testing.T) {
	w := New(newFakeExecutor(), nil)
	if got := w.Recent(10); len(got) != 0 {
		t.Fatalf("empty watcher Recent(10) len = %d, want 0", len(got))
	}

	total := 2*ringSize + 37
	for i := 0; i < total; i++ {
		w.handleLine("u", fmt.Sprintf("ERROR line %d", i))

		// Spot-check partway through the first fill too, before any wrap.
		if i == 4 {
			got := w.Recent(0)
			if len(got) != 5 || got[0].Line != "ERROR line 0" || got[4].Line != "ERROR line 4" {
				t.Fatalf("pre-wrap Recent(0) = %+v, want lines 0..4", got)
			}
		}
	}

	all := w.Recent(0)
	if len(all) != ringSize {
		t.Fatalf("Recent(0) len = %d, want %d", len(all), ringSize)
	}
	for i, h := range all {
		want := fmt.Sprintf("ERROR line %d", total-ringSize+i)
		if h.Line != want {
			t.Fatalf("Recent(0)[%d] = %q, want %q", i, h.Line, want)
		}
	}

	last3 := w.Recent(3)
	for i, h := range last3 {
		want := fmt.Sprintf("ERROR line %d", total-3+i)
		if h.Line != want {
			t.Fatalf("Recent(3)[%d] = %q, want %q", i, h.Line, want)
		}
	}

	// The returned slice is the caller's: mutating it must not reach the
	// ring.
	last3[0].Line = "mutated"
	if got := w.Recent(3)[0].Line; got == "mutated" {
		t.Fatal("Recent returned a slice aliasing the ring")
	}
}

// BenchmarkHandleLine_FullRing measures the steady-state cost of one
// classified line once the ring has filled.
func BenchmarkHandleLine_FullRing(b *testing.B) {
	w := New(newFakeExecutor(), nil)
	for i := 0; i < ringSize; i++ {
		w.handleLine("u", "ERROR warmup")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.handleLine("u", "ERROR steady state")
	}
}
