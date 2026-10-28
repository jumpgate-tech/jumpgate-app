package server

import (
	"errors"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/vault"
)

// The passphrase routes (POST /api/vault/unlock and POST /api/system/seal)
// back off per peer uid after wrong passphrases: the first few are free (a
// typo), then each further one doubles the wait, up to passBackoffMax. A
// right passphrase clears the uid's count. Failures are logged with the uid
// only; the passphrase is never logged.
var (
	passBackoffFree = 3
	passBackoffBase = time.Second
	passBackoffMax  = 5 * time.Minute
)

// unknownUID is the bucket for a request whose peer uid cannot be read.
const unknownUID = -1

type passFailure struct {
	n     int       // wrong passphrases in a row
	until time.Time // no attempt before this
}

type passBackoff struct {
	mu   sync.Mutex
	now  func() time.Time
	fail map[int]passFailure
	busy map[int]bool // uids with an attempt in flight
}

func (b *passBackoff) clock() time.Time {
	if b.now != nil {
		return b.now()
	}
	return time.Now()
}

// begin reserves uid's one attempt. It refuses, with the wait still owed
// (0 when another attempt from uid is in flight), while uid is backing off
// or already has an attempt running: concurrent requests from one uid
// cannot each slip past the check before any failure is recorded.
func (b *passBackoff) begin(uid int) (bool, time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.busy[uid] {
		return false, 0
	}
	if f, ok := b.fail[uid]; ok {
		if d := f.until.Sub(b.clock()); d > 0 {
			return false, d
		}
	}
	if b.busy == nil {
		b.busy = map[int]bool{}
	}
	b.busy[uid] = true
	return true, 0
}

// end releases uid's reservation.
func (b *passBackoff) end(uid int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.busy, uid)
}

// failed records a wrong passphrase from uid and returns the count in a
// row and the wait it now owes.
func (b *passBackoff) failed(uid int) (int, time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.fail == nil {
		b.fail = map[int]passFailure{}
	}
	f := b.fail[uid]
	f.n++
	var d time.Duration
	if over := f.n - passBackoffFree; over > 0 {
		d = passBackoffMax
		if over <= 30 {
			if x := float64(passBackoffBase) * math.Pow(2, float64(over-1)); x < float64(passBackoffMax) {
				d = time.Duration(x)
			}
		}
	}
	f.until = b.clock().Add(d)
	b.fail[uid] = f
	return f.n, d
}

func (b *passBackoff) succeeded(uid int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.fail, uid)
}

// requestUID is the uid of the peer behind r: the admin socket's peer
// credentials, else the loopback socket's owner where it can be read, else
// unknownUID.
func (s *Server) requestUID(r *http.Request) int {
	if c, ok := peerFrom(r.Context()); ok {
		return c.UID
	}
	if s.peerUID != nil {
		if uid, v := s.peerUID(r); v == peerFound {
			return uid
		}
	}
	return unknownUID
}

// errPassUnchecked releases an attempt that ended before the passphrase
// was checked (a vault that would not open, say): it counts neither way.
var errPassUnchecked = errors.New("passphrase not checked")

// passCheckHook, when set, runs as each reserved attempt reaches the
// passphrase check (a test seam).
var passCheckHook func()

// checkPassphrase runs the KDF-bound check, after the hook.
func (s *Server) checkPassphrase(check func() error) error {
	if passCheckHook != nil {
		passCheckHook()
	}
	return check()
}

// passBegin reserves uid's attempt on route, or answers 429 and returns
// false. done records the outcome and releases the reservation; only its
// first call counts, so a route can defer done(errPassUnchecked) and still
// call done(err) once the passphrase has been checked.
func (s *Server) passBegin(w http.ResponseWriter, route string, uid int) (done func(error), ok bool) {
	ok, d := s.passBackoff.begin(uid)
	if !ok {
		secs := max(int(math.Ceil(d.Seconds())), 1)
		if d > 0 {
			log.Printf("jumpgate: %s refused for uid %d: too many wrong passphrases, %ds left to wait", route, uid, secs)
		} else {
			log.Printf("jumpgate: %s refused for uid %d: another attempt from that uid is in progress", route, uid)
		}
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		writeError(w, http.StatusTooManyRequests, "too many passphrase attempts; try again in "+strconv.Itoa(secs)+"s")
		return nil, false
	}
	var once sync.Once
	return func(err error) {
		once.Do(func() {
			defer s.passBackoff.end(uid)
			s.passResult(route, uid, err)
		})
	}, true
}

// passResult records how a passphrase attempt from uid went.
func (s *Server) passResult(route string, uid int, err error) {
	switch {
	case errors.Is(err, errPassUnchecked):
	case err == nil:
		s.passBackoff.succeeded(uid)
	case errors.Is(err, vault.ErrWrongPassphrase), errors.Is(err, vault.ErrSlotRefused):
		n, d := s.passBackoff.failed(uid)
		log.Printf("jumpgate: %s: wrong passphrase from uid %d (%d in a row; next attempt in %s)", route, uid, n, d)
	}
}
