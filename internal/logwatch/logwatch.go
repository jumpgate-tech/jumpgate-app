// Package logwatch tails journald units live via executor.Executor,
// classifies each line against a signature table of known failure modes
// (falling back to a raw level-word severity for unrecognized error-ish
// lines), and keeps a capped ring buffer of the results — fanned out to
// subscribers (the SSE stream) the same way internal/monitor fans out
// Snapshots.
package logwatch

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// Hit is one classified log line.
type Hit struct {
	Unit      string    `json:"unit"`
	Line      string    `json:"line"`
	At        time.Time `json:"at"`
	Signature string    `json:"signature"` // "" = unclassified error-ish line
	Severity  string    `json:"severity"`  // info|warn|error|critical
	Explain   string    `json:"explain"`   // canned explanation, may be ""
	LearnURL  string    `json:"learnUrl,omitempty"`
}

// ringSize caps how many Hits Watcher retains. journald units tail forever,
// so an uncapped buffer would grow without bound over a long-running
// process; 1000 is generous for "recent activity" (an operator debugging a
// live incident) without the ring becoming a meaningful memory or Recent()
// scan cost.
const ringSize = 1000

// Watcher tails a fixed set of journald units for the lifetime of a
// context, classifying every line and keeping the most recent ringSize
// Hits.
type Watcher struct {
	exec  executor.Executor
	units []string

	// ring is a fixed-capacity circular buffer. It grows by append until it
	// holds ringSize hits; after that each new hit overwrites the oldest in
	// place at index next, so a full ring costs no allocation per line.
	// Recent unrolls it back into oldest-first order.
	mu   sync.Mutex
	ring []Hit
	next int // index of the oldest hit (the next slot to overwrite) once full

	// subs is guarded by subsMu. Lock order: mu, then subsMu. publish runs
	// under both, so a backlog read under mu and a subscription made under
	// the same hold see every hit exactly once.
	subsMu sync.Mutex
	subs   map[<-chan Hit]*subscriber
}

// subscriber is one Subscribe channel and how many hits it has lost to a
// full buffer since it last resynced.
type subscriber struct {
	ch      chan Hit
	dropped int
}

// New constructs a Watcher over units. It does not start tailing — call
// Start.
func New(e executor.Executor, units []string) *Watcher {
	return &Watcher{
		exec:  e,
		units: units,
		subs:  map[<-chan Hit]*subscriber{},
	}
}

// Start begins tailing every configured unit in its own goroutine, each via
// `journalctl -u <unit> -f`, until ctx is canceled. Executor.Run is
// long-running for a follow; when ctx is canceled the underlying command
// exits and Run returns — a transport error at that point is expected and
// ignored, not treated as fatal.
func (w *Watcher) Start(ctx context.Context) {
	for _, u := range w.units {
		unit := u
		go w.tail(ctx, unit)
	}
}

// tailBackoffMin/Max bound the delay between re-invoking journalctl after
// Run returns for any reason other than ctx cancellation (SSH transport
// drop, journald restart) — doubling from tailBackoffMin up to
// tailBackoffMax. tailBackoffResetAfter: a Run that stayed up at least that
// long is treated as having recovered, so the next retry starts back at
// tailBackoffMin rather than continuing to climb toward the cap.
const (
	tailBackoffMin        = 1 * time.Second
	tailBackoffMax        = 10 * time.Second
	tailBackoffResetAfter = 30 * time.Second
)

