package relay

import (
	"context"
	"errors"
	"fmt"
)

// ErrReorgTooDeep is a reorg the poller refuses to walk. A pathological
// upstream that never agrees with the poller must not make it fetch the whole
// chain looking for an ancestor that is not there.
var ErrReorgTooDeep = errors.New("relay: reorg deeper than the walk limit")

// maxReorgDepth bounds both the remembered window and the walk back to a common
// ancestor. No honest EVM chain reorganises this far. A chain that appears to
// has really changed identity, and the right answer is to fail loudly rather
// than replay hundreds of blocks into every subscriber.
const maxReorgDepth = 128

// maxCatchUpBlocks bounds one poll's output. A relay that was paused, or a
// chain that jumped, must not turn a single poll into an unbounded fetch loop.
const maxCatchUpBlocks = 512

// BlockRef is the part of a block a head subscription needs. Reorged marks the
// first head of a replacement branch, so a consumer can undo the branch it
// already saw.
type BlockRef struct {
	Number     uint64
	Hash       string
	ParentHash string
	Reorged    bool
}

// BlockFetcher reads a chain's head. The relay calls it over plain HTTP, which
// is the whole point of terminating WebSocket here: an upstream needs no
// WebSocket support to feed a subscription.
type BlockFetcher interface {
	HeadNumber(ctx context.Context) (uint64, error)
	BlockByNumber(ctx context.Context, n uint64) (BlockRef, error)
}

// HeadPoller turns a polled chain into an ordered head stream.
//
// It owns two jobs a node used to do for a native subscription: never deliver a
// head twice, and never leave a subscriber on a branch the chain abandoned. A
// poller that skips a block silently loses logs for every subscriber on the
// chain, and nothing reports it — so gap filling is not an optimisation here.
//
// Finding a common ancestor needs memory of the branch the poller believed in.
// The upstream can only be asked for the branch it holds NOW, so after a reorg
// the old blocks are unreachable by height. The poller therefore keeps a window
// of recent hashes and backfills it once at start. Without that window it could
// walk back exactly one block, which is not enough for any real reorg.
//
// A HeadPoller is used by one goroutine at a time. One poller serves every
// subscriber on a chain, which is why N subscribers cost one upstream loop.
type HeadPoller struct {
	src BlockFetcher
	// window maps height to the hash the poller believes canonical. It holds at
	// most maxReorgDepth entries.
	window map[uint64]string
	// last is the most recent head delivered.
	last BlockRef
	// primed records that the first poll has run. The first poll delivers
	// nothing: a subscriber wants what happens next, not the current head.
	primed bool
}

// NewHeadPoller builds a poller over src.
func NewHeadPoller(src BlockFetcher) *HeadPoller {
	return &HeadPoller{src: src, window: make(map[uint64]string)}
}

// Poll returns every head since the previous call, in ascending order.
//
// A failed poll does not advance the cursor, so the next successful one
// delivers whatever the failure missed.
func (p *HeadPoller) Poll(ctx context.Context) ([]BlockRef, error) {
	headNum, err := p.src.HeadNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("relay: read head: %w", err)
	}
	head, err := p.src.BlockByNumber(ctx, headNum)
	if err != nil {
		return nil, fmt.Errorf("relay: read block %d: %w", headNum, err)
	}

	if !p.primed {
		return nil, p.prime(ctx, head)
	}
	if head.Number == p.last.Number && head.Hash == p.last.Hash {
		return nil, nil
	}

	resumeFrom, reorged, err := p.resumePoint(ctx, head)
	if err != nil {
		return nil, err
	}
	if head.Number >= resumeFrom && head.Number-resumeFrom >= maxCatchUpBlocks {
		return nil, fmt.Errorf("relay: %d blocks behind, more than one poll may catch up", head.Number-resumeFrom+1)
	}

	var out []BlockRef
	for n := resumeFrom; n <= head.Number; n++ {
		block, err := p.src.BlockByNumber(ctx, n)
		if err != nil {
			// The cursor is untouched, so the next poll retries this height.
			return nil, fmt.Errorf("relay: read block %d: %w", n, err)
		}
		if reorged && n == resumeFrom {
			block.Reorged = true
		}
		out = append(out, block)
		p.record(block)
	}
	return out, nil
}

// prime records the current branch so a later reorg has something to compare
// against. It costs up to maxReorgDepth reads, once, for the poller that serves
// a whole chain.
func (p *HeadPoller) prime(ctx context.Context, head BlockRef) error {
	p.record(head)

	lowest := uint64(0)
	if head.Number > maxReorgDepth-1 {
		lowest = head.Number - (maxReorgDepth - 1)
	}
	for n := head.Number; n > lowest; n-- {
		block, err := p.src.BlockByNumber(ctx, n-1)
		if err != nil {
			return fmt.Errorf("relay: backfill block %d: %w", n-1, err)
		}
		p.window[block.Number] = block.Hash
	}
	p.primed = true
	return nil
}

// record remembers a delivered head and prunes the window.
func (p *HeadPoller) record(block BlockRef) {
	p.last = BlockRef{Number: block.Number, Hash: block.Hash, ParentHash: block.ParentHash}
	p.window[block.Number] = block.Hash
	if block.Number < maxReorgDepth {
		return
	}
	cutoff := block.Number - maxReorgDepth
	for height := range p.window {
		if height < cutoff {
			delete(p.window, height)
		}
	}
}

// resumePoint decides which height to replay from, and whether the chain
// reorganised.
func (p *HeadPoller) resumePoint(ctx context.Context, head BlockRef) (uint64, bool, error) {
	// The head moved backwards, or sits at the same height with a different
	// hash. Either way the branch the poller remembers is gone.
	if head.Number <= p.last.Number {
		return p.walkBack(ctx, head.Number)
	}

	// The head advanced. The branch still agrees only if the block just after
	// the last delivered head still points at it.
	next, err := p.src.BlockByNumber(ctx, p.last.Number+1)
	if err != nil {
		return 0, false, fmt.Errorf("relay: read block %d: %w", p.last.Number+1, err)
	}
	if next.ParentHash == p.last.Hash {
		return p.last.Number + 1, false, nil
	}
	return p.walkBack(ctx, head.Number)
}

// walkBack finds the highest height at which the upstream's chain still matches
// the window, then returns the height after it. A height the window does not
// cover cannot be confirmed as the fork point, so the walk keeps going and
// eventually gives up — which is the honest answer, because replaying from an
// unverified height would either repeat heads or skip them.
func (p *HeadPoller) walkBack(ctx context.Context, upstreamHead uint64) (uint64, bool, error) {
	start := min(upstreamHead, p.last.Number)

	for depth := 0; depth <= maxReorgDepth; depth++ {
		if uint64(depth) > start {
			break
		}
		height := start - uint64(depth)
		block, err := p.src.BlockByNumber(ctx, height)
		if err != nil {
			return 0, false, fmt.Errorf("relay: read block %d: %w", height, err)
		}
		if known, ok := p.window[height]; ok && known == block.Hash {
			// The branches rejoin here, so everything above it is new.
			return height + 1, true, nil
		}
	}
	return 0, false, fmt.Errorf("%w: walked back %d blocks without finding a common ancestor", ErrReorgTooDeep, maxReorgDepth)
}
