package tui

import "context"

// visit is the generation token a screen keeps for work it starts: a number
// that changes whenever the work is replaced or abandoned, and a context that
// ends that work. A reply carries the gen it was started under, and one that
// no longer matches current is dropped.
type visit struct {
	gen    uint64
	ctx    context.Context
	cancel context.CancelFunc
}

// begin ends any earlier visit and starts the next one under parent.
func (v *visit) begin(parent context.Context) { v.open(v.gen+1, parent) }

// open is begin with a gen the caller counts, for a screen whose replies
// must stay distinguishable across screens (a reopened box).
func (v *visit) open(gen uint64, parent context.Context) {
	v.end()
	v.gen = gen
	v.ctx, v.cancel = context.WithCancel(parent)
}

// end cancels the visit's work. Replies still in flight stay current until
// invalidate or the next begin.
func (v *visit) end() {
	if v.cancel != nil {
		v.cancel()
		v.cancel = nil
	}
}

// invalidate ends the visit and makes every reply in flight stale.
func (v *visit) invalidate() {
	v.end()
	v.gen++
}

// current reports whether a reply started under gen is still wanted.
func (v *visit) current(gen uint64) bool { return gen == v.gen }