func (w *Watcher) tail(ctx context.Context, unit string) {
	// -n 0: don't replay backlog, only new lines from "now". -o cat: raw
	// message text only, no journald metadata prefix — classification
	// works directly on the client's own log line.
	cmd := fmt.Sprintf("journalctl -u %s -f -n 0 --no-pager -o cat", shQuote(unit))

	backoff := tailBackoffMin
	for ctx.Err() == nil {
		start := time.Now()
		_, _ = w.exec.Run(ctx, cmd, &executor.RunOpts{
			Stream: func(line string) { w.handleLine(unit, line) },
		})
		if ctx.Err() != nil {
			// Canceled — Run returning is expected, not a failure to retry.
			return
		}

		// A Run that survived a while before dropping is treated as a
		// transient blip on an otherwise-healthy tail, not a persistent
		// failure — don't keep climbing the backoff for it.
		if time.Since(start) > tailBackoffResetAfter {
			backoff = tailBackoffMin
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > tailBackoffMax {
			backoff = tailBackoffMax
		}
	}
}

func (w *Watcher) handleLine(unit, line string) {
	hit, ok := classify(unit, line, time.Now())
	if !ok {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.ring) < ringSize {
		w.ring = append(w.ring, hit)
	} else {
		w.ring[w.next] = hit
		w.next = (w.next + 1) % ringSize
	}
	// Publish under mu (its sends never block), so a subscriber that takes
	// its backlog under mu cannot also receive the same hit live.
	w.publish(hit)
}

// Recent returns up to the n most recent Hits, oldest first / newest last.
func (w *Watcher) Recent(n int) []Hit {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.recentLocked(n)
}

// recentLocked is Recent with w.mu held; n <= 0 means the whole ring.
func (w *Watcher) recentLocked(n int) []Hit {
	if n <= 0 || n > len(w.ring) {
		n = len(w.ring)
	}
	// Logical index 0 is the oldest hit, at physical index w.next (which
	// stays 0 until the ring first fills). The newest n are logical
	// indices len-n .. len-1; map each back onto the ring.
	size := len(w.ring)
	out := make([]Hit, n)
	for i := range out {
		out[i] = w.ring[(w.next+size-n+i)%size]
	}
	return out
}

// Subscribe registers a new subscriber and returns a channel that receives
// every subsequently classified Hit, plus an unsubscribe func. The channel
// is best-effort: a slow consumer that doesn't drain it may miss hits, but
// never blocks tailing or other subscribers. Callers must call the
// returned func when done to avoid leaking the subscription.
func (w *Watcher) Subscribe() (<-chan Hit, func()) {
	w.subsMu.Lock()
	defer w.subsMu.Unlock()
	return w.subscribeLocked()
}

// subscribeLocked is Subscribe with w.subsMu held.
func (w *Watcher) subscribeLocked() (<-chan Hit, func()) {
	s := &subscriber{ch: make(chan Hit, 32)}
	w.subs[s.ch] = s
	unsub := func() {
		w.subsMu.Lock()
		delete(w.subs, s.ch)
		w.subsMu.Unlock()
	}
	return s.ch, unsub
}

// SubscribeRecent is Subscribe plus the newest n hits (oldest first), taken
// atomically: every hit is either in the backlog or arrives on the channel,
// never both and never neither. n <= 0 is an empty backlog (unlike Recent,
// where it means the whole ring).
func (w *Watcher) SubscribeRecent(n int) ([]Hit, <-chan Hit, func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	backlog := []Hit{}
	if n > 0 {
		backlog = w.recentLocked(n)
	}
	w.subsMu.Lock()
	defer w.subsMu.Unlock()
	ch, unsub := w.subscribeLocked()
	return backlog, ch, unsub
}

// Resync reports whether the subscriber ch has lost hits to a full buffer
// since it subscribed or last resynced. If it has, Resync empties ch and
// returns the newest n hits (n <= 0: none) atomically, the way SubscribeRecent
// does, so the subscriber can replace what it holds instead of silently
// skipping lines.
func (w *Watcher) Resync(ch <-chan Hit, n int) ([]Hit, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.subsMu.Lock()
	defer w.subsMu.Unlock()
	s := w.subs[ch]
	if s == nil || s.dropped == 0 {
		return nil, false
	}
	s.dropped = 0
	for len(s.ch) > 0 {
		<-s.ch
	}
	if n <= 0 {
		return []Hit{}, true
	}
	return w.recentLocked(n), true
}

// publish fans hit out to every subscriber; the caller holds w.mu.
func (w *Watcher) publish(hit Hit) {
	w.subsMu.Lock()
	defer w.subsMu.Unlock()
	for _, s := range w.subs {
		select {
		case s.ch <- hit:
		default:
			// Slow consumer: drop this hit rather than block tailing or
			// other subscribers, and count it so Resync can say so.
			s.dropped++
		}
	}
}

// Classify is the exported form of classify, for callers outside the
// tailing machinery (e.g. ops' network diagnostics cross-referencing a
// one-shot journal dump against the signature table).
func Classify(unit, line string, now time.Time) (Hit, bool) {
	return classify(unit, line, now)
}

// classify matches line against the signature table (first match wins); if
// none match, an error-ish line (one carrying a warn, error or critical
// level token or level= field; see levelSeverity) still produces an
// unclassified Hit (Signature ""), severity taken from that level. A benign line with neither yields
// ok=false — no Hit at all.
func classify(unit, line string, now time.Time) (Hit, bool) {
	for _, sig := range signatures {
		if sig.pattern.MatchString(line) {
			if sig.requireErrLevel && !hasErrLevel(line) {
				continue
			}
			return Hit{
				Unit:      unit,
				Line:      line,
				At:        now,
				Signature: sig.name,
				Severity:  sig.severity,
				Explain:   sig.explain,
				LearnURL:  sig.learnURL,
			}, true
		}
	}

	if sev, ok := levelSeverity(line); ok {
		return Hit{
			Unit:     unit,
			Line:     line,
			At:       now,
			Severity: sev,
		}, true
	}

	return Hit{}, false
}

// shQuote single-quotes s for safe interpolation into a `sh -c` command
// string, escaping any embedded single quotes. Unit names are a fixed,
// caller-controlled catalog (not untrusted input), but this keeps the
// command construction consistent with internal/monitor's approach.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
