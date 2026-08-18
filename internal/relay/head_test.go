package relay

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

// Terminating WebSocket at the relay means the relay synthesises subscriptions
// by polling over HTTP. That buys a much wider upstream pool and one poll loop
// per chain instead of one connection per subscriber. It also moves two jobs a
// node used to do into this package: never deliver a head twice, and never
// deliver a head the chain has abandoned.
//
// These tests are that contract. A poller that skips a block silently loses
// logs for every subscriber on the chain.

// fakeChain is a real in-memory chain the poller reads. Tests reorganise it the
// way a real chain does, by replacing blocks at a height.
type fakeChain struct {
	mu     sync.Mutex
	blocks []BlockRef // index == height
	err    error
}

func newFakeChain() *fakeChain {
	genesis := BlockRef{Number: 0, Hash: "0x00", ParentHash: ""}
	return &fakeChain{blocks: []BlockRef{genesis}}
}

// extend appends n blocks, each linked to the one before it.
func (f *fakeChain) extend(n int, tag string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := 0; i < n; i++ {
		parent := f.blocks[len(f.blocks)-1]
		f.blocks = append(f.blocks, BlockRef{
			Number:     parent.Number + 1,
			Hash:       fmt.Sprintf("0x%s%d", tag, parent.Number+1),
			ParentHash: parent.Hash,
		})
	}
}

// reorg drops back to height then extends with differently tagged blocks, which
// is what a real reorg looks like from the outside.
func (f *fakeChain) reorg(toHeight uint64, n int, tag string) {
	f.mu.Lock()
	f.blocks = f.blocks[:toHeight+1]
	f.mu.Unlock()
	f.extend(n, tag)
}

func (f *fakeChain) HeadNumber(context.Context) (uint64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return 0, f.err
	}
	return f.blocks[len(f.blocks)-1].Number, nil
}

func (f *fakeChain) BlockByNumber(_ context.Context, n uint64) (BlockRef, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return BlockRef{}, f.err
	}
	if n >= uint64(len(f.blocks)) {
		return BlockRef{}, errors.New("no such block")
	}
	return f.blocks[n], nil
}

// collect drains one poll and returns the heads it produced.
func collect(t *testing.T, p *HeadPoller) []BlockRef {
	t.Helper()
	heads, err := p.Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	return heads
}

func TestHeadPollerEmitsNothingWhenTheChainIsStill(t *testing.T) {
	chain := newFakeChain()
	chain.extend(3, "a")
	p := NewHeadPoller(chain)

	collect(t, p) // prime on the current head
	if got := collect(t, p); len(got) != 0 {
		t.Errorf("second poll produced %d heads, want 0", len(got))
	}
}

func TestHeadPollerEmitsOneNewHead(t *testing.T) {
	chain := newFakeChain()
	chain.extend(3, "a")
	p := NewHeadPoller(chain)
	collect(t, p)

	chain.extend(1, "a")
	got := collect(t, p)
	if len(got) != 1 {
		t.Fatalf("got %d heads, want 1", len(got))
	}
	if got[0].Number != 4 {
		t.Errorf("head number = %d, want 4", got[0].Number)
	}
}

// A poll interval slower than the block time must not skip blocks. A skipped
// block is silently lost logs for every subscriber on the chain, which is the
// worst failure this package can have because nothing reports it.
func TestHeadPollerFillsAGapWithoutSkipping(t *testing.T) {
	chain := newFakeChain()
	chain.extend(2, "a")
	p := NewHeadPoller(chain)
	collect(t, p)

	chain.extend(5, "a") // five blocks between two polls
	got := collect(t, p)
	if len(got) != 5 {
		t.Fatalf("got %d heads, want 5 — the poller skipped blocks", len(got))
	}
	for i, h := range got {
		if want := uint64(3 + i); h.Number != want {
			t.Errorf("head %d has number %d, want %d", i, h.Number, want)
		}
	}
}

// Heads arrive in ascending order. A consumer building a log range from them
// depends on it.
func TestHeadPollerEmitsInOrder(t *testing.T) {
	chain := newFakeChain()
	chain.extend(1, "a")
	p := NewHeadPoller(chain)
	collect(t, p)

	chain.extend(4, "a")
	got := collect(t, p)
	for i := 1; i < len(got); i++ {
		if got[i].Number <= got[i-1].Number {
			t.Fatalf("heads out of order: %d then %d", got[i-1].Number, got[i].Number)
		}
	}
}

// The same block must never be delivered twice. A duplicate head makes a
// subscriber double-count, and in the next slice that is double billing.
func TestHeadPollerNeverRepeatsAHead(t *testing.T) {
	chain := newFakeChain()
	chain.extend(3, "a")
	p := NewHeadPoller(chain)

	seen := map[string]bool{}
	for i := 0; i < 5; i++ {
		for _, h := range collect(t, p) {
			id := fmt.Sprintf("%d:%s", h.Number, h.Hash)
			if seen[id] {
				t.Fatalf("head %s delivered twice", id)
			}
			seen[id] = true
		}
		chain.extend(1, "a")
	}
}

// A reorg at the same height is a different block, and the subscriber must hear
// about the replacement. A native subscription gets this from the node; a
// poller has to notice it.
func TestHeadPollerDetectsASameHeightReorg(t *testing.T) {
	chain := newFakeChain()
	chain.extend(3, "a")
	p := NewHeadPoller(chain)
	collect(t, p)

	chain.reorg(2, 1, "b") // height 3 becomes a different block
	got := collect(t, p)
	if len(got) != 1 {
		t.Fatalf("got %d heads after a reorg, want 1", len(got))
	}
	if got[0].Number != 3 {
		t.Errorf("number = %d, want 3", got[0].Number)
	}
	if got[0].Hash != "0xb3" {
		t.Errorf("hash = %q, want the replacement block 0xb3", got[0].Hash)
	}
	if !got[0].Reorged {
		t.Error("Reorged = false, want true — a consumer must be able to tell")
	}
}

// A deeper reorg replaces several blocks. The poller walks back to the common
// ancestor and re-emits everything after it, so a consumer can undo the branch
// it already saw.
func TestHeadPollerWalksBackToTheCommonAncestor(t *testing.T) {
	chain := newFakeChain()
	chain.extend(5, "a")
	p := NewHeadPoller(chain)
	collect(t, p)

	chain.reorg(2, 4, "b") // heights 3,4,5 replaced, and a 6th block added
	got := collect(t, p)

	if len(got) != 4 {
		t.Fatalf("got %d heads, want 4 (heights 3..6)", len(got))
	}
	if got[0].Number != 3 {
		t.Errorf("first replayed head = %d, want 3 (the common ancestor is 2)", got[0].Number)
	}
	for _, h := range got {
		if h.Hash[:3] != "0xb" {
			t.Errorf("head %d has hash %q, want a block from the new branch", h.Number, h.Hash)
		}
	}
	if !got[0].Reorged {
		t.Error("the first replayed head must be marked as a reorg")
	}
}

// A shorter new branch still wins if the node reports it as head. The poller
// follows the node rather than preferring the longer chain it remembers.
func TestHeadPollerFollowsAShorterReorgedChain(t *testing.T) {
	chain := newFakeChain()
	chain.extend(5, "a")
	p := NewHeadPoller(chain)
	collect(t, p)

	chain.reorg(2, 1, "b") // head goes backwards, from 5 to 3
	got := collect(t, p)
	if len(got) == 0 {
		t.Fatal("a backwards reorg produced no heads")
	}
	last := got[len(got)-1]
	if last.Number != 3 || last.Hash != "0xb3" {
		t.Errorf("last head = %d/%s, want 3/0xb3", last.Number, last.Hash)
	}
}

// An upstream failure is reported, not swallowed. A poller that hid an error
// would look like a quiet chain, and a subscriber would wait forever.
func TestHeadPollerReportsAnUpstreamFailure(t *testing.T) {
	chain := newFakeChain()
	chain.extend(1, "a")
	p := NewHeadPoller(chain)
	collect(t, p)

	chain.err = errors.New("upstream refused")
	if _, err := p.Poll(context.Background()); err == nil {
		t.Fatal("err = nil, want the upstream failure")
	}
}

// A failed poll must not advance the cursor. The next successful poll has to
// deliver the block the failed one missed.
func TestHeadPollerDoesNotLoseABlockAfterAFailure(t *testing.T) {
	chain := newFakeChain()
	chain.extend(1, "a")
	p := NewHeadPoller(chain)
	collect(t, p)

	chain.extend(1, "a")
	chain.err = errors.New("upstream refused")
	if _, err := p.Poll(context.Background()); err == nil {
		t.Fatal("expected the poll to fail")
	}

	chain.err = nil
	got := collect(t, p)
	if len(got) != 1 || got[0].Number != 2 {
		t.Fatalf("got %v, want the block the failed poll missed (height 2)", got)
	}
}

// The walk back is bounded. A pathological upstream that never agrees with the
// poller must not make it fetch the whole chain.
func TestHeadPollerBoundsTheReorgWalk(t *testing.T) {
	chain := newFakeChain()
	chain.extend(300, "a")
	p := NewHeadPoller(chain)
	collect(t, p)

	chain.reorg(1, 250, "b")
	_, err := p.Poll(context.Background())
	if !errors.Is(err, ErrReorgTooDeep) {
		t.Fatalf("err = %v, want ErrReorgTooDeep", err)
	}
}
